package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libCreds "github.com/akuity/kargo/pkg/credentials"
)

// patchRepoCredentialsRequest is the request body for patching repository credentials.
// Only non-empty fields are applied. For description, nil means no change, empty string means clear.
type patchRepoCredentialsRequest struct {
	Description    *string `json:"description,omitempty"`
	Type           string  `json:"type,omitempty"`
	RepoURL        string  `json:"repoUrl,omitempty"`
	RepoURLIsRegex bool    `json:"repoUrlIsRegex,omitempty"`
	Username       string  `json:"username,omitempty"`
	Password       string  `json:"password,omitempty"`
} // @name PatchRepoCredentialsRequest

// @id PatchProjectRepoCredentials
// @Summary Patch project-level repository credentials
// @Description Patch project-level repository credentials. Only provided fields
// @Description are updated. Returns a heavily redacted Kubernetes Secret resource.
// @Tags Credentials, Repo Credentials, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param repo-credentials path string true "Repo credentials name"
// @Param body body patchRepoCredentialsRequest true "Credentials"
// @Success 200 {object} object "Secret resource (k8s.io/api/core/v1.Secret)"
// @Router /v1beta1/projects/{project}/repo-credentials/{repo-credentials} [patch]
func (s *server) patchProjectRepoCredentials(c *gin.Context) {
	if !s.requireSecretManagement(c) {
		return
	}

	ctx := c.Request.Context()
	project := c.Param("project")
	name := c.Param("repo-credentials")

	var req patchRepoCredentialsRequest
	if !bindJSONOrError(c, &req) {
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

	applyPatchRepoCredentialsRequestToK8sSecret(&secret, req)

	if err := s.client.Update(ctx, &secret); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, sanitizeCredentialSecret(secret))
}

// @id PatchSharedRepoCredentials
// @Summary Patch shared repository credentials
// @Description Patch shared repository credentials. Only provided fields
// @Description are updated. Returns a heavily redacted Kubernetes Secret resource.
// @Tags Credentials, Repo Credentials, Shared
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param repo-credentials path string true "Repo credentials name"
// @Param body body patchRepoCredentialsRequest true "Credentials"
// @Success 200 {object} object "Secret resource (k8s.io/api/core/v1.Secret)"
// @Router /v1beta1/shared/repo-credentials/{repo-credentials} [patch]
func (s *server) patchSharedRepoCredentials(c *gin.Context) {
	if !s.requireSecretManagement(c) {
		return
	}

	ctx := c.Request.Context()
	name := c.Param("repo-credentials")

	var req patchRepoCredentialsRequest
	if !bindJSONOrError(c, &req) {
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

	applyPatchRepoCredentialsRequestToK8sSecret(&secret, req)
	if err := s.client.Update(ctx, &secret); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, sanitizeCredentialSecret(secret))
}

func applyPatchRepoCredentialsRequestToK8sSecret(
	secret *corev1.Secret,
	req patchRepoCredentialsRequest,
) {
	// Update description if provided (nil means don't change, empty string means clear)
	if req.Description != nil {
		if *req.Description != "" {
			if secret.Annotations == nil {
				secret.Annotations = make(map[string]string, 1)
			}
			secret.Annotations[kargoapi.AnnotationKeyDescription] = *req.Description
		} else {
			delete(secret.Annotations, kargoapi.AnnotationKeyDescription)
		}
	}

	if req.Type != "" {
		secret.Labels[kargoapi.LabelKeyCredentialType] = req.Type
	}
	if req.RepoURL != "" {
		secret.Data[libCreds.FieldRepoURL] = []byte(req.RepoURL)
		if req.RepoURLIsRegex {
			secret.Data[libCreds.FieldRepoURLIsRegex] = []byte("true")
		} else {
			delete(secret.Data, libCreds.FieldRepoURLIsRegex)
		}
	}
	if req.Username != "" {
		secret.Data["username"] = []byte(req.Username)
	}
	if req.Password != "" {
		secret.Data["password"] = []byte(req.Password)
	}
}
