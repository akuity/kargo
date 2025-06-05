package external

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/indexer"
)

func TestGithubHandler(t *testing.T) {
	const testURL = "https://webhooks.kargo.example.com/nonsense"

	const testProjectName = "fake-project"

	testScheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(testScheme))

	const testToken = "mysupersecrettoken"
	testSecretData := map[string][]byte{
		"token": []byte(testToken),
	}

	testCases := []struct {
		name       string
		client     client.Client
		secretData map[string][]byte
		baseURL    string
		req        func() *http.Request
		assertions func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "token missing from Secret data",
			req: func() *http.Request {
				return httptest.NewRequest(http.MethodPost, testURL, nil)
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rr.Code)
				require.JSONEq(t, "{}", rr.Body.String())
			},
		},
		{
			name:       "unsupported event type",
			secretData: testSecretData,
			req: func() *http.Request {
				req := httptest.NewRequest(http.MethodPost, testURL, nil)
				req.Header.Set("X-GitHub-Event", "nonsense")
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotImplemented, rr.Code)
				require.JSONEq(
					t,
					`{"error":"event type nonsense is not supported"}`,
					rr.Body.String(),
				)
			},
		},
		{
			name:       "request body too large",
			secretData: testSecretData,
			req: func() *http.Request {
				body := make([]byte, 2<<20+1)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					io.NopCloser(bytes.NewBuffer(body)),
				)
				req.Header.Set("X-GitHub-Event", "push")
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusRequestEntityTooLarge, rr.Code)
				res := map[string]string{}
				err := json.Unmarshal(rr.Body.Bytes(), &res)
				require.NoError(t, err)
				require.Contains(t, res["error"], "content exceeds limit")
			},
		},
		{
			name:       "missing signature",
			secretData: testSecretData,
			req: func() *http.Request {
				req := httptest.NewRequest(http.MethodPost, testURL, nil)
				req.Header.Set("X-GitHub-Event", "push")
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, rr.Code)
				require.JSONEq(t, `{"error":"missing signature"}`, rr.Body.String())
			},
		},
		{
			name:       "invalid signature",
			secretData: testSecretData,
			req: func() *http.Request {
				req := httptest.NewRequest(http.MethodPost, testURL, nil)
				req.Header.Set("X-GitHub-Event", "push")
				req.Header.Set("X-Hub-Signature-256", "totally-invalid-signature")
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, rr.Code)
				require.JSONEq(t, `{"error":"unauthorized"}`, rr.Body.String())
			},
		},
		{
			name:       "malformed request body",
			secretData: testSecretData,
			req: func() *http.Request {
				bodyBuf := bytes.NewBuffer([]byte("invalid json"))
				req := httptest.NewRequest(http.MethodPost, testURL, bodyBuf)
				req.Header.Set("X-Hub-Signature-256", sign(testToken, bodyBuf.Bytes()))
				req.Header.Set("X-GitHub-Event", "push")
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
				require.JSONEq(t, `{"error":"invalid request body"}`, rr.Body.String())
			},
		},
		{
			name:       "success -- ping event",
			secretData: testSecretData,
			req: func() *http.Request {
				bodyBuf := bytes.NewBuffer([]byte("{}"))
				req := httptest.NewRequest(http.MethodPost, testURL, bodyBuf)
				req.Header.Set(
					"X-Hub-Signature-256",
					sign("mysupersecrettoken", bodyBuf.Bytes()),
				)
				req.Header.Set("X-GitHub-Event", "ping")
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(
					t,
					`{"msg":"ping event received, webhook is configured correctly"}`,
					rr.Body.String(),
				)
			},
		},
		{
			name:       "success -- push event",
			secretData: testSecretData,
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "fake-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{
								RepoURL: "https://github.com/example/repo",
							},
						}},
					},
				},
			).WithIndex(
				&kargoapi.Warehouse{},
				indexer.WarehousesBySubscribedURLsField,
				indexer.WarehousesBySubscribedURLs,
			).Build(),
			req: func() *http.Request {
				bodyBuf := bytes.NewBuffer(
					[]byte(`{
		"ref": "refs/heads/main",
		"before": "1fe030abc48d0d0ee7b3d650d6e9449775990318",
		"after": "f12cd167152d80c0a2e28cb45e827c6311bba910",
		"repository": {
		  "html_url": "https://github.com/example/repo"
		},
		"pusher": {
		  "name": "username",
		  "email": "email@inbox.com"
		},
		"head_commit": {
		  "id": "f12cd167152d80c0a2e28cb45e827c6311bba910"
		}
		}`),
				)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bodyBuf,
				)
				req.Header.Set(
					"X-Hub-Signature-256",
					sign("mysupersecrettoken", bodyBuf.Bytes()),
				)
				req.Header.Set("X-GitHub-Event", "push")
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(
					t,
					`{"msg":"refreshed 1 warehouse(s)"}`,
					rr.Body.String(),
				)
			},
		},
		{
			name:       "enterprise host mismatch",
			secretData: testSecretData,
			baseURL:    "https://github.enterprise.com",
			req: func() *http.Request {
				bodyBuf := bytes.NewBuffer([]byte("{}"))
				req := httptest.NewRequest(http.MethodPost, testURL, bodyBuf)
				req.Header.Set("X-GitHub-Event", "ping")
				req.Header.Set("X-Hub-Signature-256", sign(testToken, bodyBuf.Bytes()))
				req.Header.Set("X-GitHub-Enterprise-Host", "wrong.enterprise.com")
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, rr.Code)
				require.JSONEq(
					t,
					`{"error":"invalid GitHub Enterprise host: got wrong.enterprise.com, want github.enterprise.com"}`,
					rr.Body.String(),
				)
			},
		},
		{
			name:       "enterprise host match",
			secretData: testSecretData,
			baseURL:    "https://github.enterprise.com",
			req: func() *http.Request {
				bodyBuf := bytes.NewBuffer([]byte("{}"))
				req := httptest.NewRequest(http.MethodPost, testURL, bodyBuf)
				req.Header.Set("X-GitHub-Event", "ping")
				req.Header.Set("X-Hub-Signature-256", sign(testToken, bodyBuf.Bytes()))
				req.Header.Set("X-GitHub-Enterprise-Host", "github.enterprise.com")
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(
					t,
					`{"msg":"ping event received, webhook is configured correctly"}`,
					rr.Body.String(),
				)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			(&githubWebhookReceiver{
				baseWebhookReceiver: &baseWebhookReceiver{
					client:     testCase.client,
					project:    testProjectName,
					secretData: testCase.secretData,
				},
				baseURL: testCase.baseURL,
			}).GetHandler()(w, testCase.req())
			testCase.assertions(t, w)
		})
	}
}

func sign(s string, b []byte) string {
	mac := hmac.New(sha256.New, []byte(s))
	_, _ = mac.Write(b)
	return fmt.Sprintf("sha256=%s",
		hex.EncodeToString(mac.Sum(nil)),
	)
}
