package server

import (
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

func Test_server_createFreight(t *testing.T) {
	const project = "fake-project"
	const freightName = "fake-freight"
	const metadataKey = "meta-field"

	testProject := &kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: project}}
	newFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Name: freightName},
		Origin: kargoapi.FreightOrigin{
			Kind: kargoapi.FreightOriginKindWarehouse,
			Name: "test-warehouse",
		},
	}
	// Pre-existing Freight that already carries metadata, to exercise idempotency
	// and the fill-if-missing behavior.
	existingFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: project, Name: freightName},
		Status: kargoapi.FreightStatus{
			Metadata: map[string]apiextensionsv1.JSON{
				metadataKey: {Raw: []byte(`"original"`)},
			},
		},
	}

	url := "/v1beta1/projects/" + project + "/freight"

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodPost, url,
		[]restTestCase{
			{
				name:          "freight is required",
				body:          mustJSONBody(createFreightRequest{}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "creates Freight",
				body: mustJSONBody(createFreightRequest{Freight: newFreight}),
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject).
					WithStatusSubresource(&kargoapi.Freight{}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)
					got := &kargoapi.Freight{}
					require.NoError(t, c.Get(
						t.Context(), client.ObjectKey{Namespace: project, Name: freightName}, got,
					))
					require.Equal(t, kargoapi.FreightOriginKindWarehouse, got.Origin.Kind)
				},
			},
			{
				name: "creates Freight with metadata",
				body: mustJSONBody(createFreightRequest{
					Freight: newFreight,
					Metadata: map[string]apiextensionsv1.JSON{
						metadataKey: {Raw: []byte(`"recorded"`)},
					},
				}),
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject).
					WithStatusSubresource(&kargoapi.Freight{}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)
					got := &kargoapi.Freight{}
					require.NoError(t, c.Get(
						t.Context(), client.ObjectKey{Namespace: project, Name: freightName}, got,
					))
					require.Contains(t, got.Status.Metadata, metadataKey)
					require.JSONEq(t, `"recorded"`, string(got.Status.Metadata[metadataKey].Raw))
				},
			},
			{
				name: "is idempotent and preserves existing metadata",
				body: mustJSONBody(createFreightRequest{
					Freight: newFreight,
					Metadata: map[string]apiextensionsv1.JSON{
						metadataKey: {Raw: []byte(`"overwrite-attempt"`)},
					},
				}),
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, existingFreight).
					WithStatusSubresource(&kargoapi.Freight{}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					got := &kargoapi.Freight{}
					require.NoError(t, c.Get(
						t.Context(), client.ObjectKey{Namespace: project, Name: freightName}, got,
					))
					// Fill-if-missing: the existing value is preserved.
					require.JSONEq(t, `"original"`, string(got.Status.Metadata[metadataKey].Raw))
				},
			},
		},
	)
}
