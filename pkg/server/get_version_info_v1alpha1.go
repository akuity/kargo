package server

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/x/version"
)

// CLIVersionHeader is the HTTP header used to convey the CLI version in
// requests to the API server.
//
// TODO(krancour): Move this closer to CLI-version-checking middleware once it
// exists.
const CLIVersionHeader = "X-Kargo-CLI-Version"

func (s *server) GetVersionInfo(
	context.Context,
	*connect.Request[svcv1alpha1.GetVersionInfoRequest],
) (*connect.Response[svcv1alpha1.GetVersionInfoResponse], error) {
	return connect.NewResponse(
		&svcv1alpha1.GetVersionInfoResponse{
			VersionInfo: api.ToVersionProto(version.GetVersion()),
		},
	), nil
}

// @id GetVersionInfo
// @Summary Retrieve API Server version information
// @Description Retrieve API Server version information.
// @Tags System
// @Security BearerAuth
// @Produce json
// @Success 200 {object} version.Version
// @Router /v1beta1/system/server-version [get]
func (s *server) getVersionInfo(c *gin.Context) {
	c.JSON(http.StatusOK, version.GetVersion())
}
