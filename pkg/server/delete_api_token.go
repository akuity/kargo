package server

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"

	"github.com/akuity/kargo/pkg/event"
	"github.com/akuity/kargo/pkg/logging"
)

// @id DeleteProjectAPIToken
// @Summary Delete a project-level API token
// @Description Delete a project-level API token from a project's namespace.
// @Tags Rbac, Credentials, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param apitoken path string true "API token name"
// @Success 204 "Deleted successfully"
// @Router /v1beta1/projects/{project}/api-tokens/{apitoken} [delete]
func (s *server) deleteProjectAPIToken(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	name := c.Param("apitoken")

	tokenSecret, err := s.rolesDB.DeleteAPIToken(ctx, false, project, name)
	if err != nil {
		_ = c.Error(err)
		return
	}
	s.recordAPITokenDeleted(ctx, tokenSecret, false)

	c.Status(http.StatusNoContent)
}

// @id DeleteSystemAPIToken
// @Summary Delete a system-level API token
// @Description Delete a system-level API token.
// @Tags Rbac, Credentials, System-Level
// @Security BearerAuth
// @Param apitoken path string true "API token name"
// @Success 204 "Deleted successfully"
// @Router /v1beta1/system/api-tokens/{apitoken} [delete]
func (s *server) deleteSystemAPIToken(c *gin.Context) {
	ctx := c.Request.Context()

	name := c.Param("apitoken")

	tokenSecret, err := s.rolesDB.DeleteAPIToken(ctx, true, "", name)
	if err != nil {
		_ = c.Error(err)
		return
	}
	s.recordAPITokenDeleted(ctx, tokenSecret, true)

	c.Status(http.StatusNoContent)
}

// recordAPITokenDeleted emits an event attributing the token's deletion to
// whoever requested it, completing the credential's audit trail.
func (s *server) recordAPITokenDeleted(
	ctx context.Context,
	tokenSecret *corev1.Secret,
	systemLevel bool,
) {
	if s.sender == nil {
		return
	}
	msg, actor := apiTokenEventMessage(ctx, tokenSecret, "deleted from")
	evt := event.NewAPITokenDeleted(msg, actor, tokenSecret, systemLevel)
	if err := s.sender.Send(ctx, evt); err != nil {
		logging.LoggerFromContext(ctx).Error(err, "error sending API token deleted event")
	}
}
