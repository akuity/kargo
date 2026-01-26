package server

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

func (s *server) ListAPITokens(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListAPITokensRequest],
) (*connect.Response[svcv1alpha1.ListAPITokensResponse], error) {
	systemLevel := req.Msg.SystemLevel
	project := req.Msg.Project
	if err := s.validateSystemLevelOrProject(systemLevel, project); err != nil {
		return nil, err
	}

	if !systemLevel {
		if err := s.validateProjectExists(ctx, project); err != nil {
			return nil, err
		}
	}

	roleName := req.Msg.RoleName

	tokenSecrets, err := s.rolesDB.ListAPITokens(ctx, systemLevel, project, roleName)
	if err != nil {
		if roleName == "" {
			return nil, fmt.Errorf(
				"error listing Kargo API tokens in project %q: %w",
				project, err,
			)
		}
		return nil, fmt.Errorf(
			"error listing tokens for Kargo API role %q in project %q: %w",
			roleName, project, err,
		)
	}
	secretPtrs := make([]*corev1.Secret, len(tokenSecrets))
	for i, tokenSecret := range tokenSecrets {
		secretPtrs[i] = &tokenSecret
	}

	return connect.NewResponse(
		&svcv1alpha1.ListAPITokensResponse{TokenSecrets: secretPtrs},
	), nil
}

// @id ListProjectAPITokens
// @Summary List project-level API tokens
// @Description List project-level API tokens. Returns a Kubernetes SecretList
// @Description resource containing heavily redacted Secrets.
// @Tags Rbac, Credentials, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param role query string false "Role name filter"
// @Produce json
// @Success 200 {object} object "SecretList resource (k8s.io/api/core/v1.SecretList)"
// @Router /v1beta1/projects/{project}/api-tokens [get]
func (s *server) listProjectAPITokens(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	roleName := c.Query("role")

	tokens, err := s.rolesDB.ListAPITokens(ctx, false, project, roleName)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, corev1.SecretList{Items: tokens})
}

// @id ListSystemAPITokens
// @Summary List system-level API tokens
// @Description List system-level API tokens. Returns a Kubernetes SecretList
// @Description resource containing heavily redacted Secrets.
// @Tags Rbac, Credentials, System-Level
// @Security BearerAuth
// @Param role query string false "Role name filter"
// @Produce json
// @Success 200 {object} object "SecretList resource (k8s.io/api/core/v1.SecretList)"
// @Router /v1beta1/system/api-tokens [get]
func (s *server) listSystemAPITokens(c *gin.Context) {
	ctx := c.Request.Context()

	roleName := c.Query("role")

	tokens, err := s.rolesDB.ListAPITokens(ctx, true, "", roleName)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, corev1.SecretList{Items: tokens})
}
