package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/releases"
	"github.com/akuity/kargo/pkg/server/config"
)

type mockReleaseSvc struct {
	releases []releases.Release
}

func (m *mockReleaseSvc) GetBestReleases() []releases.Release {
	return m.releases
}

func Test_server_getCLIReleases(t *testing.T) {
	gin.DefaultWriter = io.Discard
	gin.DefaultErrorWriter = io.Discard

	testCases := []struct {
		name       string
		releaseSvc releases.Service
		assertions func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success with releases",
			releaseSvc: &mockReleaseSvc{
				releases: []releases.Release{
					{
						Version: semver.MustParse("1.1.0"),
						CLIBinaries: releases.CLIBinaries{
							"linux": {"amd64": "https://example.com/dl"},
						},
					},
					{
						Version: semver.MustParse("1.0.2"),
						CLIBinaries: releases.CLIBinaries{
							"linux": {"amd64": "https://example.com/dl2"},
						},
					},
				},
			},
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)

				var body map[string]any
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
				rels, ok := body["releases"].([]any)
				require.True(t, ok)
				require.Len(t, rels, 2)
				assert.Equal(t, "1.1.0", rels[0].(map[string]any)["version"]) // nolint: forcetypeassert
				assert.Equal(t, "1.0.2", rels[1].(map[string]any)["version"]) // nolint: forcetypeassert
			},
		},
		{
			name:       "empty releases",
			releaseSvc: &mockReleaseSvc{},
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
				var body map[string]any
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
				assert.Nil(t, body["releases"])
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := &server{
				cfg:        config.ServerConfig{},
				releaseSvc: tc.releaseSvc,
			}

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/v1beta1/kargo-releases/best", nil)
			router := s.setupRESTRouter(t.Context())
			router.ServeHTTP(w, req)

			tc.assertions(t, w)
		})
	}
}
