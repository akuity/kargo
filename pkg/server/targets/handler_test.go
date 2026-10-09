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
	// err, when set, fails every call.
	err error
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

// add stores a Target directly, bypassing the store's checks.
func (s *fakeStore) add(project, name string, labels map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.tick()
	s.project(project)[name] = withDefaults(database.TargetRow{
		ID:        uuid.New(),
		Name:      name,
		Labels:    labels,
		CreatedAt: now,
		UpdatedAt: now,
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
	now := s.tick()
	stored := withDefaults(database.TargetRow{
		ID:        uuid.New(),
		Name:      target.Name,
		Labels:    target.Labels,
		Params:    target.Params,
		CreatedAt: now,
		UpdatedAt: now,
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
