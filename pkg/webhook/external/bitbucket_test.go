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

func TestBitbucketHandler(t *testing.T) {
	const testURL = "https://webhooks.kargo.example.com/nonsense"

	const testProjectName = "fake-project"

	const prFulfilledEventRequestBody = `
	{
		"pullrequest": {
			"id": 42,
			"links": {
				"html": {
					"href": "https://bitbucket.org/example/repo/pull-requests/42"
				}
			}
		},
		"repository": {
			"links": {
				"html": {
					"href": "https://bitbucket.org/example/repo"
				}
			}
		}
	}`

	const prMergedEventRequestBody = `
	{
		"pullRequest": {
			"id": 42,
			"links": {
				"self": [
					{
						"href": "https://bitbucket.example.com/projects/EXAMPLE/repos/repo/pull-requests/42"
					}
				]
			},
			"fromRef": {
				"repository": {
					"links": {
						"clone": [
							{
								"name": "http",
								"href": "https://bitbucket.example.com/scm/example/repo.git"
							},
							{
								"name": "ssh",
								"href": "ssh://git@bitbucket.example.com:7999/example/repo.git"
							}
						]
					}
				}
			}
		}
	}`

	// pushEventRequestBody uses the real Bitbucket Cloud format: bare branch
	// names without a refs/heads/ prefix.
	const pushEventRequestBody = `
	{
		"actor": {
			"name": "admin",
			"emailAddress": "admin@example.com"
		},
		"push": {
			"changes": [{"new": {"type": "branch", "name": "main"}}]
		},
		"repository": {
			"links": {
				"html": {
					"href": "https://bitbucket.org/example/repo"
				}
			}
		}
	}`

	const tagPushEventRequestBody = `
	{
		"actor": {
			"name": "admin",
			"emailAddress": "admin@example.com"
		},
		"push": {
			"changes": [{"new": {"type": "tag", "name": "v1.0.0"}}]
		},
		"repository": {
			"links": {
				"html": {
					"href": "https://bitbucket.org/example/repo"
				}
			}
		}
	}`

	const pushEventRequestBodyBitbucketServer = `
	{
		"actor": {
			"name": "admin",
			"emailAddress": "admin@example.com"
		},
		"changes": [{"ref": {"id": "refs/heads/main"}}],
		"repository": {
			"links": {
				"clone":[{
					"name": "http",
					"href": "https://example.org/bitbucket/scm/example/repo.git"
				},{
					"name": "ssh",
					"href": "ssh://git@bitbucket.example.org:7999/example/repo.git"
				}]
			}
		}
	}`

	testScheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(testScheme))

	testSecretData := map[string][]byte{
		bitbucketSecretDataKey: []byte(testSigningKey),
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
				req.Header.Set(bitbucketEventHeader, "nonsense")
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
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
				req.Header.Set(bitbucketEventHeader, bitbucketPushEvent)
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
				req.Header.Set(bitbucketEventHeader, bitbucketPushEvent)
				req.Header.Set(bitbucketSignatureHeader, "totally-invalid-signature")
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
				req.Header.Set(bitbucketEventHeader, bitbucketPushEvent)
				req.Header.Set(bitbucketSignatureHeader, sign(bodyBuf.Bytes()))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
				require.JSONEq(t, `{"error":"invalid request body"}`, rr.Body.String())
			},
		},
		{
			name:       "branch qualifier does not match ref",
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
								RepoURL: "https://bitbucket.org/example/repo",
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
				bodyBuf := bytes.NewBuffer([]byte(pushEventRequestBody))
				req := httptest.NewRequest(http.MethodPost, testURL, bodyBuf)
				req.Header.Set(bitbucketEventHeader, bitbucketPushEvent)
				req.Header.Set(bitbucketSignatureHeader, sign(bodyBuf.Bytes()))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(
					t,
					`{"msg":"refreshed 0 warehouse(s)"}`,
					rr.Body.String(),
				)
			},
		},
		{
			name:       "branch qualifier matches ref",
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
								RepoURL: "https://bitbucket.org/example/repo",
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
				bodyBuf := bytes.NewBuffer([]byte(pushEventRequestBody))
				req := httptest.NewRequest(http.MethodPost, testURL, bodyBuf)
				req.Header.Set(bitbucketEventHeader, bitbucketPushEvent)
				req.Header.Set(bitbucketSignatureHeader, sign(bodyBuf.Bytes()))
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
			name:       "tag qualifier matches ref",
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
								RepoURL:                 "https://bitbucket.org/example/repo",
								CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestTag,
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
				bodyBuf := bytes.NewBuffer([]byte(tagPushEventRequestBody))
				req := httptest.NewRequest(http.MethodPost, testURL, bodyBuf)
				req.Header.Set(bitbucketEventHeader, bitbucketPushEvent)
				req.Header.Set(bitbucketSignatureHeader, sign(bodyBuf.Bytes()))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 1 warehouse(s)"}`, rr.Body.String())
			},
		},
		{
			name:       "tag qualifier does not match ref",
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
								RepoURL: "https://bitbucket.org/example/repo",
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
				bodyBuf := bytes.NewBuffer([]byte(tagPushEventRequestBody))
				req := httptest.NewRequest(http.MethodPost, testURL, bodyBuf)
				req.Header.Set(bitbucketEventHeader, bitbucketPushEvent)
				req.Header.Set(bitbucketSignatureHeader, sign(bodyBuf.Bytes()))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 0 warehouse(s)"}`, rr.Body.String())
			},
		},
		{
			name:       "bitbucket server refreshed ssh url",
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
								RepoURL: "ssh://git@bitbucket.example.org:7999/example/repo.git",
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
				bodyBuf := bytes.NewBuffer([]byte(pushEventRequestBodyBitbucketServer))
				req := httptest.NewRequest(http.MethodPost, testURL, bodyBuf)
				req.Header.Set(bitbucketEventHeader, bitbucketRefsChangedEvent)
				req.Header.Set(bitbucketSignatureHeader, sign(bodyBuf.Bytes()))
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
			name:       "pullrequest:fulfilled refreshes matching Promotion",
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
										"url": "https://bitbucket.org/example/repo/pull-requests/42",
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
				bodyBuf := bytes.NewBuffer([]byte(prFulfilledEventRequestBody))
				req := httptest.NewRequest(http.MethodPost, testURL, bodyBuf)
				req.Header.Set(bitbucketEventHeader, bitbucketPRFulfilledEvent)
				req.Header.Set(bitbucketSignatureHeader, sign(bodyBuf.Bytes()))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 1 promotion(s)"}`, rr.Body.String())
			},
		},
		{
			name:       "pullrequest:rejected refreshes matching Promotion",
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
										"url": "https://bitbucket.org/example/repo/pull-requests/42",
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
				bodyBuf := bytes.NewBuffer([]byte(prFulfilledEventRequestBody))
				req := httptest.NewRequest(http.MethodPost, testURL, bodyBuf)
				req.Header.Set(bitbucketEventHeader, bitbucketPRRejectedEvent)
				req.Header.Set(bitbucketSignatureHeader, sign(bodyBuf.Bytes()))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 1 promotion(s)"}`, rr.Body.String())
			},
		},
		{
			name:       "pullrequest:fulfilled with no matching Promotions",
			secretData: testSecretData,
			client: fake.NewClientBuilder().WithScheme(testScheme).WithIndex(
				&kargoapi.Promotion{},
				indexer.RunningPromotionsByPullRequestURLField,
				indexer.RunningPromotionsByPullRequestURL,
			).Build(),
			req: func() *http.Request {
				bodyBuf := bytes.NewBuffer([]byte(prFulfilledEventRequestBody))
				req := httptest.NewRequest(http.MethodPost, testURL, bodyBuf)
				req.Header.Set(bitbucketEventHeader, bitbucketPRFulfilledEvent)
				req.Header.Set(bitbucketSignatureHeader, sign(bodyBuf.Bytes()))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 0 promotion(s)"}`, rr.Body.String())
			},
		},
		{
			name:       "pr:merged refreshes matching Promotion",
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
										"url": "https://bitbucket.example.com/projects/EXAMPLE/repos/repo/pull-requests/42",
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
				bodyBuf := bytes.NewBuffer([]byte(prMergedEventRequestBody))
				req := httptest.NewRequest(http.MethodPost, testURL, bodyBuf)
				req.Header.Set(bitbucketEventHeader, bitbucketPRMergedEvent)
				req.Header.Set(bitbucketSignatureHeader, sign(bodyBuf.Bytes()))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 1 promotion(s)"}`, rr.Body.String())
			},
		},
		{
			name:       "pr:declined refreshes matching Promotion",
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
										"url": "https://bitbucket.example.com/projects/EXAMPLE/repos/repo/pull-requests/42",
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
				bodyBuf := bytes.NewBuffer([]byte(prMergedEventRequestBody))
				req := httptest.NewRequest(http.MethodPost, testURL, bodyBuf)
				req.Header.Set(bitbucketEventHeader, bitbucketPRDeclinedEvent)
				req.Header.Set(bitbucketSignatureHeader, sign(bodyBuf.Bytes()))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 1 promotion(s)"}`, rr.Body.String())
			},
		},
		{
			name:       "pr:merged with no matching Promotions",
			secretData: testSecretData,
			client: fake.NewClientBuilder().WithScheme(testScheme).WithIndex(
				&kargoapi.Promotion{},
				indexer.RunningPromotionsByPullRequestURLField,
				indexer.RunningPromotionsByPullRequestURL,
			).Build(),
			req: func() *http.Request {
				bodyBuf := bytes.NewBuffer([]byte(prMergedEventRequestBody))
				req := httptest.NewRequest(http.MethodPost, testURL, bodyBuf)
				req.Header.Set(bitbucketEventHeader, bitbucketPRMergedEvent)
				req.Header.Set(bitbucketSignatureHeader, sign(bodyBuf.Bytes()))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 0 promotion(s)"}`, rr.Body.String())
			},
		},
		{
			name:       "bitbucket server refreshed https url",
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
								RepoURL: "https://example.org/bitbucket/scm/example/repo.git",
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
				bodyBuf := bytes.NewBuffer([]byte(pushEventRequestBodyBitbucketServer))
				req := httptest.NewRequest(http.MethodPost, testURL, bodyBuf)
				req.Header.Set(bitbucketEventHeader, bitbucketRefsChangedEvent)
				req.Header.Set(bitbucketSignatureHeader, sign(bodyBuf.Bytes()))
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
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			requestBody, err := io.ReadAll(testCase.req().Body)
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = testCase.req().Body.Close()
			})

			w := httptest.NewRecorder()
			(&bitbucketWebhookReceiver{
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
