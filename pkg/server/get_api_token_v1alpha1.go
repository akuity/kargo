package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	sigyaml "sigs.k8s.io/yaml"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

func (s *server) GetAPIToken(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetAPITokenRequest],
) (*connect.Response[svcv1alpha1.GetAPITokenResponse], error) {
	systemLevel := req.Msg.SystemLevel
	project := req.Msg.Project
	if err := s.validateSystemLevelOrProject(systemLevel, project); err != nil {
		return nil, err
	}

	name := req.Msg.GetName()
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	if !systemLevel {
		if err := s.validateProjectExists(ctx, project); err != nil {
			return nil, err
		}
	}

	tokenSecret, err := s.rolesDB.GetAPIToken(
		ctx, systemLevel, project, name,
	)
	if err != nil {
		return nil, fmt.Errorf("error getting API token Secret: %w", err)
	}

	var rawBytes []byte
	switch req.Msg.Format {
	case svcv1alpha1.RawFormat_RAW_FORMAT_JSON:
		if rawBytes, err = json.Marshal(tokenSecret); err != nil {
			return nil,
				fmt.Errorf("error marshaling API token Secret to raw JSON: %w", err)
		}
	case svcv1alpha1.RawFormat_RAW_FORMAT_YAML:
		if rawBytes, err = sigyaml.Marshal(tokenSecret); err != nil {
			return nil,
				fmt.Errorf("error marshaling API token Secret to raw YAML: %w", err)
		}
	default:
		return connect.NewResponse(&svcv1alpha1.GetAPITokenResponse{
			Result: &svcv1alpha1.GetAPITokenResponse_TokenSecret{
				TokenSecret: tokenSecret,
			},
		}), nil
	}

	return connect.NewResponse(&svcv1alpha1.GetAPITokenResponse{
		Result: &svcv1alpha1.GetAPITokenResponse_Raw{
			Raw: rawBytes,
		},
	}), nil
}

// @id GetProjectAPIToken
// @Summary Retrieve a project-level API token
// @Description Retrieve a project-level API token by name. Returns a heavily
// @Description redacted Kubernetes Secret resource.
// @Tags Rbac, Credentials, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param apitoken path string true "API token name"
// @Produce json
// @Success 200 {object} object "Secret resource (k8s.io/api/core/v1.Secret)"
// @Router /v1beta1/projects/{project}/api-tokens/{apitoken} [get]
func (s *server) getProjectAPIToken(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	name := c.Param("apitoken")

	tokenSecret, err := s.rolesDB.GetAPIToken(ctx, false, project, name)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, tokenSecret)
}

// @id GetSystemAPIToken
// @Summary Retrieve a system-level API token
// @Description Retrieve a system-level API token by name. Returns a heavily
// @Description redacted Kubernetes Secret resource.
// @Tags Rbac, Credentials, System-Level
// @Security BearerAuth
// @Param apitoken path string true "API token name"
// @Produce json
// @Success 200 {object} object "Secret resource (k8s.io/api/core/v1.Secret)"
// @Router /v1beta1/system/api-tokens/{apitoken} [get]
func (s *server) getSystemAPIToken(c *gin.Context) {
	ctx := c.Request.Context()

	name := c.Param("apitoken")

	tokenSecret, err := s.rolesDB.GetAPIToken(ctx, true, "", name)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, tokenSecret)
}
