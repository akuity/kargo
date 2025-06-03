package external

import (
	"bytes"
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

func TestQuayHandler(t *testing.T) {
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
			name: "request too large",
			kClient: func() client.Client {
				scheme := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(scheme))
				require.NoError(t, kargoapi.AddToScheme(scheme))
				return fake.NewClientBuilder().
					WithScheme(scheme).
					Build()
			},
			req: func() *http.Request {
				const maxBytes = 2 << 20 // 2MB
				body := make([]byte, maxBytes+1)
				b := io.NopCloser(bytes.NewBuffer(body))
				req := httptest.NewRequest(http.MethodPost, url, b)
				return req
			},
			msg:  "{\"error\":\"failed to read request body: content exceeds limit of 2097152 bytes\"}\n",
			code: http.StatusRequestEntityTooLarge,
		},
		{
			name: "missing payload url",
			kClient: func() client.Client {
				scheme := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(scheme))
				require.NoError(t, kargoapi.AddToScheme(scheme))
				return fake.NewClientBuilder().
					WithScheme(scheme).
					Build()
			},
			req: func() *http.Request {
				b := bytes.NewBuffer([]byte(`{}`))
				req := httptest.NewRequest(http.MethodPost, url, b)
				return req
			},
			msg:  "{\"error\":\"missing repository web URL in request body\"}\n",
			code: http.StatusBadRequest,
		},
		{
			name: "success",
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
								kargoapi.WebhookReceiverSecretKeyQuay: []byte("mysupersecrettoken"),
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
										Quay: &kargoapi.QuayWebhookReceiver{
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
											kargoapi.WebhookReceiverTypeQuay,
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
										Image: &kargoapi.ImageSubscription{
											RepoURL: "quay.io/mynamespace/repository",
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
				b := newQuayPayload()
				req := httptest.NewRequest(http.MethodPost, url, b)
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
			h := quayHandler(
				test.kClient(),
				namespace,
			)
			h(w, req)
			require.Equal(t, test.code, w.Code)
			require.Contains(t, w.Body.String(), test.msg)
		})
	}
}

func newQuayPayload() *bytes.Buffer {
	return bytes.NewBuffer([]byte(`
		{
			"name": "repository",
			"repository": "mynamespace/repository",
			"namespace": "mynamespace",
			"docker_url": "quay.io/mynamespace/repository",
			"homepage": "https://quay.io/repository/mynamespace/repository",
			"updated_tags": [
			  "latest"
			]
		  }
	`))
}
