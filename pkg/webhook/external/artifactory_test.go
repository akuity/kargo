package external

import (
	"bytes"
	"encoding/json"
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

func TestArtifactoryHandler(t *testing.T) {
	const testURL = "https://webhooks.kargo.example.com/nonsense"

	const testProjectName = "fake-project"

	validImagePushEvent := artifactoryEvent{
		Domain:    artifactoryDockerDomain,
		EventType: artifactoryPushedEventType,
		Data: artifactoryEventData{
			Tag:       "v1.0.0",
			Path:      "test-image/latest/manifest.json",
			ImageType: artifactoryDockerDomain,
			RepoKey:   "test-repo",
			ImageName: "test-image",
		},
		Origin: "https://artifactory.example.com",
	}

	validImagePushEventWithPathPrefix := artifactoryEvent{
		Domain:    artifactoryDockerDomain,
		EventType: artifactoryPushedEventType,
		Data: artifactoryEventData{
			Tag:       "v1.0.0",
			Path:      "foo/bar/test-image/latest/manifest.json",
			ImageType: artifactoryDockerDomain,
			RepoKey:   "test-repo",
			ImageName: "test-image",
		},
		Origin: "https://artifactory.example.com",
	}

	validImagePushEventWithUnsetOrigin := validImagePushEvent
	validImagePushEventWithUnsetOrigin.Origin = ""

	validChartPushEvent := artifactoryEvent{
		Domain:    artifactoryDockerDomain,
		EventType: artifactoryPushedEventType,
		Data: artifactoryEventData{
			Tag:       "v1.0.0",
			Path:      "test-chart/latest/chart.tgz",
			ImageType: artifactoryChartImageType,
			RepoKey:   "test-repo",
			ImageName: "test-chart",
		},
		Origin: "https://artifactory.example.com",
	}

	testScheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(testScheme))

	testSecretData := map[string][]byte{
		artifactorySecretDataKey: []byte(testSigningKey),
	}

	testCases := []struct {
		name            string
		client          client.Client
		secretData      map[string][]byte
		virtualRepoName string
		req             func() *http.Request
		assertions      func(*testing.T, *httptest.ResponseRecorder)
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
				body := bytes.NewBufferString(`{
					"event_type":"nonsense",
					"domain":"docker"
				}`)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					body,
				)
				req.Header.Set(artifactoryAuthHeader, signWithoutAlgoPrefix(body.Bytes()))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotImplemented, rr.Code)
				require.JSONEq(
					t,
					`{"error":"event type must be 'pushed' and domain must be 'docker'"}`,
					rr.Body.String(),
				)
			},
		},
		{
			name:       "unsupported domain",
			secretData: testSecretData,
			req: func() *http.Request {
				body := bytes.NewBufferString(`{
					"event_type":"nonsense", 
					"domain":"nonsense"
				}`)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					body,
				)
				req.Header.Set(artifactoryAuthHeader, signWithoutAlgoPrefix(body.Bytes()))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotImplemented, rr.Code)
				require.JSONEq(
					t,
					`{"error":"event type must be 'pushed' and domain must be 'docker'"}`,
					rr.Body.String(),
				)
			},
		},
		{
			name:       "missing signature",
			secretData: testSecretData,
			req: func() *http.Request {
				req := httptest.NewRequest(http.MethodPost, testURL, nil)
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
				req.Header.Set(artifactoryAuthHeader, "invalid-signature")
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
				body := []byte("invalid json")
				bodyBuf := bytes.NewBuffer(body)
				req := httptest.NewRequest(http.MethodPost, testURL, bodyBuf)
				req.Header.Set(artifactoryAuthHeader, signWithoutAlgoPrefix(body))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
				require.JSONEq(t, `{"error":"invalid request body"}`, rr.Body.String())
			},
		},
		{
			name:       "prevent panic if path sections are less than 2",
			secretData: testSecretData,
			req: func() *http.Request {
				validImagePushEventWithInvalidPath := validImagePushEvent
				validImagePushEventWithInvalidPath.Data.Path = "invalidpath"
				bodyBytes, err := json.Marshal(validImagePushEventWithInvalidPath)
				require.NoError(t, err)
				bodyBuf := bytes.NewBuffer(bodyBytes)
				req := httptest.NewRequest(http.MethodPost, testURL, bodyBuf)
				req.Header.Set(artifactoryAuthHeader, signWithoutAlgoPrefix(bodyBytes))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
				require.JSONEq(t, `{"error":"invalid value \"invalidpath\" in payload's data.path field"}`,
					rr.Body.String(),
				)
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
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Image: &kargoapi.ImageSubscription{
								RepoURL:    "artifactory.example.com/test-repo/test-image",
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
				bodyBytes, err := json.Marshal(validImagePushEvent)
				require.NoError(t, err)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(bodyBytes),
				)
				req.Header.Set(artifactoryAuthHeader, signWithoutAlgoPrefix(bodyBytes))
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
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Image: &kargoapi.ImageSubscription{
								// Artifactory supports nested repository structures
								RepoURL:    "artifactory.example.com/test-repo/foo/bar/test-image",
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
				bodyBytes, err := json.Marshal(validImagePushEventWithPathPrefix)
				require.NoError(t, err)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(bodyBytes),
				)
				req.Header.Set(artifactoryAuthHeader, signWithoutAlgoPrefix(bodyBytes))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 1 warehouse(s)"}`, rr.Body.String())
			},
		},
		{
			name: "no version match (chart)",
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
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Chart: &kargoapi.ChartSubscription{
								RepoURL:          "oci://artifactory.example.com/test-repo/test-chart",
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
				bodyBytes, err := json.Marshal(validChartPushEvent)
				require.NoError(t, err)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(bodyBytes),
				)
				req.Header.Set(artifactoryAuthHeader, signWithoutAlgoPrefix(bodyBytes))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 0 warehouse(s)"}`, rr.Body.String())
			},
		},
		{
			name:       "warehouse refreshed (chart)",
			secretData: testSecretData,
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "fake-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Chart: &kargoapi.ChartSubscription{
								RepoURL:          "oci://artifactory.example.com/test-repo/test-chart",
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
				bodyBytes, err := json.Marshal(validChartPushEvent)
				require.NoError(t, err)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(bodyBytes),
				)
				req.Header.Set(artifactoryAuthHeader, signWithoutAlgoPrefix(bodyBytes))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 1 warehouse(s)"}`, rr.Body.String())
			},
		},
		{
			name:            "warehouse refreshed (virtual repo)",
			secretData:      testSecretData,
			virtualRepoName: "virtual-test-repo",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "fake-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Chart: &kargoapi.ChartSubscription{
								RepoURL:          "oci://artifactory.example.com/virtual-test-repo/foo/bar/test-chart",
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
				validNestedChartPushEvent := validChartPushEvent
				validNestedChartPushEvent.Data.Path = "foo/bar/test-chart/latest/chart.tgz"
				bodyBytes, err := json.Marshal(validNestedChartPushEvent)
				require.NoError(t, err)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(bodyBytes),
				)
				req.Header.Set(artifactoryAuthHeader, signWithoutAlgoPrefix(bodyBytes))
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
			(&artifactoryWebhookReceiver{
				baseWebhookReceiver: &baseWebhookReceiver{
					client:     testCase.client,
					project:    testProjectName,
					secretData: testCase.secretData,
				},
				virtualRepoName: testCase.virtualRepoName,
			}).getHandler(requestBody)(w, testCase.req())

			testCase.assertions(t, w)
		})
	}
}
