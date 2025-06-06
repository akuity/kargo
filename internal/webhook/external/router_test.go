package external

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/indexer"
)

func TestServer_route(t *testing.T) {
	const testPath = "/nonsense"
	testURL, err := url.JoinPath("https://webhooks.kargo.example.com", testPath)
	require.NoError(t, err)

	const testProjectName = "fake-project"
	const testReceiverName = "fake-receiver"

	testScheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(testScheme))
	require.NoError(t, kargoapi.AddToScheme(testScheme))

	testCases := []struct {
		name       string
		server     *server
		assertions func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "error listing ProjectConfigs",
			server: &server{
				client: fake.NewClientBuilder().WithScheme(testScheme).
					WithInterceptorFuncs(interceptor.Funcs{
						List: func(
							context.Context,
							client.WithWatch,
							client.ObjectList,
							...client.ListOption,
						) error {
							return errors.New("something went wrong")
						},
					}).Build(),
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rr.Result().StatusCode)
				require.JSONEq(t, "{}", rr.Body.String())
			},
		},
		{
			name: "error getting ClusterConfig",
			server: &server{
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
				).WithIndex(
					&kargoapi.ProjectConfig{},
					indexer.ProjectConfigsByWebhookReceiverPathsField,
					indexer.ProjectConfigsByWebhookReceiverPaths,
				).Build(),
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rr.Result().StatusCode)
				require.JSONEq(t, "{}", rr.Body.String())
			},
		},
		{
			name: "no ClusterConfig found",
			server: &server{
				client: fake.NewClientBuilder().WithScheme(testScheme).WithIndex(
					&kargoapi.ProjectConfig{},
					indexer.ProjectConfigsByWebhookReceiverPathsField,
					indexer.ProjectConfigsByWebhookReceiverPaths,
				).Build(),
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, rr.Result().StatusCode)
				require.JSONEq(t, "{}", rr.Body.String())
			},
		},
		{
			name: "error getting receiver config",
			server: &server{
				client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
					&kargoapi.ProjectConfig{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProjectName,
							Name:      testProjectName,
						},
						// There is no config matching the receiver in the status
						Spec: kargoapi.ProjectConfigSpec{},
						Status: kargoapi.ProjectConfigStatus{
							WebhookReceivers: []kargoapi.WebhookReceiverDetails{{
								Name: testReceiverName,
								Path: testPath,
							}},
						},
					},
				).WithIndex(
					&kargoapi.ProjectConfig{},
					indexer.ProjectConfigsByWebhookReceiverPathsField,
					indexer.ProjectConfigsByWebhookReceiverPaths,
				).Build(),
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, rr.Result().StatusCode)
				require.JSONEq(t, "{}", rr.Body.String())
			},
		},
		{
			name: "success with ProjectConfig",
			server: &server{
				client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
					&kargoapi.ProjectConfig{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProjectName,
							Name:      testProjectName,
						},
						Spec: kargoapi.ProjectConfigSpec{
							WebhookReceivers: []kargoapi.WebhookReceiverConfig{{
								Name: testReceiverName,
								GitHub: &kargoapi.GitHubWebhookReceiverConfig{
									SecretRef: corev1.LocalObjectReference{Name: "fake-secret"},
								},
							}},
						},
						Status: kargoapi.ProjectConfigStatus{
							WebhookReceivers: []kargoapi.WebhookReceiverDetails{{
								Name: testReceiverName,
								Path: testPath,
							}},
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProjectName,
							Name:      "fake-secret",
						},
						Data: map[string][]byte{GithubSecretDataKey: []byte("my-super-secret-token")},
					},
				).WithIndex(
					&kargoapi.ProjectConfig{},
					indexer.ProjectConfigsByWebhookReceiverPathsField,
					indexer.ProjectConfigsByWebhookReceiverPaths,
				).Build(),
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				// The X-GitHub-Event header is not set. If we get back a 501 not
				// implemented, it means we successfully routed the request to the
				// a gitHubWebhookReceiver.
				require.Equal(t, http.StatusNotImplemented, rr.Result().StatusCode)
			},
		},
		{
			name: "success with ClusterConfig",
			server: &server{
				cfg: ServerConfig{
					ClusterSecretsNamespace: "fake-namespace",
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
				).WithIndex(
					&kargoapi.ProjectConfig{},
					indexer.ProjectConfigsByWebhookReceiverPathsField,
					indexer.ProjectConfigsByWebhookReceiverPaths,
				).Build(),
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				// The X-GitHub-Event header is not set. If we get back a 501 not
				// implemented, it means we successfully routed the request to the
				// a gitHubWebhookReceiver.
				require.Equal(t, http.StatusNotImplemented, rr.Result().StatusCode)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			testCase.server.route(
				rr,
				httptest.NewRequest(http.MethodPost, testURL, nil),
			)
			testCase.assertions(t, rr)
		})
	}
}
