package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/akuity/kargo/pkg/heartbeat"
	"github.com/akuity/kargo/pkg/heartbeat/heartbeattest"
	"github.com/akuity/kargo/pkg/server/config"
)

func Test_server_getControllerHeartbeats(t *testing.T) {
	const leaseDuration = 30 * time.Second
	now := time.Now()
	freshTime := now.Add(-5 * time.Second)
	staleTime := now.Add(-1 * time.Hour)

	testRESTEndpoint(
		t,
		&config.ServerConfig{
			KargoNamespace: testKargoNamespace,
		},
		http.MethodGet, "/v1beta1/system/controller-heartbeats",
		[]restTestCase{
			{
				name: "no heartbeats",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					var resp getControllerHeartbeatsResponse
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
					require.Empty(t, resp.Heartbeats)
					require.Empty(t, resp.DefaultController,
						"DefaultController defaults to empty (the unnamed controller)")
				},
			},
			{
				name: "operator-configured default controller name is reflected in response",
				serverConfig: &config.ServerConfig{
					KargoNamespace:        testKargoNamespace,
					DefaultControllerName: "primary",
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					var resp getControllerHeartbeatsResponse
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
					require.Equal(t, "primary", resp.DefaultController)
				},
			},
			{
				name: "alive controller",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					heartbeattest.NewHeartbeatLease(
						testKargoNamespace, "alpha",
						heartbeattest.WithRenewedAt(freshTime),
						heartbeattest.WithDuration(leaseDuration),
					),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					var resp getControllerHeartbeatsResponse
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
					require.Len(t, resp.Heartbeats, 1)
					hb, ok := resp.Heartbeats["alpha"]
					require.True(t, ok)
					require.Equal(t, "alpha", hb.Controller)
					require.Equal(t, heartbeat.StatusAlive, hb.Status)
					require.NotNil(t, hb.Timestamp)
				},
			},
			{
				name: "dead controller",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					heartbeattest.NewHeartbeatLease(
						testKargoNamespace, "beta",
						heartbeattest.WithRenewedAt(staleTime),
						heartbeattest.WithDuration(leaseDuration),
					),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					var resp getControllerHeartbeatsResponse
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
					require.Len(t, resp.Heartbeats, 1)
					hb, ok := resp.Heartbeats["beta"]
					require.True(t, ok)
					require.Equal(t, heartbeat.StatusDead, hb.Status)
				},
			},
			{
				name: "unnamed controller is keyed by empty string",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					heartbeattest.NewHeartbeatLease(
						testKargoNamespace, "",
						heartbeattest.WithRenewedAt(freshTime),
						heartbeattest.WithDuration(leaseDuration),
					),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					var resp getControllerHeartbeatsResponse
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
					require.Len(t, resp.Heartbeats, 1)
					hb, ok := resp.Heartbeats[""]
					require.True(t, ok,
						"unnamed controller heartbeat must be keyed by empty string, "+
							"so that response.DefaultController=\"\" finds it")
					require.Equal(t, heartbeat.StatusAlive, hb.Status)
					require.Empty(t, hb.Controller)
				},
			},
			{
				name: "multiple controllers reported in one response",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					heartbeattest.NewHeartbeatLease(
						testKargoNamespace, "alpha",
						heartbeattest.WithRenewedAt(freshTime),
						heartbeattest.WithDuration(leaseDuration),
					),
					heartbeattest.NewHeartbeatLease(
						testKargoNamespace, "beta",
						heartbeattest.WithRenewedAt(staleTime),
						heartbeattest.WithDuration(leaseDuration),
					),
					heartbeattest.NewHeartbeatLease(
						testKargoNamespace, "",
						heartbeattest.WithRenewedAt(freshTime),
						heartbeattest.WithDuration(leaseDuration),
					),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					var resp getControllerHeartbeatsResponse
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
					require.Len(t, resp.Heartbeats, 3)
					require.Equal(t, heartbeat.StatusAlive, resp.Heartbeats["alpha"].Status)
					require.Equal(t, heartbeat.StatusDead, resp.Heartbeats["beta"].Status)
					require.Equal(t, heartbeat.StatusAlive, resp.Heartbeats[""].Status)
				},
			},
		},
	)
}
