package handler

import (
	"context"

	"github.com/bufbuild/connect-go"

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
	return func(context.Context, *connect.Request[svcv1alpha1.GetPublicConfigRequest],
	) (*connect.Response[svcv1alpha1.GetPublicConfigResponse], error) {
		var oidcCfg *svcv1alpha1.OIDCConfig
		if cfg.OIDCConfig != nil {
			oidcCfg = &svcv1alpha1.OIDCConfig{
				IssuerUrl: cfg.OIDCConfig.IssuerURL,
				ClientId:  cfg.OIDCConfig.ClientID,
				Scopes:    cfg.OIDCConfig.Scopes,
			}
		}
		return connect.NewResponse(&svcv1alpha1.GetPublicConfigResponse{
			AdminAccountEnabled: cfg.AdminConfig != nil,
			OidcConfig:          oidcCfg,
		}), nil
	}
}
