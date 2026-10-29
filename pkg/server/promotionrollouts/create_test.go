package promotionrollouts

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/server/auth/can"
)

func TestCreate(t *testing.T) {
	t.Parallel()

	// body encodes a request to promote the Freight, as the API takes it, to
	// the named Targets or, with none, to every one the Stage governs.
	body := func(freight string, targets ...string) *bytes.Buffer {
		raw, err := json.Marshal(createPromotionRolloutRequest{Freight: freight, Targets: targets})
		require.NoError(t, err)
		return bytes.NewBuffer(raw)
	}

	freight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject,
			Name:      "fake-freight",
		},
		Origin: kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "fake-warehouse"},
	}
	// The Stage accepts Freight directly from the Warehouse, so the Freight
	// above is available to it, and governs Targets in region us: two of the
	// Project's three. It tolerates half of them failing.
	targetAware := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{Namespace: testProject, Name: "prod"},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{{
				Origin:  freight.Origin,
				Sources: kargoapi.FreightSources{Direct: true},
			}},
			Targets: &kargoapi.StageTargets{
				Selectors: []metav1.LabelSelector{{MatchLabels: map[string]string{"region": "us"}}},
				UpdateStrategy: kargoapi.TargetUpdateStrategy{
					MaxConcurrent: new(intstr.FromInt32(2)),
					MaxFailures:   new(intstr.FromString("50%")),
				},
			},
		},
	}
	classic := targetAware.DeepCopy()
	classic.Name = "classic"
	classic.Spec.Targets = nil
	unrelated := targetAware.DeepCopy()
	unrelated.Name = "unrelated"
	unrelated.Spec.RequestedFreight = []kargoapi.FreightRequest{{
		Origin:  kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "other"},
		Sources: kargoapi.FreightSources{Direct: true},
	}}

	// A selector no Target can be matched against.
	broken := targetAware.DeepCopy()
	broken.Name = "broken"
	broken.Spec.Targets.Selectors = []metav1.LabelSelector{{
		MatchExpressions: []metav1.LabelSelectorRequirement{{Key: "region", Operator: "Bogus"}},
	}}

	objects := []client.Object{targetAware, classic, unrelated, broken, freight}
	usEast := targetRow("us-east", map[string]string{"region": "us"})
	usWest := targetRow("us-west", map[string]string{"region": "us"})
	fleet := fakeTargets{rows: []database.TargetRow{
		usWest,
		targetRow("eu-west", map[string]string{"region": "eu"}),
		usEast,
	}}

	// A third Target in region us, for a case that promotes to one of three.
	wideFleet := fakeTargets{rows: append(
		slices.Clone(fleet.rows), targetRow("us-central", map[string]string{"region": "us"}),
	)}

	forbiddenStore := &fakeStore{}
	disabledStore := &fakeStore{targets: fleet.rows}
	var disabledAuthorizations []can.Access
	var authorizations []can.Access
	recordingStore := &fakeStore{}
	happyStore := &fakeStore{targets: fleet.rows}

	runTestCases(t, []testCase{
		{
			name:   "database is not configured",
			method: http.MethodPost,
			url:    stageURL("prod"),
			body:   body("fake-freight"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotImplemented, w.Code)
			},
		},
		{
			name:           "creating is not supported by this server",
			store:          disabledStore,
			targets:        fleet,
			authorize:      recordAuthorizations(&disabledAuthorizations),
			createDisabled: true,
			objects:        objects,
			method:         http.MethodPost,
			url:            stageURL("prod"),
			body:           body("fake-freight"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotImplemented, w.Code)
				require.Contains(t, w.Body.String(), "not supported by this server")
				// Refused before anything is read or authorized.
				require.Zero(t, disabledStore.calls)
				require.Empty(t, disabledAuthorizations)
			},
		},
		{
			name:   "malformed body",
			store:  &fakeStore{},
			method: http.MethodPost,
			url:    stageURL("prod"),
			body:   strings.NewReader("{"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			name:   "Freight is required",
			store:  &fakeStore{},
			method: http.MethodPost,
			url:    stageURL("prod"),
			body:   body(""),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnprocessableEntity, w.Code)
				require.Contains(t, w.Body.String(), "freight")
			},
		},
		{
			name:    "Stage not found",
			store:   &fakeStore{},
			objects: objects,
			method:  http.MethodPost,
			url:     stageURL("missing"),
			body:    body("fake-freight"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		{
			name:      "not authorized to promote",
			store:     forbiddenStore,
			authorize: forbidEverything,
			objects:   objects,
			method:    http.MethodPost,
			url:       stageURL("prod"),
			body:      body("fake-freight"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
				require.Zero(t, forbiddenStore.calls)
			},
		},
		{
			name:      "the promote verb on the Stage is what is checked",
			store:     recordingStore,
			authorize: recordAuthorizations(&authorizations),
			objects:   objects,
			method:    http.MethodPost,
			url:       stageURL("prod"),
			body:      body("fake-freight"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, w.Code)
				require.Equal(t, []can.Access{can.Promote().Stage(testProject, "prod")}, authorizations)
			},
		},
		{
			name:    "Stage does not select Targets",
			store:   &fakeStore{},
			objects: objects,
			method:  http.MethodPost,
			url:     stageURL("classic"),
			body:    body("fake-freight"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
				require.Contains(t, w.Body.String(), "does not select Targets")
			},
		},
		{
			name:    "Freight not found by name",
			store:   &fakeStore{},
			objects: objects,
			method:  http.MethodPost,
			url:     stageURL("prod"),
			body:    body("missing"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		{
			name:    "Freight not available to the Stage",
			store:   &fakeStore{},
			objects: objects,
			method:  http.MethodPost,
			url:     stageURL("unrelated"),
			body:    body("fake-freight"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
				require.Contains(t, w.Body.String(), "not available")
			},
		},
		{
			name:    "created, promoting to the Targets the Stage governs",
			store:   happyStore,
			targets: fleet,
			objects: objects,
			method:  http.MethodPost,
			url:     stageURL("prod"),
			body:    body("fake-freight"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, w.Code)
				rollout := &kargoapi.PromotionRollout{}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), rollout))
				require.True(t, strings.HasPrefix(rollout.Name, "prod."))
				require.Equal(t, testProject, rollout.Namespace)
				require.NotEmpty(t, rollout.UID)
				require.EqualValues(t, 1, rollout.Number)
				require.Equal(t, "prod", rollout.Spec.Stage)
				require.Equal(t, "fake-freight", rollout.Spec.Freight)
				// The Stage's selectors are resolved once, as the rollout is
				// created, and the strategy against how many Targets matched.
				require.Equal(
					t,
					[]kargoapi.PromotionRolloutTarget{{Name: "us-east"}, {Name: "us-west"}},
					rollout.Spec.Targets,
				)
				require.Equal(
					t,
					kargoapi.PromotionRolloutUpdateStrategy{MaxConcurrent: 2, MaxFailures: 1},
					rollout.Spec.UpdateStrategy,
				)
				require.Equal(t, kargoapi.PromotionRolloutPhasePending, rollout.Status.Phase)
				require.Nil(t, rollout.Status.Targets)
				stored, err := happyStore.Get(t.Context(), testProject, rollout.Name)
				require.NoError(t, err)
				ids := make([]uuid.UUID, len(stored.Targets))
				for i, target := range stored.Targets {
					ids[i] = target.TargetID
				}
				require.ElementsMatch(t, []uuid.UUID{usEast.ID, usWest.ID}, ids)
			},
		},
		{
			name:    "created for some of the Targets the Stage governs",
			store:   &fakeStore{targets: wideFleet.rows},
			targets: wideFleet,
			objects: objects,
			method:  http.MethodPost,
			url:     stageURL("prod"),
			body:    body("fake-freight", "us-west"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, w.Code)
				rollout := &kargoapi.PromotionRollout{}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), rollout))
				require.Equal(t, []kargoapi.PromotionRolloutTarget{{Name: "us-west"}}, rollout.Spec.Targets)
				// The strategy is resolved against the Targets promoted to,
				// rounding up: 50% of the one named is 1, where 50% of the
				// three the Stage governs would have been 2.
				require.Equal(
					t,
					kargoapi.PromotionRolloutUpdateStrategy{MaxConcurrent: 2, MaxFailures: 1},
					rollout.Spec.UpdateStrategy,
				)
			},
		},
		{
			name:    "a Target the Stage does not govern is refused",
			store:   &fakeStore{targets: fleet.rows},
			targets: fleet,
			objects: objects,
			method:  http.MethodPost,
			url:     stageURL("prod"),
			body:    body("fake-freight", "us-east", "eu-west", "nowhere"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
				require.Contains(t, w.Body.String(), `does not govern Targets eu-west, nowhere`)
			},
		},
		{
			name:    "a Stage governing no Targets records a rollout to none",
			store:   &fakeStore{},
			objects: objects,
			method:  http.MethodPost,
			url:     stageURL("prod"),
			body:    body("fake-freight"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, w.Code)
				rollout := &kargoapi.PromotionRollout{}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), rollout))
				require.NotNil(t, rollout.Spec.Targets)
				require.Empty(t, rollout.Spec.Targets)
			},
		},
		{
			name:    "a Stage whose selectors cannot be resolved",
			store:   &fakeStore{targets: fleet.rows},
			targets: fleet,
			objects: objects,
			method:  http.MethodPost,
			url:     stageURL("broken"),
			body:    body("fake-freight"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			name:    "listing the Targets fails",
			store:   &fakeStore{},
			targets: fakeTargets{err: errors.New("database is down")},
			objects: objects,
			method:  http.MethodPost,
			url:     stageURL("prod"),
			body:    body("fake-freight"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, w.Code)
			},
		},
		{
			// The Target was listed, then deleted before the rollout was
			// recorded; the store refuses the whole create.
			name:    "a Target deleted while the rollout is created",
			store:   &fakeStore{targets: []database.TargetRow{usEast}},
			targets: fleet,
			objects: objects,
			method:  http.MethodPost,
			url:     stageURL("prod"),
			body:    body("fake-freight"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, w.Code)
			},
		},
		{
			name:    "the store fails",
			store:   &fakeStore{err: errors.New("database is down")},
			targets: fleet,
			objects: objects,
			method:  http.MethodPost,
			url:     stageURL("prod"),
			body:    body("fake-freight"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, w.Code)
			},
		},
	})
}
