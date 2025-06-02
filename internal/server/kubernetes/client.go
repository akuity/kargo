package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	authv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/cli-utils/pkg/flowcontrol"
	libClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	libCluster "sigs.k8s.io/controller-runtime/pkg/cluster"

	rollouts "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/indexer"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/internal/server/user"
)

// ClientOptions specifies options for customizing the client returned by the
// NewClient function.
type ClientOptions struct {
	// SkipAuthorization, if true, will cause the implementation of the Client
	// interface to bypass efforts to authorize the Kargo API user's authority to
	// perform any desired operation, in which case, such operations are
	// unconditionally executed using the implementation's own internal client.
	// This does NOT bypass authorization entirely. The Kargo API server will
	// still be constrained by the permissions of the Kubernetes user from whose
	// configuration the internal client was constructed. This option is useful
	// for scenarios where the Kargo API server is executed locally on a user's
	// system and the user wished to provide the API server with their own
	// Kubernetes client configuration. This is used, for instance, by the
	// `kargo server` command.
	SkipAuthorization bool
	// GlobalServiceAccountNamespaces is a list of namespaces in which we should
	// always look for ServiceAccounts when attempting to authorize a user.
	GlobalServiceAccountNamespaces []string
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
			return opts, fmt.Errorf("error adding Kubernetes core API to scheme: %w", err)
		}
		if err := rbacv1.AddToScheme(opts.Scheme); err != nil {
			return opts, fmt.Errorf("error adding Kubernetes RBAC API to scheme: %w", err)
		}
		if err := rollouts.AddToScheme(opts.Scheme); err != nil {
			return opts, fmt.Errorf("error adding Argo Rollouts API to Kargo API manager scheme: %w", err)
		}
		if err := kargoapi.AddToScheme(opts.Scheme); err != nil {
			return opts, fmt.Errorf("error adding Kargo API to scheme: %w", err)
		}
	}
	if opts.NewInternalClient == nil {
		opts.NewInternalClient = newDefaultInternalClient
	}
	if opts.NewInternalDynamicClient == nil {
		opts.NewInternalDynamicClient = func(c *rest.Config) (dynamic.Interface, error) {
			return dynamic.NewForConfig(c)
		}
	}
	return opts, nil
}

// The Client interface combines the familiar controller-runtime Client
// interface with helpful Authorized and Watch functions that are absent from
// that interface.
type Client interface {
	libClient.Client

	// Authorize attempts to authorize the user to perform the desired operation
	// on the specified resource. If the user is not authorized, an error is
	// returned.
	Authorize(
		ctx context.Context,
		verb string,
		gvr schema.GroupVersionResource,
		subresource string,
		key libClient.ObjectKey,
	) error

	// InternalClient returns the internal controller-runtime client used by this
	// client. This is useful for cases where the API server needs to bypass
	// the extra authorization checks performed by this client.
	InternalClient() libClient.Client

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
	internalDynamicClient dynamic.Interface
	opts                  ClientOptions

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
		return nil, fmt.Errorf("error setting client options defaults: %w", err)
	}
	internalClient, err :=
		opts.NewInternalClient(ctx, restCfg, opts.Scheme)
	if err != nil {
		return nil, fmt.Errorf("error building internal client: %w", err)
	}
	internalDynamicClient, err :=
		opts.NewInternalDynamicClient(restCfg)
	if err != nil {
		return nil, fmt.Errorf("error building internal dynamic client: %w", err)
	}
	c := &client{
		internalClient:        internalClient,
		internalDynamicClient: internalDynamicClient,
		opts:                  opts,
	}
	if opts.SkipAuthorization {
		c.getAuthorizedClientFn = func(
			context.Context,
			libClient.Client,
			string,
			schema.GroupVersionResource,
			string,
			libClient.ObjectKey,
		) (libClient.Client, error) {
			return internalClient, nil // Unconditionally return the internal client
		}
	} else {
		// Examine the context-bound user.Info to determine what ServiceAccounts
		// they are associated with and whether any of those have sufficient
		// permissions to perform the desired operation.
		c.getAuthorizedClientFn = getAuthorizedClient(opts.GlobalServiceAccountNamespaces)
	}
	return c, nil
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
			clusterOptions.Client = libClient.Options{
				Cache: &libClient.CacheOptions{
					DisableFor: []libClient.Object{
						&corev1.Secret{},
					},
				},
			}
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating controller-runtime cluster: %w", err)
	}

	// Add all indices required by the API server
	if err = cluster.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.Promotion{},
		indexer.PromotionsByStageField,
		indexer.PromotionsByStage,
	); err != nil {
		return nil, fmt.Errorf("error indexing Promotions by Stage: %w", err)
	}
	if err = cluster.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.Freight{},
		indexer.FreightByWarehouseField,
		indexer.FreightByWarehouse,
	); err != nil {
		return nil, fmt.Errorf("error indexing Freight by Warehouse: %w", err)
	}
	if err = cluster.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.Freight{},
		indexer.FreightByVerifiedStagesField,
		indexer.FreightByVerifiedStages,
	); err != nil {
		return nil, fmt.Errorf("error indexing Freight by Stages in which it has been verified: %w", err)
	}
	if err = cluster.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.Freight{},
		indexer.FreightApprovedForStagesField,
		indexer.FreightApprovedForStages,
	); err != nil {
		return nil, fmt.Errorf("error indexing Freight by Stages for which it has been approved: %w", err)
	}
	if err = cluster.GetFieldIndexer().IndexField(
		ctx,
		&corev1.ServiceAccount{},
		indexer.ServiceAccountsByOIDCClaimsField,
		indexer.ServiceAccountsByOIDCClaims,
	); err != nil {
		return nil, fmt.Errorf("index ServiceAccounts by OIDC claims: %w", err)
	}
	if err = cluster.GetFieldIndexer().IndexField(
		ctx,
		&corev1.Event{},
		indexer.EventsByInvolvedObjectAPIGroupField,
		indexer.EventsByInvolvedObjectAPIGroup,
	); err != nil {
		return nil, fmt.Errorf("error indexing Events by InvolvedObject's API group: %w", err)
	}

	go func() {
		err = cluster.Start(ctx)
	}()
	if !cluster.GetCache().WaitForCacheSync(ctx) {
		return nil, errors.New("error waiting for cache sync")
	}
	if err != nil {
		return nil, fmt.Errorf("error starting cluster: %w", err)
	}
	return cluster.GetClient(), nil
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
		if clOpt, ok := opt.(*libClient.ListOptions); ok {
			key.Namespace = clOpt.Namespace
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
		if clOpt, ok := opt.(*libClient.DeleteAllOfOptions); ok {
			key.Namespace = clOpt.Namespace
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
	return c.SubResource("status")
}

func (c *client) SubResource(subResource string) libClient.SubResourceClient {
	return &authorizingSubResourceClient{
		subResourceType:       subResource,
		internalClient:        c.internalClient,
		getAuthorizedClientFn: c.getAuthorizedClientFn,
	}
}

func (c *client) GroupVersionKindFor(
	obj runtime.Object,
) (schema.GroupVersionKind, error) {
	return c.internalClient.GroupVersionKindFor(obj)
}

func (c *client) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	return c.internalClient.IsObjectNamespaced(obj)
}

func (c *client) Scheme() *runtime.Scheme {
	return c.internalClient.Scheme()
}

func (c *client) RESTMapper() meta.RESTMapper {
	return c.internalClient.RESTMapper()
}

// authorizingSubResourceClient implements libClient.SubResourceClient.
type authorizingSubResourceClient struct {
	subResourceType string

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

func (a *authorizingSubResourceClient) Get(
	ctx context.Context,
	obj libClient.Object,
	subResource libClient.Object,
	opts ...libClient.SubResourceGetOption,
) error {
	gvr, key, err := gvrAndKeyFromObj(obj, obj, a.internalClient.Scheme())
	if err != nil {
		return err
	}
	client, err := a.getAuthorizedClientFn(
		ctx,
		a.internalClient,
		"get",
		gvr,
		a.subResourceType,
		*key,
	)
	if err != nil {
		return err
	}
	return client.SubResource(a.subResourceType).
		Get(ctx, obj, subResource, opts...)
}

func (a *authorizingSubResourceClient) Create(
	ctx context.Context,
	obj libClient.Object,
	subResource libClient.Object,
	opts ...libClient.SubResourceCreateOption,
) error {
	gvr, key, err := gvrAndKeyFromObj(obj, obj, a.internalClient.Scheme())
	if err != nil {
		return err
	}
	client, err := a.getAuthorizedClientFn(
		ctx,
		a.internalClient,
		"create",
		gvr,
		a.subResourceType,
		*key,
	)
	if err != nil {
		return err
	}
	return client.SubResource(a.subResourceType).
		Create(ctx, obj, subResource, opts...)
}

func (a *authorizingSubResourceClient) Update(
	ctx context.Context,
	obj libClient.Object,
	opts ...libClient.SubResourceUpdateOption,
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
		a.subResourceType,
		*key,
	)
	if err != nil {
		return err
	}
	return client.SubResource(a.subResourceType).Update(ctx, obj, opts...)
}

func (a *authorizingSubResourceClient) Patch(
	ctx context.Context,
	obj libClient.Object,
	patch libClient.Patch,
	opts ...libClient.SubResourcePatchOption,
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
		a.subResourceType,
		*key,
	)
	if err != nil {
		return err
	}
	return client.SubResource(a.subResourceType).Patch(ctx, obj, patch, opts...)
}

func (c *client) Authorize(
	ctx context.Context,
	verb string,
	gvr schema.GroupVersionResource,
	subresource string,
	key libClient.ObjectKey,
) error {
	if _, err := c.getAuthorizedClientFn(
		ctx,
		c.internalClient,
		verb,
		gvr,
		subresource,
		key,
	); err != nil {
		return err
	}
	return nil
}

func (c *client) InternalClient() libClient.Client {
	return c.internalClient
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
		if err != nil {
			return cfg, fmt.Errorf("error loading in-cluster REST config: %w", err)
		}
		return cfg, nil
	}

	logger.Debug(
		"loading REST config from path",
		"path", path,
	)
	cfg, err := clientcmd.BuildConfigFromFlags("", path)
	if err != nil {
		return cfg, fmt.Errorf("error loading REST config from %q: %w", path, err)
	}
	return cfg, nil
}

// ConfigureQPSBurst configures the provided REST config to use the specified
// QPS and burst values, unless PriorityAndFairness flow control is enabled in
// the cluster, in which case, it disables QPS and burst.
//
// For more information on PriorityAndFairness flow control, see:
// https://kubernetes.io/docs/concepts/cluster-administration/flow-control/
func ConfigureQPSBurst(ctx context.Context, cfg *rest.Config, qps float32, burst int) {
	logger := logging.LoggerFromContext(ctx)

	if ok, err := flowcontrol.IsEnabled(ctx, cfg); err != nil && ok {
		logger.Debug("PriorityAndFairness flow control is enabled; disabling QPS and burst")
		cfg.QPS = -1
		cfg.Burst = -1
		return
	}

	logger.Debug("configuring QPS and burst for REST config", "qps", qps, "burst", burst)
	cfg.QPS = qps
	cfg.Burst = burst
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
		return schema.GroupVersionResource{}, nil, fmt.Errorf("error extracting GVK from object: %w", err)
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
func getAuthorizedClient(globalServiceAccountNamespaces []string) func(
	context.Context,
	libClient.Client,
	string,
	schema.GroupVersionResource,
	string,
	libClient.ObjectKey,
) (libClient.Client, error) {
	return func(
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

		ra := authv1.ResourceAttributes{
			Verb:        verb,
			Group:       gvr.Group,
			Version:     gvr.Version,
			Resource:    gvr.Resource,
			Subresource: subresource,
			Namespace:   key.Namespace,
			Name:        key.Name,
		}

		// sub is a standard claim. If the user has this claim, we can infer that
		// they authenticated using OIDC.
		if _, ok := userInfo.Claims["sub"]; ok {
			var namespacesToCheck []string
			if key.Namespace != "" {
				// This is written the way it is to keep key.Namespace as the first
				// element in the slice, so it is checked first, because this is where
				// there is the highest likelihood of finding a ServiceAccount with
				// the required permissions.
				namespacesToCheck = make([]string, 0, 1+len(globalServiceAccountNamespaces))
				namespacesToCheck = append(namespacesToCheck, key.Namespace)
				namespacesToCheck = append(namespacesToCheck, globalServiceAccountNamespaces...)
			} else {
				// Check ONLY globalServiceAccountNamespaces. i.e. We will NOT check
				// project namespaces to find suitable ServiceAccounts for dealing with
				// cluster-scoped resources.
				namespacesToCheck = globalServiceAccountNamespaces
			}
			for _, namespaceToCheck := range namespacesToCheck {
				serviceAccountsToCheck := userInfo.ServiceAccountsByNamespace[namespaceToCheck]
				for serviceAccountToCheck := range serviceAccountsToCheck {
					err := reviewSubjectAccess(
						ctx,
						internalClient.Scheme(),
						ra,
						withServiceAccount(serviceAccountToCheck),
					)
					if err == nil {
						return internalClient, nil
					}
					if !apierrors.IsForbidden(err) {
						return nil, fmt.Errorf("review subject access: %w", err)
					}
				}
			}
		}

		// If we get to here, we're dealing with a user who "authenticated" by just
		// passing their bearer token for the Kubernetes API server.
		if err := reviewSubjectAccess(
			ctx,
			internalClient.Scheme(),
			ra,
			withBearerToken(userInfo.BearerToken),
		); err != nil {
			return nil, fmt.Errorf("review subject access: %w", err)
		}
		return internalClient, nil
	}
}

type subjectOption func(*userClientOptions)

func withBearerToken(bearerToken string) subjectOption {
	return func(opts *userClientOptions) {
		opts.bearerToken = bearerToken
	}
}

func withServiceAccount(name types.NamespacedName) subjectOption {
	return func(opts *userClientOptions) {
		opts.subject = &userClientSubject{
			username: fmt.Sprintf("system:serviceaccount:%s:%s", name.Namespace, name.Name),
		}
	}
}

type userClientSubject struct {
	username string
}

type userClientOptions struct {
	bearerToken string
	subject     *userClientSubject
}

// reviewSubjectAccess submits a (Self)SubjectAccessReview to determine
// whether the subject that configured with subjectOption is allowed to
// do the desired operation.
func reviewSubjectAccess(
	ctx context.Context,
	scheme *runtime.Scheme,
	ra authv1.ResourceAttributes,
	opts ...subjectOption,
) error {
	cfg, err := GetRestConfig(ctx, os.Getenv("KUBECONFIG"))
	if err != nil {
		return fmt.Errorf("get REST config: %w", err)
	}

	var opt userClientOptions
	for _, apply := range opts {
		apply(&opt)
	}

	if opt.bearerToken != "" {
		cfg.BearerToken = opt.bearerToken
		// These MUST be blanked out because they all seem to take precedence over the
		// cfg.BearerToken field.
		// TODO: Are there more things to blank out here?
		cfg.BearerTokenFile = ""
		cfg.CertData = nil
		cfg.CertFile = ""
	}

	userClient, err := libClient.New(
		cfg,
		libClient.Options{
			Scheme: scheme,
		},
	)
	if err != nil {
		return fmt.Errorf("create user-specific Kubernetes client: %w", err)
	}

	if opt.subject != nil {
		review := &authv1.SubjectAccessReview{
			Spec: authv1.SubjectAccessReviewSpec{
				ResourceAttributes: &ra,
				User:               opt.subject.username,
			},
		}
		if err := userClient.Create(ctx, review); err != nil {
			return fmt.Errorf("submit SubjectAccessReview: %w", err)
		}
		if review.Status.Allowed {
			return nil
		}
		return newForbiddenError(ra)
	}

	review := &authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &ra,
		},
	}
	if err := userClient.Create(ctx, review); err != nil {
		return fmt.Errorf("submit SelfSubjectAccessReview: %w", err)
	}
	if review.Status.Allowed {
		return nil
	}
	return newForbiddenError(ra)
}

func newForbiddenError(ra authv1.ResourceAttributes) error {
	return apierrors.NewForbidden(
		schema.GroupResource{
			Group:    ra.Group,
			Resource: ra.Resource,
		},
		ra.Name,
		fmt.Errorf("%s is not permitted", ra.Verb),
	)
}
