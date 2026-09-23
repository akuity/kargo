package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"

	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/event"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/server/user"
)

// createAPITokenRequest is the request body for creating an API token.
type createAPITokenRequest struct {
	Name string `json:"name"`
} // @name CreateAPITokenRequest

// @id CreateProjectAPIToken
// @Summary Create a project-level API token
// @Description Create a project-level API token associated with a Kargo Role
// @Description virtual resource. Returns a Kubernetes Secret resource
// @Description representing the token. Store it securely. The token is not
// @Description retrievable via the Kargo API after creation except in a
// @Description redacted form.
// @Tags Rbac, Credentials, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param role path string true "Role name"
// @Param body body createAPITokenRequest true "Token"
// @Success 201 {object} corev1.Secret "Secret resource (k8s.io/api/core/v1.Secret)"
// @Router /v1beta1/projects/{project}/roles/{role}/api-tokens [post]
func (s *server) createProjectAPIToken(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")
	role := c.Param("role")

	var req createAPITokenRequest
	if !bindJSONOrError(c, &req) {
		return
	}
	if req.Name == "" {
		_ = c.Error(libhttp.Error(
			errors.New("name should not be empty"),
			http.StatusBadRequest,
		))
		return
	}

	var tokenSecret *corev1.Secret
	tokenSecret, err := s.rolesDB.CreateAPIToken(
		ctx, false, project, role, req.Name,
	)
	if err != nil {
		_ = c.Error(err)
		return
	}
	s.recordAPITokenCreated(ctx, tokenSecret)

	c.JSON(http.StatusCreated, tokenSecret)
}

// @id CreateSystemAPIToken
// @Summary Create a system-level API token
// @Description Create a system-level API token associated with a system-level
// @Description Kargo Role virtual resource. Returns a Kubernetes Secret
// @Description resource representing the token. Store it securely. The token
// @Description is not retrievable via the Kargo API after creation except in
// @Description a redacted form.
// @Tags Rbac, Credentials, System-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param role path string true "Role name"
// @Param body body createAPITokenRequest true "Token"
// @Success 201 {object} corev1.Secret "Secret resource (k8s.io/api/core/v1.Secret)"
// @Router /v1beta1/system/roles/{role}/api-tokens [post]
func (s *server) createSystemAPIToken(c *gin.Context) {
	ctx := c.Request.Context()
	role := c.Param("role")

	var req createAPITokenRequest
	if !bindJSONOrError(c, &req) {
		return
	}
	if req.Name == "" {
		_ = c.Error(libhttp.Error(
			errors.New("name should not be empty"),
			http.StatusBadRequest,
		))
		return
	}

	tokenSecret, err := s.rolesDB.CreateAPIToken(
		ctx, true, "", role, req.Name,
	)
	if err != nil {
		_ = c.Error(err)
		return
	}
	s.recordAPITokenCreated(ctx, tokenSecret)

	c.JSON(http.StatusCreated, tokenSecret)
}

// recordAPITokenCreated emits an event attributing the new token to whoever
// minted it, so that credential creation leaves an audit trail.
func (s *server) recordAPITokenCreated(ctx context.Context, tokenSecret *corev1.Secret) {
	if s.sender == nil {
		return
	}
	var actor string
	msg := fmt.Sprintf(
		"API token %q created for Role %q",
		tokenSecret.Name, tokenSecret.Annotations[corev1.ServiceAccountNameKey],
	)
	if u, ok := user.InfoFromContext(ctx); ok {
		actor = api.FormatEventUserActor(u)
		msg += fmt.Sprintf(" by %q", actor)
	}
	if err := s.sender.Send(ctx, event.NewAPITokenCreated(msg, actor, tokenSecret)); err != nil {
		logging.LoggerFromContext(ctx).Error(err, "error sending API token created event")
	}
}
