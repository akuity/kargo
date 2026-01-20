package external

import (
	"bytes"
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
	"github.com/akuity/kargo/pkg/indexer"
)

func TestAzureHandler(t *testing.T) {
	const testURL = "https://webhooks.kargo.example.com/nonsense"

	const testProjectName = "fake-project"

	testScheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(testScheme))

	testCases := []struct {
		name       string
		client     client.Client
		req        func() *http.Request
		assertions func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "malformed request body",
			req: func() *http.Request {
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer([]byte("invalid json")),
				)
				req.Header.Set("User-Agent", acrUserAgentPrefix)
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
				require.JSONEq(t, `{"error":"invalid request body"}`, rr.Body.String())
			},
		},
		{
			name: "unsupported user agent",
			req: func() *http.Request {
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					newAzurePayload(acrPingEvent, ""),
				)
				req.Header.Set("User-Agent", "invalid-user-agent")
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
				require.JSONEq(t,
					`{"error":"request does not appear to have originated from a supported service"}`,
					rr.Body.String(),
				)
			},
		},
		{
			name: "successful ping",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "fake-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Image: &kargoapi.ImageSubscription{
								RepoURL: "fakeregistry.azurecr.io/fakeimage",
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
					newAzurePayload(acrPingEvent, ""),
				)
				req.Header.Set("User-Agent", acrUserAgentPrefix)
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
			name: "no tag match (image)",
			// This event would prompt the Warehouse to refresh if not for the tag
			// in the event falling outside the subscription's semver range.
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "fake-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Image: &kargoapi.ImageSubscription{
								RepoURL:    "fakeregistry.azurecr.io/fakeimage",
								Constraint: "^2.0.0", // Constraint won't be met
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
					// Real world testing shows this media type is what the payload will
					// contain when an image has been pushed to ACR.
					newAzurePayload(acrPushEvent, dockerManifestMediaType),
				)
				req.Header.Set("User-Agent", acrUserAgentPrefix)
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
			name: "warehouse refreshed (image)",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "fake-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Image: &kargoapi.ImageSubscription{
								RepoURL:    "fakeregistry.azurecr.io/fakeimage",
								Constraint: "^1.0.0",
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
					// Real world testing shows this media type is what the payload will
					// contain when an image has been pushed to ACR.
					newAzurePayload(acrPushEvent, dockerManifestMediaType),
				)
				req.Header.Set("User-Agent", acrUserAgentPrefix)
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
			name: "no version match (chart)",
			// This event would prompt the Warehouse to refresh if not for the tag
			// in the event falling outside the subscription's semver range.
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "fake-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Chart: &kargoapi.ChartSubscription{
								RepoURL:          "oci://fakeregistry.azurecr.io/fakeimage",
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
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					// Real world testing shows this media type is what the payload will
					// contain when a Helm chart is pushed to ACR.
					newAzurePayload(acrPushEvent, ociImageManifestMediaType),
				)
				req.Header.Set("User-Agent", acrUserAgentPrefix)
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
			name: "warehouse refreshed (chart)",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "fake-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Chart: &kargoapi.ChartSubscription{
								RepoURL:          "oci://fakeregistry.azurecr.io/fakeimage",
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
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					// Real world testing shows this media type is what the payload will
					// contain when a Helm chart is pushed to ACR.
					newAzurePayload(acrPushEvent, ociImageManifestMediaType),
				)
				req.Header.Set("User-Agent", acrUserAgentPrefix)
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
			name: "no ref match (git)",
			// This event would prompt the Warehouse to refresh if not for the ref in
			// the event being for the main branch whilst the subscription is
			// interested in commits from a different branch.
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "fake-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{
								RepoURL: "https://dev.azure.com/testorg/testproject/_git/testrepo",
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
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					newAzurePayload(azureDevOpsPushEvent, ""),
				)
				req.Header.Set("User-Agent", azureDevOpsUserAgentPrefix)
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
			name: "warehouse refreshed (git)",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "fake-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{
								RepoURL: "https://dev.azure.com/testorg/testproject/_git/testrepo",
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
					newAzurePayload(azureDevOpsPushEvent, ""),
				)
				req.Header.Set("User-Agent", azureDevOpsUserAgentPrefix)
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
			(&azureWebhookReceiver{
				baseWebhookReceiver: &baseWebhookReceiver{
					client:  testCase.client,
					project: testProjectName,
				},
			}).getHandler(requestBody)(w, testCase.req())

			testCase.assertions(t, w)
		})
	}
}

func newAzurePayload(event, mediaType string) *bytes.Buffer {
	switch event {
	case acrPingEvent:
		return bytes.NewBufferString(`{"action": "ping"}`)
	case acrPushEvent:
		return bytes.NewBufferString(fmt.Sprintf(`
			{
				"action": "push",
				"target": {
					"repository": "fakeimage",
					"mediaType": %q,
					"tag": "v1.0.0"
				},
				"request": {"host": "fakeregistry.azurecr.io"}
			}`, mediaType,
		))
	case azureDevOpsPushEvent:
		return bytes.NewBufferString(`
		{
			"eventType": "git.push",
			"resource": {
				"refUpdates": [
					{
						"name": "refs/heads/main"
					}
				],
				"repository": {
					"remoteUrl": "https://dev.azure.com/testorg/testproject/_git/testrepo"
				}
			}
		}`)
	default:
		return bytes.NewBufferString(`{"action": "unknown"}`)
	}
}
