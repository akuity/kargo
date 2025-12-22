package external

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	gh "github.com/google/go-github/v76/github"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/indexer"
	"github.com/akuity/kargo/pkg/logging"
)

const githubSigningKey = "mysupersecrettoken"

func TestGithubHandler(t *testing.T) {
	const testURL = "https://webhooks.kargo.example.com/nonsense"

	const testProjectName = "fake-project"

	validPushEvent := &gh.PushEvent{
		Ref: gh.Ptr("refs/heads/main"),
		Repo: &gh.PushEventRepository{
			CloneURL: gh.Ptr("https://github.com/example/repo"),
		},
	}
	validPushEventSshRepoURL := &gh.PushEvent{
		Ref: gh.Ptr("refs/heads/main"),
		Repo: &gh.PushEventRepository{
			SSHURL: gh.Ptr("git@github.com:user/repo.git"),
		},
	}
	validPackageEventImage := &gh.PackageEvent{
		Action: gh.Ptr("published"),
		Package: &gh.Package{
			PackageType: gh.Ptr(ghcrPackageTypeContainer),
			PackageVersion: &gh.PackageVersion{
				PackageURL: gh.Ptr("ghcr.io/example/repo:latest"),
				ContainerMetadata: &gh.PackageEventContainerMetadata{
					Tag: &gh.PackageEventContainerMetadataTag{
						Name: gh.Ptr("v1.0.0"),
					},
					Manifest: map[string]any{
						"config": map[string]any{
							// Real world testing shows this media type is what the payload
							// will contain when an image has been pushed to GHCR.
							"media_type": dockerImageConfigBlobMediaType,
						},
					},
				},
			},
		},
	}
	validPackageEventChart := &gh.PackageEvent{
		Action: gh.Ptr("published"),
		Package: &gh.Package{
			PackageType: gh.Ptr(ghcrPackageTypeContainer),
			PackageVersion: &gh.PackageVersion{
				PackageURL: gh.Ptr("ghcr.io/example/repo:latest"),
				ContainerMetadata: &gh.PackageEventContainerMetadata{
					Tag: &gh.PackageEventContainerMetadataTag{
						Name: gh.Ptr("v1.0.0"),
					},
					Manifest: map[string]any{
						"config": map[string]any{
							// Real world testing shows this media type is what the payload
							// will contain when an image has been pushed to GHCR.
							"media_type": helmChartConfigBlobMediaType,
						},
					},
				},
			},
		},
	}

	validRegistryPackageEventImage := &gh.RegistryPackageEvent{
		Action: gh.Ptr("published"),
		RegistryPackage: &gh.Package{
			PackageType: gh.Ptr(ghcrPackageTypeContainer),
			PackageVersion: &gh.PackageVersion{
				PackageURL: gh.Ptr("ghcr.io/example/repo:latest"),
				ContainerMetadata: &gh.PackageEventContainerMetadata{
					Tag: &gh.PackageEventContainerMetadataTag{
						Name: gh.Ptr("v1.0.0"),
					},
					Manifest: map[string]any{
						"config": map[string]any{
							// Real world testing shows this media type is what the payload
							// will contain when an image has been pushed to GHCR.
							"media_type": dockerImageConfigBlobMediaType,
						},
					},
				},
			},
		},
	}

	validRegistryPackageEventChart := &gh.RegistryPackageEvent{
		Action: gh.Ptr("published"),
		RegistryPackage: &gh.Package{
			PackageType: gh.Ptr(ghcrPackageTypeContainer),
			PackageVersion: &gh.PackageVersion{
				PackageURL: gh.Ptr("ghcr.io/example/repo:latest"),
				ContainerMetadata: &gh.PackageEventContainerMetadata{
					Tag: &gh.PackageEventContainerMetadataTag{
						Name: gh.Ptr("v1.0.0"),
					},
					Manifest: map[string]any{
						"config": map[string]any{
							// Real world testing shows this media type is what the payload
							// will contain when an image has been pushed to GHCR.
							"media_type": helmChartConfigBlobMediaType,
						},
					},
				},
			},
		},
	}

	testScheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(testScheme))

	testSecretData := map[string][]byte{
		GithubSecretDataKey: []byte(githubSigningKey),
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
				req.Header.Set(gh.EventTypeHeader, "nonsense")
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
				req.Header.Set(gh.EventTypeHeader, githubEventTypePush)
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
				req.Header.Set(gh.EventTypeHeader, githubEventTypePush)
				req.Header.Set(gh.SHA256SignatureHeader, "totally-invalid-signature")
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
				req.Header.Set(gh.EventTypeHeader, githubEventTypePush)
				req.Header.Set(gh.SHA256SignatureHeader, sign(bodyBuf.Bytes()))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
				require.JSONEq(t, `{"error":"invalid request body"}`, rr.Body.String())
			},
		},
		{
			name:       "successful ping event",
			secretData: testSecretData,
			req: func() *http.Request {
				bodyBuf := bytes.NewBuffer([]byte("{}"))
				req := httptest.NewRequest(http.MethodPost, testURL, bodyBuf)
				req.Header.Set(gh.EventTypeHeader, githubEventTypePing)
				req.Header.Set(gh.SHA256SignatureHeader, sign(bodyBuf.Bytes()))
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
			name:       "unsupported action (package event)",
			secretData: testSecretData,
			req: func() *http.Request {
				bodyBytes, err := json.Marshal(
					&gh.PackageEvent{Action: gh.Ptr("deleted")},
				)
				require.NoError(t, err)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(bodyBytes),
				)
				req.Header.Set(gh.EventTypeHeader, githubEventTypePackage)
				req.Header.Set(gh.SHA256SignatureHeader, sign(bodyBytes))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, "{}", rr.Body.String())
			},
		},
		{
			name:       "package missing (package event)",
			secretData: testSecretData,
			req: func() *http.Request {
				bodyBytes, err := json.Marshal(
					&gh.PackageEvent{Action: gh.Ptr("published")},
				)
				require.NoError(t, err)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(bodyBytes),
				)
				req.Header.Set(gh.EventTypeHeader, githubEventTypePackage)
				req.Header.Set(gh.SHA256SignatureHeader, sign(bodyBytes))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
				require.JSONEq(t, `{"error":"invalid request body"}`, rr.Body.String())
			},
		},
		{
			name:       "unsupported package type (package event)",
			secretData: testSecretData,
			req: func() *http.Request {
				bodyBytes, err := json.Marshal(
					&gh.PackageEvent{
						Action: gh.Ptr("published"),
						Package: &gh.Package{
							PackageType:    gh.Ptr("npm"),
							PackageVersion: &gh.PackageVersion{},
						},
					},
				)
				require.NoError(t, err)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(bodyBytes),
				)
				req.Header.Set(gh.EventTypeHeader, githubEventTypePackage)
				req.Header.Set(gh.SHA256SignatureHeader, sign(bodyBytes))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, "{}", rr.Body.String())
			},
		},
		{
			name: "no tag match (package event, image)",
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
								RepoURL:    "ghcr.io/example/repo",
								Constraint: "^v2.0.0", // Constraint won't be met
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
				bodyBytes, err := json.Marshal(validPackageEventImage)
				require.NoError(t, err)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(bodyBytes),
				)
				req.Header.Set(gh.EventTypeHeader, githubEventTypePackage)
				req.Header.Set(gh.SHA256SignatureHeader, sign(bodyBytes))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 0 warehouse(s)"}`, rr.Body.String())
			},
		},
		{
			name:       "warehouse refreshed (package event, image)",
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
								RepoURL:    "ghcr.io/example/repo",
								Constraint: "^v1.0.0",
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
				bodyBytes, err := json.Marshal(validPackageEventImage)
				require.NoError(t, err)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(bodyBytes),
				)
				req.Header.Set(gh.EventTypeHeader, githubEventTypePackage)
				req.Header.Set(gh.SHA256SignatureHeader, sign(bodyBytes))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 1 warehouse(s)"}`, rr.Body.String())
			},
		},
		{
			name: "no version match (package event, chart)",
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
								RepoURL:          "oci://ghcr.io/example/repo",
								SemverConstraint: "^v2.0.0", // Constraint won't be met
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
				bodyBytes, err := json.Marshal(validPackageEventChart)
				require.NoError(t, err)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(bodyBytes),
				)
				req.Header.Set(gh.EventTypeHeader, githubEventTypePackage)
				req.Header.Set(gh.SHA256SignatureHeader, sign(bodyBytes))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 0 warehouse(s)"}`, rr.Body.String())
			},
		},
		{
			name:       "warehouse refreshed (package event, chart)",
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
								RepoURL:          "oci://ghcr.io/example/repo",
								SemverConstraint: "^v1.0.0",
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
				bodyBytes, err := json.Marshal(validPackageEventChart)
				require.NoError(t, err)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(bodyBytes),
				)
				req.Header.Set(gh.EventTypeHeader, githubEventTypePackage)
				req.Header.Set(gh.SHA256SignatureHeader, sign(bodyBytes))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 1 warehouse(s)"}`, rr.Body.String())
			},
		},
		{
			name: "no ref match (push event, git)",
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
								RepoURL: "https://github.com/example/repo",
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
				bodyBytes, err := json.Marshal(validPushEvent)
				require.NoError(t, err)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(bodyBytes),
				)
				req.Header.Set(gh.EventTypeHeader, githubEventTypePush)
				req.Header.Set(gh.SHA256SignatureHeader, sign(bodyBytes))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 0 warehouse(s)"}`, rr.Body.String())
			},
		},
		{
			name:       "warehouse refreshed (push event, git, https)",
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
								RepoURL: "https://github.com/example/repo",
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
				bodyBytes, err := json.Marshal(validPushEvent)
				require.NoError(t, err)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(bodyBytes),
				)
				req.Header.Set(gh.EventTypeHeader, githubEventTypePush)
				req.Header.Set(gh.SHA256SignatureHeader, sign(bodyBytes))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 1 warehouse(s)"}`, rr.Body.String())
			},
		},
		{
			name:       "warehouse refreshed (push event, git, ssh)",
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
								RepoURL: "git@github.com:user/repo.git",
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
				bodyBytes, err := json.Marshal(validPushEventSshRepoURL)
				require.NoError(t, err)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(bodyBytes),
				)
				req.Header.Set(gh.EventTypeHeader, githubEventTypePush)
				req.Header.Set(gh.SHA256SignatureHeader, sign(bodyBytes))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 1 warehouse(s)"}`, rr.Body.String())
			},
		},
		{
			name:       "unsupported action (registry_package event)",
			secretData: testSecretData,
			req: func() *http.Request {
				bodyBytes, err := json.Marshal(
					&gh.RegistryPackageEvent{Action: gh.Ptr("deleted")},
				)
				require.NoError(t, err)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(bodyBytes),
				)
				req.Header.Set(gh.EventTypeHeader, githubEventTypeRegistryPackage)
				req.Header.Set(gh.SHA256SignatureHeader, sign(bodyBytes))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, "{}", rr.Body.String())
			},
		},
		{
			name:       "registry package missing (registry_package event)",
			secretData: testSecretData,
			req: func() *http.Request {
				bodyBytes, err := json.Marshal(
					&gh.RegistryPackageEvent{Action: gh.Ptr("published")},
				)
				require.NoError(t, err)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(bodyBytes),
				)
				req.Header.Set(gh.EventTypeHeader, githubEventTypeRegistryPackage)
				req.Header.Set(gh.SHA256SignatureHeader, sign(bodyBytes))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
				require.JSONEq(t, `{"error":"invalid request body"}`, rr.Body.String())
			},
		},
		{
			name:       "unsupported package type (registry_package event)",
			secretData: testSecretData,
			req: func() *http.Request {
				bodyBytes, err := json.Marshal(
					&gh.RegistryPackageEvent{
						Action: gh.Ptr("published"),
						RegistryPackage: &gh.Package{
							PackageType:    gh.Ptr("npm"),
							PackageVersion: &gh.PackageVersion{},
						},
					},
				)
				require.NoError(t, err)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(bodyBytes),
				)
				req.Header.Set(gh.EventTypeHeader, githubEventTypeRegistryPackage)
				req.Header.Set(gh.SHA256SignatureHeader, sign(bodyBytes))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, "{}", rr.Body.String())
			},
		},
		{
			name: "no tag match (registry_package event, image)",
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
								RepoURL:    "ghcr.io/example/repo",
								Constraint: "^v2.0.0", // Constraint won't be met
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
				bodyBytes, err := json.Marshal(validRegistryPackageEventImage)
				require.NoError(t, err)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(bodyBytes),
				)
				req.Header.Set(gh.EventTypeHeader, githubEventTypeRegistryPackage)
				req.Header.Set(gh.SHA256SignatureHeader, sign(bodyBytes))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 0 warehouse(s)"}`, rr.Body.String())
			},
		},
		{
			name:       "warehouse refreshed -- (registry_package event, image)",
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
								RepoURL:    "ghcr.io/example/repo",
								Constraint: "^v1.0.0",
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
				bodyBytes, err := json.Marshal(validRegistryPackageEventImage)
				require.NoError(t, err)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(bodyBytes),
				)
				req.Header.Set(gh.EventTypeHeader, githubEventTypeRegistryPackage)
				req.Header.Set(gh.SHA256SignatureHeader, sign(bodyBytes))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 1 warehouse(s)"}`, rr.Body.String())
			},
		},
		{
			name: "no version match (registry_package event, chart)",
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
								RepoURL:          "oci://ghcr.io/example/repo",
								SemverConstraint: "^v2.0.0", // Constraint won't be met
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
				bodyBytes, err := json.Marshal(validRegistryPackageEventChart)
				require.NoError(t, err)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(bodyBytes),
				)
				req.Header.Set(gh.EventTypeHeader, githubEventTypeRegistryPackage)
				req.Header.Set(gh.SHA256SignatureHeader, sign(bodyBytes))
				return req
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 0 warehouse(s)"}`, rr.Body.String())
			},
		},
		{
			name:       "warehouse refreshed (registry_package event, chart)",
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
								RepoURL:          "oci://ghcr.io/example/repo",
								SemverConstraint: "^v1.0.0",
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
				bodyBytes, err := json.Marshal(validRegistryPackageEventChart)
				require.NoError(t, err)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					bytes.NewBuffer(bodyBytes),
				)
				req.Header.Set(gh.EventTypeHeader, githubEventTypeRegistryPackage)
				req.Header.Set(gh.SHA256SignatureHeader, sign(bodyBytes))
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

			logger := logging.NewLoggerOrDie(logging.DebugLevel, logging.DefaultFormat)
			ctx := logging.ContextWithLogger(testCase.req().Context(), logger)

			w := httptest.NewRecorder()
			(&githubWebhookReceiver{
				baseWebhookReceiver: &baseWebhookReceiver{
					client:     testCase.client,
					project:    testProjectName,
					secretData: testCase.secretData,
				},
			}).getHandler(requestBody)(w, testCase.req().WithContext(ctx))

			testCase.assertions(t, w)
		})
	}
}
