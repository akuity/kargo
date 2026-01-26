package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libCreds "github.com/akuity/kargo/pkg/credentials"
	libhttp "github.com/akuity/kargo/pkg/http"
)

func (s *server) UpdateRepoCredentials(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.UpdateRepoCredentialsRequest],
) (*connect.Response[svcv1alpha1.UpdateRepoCredentialsResponse], error) {
	// Check if secret management is enabled
	if !s.cfg.SecretManagementEnabled {
		return nil, connect.NewError(connect.CodeUnimplemented, errSecretManagementDisabled)
	}

	if err := validateFieldNotEmpty("project", req.Msg.Project); err != nil {
		return nil, err
	}

	project := req.Msg.GetProject()
	if project != "" {
		if err := s.validateProjectExists(ctx, project); err != nil {
			return nil, err
		}
	}
	namespace := project
	if namespace == "" {
		namespace = s.cfg.SharedResourcesNamespace
	}

	secret := corev1.Secret{}
	if err := s.client.Get(
		ctx,
		types.NamespacedName{
			Namespace: namespace,
			Name:      req.Msg.Name,
		},
		&secret,
	); err != nil {
		return nil, fmt.Errorf("get secret: %w", err)
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

	applyUpdateRepoCredentialsRequestToK8sSecret(
		&secret,
		updateRepoCredentialsRequest{
			Description:    req.Msg.Description,
			Type:           req.Msg.GetType(),
			RepoURL:        req.Msg.GetRepoUrl(),
			RepoURLIsRegex: req.Msg.GetRepoUrlIsRegex(),
			Username:       req.Msg.GetUsername(),
			Password:       req.Msg.GetPassword(),
		},
	)
	if err := s.client.Update(ctx, &secret); err != nil {
		return nil, fmt.Errorf("update secret: %w", err)
	}

	return connect.NewResponse(
		&svcv1alpha1.UpdateRepoCredentialsResponse{
			Credentials: sanitizeCredentialSecret(secret),
		},
	), nil
}

// updateRepoCredentialsRequest is the request body for replacing repository credentials.
// All required fields must be provided as the entire secret data is replaced.
type updateRepoCredentialsRequest struct {
	Description    string `json:"description,omitempty"`
	Type           string `json:"type"`
	RepoURL        string `json:"repoUrl"`
	RepoURLIsRegex bool   `json:"repoUrlIsRegex,omitempty"`
	Username       string `json:"username"`
	Password       string `json:"password"`
} // @name UpdateRepoCredentialsRequest

// @id UpdateProjectRepoCredentials
// @Summary Replace project-level repository credentials
// @Description Replace project-level repository credentials. All fields are replaced.
// @Description Returns a heavily redacted Kubernetes Secret resource.
// @Tags Credentials, Repo Credentials, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param repo-credentials path string true "Repo credentials name"
// @Param body body updateRepoCredentialsRequest true "Credentials"
// @Success 200 {object} object "Secret resource (k8s.io/api/core/v1.Secret)"
// @Router /v1beta1/projects/{project}/repo-credentials/{repo-credentials} [put]
func (s *server) updateProjectRepoCredentials(c *gin.Context) {
	if !s.requireSecretManagement(c) {
		return
	}

	ctx := c.Request.Context()
	project := c.Param("project")
	name := c.Param("repo-credentials")

	var req updateRepoCredentialsRequest
	if !bindJSONOrError(c, &req) {
		return
	}

	if err := validateUpdateRepoCredentialsRequest(req); err != nil {
		_ = c.Error(libhttp.Error(err, http.StatusBadRequest))
		return
	}

	secret := corev1.Secret{}
	if err := s.client.Get(
		ctx,
		types.NamespacedName{Namespace: project, Name: name},
		&secret,
	); err != nil {
		_ = c.Error(err)
		return
	}

	if err := validateRepoCredentialSecret(&secret); err != nil {
		_ = c.Error(err)
		return
	}

	applyUpdateRepoCredentialsRequestToK8sSecret(&secret, req)

	if err := s.client.Update(ctx, &secret); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, sanitizeCredentialSecret(secret))
}

// @id UpdateSharedRepoCredentials
// @Summary Replace shared repository credentials
// @Description Replace shared repository credentials. All fields are replaced.
// @Description Returns a heavily redacted Kubernetes Secret resource.
// @Tags Credentials, Repo Credentials, Shared
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param repo-credentials path string true "Repo credentials name"
// @Param body body updateRepoCredentialsRequest true "Credentials"
// @Success 200 {object} object "Secret resource (k8s.io/api/core/v1.Secret)"
// @Router /v1beta1/shared/repo-credentials/{repo-credentials} [put]
func (s *server) updateSharedRepoCredentials(c *gin.Context) {
	if !s.requireSecretManagement(c) {
		return
	}

	ctx := c.Request.Context()
	name := c.Param("repo-credentials")

	var req updateRepoCredentialsRequest
	if !bindJSONOrError(c, &req) {
		return
	}

	if err := validateUpdateRepoCredentialsRequest(req); err != nil {
		_ = c.Error(libhttp.Error(err, http.StatusBadRequest))
		return
	}

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

	applyUpdateRepoCredentialsRequestToK8sSecret(&secret, req)

	if err := s.client.Update(ctx, &secret); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, sanitizeCredentialSecret(secret))
}

func validateUpdateRepoCredentialsRequest(req updateRepoCredentialsRequest) error {
	if req.Type == "" {
		return errors.New("type should not be empty")
	}
	switch req.Type {
	case kargoapi.LabelValueCredentialTypeGit,
		kargoapi.LabelValueCredentialTypeHelm,
		kargoapi.LabelValueCredentialTypeImage:
	default:
		return errors.New("type should be one of git, helm, or image")
	}
	if req.RepoURL == "" {
		return errors.New("repoUrl should not be empty")
	}
	if req.Username == "" {
		return errors.New("username should not be empty")
	}
	if req.Password == "" {
		return errors.New("password should not be empty")
	}
	return nil
}

func applyUpdateRepoCredentialsRequestToK8sSecret(
	secret *corev1.Secret,
	req updateRepoCredentialsRequest,
) {
	// Set or clear description
	if req.Description != "" {
		if secret.Annotations == nil {
			secret.Annotations = make(map[string]string, 1)
		}
		secret.Annotations[kargoapi.AnnotationKeyDescription] = req.Description
	} else {
		delete(secret.Annotations, kargoapi.AnnotationKeyDescription)
	}

	// Replace all credential data
	secret.Labels[kargoapi.LabelKeyCredentialType] = req.Type

	// Replace the entire data map
	secret.Data = map[string][]byte{
		libCreds.FieldRepoURL:  []byte(req.RepoURL),
		libCreds.FieldUsername: []byte(req.Username),
		libCreds.FieldPassword: []byte(req.Password),
	}
	if req.RepoURLIsRegex {
		secret.Data[libCreds.FieldRepoURLIsRegex] = []byte("true")
	}
}
