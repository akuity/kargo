package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	kargoEvent "github.com/akuity/kargo/pkg/event"
	eventfake "github.com/akuity/kargo/pkg/event/fake"
	"github.com/akuity/kargo/pkg/server/config"
)

func Test_server_markFreightCurrentlyInStage(t *testing.T) {
	const stageName = "fake-stage"
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	whOrigin := kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "wh"}
	otherOrigin := kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "other-wh"}
	// Whole seconds, since metav1.Time serializes at second precision.
	since := metav1.NewTime(time.Now().Add(-time.Hour).Truncate(time.Second))

	// The Freight being marked current.
	newFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: testProject.Name, Name: "new-freight"},
		Origin:     whOrigin,
	}
	// Previously current Freight from the same origin, verified in the Stage, so
	// unmarking it finalizes its soak time.
	oldFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: testProject.Name, Name: "old-freight"},
		Origin:     whOrigin,
		Status: kargoapi.FreightStatus{
			CurrentlyIn: map[string]kargoapi.CurrentStage{stageName: {Since: &since}},
			VerifiedIn:  map[string]kargoapi.VerifiedStage{stageName: {}},
		},
	}
	// Current Freight from a different origin, which must be left alone.
	otherOriginFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: testProject.Name, Name: "other-origin-freight"},
		Origin:     otherOrigin,
		Status: kargoapi.FreightStatus{
			CurrentlyIn: map[string]kargoapi.CurrentStage{stageName: {Since: &since}},
		},
	}
	// The Freight being marked, already verified in the Stage.
	newFreightVerified := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: testProject.Name, Name: "new-freight"},
		Origin:     whOrigin,
		Status: kargoapi.FreightStatus{
			VerifiedIn: map[string]kargoapi.VerifiedStage{stageName: {}},
		},
	}
	// The Freight being marked, already current in the Stage.
	newFreightCurrent := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: testProject.Name, Name: "new-freight"},
		Origin:     whOrigin,
		Status: kargoapi.FreightStatus{
			CurrentlyIn: map[string]kargoapi.CurrentStage{stageName: {Since: &since}},
		},
	}
	url := "/v1beta1/projects/" + testProject.Name + "/stages/" + stageName + "/freight"
	body := func() io.Reader {
		return mustJSONBody(markFreightCurrentlyInStageRequest{
			Freight: newFreight.Name,
			Origin:  whOrigin,
		})
	}

	// Senders captured per case so assertions can inspect what was emitted.
	unverifiedSender := &eventfake.Sender{}
	verifiedSender := &eventfake.Sender{}
	alreadyCurrentSender := &eventfake.Sender{}

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
	getFreight := func(t *testing.T, c client.Client, f *kargoapi.Freight) *kargoapi.Freight {
		got := &kargoapi.Freight{}
		require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(f), got))
		return got
	}

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodPost, url,
		[]restTestCase{
			{
				name:          "Project not found",
				body:          body(),
				clientBuilder: freightWithIndexes(t),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "Freight name missing",
				body:          mustJSONBody(markFreightCurrentlyInStageRequest{Origin: whOrigin}),
				clientBuilder: freightWithIndexes(t).WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
					require.Contains(t, w.Body.String(), "freight is required")
				},
			},
			{
				name: "origin incomplete",
				body: mustJSONBody(markFreightCurrentlyInStageRequest{
					Freight: newFreight.Name,
					Origin:  kargoapi.FreightOrigin{Name: "wh"},
				}),
				clientBuilder: freightWithIndexes(t).WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
					require.Contains(t, w.Body.String(), "origin kind and name are required")
				},
			},
			{
				name:          "Freight not found",
				body:          body(),
				clientBuilder: freightWithIndexes(t).WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "origin does not match the Freight's",
				body: mustJSONBody(markFreightCurrentlyInStageRequest{
					Freight: newFreight.Name,
					Origin:  otherOrigin,
				}),
				clientBuilder: freightWithIndexes(t).
					WithObjects(testProject, newFreight, otherOriginFreight).
					WithStatusSubresource(newFreight, otherOriginFreight),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
					require.Contains(t, w.Body.String(), "has origin")
					// Nothing is written.
					require.False(t, getFreight(t, c, newFreight).IsCurrentlyIn(stageName))
					require.True(t, getFreight(t, c, otherOriginFreight).IsCurrentlyIn(stageName))
				},
			},
			{
				name: "marks Freight current and unmarks other Freight from the same origin",
				body: body(),
				clientBuilder: freightWithIndexes(t).
					WithObjects(testProject, newFreight, oldFreight, otherOriginFreight).
					WithStatusSubresource(newFreight, oldFreight, otherOriginFreight),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					got := getFreight(t, c, newFreight)
					require.True(t, got.IsCurrentlyIn(stageName))
					require.False(t, got.IsVerifiedIn(stageName))

					old := getFreight(t, c, oldFreight)
					require.False(t, old.IsCurrentlyIn(stageName))
					require.True(t, old.IsVerifiedIn(stageName))
					require.NotNil(t, old.Status.VerifiedIn[stageName].LongestCompletedSoak)

					other := getFreight(t, c, otherOriginFreight)
					require.True(t, other.IsCurrentlyIn(stageName))
				},
			},
			{
				name: "emits an unverified currentlyIn event when newly current",
				body: body(),
				clientBuilder: freightWithIndexes(t).
					WithObjects(testProject, newFreight).
					WithStatusSubresource(newFreight),
				serverSetup: func(_ *testing.T, s *server) {
					s.sender = unverifiedSender
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					require.Equal(t,
						[]string{string(kargoapi.EventTypeFreightCurrentlyInStage)},
						sentTypes(unverifiedSender),
					)
					require.False(t, currentlyInVerified(t, unverifiedSender))
				},
			},
			{
				name: "emits a verified currentlyIn event when already verified",
				body: body(),
				clientBuilder: freightWithIndexes(t).
					WithObjects(testProject, newFreightVerified).
					WithStatusSubresource(newFreightVerified),
				serverSetup: func(_ *testing.T, s *server) {
					s.sender = verifiedSender
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					require.Equal(t,
						[]string{string(kargoapi.EventTypeFreightCurrentlyInStage)},
						sentTypes(verifiedSender),
					)
					require.True(t, currentlyInVerified(t, verifiedSender))
				},
			},
			{
				name: "does not emit events when already current",
				body: body(),
				clientBuilder: freightWithIndexes(t).
					WithObjects(testProject, newFreightCurrent).
					WithStatusSubresource(newFreightCurrent),
				serverSetup: func(_ *testing.T, s *server) {
					s.sender = alreadyCurrentSender
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					require.Empty(t, alreadyCurrentSender.Sent())
					// The original "since" is kept.
					got := getFreight(t, c, newFreightCurrent)
					require.True(t, got.Status.CurrentlyIn[stageName].Since.Equal(&since))
				},
			},
		},
	)
}

func Test_server_markFreightVerifiedInStage(t *testing.T) {
	const stageName = "fake-stage"
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: testProject.Name, Name: "fake-freight"},
	}
	earlier := metav1.NewTime(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	testFreightAlreadyVerified := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: testProject.Name, Name: "fake-freight"},
		Status: kargoapi.FreightStatus{
			VerifiedIn: map[string]kargoapi.VerifiedStage{stageName: {VerifiedAt: &earlier}},
		},
	}
	verifiedAt := metav1.NewTime(time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC))
	url := "/v1beta1/projects/" + testProject.Name +
		"/freight/" + testFreight.Name + "/status/stage/" + stageName

	newlyVerifiedSender := &eventfake.Sender{}
	reverifiedSender := &eventfake.Sender{}

	getFreight := func(t *testing.T, c client.Client) *kargoapi.Freight {
		got := &kargoapi.Freight{}
		require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(testFreight), got))
		return got
	}

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodPost, url,
		[]restTestCase{
			{
				name:          "Project not found",
				body:          mustJSONBody(markFreightVerifiedInStageRequest{}),
				clientBuilder: fake.NewClientBuilder(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "Freight not found",
				body:          mustJSONBody(markFreightVerifiedInStageRequest{}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "marks Freight verified without touching currentlyIn and emits an event",
				body: mustJSONBody(markFreightVerifiedInStageRequest{VerifiedAt: &verifiedAt}),
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, testFreight).
					WithStatusSubresource(testFreight),
				serverSetup: func(_ *testing.T, s *server) {
					s.sender = newlyVerifiedSender
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					got := getFreight(t, c)
					require.True(t, got.IsVerifiedIn(stageName))
					require.False(t, got.IsCurrentlyIn(stageName))
					require.True(t, got.Status.VerifiedIn[stageName].VerifiedAt.Equal(&verifiedAt))

					require.Len(t, newlyVerifiedSender.Sent(), 1)
					require.Equal(t,
						string(kargoapi.EventTypeFreightVerifiedInStage),
						newlyVerifiedSender.Sent()[0].Type(),
					)
				},
			},
			{
				name: "keeps the original verification and emits nothing when already verified",
				body: mustJSONBody(markFreightVerifiedInStageRequest{VerifiedAt: &verifiedAt}),
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, testFreightAlreadyVerified).
					WithStatusSubresource(testFreightAlreadyVerified),
				serverSetup: func(_ *testing.T, s *server) {
					s.sender = reverifiedSender
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					require.True(t, getFreight(t, c).Status.VerifiedIn[stageName].VerifiedAt.Equal(&earlier))
					require.Empty(t, reverifiedSender.Sent())
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
