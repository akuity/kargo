package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/akuity/kargo/pkg/releases"
)

type bestReleasesResponse struct {
	Releases []releases.Release `json:"releases"`
} // @name BestReleasesResponse

// @id GetBestReleases
// @Summary List latest patch release for each minor version of Kargo
// @Description Returns the latest patch release for each minor version of Kargo.
// @Tags Utility
// @Security BearerAuth
// @Produce json
// @Success 200 {object} bestReleasesResponse
// @Router /v1beta1/kargo-releases/best [get]
func (s *server) getBestReleases(c *gin.Context) {
	c.JSON(http.StatusOK, bestReleasesResponse{
		Releases: s.releaseSvc.GetBestReleases(),
	})
}
