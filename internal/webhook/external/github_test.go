package external

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	gh "github.com/google/go-github/v71/github"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/indexer"
)

const githubSigningKey = "mysupersecrettoken"

func TestGithubHandler(t *testing.T) {
	const testURL = "https://webhooks.kargo.example.com/nonsense"

	const testProjectName = "fake-project"

	var validPushEvent = &gh.PushEvent{
		Ref: gh.Ptr("refs/heads/main"),
		Repo: &gh.PushEventRepository{
			CloneURL: gh.Ptr("https://github.com/example/repo"),
		},
	}
	var validPackageEvent = &gh.PackageEvent{
		Action: gh.Ptr("published"),
		Package: &gh.Package{
			PackageType: gh.Ptr(ghcrPackageTypeContainer),
			PackageVersion: &gh.PackageVersion{
				PackageURL: gh.Ptr("ghcr.io/example/repo:latest"),
			},
		},
	}

	testScheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(testScheme))

	testSecretData := map[string][]byte{
		GithubSecretDataKey: []byte(githubSigningKey),
	}

	testCases := []struct {
		name       string
		client     client.Client
		secretData map[string][]byte
		req        func() *http.Request
		assertions func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "signing key (shared secret) missing from Secret data",
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
				req.Header.Set("X-Hub-Signature-256", sign(bodyBuf.Bytes()))
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
				req.Header.Set("X-Hub-Signature-256", sign(bodyBuf.Bytes()))
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
			name:       "unsupported package event action",
			secretData: testSecretData,
			req: func() *http.Request {
				bodyBytes, err := json.Marshal(
					&gh.PackageEvent{Action: gh.Ptr("deleted")},
				)
				require.NoError(t, err)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(bodyBytes),
				)
				req.Header.Set("X-Hub-Signature-256", sign(bodyBytes))
				req.Header.Set("X-GitHub-Event", "package")
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, "{}", rr.Body.String())
			},
		},
		{
			name:       "package event missing package",
			secretData: testSecretData,
			req: func() *http.Request {
				bodyBytes, err := json.Marshal(
					&gh.PackageEvent{Action: gh.Ptr("published")},
				)
				require.NoError(t, err)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(bodyBytes),
				)
				req.Header.Set("X-Hub-Signature-256", sign(bodyBytes))
				req.Header.Set("X-GitHub-Event", "package")
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
				require.JSONEq(t, `{"error":"invalid request body"}`, rr.Body.String())
			},
		},
		{
			name:       "unsupported package type",
			secretData: testSecretData,
			req: func() *http.Request {
				bodyBytes, err := json.Marshal(
					&gh.PackageEvent{
						Action: gh.Ptr("published"),
						Package: &gh.Package{
							PackageType:    gh.Ptr("npm"),
							PackageVersion: &gh.PackageVersion{},
						},
					},
				)
				require.NoError(t, err)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(bodyBytes),
				)
				req.Header.Set("X-Hub-Signature-256", sign(bodyBytes))
				req.Header.Set("X-GitHub-Event", "package")
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, "{}", rr.Body.String())
			},
		},
		{
			name:       "success -- package event",
			secretData: testSecretData,
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "fake-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Image: &kargoapi.ImageSubscription{RepoURL: "ghcr.io/example/repo"},
						}},
					},
				},
			).WithIndex(
				&kargoapi.Warehouse{},
				indexer.WarehousesBySubscribedURLsField,
				indexer.WarehousesBySubscribedURLs,
			).Build(),
			req: func() *http.Request {
				bodyBytes, err := json.Marshal(validPackageEvent)
				require.NoError(t, err)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(bodyBytes),
				)
				req.Header.Set("X-Hub-Signature-256", sign(bodyBytes))
				req.Header.Set("X-GitHub-Event", "package")
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 1 warehouse(s)"}`, rr.Body.String())
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
				bodyBytes, err := json.Marshal(validPushEvent)
				require.NoError(t, err)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(bodyBytes),
				)
				req.Header.Set("X-Hub-Signature-256", sign(bodyBytes))
				req.Header.Set("X-GitHub-Event", "push")
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 1 warehouse(s)"}`, rr.Body.String())
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			requestBody, err := io.ReadAll(testCase.req().Body)
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = testCase.req().Body.Close()
			})

			w := httptest.NewRecorder()
			(&githubWebhookReceiver{
				baseWebhookReceiver: &baseWebhookReceiver{
					client:     testCase.client,
					project:    testProjectName,
					secretData: testCase.secretData,
				},
			}).getHandler(requestBody)(w, testCase.req())

			testCase.assertions(t, w)
		})
	}
}
