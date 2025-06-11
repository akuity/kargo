package external

import (
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/akuity/kargo/internal/indexer"
)

const dockerhubWebhookRequestBody = `
{
	"repository": {
		"repo_name": "example/repo"
	}
}`

func TestDockerHubHandler(t *testing.T) {
	const testURL = "https://webhooks.kargo.example.com/nonsense"

	const testProjectName = "fake-project"

	testScheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(testScheme))

	const testToken = "mysupersecrettoken"
	testSecretData := map[string][]byte{
		dockerhubSecretDataKey: []byte(testToken),
	}

	testCases := []struct {
		name       string
		client     client.Client
		secretData map[string][]byte
		req        func() *http.Request
		assertions func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:       "request body too large",
			secretData: testSecretData,
			req: func() *http.Request {
				body := make([]byte, 2<<20+1)
				return httptest.NewRequest(
					http.MethodPost,
					testURL,
					io.NopCloser(bytes.NewBuffer(body)),
				)
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusRequestEntityTooLarge, rr.Code)
				res := map[string]string{}
				err := json.Unmarshal(rr.Body.Bytes(), &res)
				require.NoError(t, err)
				require.Contains(t, res["error"], "content exceeds limit")
			},
		},
		{
			name:       "malformed request body",
			secretData: testSecretData,
			req: func() *http.Request {
				bodyBuf := bytes.NewBuffer([]byte("invalid json"))
				return httptest.NewRequest(http.MethodPost, testURL, bodyBuf)
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
				require.JSONEq(t, `{"error":"invalid request body"}`, rr.Body.String())
			},
		},
		{
			name:       "partial success",
			secretData: testSecretData,
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "fake-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Image: &kargoapi.ImageSubscription{RepoURL: "example/repo"},
						}},
					},
				},
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "another-fake-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Image: &kargoapi.ImageSubscription{RepoURL: "example/repo"},
						}},
					},
				},
			).WithInterceptorFuncs(interceptor.Funcs{
				Patch: func(
					_ context.Context,
					_ client.WithWatch,
					obj client.Object,
					_ client.Patch,
					_ ...client.PatchOption,
				) error {
					if obj.GetName() == "another-fake-warehouse" {
						return errors.New("something went wrong")
					}
					return nil
				},
			}).WithIndex(
				&kargoapi.Warehouse{},
				indexer.WarehousesBySubscribedURLsField,
				indexer.WarehousesBySubscribedURLs,
			).Build(),
			req: func() *http.Request {
				bodyBuf := bytes.NewBuffer([]byte(dockerhubWebhookRequestBody))
				return httptest.NewRequest(http.MethodPost, testURL, bodyBuf)
			},
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rr.Code)
				require.JSONEq(
					t,
					`{"error":"failed to refresh 1 of 2 warehouses"}`,
					rr.Body.String(),
				)
			},
		},
		{
			name: "complete success",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      "fake-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Image: &kargoapi.ImageSubscription{RepoURL: "example/repo"},
						}},
					},
				},
			).WithIndex(
				&kargoapi.Warehouse{},
				indexer.WarehousesBySubscribedURLsField,
				indexer.WarehousesBySubscribedURLs,
			).Build(),
			req: func() *http.Request {
				bodyBuf := bytes.NewBuffer([]byte(dockerhubWebhookRequestBody))
				return httptest.NewRequest(http.MethodPost, testURL, bodyBuf)
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
			w := httptest.NewRecorder()
			(&dockerhubWebhookReceiver{
				baseWebhookReceiver: &baseWebhookReceiver{
					client:     testCase.client,
					project:    testProjectName,
					secretData: testCase.secretData,
				},
			}).GetHandler()(w, testCase.req())
			testCase.assertions(t, w)
		})
	}
}

func TestNormalizeDockerImageRef(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		// Docker Hub normalization cases
		{"simple repo (nginx)", "nginx", "docker.io/library/nginx:latest", false},
		{"simple repo (Ubuntu, uppercase)", "Ubuntu", "docker.io/library/ubuntu:latest", false},
		{"LiBrArY/NGINX (mixed case library)", "LiBrArY/NGINX", "docker.io/library/nginx:latest", false},
		{"docker.io/library/nginx (explicit)", "docker.io/library/nginx", "docker.io/library/nginx:latest", false},
		{"docker.io/nginx (implicit library)", "docker.io/nginx", "docker.io/library/nginx:latest", false},
		{"nginx with tag", "nginx:1.25", "docker.io/library/nginx:1.25", false},
		{"docker.io/library/nginx with tag", "docker.io/library/nginx:1.25", "docker.io/library/nginx:1.25", false},
		{"docker.io/foo/bar (default tag)", "docker.io/foo/bar", "docker.io/foo/bar:latest", false},
		{"implicit docker.io with namespace", "foo/bar", "docker.io/foo/bar:latest", false},
		{"personal repo (foo/app)", "foo/app", "docker.io/foo/app:latest", false},
		{"personal repo with tag (foo/app:v1.2)", "foo/app:v1.2", "docker.io/foo/app:v1.2", false},
		{"fully qualified docker.io with user", "docker.io/foo/app:v2.0", "docker.io/foo/app:v2.0", false},
		{"explicit docker.io/library/alpine", "docker.io/library/alpine", "docker.io/library/alpine:latest", false},
		{"repo with multiple slashes and tag", "docker.io/org/namespace/app:dev", "docker.io/org/namespace/app:dev", false},
		{"UPPERCASE/INVALID (normalizes to lowercase)", "UPPERCASE/INVALID", "docker.io/uppercase/invalid:latest", false},
		{"single character repo", "a", "docker.io/library/a:latest", false},
		{"repo with dash", "foo-bar", "docker.io/library/foo-bar:latest", false},
		{"repo with underscore", "foo_bar", "docker.io/library/foo_bar:latest", false},
		{"repo with dot", "foo.bar", "docker.io/library/foo.bar:latest", false},
		{"repo with leading/trailing whitespace", "  nginx  ", "docker.io/library/nginx:latest", false},

		// Invalid cases for Docker Hub
		{"invalid characters in name", "invalid!name", "", true},
		{"invalid characters in full ref", "docker.io/Invalid$$Name", "", true},
		{"empty input", "", "", true},
		{"repo with trailing slash", "nginx/", "", true},
		{"repo with trailing colon but no tag", "nginx:", "", true},
		{"too many path components", "foo/bar/baz/qux", "docker.io/foo/bar/baz/qux:latest", false},
		{"malformed digest", "nginx@sha256", "", true},

		// Digest-based reference for Docker Hub
		{
			name:     "digest-based ref (nginx@sha256)",
			input:    "nginx@sha256:123abcdeffedcba3210987654321abcdefabcdefabcdefabcdefabcdefabcdef",
			expected: "docker.io/library/nginx@sha256:123abcdeffedcba3210987654321abcdefabcdefabcdefabcdefabcdefabcdef",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeDockerImageRef(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, got)
			}
		})
	}
}
