package kubernetes

import (
	"context"
	"os"
	"strings"

	"github.com/pkg/errors"
	authv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	libClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	libCluster "sigs.k8s.io/controller-runtime/pkg/cluster"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/user"
	"github.com/akuity/kargo/internal/logging"
)

// ClientOptions specifies options for customizing the client returned by the
// NewClient function.
type ClientOptions struct {
	// NewInternalClient may be used to take control of how the client's own
	// internal/underlying controller-runtime client is created. This is mainly
	// useful for tests wherein one may, for instance, wish to inject a custom
	// implementation of that interface created using fake.NewClientBuilder().
	// Ordinarily, the value of this field should be left as nil/unspecified, in
	// which case, the NewClient function to which this struct is passed will
	// supply its own default implementation.
	NewInternalClient func(
		context.Context,
		*rest.Config,
		*runtime.Scheme,
	) (libClient.Client, error)
	// NewInternalDynamicClient may be used to take control of how the client's
	// own internal/underlying client-go dynamic client is created. This is mainly
	// useful for tests wherein one may wish to inject a custom implementation of
	// that interface. Ordinarily, the value of this field should be left as
	// nil/unspecified, in which case, the NewClient function to which this struct
	// is passed will supply its own default implementation.
	NewInternalDynamicClient func(*rest.Config) (dynamic.Interface, error)
	// Scheme may be used to take control of the scheme used by the client's own
	// internal/underlying controller-runtime client. Ordinarily, the value of
	// this field should be left as nil/unspecified, in which case, the NewClient
	// function to which this struct is passed will supply a default scheme that
	// includes all Kubernetes APIs used by the Kargo API server.
	Scheme *runtime.Scheme
}

// setOptionsDefaults sets default values for any unspecified fields in the
// provided ClientOptions struct.
func setOptionsDefaults(opts ClientOptions) (ClientOptions, error) {
	if opts.Scheme == nil {
		opts.Scheme = runtime.NewScheme()
		if err := kubescheme.AddToScheme(opts.Scheme); err != nil {
			return opts,
				errors.Wrap(err, "error adding Kubernetes API to scheme")
		}
		if err := kargoapi.AddToScheme(opts.Scheme); err != nil {
			return opts, errors.Wrap(err, "error adding Kargo API to scheme")
		}
	}
	if opts.NewInternalClient == nil {
		opts.NewInternalClient = newDefaultInternalClient
	}
	if opts.NewInternalDynamicClient == nil {
		opts.NewInternalDynamicClient = dynamic.NewForConfig
	}
	return opts, nil
}

// The Client interface combines the familiar controller-runtime Client
// interface with a helpful Watch function that is absent from that interface.
type Client interface {
	libClient.Client
	// Watch returns a suitable implementation of the watch.Interface for
	// subscribing to the resources described by the provided arguments.
	Watch(
		ctx context.Context,
		obj libClient.Object,
		namespace string,
		opts metav1.ListOptions,
	) (watch.Interface, error)
}

// client implements Client.
type client struct {
	internalClient        libClient.Client
	statusWriter          *authorizingStatusWriterWrapper
	internalDynamicClient dynamic.Interface

	getAuthorizedClientFn func(
		ctx context.Context,
		internalClient libClient.Client,
		verb string,
		gvr schema.GroupVersionResource,
		subresource string,
		key libClient.ObjectKey,
	) (libClient.Client, error)
}

// NewClient returns an implementation of the Client interface. The interface
// and implementation offer two key advantages:
//  1. The Client interface combines the familiar controller-runtime Client
//     interface with a helpful Watch function that is absent from that
//     interface.
//  2. The implementation enforces RBAC by retrieving context-bound user.Info
//     and using it to conduct a SubjectAccessReview or SelfSubjectAccessReview
//     before (if successful) performing the desired operation. This permits
//     this client to retain the benefits of using a single underlying client
//     (typically with a built-in cache), while still enforcing RBAC as if the
//     operation had been performed with a user-specific client constructed
//     ad-hoc using the user's own credentials.
func NewClient(
	ctx context.Context,
	restCfg *rest.Config,
	opts ClientOptions,
) (Client, error) {
	var err error
	if opts, err = setOptionsDefaults(opts); err != nil {
		return nil, errors.Wrap(err, "error setting client options defaults")
	}
	internalClient, err :=
		opts.NewInternalClient(ctx, restCfg, opts.Scheme)
	if err != nil {
		return nil, errors.Wrap(err, "error building internal client")
	}
	internalDynamicClient, err :=
		opts.NewInternalDynamicClient(restCfg)
	if err != nil {
		return nil, errors.Wrap(err, "error building internal dynamic client")
	}
	return &client{
		internalClient: internalClient,
		statusWriter: &authorizingStatusWriterWrapper{
			internalClient:        internalClient,
			getAuthorizedClientFn: getAuthorizedClient,
		},
		internalDynamicClient: internalDynamicClient,
		getAuthorizedClientFn: getAuthorizedClient,
	}, nil
}

func newDefaultInternalClient(
	ctx context.Context,
	restCfg *rest.Config,
	scheme *runtime.Scheme,
) (libClient.Client, error) {
	cluster, err := libCluster.New(
		restCfg,
		func(clusterOptions *libCluster.Options) {
			clusterOptions.Scheme = scheme
		},
	)
	if err != nil {
		return nil,
			errors.Wrap(err, "error creating controller-runtime cluster")
	}
	go func() {
		err = cluster.Start(ctx)
	}()
	if !cluster.GetCache().WaitForCacheSync(ctx) {
		return nil, errors.New("error waiting for cache sync")
	}
	return cluster.GetClient(), errors.Wrap(err, "error starting cluster")
}

func (c *client) Get(
	ctx context.Context,
	key libClient.ObjectKey,
	obj libClient.Object,
	opts ...libClient.GetOption,
) error {
	// We don't want to use the key that is returned by this call. We want to use
	// the key that was passed to us.
	gvr, _, err := gvrAndKeyFromObj(obj, nil, c.internalClient.Scheme())
	if err != nil {
		return err
	}
	client, err := c.getAuthorizedClientFn(
		ctx,
		c.internalClient,
		"get",
		gvr,
		"", // No subresource
		key,
	)
	if err != nil {
		return err
	}
	return client.Get(ctx, key, obj, opts...)
}

func (c *client) List(
	ctx context.Context,
	list libClient.ObjectList,
	opts ...libClient.ListOption,
) error {
	// We don't want to use the key that is returned by this call. We want to
	// build one ourselves since, in the case of a list operation, namespace, if
	// specified, is provided in the form of an option.
	gvr, _, err := gvrAndKeyFromObj(list, nil, c.internalClient.Scheme())
	if err != nil {
		return err
	}
	var key libClient.ObjectKey
	for _, opt := range opts { // Need to find namespace, if any, from opts
		if n, ok := opt.(libClient.InNamespace); ok {
			key.Namespace = string(n)
			break
		}
	}
	client, err := c.getAuthorizedClientFn(
		ctx,
		c.internalClient,
		"list",
		gvr,
		"",  // No subresource
		key, // Has empty Name field; Name makes no sense for List
	)
	if err != nil {
		return err
	}
	return client.List(ctx, list, opts...)
}

func (c *client) Create(
	ctx context.Context,
	obj libClient.Object,
	opts ...libClient.CreateOption,
) error {
	gvr, key, err := gvrAndKeyFromObj(obj, obj, c.internalClient.Scheme())
	if err != nil {
		return err
	}
	client, err := c.getAuthorizedClientFn(
		ctx,
		c.internalClient,
		"create",
		gvr,
		"", // No subresource
		*key,
	)
	if err != nil {
		return err
	}
	return client.Create(ctx, obj, opts...)
}

func (c *client) Delete(
	ctx context.Context,
	obj libClient.Object,
	opts ...libClient.DeleteOption,
) error {
	gvr, key, err := gvrAndKeyFromObj(obj, obj, c.internalClient.Scheme())
	if err != nil {
		return err
	}
	client, err := c.getAuthorizedClientFn(
		ctx,
		c.internalClient,
		"delete",
		gvr,
		"", // No subresource
		*key,
	)
	if err != nil {
		return err
	}
	return client.Delete(ctx, obj, opts...)
}

func (c *client) Update(
	ctx context.Context,
	obj libClient.Object,
	opts ...libClient.UpdateOption,
) error {
	gvr, key, err := gvrAndKeyFromObj(obj, obj, c.internalClient.Scheme())
	if err != nil {
		return err
	}
	client, err := c.getAuthorizedClientFn(
		ctx,
		c.internalClient,
		"update",
		gvr,
		"", // No subresource
		*key,
	)
	if err != nil {
		return err
	}
	return client.Update(ctx, obj, opts...)
}

func (c *client) Patch(
	ctx context.Context,
	obj libClient.Object,
	patch libClient.Patch,
	opts ...libClient.PatchOption,
) error {
	gvr, key, err := gvrAndKeyFromObj(obj, obj, c.internalClient.Scheme())
	if err != nil {
		return err
	}
	client, err := c.getAuthorizedClientFn(
		ctx,
		c.internalClient,
		"patch",
		gvr,
		"", // No subresource
		*key,
	)
	if err != nil {
		return err
	}
	return client.Patch(ctx, obj, patch, opts...)
}

func (c *client) DeleteAllOf(
	ctx context.Context,
	obj libClient.Object,
	opts ...libClient.DeleteAllOfOption,
) error {
	// We don't want to use the key that is returned by this call. We want to
	// build one ourselves since, in the case of a delete all operation,
	// namespace, if specified, is provided in the form of an option.
	gvr, _, err := gvrAndKeyFromObj(obj, nil, c.internalClient.Scheme())
	if err != nil {
		return err
	}
	var key libClient.ObjectKey
	for _, opt := range opts { // Need to find namespace, if any, from opts
		if n, ok := opt.(libClient.InNamespace); ok {
			key.Namespace = string(n)
			break
		}
	}
	client, err := c.getAuthorizedClientFn(
		ctx,
		c.internalClient,
		"deletecollection",
		gvr,
		"",  // No subresource
		key, // Has empty Name field; Name makes no sense for DeleteAllOf
	)
	if err != nil {
		return err
	}
	return client.DeleteAllOf(ctx, obj, opts...)
}

func (c *client) Status() libClient.StatusWriter {
	return c.statusWriter
}

func (c *client) Scheme() *runtime.Scheme {
	return c.internalClient.Scheme()
}

func (c *client) RESTMapper() meta.RESTMapper {
	return c.internalClient.RESTMapper()
}

// authorizingStatusWriterWrapper implements libClient.StatusWriter.
type authorizingStatusWriterWrapper struct {
	internalClient libClient.Client

	getAuthorizedClientFn func(
		ctx context.Context,
		internalClient libClient.Client,
		verb string,
		gvr schema.GroupVersionResource,
		subresource string,
		key libClient.ObjectKey,
	) (libClient.Client, error)
}

func (a *authorizingStatusWriterWrapper) Update(
	ctx context.Context,
	obj libClient.Object,
	opts ...libClient.UpdateOption,
) error {
	gvr, key, err := gvrAndKeyFromObj(obj, obj, a.internalClient.Scheme())
	if err != nil {
		return err
	}
	client, err := a.getAuthorizedClientFn(
		ctx,
		a.internalClient,
		"update",
		gvr,
		"status", // Subresource
		*key,
	)
	if err != nil {
		return err
	}
	return client.Status().Update(ctx, obj, opts...)
}

func (a *authorizingStatusWriterWrapper) Patch(
	ctx context.Context,
	obj libClient.Object,
	patch libClient.Patch,
	opts ...libClient.PatchOption,
) error {
	gvr, key, err := gvrAndKeyFromObj(obj, obj, a.internalClient.Scheme())
	if err != nil {
		return err
	}
	client, err := a.getAuthorizedClientFn(
		ctx,
		a.internalClient,
		"patch",
		gvr,
		"status", // Subresource
		*key,
	)
	if err != nil {
		return err
	}
	return client.Status().Patch(ctx, obj, patch, opts...)
}

func (c *client) Watch(
	ctx context.Context,
	obj libClient.Object,
	namespace string,
	opts metav1.ListOptions,
) (watch.Interface, error) {
	gvr, _, err := gvrAndKeyFromObj(obj, obj, c.internalClient.Scheme())
	if err != nil {
		return nil, err
	}
	if _, err = c.getAuthorizedClientFn(
		ctx,
		c.internalClient,
		"watch",
		gvr,
		"", // No subresource
		libClient.ObjectKey{
			Namespace: namespace,
		},
	); err != nil {
		return nil, err
	}
	var ri dynamic.ResourceInterface
	if namespace != "" {
		ri = c.internalDynamicClient.Resource(gvr).Namespace(namespace)
	} else {
		ri = c.internalDynamicClient.Resource(gvr)
	}
	return ri.Watch(ctx, opts)
}

func GetRestConfig(ctx context.Context, path string) (*rest.Config, error) {
	logger := logging.LoggerFromContext(ctx)

	// clientcmd.BuildConfigFromFlags will fall back on in-cluster config if path
	// is empty, but will issue a warning that we can suppress by checking for
	// that condition ourselves and calling rest.InClusterConfig() directly.
	if path == "" {
		logger.Debug("loading in-cluster REST config")
		cfg, err := rest.InClusterConfig()
		return cfg, errors.Wrap(err, "error loading in-cluster REST config")
	}

	logger.WithField("path", path).Debug("loading REST config from path")
	cfg, err := clientcmd.BuildConfigFromFlags("", path)
	return cfg, errors.Wrapf(err, "error loading REST config from %q", path)
}

// gvrAndKeyFromObj extracts the group, version, and plural resource type
// information from the provided object.
func gvrAndKeyFromObj(
	runtimeObj runtime.Object, // Could be a list
	clientObj libClient.Object, // Can not be a list
	scheme *runtime.Scheme,
) (schema.GroupVersionResource, *libClient.ObjectKey, error) {
	gvk, err := apiutil.GVKForObject(runtimeObj, scheme)
	if err != nil {
		return schema.GroupVersionResource{}, nil,
			errors.Wrap(err, "error extracting GVK from object")
	}
	// In case this was a list, we trim the "List" suffix to get the kind that's
	// IN the list.
	gvk.Kind = strings.TrimSuffix(gvk.Kind, "List")
	// The first return value is pluralized and that's the one we want...
	pluralizedGVR, _ := meta.UnsafeGuessKindToResource(gvk)
	var key *libClient.ObjectKey
	if clientObj != nil {
		key = &libClient.ObjectKey{
			Namespace: clientObj.GetNamespace(),
			Name:      clientObj.GetName(),
		}
	}
	return pluralizedGVR, key, nil
}

// getAuthorizedClient examines context-bound user.Info and uses information
// found therein to attempt to identify or build an appropriate client for
// performing the desired operation. If it is unable to do so, it amounts to the
// operation being unauthorized and an error is returned.
func getAuthorizedClient(
	ctx context.Context,
	internalClient libClient.Client,
	verb string,
	gvr schema.GroupVersionResource,
	subresource string,
	key libClient.ObjectKey,
) (libClient.Client, error) {
	userInfo, ok := user.InfoFromContext(ctx)
	if !ok {
		return nil, errors.New("not allowed")
	}

	// Admins get to use the Kargo API server's own Kubernetes client. i.e. They
	// can do everything the server can do.
	if userInfo.IsAdmin {
		return internalClient, nil
	}

	if userInfo.Username != "" {
		// TODO: We are not yet mapping users to Kubernetes SAs, so anyone we could
		// authenticate currently gets all the same permissions as the Kargo API
		// server itself.
		//
		// We should replace this with a SubjectAccessReview on the SA to determine
		// if the desired operation is allowed. If it is, we can return the
		// internalClient.
		return internalClient, nil
	}

	// If we get to here, we're dealing with a user who "authenticated" by just
	// passing their bearer token for the Kubernetes API server.

	// Calling getUserClient will build a user-specific client and conduct a
	// SelfSubjectAccessReview. If the review succeeds, we'll have obtained a
	// client that can perform the desired operation. If the review fails, the
	// returned client will be nil.
	client, err := getUserClient(
		ctx,
		internalClient.Scheme(),
		userInfo.BearerToken,
		authv1.ResourceAttributes{
			Verb:        verb,
			Group:       gvr.Group,
			Version:     gvr.Version,
			Resource:    gvr.Resource,
			Subresource: subresource,
			Namespace:   key.Namespace,
			Name:        key.Name,
		},
	)
	if err != nil {
		return nil,
			errors.Wrap(err, "error getting user-specific Kubernetes client")
	}
	if client == nil {
		return nil, errors.New("not allowed")
	}
	// Note: At present, we never return the user-specific client. We always
	// return the internal client so that most operations are performed by a
	// single client with a built-in cache.
	return internalClient, nil
}

// getUserClient creates a user-specific client using the provided scheme and
// bearer token. It then uses that client to conduct a SelfSubjectAccessReview
// to determine whether the token provided permits the desired operation. If so,
// the user-specific client is returned. Otherwise, nil is returned.
func getUserClient(
	ctx context.Context,
	scheme *runtime.Scheme,
	bearerToken string,
	resourceAttrs authv1.ResourceAttributes,
) (libClient.Client, error) {
	cfg, err := GetRestConfig(ctx, os.Getenv("KUBECONFIG"))
	if err != nil {
		return nil, errors.Wrap(err, "error getting REST config")
	}

	// Tweak the REST config...
	cfg.BearerToken = bearerToken
	// These MUST be blanked out because they all seem to take precedence over the
	// cfg.BearerToken field.
	// TODO: Are there more things to blank out here?
	cfg.BearerTokenFile = ""
	cfg.CertData = nil
	cfg.CertFile = ""

	userClient, err := libClient.New(
		cfg,
		libClient.Options{
			Scheme: scheme,
		},
	)
	if err != nil {
		return nil,
			errors.Wrap(err, "error creating user-specific Kubernetes client")
	}
	ssar := &authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &resourceAttrs,
		},
	}
	if err = userClient.Create(ctx, ssar); err != nil {
		return nil, errors.Wrap(err, "error conducting SelfSubjectAccessReview")
	}
	if ssar.Status.Allowed {
		return userClient, nil
	}
	return nil, nil
}
