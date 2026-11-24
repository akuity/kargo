package targetstatuses

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/database/targetstatusstore"
	"github.com/akuity/kargo/pkg/server/auth/can"
	"github.com/akuity/kargo/pkg/server/middleware"
)

const testProject = "fake-project"

// fakeStore is an in-memory targetstatusstore.Store. It derives nothing: it
// holds the entry each pair should be given, and applies the State filter
// and the limit the way the real store does, so that what the handlers ask
// for and what they make of the answer can both be checked.
type fakeStore struct {
	mu sync.Mutex
	// targets holds every Target by Project.
	targets map[string][]database.ListTargetLabelsRow
	// entries holds the entry of a pair, by Target name and Stage id; see
	// pairKey. A pair without one has never been promoted to.
	entries map[string]targetstatusstore.Entry
	// summary is what Summarize returns.
	summary targetstatusstore.Summary
	// promotions holds the Promotions to a Target, by Target name.
	promotions map[string][]targetstatusstore.Promotion
	// err, when set, fails every call.
	err error
	// calls counts store reads, so tests can prove that a refused request
	// never touched the store.
	calls int
	// scopes, states, limits and promotionOpts record what was asked for.
	scopes        []targetstatusstore.Scope
	states        [][]targetstatusstore.State
	limits        []int32
	promotionOpts []targetstatusstore.PromotionListOptions
}

var _ targetstatusstore.Store = (*fakeStore)(nil)

func pairKey(target, stageID string) string { return target + "/" + stageID }

// add stores a Target in the test Project.
func (s *fakeStore) add(name string, labels map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.targets == nil {
		s.targets = map[string][]database.ListTargetLabelsRow{}
	}
	s.targets[testProject] = append(
		s.targets[testProject],
		database.ListTargetLabelsRow{ID: uuid.New(), Name: name, Labels: labels},
	)
}

// set gives the pair of the Target and the Stage an entry.
func (s *fakeStore) set(target, stageID string, entry targetstatusstore.Entry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.entries == nil {
		s.entries = map[string]targetstatusstore.Entry{}
	}
	s.entries[pairKey(target, stageID)] = entry
}

func (s *fakeStore) Targets(
	_ context.Context,
	project string,
) ([]database.ListTargetLabelsRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	rows := slices.Clone(s.targets[project])
	slices.SortFunc(rows, func(a, b database.ListTargetLabelsRow) int {
		return strings.Compare(a.Name, b.Name)
	})
	return rows, nil
}

func (s *fakeStore) List(
	_ context.Context,
	scope targetstatusstore.Scope,
	states []targetstatusstore.State,
	limit int32,
) ([]targetstatusstore.Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	s.scopes = append(s.scopes, scope)
	s.states = append(s.states, states)
	s.limits = append(s.limits, limit)
	entries := []targetstatusstore.Entry{}
	for _, pair := range scope.Pairs {
		entry, ok := s.entries[pairKey(pair.TargetName, pair.StageID)]
		if !ok {
			entry = targetstatusstore.Entry{State: targetstatusstore.StateNeverPromoted}
		}
		if len(states) > 0 && !slices.Contains(states, entry.State) {
			continue
		}
		entry.TargetID, entry.StageID = pair.TargetID, pair.StageID
		if entries = append(entries, entry); len(entries) == int(limit) {
			break
		}
	}
	return entries, nil
}

func (s *fakeStore) Summarize(
	_ context.Context,
	scope targetstatusstore.Scope,
) (targetstatusstore.Summary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return targetstatusstore.Summary{}, s.err
	}
	s.scopes = append(s.scopes, scope)
	return s.summary, nil
}

func (s *fakeStore) ListPromotions(
	_ context.Context,
	project string,
	target string,
	opts targetstatusstore.PromotionListOptions,
) ([]targetstatusstore.Promotion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	s.promotionOpts = append(s.promotionOpts, opts)
	promotions := []targetstatusstore.Promotion{}
	if project != testProject {
		return promotions, nil
	}
	for _, promotion := range s.promotions[target] {
		if opts.Before != nil && !promotion.CreatedAt.Before(opts.Before.CreatedAt) {
			continue
		}
		if promotions = append(promotions, promotion); len(promotions) == int(opts.Limit) {
			break
		}
	}
	return promotions, nil
}

// newStage returns a Stage of the test Project with the UID "<name>-uid".
// With selectors it is target-aware; without, classic.
func newStage(name string, selectors ...metav1.LabelSelector) *kargoapi.Stage {
	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject,
			Name:      name,
			UID:       types.UID(name + "-uid"),
		},
	}
	if selectors != nil {
		stage.Spec.Targets = &kargoapi.StageTargets{Selectors: selectors}
	}
	return stage
}

// onFreight records the named Freight as the Stage's current collection, one
// piece per Warehouse, and returns the Stage.
func onFreight(stage *kargoapi.Stage, names ...string) *kargoapi.Stage {
	collection := &kargoapi.FreightCollection{}
	for _, name := range names {
		collection.UpdateOrPush(kargoapi.FreightReference{
			Name: name,
			Origin: kargoapi.FreightOrigin{
				Kind: kargoapi.FreightOriginKindWarehouse,
				Name: "warehouse-" + name,
			},
		})
	}
	stage.Status.FreightHistory = kargoapi.FreightHistory{collection}
	return stage
}

// everything selects every Target.
func everything() metav1.LabelSelector { return metav1.LabelSelector{} }

// labeled selects the Targets with the label.
func labeled(key, value string) metav1.LabelSelector {
	return metav1.LabelSelector{MatchLabels: map[string]string{key: value}}
}

// recordAuthorizations returns an authorize function that records every
// access it is asked about and grants it.
func recordAuthorizations(calls *[]can.Access) func(context.Context, can.Access) error {
	return func(_ context.Context, access can.Access) error {
		*calls = append(*calls, access)
		return nil
	}
}

// forbid returns an authorize function that refuses the accesses to the
// named resources and grants the rest. With no resources it refuses all.
func forbid(resources ...string) func(context.Context, can.Access) error {
	return func(_ context.Context, access can.Access) error {
		if len(resources) > 0 && !slices.Contains(resources, access.Resource.Resource) {
			return nil
		}
		return apierrors.NewForbidden(
			access.Resource.GroupResource(),
			access.Key.Name,
			errors.New("not authorized"),
		)
	}
}

// testCase drives one request through a Handler mounted the way the API
// server mounts it, under /v1beta1/projects/:project.
type testCase struct {
	name string
	// store is nil for a server without a database.
	store      *fakeStore
	authorize  func(context.Context, can.Access) error
	objects    []client.Object
	url        string
	assertions func(*testing.T, *httptest.ResponseRecorder)
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	gin.DefaultWriter = io.Discard
	gin.DefaultErrorWriter = io.Discard
	os.Exit(m.Run())
}

// newRouter mounts a Handler the way the API server mounts it, behind the
// same error handling: an error with an HTTP status is reported with it,
// anything else is a 500.
func newRouter(
	store targetstatusstore.Store,
	authorize func(context.Context, can.Access) error,
	kube client.Reader,
) *gin.Engine {
	router := gin.New()
	router.Use(middleware.HandleErrors(), middleware.Recover())
	New(store, authorize, kube).Register(router.Group("/v1beta1/projects/:project"))
	return router
}

// newKube returns a Kubernetes client holding the objects.
func newKube(t *testing.T, objects ...client.Object) client.Client {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
}

func runTestCases(t *testing.T, cases []testCase) {
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var store targetstatusstore.Store
			if tc.store != nil {
				store = tc.store
			}
			authorize := tc.authorize
			if authorize == nil {
				authorize = recordAuthorizations(&[]can.Access{})
			}
			w := httptest.NewRecorder()
			newRouter(store, authorize, newKube(t, tc.objects...)).ServeHTTP(
				w, httptest.NewRequest(http.MethodGet, tc.url, nil),
			)
			tc.assertions(t, w)
		})
	}
}

func projectURL(path string) string {
	return "/v1beta1/projects/" + testProject + path
}

// decode reads a 200 response's body into a value of the type.
func decode[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var out T
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	return out
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

func TestRegister(t *testing.T) {
	t.Parallel()
	routes := []string{
		projectURL("/target-statuses"),
		projectURL("/target-statuses/summary"),
		projectURL("/targets/us-east-1/promotions"),
	}

	var cases []testCase
	for _, url := range routes {
		cases = append(cases, testCase{
			name: "database is not configured: " + url,
			url:  url,
			// Not being able to serve the request comes before whether the
			// caller may make it.
			authorize: forbid(),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotImplemented, w.Code)
			},
		})
	}

	// Each route is refused, before the store is read, to a caller who
	// lacks any one of the accesses it takes.
	for _, url := range routes[:2] {
		for _, resource := range []string{"targets", "promotions", "promotionrollouts"} {
			store := &fakeStore{}
			cases = append(cases, testCase{
				name:      "may not list " + resource + ": " + url,
				store:     store,
				authorize: forbid(resource),
				url:       url,
				assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
					require.Equal(t, http.StatusForbidden, w.Code)
					require.Zero(t, store.calls)
				},
			})
		}
	}
	for _, resource := range []string{"targets", "promotions"} {
		store := &fakeStore{}
		cases = append(cases, testCase{
			name:      "may not read " + resource + ": " + routes[2],
			store:     store,
			authorize: forbid(resource),
			url:       routes[2],
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
				require.Zero(t, store.calls)
			},
		})
	}

	// What each route asks for.
	fleet := []can.Access{
		can.List().Target(testProject, ""),
		can.List().Promotion(testProject, ""),
		can.List().PromotionRollout(testProject, ""),
	}
	for url, expected := range map[string][]can.Access{
		routes[0]: fleet,
		routes[1]: fleet,
		routes[2]: {
			can.Get().Target(testProject, "us-east-1"),
			can.List().Promotion(testProject, ""),
		},
	} {
		var calls []can.Access
		cases = append(cases, testCase{
			name:      "authorizes: " + url,
			store:     &fakeStore{},
			authorize: recordAuthorizations(&calls),
			url:       url,
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
				require.Equal(t, expected, calls)
			},
		})
	}
	runTestCases(t, cases)

	t.Run("no authorizer fails closed", func(t *testing.T) {
		t.Parallel()
		for _, url := range routes {
			w := httptest.NewRecorder()
			newRouter(&fakeStore{}, nil, newKube(t)).ServeHTTP(
				w, httptest.NewRequest(http.MethodGet, url, nil),
			)
			require.Equal(t, http.StatusInternalServerError, w.Code)
		}
	})
}
