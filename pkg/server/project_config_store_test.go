package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	authnv1 "k8s.io/api/authentication/v1"
	authv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/database/projectconfigstore"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/projectconfigs"
	"github.com/akuity/kargo/pkg/server/user"
)

// fakeProjectConfigStore serves ProjectConfigs from memory, for the handlers
// that read them and for the generic resources endpoints, which write them.
// The operations none of these use are left to the embedded interface, which
// is nil, so calling them fails the test.
type fakeProjectConfigStore struct {
	projectconfigstore.Store
	// configs holds the ProjectConfigs, keyed by Project name.
	configs map[string]projectconfigstore.Config
	// err, when set, fails every call.
	err error
}

func (s *fakeProjectConfigStore) Get(
	_ context.Context,
	project string,
	projectUID string,
) (projectconfigstore.Config, error) {
	if s.err != nil {
		return projectconfigstore.Config{}, s.err
	}
	cfg, ok := s.configs[project]
	if !ok || cfg.ProjectUID != projectUID {
		return projectconfigstore.Config{}, fmt.Errorf("ProjectConfig %q: %w", project, database.ErrNotFound)
	}
	return cfg, nil
}

func (s *fakeProjectConfigStore) Create(
	_ context.Context,
	cfg *kargoapi.ProjectConfig,
	projectUID string,
) (projectconfigstore.Config, error) {
	if s.err != nil {
		return projectconfigstore.Config{}, s.err
	}
	if current, ok := s.configs[cfg.Name]; ok && current.ProjectUID == projectUID {
		return projectconfigstore.Config{}, fmt.Errorf("ProjectConfig %q: %w", cfg.Name, database.ErrAlreadyExists)
	}
	return s.put(cfg, projectUID, 1), nil
}

func (s *fakeProjectConfigStore) Update(
	ctx context.Context,
	cfg *kargoapi.ProjectConfig,
	projectUID string,
) (projectconfigstore.Config, error) {
	current, err := s.Get(ctx, cfg.Name, projectUID)
	if err != nil {
		return projectconfigstore.Config{}, err
	}
	return s.put(cfg, projectUID, current.Generation+1), nil
}

func (s *fakeProjectConfigStore) Delete(ctx context.Context, project, projectUID string) error {
	if _, err := s.Get(ctx, project, projectUID); err != nil {
		return err
	}
	delete(s.configs, project)
	return nil
}

// put stores a ProjectConfig for the Project with the given UID.
func (s *fakeProjectConfigStore) put(
	cfg *kargoapi.ProjectConfig,
	projectUID string,
	generation int64,
) projectconfigstore.Config {
	data, err := json.Marshal(cfg.Spec)
	if err != nil {
		panic(err)
	}
	if s.configs == nil {
		s.configs = map[string]projectconfigstore.Config{}
	}
	stored := projectconfigstore.Config{ProjectConfigRow: database.ProjectConfigRow{
		ProjectName: cfg.Name,
		ProjectUID:  projectUID,
		Labels:      cfg.Labels,
		Annotations: cfg.Annotations,
		Spec:        data,
		Generation:  generation,
	}}
	s.configs[cfg.Name] = stored
	return stored
}

// withProjectConfig returns a serverSetup that gives the server a store
// holding a ProjectConfig with the given spec for the Project with the given
// name and UID.
func withProjectConfig(
	project *kargoapi.Project,
	spec kargoapi.ProjectConfigSpec,
) func(*testing.T, *server) {
	return func(t *testing.T, s *server) {
		data, err := json.Marshal(spec)
		require.NoError(t, err)
		s.projectConfigs = &fakeProjectConfigStore{
			configs: map[string]projectconfigstore.Config{
				project.Name: {ProjectConfigRow: database.ProjectConfigRow{
					ProjectName: project.Name,
					ProjectUID:  string(project.UID),
					Spec:        data,
				}},
			},
		}
	}
}

// testResourceProject returns a Project with the given name and UID, as a
// manifest sent to the generic resources endpoints would carry it.
func testResourceProject(name, uid string) *kargoapi.Project {
	return &kargoapi.Project{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kargoapi.GroupVersion.String(),
			Kind:       "Project",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			UID:  types.UID(uid),
		},
	}
}

// testResourceProjectConfig returns a valid ProjectConfig manifest for the
// Project with the given name.
func testResourceProjectConfig(project string) *projectconfigs.Resource {
	return projectconfigs.ResponseBody(&kargoapi.ProjectConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      project,
			Namespace: project,
		},
		Spec: kargoapi.ProjectConfigSpec{
			FreightLinks: []kargoapi.DeepLink{{
				Title: "t",
				URL:   "https://example.com",
			}},
		},
	})
}

// staleResourceProjectConfig returns the resource a client that read the
// Project's ProjectConfig before it was deleted would write back.
func staleResourceProjectConfig(project string) *projectconfigs.Resource {
	res := testResourceProjectConfig(project)
	res.UID = "deleted-uid"
	res.ResourceVersion = "7"
	return res
}

// storeHolding returns a store holding a ProjectConfig for each of the given
// Projects.
func storeHolding(projects ...*kargoapi.Project) *fakeProjectConfigStore {
	store := &fakeProjectConfigStore{}
	for _, project := range projects {
		store.put(
			&kargoapi.ProjectConfig{ObjectMeta: metav1.ObjectMeta{Name: project.Name}},
			string(project.UID),
			1,
		)
	}
	return store
}

// withProjectConfigStore returns a serverSetup that gives the server the
// given store, and optionally an authorizeFn, and records the store in
// *captured for assertions to inspect.
func withProjectConfigStore(
	store *fakeProjectConfigStore,
	captured **fakeProjectConfigStore,
	authorizeFn func(
		context.Context,
		string,
		schema.GroupVersionResource,
		string,
		client.ObjectKey,
	) error,
) func(*testing.T, *server) {
	return func(_ *testing.T, s *server) {
		s.projectConfigs = store
		*captured = store
		if authorizeFn != nil {
			s.authorizeFn = authorizeFn
		}
	}
}

// forbidProjectConfigs is an authorizeFn that refuses every access to
// ProjectConfigs and grants every other.
func forbidProjectConfigs(
	_ context.Context,
	_ string,
	gvr schema.GroupVersionResource,
	_ string,
	key client.ObjectKey,
) error {
	if gvr.Resource == "projectconfigs" {
		return apierrors.NewForbidden(
			gvr.GroupResource(),
			key.Name,
			errors.New("not permitted"),
		)
	}
	return nil
}

// createdProjectUID is the UID assigningProjectUIDs gives each Project.
const createdProjectUID = "new-uid"

// assigningProjectUIDs returns a client builder whose client gives each
// Project it creates createdProjectUID, as the Kubernetes API server gives
// each a UID of its own. It starts out holding objs.
func assigningProjectUIDs(objs ...client.Object) *fake.ClientBuilder {
	return fake.NewClientBuilder().WithObjects(objs...).WithInterceptorFuncs(interceptor.Funcs{
		Create: func(
			ctx context.Context,
			c client.WithWatch,
			obj client.Object,
			opts ...client.CreateOption,
		) error {
			if obj.GetObjectKind().GroupVersionKind().Kind == "Project" {
				obj.SetUID(types.UID(createdProjectUID))
			}
			return c.Create(ctx, obj, opts...)
		},
	})
}

// requireNoProjectConfigInKubernetes fails the test if a ProjectConfig for
// the named Project reached Kubernetes.
func requireNoProjectConfigInKubernetes(t *testing.T, c client.Client, project string) {
	t.Helper()
	// The ProjectConfig kind is not registered with any scheme, so it is
	// read as an unstructured object.
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(kargoapi.GroupVersion.WithKind("ProjectConfig"))
	err := c.Get(
		t.Context(),
		client.ObjectKey{Namespace: project, Name: project},
		obj,
	)
	require.True(t, apierrors.IsNotFound(err), "unexpected error: %v", err)
}

// projectScopedUser is an identity whose access comes only from
// project-scoped RBAC: it holds no verb on Projects themselves.
var projectScopedUser = user.KubernetesUser{
	UserInfo: authnv1.UserInfo{Username: "project-scoped-user"},
}

// withClientForbiddingProjects returns a serverSetup that gives the server an
// authorizing client which answers SubjectAccessReviews like project-scoped
// RBAC: every access is granted except any access to Projects. The client
// starts out holding objs. It also gives the server store, recording it in
// *captured for assertions to inspect.
func withClientForbiddingProjects(
	store *fakeProjectConfigStore,
	captured **fakeProjectConfigStore,
	objs ...client.Object,
) func(*testing.T, *server) {
	return func(t *testing.T, s *server) {
		scheme := newTestScheme(t)
		require.NoError(t, authv1.AddToScheme(scheme))
		internalClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithRESTMapper(testRESTMapper(scheme)).
			WithObjects(objs...).
			WithInterceptorFuncs(interceptor.Funcs{
				Create: func(
					ctx context.Context,
					c client.WithWatch,
					obj client.Object,
					opts ...client.CreateOption,
				) error {
					review, ok := obj.(*authv1.SubjectAccessReview)
					if !ok {
						return c.Create(ctx, obj, opts...)
					}
					review.Status.Allowed = review.Spec.ResourceAttributes.Resource != "projects"
					return nil
				},
			}).
			Build()
		c, err := kubernetes.NewClient(
			t.Context(),
			&rest.Config{},
			kubernetes.ClientOptions{
				NewInternalClient: func(
					context.Context,
					*rest.Config,
					*runtime.Scheme,
					string,
				) (client.WithWatch, error) {
					return internalClient, nil
				},
			},
		)
		require.NoError(t, err)
		s.client = c
		s.authorizeFn = c.Authorize
		s.projectConfigs = store
		*captured = store
	}
}

// asProjectScopedUser is a ctxSetup that makes projectScopedUser the caller.
func asProjectScopedUser(ctx context.Context) context.Context {
	return user.ContextWithIdentity(ctx, projectScopedUser)
}
