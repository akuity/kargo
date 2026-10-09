package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	rollouts "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/rbac"
)

const (
	testKargoNamespace           = "kargo"
	testSystemResourcesNamespace = "kargo-system-resources"
	testSharedResourcesNamespace = "kargo-shared-resources"
)

type restTestCase struct {
	name          string
	url           string
	body          io.Reader
	headers       map[string]string
	clientBuilder *fake.ClientBuilder
	serverConfig  *config.ServerConfig
	// serverSetup is an optional function that can be used to perform additional
	// case-specific server initialization.
	serverSetup func(*testing.T, *server)
	// ctxSetup optionally transforms the request context before the request
	// is served. Use this to inject context-bound values like a user.Identity.
	ctxSetup   func(context.Context) context.Context
	assertions func(*testing.T, *httptest.ResponseRecorder, client.Client)
}

// clusterScopedTestKinds mirrors the +kubebuilder:resource:scope=Cluster
// markers in api/v1alpha1, plus ClusterAnalysisTemplate (cluster-scoped in
// its own upstream Argo Rollouts CRD). meta.NewDefaultRESTMapper can't infer
// scope from a scheme, so getting this wrong would make every kind look
// namespaced to fake clients and mask scope-dependent bugs in tests.
var clusterScopedTestKinds = map[string]bool{
	"Project":                 true,
	"ClusterConfig":           true,
	"ClusterPromotionTask":    true,
	"ClusterAnalysisTemplate": true,
}

// testRESTMapper builds a RESTMapper from a scheme so fake clients can resolve
// a resource type's API group and scope the way the production
// discovery-backed mapper does. It mirrors a real cluster by omitting
// rbac.kargo.akuity.io types, which are virtual API objects rather than
// resources the cluster actually serves.
func testRESTMapper(s *runtime.Scheme) meta.RESTMapper {
	m := meta.NewDefaultRESTMapper(nil)
	for gvk := range s.AllKnownTypes() {
		if strings.HasSuffix(gvk.Kind, "List") || gvk.Group == rbacapi.GroupVersion.Group {
			continue
		}
		scope := meta.RESTScopeNamespace
		if clusterScopedTestKinds[gvk.Kind] {
			scope = meta.RESTScopeRoot
		}
		m.Add(gvk, scope)
	}
	return m
}

// newTestScheme builds the scheme used by the REST test harness, and by
// tests that build their own kubernetes.Client (e.g. to control
// authorization the harness's SkipAuthorization: true bypasses).
func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	testScheme := runtime.NewScheme()

	// k8s APIs
	require.NoError(t, corev1.AddToScheme(testScheme))
	require.NoError(t, coordinationv1.AddToScheme(testScheme))
	require.NoError(t, rbacv1.AddToScheme(testScheme))

	// Kargo APIs
	require.NoError(t, kargoapi.AddToScheme(testScheme))
	require.NoError(t, rbacapi.AddToScheme(testScheme))

	// Third-party APIs
	require.NoError(t, rollouts.AddToScheme(testScheme))

	return testScheme
}

func testRESTEndpoint(
	t *testing.T,
	serverCfg *config.ServerConfig,
	method string,
	url string,
	testCases []restTestCase,
) {
	testScheme := newTestScheme(t)

	gin.DefaultWriter = io.Discard
	gin.DefaultErrorWriter = io.Discard

	if serverCfg == nil {
		serverCfg = &config.ServerConfig{}
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			s := &server{cfg: *serverCfg}

			if testCase.serverConfig != nil {
				s.cfg = *testCase.serverConfig
			}

			if testCase.clientBuilder == nil {
				testCase.clientBuilder = fake.NewClientBuilder()
			}
			internalClient := testCase.clientBuilder.
				WithScheme(testScheme).
				WithRESTMapper(testRESTMapper(testScheme)).
				Build()
			var err error
			s.client, err = kubernetes.NewClient(
				t.Context(),
				&rest.Config{},
				kubernetes.ClientOptions{
					SkipAuthorization: true,
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
			s.authorizeFn = s.client.Authorize
			s.rolesDB = rbac.NewKubernetesRolesDatabase(
				s.client,
				s.client,
				rbac.RolesDatabaseConfig{KargoNamespace: testKargoNamespace},
			)
			// Mirror the Fn wiring NewServer performs; serverSetup may override.
			s.createPromotionFn = s.client.Create
			s.getStageFn = api.GetStage
			s.patchFreightStatusFn = s.patchFreightStatus

			if testCase.serverSetup != nil {
				testCase.serverSetup(t, s)
			}

			u := url
			if testCase.url != "" {
				u = testCase.url
			}
			w := httptest.NewRecorder()
			req := httptest.NewRequest(method, u, testCase.body)
			if testCase.ctxSetup != nil {
				req = req.WithContext(testCase.ctxSetup(req.Context()))
			}
			for key, value := range testCase.headers {
				req.Header.Set(key, value)
			}
			router, err := s.setupRESTRouter(t.Context())
			require.NoError(t, err)

			router.ServeHTTP(w, req)

			testCase.assertions(t, w, internalClient)
		})
	}
}

// restWatchTestCase represents a test case for a REST watch endpoint that uses
// SSE.
type restWatchTestCase struct {
	name          string
	url           string
	headers       map[string]string
	clientBuilder *fake.ClientBuilder
	serverConfig  *config.ServerConfig
	// operations is an optional function that performs operations on the client
	// asynchronously after the watch has been established. This allows tests to
	// trigger events (Create, Update, Delete) that the watch will observe.
	operations func(context.Context, client.Client)
	assertions func(*testing.T, *httptest.ResponseRecorder, client.Client)
}

// testRESTWatchEndpoint tests a REST endpoint that supports SSE watch
// functionality. It follows the same pattern as testRESTEndpoint but is
// specialized for watch/streaming endpoints. Watch endpoints always use GET.
func testRESTWatchEndpoint(
	t *testing.T,
	serverCfg *config.ServerConfig,
	url string,
	testCases []restWatchTestCase,
) {
	testScheme := newTestScheme(t)

	gin.DefaultWriter = io.Discard
	gin.DefaultErrorWriter = io.Discard

	if serverCfg == nil {
		serverCfg = &config.ServerConfig{}
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			s := &server{cfg: *serverCfg}

			if testCase.serverConfig != nil {
				s.cfg = *testCase.serverConfig
			}

			if testCase.clientBuilder == nil {
				testCase.clientBuilder = fake.NewClientBuilder()
			}
			internalClient := testCase.clientBuilder.
				WithScheme(testScheme).
				WithRESTMapper(testRESTMapper(testScheme)).
				Build()
			watching := &watchStartedClient{
				WithWatch: internalClient,
				started:   make(chan struct{}),
			}
			var err error
			s.client, err = kubernetes.NewClient(
				t.Context(),
				&rest.Config{},
				kubernetes.ClientOptions{
					SkipAuthorization: true,
					NewInternalClient: func(
						context.Context,
						*rest.Config,
						*runtime.Scheme,
						string,
					) (client.WithWatch, error) {
						return watching, nil
					},
				},
			)
			require.NoError(t, err)
			s.authorizeFn = s.client.Authorize
			s.rolesDB = rbac.NewKubernetesRolesDatabase(
				s.client,
				s.client,
				rbac.RolesDatabaseConfig{KargoNamespace: testKargoNamespace},
			)

			u := url
			if testCase.url != "" {
				u = testCase.url
			}
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, u, nil)
			for key, value := range testCase.headers {
				req.Header.Set(key, value)
			}

			// A watch handler streams until its request is canceled. Operations
			// run only once the handler has started its watch, so that the watch
			// sees them however long routing took, and the request is then given
			// a fixed window to stream the events. A handler that fails before
			// watching returns on its own; the fallback stops one that neither
			// watches nor returns.
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			req = req.WithContext(ctx)
			go func() {
				defer cancel()
				select {
				case <-watching.started:
				case <-time.After(watchStartTimeout):
					return
				case <-ctx.Done():
					return
				}
				if testCase.operations != nil {
					testCase.operations(ctx, internalClient)
				}
				select {
				case <-time.After(watchStreamWindow):
				case <-ctx.Done():
				}
			}()

			router, err := s.setupRESTRouter(t.Context())
			require.NoError(t, err)
			router.ServeHTTP(w, req)

			testCase.assertions(t, w, internalClient)
		})
	}
}

const (
	// watchStartTimeout bounds how long a watch test waits for the handler
	// to start watching before it cancels the request.
	watchStartTimeout = 5 * time.Second
	// watchStreamWindow is how long a watch test lets the handler stream
	// after its operations have run.
	watchStreamWindow = 100 * time.Millisecond
)

// watchStartedClient closes started once the handler under test has
// registered its first watch, so that a test's operations are never made
// before the watch can observe them.
type watchStartedClient struct {
	client.WithWatch
	once    sync.Once
	started chan struct{}
}

func (c *watchStartedClient) Watch(
	ctx context.Context,
	list client.ObjectList,
	opts ...client.ListOption,
) (watch.Interface, error) {
	w, err := c.WithWatch.Watch(ctx, list, opts...)
	c.once.Do(func() { close(c.started) })
	return w, err
}

// mustJSONBody marshals the given value to JSON and returns it as an io.Reader.
// It panics if marshaling fails.
func mustJSONBody(v any) io.Reader {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return bytes.NewReader(b)
}

// mustYAMLBody marshals objects to YAML and returns it as an io.Reader.
// Multiple objects are separated by "---". It panics if marshaling fails.
func mustYAMLBody(objs ...any) io.Reader {
	return bytes.NewReader(mustYAML(objs...))
}

// mustJSONArrayBody marshals objects as a JSON array and returns it as an io.Reader.
// It panics if marshaling fails.
func mustJSONArrayBody(objs ...any) io.Reader {
	var parts []string
	for _, obj := range objs {
		b, err := json.Marshal(obj)
		if err != nil {
			panic(err)
		}
		parts = append(parts, string(b))
	}
	return strings.NewReader("[" + strings.Join(parts, ",") + "]")
}
