package promotionrollouts

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// add stores a rollout directly, bypassing the store's checks, with its
// Targets already resolved to the given names.
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
			ID:                uuid.New(),
			ProjectName:       project,
			Name:              name,
			StageName:         stage,
			Number:            s.numbers[project+"/"+stage],
			FreightName:       "fake-freight",
			CreatedAt:         now,
			UpdatedAt:         now,
			TargetsResolvedAt: &now,
			StatusPhase:       string(phase),
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
	if s.numbers == nil {
		s.numbers = map[string]int64{}
	}
	s.numbers[project+"/"+row.StageName]++
	now := s.tick()
	rollout := promotionrolloutstore.Rollout{
		PromotionRolloutRow: database.PromotionRolloutRow{
			ID:            uuid.New(),
			ProjectName:   project,
			Name:          row.Name,
			StageName:     row.StageName,
			Number:        s.numbers[project+"/"+row.StageName],
			FreightName:   row.FreightName,
			MaxConcurrent: row.MaxConcurrent,
			CreatedBy:     row.CreatedBy,
			CreatedAt:     now,
			UpdatedAt:     now,
			StatusPhase:   "Pending",
		},
		Targets: []database.ListPromotionRolloutTargetsRow{},
	}
	s.project(project)[row.Name] = rollout
	return rollout, nil
}

func (s *fakeStore) ResolveTargets(
	_ context.Context,
	rollout promotionrolloutstore.Rollout,
	targetIDs []uuid.UUID,
) (promotionrolloutstore.Rollout, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return promotionrolloutstore.Rollout{}, s.err
	}
	current, ok := s.project(rollout.ProjectName)[rollout.Name]
	if !ok {
		return promotionrolloutstore.Rollout{}, fmt.Errorf("PromotionRollout %q: %w", rollout.Name, database.ErrNotFound)
	}
	if current.TargetsResolvedAt != nil {
		return promotionrolloutstore.Rollout{}, fmt.Errorf("PromotionRollout %q: %w", rollout.Name, database.ErrConflict)
	}
	now := s.tick()
	current.TargetsResolvedAt = &now
	current.UpdatedAt = now
	for _, id := range targetIDs {
		current.Targets = append(current.Targets, database.ListPromotionRolloutTargetsRow{
			PromotionRolloutID: current.ID,
			TargetID:           id,
			TargetName:         id.String(),
		})
	}
	s.project(rollout.ProjectName)[rollout.Name] = current
	return current, nil
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

func (s *fakeStore) List(_ context.Context, project, stage string) ([]promotionrolloutstore.Rollout, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	rollouts := make([]promotionrolloutstore.Rollout, 0, len(s.project(project)))
	for _, rollout := range s.project(project) {
		if stage == "" || rollout.StageName == stage {
			rollouts = append(rollouts, rollout)
		}
	}
	slices.SortFunc(rollouts, func(a, b promotionrolloutstore.Rollout) int {
		return a.CreatedAt.Compare(b.CreatedAt)
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
	store      *fakeStore
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
	authorize func(context.Context, can.Access) error,
	kube client.Client,
) *gin.Engine {
	router := gin.New()
	router.Use(middleware.HandleErrors(), middleware.Recover())
	New(store, authorize, kube).Register(router.Group("/v1beta1/projects/:project"))
	return router
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
			newRouter(store, authorize, kube).ServeHTTP(
				w, httptest.NewRequest(tc.method, tc.url, tc.body),
			)
			tc.assertions(t, w)
		})
	}
}

func projectURL(path string) string {
	return "/v1beta1/projects/" + testProject + "/promotion-rollouts" + path
}

// child builds a Promotion of the "prod" Stage created by the named rollout
// to promote to the Target, in the given phase.
func child(name, rollout, target string, phase kargoapi.PromotionPhase) *kargoapi.Promotion {
	const stage = "prod"
	return &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   testProject,
			Name:        name,
			Labels:      map[string]string{kargoapi.LabelKeyStage: stage},
			Annotations: map[string]string{kargoapi.AnnotationKeyPromotionRollout: rollout},
		},
		Spec:   kargoapi.PromotionSpec{Stage: stage, Target: target},
		Status: kargoapi.PromotionStatus{Phase: phase},
	}
}

func TestNew(t *testing.T) {
	t.Parallel()
	// A nil store leaves the handler without one, which RequireFeature turns
	// into 501 for every request.
	h := New(nil, nil, nil)
	require.Nil(t, h.store)
	require.Nil(t, h.creator)
	h = New(&fakeStore{}, nil, nil)
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
		newRouter(&fakeStore{}, nil, nil).ServeHTTP(
			w, httptest.NewRequest(http.MethodGet, projectURL(""), nil),
		)
		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
