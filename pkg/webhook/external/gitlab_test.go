package external

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	gl "gitlab.com/gitlab-org/api/client-go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/indexer"
)

const gitlabPushEventRequestBody = `
{
	"ref": "refs/heads/main",
	"repository":{
		"git_http_url": "https://gitlab.com/example/repo.git",
		"git_ssh_url": "git@gitlab.com:example/repo.git"
	}
}`

const gitlabTagPushEventRequestBody = `
{
	"ref": "refs/tags/v1.0.0",
	"repository":{
		"git_http_url": "https://gitlab.com/example/repo.git",
		"git_ssh_url": "git@gitlab.com:example/repo.git"
	}
}`

func TestGitLabHandler(t *testing.T) {
	const testURL = "https://webhooks.kargo.example.com/nonsense"

	const testProjectName = "fake-project"

	testScheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(testScheme))

	const testToken = "mysupersecrettoken"
	testSecretData := map[string][]byte{
		gitLabSecretDataKey: []byte(testToken),
	}

	testCases := []struct {
		name       string
		client     client.Client
		secretData map[string][]byte
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
			name:       "unauthorized",
			secretData: testSecretData,
			req: func() *http.Request {
				return httptest.NewRequest(http.MethodPost, testURL, nil)
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, rr.Code)
			},
		},
		{
			name:       "unsupported event type",
			secretData: testSecretData,
			req: func() *http.Request {
				req := httptest.NewRequest(http.MethodPost, testURL, nil)
				req.Header.Set(gitlabTokenHeader, testToken)
				req.Header.Set(gitlabEventHeader, "nonsense")
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
			name:       "malformed request body",
			secretData: testSecretData,
			req: func() *http.Request {
				bodyBuf := bytes.NewBuffer([]byte("invalid json"))
				req := httptest.NewRequest(http.MethodPost, testURL, bodyBuf)
				req.Header.Set(gitlabTokenHeader, testToken)
				req.Header.Set(gitlabEventHeader, string(gl.EventTypePush))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
				require.JSONEq(t, `{"error":"invalid request body"}`, rr.Body.String())
			},
		},
		{
			name: "no ref match (push event)",
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
								RepoURL: "https://gitlab.com/example/repo",
								Branch:  "not-main", // This constraint won't be met
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
				bodyBuf := bytes.NewBuffer([]byte(gitlabPushEventRequestBody))
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bodyBuf,
				)
				req.Header.Set(gitlabTokenHeader, testToken)
				req.Header.Set(gitlabEventHeader, string(gl.EventTypePush))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(
					t, `{"msg":"refreshed 0 warehouse(s)"}`, rr.Body.String(),
				)
			},
		},
		{
			name:       "warehouse refreshed (push event, https)",
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
								RepoURL: "https://gitlab.com/example/repo",
								Branch:  "main",
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
				bodyBuf := bytes.NewBuffer([]byte(gitlabPushEventRequestBody))
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bodyBuf,
				)
				req.Header.Set(gitlabTokenHeader, testToken)
				req.Header.Set(gitlabEventHeader, string(gl.EventTypePush))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(
					t, `{"msg":"refreshed 1 warehouse(s)"}`, rr.Body.String(),
				)
			},
		},
		{
			name:       "warehouse refreshed (push event, ssh)",
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
								RepoURL: "git@gitlab.com:example/repo",
								Branch:  "main",
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
				bodyBuf := bytes.NewBuffer([]byte(gitlabPushEventRequestBody))
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bodyBuf,
				)
				req.Header.Set(gitlabTokenHeader, testToken)
				req.Header.Set(gitlabEventHeader, string(gl.EventTypePush))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(
					t, `{"msg":"refreshed 1 warehouse(s)"}`, rr.Body.String(),
				)
			},
		},
		{
			name: "no ref match (tag event)",
			// This event would prompt the Warehouse to refresh if not for the ref in
			// the event being for a tag falling outside the subscription's semver
			// range.
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
								RepoURL:                 "https://gitlab.com/example/repo",
								CommitSelectionStrategy: kargoapi.CommitSelectionStrategySemVer,
								SemverConstraint:        "^2.0.0", // Constraint won't be met
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
				bodyBuf := bytes.NewBuffer([]byte(gitlabTagPushEventRequestBody))
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bodyBuf,
				)
				req.Header.Set(gitlabTokenHeader, testToken)
				req.Header.Set(gitlabEventHeader, string(gl.EventTypeTagPush))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(
					t, `{"msg":"refreshed 0 warehouse(s)"}`, rr.Body.String(),
				)
			},
		},
		{
			name:       "warehouse refreshed (tag event, https)",
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
								RepoURL:                 "https://gitlab.com/example/repo",
								CommitSelectionStrategy: kargoapi.CommitSelectionStrategySemVer,
								SemverConstraint:        "^1.0.0",
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
				bodyBuf := bytes.NewBuffer([]byte(gitlabTagPushEventRequestBody))
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bodyBuf,
				)
				req.Header.Set(gitlabTokenHeader, testToken)
				req.Header.Set(gitlabEventHeader, string(gl.EventTypeTagPush))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(
					t, `{"msg":"refreshed 1 warehouse(s)"}`, rr.Body.String(),
				)
			},
		},
		{
			name:       "warehouse refreshed (tag event, ssh)",
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
								RepoURL:                 "git@gitlab.com:example/repo",
								CommitSelectionStrategy: kargoapi.CommitSelectionStrategySemVer,
								SemverConstraint:        "^1.0.0",
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
				bodyBuf := bytes.NewBuffer([]byte(gitlabTagPushEventRequestBody))
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bodyBuf,
				)
				req.Header.Set(gitlabTokenHeader, testToken)
				req.Header.Set(gitlabEventHeader, string(gl.EventTypeTagPush))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(
					t, `{"msg":"refreshed 1 warehouse(s)"}`, rr.Body.String(),
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
			(&gitlabWebhookReceiver{
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
