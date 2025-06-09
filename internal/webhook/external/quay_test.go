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

func TestQuayHandler(t *testing.T) {
	const testURL = "https://webhooks.kargo.example.com/nonsense"

	const testProjectName = "fake-project"

	testScheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(testScheme))

	const testToken = "mysupersecrettoken"
	testSecretData := map[string][]byte{
		quaySecretDataKey: []byte(testToken),
	}

	for _, testCase := range []struct {
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
				body := make([]byte, quayWebhookBodyMaxBytes+1)
				req := httptest.NewRequest(
					http.MethodPost,
					testURL,
					io.NopCloser(bytes.NewBuffer(body)),
				)
				return req
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
							Image: &kargoapi.ImageSubscription{
								RepoURL: "quay.io/mynamespace/repository",
							},
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
							Image: &kargoapi.ImageSubscription{
								RepoURL: "quay.io/mynamespace/repository",
							},
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
					println("object:", obj.GetName())
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
				return httptest.NewRequest(
					http.MethodPost,
					testURL,
					newQuayPayload(),
				)
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
			name:       "success",
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
								RepoURL: "quay.io/mynamespace/repository",
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
					newQuayPayload(),
				)
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
	} {
		t.Run(testCase.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			(&quayWebhookReceiver{
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

func newQuayPayload() *bytes.Buffer {
	return bytes.NewBufferString(`
		{
			"docker_url": "quay.io/mynamespace/repository"
		  }
	`)
}
