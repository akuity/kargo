package external

import (
	"bytes"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/indexer"
	"github.com/akuity/kargo/internal/logging"
	"github.com/stretchr/testify/require"
	"io"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"net/http"
	"net/http/httptest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestBitbucketHandler(t *testing.T) {
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
										Bitbucket: &kargoapi.BitbucketWebhookReceiver{
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
											kargoapi.WebhookReceiverTypeBitbucket,
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
				b := newBitbucketBody()
				req := httptest.NewRequest(http.MethodPost, url, b)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Hub-Signature", sign(t, testSecret, b.Bytes()))
				req.Header.Set("X-Event-Key", "repo:push")
				req.Header.Set("X-Hook-UUID", "a888a4d3-9e21-4eff-80d4-d73c48aa2aeb")
				return req
			},
			secret: testSecret,
			code:   http.StatusInternalServerError,
			msg:    "{}\n",
		},
		{
			name: "missing bitbucket-secret in secret string data",
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
										Bitbucket: &kargoapi.BitbucketWebhookReceiver{
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
											kargoapi.WebhookReceiverTypeBitbucket,
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
				b := newBitbucketBody()
				req := httptest.NewRequest(http.MethodPost, url, b)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Hub-Signature", sign(t, testSecret, b.Bytes()))
				req.Header.Set("X-Event-Key", "repo:push")
				req.Header.Set("X-Hook-UUID", "a888a4d3-9e21-4eff-80d4-d73c48aa2aeb")
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
								"bitbucket-secret": []byte("mysupersecrettoken"),
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
										Bitbucket: &kargoapi.BitbucketWebhookReceiver{
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
											kargoapi.WebhookReceiverTypeBitbucket,
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
				b := newBitbucketBody()
				req := httptest.NewRequest(http.MethodPost, url, b)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Hub-Signature", sign(t, testSecret, b.Bytes()))
				req.Header.Set("X-Event-Key", "pullrequest:created")
				req.Header.Set("X-Hook-UUID", "a888a4d3-9e21-4eff-80d4-d73c48aa2aeb")
				return req
			},
			secret: "fakesecret",
			msg:    "{\"error\":\"event type pullrequest:created is not supported\"}\n",
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
								"bitbucket-secret": []byte("mysupersecrettoken"),
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
										Bitbucket: &kargoapi.BitbucketWebhookReceiver{
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
											kargoapi.WebhookReceiverSecretKeyBitbucket,
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
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Hub-Signature", sign(t, testSecret, body))
				req.Header.Set("X-Event-Key", "repo:push")
				req.Header.Set("X-Hook-UUID", "a888a4d3-9e21-4eff-80d4-d73c48aa2aeb")
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
								"bitbucket-secret": []byte("mysupersecrettoken"),
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
										Bitbucket: &kargoapi.BitbucketWebhookReceiver{
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
											kargoapi.WebhookReceiverTypeBitbucket,
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
				b := newBitbucketBody()
				req := httptest.NewRequest(http.MethodPost, url, b)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Event-Key", "repo:push")
				req.Header.Set("X-Hook-UUID", "a888a4d3-9e21-4eff-80d4-d73c48aa2aeb")
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
								"bitbucket-secret": []byte("mysupersecrettoken"),
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
										Bitbucket: &kargoapi.BitbucketWebhookReceiver{
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
											kargoapi.WebhookReceiverTypeBitbucket,
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
				b := newBitbucketBody()
				req := httptest.NewRequest(http.MethodPost, url, b)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Hub-Signature", sign(t, "invalid-sig", b.Bytes()))
				req.Header.Set("X-Event-Key", "repo:push")
				req.Header.Set("X-Hook-UUID", "a888a4d3-9e21-4eff-80d4-d73c48aa2aeb")
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
								"bitbucket-secret": []byte("mysupersecrettoken"),
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
										Bitbucket: &kargoapi.BitbucketWebhookReceiver{
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
											kargoapi.WebhookReceiverTypeBitbucket,
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
				req.Header.Set("X-Hub-Signature", sign(t, "mysupersecrettoken", b.Bytes()))
				req.Header.Set("X-Event-Key", "repo:push")
				req.Header.Set("X-Hook-UUID", "a888a4d3-9e21-4eff-80d4-d73c48aa2aeb")
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
								"bitbucket-secret": []byte("mysupersecrettoken"),
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
										Bitbucket: &kargoapi.BitbucketWebhookReceiver{
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
											kargoapi.WebhookReceiverTypeBitbucket,
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
											RepoURL: "https://bitbucket.org/username/repo",
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
				b := newBitbucketBody()
				req := httptest.NewRequest(http.MethodPost, url, b)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Hub-Signature", sign(t, "mysupersecrettoken", b.Bytes()))
				req.Header.Set("X-Event-Key", "repo:push")
				req.Header.Set("X-Hook-UUID", "a888a4d3-9e21-4eff-80d4-d73c48aa2aeb")
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
			h := bitbucketHandler(test.kClient(), namespace, test.secret)
			h(w, req)
			require.Equal(t, test.code, w.Code)
			require.Contains(t, w.Body.String(), test.msg)
		})
	}
}

func newBitbucketBody() *bytes.Buffer {
	return bytes.NewBufferString(`
{
  "push": {
    "changes": [
      {
        "new": {
          "target": {
            "hash": "c12dd29985f86ed53bc93c550841aa58ee7331fb"
          }
        }
      }
    ]
  },
  "repository": {
    "links": {
      "html": {
        "href": "https://bitbucket.org/username/repo"
      }
    }
  }
}	
`)
}
