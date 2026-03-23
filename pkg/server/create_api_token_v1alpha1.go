package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
)

func (s *server) CreateAPIToken(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.CreateAPITokenRequest],
) (*connect.Response[svcv1alpha1.CreateAPITokenResponse], error) {
	systemLevel := req.Msg.SystemLevel
	project := req.Msg.Project
	if err := s.validateSystemLevelOrProject(systemLevel, project); err != nil {
		return nil, err
	}

	roleName := req.Msg.RoleName
	if err := validateFieldNotEmpty("role_name", roleName); err != nil {
		return nil, err
	}

	name := req.Msg.Name
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	if !systemLevel {
		if err := s.validateProjectExists(ctx, project); err != nil {
			return nil, err
		}
	}

	tokenSecret, err := s.rolesDB.CreateAPIToken(
		ctx, systemLevel, project, roleName, name,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating new token Secret: %w", err)
	}

	return connect.NewResponse(
		&svcv1alpha1.CreateAPITokenResponse{TokenSecret: tokenSecret},
	), nil
}

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
// @Success 201 {object} object "Secret resource (k8s.io/api/core/v1.Secret)"
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

	tokenSecret, err := s.rolesDB.CreateAPIToken(
		ctx, false, project, role, req.Name,
	)
	if err != nil {
		_ = c.Error(err)
		return
	}

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
// @Success 201 {object} object "Secret resource (k8s.io/api/core/v1.Secret)"
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

	c.JSON(http.StatusCreated, tokenSecret)
}
