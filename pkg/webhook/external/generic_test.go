package external

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/indexer"
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
			name:    "failure creating base env",
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
			name:    "condition not met - failed to compile expression",
			kClient: fake.NewClientBuilder().WithScheme(testScheme).Build(),
			config: &kargoapi.GenericWebhookReceiverConfig{
				Actions: []kargoapi.GenericWebhookAction{
					{
						ActionType:     kargoapi.GenericWebhookActionTypeRefresh,
						WhenExpression: "{x!kj\"}",
						TargetSelectionCriteria: []kargoapi.GenericWebhookTargetSelectionCriteria{
							{
								Kind: kargoapi.GenericWebhookTargetKindWarehouse,
								Name: "doesntmatter",
							},
						},
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
				t.Logf("response body: %s", w.Body.String())
				expected := `
					{
						"results":[
							{
								"action":"Refresh",
								"whenExpression":"{x!kj\"}",
								"targetSelectionCriteria":[
									{
										"kind":"Warehouse",
										"name":"doesntmatter",
										"labelSelector":{},
										"indexSelector":{}
									}
								],
								"matchedWhenExpression":false,
								"result":"Error",
								"summary":"Error evaluating whenExpression"
							}
						]
					}
				`
				require.JSONEq(t, expected, w.Body.String())
			},
		},
		{
			name:    "condition not met - failed to run expression",
			kClient: fake.NewClientBuilder().WithScheme(testScheme).Build(),
			config: &kargoapi.GenericWebhookReceiverConfig{
				Actions: []kargoapi.GenericWebhookAction{
					{
						ActionType: kargoapi.GenericWebhookActionTypeRefresh,
						// foo is not defined, so evaluation will fail
						WhenExpression: "foo()",
						TargetSelectionCriteria: []kargoapi.GenericWebhookTargetSelectionCriteria{
							{
								Kind: kargoapi.GenericWebhookTargetKindWarehouse,
								Name: "doesntmatter",
							},
						},
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
				t.Logf("response body: %s", w.Body.String())
				expected := `
					{
					"results": [
						{
							"action": "Refresh",
							"whenExpression": "foo()",
							"targetSelectionCriteria": [
								{
								"kind": "Warehouse",
								"name": "doesntmatter",
								"labelSelector": {},
								"indexSelector": {}
								}
							],
							"matchedWhenExpression": false,
							"result": "Error",
							"summary": "Error evaluating whenExpression"
						}
					]
				}`
				require.JSONEq(t, expected, w.Body.String())
			},
		},
		{
			name:    "condition not met - evaluated to false",
			kClient: fake.NewClientBuilder().WithScheme(testScheme).Build(),
			config: &kargoapi.GenericWebhookReceiverConfig{
				Actions: []kargoapi.GenericWebhookAction{
					{
						ActionType:     kargoapi.GenericWebhookActionTypeRefresh,
						WhenExpression: "request.header('X-Event-Type') == 'push'",
						TargetSelectionCriteria: []kargoapi.GenericWebhookTargetSelectionCriteria{
							{
								Kind: kargoapi.GenericWebhookTargetKindWarehouse,
								Name: "doesntmatter",
							},
						},
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
				t.Logf("response body: %s", w.Body.String())
				expected := `{
					"results":[
						{
							"action":"Refresh",
							"whenExpression":"request.header('X-Event-Type') == 'push'",
							"matchedWhenExpression":false,
							"targetSelectionCriteria":[
								{
									"kind":"Warehouse",
									"name":"doesntmatter",
									"labelSelector":{},
									"indexSelector":{}
								}
							],
							"result":"NotApplicable",
							"summary":"Request did not match whenExpression"
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
						ActionType:     kargoapi.GenericWebhookActionTypeRefresh,
						WhenExpression: "request.header('X-Event-Type')",
						TargetSelectionCriteria: []kargoapi.GenericWebhookTargetSelectionCriteria{
							{
								Kind: kargoapi.GenericWebhookTargetKindWarehouse,
								Name: "doesntmatter",
							},
						},
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
				t.Logf("response body: %s", w.Body.String())
				expected := `
				{
					"results": [
						{
							"action": "Refresh",
							"whenExpression": "request.header('X-Event-Type')",
							"targetSelectionCriteria": [
								{
								"kind": "Warehouse",
								"name": "doesntmatter",
								"labelSelector": {},
								"indexSelector": {}
								}
							],
							"matchedWhenExpression": false,
							"result": "Error",
							"summary": "Error evaluating whenExpression"
						}
					]
				}`
				require.JSONEq(t, expected, w.Body.String())
			},
		},
		{
			name: "list error",
			kClient: fake.NewClientBuilder().WithScheme(testScheme).WithInterceptorFuncs(
				interceptor.Funcs{
					List: func(
						_ context.Context,
						_ client.WithWatch,
						_ client.ObjectList,
						_ ...client.ListOption,
					) error {
						return errors.New("oops")
					},
				},
			).Build(),
			config: &kargoapi.GenericWebhookReceiverConfig{
				Actions: []kargoapi.GenericWebhookAction{
					{
						ActionType:     kargoapi.GenericWebhookActionTypeRefresh,
						WhenExpression: "true",
						TargetSelectionCriteria: []kargoapi.GenericWebhookTargetSelectionCriteria{
							{
								Kind: kargoapi.GenericWebhookTargetKindWarehouse,
								Name: "doesntmatter",
							},
						},
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
				t.Logf("response body: %s", w.Body.String())
				expected := `
				{
					"results": [
						{
							"action": "Refresh",
							"whenExpression": "true",
							"targetSelectionCriteria": [
								{
								"kind": "Warehouse",
								"name": "doesntmatter",
								"labelSelector": {},
								"indexSelector": {}
								}
							],
							"matchedWhenExpression": true,
							"result": "Error",
							"summary": "Error evaluating targetSelectionCriteria"
						}
					]
				}`
				require.JSONEq(t, expected, w.Body.String())
			},
		},
		{
			name: "partial success",
			kClient: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				// this warehouse will be refreshed successfully
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-project",
						Name:      "api-warehouse",
						Labels: map[string]string{
							"foo":  "bar",     // satisfies labelSelector.MatchLabels
							"tier": "backend", // satisfies labelSelector.MatchExpressions
						},
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{
								RepoURL: "http://github.com/example/repo.git", // satisfies index selector
							},
						}},
					},
				},
				// this warehouse will fully satisfy the label selector but not the index selector
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-project",
						Name:      "other-api-warehouse",
						Labels: map[string]string{
							"foo":  "bar",
							"tier": "backend",
						},
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{
								RepoURL: "http://github.com/example/other-repo.git", // does NOT satisfy index selector
							},
						}},
					},
				},
				// this label will satisfy the indexSelector, labelSelector.MatchLabels,
				// but not labelSelector.MatchExpressions
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-project",
						Name:      "ui-warehouse",
						Labels: map[string]string{
							"foo": "bar",
							// "tier": "frontend", // missing
						},
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{
								RepoURL: "http://github.com/example/repo.git",
							},
						}},
					},
				},
				// this warehouse will satisfy all selectors but fail during refresh
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-project",
						Name:      "failure-warehouse",
						Labels: map[string]string{
							"foo":  "bar",     // satisfies labelSelector.MatchLabels
							"tier": "backend", // satisfies labelSelector.MatchExpressions
						},
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{
								RepoURL: "http://github.com/example/repo.git", // satisfies index selector
							},
						}},
					},
				},
			).WithIndex(
				&kargoapi.Warehouse{},
				indexer.WarehousesBySubscribedURLsField,
				indexer.WarehousesBySubscribedURLs,
			).WithInterceptorFuncs(
				interceptor.Funcs{
					Patch: func(
						_ context.Context,
						_ client.WithWatch,
						obj client.Object,
						_ client.Patch,
						_ ...client.PatchOption,
					) error {
						if obj.GetName() == "failure-warehouse" {
							return errors.New("something went wrong")
						}
						return nil
					},
				},
			).Build(),
			config: &kargoapi.GenericWebhookReceiverConfig{
				Actions: []kargoapi.GenericWebhookAction{
					{
						ActionType:     kargoapi.GenericWebhookActionTypeRefresh,
						Parameters:     map[string]string{"foo": "bar"},
						WhenExpression: "request.header('X-Event-Type') == 'push'",
						// use complex combination of both label and index selectors
						TargetSelectionCriteria: []kargoapi.GenericWebhookTargetSelectionCriteria{
							{
								Kind: kargoapi.GenericWebhookTargetKindWarehouse,
								LabelSelector: metav1.LabelSelector{
									MatchLabels: map[string]string{"foo": "bar"},
									MatchExpressions: []metav1.LabelSelectorRequirement{
										{
											Key:      "tier",
											Operator: metav1.LabelSelectorOpIn,
											Values:   []string{"backend", "frontend"},
										},
									},
								},
								IndexSelector: kargoapi.IndexSelector{
									MatchIndices: []kargoapi.IndexSelectorRequirement{
										{
											Key:      indexer.WarehousesBySubscribedURLsField,
											Operator: kargoapi.IndexSelectorOperatorEqual,
											Value:    `${{ normalizeGit(request.body.repository.url) }}`,
										},
									},
								},
							},
						},
					},
				},
			},
			req: func() *http.Request {
				// not normalized so the index selector can only work if 'normalize' function is used
				b := []byte(`{"repository": {"url": "http://github.com/example/repo.git"}}`)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(b),
				)
				req.Header.Set("X-Event-Type", "push")
				return req
			},
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, w.Code)
				t.Logf("response body: %s", w.Body.String())
				expected := `
				{
					"results": [
						{
							"action": "Refresh",
							"whenExpression": "request.header('X-Event-Type') == 'push'",
							"parameters": {"foo": "bar"},
							"targetSelectionCriteria": [
								{
									"kind": "Warehouse",
									"labelSelector": {
										"matchLabels": {"foo": "bar"},
										"matchExpressions": [
										{
											"key": "tier",
											"operator": "In",
											"values": [
												"backend",
												"frontend"
											]
										}
									]
								},
								"indexSelector": {
									"matchIndices": [
											{
												"key": "subscribedURLs",
												"operator": "Equals",
												"value": "${{ normalizeGit(request.body.repository.url) }}"
											}
										]
									}
								}
							],
							"matchedWhenExpression": true,
							"selectedTargets": [
								{
									"namespace": "test-project",
									"name": "api-warehouse",
									"success": true
								},
								{
									"namespace": "test-project",
									"name": "failure-warehouse",
									"success": false
								}
							],
							"result": "PartialSuccess",
							"summary": "Refreshed 1 of 2 selected resources"
							}
						]
					}
				`
				require.JSONEq(t, expected, w.Body.String())
			},
		},
		{
			name: "full success",
			kClient: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				// this warehouse will be refreshed successfully
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-project",
						Name:      "api-warehouse",
						Labels: map[string]string{
							"foo":  "bar",     // satisfies labelSelector.MatchLabels
							"tier": "backend", // satisfies labelSelector.MatchExpressions
						},
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{
								RepoURL: "http://github.com/example/repo.git", // satisfies index selector
							},
						}},
					},
				},
			).WithIndex(
				&kargoapi.Warehouse{},
				indexer.WarehousesBySubscribedURLsField,
				indexer.WarehousesBySubscribedURLs,
			).Build(),
			config: &kargoapi.GenericWebhookReceiverConfig{
				Actions: []kargoapi.GenericWebhookAction{
					{
						ActionType:     kargoapi.GenericWebhookActionTypeRefresh,
						WhenExpression: "request.header('X-Event-Type') == 'push'",
						// use complex combination of both label and index selectors
						TargetSelectionCriteria: []kargoapi.GenericWebhookTargetSelectionCriteria{
							{
								Kind: kargoapi.GenericWebhookTargetKindWarehouse,
								LabelSelector: metav1.LabelSelector{
									MatchLabels: map[string]string{"foo": "bar"},
									MatchExpressions: []metav1.LabelSelectorRequirement{
										{
											Key:      "tier",
											Operator: metav1.LabelSelectorOpIn,
											Values:   []string{"backend", "frontend"},
										},
									},
								},
								IndexSelector: kargoapi.IndexSelector{
									MatchIndices: []kargoapi.IndexSelectorRequirement{
										{
											Key:      indexer.WarehousesBySubscribedURLsField,
											Operator: kargoapi.IndexSelectorOperatorEqual,
											Value:    `${{ normalizeGit(request.body.repository.url) }}`,
										},
									},
								},
							},
						},
					},
				},
			},
			req: func() *http.Request {
				// not normalized so the index selector can only work if 'normalizeGit' function is used
				b := []byte(`{"repository": {"url": "http://github.com/example/repo.git"}}`)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(b),
				)
				req.Header.Set("X-Event-Type", "push")
				return req
			},
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
				t.Logf("response body: %s", w.Body.String())
				expected := `
					{
						"results": [
							{
								"action": "Refresh",
								"whenExpression": "request.header('X-Event-Type') == 'push'",
								"targetSelectionCriteria": [
									{
										"kind": "Warehouse",
										"labelSelector": {
										"matchLabels": {"foo": "bar"},
										"matchExpressions": [
											{
												"key": "tier",
												"operator": "In",
												"values": [
													"backend",
													"frontend"
												]
											}
										]
									},
									"indexSelector": {
											"matchIndices": [
												{
													"key": "subscribedURLs",
													"operator": "Equals",
													"value": "${{ normalizeGit(request.body.repository.url) }}"
												}
											]
										}
									}
								],
								"matchedWhenExpression": true,
								"selectedTargets": [
									{
										"namespace": "test-project",
										"name": "api-warehouse",
										"success": true
									}
								],
								"result": "Success",
								"summary": "Refreshed 1 of 1 selected resources"
							}
						]
					}
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
