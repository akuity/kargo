package targets

import (
	"context"
	"encoding/json"
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
	"github.com/akuity/kargo/pkg/database/targetstore"
	"github.com/akuity/kargo/pkg/server/auth/can"
	"github.com/akuity/kargo/pkg/server/middleware"
)

const testProject = "fake-project"

// fakeStore is an in-memory targetstore.Store with the real store's error
// semantics. Every write advances a clock that serves as updated_at.
type fakeStore struct {
	mu    sync.Mutex
	clock int64
	// targets holds every Target by Project and then by name.
	targets map[string]map[string]database.TargetRow
	// freight holds the rows recording Targets' current Freight, by Project.
	freight map[string][]database.ListTargetCurrentFreightRow
	// err, when set, fails every call.
	err error
	// promotions holds the Promotion history by Project, in creation order.
	promotions map[string][]database.PromotionRow
	// freightErr, when set, fails every read of Targets' current Freight.
	freightErr error
	// freightReads counts reads of Targets' current Freight.
	freightReads int
	// calls counts store reads and writes, so tests can prove that a refused
	// request never touched the store.
	calls int
}

var _ targetstore.Store = (*fakeStore)(nil)

func (s *fakeStore) tick() time.Time {
	s.clock++
	return time.UnixMicro(s.clock)
}

func (s *fakeStore) project(project string) map[string]database.TargetRow {
	if s.targets == nil {
		s.targets = map[string]map[string]database.TargetRow{}
	}
	if s.targets[project] == nil {
		s.targets[project] = map[string]database.TargetRow{}
	}
	return s.targets[project]
}

// testTargetSet is the TargetSet that manages every Target the fake store
// is seeded with.
const testTargetSet = "fleet"

// add stores a Target directly, bypassing the store's checks.
func (s *fakeStore) add(project, name string, labels map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.tick()
	s.project(project)[name] = withDefaults(database.TargetRow{
		ID:            uuid.New(),
		Name:          name,
		Labels:        labels,
		TargetSetName: testTargetSet,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
}

// withDefaults gives a row the empty labels and params the database stores
// in place of absent ones.
func withDefaults(row database.TargetRow) database.TargetRow {
	if row.Labels == nil {
		row.Labels = map[string]string{}
	}
	if len(row.Params) == 0 {
		row.Params = json.RawMessage(`{}`)
	}
	return row
}

func (s *fakeStore) Create(
	_ context.Context,
	project string,
	target database.TargetRow,
) (database.TargetRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return database.TargetRow{}, s.err
	}
	if _, ok := s.project(project)[target.Name]; ok {
		return database.TargetRow{}, fmt.Errorf("Target %q: %w", target.Name, database.ErrAlreadyExists)
	}
	if target.TargetSetName == "" {
		return database.TargetRow{}, fmt.Errorf("Target %q: %w", target.Name, database.ErrInvalid)
	}
	now := s.tick()
	stored := withDefaults(database.TargetRow{
		ID:            uuid.New(),
		Name:          target.Name,
		Labels:        target.Labels,
		Params:        target.Params,
		TargetSetName: target.TargetSetName,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	s.project(project)[target.Name] = stored
	return stored, nil
}

func (s *fakeStore) Get(_ context.Context, project, name string) (database.TargetRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return database.TargetRow{}, s.err
	}
	target, ok := s.project(project)[name]
	if !ok {
		return database.TargetRow{}, fmt.Errorf("Target %q: %w", name, database.ErrNotFound)
	}
	return target, nil
}

func (s *fakeStore) List(_ context.Context, project string) ([]database.TargetRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	targets := make([]database.TargetRow, 0, len(s.project(project)))
	for _, target := range s.project(project) {
		targets = append(targets, target)
	}
	slices.SortFunc(targets, func(a, b database.TargetRow) int { return strings.Compare(a.Name, b.Name) })
	return targets, nil
}

func (s *fakeStore) Count(
	_ context.Context,
	project string,
	names []string,
) (database.CountTargetsRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return database.CountTargetsRow{}, s.err
	}
	counts := database.CountTargetsRow{Total: int64(len(s.project(project)))}
	for name := range s.project(project) {
		if slices.Contains(names, name) {
			counts.Among++
		}
	}
	return counts, nil
}

func (s *fakeStore) Update(
	_ context.Context,
	project string,
	target database.TargetRow,
) (database.TargetRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return database.TargetRow{}, s.err
	}
	current, ok := s.project(project)[target.Name]
	if !ok {
		return database.TargetRow{}, fmt.Errorf("Target %q: %w", target.Name, database.ErrNotFound)
	}
	if target.TargetSetName != current.TargetSetName {
		return database.TargetRow{}, fmt.Errorf("Target %q: %w", target.Name, database.ErrConflict)
	}
	if target.ID != uuid.Nil && target.ID != current.ID {
		return database.TargetRow{}, fmt.Errorf("Target %q: %w", target.Name, database.ErrConflict)
	}
	if !target.UpdatedAt.IsZero() && !target.UpdatedAt.Equal(current.UpdatedAt) {
		return database.TargetRow{}, fmt.Errorf("Target %q: %w", target.Name, database.ErrConflict)
	}
	current.Labels = target.Labels
	current.Params = target.Params
	current.UpdatedAt = s.tick()
	current = withDefaults(current)
	s.project(project)[target.Name] = current
	return current, nil
}

func (s *fakeStore) Delete(_ context.Context, project, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return s.err
	}
	if _, ok := s.project(project)[name]; !ok {
		return fmt.Errorf("Target %q: %w", name, database.ErrNotFound)
	}
	delete(s.project(project), name)
	return nil
}

// addFreight records that a Stage's most recent successful Promotion to a
// test Project's Target left the named Freight there, from Warehouse w.
func (s *fakeStore) addFreight(target, stage, freight string) {
	const project = testProject
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.freight == nil {
		s.freight = map[string][]database.ListTargetCurrentFreightRow{}
	}
	s.freight[project] = append(s.freight[project], database.ListTargetCurrentFreightRow{
		TargetID:    s.project(project)[target].ID,
		StageName:   stage,
		OriginKind:  string(kargoapi.FreightOriginKindWarehouse),
		OriginName:  "w",
		FreightName: freight,
	})
}

func (s *fakeStore) CurrentFreight(
	_ context.Context,
	project string,
	name string,
) ([]database.ListTargetCurrentFreightRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	s.freightReads++
	if s.err != nil {
		return nil, s.err
	}
	if s.freightErr != nil {
		return nil, s.freightErr
	}
	var rows []database.ListTargetCurrentFreightRow
	for _, row := range s.freight[project] {
		if name == "" || s.project(project)[name].ID == row.TargetID {
			rows = append(rows, row)
		}
	}
	return rows, nil
}

// addPromotion records a Promotion from a Stage to a test Project's Target,
// created the given number of minutes after the epoch. Promotions must be
// added in creation order.
func (s *fakeStore) addPromotion(target, stage, name, phase string, minute int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.promotions == nil {
		s.promotions = map[string][]database.PromotionRow{}
	}
	s.promotions[testProject] = append(s.promotions[testProject], database.PromotionRow{
		ID:          name,
		ProjectName: testProject,
		Name:        name,
		StageName:   stage,
		FreightName: "f-" + name,
		TargetName:  &target,
		Phase:       phase,
		CreatedAt:   time.Unix(int64(minute)*60, 0).UTC(),
	})
}

func (s *fakeStore) LatestPromotions(
	_ context.Context,
	project string,
) ([]database.ListTargetLatestPromotionsRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	type key struct{ target, stage string }
	latest := map[key]database.PromotionRow{}
	for _, row := range s.promotions[project] {
		latest[key{*row.TargetName, row.StageName}] = row
	}
	var rows []database.ListTargetLatestPromotionsRow
	for k, row := range latest {
		rows = append(rows, database.ListTargetLatestPromotionsRow{
			TargetID:    s.project(project)[k.target].ID,
			Name:        row.Name,
			StageName:   row.StageName,
			FreightName: row.FreightName,
			Phase:       row.Phase,
			CreatedAt:   row.CreatedAt,
			FinishedAt:  row.FinishedAt,
		})
	}
	return rows, nil
}

func (s *fakeStore) Promotions(
	_ context.Context,
	project string,
	name string,
	limit int32,
) ([]database.PromotionRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	var rows []database.PromotionRow
	for i := len(s.promotions[project]) - 1; i >= 0 && len(rows) < int(limit); i-- {
		if row := s.promotions[project][i]; *row.TargetName == name {
			rows = append(rows, row)
		}
	}
	return rows, nil
}

// collectionID is the ID of a FreightCollection holding the named Freight.
func collectionID(names ...string) string {
	collection := &kargoapi.FreightCollection{}
	for i, name := range names {
		collection.UpdateOrPush(kargoapi.FreightReference{
			Name: name,
			Origin: kargoapi.FreightOrigin{
				Kind: kargoapi.FreightOriginKindWarehouse,
				Name: fmt.Sprintf("w%d", i),
			},
		})
	}
	return collection.ID
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
	store targetstore.Store,
	authorize func(context.Context, can.Access) error,
	kube client.Reader,
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
			var store targetstore.Store
			if tc.store != nil {
				store = tc.store
			}
			authorize := tc.authorize
			if authorize == nil {
				authorize = recordAuthorizations(&[]can.Access{})
			}
			kube := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tc.objects...).Build()
			w := httptest.NewRecorder()
			newRouter(store, authorize, kube).ServeHTTP(w, httptest.NewRequest(tc.method, tc.url, tc.body))
			tc.assertions(t, w)
		})
	}
}

func projectURL(path string) string {
	return "/v1beta1/projects/" + testProject + "/targets" + path
}

func TestNew(t *testing.T) {
	t.Parallel()
	// A nil store leaves the handler without one, which RequireFeature turns
	// into 501 for every request.
	h := New(nil, nil, nil)
	require.Nil(t, h.store)
	h = New(&fakeStore{}, nil, nil)
	require.NotNil(t, h.store)
}

func TestRequireAccess(t *testing.T) {
	t.Parallel()
	var calls []can.Access
	runTestCases(t, []testCase{
		{
			name:      "records the verb, resource and key",
			store:     &fakeStore{},
			authorize: recordAuthorizations(&calls),
			method:    http.MethodDelete,
			url:       projectURL("/us-east-1"),
			assertions: func(t *testing.T, _ *httptest.ResponseRecorder) {
				require.Equal(t, []can.Access{can.Delete().Target(testProject, "us-east-1")}, calls)
			},
		},
		{
			name:      "the collection has no name",
			store:     &fakeStore{},
			authorize: recordAuthorizations(&calls),
			method:    http.MethodGet,
			url:       projectURL(""),
			assertions: func(t *testing.T, _ *httptest.ResponseRecorder) {
				require.Equal(t, can.List().Target(testProject, ""), calls[len(calls)-1])
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
