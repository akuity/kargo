package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
)

func Test_server_patchFreightAliasHandler(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	const (
		testOldAlias = "old-alias"
		testNewAlias = "new-alias"
	)
	testFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-freight",
			Namespace: testProject.Name,
			Labels: map[string]string{
				kargoapi.LabelKeyAlias: testOldAlias,
			},
		},
		Alias: testOldAlias,
	}
	testBaseURL := "/v1beta1/projects/" + testProject.Name + "/freight/"
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodPatch, testBaseURL+testFreight.Name+"/alias?newAlias="+testNewAlias,
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "Freight not found by name",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "new alias already in use by another piece of freight",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testFreight,
					&kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "other-freight",
							Namespace: testProject.Name,
							Labels:    map[string]string{kargoapi.LabelKeyAlias: testNewAlias},
						},
						Alias: testNewAlias,
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "updates alias by name",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testFreight,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Verify the Freight was updated in the cluster
					freight := &kargoapi.Freight{}
					err := c.Get(
						t.Context(),
						client.ObjectKey{Namespace: testProject.Name, Name: testFreight.Name},
						freight,
					)
					require.NoError(t, err)
					require.Equal(t, testNewAlias, freight.Labels[kargoapi.LabelKeyAlias])
					require.Equal(t, testNewAlias, freight.Alias)
				},
			},
			{
				name: "updates alias by old alias",
				url:  testBaseURL + testOldAlias + "/alias?newAlias=" + testNewAlias,
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testFreight,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Verify the Freight was updated
					freight := &kargoapi.Freight{}
					err := c.Get(
						t.Context(),
						client.ObjectKey{Namespace: testProject.Name, Name: testFreight.Name},
						freight,
					)
					require.NoError(t, err)
					require.Equal(t, testNewAlias, freight.Labels[kargoapi.LabelKeyAlias])
					require.Equal(t, testNewAlias, freight.Alias)
				},
			},
			{
				name: "newAlias query parameter is required",
				url:  testBaseURL + testFreight.Name + "/alias",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testFreight,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
		},
	)
}
