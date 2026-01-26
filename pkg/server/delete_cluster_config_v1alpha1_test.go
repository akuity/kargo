package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
)

func Test_server_deleteClusterConfig(t *testing.T) {
	testConfig := &kargoapi.ClusterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: api.ClusterConfigName},
	}
	testRESTEndpoint(
		t, nil,
		http.MethodDelete, "/v1beta1/system/cluster-config",
		[]restTestCase{
			{
				name: "ClusterConfig does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "deletes ClusterConfig",
				clientBuilder: fake.NewClientBuilder().WithObjects(testConfig),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusNoContent, w.Code)

					// Verify the ClusterConfig was deleted from the cluster
					config := &kargoapi.ClusterConfig{}
					err := c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testConfig),
						config,
					)
					require.Error(t, err)
					require.True(t, apierrors.IsNotFound(err))
				},
			},
		},
	)
}
