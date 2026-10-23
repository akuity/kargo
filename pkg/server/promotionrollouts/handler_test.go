package promotionrollouts

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/database/promotionrolloutstore"
	"github.com/akuity/kargo/pkg/server/auth/can"
	"github.com/akuity/kargo/pkg/server/middleware"
)

const testProject = "fake-project"

// fakeStore is an in-memory promotionrolloutstore.Store with the real
// store's error semantics.
type fakeStore struct {
	mu    sync.Mutex
	clock int64
	// rollouts holds every rollout by Project and then by name.
	rollouts map[string]map[string]promotionrolloutstore.Rollout
	// numbers counts rollouts per Project and Stage.
	numbers map[string]int64
	// err, when set, fails every call.
	err error
	// calls counts store reads and writes, so tests can prove that a refused
	// request never touched the store.
	calls int
	// targets are the Targets that exist, which Create checks the ids it is
	// given against as the database's foreign key does.
	targets []database.TargetRow
	// promotions are the mirrored Promotions that ListTargetStatuses reads.
	promotions []mirroredPromotion
}

// mirroredPromotion is a Promotion as the mirror records it: linked to the
// rollout that created it by the rollout's id.
type mirroredPromotion struct {
	rolloutID uuid.UUID
	name      string
	target    string
	phase     kargoapi.PromotionPhase
}

// withPromotions records mirrored Promotions in the store and returns it.
func (s *fakeStore) withPromotions(promotions ...mirroredPromotion) *fakeStore {
	s.promotions = append(s.promotions, promotions...)
	return s
}

// ListTargetStatuses answers as the query does: the newest of the rollout's
// Promotions to each Target, by name, in Target name order, with no phase
// read as Pending. A Promotion naming no Target does not count.
func (s *fakeStore) ListTargetStatuses(
	_ context.Context,
	rollout promotionrolloutstore.Rollout,
) ([]database.ListPromotionRolloutTargetStatusesRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	newest := map[string]mirroredPromotion{}
	for _, promo := range s.promotions {
		if promo.rolloutID != rollout.ID || promo.target == "" {
			continue
		}
		if current, ok := newest[promo.target]; !ok || promo.name > current.name {
			newest[promo.target] = promo
		}
	}
	rows := make([]database.ListPromotionRolloutTargetStatusesRow, 0, len(newest))
	for target, promo := range newest {
		phase := string(promo.phase)
		if phase == "" {
			phase = string(kargoapi.PromotionPhasePending)
		}
		rows = append(rows, database.ListPromotionRolloutTargetStatusesRow{
			TargetName:    target,
			PromotionName: promo.name,
			Phase:         phase,
		})
	}
	slices.SortFunc(rows, func(a, b database.ListPromotionRolloutTargetStatusesRow) int {
		return strings.Compare(a.TargetName, b.TargetName)
	})
	return rows, nil
}

var _ promotionrolloutstore.Store = (*fakeStore)(nil)

func (s *fakeStore) tick() time.Time {
	s.clock++
	return time.UnixMicro(s.clock)
}

func (s *fakeStore) project(project string) map[string]promotionrolloutstore.Rollout {
	if s.rollouts == nil {
		s.rollouts = map[string]map[string]promotionrolloutstore.Rollout{}
	}
	if s.rollouts[project] == nil {
		s.rollouts[project] = map[string]promotionrolloutstore.Rollout{}
	}
	return s.rollouts[project]
}

// add stores a rollout directly, bypassing the store's checks, promoting to
// Targets of the given names.
func (s *fakeStore) add(
	project, name, stage string,
	phase kargoapi.PromotionRolloutPhase,
	targets ...string,
) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.tick()
	if s.numbers == nil {
		s.numbers = map[string]int64{}
	}
	s.numbers[project+"/"+stage]++
	rollout := promotionrolloutstore.Rollout{
		PromotionRolloutRow: database.PromotionRolloutRow{
			ID:          rolloutID(project, name),
			ProjectName: project,
			Name:        name,
			StageName:   stage,
			Number:      s.numbers[project+"/"+stage],
			FreightName: "fake-freight",
			CreatedAt:   now,
			UpdatedAt:   now,
			StatusPhase: string(phase),
		},
		Targets: []database.ListPromotionRolloutTargetsRow{},
	}
	for _, target := range targets {
		rollout.Targets = append(rollout.Targets, database.ListPromotionRolloutTargetsRow{
			PromotionRolloutID: rollout.ID,
			TargetID:           uuid.New(),
			TargetName:         target,
		})
	}
	s.project(project)[name] = rollout
}

func (s *fakeStore) Create(
	_ context.Context,
	project string,
	row database.PromotionRolloutRow,
	targetIDs []uuid.UUID,
) (promotionrolloutstore.Rollout, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return promotionrolloutstore.Rollout{}, s.err
	}
	if _, ok := s.project(project)[row.Name]; ok {
		return promotionrolloutstore.Rollout{}, fmt.Errorf("PromotionRollout %q: %w", row.Name, database.ErrAlreadyExists)
	}
	targets := []database.ListPromotionRolloutTargetsRow{}
	for _, id := range targetIDs {
		i := slices.IndexFunc(s.targets, func(t database.TargetRow) bool { return t.ID == id })
		if i < 0 {
			return promotionrolloutstore.Rollout{}, fmt.Errorf("Target %s: %w", id, database.ErrConflict)
		}
		targets = append(targets, database.ListPromotionRolloutTargetsRow{
			PromotionRolloutID: rolloutID(project, row.Name),
			TargetID:           id,
			TargetName:         s.targets[i].Name,
		})
	}
	if s.numbers == nil {
		s.numbers = map[string]int64{}
	}
	s.numbers[project+"/"+row.StageName]++
	now := s.tick()
	rollout := promotionrolloutstore.Rollout{
		PromotionRolloutRow: database.PromotionRolloutRow{
			ID:            rolloutID(project, row.Name),
			ProjectName:   project,
			Name:          row.Name,
			StageName:     row.StageName,
			Number:        s.numbers[project+"/"+row.StageName],
			FreightName:   row.FreightName,
			MaxConcurrent: row.MaxConcurrent,
			MaxFailures:   row.MaxFailures,
			CreatedBy:     row.CreatedBy,
			CreatedAt:     now,
			UpdatedAt:     now,
			StatusPhase:   "Pending",
		},
		Targets: targets,
	}
	s.project(project)[row.Name] = rollout
	return rollout, nil
}

func (s *fakeStore) Get(_ context.Context, project, name string) (promotionrolloutstore.Rollout, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return promotionrolloutstore.Rollout{}, s.err
	}
	rollout, ok := s.project(project)[name]
	if !ok {
		return promotionrolloutstore.Rollout{}, fmt.Errorf("PromotionRollout %q: %w", name, database.ErrNotFound)
	}
	return rollout, nil
}

func (s *fakeStore) GetMany(
	_ context.Context,
	project string,
	names []string,
) ([]promotionrolloutstore.Rollout, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	var found []promotionrolloutstore.Rollout
	for _, name := range names {
		if rollout, ok := s.project(project)[name]; ok {
			found = append(found, rollout)
		}
	}
	return found, nil
}

// List serves a page as the store would: newest first, or oldest first
// after a cursor, bounded by the limit. A Stage's rollouts are ordered by
// number and a Project's by creation time, as in the database.
func (s *fakeStore) List(
	_ context.Context,
	project string,
	opts promotionrolloutstore.ListOptions,
) ([]promotionrolloutstore.Rollout, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	limit := int(opts.Limit)
	if limit <= 0 {
		limit = promotionrolloutstore.DefaultPageSize
	}
	var rollouts []promotionrolloutstore.Rollout
	for _, rollout := range s.project(project) {
		if opts.Stage != "" && rollout.StageName != opts.Stage {
			continue
		}
		if opts.Freight != "" && rollout.FreightName != opts.Freight {
			continue
		}
		switch {
		case opts.After != nil && rollout.Number <= opts.After.Number:
			continue
		case opts.Before != nil && opts.Stage != "" && rollout.Number >= opts.Before.Number:
			continue
		case opts.Before != nil && opts.Stage == "" && !rollout.CreatedAt.Before(opts.Before.CreatedAt):
			continue
		}
		rollouts = append(rollouts, rollout)
	}
	slices.SortFunc(rollouts, func(a, b promotionrolloutstore.Rollout) int {
		if opts.Stage != "" {
			return cmp.Compare(a.Number, b.Number)
		}
		return a.CreatedAt.Compare(b.CreatedAt)
	})
	if opts.After == nil {
		slices.Reverse(rollouts)
	}
	if len(rollouts) > limit {
		rollouts = rollouts[:limit]
	}
	return rollouts, nil
}

// ListLatest serves, for each Stage of the Project or the one named, its
// open rollout -- Running, else the oldest Pending -- and its newest
// finished one.
func (s *fakeStore) ListLatest(
	_ context.Context,
	project string,
	stage string,
) ([]promotionrolloutstore.Rollout, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	open := map[string]promotionrolloutstore.Rollout{}
	finished := map[string]promotionrolloutstore.Rollout{}
	for _, rollout := range s.project(project) {
		if stage != "" && rollout.StageName != stage {
			continue
		}
		phase := kargoapi.PromotionRolloutPhase(rollout.StatusPhase)
		if phase.IsTerminal() {
			if current, ok := finished[rollout.StageName]; !ok || rollout.Number > current.Number {
				finished[rollout.StageName] = rollout
			}
			continue
		}
		current, ok := open[rollout.StageName]
		running := phase == kargoapi.PromotionRolloutPhaseRunning
		currentRunning := kargoapi.PromotionRolloutPhase(current.StatusPhase) == kargoapi.PromotionRolloutPhaseRunning
		if !ok || (running && !currentRunning) || (running == currentRunning && rollout.Number < current.Number) {
			open[rollout.StageName] = rollout
		}
	}
	var rollouts []promotionrolloutstore.Rollout
	for _, byStage := range []map[string]promotionrolloutstore.Rollout{open, finished} {
		for _, rollout := range byStage {
			rollouts = append(rollouts, rollout)
		}
	}
	slices.SortFunc(rollouts, func(a, b promotionrolloutstore.Rollout) int {
		return cmp.Or(strings.Compare(a.StageName, b.StageName), cmp.Compare(a.Number, b.Number))
	})
	return rollouts, nil
}

func (s *fakeStore) ListOpen(context.Context) ([]promotionrolloutstore.Key, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	var keys []promotionrolloutstore.Key
	for project, rollouts := range s.rollouts {
		for name, rollout := range rollouts {
			if !kargoapi.PromotionRolloutPhase(rollout.StatusPhase).IsTerminal() {
				keys = append(keys, promotionrolloutstore.Key{Project: project, Name: name})
			}
		}
	}
	return keys, nil
}

// Start admits the rollout as the store would: only the oldest of its
// Stage's open rollouts, and only while none is Running.
func (s *fakeStore) Start(
	_ context.Context,
	rollout promotionrolloutstore.Rollout,
) (promotionrolloutstore.Rollout, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return promotionrolloutstore.Rollout{}, s.err
	}
	current, ok := s.project(rollout.ProjectName)[rollout.Name]
	if !ok || current.StatusPhase != string(kargoapi.PromotionRolloutPhasePending) {
		return promotionrolloutstore.Rollout{}, database.ErrConflict
	}
	for _, other := range s.project(rollout.ProjectName) {
		if other.StageName != current.StageName {
			continue
		}
		phase := kargoapi.PromotionRolloutPhase(other.StatusPhase)
		if phase == kargoapi.PromotionRolloutPhaseRunning || (!phase.IsTerminal() && other.Number < current.Number) {
			return promotionrolloutstore.Rollout{}, database.ErrConflict
		}
	}
	current.StatusPhase = string(kargoapi.PromotionRolloutPhaseRunning)
	current.UpdatedAt = s.tick()
	s.project(rollout.ProjectName)[rollout.Name] = current
	return current, nil
}

func (s *fakeStore) UpdateStatus(
	_ context.Context,
	project string,
	row database.PromotionRolloutRow,
) (database.PromotionRolloutRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return database.PromotionRolloutRow{}, s.err
	}
	current, ok := s.project(project)[row.Name]
	if !ok {
		return database.PromotionRolloutRow{}, fmt.Errorf("PromotionRollout %q: %w", row.Name, database.ErrNotFound)
	}
	current.StatusPhase = row.StatusPhase
	current.StatusReason = row.StatusReason
	current.StatusMessage = row.StatusMessage
	current.UpdatedAt = s.tick()
	s.project(project)[row.Name] = current
	return current.PromotionRolloutRow, nil
}

// recordAuthorizations returns an authorize function that records every
// access it is asked about and grants it.
func recordAuthorizations(calls *[]can.Access) func(context.Context, can.Access) error {
	return func(_ context.Context, access can.Access) error {
		*calls = append(*calls, access)
		return nil
	}
}

// forbidEverything refuses every access.
func forbidEverything(_ context.Context, access can.Access) error {
	return apierrors.NewForbidden(
		access.Resource.GroupResource(),
		access.Key.Name,
		errors.New("not authorized"),
	)
}

// testCase drives one request through a Handler mounted the way the API
// server mounts it, under /v1beta1/projects/:project.
type testCase struct {
	name string
	// store is nil for a server without a database. A *fakeStore that is nil
	// must not be passed; use the field's zero value.
	store *fakeStore
	// targets are the Project's Targets, which rollouts are created to
	// promote to.
	targets    fakeTargets
	authorize  func(context.Context, can.Access) error
	objects    []client.Object
	method     string
	url        string
	body       io.Reader
	assertions func(*testing.T, *httptest.ResponseRecorder)
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	gin.DefaultWriter = io.Discard
	gin.DefaultErrorWriter = io.Discard
	os.Exit(m.Run())
}

// newRouter mounts a Handler the way the API server mounts it, under
// /v1beta1/projects/:project, behind the same error handling: an error with
// an HTTP status is reported with it, anything else is a 500.
func newRouter(
	store promotionrolloutstore.Store,
	targets fakeTargets,
	authorize func(context.Context, can.Access) error,
	kube client.Client,
) *gin.Engine {
	router := gin.New()
	router.Use(middleware.HandleErrors(), middleware.Recover())
	New(store, targets, authorize, kube).Register(router.Group("/v1beta1/projects/:project"))
	return router
}

// fakeTargets lists a fixed set of Targets, or fails.
type fakeTargets struct {
	rows []database.TargetRow
	err  error
}

func (f fakeTargets) List(context.Context, string) ([]database.TargetRow, error) {
	return f.rows, f.err
}

// targetRow builds a Target of the Project with the given labels and an id
// derived from its name.
func targetRow(name string, labels map[string]string) database.TargetRow {
	return database.TargetRow{
		ID:            uuid.NewSHA1(uuid.NameSpaceURL, []byte("target/"+name)),
		ProjectName:   testProject,
		Name:          name,
		TargetSetName: "fleet",
		Labels:        labels,
		Params:        []byte("{}"),
	}
}

func runTestCases(t *testing.T, cases []testCase) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var store promotionrolloutstore.Store
			if tc.store != nil {
				store = tc.store
			}
			authorize := tc.authorize
			if authorize == nil {
				authorize = recordAuthorizations(&[]can.Access{})
			}
			kube := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tc.objects...).Build()
			w := httptest.NewRecorder()
			newRouter(store, tc.targets, authorize, kube).ServeHTTP(
				w, httptest.NewRequest(tc.method, tc.url, tc.body),
			)
			tc.assertions(t, w)
		})
	}
}

func projectURL(path string) string {
	return "/v1beta1/projects/" + testProject + "/promotion-rollouts" + path
}

// rolloutID is the id the fake store gives the named rollout, so that a
// rollout's children can name it before it is stored.
func rolloutID(project, name string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(project+"/"+name))
}

// child builds a mirrored Promotion created by the named rollout to promote
// to the Target, in the given phase.
func child(name, rollout, target string, phase kargoapi.PromotionPhase) mirroredPromotion {
	return mirroredPromotion{
		rolloutID: rolloutID(testProject, rollout),
		name:      name,
		target:    target,
		phase:     phase,
	}
}

// forbidPromotions refuses access to Promotions and allows everything else.
func forbidPromotions(ctx context.Context, access can.Access) error {
	if access.Resource.Resource == "promotions" {
		return forbidEverything(ctx, access)
	}
	return nil
}

func TestNew(t *testing.T) {
	t.Parallel()
	// A nil store leaves the handler without one, which RequireFeature turns
	// into 501 for every request.
	h := New(nil, nil, nil, nil)
	require.Nil(t, h.store)
	require.Nil(t, h.creator)
	h = New(&fakeStore{}, fakeTargets{}, nil, nil)
	require.NotNil(t, h.store)
	require.NotNil(t, h.creator)
}

func TestRequireAccess(t *testing.T) {
	t.Parallel()
	var calls []can.Access
	runTestCases(t, []testCase{
		{
			name:      "records the verb, resource and key",
			store:     &fakeStore{},
			authorize: recordAuthorizations(&calls),
			method:    http.MethodGet,
			url:       projectURL("/prod.01A.abc"),
			assertions: func(t *testing.T, _ *httptest.ResponseRecorder) {
				require.Equal(t, []can.Access{can.Get().PromotionRollout(testProject, "prod.01A.abc")}, calls)
			},
		},
		{
			name:      "the collection has no name",
			store:     &fakeStore{},
			authorize: recordAuthorizations(&calls),
			method:    http.MethodGet,
			url:       projectURL(""),
			assertions: func(t *testing.T, _ *httptest.ResponseRecorder) {
				require.Equal(t, can.List().PromotionRollout(testProject, ""), calls[len(calls)-1])
			},
		},
	})

	t.Run("no authorizer fails closed", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		newRouter(&fakeStore{}, fakeTargets{}, nil, nil).ServeHTTP(
			w, httptest.NewRequest(http.MethodGet, projectURL(""), nil),
		)
		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
