package external

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/indexer"
	"github.com/akuity/kargo/internal/logging"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGitLabHandler(t *testing.T) {
	url := "http://doesntmatter.com"
	for _, test := range []struct {
		name    string
		kClient func() client.Client
		req     func() *http.Request
		secret  string
		code    int
		msg     string
	}{
		{
			name: "secret not found",
			kClient: func() client.Client {
				scheme := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(scheme))
				require.NoError(t, kargoapi.AddToScheme(scheme))
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&kargoapi.ProjectConfig{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "fakenamespace",
								Name:      "fakename",
							},
							Spec: kargoapi.ProjectConfigSpec{
								WebhookReceivers: []kargoapi.WebhookReceiverConfig{
									{
										GitLab: &kargoapi.GitLabWebhookReceiver{
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
											kargoapi.WebhookReceiverTypeGitlab,
											"fakesecret",
										),
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
			},
			req: func() *http.Request {
				b := newGitlabEventBody()
				req := httptest.NewRequest(http.MethodPost, url, b)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Gitlab-Token", testSecret)
				req.Header.Set("X-Gitlab-Event", "push")
				return req
			},
			secret: testSecret,
			code:   http.StatusInternalServerError,
			msg:    "{}\n", // 500s get obfuscated
		},
		{
			name: "missing token in secret string data",
			kClient: func() client.Client {
				scheme := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(scheme))
				require.NoError(t, kargoapi.AddToScheme(scheme))
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "fakesecret",
								Namespace: "fakenamespace",
							},
							Data: map[string][]byte{
								"not-a-token-key": []byte("doesnt-matter"),
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
										GitLab: &kargoapi.GitLabWebhookReceiver{
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
											kargoapi.WebhookReceiverTypeGitlab,
											"fakesecret",
										),
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
			},
			req: func() *http.Request {
				b := newGitlabEventBody()
				req := httptest.NewRequest(http.MethodPost, url, b)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Gitlab-Token", "fakesecret")
				req.Header.Set("X-Gitlab-Event", "push")
				return req
			},
			secret: "fakesecret",
			code:   http.StatusInternalServerError,
			msg:    "{}\n", // 500s get obfuscated
		},
		{
			name: "bad request - unsupported event type",
			kClient: func() client.Client {
				scheme := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(scheme))
				require.NoError(t, kargoapi.AddToScheme(scheme))
				return fake.NewClientBuilder().
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
										GitLab: &kargoapi.GitLabWebhookReceiver{
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
											kargoapi.WebhookReceiverTypeGitlab,
											"mysupersecrettoken",
										),
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
			},
			req: func() *http.Request {
				b := newGitlabEventBody()
				req := httptest.NewRequest(http.MethodPost, url, b)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Gitlab-Token", testSecret)
				req.Header.Set("X-Gitlab-Event", "deployment")
				return req
			},
			secret: "fakesecret",
			msg:    "{\"error\":\"event type deployment is not supported\"}\n",
			code:   http.StatusNotImplemented,
		},
		{
			name: "request too large",
			kClient: func() client.Client {
				scheme := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(scheme))
				require.NoError(t, kargoapi.AddToScheme(scheme))
				return fake.NewClientBuilder().
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
										GitLab: &kargoapi.GitLabWebhookReceiver{
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
											kargoapi.WebhookReceiverTypeGitlab,
											"mysupersecrettoken",
										),
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
			},
			req: func() *http.Request {
				const maxBytes = 2 << 20 // 2MB
				body := make([]byte, maxBytes+1)
				b := io.NopCloser(bytes.NewBuffer(body))
				req := httptest.NewRequest(http.MethodPost, url, b)
				req.Header.Set("X-Gitlab-Token", sign(t, testSecret, body))
				req.Header.Set("X-Gitlab-Event", "push")
				return req
			},
			secret: "fakesecret",
			msg:    "{\"error\":\"failed to read request body: content exceeds limit of 2097152 bytes\"}\n",
			code:   http.StatusRequestEntityTooLarge,
		},
		{
			name: "unauthorized - missing signature",
			kClient: func() client.Client {
				scheme := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(scheme))
				require.NoError(t, kargoapi.AddToScheme(scheme))
				return fake.NewClientBuilder().
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
										GitLab: &kargoapi.GitLabWebhookReceiver{
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
											kargoapi.WebhookReceiverTypeGitlab,
											"mysupersecrettoken",
										),
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
			},
			req: func() *http.Request {
				b := newGitlabEventBody()
				req := httptest.NewRequest(http.MethodPost, url, b)
				req.Header.Set("X-Gitlab-Event", "push")
				return req
			},
			secret: "fakesecret",
			msg:    "{\"error\":\"missing signature\"}\n",
			code:   http.StatusUnauthorized,
		},
		{
			name: "unauthorized - invalid signature",
			kClient: func() client.Client {
				scheme := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(scheme))
				require.NoError(t, kargoapi.AddToScheme(scheme))
				return fake.NewClientBuilder().
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
										GitLab: &kargoapi.GitLabWebhookReceiver{
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
											kargoapi.WebhookReceiverTypeGitlab,
											"mysupersecrettoken",
										),
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
			},
			req: func() *http.Request {
				b := newGitlabEventBody()
				req := httptest.NewRequest(http.MethodPost, url, b)
				req.Header.Set("X-Gitlab-Token", sign(t, "invalid-sig", b.Bytes()))
				req.Header.Set("X-Gitlab-Event", "push")
				return req
			},
			secret: "fakesecret",
			msg:    "{\"error\":\"unauthorized\"}\n",
			code:   http.StatusUnauthorized,
		},
		{
			name: "malformed request",
			kClient: func() client.Client {
				scheme := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(scheme))
				require.NoError(t, kargoapi.AddToScheme(scheme))
				return fake.NewClientBuilder().
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
										GitLab: &kargoapi.GitLabWebhookReceiver{
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
											kargoapi.WebhookReceiverTypeGitlab,
											"mysupersecrettoken",
										),
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
			},
			req: func() *http.Request {
				b := bytes.NewBuffer([]byte("invalid json"))
				req := httptest.NewRequest(http.MethodPost, url, b)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Gitlab-Token", "mysupersecrettoken")
				req.Header.Set("X-Gitlab-Event", "push")
				return req
			},
			secret: "fakesecret",
			msg:    "{\"error\":\"failed to parse webhook event: invalid character 'i' looking for beginning of value\"}\n",
			code:   http.StatusBadRequest,
		},
		{
			name: "success - push event",
			kClient: func() client.Client {
				scheme := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(scheme))
				require.NoError(t, kargoapi.AddToScheme(scheme))
				return fake.NewClientBuilder().
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
										GitLab: &kargoapi.GitLabWebhookReceiver{
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
											kargoapi.WebhookReceiverTypeGitlab,
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
			},
			req: func() *http.Request {
				b := newGitlabEventBody()
				req := httptest.NewRequest(http.MethodPost, url, b)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Gitlab-Token", "mysupersecrettoken")
				req.Header.Set("X-Gitlab-Event", "push")
				return req
			},
			secret: "fakesecret",
			msg:    "{\"msg\":\"refreshed 1 warehouse(s)\"}\n",
			code:   http.StatusOK,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := test.req()
			l := logging.NewLogger(logging.DebugLevel)
			ctx := logging.ContextWithLogger(req.Context(), l)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()
			namespace := "fakenamespace"
			h := gitlabHandler(test.kClient(), namespace, test.secret)
			h(w, req)
			require.Equal(t, test.code, w.Code)
			require.Contains(t, w.Body.String(), test.msg)
		})
	}
}

func newGitlabEventBody() *bytes.Buffer {
	return bytes.NewBufferString(`
{
  "repository":{
    "url": "git@example.com:mike/diaspora.git",
    "homepage": "http://example.com/mike/diaspora",
    "git_http_url":"http://example.com/mike/diaspora.git",
    "git_ssh_url":"git@example.com:mike/diaspora.git",
  }
  }	
`)
}
