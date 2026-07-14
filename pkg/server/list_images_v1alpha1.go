package server

import (
	"fmt"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// ImageStageMap represents the stages where an image is used
type ImageStageMap struct {
	Stages map[string]int32 `json:"stages"`
} // @name ImageStageMap

// TagMap represents the tags for a repository
type TagMap struct {
	Tags map[string]*ImageStageMap `json:"tags"`
} // @name TagMap

// @id ListImages
// @Summary List container images
// @Description List container images referenced by Freight resources in a
// @Description project's namespace.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Produce json
// @Success 200 {object} map[string]TagMap
// @Router /v1beta1/projects/{project}/images [get]
func (s *server) listImages(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")

	list := &kargoapi.StageList{}
	if err := s.client.List(
		ctx, list, client.InNamespace(project),
	); err != nil {
		_ = c.Error(err)
		return
	}

	images := make(map[string]*TagMap)
	for _, stage := range list.Items {
		for i, freightGroup := range stage.Status.FreightHistory {
			if i > math.MaxInt32 {
				_ = c.Error(fmt.Errorf("index %d exceeds maximum value for int32", i))
				return
			}
			safeI := int32(math.Min(float64(i), math.MaxInt32))
			for _, freight := range freightGroup.Freight {
				for _, image := range freight.Images {
					repo, ok := images[image.RepoURL]
					if !ok || repo == nil {
						repo = &TagMap{}
						images[image.RepoURL] = repo
					}
					if repo.Tags == nil {
						repo.Tags = make(map[string]*ImageStageMap)
					}
					stagemap, ok := repo.Tags[image.Tag]
					if !ok || stagemap == nil {
						repo.Tags[image.Tag] = &ImageStageMap{}
						stagemap = repo.Tags[image.Tag]
					}
					if stagemap.Stages == nil {
						stagemap.Stages = make(map[string]int32)
					}
					if _, ok := stagemap.Stages[stage.Name]; !ok {
						stagemap.Stages[stage.Name] = safeI
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, images)
}
