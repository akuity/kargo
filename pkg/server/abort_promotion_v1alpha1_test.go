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

func Test_server_abortPromotion(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fake-project",
		},
	}
	testPromotion := &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-promotion",
			Namespace: testProject.Name,
		},
		Status: kargoapi.PromotionStatus{
			Phase: kargoapi.PromotionPhaseRunning,
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodPost, "/v1beta1/projects/"+testProject.Name+"/promotions/"+testPromotion.Name+"/abort",
		[]restTestCase{{
			name: "success",
			clientBuilder: fake.NewClientBuilder().WithObjects(
				testProject,
				testPromotion,
			),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
				require.Equal(t, http.StatusOK, w.Code)
			},
		}},
	)
}
