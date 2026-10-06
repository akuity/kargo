package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func Test_server_watchTargets(t *testing.T) {
	const projectName = "fake-project"

	// The watch helper runs cases in parallel and the fake client mutates the
	// objects it is built with, so every case gets fixtures of its own.
	newProject := func() *kargoapi.Project {
		return &kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: projectName}}
	}

	newTarget := func(name string, lbls map[string]string) *kargoapi.Target {
		return &kargoapi.Target{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: projectName,
				Name:      name,
				Labels:    lbls,
			},
		}
	}

	newFleetStage := func() *kargoapi.Stage {
		return &kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: projectName,
				Name:      "fleet",
			},
			Spec: kargoapi.StageSpec{
				Targets: &kargoapi.StageTargets{
					Selectors: []metav1.LabelSelector{
						{MatchLabels: map[string]string{"region": "us"}},
						{MatchLabels: map[string]string{"region": "eu"}},
					},
				},
			},
		}
	}

	// events decodes every SSE event in the body as type -> Target names, in
	// the order received.
	type event struct {
		Type   string
		Object kargoapi.Target
	}
	events := func(t *testing.T, w *httptest.ResponseRecorder) []string {
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
		var got []string
		for _, line := range strings.Split(w.Body.String(), "\n") {
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			e := event{}
			require.NoError(t, json.Unmarshal([]byte(line[len("data: "):]), &e))
			got = append(got, e.Type+" "+e.Object.Name)
		}
		return got
	}

	baseURL := "/v1beta1/projects/" + projectName + "/targets?watch=true"

	testRESTWatchEndpoint(
		t, &config.ServerConfig{},
		baseURL,
		[]restWatchTestCase{
			{
				name: "Project does not exist",
				url:  "/v1beta1/projects/nope/targets?watch=true",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "rejects a malformed label selector before watching",
				url:           baseURL + "&labelSelector=region%3D%3D%3D",
				clientBuilder: fake.NewClientBuilder().WithObjects(newProject()),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name:          "Stage filter names a Stage that does not exist",
				url:           baseURL + "&stage=missing",
				clientBuilder: fake.NewClientBuilder().WithObjects(newProject()),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "streams every Target without a filter",
				clientBuilder: fake.NewClientBuilder().WithObjects(newProject()),
				operations: func(ctx context.Context, c client.Client) {
					_ = c.Create(ctx, newTarget("us-east-1", map[string]string{"region": "us"}))
					_ = c.Create(ctx, newTarget("ap-south-1", map[string]string{"region": "ap"}))
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(
						t,
						[]string{"ADDED us-east-1", "ADDED ap-south-1"},
						events(t, w),
					)
				},
			},
			{
				name: "Stage filter streams only governed Targets and turns a departure into a delete",
				url:  baseURL + "&stage=fleet",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					newProject(),
					newFleetStage(),
					newTarget("eu-central-1", map[string]string{"region": "eu"}),
				),
				operations: func(ctx context.Context, c client.Client) {
					// Governed: sent as-is.
					_ = c.Create(ctx, newTarget("us-east-1", map[string]string{"region": "us"}))
					// Not governed: dropped.
					_ = c.Create(ctx, newTarget("ap-south-1", map[string]string{"region": "ap"}))
					// Leaves the fleet: a modification the watcher sees as a delete.
					departing := &kargoapi.Target{}
					_ = c.Get(
						ctx,
						client.ObjectKey{Namespace: projectName, Name: "eu-central-1"},
						departing,
					)
					departing.Labels = map[string]string{"region": "ap"}
					_ = c.Update(ctx, departing)
					// Joins the fleet: a modification sent as such so the watcher adds it.
					joining := &kargoapi.Target{}
					_ = c.Get(
						ctx,
						client.ObjectKey{Namespace: projectName, Name: "ap-south-1"},
						joining,
					)
					joining.Labels = map[string]string{"region": "us"}
					_ = c.Update(ctx, joining)
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(
						t,
						[]string{
							"ADDED us-east-1",
							"DELETED eu-central-1",
							"MODIFIED ap-south-1",
						},
						events(t, w),
					)
				},
			},
		},
	)
}
