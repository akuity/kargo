package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
)

// @id GetPromotionRequest
// @Summary Retrieve a PromotionRequest
// @Description Retrieve a PromotionRequest resource from a project's namespace.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param promotion-request path string true "PromotionRequest name"
// @Produce json
// @Success 200 {object} kargoapi.PromotionRequest "PromotionRequest custom resource"
// @Router /v1beta1/projects/{project}/promotion-requests/{promotion-request} [get]
func (s *server) getPromotionRequest(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")
	name := c.Param("promotion-request")

	if s.store == nil {
		_ = c.Error(errDatabaseNotConfigured)
		return
	}

	if err := s.authorizeStoreRead(ctx, "get", "promotionrequests", project, name); err != nil {
		_ = c.Error(err)
		return
	}

	promotionRequest, err := s.getPromotionRequestFromStore(ctx, project, name)
	if err != nil {
		_ = c.Error(err)
		return
	}

	if watchMode := c.Query("watch") == trueStr; watchMode {
		if err = s.authorizeStoreRead(ctx, "watch", "promotionrequests", project, name); err != nil {
			_ = c.Error(err)
			return
		}
		servePolledWatch(c, s.storePollInterval, "", polledWatch[kargoapi.PromotionRequest]{
			// A request that has been deleted yields an empty snapshot, which
			// the watch reports as a DELETED event.
			snapshot: func(ctx context.Context) ([]kargoapi.PromotionRequest, error) {
				current, snapshotErr := s.getPromotionRequestFromStore(ctx, project, name)
				if snapshotErr != nil {
					if errors.Is(snapshotErr, database.ErrNotFound) {
						return nil, nil
					}
					return nil, snapshotErr
				}
				return []kargoapi.PromotionRequest{*current}, nil
			},
			name:    func(request kargoapi.PromotionRequest) string { return request.Name },
			version: func(request kargoapi.PromotionRequest) string { return request.ResourceVersion },
		})
		return
	}

	c.JSON(http.StatusOK, promotionRequest)
}

// getPromotionRequestFromStore reads one PromotionRequest. A missing one
// yields an error that satisfies both errors.Is(err, database.ErrNotFound)
// and a 404 response.
func (s *server) getPromotionRequestFromStore(
	ctx context.Context,
	project string,
	name string,
) (*kargoapi.PromotionRequest, error) {
	snapshot, err := s.store.GetPromotionRequest(ctx, database.GetPromotionRequestParams{
		ProjectName: project,
		Name:        name,
	})
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, errors.Join(storeNotFound("promotionrequests", name), err)
		}
		return nil, fmt.Errorf("error getting PromotionRequest %q in Project %q: %w", name, project, err)
	}
	promotionRequest := database.PromotionRequestFromSnapshot(snapshot)
	return &promotionRequest, nil
}
