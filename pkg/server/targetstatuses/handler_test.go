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
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/akuity/kargo/pkg/database/targetstatusstore"
	"github.com/akuity/kargo/pkg/server/auth/can"
	"github.com/akuity/kargo/pkg/server/middleware"
)

const testProject = "fake-project"

// fakeStore is an in-memory targetstatusstore.Store. It holds what the
// database is taken to have on record, and bounds what it returns the way
// the real store does, so that what the handlers ask for and what they make
// of the answer can both be checked.
type fakeStore struct {
	// The store's interface is embedded so that the fake keeps satisfying
	// it as the store grows. A method the fake does not implement panics if
	// a handler calls it.
	targetstatusstore.Store
	mu sync.Mutex
	// promotions holds the Promotions to a Target, by Target name.
	promotions map[string][]targetstatusstore.Promotion
	// err, when set, fails every call.
	err error
	// calls counts store reads, so tests can prove that a refused request
	// never touched the store.
	calls int
	// promotionOpts records what was asked for.
	promotionOpts []targetstatusstore.PromotionListOptions
}

var _ targetstatusstore.Store = (*fakeStore)(nil)

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
) *gin.Engine {
	router := gin.New()
	router.Use(middleware.HandleErrors(), middleware.Recover())
	New(store, authorize).Register(router.Group("/v1beta1/projects/:project"))
	return router
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
			newRouter(store, authorize).ServeHTTP(
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
	h := New(nil, nil)
	require.Nil(t, h.store)
	h = New(&fakeStore{}, nil)
	require.NotNil(t, h.store)
}

func TestRegister(t *testing.T) {
	t.Parallel()
	route := projectURL("/targets/us-east-1/promotions")

	cases := []testCase{{
		name: "database is not configured",
		url:  route,
		// Not being able to serve the request comes before whether the
		// caller may make it.
		authorize: forbid(),
		assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
			require.Equal(t, http.StatusNotImplemented, w.Code)
		},
	}}

	// The route is refused, before the store is read, to a caller who lacks
	// either of the accesses it takes.
	for _, resource := range []string{"targets", "promotions"} {
		store := &fakeStore{}
		cases = append(cases, testCase{
			name:      "may not read " + resource,
			store:     store,
			authorize: forbid(resource),
			url:       route,
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
				require.Zero(t, store.calls)
			},
		})
	}

	var calls []can.Access
	cases = append(cases, testCase{
		name:      "authorizes get on the Target and list on the Promotions",
		store:     &fakeStore{},
		authorize: recordAuthorizations(&calls),
		url:       route,
		assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(
				t,
				[]can.Access{
					can.Get().Target(testProject, "us-east-1"),
					can.List().Promotion(testProject, ""),
				},
				calls,
			)
		},
	})
	runTestCases(t, cases)

	t.Run("no authorizer fails closed", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		newRouter(&fakeStore{}, nil).ServeHTTP(
			w, httptest.NewRequest(http.MethodGet, route, nil),
		)
		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
