package server

import (
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/akuity/kargo/pkg/server/config"
)

func Test_server_listShards(t *testing.T) {
	const deadline = 10 * time.Minute
	now := time.Now()
	freshTime := now.Add(-1 * time.Minute)
	staleTime := now.Add(-1 * time.Hour)

	heartbeatCM := func(shard string, observedAt *time.Time, extraData map[string]string) *corev1.ConfigMap {
		data := map[string]string{}
		if observedAt != nil {
			data[agentObservedAtKey] = observedAt.Format(time.RFC3339)
		}
		maps.Copy(data, extraData)
		return &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testKargoNamespace,
				Name:      "agent-" + shard + ".status",
				Labels: map[string]string{
					kargoAgentLabelKey: shard,
				},
			},
			Data: data,
		}
	}

	testRESTEndpoint(
		t,
		&config.ServerConfig{
			KargoNamespace:      testKargoNamespace,
			AgentStatusDeadline: deadline,
		},
		http.MethodGet, "/v1beta1/system/shards",
		[]restTestCase{
			{
				name: "no heartbeat ConfigMaps exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					var resp listShardsResponse
					err := json.Unmarshal(w.Body.Bytes(), &resp)
					require.NoError(t, err)
					require.Empty(t, resp.Shards)
					require.Empty(t, resp.DefaultShardName)
				},
			},
			{
				name: "default shard name is reported",
				serverConfig: &config.ServerConfig{
					KargoNamespace:      testKargoNamespace,
					AgentStatusDeadline: deadline,
					DefaultShardName:    "primary",
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					var resp listShardsResponse
					err := json.Unmarshal(w.Body.Bytes(), &resp)
					require.NoError(t, err)
					require.Empty(t, resp.Shards)
					require.Equal(t, "primary", resp.DefaultShardName)
				},
			},
			{
				name: "alive shard with fresh heartbeat",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					heartbeatCM("alpha", &freshTime, nil),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					var resp listShardsResponse
					err := json.Unmarshal(w.Body.Bytes(), &resp)
					require.NoError(t, err)
					require.Len(t, resp.Shards, 1)
					require.Equal(t, "alpha", resp.Shards[0].Name)
					require.Equal(t, shardStatusAlive, resp.Shards[0].Status)
					require.NotNil(t, resp.Shards[0].LastSeen)
				},
			},
			{
				name: "dead shard with stale heartbeat",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					heartbeatCM("beta", &staleTime, nil),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					var resp listShardsResponse
					err := json.Unmarshal(w.Body.Bytes(), &resp)
					require.NoError(t, err)
					require.Len(t, resp.Shards, 1)
					require.Equal(t, "beta", resp.Shards[0].Name)
					require.Equal(t, shardStatusDead, resp.Shards[0].Status)
					require.NotNil(t, resp.Shards[0].LastSeen)
				},
			},
			{
				name: "heartbeat with missing observedAt is dead",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					heartbeatCM("gamma", nil, map[string]string{"agentVersion": "v1"}),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					var resp listShardsResponse
					err := json.Unmarshal(w.Body.Bytes(), &resp)
					require.NoError(t, err)
					require.Len(t, resp.Shards, 1)
					require.Equal(t, "gamma", resp.Shards[0].Name)
					require.Equal(t, shardStatusDead, resp.Shards[0].Status)
					require.Nil(t, resp.Shards[0].LastSeen)
				},
			},
			{
				name: "heartbeat with unparseable observedAt is dead",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testKargoNamespace,
							Name:      "agent-delta.status",
							Labels: map[string]string{
								kargoAgentLabelKey: "delta",
							},
						},
						Data: map[string]string{
							agentObservedAtKey: "not-a-timestamp",
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					var resp listShardsResponse
					err := json.Unmarshal(w.Body.Bytes(), &resp)
					require.NoError(t, err)
					require.Len(t, resp.Shards, 1)
					require.Equal(t, "delta", resp.Shards[0].Name)
					require.Equal(t, shardStatusDead, resp.Shards[0].Status)
					require.Nil(t, resp.Shards[0].LastSeen)
				},
			},
			{
				name: "ConfigMaps without the agent label are excluded",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					heartbeatCM("alpha", &freshTime, nil),
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testKargoNamespace,
							Name:      "unrelated-cm",
						},
						Data: map[string]string{
							agentObservedAtKey: freshTime.Format(time.RFC3339),
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					var resp listShardsResponse
					err := json.Unmarshal(w.Body.Bytes(), &resp)
					require.NoError(t, err)
					require.Len(t, resp.Shards, 1)
					require.Equal(t, "alpha", resp.Shards[0].Name)
				},
			},
			{
				name: "ConfigMaps in other namespaces are excluded",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					heartbeatCM("alpha", &freshTime, nil),
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "other-namespace",
							Name:      "agent-stranger.status",
							Labels: map[string]string{
								kargoAgentLabelKey: "stranger",
							},
						},
						Data: map[string]string{
							agentObservedAtKey: freshTime.Format(time.RFC3339),
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					var resp listShardsResponse
					err := json.Unmarshal(w.Body.Bytes(), &resp)
					require.NoError(t, err)
					require.Len(t, resp.Shards, 1)
					require.Equal(t, "alpha", resp.Shards[0].Name)
				},
			},
			{
				name: "shards are sorted by name",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					heartbeatCM("zeta", &freshTime, nil),
					heartbeatCM("alpha", &staleTime, nil),
					heartbeatCM("mu", &freshTime, nil),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					var resp listShardsResponse
					err := json.Unmarshal(w.Body.Bytes(), &resp)
					require.NoError(t, err)
					require.Len(t, resp.Shards, 3)
					require.Equal(t, "alpha", resp.Shards[0].Name)
					require.Equal(t, shardStatusDead, resp.Shards[0].Status)
					require.Equal(t, "mu", resp.Shards[1].Name)
					require.Equal(t, shardStatusAlive, resp.Shards[1].Status)
					require.Equal(t, "zeta", resp.Shards[2].Name)
					require.Equal(t, shardStatusAlive, resp.Shards[2].Status)
				},
			},
		},
	)
}

func Test_deriveShardInfo(t *testing.T) {
	const deadline = 10 * time.Minute
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	testCases := []struct {
		name   string
		data   map[string]string
		assert func(*testing.T, shardInfo)
	}{
		{
			name: "empty data is dead",
			data: map[string]string{},
			assert: func(t *testing.T, info shardInfo) {
				require.Equal(t, shardStatusDead, info.Status)
				require.Nil(t, info.LastSeen)
			},
		},
		{
			name: "missing observedAt is dead",
			data: map[string]string{"agentVersion": "v1"},
			assert: func(t *testing.T, info shardInfo) {
				require.Equal(t, shardStatusDead, info.Status)
				require.Nil(t, info.LastSeen)
			},
		},
		{
			name: "blank observedAt is dead",
			data: map[string]string{agentObservedAtKey: ""},
			assert: func(t *testing.T, info shardInfo) {
				require.Equal(t, shardStatusDead, info.Status)
				require.Nil(t, info.LastSeen)
			},
		},
		{
			name: "unparseable observedAt is dead",
			data: map[string]string{agentObservedAtKey: "garbage"},
			assert: func(t *testing.T, info shardInfo) {
				require.Equal(t, shardStatusDead, info.Status)
				require.Nil(t, info.LastSeen)
			},
		},
		{
			name: "fresh observedAt is alive",
			data: map[string]string{
				agentObservedAtKey: now.Add(-1 * time.Minute).Format(time.RFC3339),
			},
			assert: func(t *testing.T, info shardInfo) {
				require.Equal(t, shardStatusAlive, info.Status)
				require.NotNil(t, info.LastSeen)
			},
		},
		{
			name: "observedAt exactly at deadline is dead",
			data: map[string]string{
				agentObservedAtKey: now.Add(-deadline).Format(time.RFC3339),
			},
			assert: func(t *testing.T, info shardInfo) {
				require.Equal(t, shardStatusDead, info.Status)
				require.NotNil(t, info.LastSeen)
			},
		},
		{
			name: "stale observedAt is dead",
			data: map[string]string{
				agentObservedAtKey: now.Add(-1 * time.Hour).Format(time.RFC3339),
			},
			assert: func(t *testing.T, info shardInfo) {
				require.Equal(t, shardStatusDead, info.Status)
				require.NotNil(t, info.LastSeen)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			info := deriveShardInfo("test-shard", tc.data, now, deadline)
			require.Equal(t, "test-shard", info.Name)
			tc.assert(t, info)
		})
	}
}
