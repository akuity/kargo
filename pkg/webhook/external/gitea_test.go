package external

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/indexer"
)

const giteaWebhookRequestBodyPullRequestClosedMerged = `
{
	"action": "closed",
	"number": 42,
	"pull_request": {
		"merged": true,
		"html_url": "https://gitea.com/example/repo/pulls/42"
	},
	"repository": {
		"clone_url": "https://gitea.com/example/repo.git"
	}
}`

const giteaWebhookRequestBodyPullRequestClosedNotMerged = `
{
	"action": "closed",
	"number": 42,
	"pull_request": {
		"merged": false,
		"html_url": "https://gitea.com/example/repo/pulls/42"
	},
	"repository": {
		"clone_url": "https://gitea.com/example/repo.git"
	}
}`

const giteaWebhookRequestBodyPullRequestOpened = `
{
	"action": "opened",
	"number": 42,
	"repository": {
		"clone_url": "https://gitea.com/example/repo.git"
	}
}`

const giteaWebhookRequestBodyPush = `
{
	"ref": "refs/heads/main",
	"repository": {
		"clone_url": "https://gitea.com/example/repo.git"
	}
}`

const giteaWebhookRequestBodyPushWithCommits = `
{
	"ref": "refs/heads/main",
	"repository": {
		"clone_url": "https://gitea.com/example/repo.git"
	},
	"commits": [
		{
			"added": ["apps/foo/values.yaml"],
			"modified": ["apps/foo/deployment.yaml"],
			"removed": []
		}
	]
}`

const giteaWebhookRequestBodyPushTruncated = `
{
	"ref": "refs/heads/main",
	"repository": {
		"clone_url": "https://gitea.com/example/repo.git"
	},
	"commits": [
		{
			"added": ["apps/bar/values.yaml"],
			"modified": [],
			"removed": []
		}
	],
	"total_commits": 25
}`

func TestGiteaHandler(t *testing.T) {
	const testURL = "https://webhooks.kargo.example.com/nonsense"

	const testProjectName = "fake-project"

	testScheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(testScheme))

	testSecretData := map[string][]byte{
		giteaSecretDataKey: []byte(testSigningKey),
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
				req.Header.Set(giteaEventTypeHeader, "nonsense")
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
				req.Header.Set(giteaEventTypeHeader, giteaEventTypePush)
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
				req.Header.Set(giteaEventTypeHeader, giteaEventTypePush)
				req.Header.Set(giteaSignatureHeader, "totally-invalid-signature")
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
				req.Header.Set(giteaSignatureHeader, sign(bodyBuf.Bytes()))
				req.Header.Set(giteaEventTypeHeader, giteaEventTypePush)
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
				require.JSONEq(t, `{"error":"invalid request body"}`, rr.Body.String())
			},
		},
		{
			name: "no ref match",
			// This event would prompt the Warehouse to refresh if not for the ref in
			// the event being for the main branch whilst the subscription is
			// interested in commits from a different branch.
			secretData: testSecretData,
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "fake-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{
								RepoURL: "https://gitea.com/example/repo",
								Branch:  "not-main", // Constraint won't be met
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
				b := []byte(giteaWebhookRequestBodyPush)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(b),
				)
				req.Header.Set(giteaSignatureHeader, sign(b))
				req.Header.Set(giteaEventTypeHeader, giteaEventTypePush)
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 0 warehouse(s)"}`, rr.Body.String())
			},
		},
		{
			name:       "warehouse refreshed",
			secretData: testSecretData,
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "fake-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{
								RepoURL: "https://gitea.com/example/repo",
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
				b := []byte(giteaWebhookRequestBodyPush)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(b),
				)
				req.Header.Set(giteaSignatureHeader, sign(b))
				req.Header.Set(giteaEventTypeHeader, giteaEventTypePush)
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 1 warehouse(s)"}`, rr.Body.String())
			},
		},
		{
			name:       "path mismatch — warehouse not refreshed",
			secretData: testSecretData,
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "fake-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{
								RepoURL:      "https://gitea.com/example/repo",
								IncludePaths: []string{"glob:apps/bar/**"},
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
				b := []byte(giteaWebhookRequestBodyPushWithCommits)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(b),
				)
				req.Header.Set(giteaSignatureHeader, sign(b))
				req.Header.Set(giteaEventTypeHeader, giteaEventTypePush)
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 0 warehouse(s)"}`, rr.Body.String())
			},
		},
		{
			name:       "path match — warehouse refreshed",
			secretData: testSecretData,
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "fake-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{
								RepoURL:      "https://gitea.com/example/repo",
								IncludePaths: []string{"glob:apps/foo/**"},
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
				b := []byte(giteaWebhookRequestBodyPushWithCommits)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(b),
				)
				req.Header.Set(giteaSignatureHeader, sign(b))
				req.Header.Set(giteaEventTypeHeader, giteaEventTypePush)
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 1 warehouse(s)"}`, rr.Body.String())
			},
		},
		{
			name:       "non-closed pull_request action returns 200 OK",
			secretData: testSecretData,
			req: func() *http.Request {
				b := []byte(giteaWebhookRequestBodyPullRequestOpened)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(b),
				)
				req.Header.Set(giteaSignatureHeader, sign(b))
				req.Header.Set(giteaEventTypeHeader, giteaEventTypePullRequest)
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, "{}", rr.Body.String())
			},
		},
		{
			name:       "closed+merged pull_request refreshes matching Promotion",
			secretData: testSecretData,
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "promo-wait-for-pr",
					},
					Spec: kargoapi.PromotionSpec{
						Stage: "test-stage",
						Steps: []kargoapi.PromotionStep{{
							Uses: "git-wait-for-pr",
							As:   "wait-pr",
						}},
					},
					Status: kargoapi.PromotionStatus{
						Phase:       kargoapi.PromotionPhaseRunning,
						CurrentStep: 0,
						State: &apiextensionsv1.JSON{
							Raw: []byte(`{
								"wait-pr": {
									"pr": {
										"url": "https://gitea.com/example/repo/pulls/42",
										"open": true,
										"merged": false
									}
								}
							}`),
						},
					},
				},
			).WithIndex(
				&kargoapi.Promotion{},
				indexer.RunningPromotionsByPullRequestURLField,
				indexer.RunningPromotionsByPullRequestURL,
			).Build(),
			req: func() *http.Request {
				b := []byte(giteaWebhookRequestBodyPullRequestClosedMerged)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(b),
				)
				req.Header.Set(giteaSignatureHeader, sign(b))
				req.Header.Set(giteaEventTypeHeader, giteaEventTypePullRequest)
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 1 promotion(s)"}`, rr.Body.String())
			},
		},
		{
			name:       "closed+not-merged pull_request refreshes matching Promotion",
			secretData: testSecretData,
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "promo-wait-for-pr",
					},
					Spec: kargoapi.PromotionSpec{
						Stage: "test-stage",
						Steps: []kargoapi.PromotionStep{{
							Uses: "git-wait-for-pr",
							As:   "wait-pr",
						}},
					},
					Status: kargoapi.PromotionStatus{
						Phase:       kargoapi.PromotionPhaseRunning,
						CurrentStep: 0,
						State: &apiextensionsv1.JSON{
							Raw: []byte(`{
								"wait-pr": {
									"pr": {
										"url": "https://gitea.com/example/repo/pulls/42",
										"open": true,
										"merged": false
									}
								}
							}`),
						},
					},
				},
			).WithIndex(
				&kargoapi.Promotion{},
				indexer.RunningPromotionsByPullRequestURLField,
				indexer.RunningPromotionsByPullRequestURL,
			).Build(),
			req: func() *http.Request {
				b := []byte(giteaWebhookRequestBodyPullRequestClosedNotMerged)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(b),
				)
				req.Header.Set(giteaSignatureHeader, sign(b))
				req.Header.Set(giteaEventTypeHeader, giteaEventTypePullRequest)
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 1 promotion(s)"}`, rr.Body.String())
			},
		},
		{
			name:       "closed pull_request with no matching Promotions",
			secretData: testSecretData,
			client: fake.NewClientBuilder().WithScheme(testScheme).WithIndex(
				&kargoapi.Promotion{},
				indexer.RunningPromotionsByPullRequestURLField,
				indexer.RunningPromotionsByPullRequestURL,
			).Build(),
			req: func() *http.Request {
				b := []byte(giteaWebhookRequestBodyPullRequestClosedMerged)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(b),
				)
				req.Header.Set(giteaSignatureHeader, sign(b))
				req.Header.Set(giteaEventTypeHeader, giteaEventTypePullRequest)
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 0 promotion(s)"}`, rr.Body.String())
			},
		},
		{
			name:       "truncated commits — path filtering skipped, warehouse refreshed",
			secretData: testSecretData,
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "fake-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{
								RepoURL:      "https://gitea.com/example/repo",
								IncludePaths: []string{"glob:apps/foo/**"},
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
				b := []byte(giteaWebhookRequestBodyPushTruncated)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(b),
				)
				req.Header.Set(giteaSignatureHeader, sign(b))
				req.Header.Set(giteaEventTypeHeader, giteaEventTypePush)
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
			(&giteaWebhookReceiver{
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
