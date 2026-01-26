package server

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

func (s *server) GetPublicConfig(
	context.Context,
	*connect.Request[svcv1alpha1.GetPublicConfigRequest],
) (*connect.Response[svcv1alpha1.GetPublicConfigResponse], error) {
	var oidcCfg *svcv1alpha1.OIDCConfig
	if s.cfg.OIDCConfig != nil {
		oidcCfg = &svcv1alpha1.OIDCConfig{
			IssuerUrl:   s.cfg.OIDCConfig.IssuerURL,
			ClientId:    s.cfg.OIDCConfig.ClientID,
			CliClientId: s.cfg.OIDCConfig.CLIClientID,
			Scopes:      append(s.cfg.OIDCConfig.DefaultScopes, s.cfg.OIDCConfig.AdditionalScopes...),
		}
	}
	resp := &svcv1alpha1.GetPublicConfigResponse{
		AdminAccountEnabled: s.cfg.AdminConfig != nil,
		OidcConfig:          oidcCfg,
		SkipAuth:            s.cfg.LocalMode,
	}
	return connect.NewResponse(resp), nil
}

type publicConfig struct {
	AdminAccountEnabled bool        `json:"adminAccountEnabled"`
	OIDCConfig          *oidcConfig `json:"oidcConfig,omitempty"`
	SkipAuth            bool        `json:"skipAuth"`
} // @name PublicConfig

type oidcConfig struct {
	IssuerURL   string   `json:"issuerUrl"`
	ClientID    string   `json:"clientId"`
	CLIClientID string   `json:"cliClientId,omitempty"`
	Scopes      []string `json:"scopes,omitempty"`
} // @name OIDCConfig

// @id GetPublicConfig
// @Summary Retrieve public server configuration
// @Description Retrieve information a client may need to know about how the
// @Description Kargo API server is configured in order to proceed with
// @Description authentication.
// @Tags System, Config
// @Produce json
// @Success 200 {object} publicConfig
// @Router /v1beta1/system/public-server-config [get]
func (s *server) getPublicConfig(c *gin.Context) {
	var oidcCfg *oidcConfig
	if s.cfg.OIDCConfig != nil {
		oidcCfg = &oidcConfig{
			IssuerURL:   s.cfg.OIDCConfig.IssuerURL,
			ClientID:    s.cfg.OIDCConfig.ClientID,
			CLIClientID: s.cfg.OIDCConfig.CLIClientID,
			Scopes:      append(s.cfg.OIDCConfig.DefaultScopes, s.cfg.OIDCConfig.AdditionalScopes...),
		}
	}
	resp := publicConfig{
		AdminAccountEnabled: s.cfg.AdminConfig != nil,
		OIDCConfig:          oidcCfg,
		SkipAuth:            s.cfg.LocalMode,
	}
	c.JSON(http.StatusOK, resp)
}
