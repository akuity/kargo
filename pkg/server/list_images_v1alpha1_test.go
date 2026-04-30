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

func Test_server_listImages(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/images",
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "no Stages exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					images := make(map[string]*TagMap)
					err := json.Unmarshal(w.Body.Bytes(), &images)
					require.NoError(t, err)
					require.Empty(t, images)
				},
			},
			{
				name: "lists images from Stages",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					&kargoapi.Stage{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "stage-1",
							Namespace: testProject.Name,
						},
						Status: kargoapi.StageStatus{
							FreightHistory: kargoapi.FreightHistory{
								{
									Freight: map[string]kargoapi.FreightReference{
										"fake-warehouse": {
											Origin: kargoapi.FreightOrigin{
												Kind: kargoapi.FreightOriginKindWarehouse,
												Name: "fake-warehouse",
											},
											Images: []kargoapi.Image{{
												RepoURL: "nginx",
												Tag:     "1.19.0",
											}},
										},
									},
								},
							},
						},
					},
					&kargoapi.Stage{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "stage-2",
							Namespace: testProject.Name,
						},
						Status: kargoapi.StageStatus{
							FreightHistory: kargoapi.FreightHistory{
								{
									Freight: map[string]kargoapi.FreightReference{
										"fake-warehouse": {
											Origin: kargoapi.FreightOrigin{
												Kind: kargoapi.FreightOriginKindWarehouse,
												Name: "fake-warehouse",
											},
											Images: []kargoapi.Image{{
												RepoURL: "nginx",
												Tag:     "1.20.0",
											}},
										},
									},
								},
							},
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					images := make(map[string]*TagMap)
					err := json.Unmarshal(w.Body.Bytes(), &images)
					require.NoError(t, err)
					require.Len(t, images, 1)
					tags, ok := images["nginx"]
					require.True(t, ok)
					require.Len(t, tags.Tags, 2)
					require.Contains(t, tags.Tags, "1.19.0")
					require.Contains(t, tags.Tags, "1.20.0")
				},
			},
		},
	)
}
