package external

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	gh "github.com/google/go-github/v76/github"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/projectconfigs/wire"
)

// fakeLookup is a receiverLookup returning a canned result, recording the
// paths it was asked for.
type fakeLookup struct {
	res   *wire.WebhookReceiverLookup
	err   error
	paths []string
}

func (f *fakeLookup) LookupWebhookReceiver(
	_ context.Context,
	path string,
) (*wire.WebhookReceiverLookup, error) {
	f.paths = append(f.paths, path)
	return f.res, f.err
}

func TestServer_route(t *testing.T) {
	const testPath = "/nonsense"
	testURL, err := url.JoinPath("https://webhooks.kargo.example.com", testPath)
	require.NoError(t, err)

	const testProjectName = "fake-project"
	const testReceiverName = "fake-receiver"

	testScheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(testScheme))
	require.NoError(t, kargoapi.AddToScheme(testScheme))

	// projectReceiverClient returns a client holding the Secret of a GitHub
	// receiver of testProjectName.
	projectReceiverClient := func() client.Client {
		return fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProjectName,
					Name:      "fake-secret",
				},
				Data: map[string][]byte{GithubSecretDataKey: []byte(testSigningKey)},
			},
		).Build()
	}

	// projectReceiverLookup returns a lookup resolving testPath to the GitHub
	// receiver of testProjectName.
	projectReceiverLookup := &fakeLookup{res: &wire.WebhookReceiverLookup{
		Project: testProjectName,
		Config: kargoapi.WebhookReceiverConfig{
			Name: testReceiverName,
			GitHub: &kargoapi.GitHubWebhookReceiverConfig{
				SecretRef: corev1.LocalObjectReference{Name: "fake-secret"},
			},
		},
		Receiver: kargoapi.WebhookReceiverDetails{
			Name: testReceiverName,
			Path: testPath,
		},
	}}

	testCases := []struct {
		name          string
		server        *server
		requestLabels []string
		// req defaults to an empty POST to testURL.
		req        func() *http.Request
		assertions func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:          "error looking up the receiver",
			requestLabels: []string{"", "unknown", "500"},
			server: &server{
				lookup: &fakeLookup{err: errors.New("something went wrong")},
				// ClusterConfig must not be consulted when the lookup fails.
				client: fake.NewClientBuilder().WithScheme(testScheme).
					WithInterceptorFuncs(interceptor.Funcs{
						Get: func(
							context.Context,
							client.WithWatch,
							client.ObjectKey,
							client.Object,
							...client.GetOption,
						) error {
							t.Error("ClusterConfig read after a failed lookup")
							return nil
						},
					}).Build(),
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rr.Result().StatusCode)
				require.JSONEq(t, "{}", rr.Body.String())
			},
		},
		{
			name:          "error getting ClusterConfig",
			requestLabels: []string{"", "unknown", "500"},
			server: &server{
				lookup: &fakeLookup{},
				client: fake.NewClientBuilder().WithScheme(testScheme).WithInterceptorFuncs(
					interceptor.Funcs{
						Get: func(
							context.Context,
							client.WithWatch,
							client.ObjectKey,
							client.Object,
							...client.GetOption,
						) error {
							return errors.New("something went wrong")
						},
					},
				).Build(),
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rr.Result().StatusCode)
				require.JSONEq(t, "{}", rr.Body.String())
			},
		},
		{
			name:          "no ClusterConfig found",
			requestLabels: []string{"", "unknown", "404"},
			server: &server{
				lookup: &fakeLookup{},
				client: fake.NewClientBuilder().WithScheme(testScheme).Build(),
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, rr.Result().StatusCode)
				require.JSONEq(t, "{}", rr.Body.String())
			},
		},
		{
			name:          "error getting receiver config",
			requestLabels: []string{testProjectName, "unknown", "404"},
			server: &server{
				// The config does not match the receiver details.
				lookup: &fakeLookup{res: &wire.WebhookReceiverLookup{
					Project: testProjectName,
					Config:  kargoapi.WebhookReceiverConfig{Name: "other"},
					Receiver: kargoapi.WebhookReceiverDetails{
						Name: testReceiverName,
						Path: testPath,
					},
				}},
				client: fake.NewClientBuilder().WithScheme(testScheme).Build(),
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, rr.Result().StatusCode)
				require.JSONEq(t, "{}", rr.Body.String())
			},
		},
		{
			name:          "success with ProjectConfig",
			requestLabels: []string{testProjectName, github, "400"},
			server: &server{
				lookup: projectReceiverLookup,
				client: projectReceiverClient(),
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				// The X-GitHub-Event header is not set. If we get back a 400 bad
				// request, it means we successfully routed the request to the a
				// gitHubWebhookReceiver.
				require.Equal(t, http.StatusBadRequest, rr.Result().StatusCode)
				require.Contains(t, projectReceiverLookup.paths, testPath)
			},
		},
		{
			name:          "GitHub ping",
			requestLabels: []string{testProjectName, github, "200"},
			server: &server{
				lookup: projectReceiverLookup,
				client: projectReceiverClient(),
			},
			req: func() *http.Request {
				body := []byte("{}")
				req := httptest.NewRequest(http.MethodPost, testURL, bytes.NewReader(body))
				req.Header.Set(gh.EventTypeHeader, githubEventTypePing)
				req.Header.Set(gh.SHA256SignatureHeader, sign(body))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Result().StatusCode)
			},
		},
		{
			name:          "GitHub invalid signature",
			requestLabels: []string{testProjectName, github, "401"},
			server: &server{
				lookup: projectReceiverLookup,
				client: projectReceiverClient(),
			},
			req: func() *http.Request {
				req := httptest.NewRequest(http.MethodPost, testURL, bytes.NewReader([]byte("{}")))
				req.Header.Set(gh.EventTypeHeader, githubEventTypePing)
				req.Header.Set(gh.SHA256SignatureHeader, "totally-invalid-signature")
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, rr.Result().StatusCode)
			},
		},
		{
			name:          "success with ClusterConfig",
			requestLabels: []string{"", github, "400"},
			server: &server{
				lookup: &fakeLookup{},
				cfg: ServerConfig{
					SystemResourcesNamespace: "fake-namespace",
				},
				client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
					&kargoapi.ClusterConfig{
						ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
						Spec: kargoapi.ClusterConfigSpec{
							WebhookReceivers: []kargoapi.WebhookReceiverConfig{{
								Name: testReceiverName,
								GitHub: &kargoapi.GitHubWebhookReceiverConfig{
									SecretRef: corev1.LocalObjectReference{
										Name: "fake-secret",
									},
								},
							}},
						},
						Status: kargoapi.ClusterConfigStatus{
							WebhookReceivers: []kargoapi.WebhookReceiverDetails{{
								Name: testReceiverName,
								Path: testPath,
							}},
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "fake-namespace",
							Name:      "fake-secret",
						},
						Data: map[string][]byte{GithubSecretDataKey: []byte("my-super-secret-token")},
					},
				).Build(),
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				// The X-GitHub-Event header is not set. If we get back a 400 bad
				// request, it means we successfully routed the request to the a
				// gitHubWebhookReceiver.
				require.Equal(t, http.StatusBadRequest, rr.Result().StatusCode)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			// succinct way of setting the apiReader to the same fake client
			// since the fake client doesn't use the cache at all.
			testCase.server.apiReader = testCase.server.client
			counter := prometheus.NewCounterVec(
				prometheus.CounterOpts{Name: "test_requests_total"},
				[]string{"project", "receiverType", "code"},
			)
			saved := requests
			requests = counter
			t.Cleanup(func() { requests = saved })
			req := httptest.NewRequest(http.MethodPost, testURL, nil)
			if testCase.req != nil {
				req = testCase.req()
			}
			testCase.server.route(rr, req)
			testCase.assertions(t, rr)
			require.Equal(t, 1, testutil.CollectAndCount(counter))
			require.Equal(
				t,
				1.0,
				testutil.ToFloat64(counter.WithLabelValues(testCase.requestLabels...)),
			)
		})
	}
}
