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

var genericRequestBody = []byte(`{
	"repository": {
		"repo_name": "https://git.example.com/repo.git"
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
				WarehouseRefresh: &kargoapi.GenericWarehouseRefreshConfig{
					Predicate: "not a valid expression",
				},
			},
			req: func() *http.Request {
				return httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(genericRequestBody),
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
				WarehouseRefresh: &kargoapi.GenericWarehouseRefreshConfig{
					Predicate: "nonexistent()",
				},
			},
			req: func() *http.Request {
				return httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(genericRequestBody),
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
				WarehouseRefresh: &kargoapi.GenericWarehouseRefreshConfig{
					Predicate: "'not a boolean'",
				},
			},
			req: func() *http.Request {
				return httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(genericRequestBody),
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
				WarehouseRefresh: &kargoapi.GenericWarehouseRefreshConfig{
					Predicate: "false",
				},
			},
			req: func() *http.Request {
				return httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(genericRequestBody),
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
				WarehouseRefresh: &kargoapi.GenericWarehouseRefreshConfig{
					Predicate: "true",
					RepoURL:   "not a valid expression",
				},
			},
			req: func() *http.Request {
				return httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(genericRequestBody),
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
				WarehouseRefresh: &kargoapi.GenericWarehouseRefreshConfig{
					Predicate: "true",
					RepoURL:   "nonexistent()",
				},
			},
			req: func() *http.Request {
				return httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(genericRequestBody),
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
				WarehouseRefresh: &kargoapi.GenericWarehouseRefreshConfig{
					Predicate: "true",
					RepoURL:   "42",
				},
			},
			req: func() *http.Request {
				return httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(genericRequestBody),
				)
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rr.Code)
				require.JSONEq(t, "{}", rr.Body.String())
			},
		},
		{
			name: "success",
			cfg: kargoapi.GenericWebhookReceiverConfig{
				WarehouseRefresh: &kargoapi.GenericWarehouseRefreshConfig{
					Predicate: "request.header('X-Kargo-Git-Event') == 'push'",
					RepoURL:   "request.body.repository.repo_name",
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
					bytes.NewBuffer(genericRequestBody),
				)
				req.Header.Set("X-Kargo-Git-Event", "push")
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
