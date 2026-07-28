package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewHealthHandler(t *testing.T) {
	testCases := []struct {
		name   string
		method string
		assert func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "GET returns 200",
			method: http.MethodGet,
			assert: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
				require.Equal(t, "ok", rec.Body.String())
			},
		},
		{
			name:   "HEAD returns 200",
			method: http.MethodHead,
			assert: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
			},
		},
		{
			name:   "POST is not allowed",
			method: http.MethodPost,
			assert: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusMethodNotAllowed, rec.Code)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			req := httptest.NewRequest(testCase.method, "/healthz", nil)
			rec := httptest.NewRecorder()
			newHealthHandler().ServeHTTP(rec, req)
			testCase.assert(t, rec)
		})
	}
}
