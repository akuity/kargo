package version

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	kargogen "github.com/akuity/kargo/pkg/x/client/generated"
)

func TestGetServerVersion(t *testing.T) {
	want := &kargogen.VersionInfo{}
	want.SetVersion("v1.99.0")
	wantJSON, err := json.Marshal(want)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(wantJSON)
		},
	))
	defer srv.Close()

	cfg := config.CLIConfig{APIAddress: srv.URL, BearerToken: "test-token"}
	got, err := getServerVersion(t.Context(), cfg, client.Options{})
	require.NoError(t, err)
	require.Equal(t, want.GetVersion(), got.GetVersion())
}

func TestGetServerVersion_NotLoggedIn(t *testing.T) {
	got, err := getServerVersion(t.Context(), config.CLIConfig{}, client.Options{})
	require.NoError(t, err)
	require.Nil(t, got)
}
