package external

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/indexer"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/internal/server/kubernetes"
)

func TestNewServer(t *testing.T) {
	testServerConfig := ServerConfig{}
	testClient, err := kubernetes.NewClient(
		context.Background(),
		&rest.Config{},
		kubernetes.ClientOptions{
			NewInternalClient: func(
				context.Context,
				*rest.Config,
				*runtime.Scheme,
			) (client.Client, error) {
				return fake.NewClientBuilder().Build(), nil
			},
		},
	)
	require.NoError(t, err)

	s, ok := NewServer(testServerConfig, testClient).(*server)
	require.True(t, ok)
	require.NotNil(t, s)
}

func TestRouteHandler(t *testing.T) {
	for _, test := range []struct {
		name  string
		setup func(t *testing.T) *server
		path  string
		code  int
		body  string
	}{
		{
			name: "failed to list project configs",
			setup: func(t *testing.T) *server {
				scheme := runtime.NewScheme()
				require.NoError(t, kargoapi.AddToScheme(scheme))
				kClient := fake.NewClientBuilder().WithScheme(scheme).Build()
				s, ok := NewServer(ServerConfig{}, kClient).(*server)
				require.True(t, ok)
				return s
			},
			path: "/doesntmatter",
			code: http.StatusInternalServerError,
			body: "no index with name receiverPaths has been registered",
		},
		{
			name: "no project configs for the given URL",
			setup: func(t *testing.T) *server {
				scheme := runtime.NewScheme()
				require.NoError(t, kargoapi.AddToScheme(scheme))
				kClient := fake.NewClientBuilder().
					WithScheme(scheme).
					WithIndex(
						&kargoapi.Warehouse{},
						indexer.WarehousesBySubscribedURLsField,
						indexer.WarehousesBySubscribedURLs,
					).
					WithIndex(
						&kargoapi.ProjectConfig{},
						indexer.ProjectConfigsByWebhookReceiverPathsField,
						indexer.ProjectConfigsByWebhookReceiverPaths,
					).
					Build()
				s, ok := NewServer(ServerConfig{}, kClient).(*server)
				require.True(t, ok)
				return s
			},
			path: "/doesntmatter",
			code: http.StatusNotFound,
			body: "no project configs found for the request",
		},
		{
			name: "success",
			setup: func(t *testing.T) *server {
				scheme := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(scheme))
				require.NoError(t, kargoapi.AddToScheme(scheme))
				kClient := fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "fakesecret",
								Namespace: "fakenamespace",
							},
							Data: map[string][]byte{
								"token": []byte("mysupersecrettoken"),
							},
						},
						&kargoapi.ProjectConfig{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "fakenamespace",
								Name:      "fakename",
							},
							Spec: kargoapi.ProjectConfigSpec{
								WebhookReceivers: []kargoapi.WebhookReceiverConfig{
									{
										GitHub: &kargoapi.GitHubWebhookReceiver{
											SecretRef: corev1.LocalObjectReference{
												Name: "fakesecret",
											},
										},
									},
								},
							},
							Status: kargoapi.ProjectConfigStatus{
								WebhookReceivers: []kargoapi.WebhookReceiver{
									{
										Path: GenerateWebhookPath(
											"fake-webhook-receiver-name",
											"fakename",
											kargoapi.WebhookReceiverTypeGitHub,
											"mysupersecrettoken",
										),
									},
								},
							},
						},
						&kargoapi.Warehouse{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "fakenamespace",
								Name:      "fakename",
							},
							Spec: kargoapi.WarehouseSpec{
								Subscriptions: []kargoapi.RepoSubscription{
									{
										Git: &kargoapi.GitSubscription{
											RepoURL: "https://github.com/username/repo",
										},
									},
								},
							},
						},
					).
					WithIndex(
						&kargoapi.Warehouse{},
						indexer.WarehousesBySubscribedURLsField,
						indexer.WarehousesBySubscribedURLs,
					).
					WithIndex(
						&kargoapi.ProjectConfig{},
						indexer.ProjectConfigsByWebhookReceiverPathsField,
						indexer.ProjectConfigsByWebhookReceiverPaths,
					).
					Build()
				s, ok := NewServer(ServerConfig{}, kClient).(*server)
				require.True(t, ok)
				return s
			},
			path: GenerateWebhookPath(
				"fake-webhook-receiver-name",
				"fakename",
				kargoapi.WebhookReceiverTypeGitHub,
				"mysupersecrettoken",
			),
			code: http.StatusOK,
			body: "{\"msg\":\"refreshed 1 warehouse(s)\"}\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			logger := logging.NewLogger(logging.DebugLevel)
			ctx := logging.ContextWithLogger(t.Context(), logger)
			testServer := test.setup(t)
			w := httptest.NewRecorder()
			var body io.Reader
			if test.code == http.StatusOK {
				body = newGitHubEventBody("push")
			}
			req, err := http.NewRequestWithContext(
				ctx,
				http.MethodPost,
				test.path,
				body,
			)
			require.NoError(t, err)
			if test.code == http.StatusOK {
				req.Header.Set("X-GitHub-Event", "push")
				req.Header.Set("X-Hub-Signature-256", sign(t, "mysupersecrettoken", newGitHubEventBody("push").Bytes()))
			}
			testServer.route(w, req)
			require.Equal(t, test.code, w.Result().StatusCode)
			require.Contains(t, w.Body.String(), test.body)
		})
	}
}
