package targets

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/rest"
)

const testProject = "fake-project"

// fakeStore is an in-memory Store with the real store's error semantics.
// Every write advances a clock that serves as the resource version.
type fakeStore struct {
	mu    sync.Mutex
	clock int64
	// targets holds every Target by Project and then by name.
	targets map[string]map[string]kargoapi.Target
	// unmirrored names Projects that have not reached the database yet.
	unmirrored map[string]bool
	// err, when set, fails every call.
	err error
	// calls counts store reads and writes, so tests can prove that a refused
	// request never touched the store.
	calls int
}

func (s *fakeStore) tick() string {
	s.clock++
	return strconv.FormatInt(s.clock, 10)
}

func (s *fakeStore) project(project string) map[string]kargoapi.Target {
	if s.targets == nil {
		s.targets = map[string]map[string]kargoapi.Target{}
	}
	if s.targets[project] == nil {
		s.targets[project] = map[string]kargoapi.Target{}
	}
	return s.targets[project]
}

// add stores a Target directly, bypassing the store's checks.
func (s *fakeStore) add(project, name string, labels map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	version := s.tick()
	s.project(project)[name] = kargoapi.Target{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         project,
			Name:              name,
			UID:               types.UID(uuid.NewString()),
			ResourceVersion:   version,
			CreationTimestamp: metav1.NewTime(time.UnixMicro(s.clock)),
			Labels:            labels,
		},
	}
}

func (s *fakeStore) CreateTarget(
	_ context.Context,
	project string,
	target *kargoapi.Target,
) (*kargoapi.Target, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	if s.unmirrored[project] {
		return nil, fmt.Errorf("Target %q: %w", target.Name, database.ErrProjectNotMirrored)
	}
	if _, ok := s.project(project)[target.Name]; ok {
		return nil, fmt.Errorf("Target %q: %w", target.Name, database.ErrAlreadyExists)
	}
	stored := *target.DeepCopy()
	stored.Namespace = project
	stored.UID = types.UID(uuid.NewString())
	stored.ResourceVersion = s.tick()
	stored.CreationTimestamp = metav1.NewTime(time.UnixMicro(s.clock))
	stored.Status = kargoapi.TargetStatus{}
	s.project(project)[target.Name] = stored
	return &stored, nil
}

func (s *fakeStore) GetTarget(_ context.Context, project, name string) (*kargoapi.Target, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	target, ok := s.project(project)[name]
	if !ok {
		return nil, fmt.Errorf("Target %q: %w", name, database.ErrNotFound)
	}
	return &target, nil
}

func (s *fakeStore) ListTargets(_ context.Context, project string) ([]kargoapi.Target, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	targets := make([]kargoapi.Target, 0, len(s.project(project)))
	for _, target := range s.project(project) {
		targets = append(targets, target)
	}
	slices.SortFunc(targets, func(a, b kargoapi.Target) int { return strings.Compare(a.Name, b.Name) })
	return targets, nil
}

func (s *fakeStore) UpdateTarget(
	_ context.Context,
	project string,
	target *kargoapi.Target,
) (*kargoapi.Target, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	current, ok := s.project(project)[target.Name]
	if !ok {
		return nil, fmt.Errorf("Target %q: %w", target.Name, database.ErrNotFound)
	}
	if target.UID != "" && target.UID != current.UID {
		return nil, fmt.Errorf("Target %q: %w", target.Name, database.ErrConflict)
	}
	if target.ResourceVersion != "" && target.ResourceVersion != current.ResourceVersion {
		return nil, fmt.Errorf("Target %q: %w", target.Name, database.ErrConflict)
	}
	current.Labels = target.Labels
	current.Spec = *target.Spec.DeepCopy()
	current.ResourceVersion = s.tick()
	s.project(project)[target.Name] = current
	return &current, nil
}

func (s *fakeStore) DeleteTarget(_ context.Context, project, name string) error {
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

// authorizationCall records what the middleware asked to authorize.
type authorizationCall struct {
	verb string
	gvr  schema.GroupVersionResource
	key  client.ObjectKey
}

// recordAuthorizations returns an authorizer that records every call and
// grants it.
func recordAuthorizations(calls *[]authorizationCall) kubernetes.Authorizer {
	return rest.Func(func(
		_ context.Context,
		verb string,
		gvr schema.GroupVersionResource,
		_ string,
		key client.ObjectKey,
	) error {
		*calls = append(*calls, authorizationCall{verb: verb, gvr: gvr, key: key})
		return nil
	})
}

// forbidEverything is an authorizer that refuses every call.
var forbidEverything = rest.Func(func(
	_ context.Context,
	_ string,
	gvr schema.GroupVersionResource,
	_ string,
	key client.ObjectKey,
) error {
	return apierrors.NewForbidden(gvr.GroupResource(), key.Name, errors.New("not authorized"))
})

// testCase drives one request through a Handler mounted the way the API
// server mounts it, under /v1beta1/projects/:project.
type testCase struct {
	name string
	// store is nil for a server without a database. A *fakeStore that is nil
	// must not be passed; use the field's zero value.
	store      *fakeStore
	authorize  kubernetes.Authorizer
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
func newRouter(store database.Store, authorize kubernetes.Authorizer, kube client.Reader) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			libhttp.WriteErrorJSON(c.Writer, c.Errors.Last().Err)
		}
	})
	New(store, authorize, kube).Register(router.Group("/v1beta1/projects/:project"))
	return router
}

func runTestCases(t *testing.T, cases []testCase) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var store database.Store
			if tc.store != nil {
				store = storeAdapter{tc.store}
			}
			authorize := tc.authorize
			if authorize == nil {
				authorize = recordAuthorizations(&[]authorizationCall{})
			}
			kube := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tc.objects...).Build()
			w := httptest.NewRecorder()
			newRouter(store, authorize, kube).ServeHTTP(w, httptest.NewRequest(tc.method, tc.url, tc.body))
			tc.assertions(t, w)
		})
	}
}

// storeAdapter lets a *fakeStore stand in for the database.Store that New
// takes, which has more methods than the handlers use.
type storeAdapter struct {
	*fakeStore
}

func (storeAdapter) UpsertProject(context.Context, database.UpsertProjectParams) error {
	panic("not implemented")
}

func (storeAdapter) DeleteProjectByName(context.Context, string) error {
	panic("not implemented")
}

func (storeAdapter) ListProjects(context.Context) ([]database.ProjectRow, error) {
	panic("not implemented")
}

func (storeAdapter) DeleteProjectsByID(context.Context, []string) error {
	panic("not implemented")
}

func projectURL(path string) string {
	return "/v1beta1/projects/" + testProject + "/targets" + path
}

func TestNew(t *testing.T) {
	t.Parallel()
	// A nil database.Store leaves the handler without a store, which
	// requireStore turns into 501 for every request.
	h := New(nil, nil, nil)
	require.Nil(t, h.store)
	h = New(storeAdapter{&fakeStore{}}, nil, nil)
	require.NotNil(t, h.store)
}

func TestRequireAccess(t *testing.T) {
	t.Parallel()
	var calls []authorizationCall
	runTestCases(t, []testCase{
		{
			name:      "records the verb, resource and key",
			store:     &fakeStore{},
			authorize: recordAuthorizations(&calls),
			method:    http.MethodDelete,
			url:       projectURL("/us-east-1"),
			assertions: func(t *testing.T, _ *httptest.ResponseRecorder) {
				require.Equal(t, []authorizationCall{{
					verb: "delete",
					gvr:  kargoapi.GroupVersion.WithResource("targets"),
					key:  client.ObjectKey{Namespace: testProject, Name: "us-east-1"},
				}}, calls)
			},
		},
		{
			name:      "the collection has no name",
			store:     &fakeStore{},
			authorize: recordAuthorizations(&calls),
			method:    http.MethodGet,
			url:       projectURL(""),
			assertions: func(t *testing.T, _ *httptest.ResponseRecorder) {
				require.Equal(t, authorizationCall{
					verb: "list",
					gvr:  kargoapi.GroupVersion.WithResource("targets"),
					key:  client.ObjectKey{Namespace: testProject},
				}, calls[len(calls)-1])
			},
		},
	})

	t.Run("no authorizer fails closed", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		newRouter(storeAdapter{&fakeStore{}}, nil, nil).ServeHTTP(
			w, httptest.NewRequest(http.MethodGet, projectURL(""), nil),
		)
		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
