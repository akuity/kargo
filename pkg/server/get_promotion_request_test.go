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

func Test_server_getPromotionRequest(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	newStore := func() *fakePromotionStore {
		store := &fakePromotionStore{}
		store.addRequest(testProject.Name, "fake-stage", "request-1", "fake-freight",
			kargoapi.PromotionRequestPhaseRunning, "us-east", "eu-west")
		return store
	}
	var authorizations []authorizationCall
	forbiddenStore := newStore()

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/promotion-requests/request-1",
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
				name:          "PromotionRequest does not exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				serverSetup:   withStore(&fakePromotionStore{}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "returns the PromotionRequest",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				serverSetup:   serverSetups(withStore(newStore()), recordAuthorizations(&authorizations)),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					request := &kargoapi.PromotionRequest{}
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), request))
					require.Equal(t, "request-1", request.Name)
					require.Equal(t, testProject.Name, request.Namespace)
					require.Equal(t, "fake-stage", request.Spec.Stage)
					require.Equal(t, "fake-freight", request.Spec.Freight)
					require.Equal(
						t,
						[]kargoapi.PromotionRequestTarget{{Name: "us-east"}, {Name: "eu-west"}},
						request.Spec.Targets,
					)
					require.Equal(t, kargoapi.PromotionRequestPhaseRunning, request.Status.Phase)
					require.Equal(t, []authorizationCall{{
						verb: "get",
						gvr:  kargoapi.GroupVersion.WithResource("promotionrequests"),
						key:  client.ObjectKey{Namespace: testProject.Name, Name: "request-1"},
					}}, authorizations)
				},
			},
		},
	)
}

func Test_server_watchPromotionRequest(t *testing.T) {
	const projectName = "fake-project"
	newProject := func() *kargoapi.Project {
		return &kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: projectName}}
	}
	requestName := func(request kargoapi.PromotionRequest) string { return request.Name }
	pause := func() { time.Sleep(15 * time.Millisecond) }

	changingStore := &fakePromotionStore{}
	changingStore.addRequest(projectName, "fake-stage", "request-1", "fake-freight",
		kargoapi.PromotionRequestPhasePending)
	var authorizations []authorizationCall

	testRESTWatchEndpoint(
		t, &config.ServerConfig{},
		"/v1beta1/projects/"+projectName+"/promotion-requests/request-1?watch=true",
		[]restWatchTestCase{
			{
				name:          "PromotionRequest does not exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(newProject()),
				serverSetup:   withStore(&fakePromotionStore{}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "streams the request's changes until it is gone",
				clientBuilder: fake.NewClientBuilder().WithObjects(newProject()),
				serverSetup:   serverSetups(withStore(changingStore), recordAuthorizations(&authorizations)),
				operations: func(context.Context, client.Client) {
					changingStore.setRequestPhase("request-1", kargoapi.PromotionRequestPhaseRunning)
					pause()
					// Another request of the Stage is not this watch's business.
					changingStore.addRequest(projectName, "fake-stage", "request-2", "fake-freight",
						kargoapi.PromotionRequestPhasePending)
					pause()
					changingStore.removeRequest("request-1")
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					require.Equal(
						t,
						[]string{"ADDED request-1", "MODIFIED request-1", "DELETED request-1"},
						sseEvents(t, w.Body.String(), requestName),
					)
					// The request is fetched, then watched.
					require.Equal(t, []authorizationCall{
						{
							verb: "get",
							gvr:  kargoapi.GroupVersion.WithResource("promotionrequests"),
							key:  client.ObjectKey{Namespace: projectName, Name: "request-1"},
						},
						{
							verb: "watch",
							gvr:  kargoapi.GroupVersion.WithResource("promotionrequests"),
							key:  client.ObjectKey{Namespace: projectName, Name: "request-1"},
						},
					}, authorizations)
				},
			},
		},
	)
}
