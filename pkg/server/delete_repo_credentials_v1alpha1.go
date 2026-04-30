package server

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

func (s *server) DeleteRepoCredentials(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteRepoCredentialsRequest],
) (*connect.Response[svcv1alpha1.DeleteRepoCredentialsResponse], error) {
	// Check if secret management is enabled
	if !s.cfg.SecretManagementEnabled {
		return nil, connect.NewError(connect.CodeUnimplemented, errSecretManagementDisabled)
	}

	logger := logging.LoggerFromContext(ctx)
	project := req.Msg.GetProject()
	if project == "" {
		logger.Debug("no project specified, defaulting to shared resources namespace",
			"sharedResourcesNamespace", s.cfg.SharedResourcesNamespace,
		)
		project = s.cfg.SharedResourcesNamespace
	} else {
		if err := s.validateProjectExists(ctx, project); err != nil {
			logger.Error(err, "project does not exist", "project", project)
			return nil, err
		}
	}

	name := req.Msg.GetName()
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	secret := corev1.Secret{}
	if err := s.client.Get(
		ctx,
		types.NamespacedName{
			Namespace: project,
			Name:      name,
		},
		&secret,
	); err != nil {
		return nil, connect.NewError(
			connect.CodeNotFound,
			fmt.Errorf(
				"get secret %s/%s: %w",
				project,
				name,
				err,
			),
		)
	}

	// If this isn't labeled as repository credentials, return not found.
	if _, isCredentials := secret.Labels[kargoapi.LabelKeyCredentialType]; !isCredentials {
		return nil, connect.NewError(
			connect.CodeNotFound,
			fmt.Errorf(
				"secret %s/%s exists, but is not labeled with %s",
				secret.Namespace,
				secret.Name,
				kargoapi.LabelKeyCredentialType,
			),
		)
	}

	if err := s.client.Delete(ctx, &secret); err != nil {
		return nil, connect.NewError(
			connect.CodeInternal,
			fmt.Errorf(
				"delete secret %s/%s: %w",
				secret.Namespace,
				secret.Name,
				err,
			),
		)
	}

	return connect.NewResponse(&svcv1alpha1.DeleteRepoCredentialsResponse{}), nil
}

// @id DeleteProjectRepoCredentials
// @Summary Delete project-level repository credentials
// @Description Delete repository credentials from a project's namespace.
// @Tags Credentials, Repo Credentials, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param repo-credentials path string true "Credentials name"
// @Success 204 "Deleted successfully"
// @Router /v1beta1/projects/{project}/repo-credentials/{repo-credentials} [delete]
func (s *server) deleteProjectRepoCredentials(c *gin.Context) {
	if !s.requireSecretManagement(c) {
		return
	}

	ctx := c.Request.Context()

	project := c.Param("project")
	name := c.Param("repo-credentials")

	secret := corev1.Secret{}
	if err := s.client.Get(
		ctx,
		types.NamespacedName{
			Namespace: project,
			Name:      name,
		},
		&secret,
	); err != nil {
		_ = c.Error(err)
		return
	}

	if err := validateRepoCredentialSecret(&secret); err != nil {
		_ = c.Error(err)
		return
	}

	if err := s.client.Delete(ctx, &secret); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @id DeleteSharedRepoCredentials
// @Summary Delete shared repository credentials
// @Description Delete shared repository credentials.
// @Tags Credentials, Repo Credentials, Shared
// @Security BearerAuth
// @Param repo-credentials path string true "Credentials name"
// @Success 204 "Deleted successfully"
// @Router /v1beta1/shared/repo-credentials/{repo-credentials} [delete]
func (s *server) deleteSharedRepoCredentials(c *gin.Context) {
	if !s.requireSecretManagement(c) {
		return
	}

	ctx := c.Request.Context()
	name := c.Param("repo-credentials")

	secret := corev1.Secret{}
	if err := s.client.Get(
		ctx,
		types.NamespacedName{Namespace: s.cfg.SharedResourcesNamespace, Name: name},
		&secret,
	); err != nil {
		_ = c.Error(err)
		return
	}

	if err := validateRepoCredentialSecret(&secret); err != nil {
		_ = c.Error(err)
		return
	}

	if err := s.client.Delete(ctx, &secret); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
