package projectconfigs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/server/auth/can"
	"github.com/akuity/kargo/pkg/server/middleware"
	"github.com/akuity/kargo/pkg/server/projectconfigs/wire"
)

const (
	testReceiverName = "my-receiver"
	testReceiverPath = "/webhook/abc123"
	testReceiverURL  = "https://kargo.example.com/webhook/abc123"
)

func lookupURL(path string) string {
	u := "/v1beta1/system/webhook-receivers"
	if path != "" {
		u += "?path=" + url.QueryEscape(path)
	}
	return u
}

func newSystemRouter(
	store *fakeStore,
	authorize func(context.Context, can.Access) error,
) *gin.Engine {
	router := gin.New()
	router.Use(middleware.HandleErrors(), middleware.Recover())
	New(store, authorize, nil).RegisterSystem(router.Group("/v1beta1/system"))
	return router
}

// storeWithReceiver returns a store whose ProjectConfig has the named
// receivers in its spec and a status entry for testReceiverName.
func storeWithReceiver(specNames ...string) *fakeStore {
	store := &fakeStore{}
	spec := kargoapi.ProjectConfigSpec{}
	for _, n := range specNames {
		spec.WebhookReceivers = append(
			spec.WebhookReceivers,
			kargoapi.WebhookReceiverConfig{Name: n},
		)
	}
	cfg := store.add(spec)
	cfg.Receivers = []database.ProjectConfigWebhookReceiverRow{{
		Name: testReceiverName,
		Path: testReceiverPath,
		URL:  testReceiverURL,
	}}
	store.configs[testProject] = cfg
	return store
}

func TestLookupWebhookReceiver(t *testing.T) {
	t.Parallel()
	var authorizations []can.Access
	testCases := []struct {
		name      string
		store     *fakeStore
		authorize func(context.Context, can.Access) error
		path      string
		assert    func(*testing.T, *fakeStore, *httptest.ResponseRecorder)
	}{
		{
			name:      "not authorized",
			store:     storeWithReceiver(testReceiverName),
			authorize: forbidEverything,
			path:      testReceiverPath,
			assert: func(t *testing.T, s *fakeStore, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
				require.Zero(t, s.calls)
			},
		},
		{
			name:      "missing path",
			store:     storeWithReceiver(testReceiverName),
			authorize: recordAuthorizations(&[]can.Access{}),
			assert: func(t *testing.T, s *fakeStore, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
				require.Zero(t, s.calls)
			},
		},
		{
			name:      "no visible owner",
			store:     &fakeStore{},
			authorize: recordAuthorizations(&[]can.Access{}),
			path:      testReceiverPath,
			assert: func(t *testing.T, _ *fakeStore, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		{
			name:      "owner's spec no longer has the receiver",
			store:     storeWithReceiver("another-receiver"),
			authorize: recordAuthorizations(&[]can.Access{}),
			path:      testReceiverPath,
			assert: func(t *testing.T, _ *fakeStore, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		{
			name:      "found",
			store:     storeWithReceiver("another-receiver", testReceiverName),
			authorize: recordAuthorizations(&authorizations),
			path:      testReceiverPath,
			assert: func(t *testing.T, _ *fakeStore, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
				require.Equal(
					t,
					[]can.Access{can.List().Kargo("projectconfigs")},
					authorizations,
				)
				var resp wire.WebhookReceiverLookup
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
				require.Equal(t, testProject, resp.Project)
				require.Equal(t, testReceiverName, resp.Config.Name)
				require.Equal(
					t,
					kargoapi.WebhookReceiverDetails{
						Name: testReceiverName,
						Path: testReceiverPath,
						URL:  testReceiverURL,
					},
					resp.Receiver,
				)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			newSystemRouter(tc.store, tc.authorize).ServeHTTP(
				w,
				httptest.NewRequest(http.MethodGet, lookupURL(tc.path), nil),
			)
			tc.assert(t, tc.store, w)
		})
	}
}
