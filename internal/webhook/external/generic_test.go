package external

import (
	"bytes"
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

var genericGitPushRequestBody = []byte(`{
	"repository": {
		"repo_name": "https://git.example.com/repo.git"
	}
}`)

var genericImagePushRequestBody = []byte(`{
	"repository": {
		"repo_name": "example/repo"
	}
}`)

var genericChartPushRequestBody = []byte(`{
	"repository": {
		"repo_name": "oci://charts.example.com/example/repo"
	}
}`)

func TestGenericHandler(t *testing.T) {
	const testURL = "https://webhooks.kargo.example.com/nonsense"

	const testProjectName = "fake-project"

	testScheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(testScheme))

	testCases := []struct {
		name       string
		cfg        kargoapi.GenericWebhookReceiverConfig
		client     client.Client
		req        func() *http.Request
		assertions func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "malformed request body",
			req: func() *http.Request {
				return httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer([]byte("not valid json")),
				)
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
				require.JSONEq(t, `{"error":"invalid request body"}`, rr.Body.String())
			},
		},
		{
			name: "error compiling predicate expression",
			cfg: kargoapi.GenericWebhookReceiverConfig{
				Refresh: &kargoapi.GenericRefreshConfig{
					Predicate: "not a valid expression",
				},
			},
			req: func() *http.Request {
				return httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(genericGitPushRequestBody),
				)
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rr.Code)
				require.JSONEq(t, "{}", rr.Body.String())
			},
		},
		{
			name: "error evaluating predicate expression",
			cfg: kargoapi.GenericWebhookReceiverConfig{
				Refresh: &kargoapi.GenericRefreshConfig{
					Predicate: "nonexistent()",
				},
			},
			req: func() *http.Request {
				return httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(genericGitPushRequestBody),
				)
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rr.Code)
				require.JSONEq(t, "{}", rr.Body.String())
			},
		},
		{
			name: "predicate evaluates to a non-boolean",
			cfg: kargoapi.GenericWebhookReceiverConfig{
				Refresh: &kargoapi.GenericRefreshConfig{
					Predicate: "'not a boolean'",
				},
			},
			req: func() *http.Request {
				return httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(genericGitPushRequestBody),
				)
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rr.Code)
				require.JSONEq(t, "{}", rr.Body.String())
			},
		},
		{
			name: "predicate evaluates to false",
			cfg: kargoapi.GenericWebhookReceiverConfig{
				Refresh: &kargoapi.GenericRefreshConfig{
					Predicate: "false",
				},
			},
			req: func() *http.Request {
				return httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(genericGitPushRequestBody),
				)
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"no action taken"}`, rr.Body.String())
			},
		},
		{
			name: "error compiling repo URL expression",
			cfg: kargoapi.GenericWebhookReceiverConfig{
				Refresh: &kargoapi.GenericRefreshConfig{
					Predicate: "true",
					Selectors: kargoapi.GenericRefreshSelectors{
						ChartRepoURL: "not a valid expression",
					},
				},
			},
			req: func() *http.Request {
				return httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(genericGitPushRequestBody),
				)
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rr.Code)
				require.JSONEq(t, "{}", rr.Body.String())
			},
		},
		{
			name: "error evaluating repo URL expression",
			cfg: kargoapi.GenericWebhookReceiverConfig{
				Refresh: &kargoapi.GenericRefreshConfig{
					Predicate: "true",
					Selectors: kargoapi.GenericRefreshSelectors{
						GitRepoURL: "nonexistent()",
					},
				},
			},
			req: func() *http.Request {
				return httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(genericGitPushRequestBody),
				)
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rr.Code)
				require.JSONEq(t, "{}", rr.Body.String())
			},
		},
		{
			name: "repo URL expression evaluates to a non-string",
			cfg: kargoapi.GenericWebhookReceiverConfig{
				Refresh: &kargoapi.GenericRefreshConfig{
					Predicate: "true",
					Selectors: kargoapi.GenericRefreshSelectors{
						GitRepoURL: "42",
					},
				},
			},
			req: func() *http.Request {
				return httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(genericGitPushRequestBody),
				)
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rr.Code)
				require.JSONEq(t, "{}", rr.Body.String())
			},
		},
		{
			name: "success -- git commit pushed",
			cfg: kargoapi.GenericWebhookReceiverConfig{
				Refresh: &kargoapi.GenericRefreshConfig{
					Predicate: "request.header('X-Kargo-Event') == 'push'",
					Selectors: kargoapi.GenericRefreshSelectors{
						GitRepoURL: "request.body.repository.repo_name",
					},
				},
			},
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "fake-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{
								RepoURL: "https://git.example.com/repo.git",
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
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(genericGitPushRequestBody),
				)
				req.Header.Set("X-Kargo-Event", "push")
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 1 warehouse(s)"}`, rr.Body.String())
			},
		},
		{
			name: "success -- container image pushed",
			cfg: kargoapi.GenericWebhookReceiverConfig{
				Refresh: &kargoapi.GenericRefreshConfig{
					Predicate: "request.header('X-Kargo-Event') == 'push'",
					Selectors: kargoapi.GenericRefreshSelectors{
						ImageRepoURL: "request.body.repository.repo_name",
					},
				},
			},
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "fake-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Image: &kargoapi.ImageSubscription{
								RepoURL: "example/repo",
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
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(genericImagePushRequestBody),
				)
				req.Header.Set("X-Kargo-Event", "push")
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 1 warehouse(s)"}`, rr.Body.String())
			},
		},
		{
			name: "success -- chart pushed",
			cfg: kargoapi.GenericWebhookReceiverConfig{
				Refresh: &kargoapi.GenericRefreshConfig{
					Predicate: "request.header('X-Kargo-Event') == 'push'",
					Selectors: kargoapi.GenericRefreshSelectors{
						ChartRepoURL: "request.body.repository.repo_name",
					},
				},
			},
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "fake-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Chart: &kargoapi.ChartSubscription{
								RepoURL: "oci://charts.example.com/example/repo",
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
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(genericChartPushRequestBody),
				)
				req.Header.Set("X-Kargo-Event", "push")
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
			(&genericWebhookReceiver{
				baseWebhookReceiver: &baseWebhookReceiver{
					client:  testCase.client,
					project: testProjectName,
				},
				cfg: testCase.cfg,
			}).getHandler(requestBody)(w, testCase.req())

			testCase.assertions(t, w)
		})
	}
}
