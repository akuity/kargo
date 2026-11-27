package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
)

func Test_server_patchFreightMetadata(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: testProject.Name, Name: "fake-freight"},
		Status: kargoapi.FreightStatus{
			Metadata: map[string]apiextensionsv1.JSON{
				"kept":        {Raw: []byte(`"untouched"`)},
				"overwritten": {Raw: []byte(`"old"`)},
			},
		},
	}
	url := "/v1beta1/projects/" + testProject.Name +
		"/internal/freight/" + testFreight.Name + "/metadata"

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodPatch, url,
		[]restTestCase{
			{
				name:          "missing body",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testFreight),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name:          "empty metadata",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testFreight),
				body:          bytes.NewBufferString(`{"metadata":{}}`),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
					require.Contains(t, w.Body.String(), "metadata is required")
				},
			},
			{
				name:          "empty key",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testFreight),
				body:          bytes.NewBufferString(`{"metadata":{"":"value"}}`),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
					require.Contains(t, w.Body.String(), "must not be empty")
				},
			},
			{
				name:          "Freight not found",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				body:          bytes.NewBufferString(`{"metadata":{"key":"value"}}`),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "upserts metadata",
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, testFreight).
					WithStatusSubresource(testFreight),
				body: bytes.NewBufferString(
					`{"metadata":{"overwritten":"new","added":{"nested":[1,2]}}}`,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					freight := &kargoapi.Freight{}
					require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(testFreight), freight))
					require.Len(t, freight.Status.Metadata, 3)
					require.JSONEq(t, `"untouched"`, string(freight.Status.Metadata["kept"].Raw))
					require.JSONEq(t, `"new"`, string(freight.Status.Metadata["overwritten"].Raw))
					require.JSONEq(t, `{"nested":[1,2]}`, string(freight.Status.Metadata["added"].Raw))
				},
			},
		},
	)
}
