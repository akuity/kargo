package server

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	libargocd "github.com/akuity/kargo/pkg/argocd"
)

func (s *server) GetConfig(
	context.Context,
	*connect.Request[svcv1alpha1.GetConfigRequest],
) (*connect.Response[svcv1alpha1.GetConfigResponse], error) {
	resp := svcv1alpha1.GetConfigResponse{
		ArgocdShards:                  make(map[string]*svcv1alpha1.ArgoCDShard),
		SecretManagementEnabled:       s.cfg.SecretManagementEnabled,
		SystemResourcesNamespace:      s.cfg.SystemResourcesNamespace,
		HasAnalysisRunLogsUrlTemplate: s.cfg.AnalysisRunLogURLTemplate != "",
	}
	for shardName, url := range s.cfg.ArgoCDConfig.URLs {
		resp.ArgocdShards[shardName] = &svcv1alpha1.ArgoCDShard{
			Url: url,
			// TODO: currently, all shards must use the same namespace
			Namespace: libargocd.Namespace(),
		}
	}
	return connect.NewResponse(&resp), nil
}

// getConfigResponse represents the server configuration response
type getConfigResponse struct {
	ArgocdShards                  map[string]*argoCDShard `json:"argocdShards"`
	SecretManagementEnabled       bool                    `json:"secretManagementEnabled"`
	SystemResourcesNamespace      string                  `json:"systemResourcesNamespace"`
	SharedResourcesNamespace      string                  `json:"sharedResourcesNamespace"`
	KargoNamespace                string                  `json:"kargoNamespace"`
	HasAnalysisRunLogsUrlTemplate bool                    `json:"hasAnalysisRunLogsUrlTemplate"`
} // @name GetConfigResponse

// ArgoCDShard represents Argo CD shard configuration
type argoCDShard struct {
	URL       string `json:"url"`
	Namespace string `json:"namespace"`
} // @name ArgoCDShard

// @id GetConfig
// @Summary Retrieve server configuration
// @Description Retrieve information a client may need to know about how the
// @Description Kargo API server is configured.
// @Tags System, Config
// @Security BearerAuth
// @Produce json
// @Success 200 {object} getConfigResponse
// @Router /v1beta1/system/server-config [get]
func (s *server) getConfig(c *gin.Context) {
	resp := getConfigResponse{
		ArgocdShards:                  make(map[string]*argoCDShard),
		SecretManagementEnabled:       s.cfg.SecretManagementEnabled,
		SystemResourcesNamespace:      s.cfg.SystemResourcesNamespace,
		SharedResourcesNamespace:      s.cfg.SharedResourcesNamespace,
		KargoNamespace:                s.cfg.KargoNamespace,
		HasAnalysisRunLogsUrlTemplate: s.cfg.AnalysisRunLogURLTemplate != "",
	}
	for shardName, url := range s.cfg.ArgoCDConfig.URLs {
		resp.ArgocdShards[shardName] = &argoCDShard{
			URL:       url,
			Namespace: libargocd.Namespace(),
		}
	}
	c.JSON(http.StatusOK, resp)
}
