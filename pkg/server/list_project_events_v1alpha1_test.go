package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/indexer"
	"github.com/akuity/kargo/pkg/server/config"
)

func Test_server_listProjectEvents(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/events",
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "no Events exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject).WithIndex(
					&corev1.Event{},
					indexer.EventsByInvolvedObjectAPIGroupField,
					indexer.EventsByInvolvedObjectAPIGroup,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					list := &corev1.EventList{}
					err := json.Unmarshal(w.Body.Bytes(), list)
					require.NoError(t, err)
					require.Empty(t, list.Items)
				},
			},
			{
				name: "lists Events",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					&corev1.Event{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      "event-1",
						},
						InvolvedObject: corev1.ObjectReference{
							APIVersion: kargoapi.GroupVersion.Group + "/" + kargoapi.GroupVersion.Version,
							Kind:       "Warehouse",
							Name:       "fake-warehouse",
						},
					},
					&corev1.Event{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      "event-2",
						},
						InvolvedObject: corev1.ObjectReference{
							APIVersion: kargoapi.GroupVersion.Group + "/" + kargoapi.GroupVersion.Version,
							Kind:       "Stage",
							Name:       "fake-stage",
						},
					},
				).WithIndex(
					&corev1.Event{},
					indexer.EventsByInvolvedObjectAPIGroupField,
					indexer.EventsByInvolvedObjectAPIGroup,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Events in the response
					events := &corev1.EventList{}
					err := json.Unmarshal(w.Body.Bytes(), events)
					require.NoError(t, err)
					require.Len(t, events.Items, 2)
				},
			},
		},
	)
}
