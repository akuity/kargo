package external

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/indexer"
	"github.com/akuity/kargo/internal/logging"
)

const testSecret = "testsecret" // nolint: gosec

func TestGithubHandler(t *testing.T) {
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
				b := newGitHubEventBody("push")
				req := httptest.NewRequest(http.MethodPost, url, b)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Hub-Signature-256", sign(t, testSecret, b.Bytes()))
				req.Header.Set("X-GitHub-Event", "push")
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
				b := newGitHubEventBody("push")
				req := httptest.NewRequest(http.MethodPost, url, b)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Hub-Signature-256", sign(t, "fakesecret", b.Bytes()))
				req.Header.Set("X-GitHub-Event", "push")
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
				b := newGitHubEventBody("deployment")
				req := httptest.NewRequest(http.MethodPost, url, b)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Hub-Signature-256", sign(t, testSecret, b.Bytes()))
				req.Header.Set("X-GitHub-Event", "deployment")
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
				req.Header.Set("X-Hub-Signature-256", sign(t, testSecret, body))
				req.Header.Set("X-GitHub-Event", "push")
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
				b := newGitHubEventBody("push")
				req := httptest.NewRequest(http.MethodPost, url, b)
				req.Header.Set("X-GitHub-Event", "push")
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
				b := newGitHubEventBody("push")
				req := httptest.NewRequest(http.MethodPost, url, b)
				req.Header.Set("X-Hub-Signature-256", sign(t, "invalid-sig", b.Bytes()))
				req.Header.Set("X-GitHub-Event", "push")
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
				req.Header.Set("X-Hub-Signature-256", sign(t, "mysupersecrettoken", b.Bytes()))
				req.Header.Set("X-GitHub-Event", "push")
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
			},
			req: func() *http.Request {
				b := newGitHubEventBody("push")
				req := httptest.NewRequest(http.MethodPost, url, b)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Hub-Signature-256", sign(t, "mysupersecrettoken", b.Bytes()))
				req.Header.Set("X-GitHub-Event", "push")
				return req
			},
			secret: "fakesecret",
			msg:    "{\"msg\":\"refreshed 1 warehouse(s)\"}\n",
			code:   http.StatusOK,
		},
		{
			name: "success - ping event",
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
					).
					Build()
			},
			req: func() *http.Request {
				b := newGitHubEventBody("ping")
				req := httptest.NewRequest(http.MethodPost, url, b)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Hub-Signature-256", sign(t, "mysupersecrettoken", b.Bytes()))
				req.Header.Set("X-GitHub-Event", "ping")
				return req
			},
			secret: "fakesecret",
			msg:    "{\"msg\":\"ping event received, webhook is configured correctly for https://github.com/username/repo\"}\n", // nolint: lll
			code:   http.StatusOK,
		},
		{
			name: "success - package event",
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
											RepoURL: "ghcr.io/username/repo",
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
				b := newGitHubEventBody("package")
				req := httptest.NewRequest(http.MethodPost, url, b)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Hub-Signature-256", sign(t, "mysupersecrettoken", b.Bytes()))
				req.Header.Set("X-GitHub-Event", "package")
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
			h := githubHandler(test.kClient(), namespace, test.secret)
			h(w, req)
			require.Equal(t, test.code, w.Code)
			require.Contains(t, w.Body.String(), test.msg)
		})
	}
}

func sign(t *testing.T, s string, b []byte) string {
	t.Helper()

	mac := hmac.New(sha256.New, []byte(s))
	_, _ = mac.Write(b)
	return fmt.Sprintf("sha256=%s",
		hex.EncodeToString(mac.Sum(nil)),
	)
}

func newGitHubEventBody(eventType string) *bytes.Buffer {
	switch eventType {
	case "push", "ping", "deployment":
		return bytes.NewBuffer([]byte(`
{
	"ref": "refs/heads/main",
	"before": "1fe030abc48d0d0ee7b3d650d6e9449775990318",
	"after": "f12cd167152d80c0a2e28cb45e827c6311bba910",
	"repository": {
		"html_url": "https://github.com/username/repo"
	},
	"pusher": {
		"name": "username",
		"email": "email@inbox.com"
	},
	"head_commit": {
		"id": "f12cd167152d80c0a2e28cb45e827c6311bba910"
	}
}
`))
	case "package":
		return bytes.NewBuffer([]byte(`
{
  "action": "published",
  "package": {
    "id": 7656404,
    "name": "repo",
    "namespace": "username",
    "description": "",
    "ecosystem": "CONTAINER",
    "package_type": "CONTAINER",
    "html_url": "https://github.com/username/packages/7656404",
    "created_at": "2025-02-10T10:47:09Z",
    "updated_at": "2025-06-04T07:37:38Z",
    "owner": {
      "login": "username",
      "id": 86051118,
      "node_id": "MDQ6VXNlcjg2MDUxMTE4",
      "avatar_url": "https://avatars.githubusercontent.com/u/86051118?v=4",
      "gravatar_id": "",
      "url": "https://api.github.com/users/username",
      "html_url": "https://github.com/username",
      "followers_url": "https://api.github.com/users/username/followers",
      "following_url": "https://api.github.com/users/username/following{/other_user}",
      "gists_url": "https://api.github.com/users/username/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/username/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/username/subscriptions",
      "organizations_url": "https://api.github.com/users/username/orgs",
      "repos_url": "https://api.github.com/users/username/repos",
      "events_url": "https://api.github.com/users/username/events{/privacy}",
      "received_events_url": "https://api.github.com/users/username/received_events",
      "type": "User",
      "user_view_type": "public",
      "site_admin": false
    }
  },
  "package_version": {
    "id": 430232,
    "version": "sha256:ceebe538c5f4c3b32dba99b9c3b1d7401d6815db8d7240e8854c87b570a991e9",
    "name": "sha256:ceebe538c5f4c3b32dba99b9c3b1d7401d6815db8d7240e8854c87b570a991e9",
    "package_url": "ghcr.io/username/repo:v0.1.6",
    "body": {
      "repository": {
        "repository": {
          "name": "repo",
          "owner_login": "username"
        }
      }
    }
  },
  "sender": {
    "login": "username",
    "id": 86051118,
    "node_id": "MDQ6VXNlcjg2MDUxMTE4",
    "avatar_url": "https://avatars.githubusercontent.com/u/86051118?v=4",
    "gravatar_id": "",
    "url": "https://api.github.com/users/username",
    "html_url": "https://github.com/username",
    "followers_url": "https://api.github.com/users/username/followers",
    "following_url": "https://api.github.com/users/username/following{/other_user}",
    "gists_url": "https://api.github.com/users/username/gists{/gist_id}",
    "starred_url": "https://api.github.com/users/username/starred{/owner}{/repo}",
    "subscriptions_url": "https://api.github.com/users/username/subscriptions",
    "organizations_url": "https://api.github.com/users/username/orgs",
    "repos_url": "https://api.github.com/users/username/repos",
    "events_url": "https://api.github.com/users/username/events{/privacy}",
    "received_events_url": "https://api.github.com/users/username/received_events",
    "type": "User",
    "user_view_type": "public",
    "site_admin": false
  }
}
`))
	default:
		return bytes.NewBuffer([]byte(`{}`))
	}
}
