package server

import (
	"encoding/json"
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

func Test_server_getTarget(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	newStore := func() *fakePromotionStore {
		store := &fakePromotionStore{}
		store.addTarget(testProject.Name, "us-east-1", map[string]string{"region": "us"})
		return store
	}
	var authorizations []authorizationCall
	forbiddenStore := newStore()

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/targets/us-east-1",
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "database is not configured",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotImplemented, w.Code)
				},
			},
			{
				name:          "not authorized",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				serverSetup:   serverSetups(withStore(forbiddenStore), forbidEverything),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusForbidden, w.Code)
					// The store was never consulted.
					require.Zero(t, forbiddenStore.calls)
				},
			},
			{
				name:          "Target does not exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				serverSetup:   withStore(&fakePromotionStore{}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "returns the Target",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				serverSetup:   serverSetups(withStore(newStore()), recordAuthorizations(&authorizations)),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					target := &kargoapi.Target{}
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), target))
					require.Equal(t, "us-east-1", target.Name)
					require.Equal(t, testProject.Name, target.Namespace)
					require.Equal(t, map[string]string{"region": "us"}, target.Labels)
					require.NotEmpty(t, target.UID)
					require.NotEmpty(t, target.ResourceVersion)
					require.Equal(t, []authorizationCall{{
						verb: "get",
						gvr:  kargoapi.GroupVersion.WithResource("targets"),
						key:  client.ObjectKey{Namespace: testProject.Name, Name: "us-east-1"},
					}}, authorizations)
				},
			},
		},
	)
}
