package server

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	libargocd "github.com/akuity/kargo/pkg/server/argocd"
	"github.com/akuity/kargo/pkg/server/config"
)

func TestGetConfig(t *testing.T) {
	testCases := map[string]struct {
		req           *svcv1alpha1.GetConfigRequest
		cfg           config.ServerConfig
		setupURLStore func() libargocd.URLStore
		assertions    func(res *svcv1alpha1.GetConfigResponse)
	}{
		"get config with static shards": {
			req: &svcv1alpha1.GetConfigRequest{},
			cfg: config.ServerConfig{},
			setupURLStore: func() libargocd.URLStore {
				store := libargocd.NewURLStore()
				store.SetStaticShards(map[string]string{
					"": "https://argocd.example.com",
				}, "argocd")
				return store
			},
			assertions: func(res *svcv1alpha1.GetConfigResponse) {
				require.Equal(t, "argocd", res.ArgocdShards[""].Namespace)
				require.Equal(t, "https://argocd.example.com", res.ArgocdShards[""].Url)
			},
		},
		"get config with dynamic shards": {
			req: &svcv1alpha1.GetConfigRequest{},
			cfg: config.ServerConfig{},
			setupURLStore: func() libargocd.URLStore {
				store := libargocd.NewURLStore()
				store.SetStaticShards(nil, "argocd")
				store.UpdateDynamicShard("production", "https://argocd-prod.example.com")
				return store
			},
			assertions: func(res *svcv1alpha1.GetConfigResponse) {
				require.Len(t, res.ArgocdShards, 1)
				require.Equal(t, "https://argocd-prod.example.com", res.ArgocdShards["production"].Url)
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			svr := &server{
				cfg:            tc.cfg,
				argoCDURLStore: tc.setupURLStore(),
			}
			res, err := svr.GetConfig(context.Background(), connect.NewRequest(tc.req))
			require.NoError(t, err)
			if tc.assertions != nil {
				tc.assertions(res.Msg)
			}
		})
	}
}
