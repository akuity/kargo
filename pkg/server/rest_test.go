package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	rollouts "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
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
	assertions    func(*testing.T, *httptest.ResponseRecorder, client.Client)
}

func testRESTEndpoint(
	t *testing.T,
	serverCfg *config.ServerConfig,
	method string,
	url string,
	testCases []restTestCase,
) {
	testScheme := runtime.NewScheme()

	// k8s APIs
	err := corev1.AddToScheme(testScheme)
	require.NoError(t, err)
	err = rbacv1.AddToScheme(testScheme)
	require.NoError(t, err)

	// Kargo APIs
	err = kargoapi.AddToScheme(testScheme)
	require.NoError(t, err)
	err = rbacapi.AddToScheme(testScheme)
	require.NoError(t, err)

	// Third-party APIs
	err = rollouts.AddToScheme(testScheme)
	require.NoError(t, err)

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
			internalClient := testCase.clientBuilder.WithScheme(testScheme).Build()
			s.client, err = kubernetes.NewClient(
				t.Context(),
				&rest.Config{},
				kubernetes.ClientOptions{
					SkipAuthorization: true,
					NewInternalClient: func(
						context.Context,
						*rest.Config,
						*runtime.Scheme,
					) (client.WithWatch, error) {
						return internalClient, nil
					},
				},
			)
			require.NoError(t, err)
			s.rolesDB = rbac.NewKubernetesRolesDatabase(
				internalClient,
				rbac.RolesDatabaseConfig{KargoNamespace: testKargoNamespace},
			)

			u := url
			if testCase.url != "" {
				u = testCase.url
			}
			w := httptest.NewRecorder()
			req := httptest.NewRequest(method, u, testCase.body)
			for key, value := range testCase.headers {
				req.Header.Set(key, value)
			}
			router := s.setupRESTRouter(t.Context())

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
	testScheme := runtime.NewScheme()

	// k8s APIs
	err := corev1.AddToScheme(testScheme)
	require.NoError(t, err)
	err = rbacv1.AddToScheme(testScheme)
	require.NoError(t, err)

	// Kargo APIs
	err = kargoapi.AddToScheme(testScheme)
	require.NoError(t, err)
	err = rbacapi.AddToScheme(testScheme)
	require.NoError(t, err)

	// Third-party APIs
	err = rollouts.AddToScheme(testScheme)
	require.NoError(t, err)

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
			internalClient := testCase.clientBuilder.WithScheme(testScheme).Build()
			var err error
			s.client, err = kubernetes.NewClient(
				context.Background(),
				&rest.Config{},
				kubernetes.ClientOptions{
					SkipAuthorization: true,
					NewInternalClient: func(
						context.Context,
						*rest.Config,
						*runtime.Scheme,
					) (client.WithWatch, error) {
						return internalClient, nil
					},
				},
			)
			require.NoError(t, err)
			s.rolesDB = rbac.NewKubernetesRolesDatabase(
				internalClient,
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

			// Create a context with timeout to prevent hanging on watch endpoints
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			req = req.WithContext(ctx)

			// If operations are provided, run them asynchronously after a small delay
			// to allow the watch to be established first
			if testCase.operations != nil {
				go func() {
					time.Sleep(10 * time.Millisecond)
					testCase.operations(ctx, internalClient)
				}()
			}

			router := s.setupRESTRouter(context.Background())
			router.ServeHTTP(w, req)

			testCase.assertions(t, w, internalClient)
		})
	}
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
