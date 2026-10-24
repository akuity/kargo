package targets

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/server/auth/can"
)

func TestListPromotions(t *testing.T) {
	t.Parallel()
	newStore := func() *fakeStore {
		store := &fakeStore{}
		store.add(testProject, "a", nil)
		store.add(testProject, "b", nil)
		store.addPromotion("a", "fleet", "p1", "Succeeded", 1)
		store.addPromotion("b", "fleet", "p2", "Succeeded", 2)
		store.addPromotion("a", "europe", "p3", "", 3)
		return store
	}
	names := func(t *testing.T, w *httptest.ResponseRecorder) []string {
		require.Equal(t, http.StatusOK, w.Code)
		list := &TargetPromotionList{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), list))
		require.NotNil(t, list.Items)
		out := make([]string, 0, len(list.Items))
		for _, promotion := range list.Items {
			out = append(out, promotion.Name)
		}
		return out
	}
	var authorizations []can.Access
	forbiddenStore := newStore()

	runTestCases(t, []testCase{
		{
			name:   "database is not configured",
			method: http.MethodGet,
			url:    projectURL("/a/promotions"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotImplemented, w.Code)
			},
		},
		{
			name:      "not authorized",
			store:     forbiddenStore,
			authorize: forbidEverything,
			method:    http.MethodGet,
			url:       projectURL("/a/promotions"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
				require.Zero(t, forbiddenStore.calls)
			},
		},
		{
			name:      "lists the Target's own Promotions, newest first",
			store:     newStore(),
			authorize: recordAuthorizations(&authorizations),
			method:    http.MethodGet,
			url:       projectURL("/a/promotions"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
				list := &TargetPromotionList{}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), list))
				require.Len(t, list.Items, 2)
				require.Equal(t, "p3", list.Items[0].Name)
				require.Equal(t, "europe", list.Items[0].Stage)
				require.Equal(t, "f-p3", list.Items[0].Freight)
				require.Empty(t, list.Items[0].Phase)
				require.Equal(t, "p1", list.Items[1].Name)
				require.Equal(t, "Succeeded", list.Items[1].Phase)
				require.Equal(t, []can.Access{can.Get().Target(testProject, "a")}, authorizations)
			},
		},
		{
			name:   "a Target with no Promotions has an empty history",
			store:  newStore(),
			method: http.MethodGet,
			url:    projectURL("/c/promotions"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Empty(t, names(t, w))
			},
		},
		{
			name:   "a limit caps the history",
			store:  newStore(),
			method: http.MethodGet,
			url:    projectURL("/a/promotions?limit=1"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, []string{"p3"}, names(t, w))
			},
		},
		{
			name:   "rejects a limit out of range",
			store:  newStore(),
			method: http.MethodGet,
			url:    projectURL("/a/promotions?limit=1001"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
	})
}

func TestParseLimit(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		raw     string
		want    int32
		wantErr bool
	}{
		{raw: "", want: defaultPromotionsLimit},
		{raw: "1", want: 1},
		{raw: "1000", want: maxPromotionsLimit},
		{raw: "0", wantErr: true},
		{raw: "1001", wantErr: true},
		{raw: "-5", wantErr: true},
		{raw: "ten", wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.raw, func(t *testing.T) {
			t.Parallel()
			limit, err := parseLimit(testCase.raw)
			if testCase.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, testCase.want, limit)
		})
	}
}
