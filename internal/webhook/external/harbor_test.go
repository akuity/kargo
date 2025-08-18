package external

import (
	"bytes"
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

func TestHarborHandler(t *testing.T) {
	const testURL = "https://webhooks.kargo.example.com/nonsense"
	const testProjectName = "fake-project"
	const testAuthToken = "test-auth-token-123"

	validPushArtifactEvent := map[string]interface{}{
		"type":     harborEventTypePush,
		"occur_at": 1680501893,
		"operator": "harbor-jobservice",
		"event_data": map[string]interface{}{
			"resources": []map[string]interface{}{
				{
					"digest":       "sha256:954b378c375d852eb3c63ab88978f640b4348b01c1b3456a024a81536dafbbf4",
					"tag":          "v1.0.0",
					"resource_url": "harbor.example.com/library/alpine:v1.0.0",
				},
			},
			"repository": map[string]interface{}{
				"date_created":   1680501893,
				"name":           "alpine",
				"namespace":      "library",
				"repo_full_name": "library/alpine",
				"repo_type":      "private",
			},
		},
	}

	validPushArtifactEventMultipleResources := map[string]interface{}{
		"type":     harborEventTypePush,
		"occur_at": 1680501893,
		"operator": "harbor-jobservice",
		"event_data": map[string]interface{}{
			"resources": []map[string]interface{}{
				{
					"digest":       "sha256:954b378c375d852eb3c63ab88978f640b4348b01c1b3456a024a81536dafbbf4",
					"tag":          "v1.0.0",
					"resource_url": "harbor.example.com/library/alpine:v1.0.0",
				},
				{
					"digest":       "sha256:111b378c375d852eb3c63ab88978f640b4348b01c1b3456a024a81536dafbbf5",
					"tag":          "v1.1.0",
					"resource_url": "harbor.example.com/library/alpine:v1.1.0",
				},
			},
			"repository": map[string]interface{}{
				"date_created":   1680501893,
				"name":           "alpine",
				"namespace":      "library",
				"repo_full_name": "library/alpine",
				"repo_type":      "private",
			},
		},
	}

	validHelmChartPushEvent := map[string]interface{}{
		"type":     harborEventTypePush,
		"occur_at": 1680501893,
		"operator": "harbor-jobservice",
		"event_data": map[string]interface{}{
			"resources": []map[string]interface{}{
				{
					"digest":       "sha256:954b378c375d852eb3c63ab88978f640b4348b01c1b3456a024a81536dafbbf4",
					"tag":          "v1.0.0",
					"resource_url": "harbor.example.com/charts/my-chart:v1.0.0",
				},
			},
			"repository": map[string]interface{}{
				"date_created":   1680501893,
				"name":           "my-chart",
				"namespace":      "charts",
				"repo_full_name": "charts/my-chart",
				"repo_type":      "private",
			},
		},
	}

	testScheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(testScheme))

	testSecretData := map[string][]byte{
		harborSecretDataKey: []byte(testAuthToken),
	}

	testCases := []struct {
		name       string
		client     client.Client
		secretData map[string][]byte
		req        func() *http.Request
		assertions func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "secret missing from Secret data",
			req: func() *http.Request {
				return httptest.NewRequest(http.MethodPost, testURL, nil)
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rr.Code)
				require.JSONEq(t, "{}", rr.Body.String())
			},
		},
		{
			name:       "missing authorization header",
			secretData: testSecretData,
			req: func() *http.Request {
				body := bytes.NewBufferString(fmt.Sprintf(`{
					"type": "%s",
					"occur_at": 1680501893
				}`, harborEventTypePush))
				req := httptest.NewRequest(http.MethodPost, testURL, body)
				// No Authorization header set
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, rr.Code)
				require.JSONEq(t, `{"error":"missing authorization"}`, rr.Body.String())
			},
		},
		{
			name:       "invalid authorization header",
			secretData: testSecretData,
			req: func() *http.Request {
				body := bytes.NewBufferString(fmt.Sprintf(`{
					"type": "%s",
					"occur_at": 1680501893
				}`, harborEventTypePush))
				req := httptest.NewRequest(http.MethodPost, testURL, body)
				req.Header.Set(harborAuthHeader, "invalid-token")
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
				body := bytes.NewBufferString("invalid json")
				req := httptest.NewRequest(http.MethodPost, testURL, body)
				req.Header.Set(harborAuthHeader, testAuthToken)
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
				require.JSONEq(t, `{"error":"invalid request body"}`, rr.Body.String())
			},
		},
		{
			name:       "unsupported event type",
			secretData: testSecretData,
			req: func() *http.Request {
				body := bytes.NewBufferString(`{
					"type": "PULL_ARTIFACT",
					"occur_at": 1680501893
				}`)
				req := httptest.NewRequest(http.MethodPost, testURL, body)
				req.Header.Set(harborAuthHeader, testAuthToken)
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
				require.JSONEq(t, `{"error":"unsupported event type"}`, rr.Body.String())
			},
		},
		{
			name: "no tag match (image)",
			// This event would prompt the Warehouse to refresh if not for the tag
			// in the event falling outside the subscription's semver range.
			secretData: testSecretData,
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "fake-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Image: &kargoapi.ImageSubscription{
								RepoURL:          "harbor.example.com/library/alpine",
								SemverConstraint: "^2.0.0", // Constraint won't be met
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
				bodyBytes, err := json.Marshal(validPushArtifactEvent)
				require.NoError(t, err)
				req := httptest.NewRequest(http.MethodPost, testURL, bytes.NewBuffer(bodyBytes))
				req.Header.Set(harborAuthHeader, testAuthToken)
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 0 warehouse(s)"}`, rr.Body.String())
			},
		},
		{
			name:       "warehouse refreshed (image)",
			secretData: testSecretData,
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "fake-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Image: &kargoapi.ImageSubscription{
								RepoURL:          "harbor.example.com/library/alpine",
								SemverConstraint: "^1.0.0",
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
				bodyBytes, err := json.Marshal(validPushArtifactEvent)
				require.NoError(t, err)
				req := httptest.NewRequest(http.MethodPost, testURL, bytes.NewBuffer(bodyBytes))
				req.Header.Set(harborAuthHeader, testAuthToken)
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 1 warehouse(s)"}`, rr.Body.String())
			},
		},
		{
			name:       "warehouse refreshed (multiple resources)",
			secretData: testSecretData,
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "fake-warehouse-alpine",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Image: &kargoapi.ImageSubscription{
								RepoURL:          "harbor.example.com/library/alpine",
								SemverConstraint: "^1.0.0",
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
				bodyBytes, err := json.Marshal(validPushArtifactEventMultipleResources)
				require.NoError(t, err)
				req := httptest.NewRequest(http.MethodPost, testURL, bytes.NewBuffer(bodyBytes))
				req.Header.Set(harborAuthHeader, testAuthToken)
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 1 warehouse(s)"}`, rr.Body.String())
			},
		},
		{
			name: "no version match (helm chart)",
			// This event would prompt the Warehouse to refresh if not for the tag
			// in the event falling outside the subscription's semver range.
			secretData: testSecretData,
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "fake-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Chart: &kargoapi.ChartSubscription{
								RepoURL:          "oci://harbor.example.com/charts/my-chart",
								SemverConstraint: "^2.0.0", // Constraint won't be met
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
				bodyBytes, err := json.Marshal(validHelmChartPushEvent)
				require.NoError(t, err)
				req := httptest.NewRequest(http.MethodPost, testURL, bytes.NewBuffer(bodyBytes))
				req.Header.Set(harborAuthHeader, testAuthToken)
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 0 warehouse(s)"}`, rr.Body.String())
			},
		},
		{
			name:       "warehouse refreshed (helm chart)",
			secretData: testSecretData,
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "fake-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Chart: &kargoapi.ChartSubscription{
								RepoURL:          "oci://harbor.example.com/charts/my-chart",
								SemverConstraint: "^1.0.0",
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
				bodyBytes, err := json.Marshal(validHelmChartPushEvent)
				require.NoError(t, err)
				req := httptest.NewRequest(http.MethodPost, testURL, bytes.NewBuffer(bodyBytes))
				req.Header.Set(harborAuthHeader, testAuthToken)
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 1 warehouse(s)"}`, rr.Body.String())
			},
		},
		{
			name:       "empty resources array",
			secretData: testSecretData,
			req: func() *http.Request {
				body := bytes.NewBufferString(fmt.Sprintf(`{
					"type": "%s",
					"occur_at": 1680501893,
					"operator": "harbor-jobservice",
					"event_data": {
						"resources": [],
						"repository": {
							"date_created": 1680501893,
							"name": "alpine",
							"namespace": "library",
							"repo_full_name": "library/alpine",
							"repo_type": "private"
						}
					}
				}`, harborEventTypePush))
				req := httptest.NewRequest(http.MethodPost, testURL, body)
				req.Header.Set(harborAuthHeader, testAuthToken)
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 0 warehouse(s)"}`, rr.Body.String())
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
			(&harborWebhookReceiver{
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
