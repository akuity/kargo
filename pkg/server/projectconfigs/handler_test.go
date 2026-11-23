package projectconfigs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/database/projectconfigstore"
	natsrec "github.com/akuity/kargo/pkg/reconciler/nats"
	"github.com/akuity/kargo/pkg/server/auth/can"
	"github.com/akuity/kargo/pkg/server/middleware"
)

const (
	testProject    = "fake-project"
	testProjectUID = "fake-project-uid"
)

// fakeStore is an in-memory projectconfigstore.Store with the real store's
// error semantics for the operations the handlers use. The rest are left to
// the embedded interface, which is nil, so calling them fails the test.
type fakeStore struct {
	projectconfigstore.Store

	mu    sync.Mutex
	clock int64
	// configs holds the ProjectConfig of each Project, by Project name.
	configs map[string]projectconfigstore.Config
	// raceOnCreate makes Create store a ProjectConfig of its own and report
	// it as already existing, as if another request had created it first.
	raceOnCreate bool
	// err, when set, fails every call.
	err error
	// calls counts store reads and writes, so tests can prove that a refused
	// request never touched the store.
	calls int
	// written is the last ProjectConfig passed to Create or Update.
	written *kargoapi.ProjectConfig
	// status is the last status passed to UpdateStatus.
	status *projectconfigstore.Status
	// skipped is what UpdateStatus reports as skipped.
	skipped []string
}

var _ projectconfigstore.Store = (*fakeStore)(nil)

func (s *fakeStore) tick() time.Time {
	s.clock++
	return time.UnixMicro(s.clock)
}

// add stores a ProjectConfig for the test Project directly, bypassing the
// store's checks, and returns it.
func (s *fakeStore) add(spec kargoapi.ProjectConfigSpec) projectconfigstore.Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.insert(&kargoapi.ProjectConfig{
		ObjectMeta: metav1.ObjectMeta{Name: testProject},
		Spec:       spec,
	}, testProjectUID)
}

func (s *fakeStore) insert(cfg *kargoapi.ProjectConfig, projectUID string) projectconfigstore.Config {
	if s.configs == nil {
		s.configs = map[string]projectconfigstore.Config{}
	}
	now := s.tick()
	stored := projectconfigstore.Config{ProjectConfigRow: database.ProjectConfigRow{
		ID:          uuid.New(),
		ProjectName: cfg.Name,
		ProjectUID:  projectUID,
		Labels:      maps.Clone(cfg.Labels),
		Annotations: maps.Clone(cfg.Annotations),
		Spec:        specJSON(cfg.Spec),
		Generation:  1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}}
	s.configs[cfg.Name] = stored
	return stored
}

// specJSON encodes a spec as the real store does.
// leftoverStore holds a ProjectConfig created for an earlier Project of the
// test Project's name, which the store treats as absent.
func leftoverStore() *fakeStore {
	store := &fakeStore{}
	store.insert(
		&kargoapi.ProjectConfig{ObjectMeta: metav1.ObjectMeta{Name: testProject}},
		"deleted-project-uid",
	)
	return store
}

func specJSON(spec kargoapi.ProjectConfigSpec) []byte {
	data, err := json.Marshal(spec)
	if err != nil {
		panic(err)
	}
	return data
}

// get reads a Project's ProjectConfig as the real store does: one created
// for another Project UID is not found.
func (s *fakeStore) get(project, projectUID string) (projectconfigstore.Config, error) {
	cfg, ok := s.configs[project]
	if !ok || cfg.ProjectUID != projectUID {
		return projectconfigstore.Config{}, fmt.Errorf("ProjectConfig %q: %w", project, database.ErrNotFound)
	}
	return cfg, nil
}

func (s *fakeStore) Get(_ context.Context, project, projectUID string) (projectconfigstore.Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return projectconfigstore.Config{}, s.err
	}
	return s.get(project, projectUID)
}

func (s *fakeStore) Create(
	_ context.Context,
	cfg *kargoapi.ProjectConfig,
	projectUID string,
) (projectconfigstore.Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	s.written = cfg.DeepCopy()
	if s.err != nil {
		return projectconfigstore.Config{}, s.err
	}
	if s.raceOnCreate {
		s.insert(cfg, projectUID)
	}
	if _, err := s.get(cfg.Name, projectUID); err == nil {
		return projectconfigstore.Config{}, fmt.Errorf("ProjectConfig %q: %w", cfg.Name, database.ErrAlreadyExists)
	}
	return s.insert(cfg, projectUID), nil
}

func (s *fakeStore) Update(
	_ context.Context,
	cfg *kargoapi.ProjectConfig,
	projectUID string,
) (projectconfigstore.Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	s.written = cfg.DeepCopy()
	if s.err != nil {
		return projectconfigstore.Config{}, s.err
	}
	current, err := s.get(cfg.Name, projectUID)
	if err != nil {
		return projectconfigstore.Config{}, err
	}
	if cfg.UID != "" && string(cfg.UID) != current.ID.String() {
		return projectconfigstore.Config{}, fmt.Errorf("ProjectConfig %q: %w", cfg.Name, database.ErrConflict)
	}
	if cfg.ResourceVersion != "" &&
		cfg.ResourceVersion != strconv.FormatInt(current.UpdatedAt.UnixMicro(), 10) {
		return projectconfigstore.Config{}, fmt.Errorf("ProjectConfig %q: %w", cfg.Name, database.ErrConflict)
	}
	current.Labels = maps.Clone(cfg.Labels)
	current.Annotations = maps.Clone(cfg.Annotations)
	current.Spec = specJSON(cfg.Spec)
	current.Generation++
	current.UpdatedAt = s.tick()
	s.configs[cfg.Name] = current
	return current, nil
}

func (s *fakeStore) Delete(_ context.Context, project, projectUID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return s.err
	}
	if _, err := s.get(project, projectUID); err != nil {
		return err
	}
	delete(s.configs, project)
	return nil
}

func (s *fakeStore) UpdateStatus(
	_ context.Context,
	project string,
	status projectconfigstore.Status,
) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	s.status = &status
	if s.err != nil {
		return nil, s.err
	}
	current, ok := s.configs[project]
	if !ok {
		return nil, fmt.Errorf("ProjectConfig %q: %w", project, database.ErrNotFound)
	}
	if current.ID != status.ObservedID || current.Generation != status.ObservedGeneration {
		return nil, fmt.Errorf("ProjectConfig %q: %w", project, database.ErrConflict)
	}
	return s.skipped, nil
}

// GetByWebhookReceiverPath returns the ProjectConfig whose receivers include
// the path, as the real store does, or database.ErrNotFound.
func (s *fakeStore) GetByWebhookReceiverPath(_ context.Context, path string) (projectconfigstore.Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return projectconfigstore.Config{}, s.err
	}
	for _, cfg := range s.configs {
		for _, r := range cfg.Receivers {
			if r.Path == path {
				return cfg, nil
			}
		}
	}
	return projectconfigstore.Config{}, fmt.Errorf("webhook receiver path %q: %w", path, database.ErrNotFound)
}

func (s *fakeStore) RequestRefresh(_ context.Context, project, projectUID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return "", s.err
	}
	current, err := s.get(project, projectUID)
	if err != nil {
		return "", err
	}
	token := strconv.FormatInt(s.tick().UnixMicro(), 10)
	current.RefreshToken = &token
	s.configs[project] = current
	return token, nil
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

// forbid refuses one verb and grants the rest.
func forbid(verb can.Verb) func(context.Context, can.Access) error {
	return func(ctx context.Context, access can.Access) error {
		if access.Verb == string(verb) {
			return forbidEverything(ctx, access)
		}
		return nil
	}
}

// testCase drives one request through a Handler mounted the way the API
// server mounts it, under /v1beta1/projects/:project.
type testCase struct {
	name      string
	store     *fakeStore
	authorize func(context.Context, can.Access) error
	// conn is the NATS connection watches subscribe through. Only tests
	// that watch need one.
	conn       natsrec.Subscriber
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
// /v1beta1/projects/:project, behind the same error handling and the same
// Project lookup. The test Project exists. Watches subscribe through conn.
func newRouter(
	t *testing.T,
	store projectconfigstore.Store,
	authorize func(context.Context, can.Access) error,
	conn natsrec.Subscriber,
) *gin.Engine {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	kube := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{
			Name: testProject,
			UID:  testProjectUID,
		}},
	).Build()
	router := gin.New()
	router.Use(middleware.HandleErrors(), middleware.Recover())
	project := router.Group("/v1beta1/projects/:project")
	project.Use(middleware.RequireProject(kube))
	New(store, authorize, conn).Register(project)
	return router
}

func runTestCases(t *testing.T, cases []testCase) {
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := tc.store
			if store == nil {
				store = &fakeStore{}
			}
			authorize := tc.authorize
			if authorize == nil {
				authorize = recordAuthorizations(&[]can.Access{})
			}
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.url, tc.body)
			if tc.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			newRouter(t, store, authorize, tc.conn).ServeHTTP(w, req)
			tc.assertions(t, w)
		})
	}
}

func configURL(path string) string {
	return "/v1beta1/projects/" + testProject + "/config" + path
}

func TestRequire(t *testing.T) {
	t.Parallel()
	t.Run("no authorizer fails closed", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		newRouter(t, &fakeStore{}, nil, nil).ServeHTTP(
			w, httptest.NewRequest(http.MethodGet, configURL(""), nil),
		)
		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("a missing Project is not found", func(t *testing.T) {
		t.Parallel()
		store := &fakeStore{}
		w := httptest.NewRecorder()
		newRouter(t, store, recordAuthorizations(&[]can.Access{}), nil).ServeHTTP(
			w, httptest.NewRequest(http.MethodGet, "/v1beta1/projects/other/config", nil),
		)
		require.Equal(t, http.StatusNotFound, w.Code)
		require.Zero(t, store.calls)
	})
}
