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
)

func TestStaticHandler(t *testing.T) {
	const testURL = "https://webhooks.kargo.example.com/nonsense"

	const testProjectName = "fake-project"

	testScheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(testScheme))

	testCases := []struct {
		name       string
		client     client.Client
		req        func() *http.Request
		rule       kargoapi.StaticWebhookRule
		assertions func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "unsupported action",
			client: fake.NewClientBuilder().WithScheme(testScheme).Build(),
			rule: kargoapi.StaticWebhookRule{
				Action: "invalid-action",
			},
			req: func() *http.Request {
				return httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer([]byte(`{"foo":"bar"}`)),
				)
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
				require.JSONEq(
					t,
					`{"error":"unsupported action: \"invalid-action\""}`,
					rr.Body.String(),
				)
			},
		},
		{
			name: "single out single warehouse from target list",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "fake-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Image: &kargoapi.ImageSubscription{
								RepoURL: "somegenerichost.com/repo",
							},
						}},
					},
				},
				// this warehouse should not be refreshed (even though it's in the
				// target list) because an optional URL query parameter will target
				// only the other warehouse.
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "other-fake-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Image: &kargoapi.ImageSubscription{
								RepoURL: "somegenerichost.com/otherrepo",
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
				testURLWithQuery := testURL + "?target=fake-warehouse"
				return httptest.NewRequest(
					http.MethodPost,
					testURLWithQuery,
					bytes.NewBuffer([]byte(`{"foo":"bar"}`)),
				)
			},
			rule: kargoapi.StaticWebhookRule{
				Action: kargoapi.StaticWebhookActionRefresh,
				Targets: []kargoapi.StaticWebhookTarget{
					{
						Name:      "fake-warehouse",
						Namespace: testProjectName,
						Type:      kargoapi.StaticWebhookTargetTypeWarehouse,
					},
					{
						Name:      "other-fake-warehouse",
						Namespace: testProjectName,
						Type:      kargoapi.StaticWebhookTargetTypeWarehouse,
					},
				},
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(
					t,
					`{"msg":"successfully refreshed 1 target(s)"}`,
					rr.Body.String(),
				)
			},
		},
		{
			name: "all targeted warehouses refreshed",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "fake-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Image: &kargoapi.ImageSubscription{
								RepoURL: "somegenerichost.com/repo",
							},
						}},
					},
				},
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "other-fake-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Image: &kargoapi.ImageSubscription{
								RepoURL: "somegenerichost.com/otherrepo",
							},
						}},
					},
				},
				// this warehouse should not be refreshed because it's not targeted
				// in the static webhook rule
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "fake-warehouse-not-targeted",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Image: &kargoapi.ImageSubscription{
								RepoURL: "somegenerichost.com/foobar",
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
				return httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer([]byte(`{"foo":"bar"}`)),
				)
			},
			rule: kargoapi.StaticWebhookRule{
				Action: kargoapi.StaticWebhookActionRefresh,
				Targets: []kargoapi.StaticWebhookTarget{
					{
						Name:      "fake-warehouse",
						Namespace: testProjectName,
						Type:      kargoapi.StaticWebhookTargetTypeWarehouse,
					},
					{
						Name:      "other-fake-warehouse",
						Namespace: testProjectName,
						Type:      kargoapi.StaticWebhookTargetTypeWarehouse,
					},
				},
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(
					t,
					`{"msg":"successfully refreshed 2 target(s)"}`,
					rr.Body.String(),
				)
			},
		},
		{
			name: "partial success",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "fake-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Image: &kargoapi.ImageSubscription{
								RepoURL: "somegenerichost.com/repo",
							},
						}},
					},
				},
			).Build(),
			rule: kargoapi.StaticWebhookRule{
				Action: kargoapi.StaticWebhookActionRefresh,
				Targets: []kargoapi.StaticWebhookTarget{
					{
						Name:      "fake-warehouse",
						Namespace: testProjectName,
						Type:      kargoapi.StaticWebhookTargetTypeWarehouse,
					},
					{
						Name:      "nonexistent-warehouse",
						Namespace: testProjectName,
						Type:      kargoapi.StaticWebhookTargetTypeWarehouse,
					},
				},
			},
			req: func() *http.Request {
				return httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer([]byte(`{"foo":"bar"}`)),
				)
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rr.Code)
				require.JSONEq(
					t,
					`{"error":"failed to refresh 1 of 2 target(s)"}`,
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
			(&staticWebhookReceiver{
				baseWebhookReceiver: &baseWebhookReceiver{
					client:  testCase.client,
					project: testProjectName,
				},
				rule: testCase.rule,
			}).getHandler(requestBody)(w, testCase.req())

			testCase.assertions(t, w)
		})
	}
}
