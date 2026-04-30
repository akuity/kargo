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

func Test_server_refreshStage(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testPromotion := &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-promotion",
			Namespace: testProject.Name,
		},
	}
	testStage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-stage",
			Namespace: testProject.Name,
		},
		Status: kargoapi.StageStatus{
			CurrentPromotion: &kargoapi.PromotionReference{Name: testPromotion.Name},
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodPost, "/v1beta1/projects/"+testProject.Name+"/stages/"+testStage.Name+"/refresh",
		[]restTestCase{
			{
				name:          "Project not found",
				clientBuilder: fake.NewClientBuilder(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "Stage not found",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "refreshes Stage",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testStage,
					testPromotion,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Verify the Stage was refreshed
					stage := &kargoapi.Stage{}
					err := c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testStage),
						stage,
					)
					require.NoError(t, err)
					require.NotEmpty(t, stage.Annotations[kargoapi.AnnotationKeyRefresh])

					// Verify the current Promotion was also refreshed
					promotion := &kargoapi.Promotion{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testPromotion),
						promotion,
					)
					require.NoError(t, err)
					require.NotEmpty(t, promotion.Annotations[kargoapi.AnnotationKeyRefresh])
				},
			},
		},
	)
}
