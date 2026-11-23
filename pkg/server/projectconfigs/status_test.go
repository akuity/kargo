package projectconfigs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/auth/can"
	"github.com/akuity/kargo/pkg/server/projectconfigs/wire"
)

func TestStatus(t *testing.T) {
	t.Parallel()
	statusBody := func(t *testing.T, req wire.StatusRequest) *strings.Reader {
		data, err := json.Marshal(req)
		require.NoError(t, err)
		return strings.NewReader(string(data))
	}

	forbiddenStore := &fakeStore{}
	forbiddenCfg := forbiddenStore.add(kargoapi.ProjectConfigSpec{})
	conflictStore := &fakeStore{}
	conflictCfg := conflictStore.add(kargoapi.ProjectConfigSpec{})
	leftovers := leftoverStore()
	leftoverCfg := leftovers.configs[testProject]
	replacedStore := &fakeStore{}
	replacedStore.add(kargoapi.ProjectConfigSpec{})
	recordedStore := &fakeStore{skipped: []string{"taken"}}
	recordedCfg := recordedStore.add(kargoapi.ProjectConfigSpec{})
	var authorizations []can.Access

	conditions := []metav1.Condition{{
		Type:   "Ready",
		Status: metav1.ConditionTrue,
		Reason: "Synced",
	}}
	receivers := []kargoapi.WebhookReceiverDetails{
		{Name: "taken", Path: "/a", URL: "https://example.com/a"},
		{Name: "free", Path: "/b", URL: "https://example.com/b"},
	}

	runTestCases(t, []testCase{
		{
			name:      "not authorized",
			store:     forbiddenStore,
			authorize: forbidEverything,
			method:    http.MethodPut,
			url:       configURL("/status"),
			body: statusBody(t, wire.StatusRequest{
				ObservedUID:        forbiddenCfg.ID.String(),
				ObservedGeneration: 1,
			}),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
				require.Zero(t, forbiddenStore.calls)
			},
		},
		{
			name:   "malformed body",
			method: http.MethodPut,
			url:    configURL("/status"),
			body:   strings.NewReader(`{`),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			name:   "malformed observed UID",
			method: http.MethodPut,
			url:    configURL("/status"),
			body:   statusBody(t, wire.StatusRequest{ObservedUID: "not-a-uid"}),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnprocessableEntity, w.Code)
				require.Contains(t, w.Body.String(), "observedUID")
			},
		},
		{
			name:   "Project has no ProjectConfig",
			method: http.MethodPut,
			url:    configURL("/status"),
			body: statusBody(t, wire.StatusRequest{
				ObservedUID:        uuid.NewString(),
				ObservedGeneration: 1,
			}),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		{
			name:   "leftover row with matching observed ID returns 404",
			store:  leftovers,
			method: http.MethodPut,
			url:    configURL("/status"),
			body: statusBody(t, wire.StatusRequest{
				ObservedUID:        leftoverCfg.ID.String(),
				ObservedGeneration: leftoverCfg.Generation,
			}),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
				require.Nil(t, leftovers.status)
			},
		},
		{
			name:   "a status observed of a replaced ProjectConfig is a conflict",
			store:  replacedStore,
			method: http.MethodPut,
			url:    configURL("/status"),
			body: statusBody(t, wire.StatusRequest{
				ObservedUID:        uuid.NewString(),
				ObservedGeneration: 1,
			}),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, w.Code)
				require.Nil(t, replacedStore.status)
			},
		},
		{
			name:   "a stale status is a conflict",
			store:  conflictStore,
			method: http.MethodPut,
			url:    configURL("/status"),
			body: statusBody(t, wire.StatusRequest{
				ObservedUID:        conflictCfg.ID.String(),
				ObservedGeneration: 7,
			}),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, w.Code)
			},
		},
		{
			name:      "records the status and echoes skipped receivers",
			store:     recordedStore,
			authorize: recordAuthorizations(&authorizations),
			method:    http.MethodPut,
			url:       configURL("/status"),
			body: statusBody(t, wire.StatusRequest{
				ObservedUID:          recordedCfg.ID.String(),
				ObservedGeneration:   1,
				ObservedRefreshToken: "token",
				Conditions:           conditions,
				WebhookReceivers:     receivers,
			}),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
				access := can.Update().ProjectConfigStatus(testProject)
				require.Equal(t, "status", access.Subresource)
				require.Equal(t, []can.Access{access}, authorizations)

				var resp wire.StatusResponse
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
				require.Equal(t, []string{"taken"}, resp.SkippedReceivers)

				status := recordedStore.status
				require.NotNil(t, status)
				require.Equal(t, recordedCfg.ID, status.ObservedID)
				require.Equal(t, int64(1), status.ObservedGeneration)
				require.Equal(t, "token", status.ObservedRefreshToken)
				require.Len(t, status.Conditions, 1)
				require.Equal(t, "Ready", status.Conditions[0].Type)
				require.Equal(t, receivers, status.WebhookReceivers)
			},
		},
	})
}
