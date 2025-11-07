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
	"github.com/akuity/kargo/pkg/indexer"
	"github.com/akuity/kargo/pkg/urls"
)

func TestGenericHandler(t *testing.T) {
	const testURL = "https://webhooks.kargo.example.com/nonsense"
	testScheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(testScheme))

	testCases := []struct {
		name       string
		kClient    client.Client
		config     *kargoapi.GenericWebhookReceiverConfig
		req        func() *http.Request
		assertions func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:    "failure creating global env",
			kClient: fake.NewClientBuilder().WithScheme(testScheme).Build(),
			config:  &kargoapi.GenericWebhookReceiverConfig{},
			req: func() *http.Request {
				return httptest.NewRequest(
					http.MethodPost,
					"/",
					bytes.NewBuffer([]byte(`"invalid-json`)),
				)
			},
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
				require.Contains(t, w.Body.String(), "invalid request body")
			},
		},
		{
			name:    "condition not met - failed to evaluate",
			kClient: fake.NewClientBuilder().WithScheme(testScheme).Build(),
			config: &kargoapi.GenericWebhookReceiverConfig{
				Actions: []kargoapi.GenericWebhookAction{
					{
						Name: kargoapi.GenericWebhookActionNameRefresh,
						// foo is not defined, so evaluation will fail
						MatchExpression: "${{ foo() }}",
					},
				},
			},
			req: func() *http.Request {
				return httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer([]byte(`{"some": "data"}`)),
				)
			},
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, w.Code)
				expected := `{
					"results":[
						{
							"actionName":"Refresh",
							"conditionFailure":{
								"expression":"${{ foo() }}",
								"satisfied":false,
								"evalError": "reflect: call of reflect.Value.Call on zero Value (1:2)\n |  foo() \n | .^"
							}
						}
					]}
				`
				require.JSONEq(t, expected, w.Body.String())
			},
		},
		{
			name:    "condition not met - evaluated to false",
			kClient: fake.NewClientBuilder().WithScheme(testScheme).Build(),
			config: &kargoapi.GenericWebhookReceiverConfig{
				Actions: []kargoapi.GenericWebhookAction{
					{
						Name:            kargoapi.GenericWebhookActionNameRefresh,
						MatchExpression: `${{ request.header('X-Event-Type') == 'push' }}`,
					},
				},
			},
			req: func() *http.Request {
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer([]byte(`{"some": "data"}`)),
				)
				req.Header.Set("X-Event-Type", "pull_request")
				return req
			},
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
				expected := `{
					"results":[
						{
							"actionName":"Refresh",
							"conditionFailure":{
								"expression":"${{ request.header('X-Event-Type') == 'push' }}",
								"satisfied":false,
								"evalError": "<nil>"
							}
						}
					]}
				`
				require.JSONEq(t, expected, w.Body.String())
			},
		},
		{
			name:    "condition not met - evaluated to non-boolean type",
			kClient: fake.NewClientBuilder().WithScheme(testScheme).Build(),
			config: &kargoapi.GenericWebhookReceiverConfig{
				Actions: []kargoapi.GenericWebhookAction{
					{
						Name:            kargoapi.GenericWebhookActionNameRefresh,
						MatchExpression: `${{ request.header('X-Event-Type') }}`,
					},
				},
			},
			req: func() *http.Request {
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer([]byte(`{"some": "data"}`)),
				)
				req.Header.Set("X-Event-Type", "pull_request")
				return req
			},
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, w.Code)
				expected := `{
					"results":[
						{
							"actionName":"Refresh",
							"conditionFailure":{
								"expression":"${{ request.header('X-Event-Type') }}",
								"satisfied":false,
								"evalError": "match expression result \"pull_request\" is of type string; expected bool"
							}
						}
					]}
				`
				require.JSONEq(t, expected, w.Body.String())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestBody, err := io.ReadAll(tc.req().Body)
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = tc.req().Body.Close()
			})
			w := httptest.NewRecorder()
			(&genericWebhookReceiver{
				baseWebhookReceiver: &baseWebhookReceiver{
					client:  tc.kClient,
					project: "test-project",
				},
				config: tc.config,
			}).getHandler(requestBody).ServeHTTP(w, tc.req())
			tc.assertions(t, w)
		})
	}
}

func Test_newListOptionsForIndexSelector(t *testing.T) {
	// This is indirectly tested via Test_buildListOptionsForTarget,
	// so we just need a basic smoke test here.
	testRepoURL := urls.NormalizeGit("https://github.com/example/repo.git")
	testScheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(testScheme))

	tests := []struct {
		name      string
		kClient   client.Client
		selector  kargoapi.IndexSelector
		env       map[string]any
		expectErr bool
	}{
		{
			name: "Equal selector satisfied",
			kClient: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "some-namespace",
						Name:      "some-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{RepoURL: testRepoURL},
						}},
					},
				},
			).WithIndex(
				&kargoapi.Warehouse{},
				indexer.WarehousesBySubscribedURLsField,
				indexer.WarehousesBySubscribedURLs,
			).Build(),
			selector: kargoapi.IndexSelector{
				MatchExpressions: []kargoapi.IndexSelectorRequirement{
					{
						Key:      indexer.WarehousesBySubscribedURLsField,
						Operator: kargoapi.IndexSelectorRequirementOperatorEqual,
						Value:    "${{ request.body.repository.clone_url }}",
					},
				},
			},
			env: map[string]any{
				"request": map[string]any{
					"body": map[string]any{
						"repository": map[string]any{
							"clone_url": testRepoURL,
						},
					},
				},
			},
			expectErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listOpts, err := newListOptionsForIndexSelector(tt.selector, tt.env)
			if tt.expectErr {
				require.Error(t, err)
				return
			}
			whList := new(kargoapi.WarehouseList)
			err = tt.kClient.List(t.Context(), whList, listOpts...)
			require.NoError(t, err)
			require.Len(t, whList.Items, 1)
		})
	}
}
