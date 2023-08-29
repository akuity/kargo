package handler

import (
	"context"

	"connectrpc.com/connect"

	"github.com/akuity/kargo/internal/api/config"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type GetPublicConfigV1Alpha1Func func(
	context.Context,
	*connect.Request[svcv1alpha1.GetPublicConfigRequest],
) (*connect.Response[svcv1alpha1.GetPublicConfigResponse], error)

func GetPublicConfigV1Alpha1(
	cfg config.ServerConfig,
) GetPublicConfigV1Alpha1Func {
	var oidcCfg *svcv1alpha1.OIDCConfig
	if cfg.OIDCConfig != nil {
		oidcCfg = &svcv1alpha1.OIDCConfig{
			IssuerUrl: cfg.OIDCConfig.IssuerURL,
			ClientId:  cfg.OIDCConfig.ClientID,
			Scopes:    cfg.OIDCConfig.Scopes,
		}
	}
	resp := &svcv1alpha1.GetPublicConfigResponse{
		AdminAccountEnabled: cfg.AdminConfig != nil,
		OidcConfig:          oidcCfg,
	}
	return func(
		_ context.Context,
		_ *connect.Request[svcv1alpha1.GetPublicConfigRequest],
	) (*connect.Response[svcv1alpha1.GetPublicConfigResponse], error) {
		return connect.NewResponse(resp), nil
	}
}
