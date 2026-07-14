package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	libargocd "github.com/akuity/kargo/pkg/argocd"
)

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
