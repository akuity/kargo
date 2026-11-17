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
	kargoEvent "github.com/akuity/kargo/pkg/event"
	eventfake "github.com/akuity/kargo/pkg/event/fake"
	"github.com/akuity/kargo/pkg/server/config"
)

func Test_server_markFreightInStage(t *testing.T) {
	const stageName = "fake-stage"
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: testProject.Name, Name: "fake-freight"},
	}
	// Freight already verified in the Stage (but not current), so a re-mark is not
	// "newly verified" -- but it is "newly current".
	testFreightAlreadyVerified := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: testProject.Name, Name: "fake-freight"},
		Status: kargoapi.FreightStatus{
			VerifiedIn: map[string]kargoapi.VerifiedStage{stageName: {}},
		},
	}
	// Freight already verified in AND current in the Stage, so a re-mark is neither
	// newly verified nor newly current.
	testFreightAlreadyVerifiedAndCurrent := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: testProject.Name, Name: "fake-freight"},
		Status: kargoapi.FreightStatus{
			CurrentlyIn: map[string]kargoapi.CurrentStage{stageName: {}},
			VerifiedIn:  map[string]kargoapi.VerifiedStage{stageName: {}},
		},
	}
	url := "/v1beta1/projects/" + testProject.Name +
		"/freight/" + testFreight.Name + "/status/stage/" + stageName

	// Senders captured per case so assertions can inspect what was emitted.
	verifiedSender := &eventfake.Sender{}
	unverifiedSender := &eventfake.Sender{}
	reverifiedSender := &eventfake.Sender{}
	newlyCurrentSender := &eventfake.Sender{}

	sentTypes := func(s *eventfake.Sender) []string {
		var types []string
		for _, e := range s.Sent() {
			types = append(types, e.Type())
		}
		return types
	}
	currentlyInVerified := func(t *testing.T, s *eventfake.Sender) bool {
		for _, e := range s.Sent() {
			if e.Type() == string(kargoapi.EventTypeFreightCurrentlyInStage) {
				var d kargoEvent.FreightCurrentlyInStage
				require.NoError(t, e.DataAs(&d))
				return d.VerifiedInStage
			}
		}
		t.Fatal("no FreightCurrentlyInStage event was sent")
		return false
	}

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodPost, url,
		[]restTestCase{
			{
				name:          "Project not found",
				body:          mustJSONBody(markFreightInStageRequest{}),
				clientBuilder: fake.NewClientBuilder(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "Freight not found",
				body:          mustJSONBody(markFreightInStageRequest{}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "marks Freight currently in Stage",
				body: mustJSONBody(markFreightInStageRequest{}),
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, testFreight).
					WithStatusSubresource(testFreight),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					freight := &kargoapi.Freight{}
					require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(testFreight), freight))
					require.True(t, freight.IsCurrentlyIn(stageName))
					require.False(t, freight.IsVerifiedIn(stageName))
				},
			},
			{
				name: "marks Freight verified in Stage and emits events",
				body: mustJSONBody(markFreightInStageRequest{Verified: true}),
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, testFreight).
					WithStatusSubresource(testFreight),
				serverSetup: func(_ *testing.T, s *server) {
					s.sender = verifiedSender
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					freight := &kargoapi.Freight{}
					require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(testFreight), freight))
					require.True(t, freight.IsCurrentlyIn(stageName))
					require.True(t, freight.IsVerifiedIn(stageName))

					// Newly verified AND newly current, so both events are emitted, and
					// the currentlyIn event reports the Freight as verified.
					require.ElementsMatch(t, []string{
						string(kargoapi.EventTypeFreightVerifiedInStage),
						string(kargoapi.EventTypeFreightCurrentlyInStage),
					}, sentTypes(verifiedSender))
					require.True(t, currentlyInVerified(t, verifiedSender))
				},
			},
			{
				name: "emits an unverified currentlyIn event when not verified",
				body: mustJSONBody(markFreightInStageRequest{}),
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, testFreight).
					WithStatusSubresource(testFreight),
				serverSetup: func(_ *testing.T, s *server) {
					s.sender = unverifiedSender
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					// Newly current but not verified: a currentlyIn event fires on the
					// transition, flagged unverified; no verifiedIn event.
					require.Equal(t,
						[]string{string(kargoapi.EventTypeFreightCurrentlyInStage)},
						sentTypes(unverifiedSender),
					)
					require.False(t, currentlyInVerified(t, unverifiedSender))
				},
			},
			{
				name: "emits only a currentlyIn event when already verified but newly current",
				body: mustJSONBody(markFreightInStageRequest{Verified: true}),
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, testFreightAlreadyVerified).
					WithStatusSubresource(testFreightAlreadyVerified),
				serverSetup: func(_ *testing.T, s *server) {
					s.sender = newlyCurrentSender
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					require.Equal(t,
						[]string{string(kargoapi.EventTypeFreightCurrentlyInStage)},
						sentTypes(newlyCurrentSender),
					)
					require.True(t, currentlyInVerified(t, newlyCurrentSender))
				},
			},
			{
				name: "does not emit events when already verified and current",
				body: mustJSONBody(markFreightInStageRequest{Verified: true}),
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, testFreightAlreadyVerifiedAndCurrent).
					WithStatusSubresource(testFreightAlreadyVerifiedAndCurrent),
				serverSetup: func(_ *testing.T, s *server) {
					s.sender = reverifiedSender
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					require.Empty(t, reverifiedSender.Sent())
				},
			},
		},
	)
}

func Test_server_unmarkFreightInStage(t *testing.T) {
	const stageName = "fake-stage"
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	// Freight that currently records it is in use by the Stage.
	testFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: testProject.Name, Name: "fake-freight"},
		Status: kargoapi.FreightStatus{
			CurrentlyIn: map[string]kargoapi.CurrentStage{
				stageName: {Since: &metav1.Time{Time: metav1.Now().Time}},
			},
		},
	}
	url := "/v1beta1/projects/" + testProject.Name +
		"/freight/" + testFreight.Name + "/status/stage/" + stageName

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodPatch, url,
		[]restTestCase{
			{
				name:          "Freight not found",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "removes Stage from Freight currentlyIn",
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, testFreight).
					WithStatusSubresource(testFreight),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					freight := &kargoapi.Freight{}
					require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(testFreight), freight))
					require.False(t, freight.IsCurrentlyIn(stageName))
				},
			},
		},
	)
}

func Test_server_deleteFreightStageStatus(t *testing.T) {
	const stageName = "fake-stage"
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	// Freight that records the Stage in every status map.
	testFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: testProject.Name, Name: "fake-freight"},
		Status: kargoapi.FreightStatus{
			CurrentlyIn: map[string]kargoapi.CurrentStage{
				stageName: {Since: &metav1.Time{Time: metav1.Now().Time}},
			},
			VerifiedIn:  map[string]kargoapi.VerifiedStage{stageName: {}},
			ApprovedFor: map[string]kargoapi.ApprovedStage{stageName: {}},
		},
	}
	url := "/v1beta1/projects/" + testProject.Name +
		"/freight/" + testFreight.Name + "/status/stage/" + stageName

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodDelete, url,
		[]restTestCase{
			{
				name:          "Freight not found",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "removes Stage from every Freight status map",
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, testFreight).
					WithStatusSubresource(testFreight),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					freight := &kargoapi.Freight{}
					require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(testFreight), freight))
					require.False(t, freight.IsCurrentlyIn(stageName))
					require.False(t, freight.IsVerifiedIn(stageName))
					require.False(t, freight.IsApprovedFor(stageName))
				},
			},
		},
	)
}
