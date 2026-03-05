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

func Test_server_abortVerification(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fake-project",
		},
	}
	testStage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-stage",
			Namespace: testProject.Name,
		},
		Status: kargoapi.StageStatus{
			FreightHistory: kargoapi.FreightHistory{{
				Freight: map[string]kargoapi.FreightReference{"fake-warehouse": {}},
				VerificationHistory: []kargoapi.VerificationInfo{{
					ID:    "fake-verification",
					Phase: kargoapi.VerificationPhaseRunning,
				}},
			}},
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodPost, "/v1beta1/projects/"+testProject.Name+"/stages/"+testStage.Name+"/verification/abort",
		[]restTestCase{{
			name: "success",
			clientBuilder: fake.NewClientBuilder().WithObjects(
				testProject,
				testStage,
			),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
				require.Equal(t, http.StatusOK, w.Code)
			},
		}},
	)
}
