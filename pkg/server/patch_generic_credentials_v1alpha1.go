package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
)

type patchGenericCredentialsRequest struct {
	Description *string           `json:"description,omitempty"`
	Data        map[string]string `json:"data,omitempty"`
	RemoveKeys  []string          `json:"removeKeys,omitempty"`
} // @name PatchGenericCredentialsRequest

// @id PatchProjectGenericCredentials
// @Summary Patch project-level generic credentials
// @Description Patch project-level generic credentials. Merges provided data
// @Description with existing data. Use removeKeys to delete specific keys.
// @Description Returns a heavily redacted Kubernetes Secret resource.
// @Tags Credentials, Generic Credentials, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param generic-credentials path string true "Generic credentials name"
// @Param body body patchGenericCredentialsRequest true "GenericCredentials patch"
// @Success 200 {object} object "Secret resource (k8s.io/api/core/v1.Secret)"
// @Router /v1beta1/projects/{project}/generic-credentials/{generic-credentials} [patch]
func (s *server) patchProjectGenericCredentials(c *gin.Context) {
	if !s.requireSecretManagement(c) {
		return
	}

	ctx := c.Request.Context()

	project := c.Param("project")
	name := c.Param("generic-credentials")

	var req patchGenericCredentialsRequest
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

	if err := validateGenericCredentialSecret(&secret); err != nil {
		_ = c.Error(err)
		return
	}

	applyGenericCredentialsPatchToK8sSecret(&secret, req)

	if err := validateSecretNotEmpty(&secret); err != nil {
		_ = c.Error(libhttp.Error(err, http.StatusBadRequest))
		return
	}

	if err := s.client.Update(ctx, &secret); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, sanitizeGenericCredentials(secret))
}

// @id PatchSystemGenericCredentials
// @Summary Patch system-level generic credentials
// @Description Patch system-level generic credentials. Merges provided data
// @Description with existing data. Use removeKeys to delete specific keys.
// @Description Returns a heavily redacted Kubernetes Secret resource.
// @Tags Credentials, Generic Credentials, System-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param generic-credentials path string true "Generic credentials name"
// @Param body body patchGenericCredentialsRequest true "GenericCredentials patch"
// @Success 200 {object} object "Secret resource (k8s.io/api/core/v1.Secret)"
// @Router /v1beta1/system/generic-credentials/{generic-credentials} [patch]
func (s *server) patchSystemGenericCredentials(c *gin.Context) {
	if !s.requireSecretManagement(c) {
		return
	}

	ctx := c.Request.Context()
	name := c.Param("generic-credentials")

	var req patchGenericCredentialsRequest
	if !bindJSONOrError(c, &req) {
		return
	}

	secret := corev1.Secret{}
	if err := s.client.Get(
		ctx,
		types.NamespacedName{
			Namespace: s.cfg.SystemResourcesNamespace,
			Name:      name,
		},
		&secret,
	); err != nil {
		_ = c.Error(err)
		return
	}

	if err := validateGenericCredentialSecret(&secret); err != nil {
		_ = c.Error(err)
		return
	}

	applyGenericCredentialsPatchToK8sSecret(&secret, req)

	if err := validateSecretNotEmpty(&secret); err != nil {
		_ = c.Error(libhttp.Error(err, http.StatusBadRequest))
		return
	}

	if err := s.client.Update(ctx, &secret); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, sanitizeGenericCredentials(secret))
}

// @id PatchSharedGenericCredentials
// @Summary Patch shared generic credentials
// @Description Patch shared generic credentials. Merges provided data
// @Description with existing data. Use removeKeys to delete specific keys.
// @Description Returns a heavily redacted Kubernetes Secret resource.
// @Tags Credentials, Generic Credentials, Shared
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param generic-credentials path string true "Generic credentials name"
// @Param body body patchGenericCredentialsRequest true "GenericCredentials patch"
// @Success 200 {object} object "Secret resource (k8s.io/api/core/v1.Secret)"
// @Router /v1beta1/shared/generic-credentials/{generic-credentials} [patch]
func (s *server) patchSharedGenericCredentials(c *gin.Context) {
	if !s.requireSecretManagement(c) {
		return
	}

	ctx := c.Request.Context()
	name := c.Param("generic-credentials")

	var req patchGenericCredentialsRequest
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

	if err := validateGenericCredentialSecret(&secret); err != nil {
		_ = c.Error(err)
		return
	}

	applyGenericCredentialsPatchToK8sSecret(&secret, req)

	if err := validateSecretNotEmpty(&secret); err != nil {
		_ = c.Error(libhttp.Error(err, http.StatusBadRequest))
		return
	}

	if err := s.client.Update(ctx, &secret); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, sanitizeGenericCredentials(secret))
}

func applyGenericCredentialsPatchToK8sSecret(secret *corev1.Secret, req patchGenericCredentialsRequest) {
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

	// Remove specified keys
	for _, key := range req.RemoveKeys {
		delete(secret.Data, key)
	}

	// Merge new data (add or update keys)
	if secret.Data == nil {
		secret.Data = make(map[string][]byte, len(req.Data))
	}
	for key, value := range req.Data {
		secret.Data[key] = []byte(value)
	}
}

func validateSecretNotEmpty(secret *corev1.Secret) error {
	if len(secret.Data) == 0 {
		return errEmptySecret
	}
	return nil
}
