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

func Test_server_listPromotionRequests(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}

	// Names deliberately sort against creation order.
	newStore := func() *fakePromotionStore {
		store := &fakePromotionStore{}
		store.addRequest(testProject.Name, "fake-stage", "request-z", "fake-freight",
			kargoapi.PromotionRequestPhaseSucceeded, "us-east")
		store.addRequest(testProject.Name, "other-stage", "request-m", "fake-freight",
			kargoapi.PromotionRequestPhasePending)
		store.addRequest(testProject.Name, "fake-stage", "request-a", "fake-freight",
			kargoapi.PromotionRequestPhaseRunning)
		store.addRequest("other-project", "fake-stage", "request-b", "fake-freight",
			kargoapi.PromotionRequestPhaseRunning)
		return store
	}

	decode := func(t *testing.T, w *httptest.ResponseRecorder) *kargoapi.PromotionRequestList {
		require.Equal(t, http.StatusOK, w.Code)
		list := &kargoapi.PromotionRequestList{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), list))
		return list
	}
	names := func(list *kargoapi.PromotionRequestList) []string {
		out := make([]string, 0, len(list.Items))
		for _, request := range list.Items {
			out = append(out, request.Name)
		}
		return out
	}

	var authorizations []authorizationCall
	forbiddenStore := newStore()
	baseURL := "/v1beta1/projects/" + testProject.Name + "/promotion-requests"

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
				name:          "no PromotionRequests exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				serverSetup:   withStore(&fakePromotionStore{}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					list := decode(t, w)
					require.NotNil(t, list.Items)
					require.Empty(t, list.Items)
				},
			},
			{
				name:          "lists the Project's PromotionRequests in creation order",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				serverSetup:   serverSetups(withStore(newStore()), recordAuthorizations(&authorizations)),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					list := decode(t, w)
					require.Equal(t, []string{"request-z", "request-m", "request-a"}, names(list))
					first := list.Items[0]
					require.Equal(t, testProject.Name, first.Namespace)
					require.Equal(t, "fake-stage", first.Spec.Stage)
					require.Equal(t, "fake-freight", first.Spec.Freight)
					require.Equal(t, []kargoapi.PromotionRequestTarget{{Name: "us-east"}}, first.Spec.Targets)
					require.Equal(t, kargoapi.PromotionRequestPhaseSucceeded, first.Status.Phase)
					require.Equal(t, list.Items[2].ResourceVersion, list.ResourceVersion)
					require.Equal(t, []authorizationCall{{
						verb: "list",
						gvr:  kargoapi.GroupVersion.WithResource("promotionrequests"),
						key:  client.ObjectKey{Namespace: testProject.Name},
					}}, authorizations)
				},
			},
			{
				name:          "filters by Stage",
				url:           baseURL + "?stage=fake-stage",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				serverSetup:   withStore(newStore()),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, []string{"request-z", "request-a"}, names(decode(t, w)))
				},
			},
		},
	)
}

func Test_server_watchPromotionRequests(t *testing.T) {
	const projectName = "fake-project"
	newProject := func() *kargoapi.Project {
		return &kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: projectName}}
	}
	requestName := func(request kargoapi.PromotionRequest) string { return request.Name }
	events := func(t *testing.T, w *httptest.ResponseRecorder) []string {
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
		return sseEvents(t, w.Body.String(), requestName)
	}
	pause := func() { time.Sleep(15 * time.Millisecond) }

	changingStore := &fakePromotionStore{}
	changingStore.addRequest(projectName, "fake-stage", "request-1", "fake-freight",
		kargoapi.PromotionRequestPhasePending)
	filteredStore := &fakePromotionStore{}
	filteredStore.addRequest(projectName, "other-stage", "request-1", "fake-freight",
		kargoapi.PromotionRequestPhasePending)
	var authorizations []authorizationCall

	baseURL := "/v1beta1/projects/" + projectName + "/promotion-requests?watch=true"

	testRESTWatchEndpoint(
		t, &config.ServerConfig{},
		baseURL,
		[]restWatchTestCase{
			{
				name:          "database is not configured",
				clientBuilder: fake.NewClientBuilder().WithObjects(newProject()),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotImplemented, w.Code)
				},
			},
			{
				name:          "not authorized",
				clientBuilder: fake.NewClientBuilder().WithObjects(newProject()),
				serverSetup:   serverSetups(withStore(&fakePromotionStore{}), forbidEverything),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusForbidden, w.Code)
				},
			},
			{
				name:          "replays what exists, then streams additions, changes and removals",
				clientBuilder: fake.NewClientBuilder().WithObjects(newProject()),
				serverSetup:   serverSetups(withStore(changingStore), recordAuthorizations(&authorizations)),
				operations: func(context.Context, client.Client) {
					changingStore.addRequest(projectName, "fake-stage", "request-2", "fake-freight",
						kargoapi.PromotionRequestPhasePending)
					pause()
					changingStore.setRequestPhase("request-1", kargoapi.PromotionRequestPhaseErrored)
					pause()
					changingStore.removeRequest("request-1")
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(
						t,
						[]string{"ADDED request-1", "ADDED request-2", "MODIFIED request-1", "DELETED request-1"},
						events(t, w),
					)
					require.Equal(t, []authorizationCall{{
						verb: "watch",
						gvr:  kargoapi.GroupVersion.WithResource("promotionrequests"),
						key:  client.ObjectKey{Namespace: projectName},
					}}, authorizations)
				},
			},
			{
				name:          "Stage filter streams only that Stage's requests",
				url:           baseURL + "&stage=fake-stage",
				clientBuilder: fake.NewClientBuilder().WithObjects(newProject()),
				serverSetup:   withStore(filteredStore),
				operations: func(context.Context, client.Client) {
					filteredStore.addRequest(projectName, "fake-stage", "request-2", "fake-freight",
						kargoapi.PromotionRequestPhasePending)
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, []string{"ADDED request-2"}, events(t, w))
				},
			},
		},
	)
}
