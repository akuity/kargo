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

func Test_server_getFreightLinks(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      "fake-freight",
		},
	}

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet,
		"/v1beta1/projects/"+testProject.Name+"/freight/"+testFreight.Name+"/links",
		[]restTestCase{
			{
				name: "project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "freight does not exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "no configs exist returns empty links",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testFreight,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					var resp getFreightLinksResponse
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
					require.Empty(t, resp.Links)
					require.Empty(t, resp.Errors)
				},
			},
			{
				name: "cluster config freight links are returned",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testFreight,
					&kargoapi.ClusterConfig{
						ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
						Spec: kargoapi.ClusterConfigSpec{
							FreightLinks: []kargoapi.DeepLink{{
								Title: "Cluster Link",
								URL:   "https://example.com/{{ .freight.metadata.name }}",
							}},
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					var resp getFreightLinksResponse
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
					require.Len(t, resp.Links, 1)
					require.Equal(t, "Cluster Link", resp.Links[0].Title)
					require.Equal(t, "https://example.com/fake-freight", resp.Links[0].URL)
				},
			},
			{
				name: "project config freight links are returned",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testFreight,
					&kargoapi.ProjectConfig{
						ObjectMeta: metav1.ObjectMeta{
							Name:      testProject.Name,
							Namespace: testProject.Name,
						},
						Spec: kargoapi.ProjectConfigSpec{
							FreightLinks: []kargoapi.DeepLink{{
								Title: "Project Link",
								URL:   "https://example.com/{{ .freight.metadata.name }}",
							}},
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					var resp getFreightLinksResponse
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
					require.Len(t, resp.Links, 1)
					require.Equal(t, "Project Link", resp.Links[0].Title)
				},
			},
			{
				name: "cluster and project links are merged",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testFreight,
					&kargoapi.ClusterConfig{
						ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
						Spec: kargoapi.ClusterConfigSpec{
							FreightLinks: []kargoapi.DeepLink{{
								Title: "Cluster Link",
								URL:   "https://cluster.example.com/{{ .freight.metadata.name }}",
							}},
						},
					},
					&kargoapi.ProjectConfig{
						ObjectMeta: metav1.ObjectMeta{
							Name:      testProject.Name,
							Namespace: testProject.Name,
						},
						Spec: kargoapi.ProjectConfigSpec{
							FreightLinks: []kargoapi.DeepLink{{
								Title: "Project Link",
								URL:   "https://project.example.com/{{ .freight.metadata.name }}",
							}},
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					var resp getFreightLinksResponse
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
					require.Len(t, resp.Links, 2)
					require.Equal(t, "Cluster Link", resp.Links[0].Title)
					require.Equal(t, "Project Link", resp.Links[1].Title)
				},
			},
			{
				name: "links with false conditions are omitted",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testFreight,
					&kargoapi.ClusterConfig{
						ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
						Spec: kargoapi.ClusterConfigSpec{
							FreightLinks: []kargoapi.DeepLink{{
								Title: "Conditional Link",
								URL:   "https://example.com",
								If:    `false`,
							}},
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					var resp getFreightLinksResponse
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
					require.Empty(t, resp.Links)
				},
			},
			{
				name: "template evaluation errors are reported",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testFreight,
					&kargoapi.ClusterConfig{
						ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
						Spec: kargoapi.ClusterConfigSpec{
							FreightLinks: []kargoapi.DeepLink{{
								Title: "Bad Template",
								URL:   "https://example.com/{{ .freight.metadata.name",
							}},
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					var resp getFreightLinksResponse
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
					require.Empty(t, resp.Links)
					require.Len(t, resp.Errors, 1)
				},
			},
		},
	)
}
