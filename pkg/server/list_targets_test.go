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

func Test_server_listTargets(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}

	newTarget := func(name string, lbls map[string]string) *kargoapi.Target {
		return &kargoapi.Target{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject.Name,
				Name:      name,
				Labels:    lbls,
			},
		}
	}

	newStage := func(name string, selectors ...metav1.LabelSelector) *kargoapi.Stage {
		stage := &kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject.Name,
				Name:      name,
			},
		}
		if selectors != nil {
			stage.Spec.Targets = &kargoapi.StageTargets{Selectors: selectors}
		}
		return stage
	}

	decode := func(t *testing.T, w *httptest.ResponseRecorder) []string {
		require.Equal(t, http.StatusOK, w.Code)
		list := &kargoapi.TargetList{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), list))
		names := make([]string, 0, len(list.Items))
		for _, target := range list.Items {
			names = append(names, target.Name)
		}
		return names
	}

	baseURL := "/v1beta1/projects/" + testProject.Name + "/targets"

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, baseURL,
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "no Targets exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Empty(t, decode(t, w))
				},
			},
			{
				name: "lists Targets sorted by name",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					newTarget("us-west-2", map[string]string{"region": "us"}),
					newTarget("eu-central-1", map[string]string{"region": "eu"}),
					newTarget("us-east-1", map[string]string{"region": "us"}),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(
						t,
						[]string{"eu-central-1", "us-east-1", "us-west-2"},
						decode(t, w),
					)
				},
			},
			{
				name: "filters by label selector",
				url:  baseURL + "?labelSelector=region%3Dus",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					newTarget("us-west-2", map[string]string{"region": "us"}),
					newTarget("eu-central-1", map[string]string{"region": "eu"}),
					newTarget("us-east-1", map[string]string{"region": "us"}),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, []string{"us-east-1", "us-west-2"}, decode(t, w))
				},
			},
			{
				name:          "rejects a malformed label selector",
				url:           baseURL + "?labelSelector=region%3D%3D%3D",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name:          "Stage filter names a Stage that does not exist",
				url:           baseURL + "?stage=missing",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "Stage filter with a classic Stage governs no Targets",
				url:  baseURL + "?stage=classic",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					newStage("classic"),
					newTarget("us-east-1", map[string]string{"region": "us"}),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Empty(t, decode(t, w))
				},
			},
			{
				name: "Stage filter unions the Stage's selectors",
				url:  baseURL + "?stage=fleet",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					newStage(
						"fleet",
						metav1.LabelSelector{MatchLabels: map[string]string{"region": "us"}},
						metav1.LabelSelector{MatchLabels: map[string]string{"region": "eu"}},
					),
					newTarget("us-east-1", map[string]string{"region": "us"}),
					newTarget("eu-central-1", map[string]string{"region": "eu"}),
					newTarget("ap-south-1", map[string]string{"region": "ap"}),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, []string{"eu-central-1", "us-east-1"}, decode(t, w))
				},
			},
			{
				name: "Stage filter combines with a label selector",
				url:  baseURL + "?stage=fleet&labelSelector=tier%3Dcanary",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					newStage(
						"fleet",
						metav1.LabelSelector{MatchLabels: map[string]string{"region": "us"}},
					),
					newTarget("us-east-1", map[string]string{"region": "us", "tier": "canary"}),
					newTarget("us-west-2", map[string]string{"region": "us"}),
					newTarget("eu-central-1", map[string]string{"region": "eu", "tier": "canary"}),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, []string{"us-east-1"}, decode(t, w))
				},
			},
		},
	)
}
