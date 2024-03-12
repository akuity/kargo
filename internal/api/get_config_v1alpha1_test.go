package api

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/internal/api/config"
	"github.com/akuity/kargo/internal/api/user"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func TestGetConfig(t *testing.T) {
	testCases := map[string]struct {
		req        *svcv1alpha1.GetConfigRequest
		cfg        config.ServerConfig
		assertions func(res *svcv1alpha1.GetConfigResponse)
	}{
		"get config": {
			req: &svcv1alpha1.GetConfigRequest{},
			cfg: config.ServerConfig{
				ArgoCDConfig: config.ArgoCDConfig{
					URLs: map[string]string{
						"": "https://argocd.example.com",
					},
				},
			},
			assertions: func(res *svcv1alpha1.GetConfigResponse) {
				require.Equal(t, "argocd", res.ArgocdShards[""].Namespace)
				require.Equal(t, "https://argocd.example.com", res.ArgocdShards[""].Url)
			},
		},
	}
	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Simulate an admin user to prevent any authz issues with the authorizing
			// client.
			ctx := user.ContextWithInfo(
				context.Background(),
				user.Info{
					IsAdmin: true,
				},
			)

			svr := &server{
				cfg: tc.cfg,
			}
			res, err := svr.GetConfig(ctx, connect.NewRequest(tc.req))
			require.NoError(t, err)
			if tc.assertions != nil {
				tc.assertions(res.Msg)
			}
		})
	}
}
