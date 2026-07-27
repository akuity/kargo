package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/akuity/kargo/pkg/x/version"
)

// CLIVersionHeader is the HTTP header used to convey the CLI version in
// requests to the API server.
//
// TODO(krancour): Move this closer to CLI-version-checking middleware once it
// exists.
const CLIVersionHeader = "X-Kargo-CLI-Version"

type versionInfo version.Version // @name VersionInfo

// @id GetVersionInfo
// @Summary Retrieve API Server version information
// @Description Retrieve API Server version information.
// @Tags System
// @Security BearerAuth
// @Produce json
// @Success 200 {object} versionInfo
// @Router /v1beta1/system/server-version [get]
func (s *server) getVersionInfo(c *gin.Context) {
	c.JSON(http.StatusOK, versionInfo(version.GetVersion()))
}
