package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/client/storage"
	kargoEvent "github.com/akuity/kargo/pkg/event"
	eventfake "github.com/akuity/kargo/pkg/event/fake"
	"github.com/akuity/kargo/pkg/server/config"
)

func Test_server_markFreightsCurrentlyInStage(t *testing.T) {
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
	// A second Freight being marked current, from a different origin.
	secondNewFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: testProject.Name, Name: "second-new-freight"},
		Origin:     otherOrigin,
	}
	// Previously current Freight, verified in the Stage
	oldFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: testProject.Name, Name: "old-freight"},
		Origin:     whOrigin,
		Status: kargoapi.FreightStatus{
			CurrentlyIn: map[string]kargoapi.CurrentStage{stageName: {Since: &since}},
			VerifiedIn:  map[string]kargoapi.VerifiedStage{stageName: {}},
		},
	}
	// Previously current Freight from a different origin. Any Freight not
	// listed in the request is unmarked, whatever its origin.
	otherOriginFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: testProject.Name, Name: "other-origin-freight"},
		Origin:     otherOrigin,
		Status: kargoapi.FreightStatus{
			CurrentlyIn: map[string]kargoapi.CurrentStage{
				stageName:     {Since: &since},
				"other-stage": {Since: &since},
			},
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
	url := "/v1beta1/projects/" + testProject.Name +
		"/internal/stage-freight/" + stageName + "/current"
	body := func(names ...string) io.Reader {
		return mustJSONBody(markFreightsCurrentlyInStageRequest{FreightNames: names})
	}

	// Senders captured per case so assertions can inspect what was emitted.
	unverifiedSender := &eventfake.Sender{}
	verifiedSender := &eventfake.Sender{}
	alreadyCurrentSender := &eventfake.Sender{}
	multiSender := &eventfake.Sender{}

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
				body:          body(newFreight.Name),
				clientBuilder: freightWithIndexes(t),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "invalid body",
				body:          strings.NewReader("not json"),
				clientBuilder: freightWithIndexes(t).WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "skips missing Freight and marks the rest",
				body: body("missing-freight", newFreight.Name),
				clientBuilder: freightWithIndexes(t).
					WithObjects(testProject, newFreight, oldFreight).
					WithStatusSubresource(newFreight, oldFreight),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					require.True(t, getFreight(t, c, newFreight).IsCurrentlyIn(stageName))
					require.False(t, getFreight(t, c, oldFreight).IsCurrentlyIn(stageName))
				},
			},
			{
				name: "marks listed Freight current and unmarks all other Freight",
				body: body(newFreight.Name),
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

					// Unmarked from this Stage only.
					other := getFreight(t, c, otherOriginFreight)
					require.False(t, other.IsCurrentlyIn(stageName))
					require.True(t, other.IsCurrentlyIn("other-stage"))
				},
			},
			{
				name: "marks multiple Freight current",
				body: body(newFreight.Name, secondNewFreight.Name),
				clientBuilder: freightWithIndexes(t).
					WithObjects(testProject, newFreight, secondNewFreight, oldFreight).
					WithStatusSubresource(newFreight, secondNewFreight, oldFreight),
				serverSetup: func(_ *testing.T, s *server) {
					s.storageClient = storage.NewStorageClient(s.client, multiSender, nil)
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					require.True(t, getFreight(t, c, newFreight).IsCurrentlyIn(stageName))
					require.True(t, getFreight(t, c, secondNewFreight).IsCurrentlyIn(stageName))
					require.False(t, getFreight(t, c, oldFreight).IsCurrentlyIn(stageName))
					require.Equal(t,
						[]string{
							string(kargoapi.EventTypeFreightCurrentlyInStage),
							string(kargoapi.EventTypeFreightCurrentlyInStage),
						},
						sentTypes(multiSender),
					)
				},
			},
			{
				name: "empty list unmarks all Freight",
				body: body(),
				clientBuilder: freightWithIndexes(t).
					WithObjects(testProject, oldFreight, otherOriginFreight).
					WithStatusSubresource(oldFreight, otherOriginFreight),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					require.False(t, getFreight(t, c, oldFreight).IsCurrentlyIn(stageName))
					require.False(t, getFreight(t, c, otherOriginFreight).IsCurrentlyIn(stageName))
				},
			},
			{
				name: "emits an unverified currentlyIn event when newly current",
				body: body(newFreight.Name),
				clientBuilder: freightWithIndexes(t).
					WithObjects(testProject, newFreight).
					WithStatusSubresource(newFreight),
				serverSetup: func(_ *testing.T, s *server) {
					s.storageClient = storage.NewStorageClient(s.client, unverifiedSender, nil)
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
				body: body(newFreightVerified.Name),
				clientBuilder: freightWithIndexes(t).
					WithObjects(testProject, newFreightVerified).
					WithStatusSubresource(newFreightVerified),
				serverSetup: func(_ *testing.T, s *server) {
					s.storageClient = storage.NewStorageClient(s.client, verifiedSender, nil)
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
				name: "keeps already current Freight current without emitting events",
				body: body(newFreightCurrent.Name),
				clientBuilder: freightWithIndexes(t).
					WithObjects(testProject, newFreightCurrent).
					WithStatusSubresource(newFreightCurrent),
				serverSetup: func(_ *testing.T, s *server) {
					s.storageClient = storage.NewStorageClient(s.client, alreadyCurrentSender, nil)
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					require.Empty(t, alreadyCurrentSender.Sent())
					// The original "since" is kept.
					got := getFreight(t, c, newFreightCurrent)
					require.True(t, got.IsCurrentlyIn(stageName))
					require.True(t, got.Status.CurrentlyIn[stageName].Since.Equal(&since))
				},
			},
		},
	)
}

func Test_server_clearStageFreights(t *testing.T) {
	const stageName = "fake-stage"
	const otherStage = "other-stage"
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	since := metav1.NewTime(time.Now().Add(-time.Hour).Truncate(time.Second))

	newFreight := func(name string, status kargoapi.FreightStatus) *kargoapi.Freight {
		return &kargoapi.Freight{
			ObjectMeta: metav1.ObjectMeta{Namespace: testProject.Name, Name: name},
			Origin:     kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "wh"},
			Status:     status,
		}
	}
	currentOnly := newFreight("current-only", kargoapi.FreightStatus{
		CurrentlyIn: map[string]kargoapi.CurrentStage{
			stageName:  {Since: &since},
			otherStage: {Since: &since},
		},
	})
	verifiedOnly := newFreight("verified-only", kargoapi.FreightStatus{
		VerifiedIn: map[string]kargoapi.VerifiedStage{
			stageName:  {},
			otherStage: {},
		},
	})
	approvedOnly := newFreight("approved-only", kargoapi.FreightStatus{
		ApprovedFor: map[string]kargoapi.ApprovedStage{
			stageName:  {},
			otherStage: {},
		},
	})
	all := newFreight("all", kargoapi.FreightStatus{
		CurrentlyIn: map[string]kargoapi.CurrentStage{stageName: {Since: &since}},
		VerifiedIn:  map[string]kargoapi.VerifiedStage{stageName: {}},
		ApprovedFor: map[string]kargoapi.ApprovedStage{stageName: {}},
	})
	unrelated := newFreight("unrelated", kargoapi.FreightStatus{
		CurrentlyIn: map[string]kargoapi.CurrentStage{otherStage: {Since: &since}},
		VerifiedIn:  map[string]kargoapi.VerifiedStage{otherStage: {}},
		ApprovedFor: map[string]kargoapi.ApprovedStage{otherStage: {}},
	})
	url := "/v1beta1/projects/" + testProject.Name + "/internal/stage-freight/" + stageName

	getFreight := func(t *testing.T, c client.Client, f *kargoapi.Freight) *kargoapi.Freight {
		got := &kargoapi.Freight{}
		require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(f), got))
		return got
	}
	requireNoStage := func(t *testing.T, f *kargoapi.Freight) {
		require.NotContains(t, f.Status.CurrentlyIn, stageName)
		require.NotContains(t, f.Status.VerifiedIn, stageName)
		require.NotContains(t, f.Status.ApprovedFor, stageName)
	}

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodDelete, url,
		[]restTestCase{
			{
				name:          "Project not found",
				clientBuilder: freightWithIndexes(t),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "no Freight references the Stage",
				clientBuilder: freightWithIndexes(t).WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
				},
			},
			{
				name: "clears the Stage from all Freight referencing it",
				clientBuilder: freightWithIndexes(t).
					WithObjects(testProject, currentOnly, verifiedOnly, approvedOnly, all, unrelated).
					WithStatusSubresource(currentOnly, verifiedOnly, approvedOnly, all, unrelated),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					got := getFreight(t, c, currentOnly)
					requireNoStage(t, got)
					require.Contains(t, got.Status.CurrentlyIn, otherStage)

					got = getFreight(t, c, verifiedOnly)
					requireNoStage(t, got)
					require.Contains(t, got.Status.VerifiedIn, otherStage)

					got = getFreight(t, c, approvedOnly)
					requireNoStage(t, got)
					require.Contains(t, got.Status.ApprovedFor, otherStage)

					requireNoStage(t, getFreight(t, c, all))

					require.Equal(t, unrelated.Status, getFreight(t, c, unrelated).Status)
				},
			},
		},
	)
}

func Test_server_markFreightsVerifiedInStage(t *testing.T) {
	const stageName = "fake-stage"
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: testProject.Name, Name: "fake-freight"},
	}
	secondFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: testProject.Name, Name: "second-freight"},
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
		"/internal/stage-freight/" + stageName + "/verified"
	body := func(at *metav1.Time, names ...string) io.Reader {
		return mustJSONBody(markFreightsVerifiedInStageRequest{
			FreightNames: names,
			VerifiedAt:   at,
		})
	}
	// Whole seconds, since metav1.Time serializes at second precision.
	testStart := time.Now().Truncate(time.Second)

	newlyVerifiedSender := &eventfake.Sender{}
	reverifiedSender := &eventfake.Sender{}
	multiSender := &eventfake.Sender{}

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
				body:          body(nil, testFreight.Name),
				clientBuilder: fake.NewClientBuilder(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "invalid body",
				body:          strings.NewReader("not json"),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "skips missing Freight and marks the rest",
				body: body(&verifiedAt, "missing-freight", testFreight.Name),
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, testFreight).
					WithStatusSubresource(testFreight),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					require.True(t, getFreight(t, c, testFreight).IsVerifiedIn(stageName))
				},
			},
			{
				name: "marks Freight verified without touching currentlyIn and emits an event",
				body: body(&verifiedAt, testFreight.Name),
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, testFreight).
					WithStatusSubresource(testFreight),
				serverSetup: func(_ *testing.T, s *server) {
					s.storageClient = storage.NewStorageClient(s.client, newlyVerifiedSender, nil)
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					got := getFreight(t, c, testFreight)
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
				name: "marks multiple Freight verified",
				body: body(&verifiedAt, testFreight.Name, secondFreight.Name),
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, testFreight, secondFreight).
					WithStatusSubresource(testFreight, secondFreight),
				serverSetup: func(_ *testing.T, s *server) {
					s.storageClient = storage.NewStorageClient(s.client, multiSender, nil)
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					require.True(t, getFreight(t, c, testFreight).IsVerifiedIn(stageName))
					require.True(t, getFreight(t, c, secondFreight).IsVerifiedIn(stageName))
					require.Len(t, multiSender.Sent(), 2)
				},
			},
			{
				name: "defaults verifiedAt to the time the request is handled",
				body: body(nil, testFreight.Name),
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, testFreight).
					WithStatusSubresource(testFreight),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					got := getFreight(t, c, testFreight).Status.VerifiedIn[stageName].VerifiedAt
					require.NotNil(t, got)
					require.False(t, got.Time.Before(testStart))
				},
			},
			{
				name: "keeps the original verification and emits nothing when already verified",
				body: body(&verifiedAt, testFreightAlreadyVerified.Name),
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, testFreightAlreadyVerified).
					WithStatusSubresource(testFreightAlreadyVerified),
				serverSetup: func(_ *testing.T, s *server) {
					s.storageClient = storage.NewStorageClient(s.client, reverifiedSender, nil)
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					got := getFreight(t, c, testFreightAlreadyVerified)
					require.True(t, got.Status.VerifiedIn[stageName].VerifiedAt.Equal(&earlier))
					require.Empty(t, reverifiedSender.Sent())
				},
			},
		},
	)
}
