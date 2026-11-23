package refresh

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database/projectconfigstore"
	"github.com/akuity/kargo/pkg/x/client/watch"
)

const testToken = "tok-1"

// sseWatchClient returns a watch client backed by a server that streams the
// given objects as MODIFIED watch events and then ends the stream.
func sseWatchClient(t *testing.T, objs ...any) *watch.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			for _, obj := range objs {
				data, err := json.Marshal(map[string]any{
					"type":   "MODIFIED",
					"object": obj,
				})
				require.NoError(t, err)
				_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
				http.NewResponseController(w).Flush()
			}
		},
	))
	t.Cleanup(srv.Close)
	return watch.NewClient(srv.URL, srv.Client(), "")
}

func projectConfigObj(
	annotations map[string]string,
	lastHandled string,
) map[string]any {
	return map[string]any{
		"metadata": map[string]any{
			"name":        "my-project",
			"annotations": annotations,
		},
		"status": map[string]any{"lastHandledRefresh": lastHandled},
	}
}

func TestWaitForProjectConfig(t *testing.T) {
	tokenAnn := map[string]string{kargoapi.AnnotationKeyRefresh: testToken}
	token := testToken
	testCases := []struct {
		name   string
		token  string
		events []any
		assert func(*testing.T, error)
	}{
		{
			name:  "token matches without annotation",
			token: testToken,
			events: []any{
				projectConfigObj(nil, "older"),
				projectConfigObj(nil, testToken),
			},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:   "token never matches",
			token:  testToken,
			events: []any{projectConfigObj(nil, "older")},
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unexpected end of watch stream")
			},
		},
		{
			name: "no token falls back to annotation",
			events: []any{
				projectConfigObj(tokenAnn, "older"),
				projectConfigObj(tokenAnn, testToken),
			},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:   "no token and no annotation",
			events: []any{projectConfigObj(nil, "older")},
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "annotation")
			},
		},
		{
			// An older CLI (no token) against the new server, which
			// synthesizes the annotation from the stored token.
			name: "annotation synthesized by the store",
			events: []any{projectconfigstore.ToAPI(projectconfigstore.Config{
				ProjectName:              "my-project",
				RefreshToken:             &token,
				StatusLastHandledRefresh: &token,
			})},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(
				context.Background(), 10*time.Second,
			)
			defer cancel()
			wc := sseWatchClient(t, tc.events...)
			tc.assert(t, waitForProjectConfig(ctx, wc, "my-project", tc.token))
		})
	}
}
