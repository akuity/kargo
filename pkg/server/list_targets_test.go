package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

	// Three Targets, added out of name order, plus one in another Project.
	newStore := func() *fakePromotionStore {
		store := &fakePromotionStore{}
		store.addTarget(testProject.Name, "us-west-2", map[string]string{"region": "us"})
		store.addTarget(testProject.Name, "eu-central-1", map[string]string{"region": "eu", "tier": "canary"})
		store.addTarget(testProject.Name, "us-east-1", map[string]string{"region": "us", "tier": "canary"})
		store.addTarget("other-project", "ap-south-1", map[string]string{"region": "ap"})
		return store
	}

	decode := func(t *testing.T, w *httptest.ResponseRecorder) *kargoapi.TargetList {
		require.Equal(t, http.StatusOK, w.Code)
		list := &kargoapi.TargetList{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), list))
		return list
	}
	names := func(list *kargoapi.TargetList) []string {
		out := make([]string, 0, len(list.Items))
		for _, target := range list.Items {
			out = append(out, target.Name)
		}
		return out
	}

	var authorizations []authorizationCall
	forbiddenStore := newStore()

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
					require.Zero(t, forbiddenStore.calls)
				},
			},
			{
				name:          "no Targets exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				serverSetup:   withStore(&fakePromotionStore{}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					list := decode(t, w)
					require.NotNil(t, list.Items)
					require.Empty(t, list.Items)
					require.Empty(t, list.ResourceVersion)
				},
			},
			{
				name:          "lists the Project's Targets sorted by name",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				serverSetup:   serverSetups(withStore(newStore()), recordAuthorizations(&authorizations)),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					list := decode(t, w)
					require.Equal(t, []string{"eu-central-1", "us-east-1", "us-west-2"}, names(list))
					require.Equal(t, testProject.Name, list.Items[0].Namespace)
					// The list's version is the newest item's, for seeding a watch.
					require.Equal(t, list.Items[1].ResourceVersion, list.ResourceVersion)
					require.Equal(t, []authorizationCall{{
						verb: "list",
						gvr:  kargoapi.GroupVersion.WithResource("targets"),
						key:  client.ObjectKey{Namespace: testProject.Name},
					}}, authorizations)
				},
			},
			{
				name:          "filters by label selector",
				url:           baseURL + "?labelSelector=region%3Dus",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				serverSetup:   withStore(newStore()),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, []string{"us-east-1", "us-west-2"}, names(decode(t, w)))
				},
			},
			{
				name:          "rejects a malformed label selector",
				url:           baseURL + "?labelSelector=region%3D%3D%3D",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				serverSetup:   withStore(newStore()),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name:          "Stage filter names a Stage that does not exist",
				url:           baseURL + "?stage=missing",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				serverSetup:   withStore(newStore()),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "Stage filter with a classic Stage governs no Targets",
				url:           baseURL + "?stage=classic",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, newStage("classic")),
				serverSetup:   withStore(newStore()),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Empty(t, decode(t, w).Items)
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
				),
				serverSetup: withStore(newStore()),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, []string{"eu-central-1", "us-east-1", "us-west-2"}, names(decode(t, w)))
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
				),
				serverSetup: withStore(newStore()),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, []string{"us-east-1"}, names(decode(t, w)))
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

	targetName := func(target kargoapi.Target) string { return target.Name }
	events := func(t *testing.T, w *httptest.ResponseRecorder) []string {
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
		return sseEvents(t, w.Body.String(), targetName)
	}
	// The harness gives a watch 100ms. Operations are spaced so that the 5ms
	// poll observes each one on its own.
	pause := func() { time.Sleep(15 * time.Millisecond) }

	baseURL := "/v1beta1/projects/" + projectName + "/targets?watch=true"

	storeWithUSEast := func() *fakePromotionStore {
		store := &fakePromotionStore{}
		store.addTarget(projectName, "us-east-1", map[string]string{"region": "us"})
		return store
	}
	changingStore := storeWithUSEast()
	fleetStore := &fakePromotionStore{}
	fleetStore.addTarget(projectName, "eu-central-1", map[string]string{"region": "eu"})
	seededStore := storeWithUSEast()
	seededStore.addTarget(projectName, "eu-central-1", map[string]string{"region": "eu"})
	var authorizations []authorizationCall

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
				name:          "database is not configured",
				clientBuilder: fake.NewClientBuilder().WithObjects(newProject()),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotImplemented, w.Code)
				},
			},
			{
				name:          "rejects a malformed label selector before watching",
				url:           baseURL + "&labelSelector=region%3D%3D%3D",
				clientBuilder: fake.NewClientBuilder().WithObjects(newProject()),
				serverSetup:   withStore(storeWithUSEast()),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name:          "Stage filter names a Stage that does not exist",
				url:           baseURL + "&stage=missing",
				clientBuilder: fake.NewClientBuilder().WithObjects(newProject()),
				serverSetup:   withStore(storeWithUSEast()),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "not authorized",
				clientBuilder: fake.NewClientBuilder().WithObjects(newProject()),
				serverSetup:   serverSetups(withStore(storeWithUSEast()), forbidEverything),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusForbidden, w.Code)
				},
			},
			{
				name:          "replays what exists, then streams additions, changes and removals",
				clientBuilder: fake.NewClientBuilder().WithObjects(newProject()),
				serverSetup:   serverSetups(withStore(changingStore), recordAuthorizations(&authorizations)),
				operations: func(context.Context, client.Client) {
					changingStore.addTarget(projectName, "ap-south-1", map[string]string{"region": "ap"})
					pause()
					changingStore.setTargetLabels("us-east-1", map[string]string{"region": "us", "tier": "prod"})
					pause()
					changingStore.removeTarget("us-east-1")
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(
						t,
						[]string{"ADDED us-east-1", "ADDED ap-south-1", "MODIFIED us-east-1", "DELETED us-east-1"},
						events(t, w),
					)
					require.Equal(t, []authorizationCall{{
						verb: "watch",
						gvr:  kargoapi.GroupVersion.WithResource("targets"),
						key:  client.ObjectKey{Namespace: projectName},
					}}, authorizations)
				},
			},
			{
				name:          "Stage filter streams only governed Targets, as they join and leave",
				url:           baseURL + "&stage=fleet",
				clientBuilder: fake.NewClientBuilder().WithObjects(newProject(), newFleetStage()),
				serverSetup:   withStore(fleetStore),
				operations: func(context.Context, client.Client) {
					// Governed: sent.
					fleetStore.addTarget(projectName, "us-east-1", map[string]string{"region": "us"})
					pause()
					// Not governed: dropped.
					fleetStore.addTarget(projectName, "ap-south-1", map[string]string{"region": "ap"})
					pause()
					// Leaves the fleet: gone as far as the watcher is concerned.
					fleetStore.setTargetLabels("eu-central-1", map[string]string{"region": "ap"})
					pause()
					// Joins the fleet: new as far as the watcher is concerned.
					fleetStore.setTargetLabels("ap-south-1", map[string]string{"region": "us"})
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(
						t,
						[]string{
							"ADDED eu-central-1",
							"ADDED us-east-1",
							"DELETED eu-central-1",
							"ADDED ap-south-1",
						},
						events(t, w),
					)
				},
			},
			{
				name: "seeded with a resource version, sends only what is newer",
				// us-east-1 was written first, eu-central-1 second.
				url:           baseURL + "&resourceVersion=1",
				clientBuilder: fake.NewClientBuilder().WithObjects(newProject()),
				serverSetup:   withStore(seededStore),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, []string{"MODIFIED eu-central-1"}, events(t, w))
				},
			},
		},
	)
}
