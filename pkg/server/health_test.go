package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeHealthChecker is a HealthChecker whose results the test controls.
type fakeHealthChecker struct {
	readyErr error
	liveErr  error
}

func (f *fakeHealthChecker) Ready() error { return f.readyErr }

func (f *fakeHealthChecker) Live() error { return f.liveErr }

func TestNewHealthHandler(t *testing.T) {
	testCases := []struct {
		name   string
		method string
		check  func() error
		assert func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "POST is not allowed",
			method: http.MethodPost,
			assert: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusMethodNotAllowed, rec.Code)
			},
		},
		{
			name:   "GET without a check returns 200",
			method: http.MethodGet,
			assert: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
				require.Equal(t, "ok", rec.Body.String())
			},
		},
		{
			name:   "GET with a passing check returns 200",
			method: http.MethodGet,
			check:  func() error { return nil },
			assert: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
				require.Equal(t, "ok", rec.Body.String())
			},
		},
		{
			name:   "GET with a failing check returns 503",
			method: http.MethodGet,
			check:  func() error { return errors.New("something is wrong") },
			assert: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusServiceUnavailable, rec.Code)
				require.Equal(t, "something is wrong", rec.Body.String())
			},
		},
		{
			name:   "HEAD with a passing check returns 200",
			method: http.MethodHead,
			assert: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
				require.Empty(t, rec.Body.String())
			},
		},
		{
			name:   "HEAD with a failing check returns 503",
			method: http.MethodHead,
			check:  func() error { return errors.New("something is wrong") },
			assert: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusServiceUnavailable, rec.Code)
				require.Empty(t, rec.Body.String())
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			req := httptest.NewRequest(testCase.method, "/healthz", nil)
			rec := httptest.NewRecorder()
			newHealthHandler(testCase.check).ServeHTTP(rec, req)
			testCase.assert(t, rec)
		})
	}
}

func TestServerHandleHealth(t *testing.T) {
	testCases := []struct {
		name   string
		health HealthChecker
		assert func(t *testing.T, live, ready *httptest.ResponseRecorder)
	}{
		{
			name: "no health checker",
			assert: func(t *testing.T, live, ready *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, live.Code)
				require.Equal(t, http.StatusOK, ready.Code)
			},
		},
		{
			name:   "healthy",
			health: &fakeHealthChecker{},
			assert: func(t *testing.T, live, ready *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, live.Code)
				require.Equal(t, http.StatusOK, ready.Code)
			},
		},
		{
			name:   "live but not ready",
			health: &fakeHealthChecker{readyErr: errors.New("not ready")},
			assert: func(t *testing.T, live, ready *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, live.Code)
				require.Equal(t, http.StatusServiceUnavailable, ready.Code)
				require.Equal(t, "not ready", ready.Body.String())
			},
		},
		{
			name: "neither live nor ready",
			health: &fakeHealthChecker{
				readyErr: errors.New("not ready"),
				liveErr:  errors.New("not live"),
			},
			assert: func(t *testing.T, live, ready *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusServiceUnavailable, live.Code)
				require.Equal(t, "not live", live.Body.String())
				require.Equal(t, http.StatusServiceUnavailable, ready.Code)
				require.Equal(t, "not ready", ready.Body.String())
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			mux := http.NewServeMux()
			(&server{health: testCase.health}).handleHealth(mux)
			get := func(path string) *httptest.ResponseRecorder {
				rec := httptest.NewRecorder()
				mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
				return rec
			}
			testCase.assert(t, get("/healthz"), get("/readyz"))
		})
	}
}
