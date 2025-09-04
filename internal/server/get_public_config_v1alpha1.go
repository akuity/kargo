package server

import (
	"context"

	"connectrpc.com/connect"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

func (s *server) GetPublicConfig(
	context.Context,
	*connect.Request[svcv1alpha1.GetPublicConfigRequest],
) (*connect.Response[svcv1alpha1.GetPublicConfigResponse], error) {
	var oidcCfg *svcv1alpha1.OIDCConfig
	if s.cfg.OIDCConfig != nil {
		oidcCfg = &svcv1alpha1.OIDCConfig{
			IssuerUrl:            s.cfg.OIDCConfig.IssuerURL,
			ClientId:             s.cfg.OIDCConfig.ClientID,
			CliClientId:          s.cfg.OIDCConfig.CLIClientID,
			Scopes:               append(s.cfg.OIDCConfig.DefaultScopes, s.cfg.OIDCConfig.AdditionalScopes...),
			AdditionalParameters: s.cfg.OIDCConfig.AdditionalParameters,
		}
	}
	resp := &svcv1alpha1.GetPublicConfigResponse{
		AdminAccountEnabled: s.cfg.AdminConfig != nil,
		OidcConfig:          oidcCfg,
		SkipAuth:            s.cfg.LocalMode,
	}
	return connect.NewResponse(resp), nil
}
