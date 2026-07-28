package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
)

type updateGenericCredentialsRequest struct {
	Description string            `json:"description,omitempty"`
	Replicate   bool              `json:"replicate,omitempty"`
	Data        map[string]string `json:"data"`
} // @name UpdateGenericCredentialsRequest

// @id UpdateProjectGenericCredentials
// @Summary Replace project-level generic credentials
// @Description Replace project-level generic credentials. All existing data is
// @Description replaced. Returns a heavily redacted Kubernetes Secret resource.
// @Tags Credentials, Generic Credentials, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param generic-credentials path string true "Generic credentials name"
// @Param body body updateGenericCredentialsRequest true "GenericCredentials"
// @Success 200 {object} corev1.Secret "Secret resource (k8s.io/api/core/v1.Secret)"
// @Router /v1beta1/projects/{project}/generic-credentials/{generic-credentials} [put]
func (s *server) updateProjectGenericCredentials(c *gin.Context) {
	if !s.requireSecretManagement(c) {
		return
	}

	ctx := c.Request.Context()

	project := c.Param("project")
	name := c.Param("generic-credentials")

	var req updateGenericCredentialsRequest
	if !bindJSONOrError(c, &req) {
		return
	}

	if len(req.Data) == 0 {
		_ = c.Error(libhttp.ErrorStr(
			"cannot update to empty secret",
			http.StatusBadRequest,
		))
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

	applyGenericCredentialsUpdateToK8sSecret(&secret, req)

	if err := s.client.Update(ctx, &secret); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, sanitizeGenericCredentials(secret))
}

// @id UpdateSystemGenericCredentials
// @Summary Replace system-level generic credentials
// @Description Replace system-level generic credentials. All existing data is
// @Description replaced. Returns a heavily redacted Kubernetes Secret resource.
// @Tags Credentials, Generic Credentials, System-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param generic-credentials path string true "Generic credentials name"
// @Param body body updateGenericCredentialsRequest true "GenericCredentials"
// @Success 200 {object} corev1.Secret "Secret resource (k8s.io/api/core/v1.Secret)"
// @Router /v1beta1/system/generic-credentials/{generic-credentials} [put]
func (s *server) updateSystemGenericCredentials(c *gin.Context) {
	if !s.requireSecretManagement(c) {
		return
	}

	ctx := c.Request.Context()
	name := c.Param("generic-credentials")

	var req updateGenericCredentialsRequest
	if !bindJSONOrError(c, &req) {
		return
	}

	if len(req.Data) == 0 {
		_ = c.Error(libhttp.ErrorStr(
			"cannot update to empty secret",
			http.StatusBadRequest,
		))
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

	applyGenericCredentialsUpdateToK8sSecret(&secret, req)

	if err := s.client.Update(ctx, &secret); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, sanitizeGenericCredentials(secret))
}

// @id UpdateSharedGenericCredentials
// @Summary Replace shared generic credentials
// @Description Replace shared generic credentials. All existing data is replaced.
// @Description Returns a heavily redacted Kubernetes Secret resource.
// @Tags Credentials, Generic Credentials, Shared
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param generic-credentials path string true "Generic credentials name"
// @Param body body updateGenericCredentialsRequest true "GenericCredentials"
// @Success 200 {object} corev1.Secret "Secret resource (k8s.io/api/core/v1.Secret)"
// @Router /v1beta1/shared/generic-credentials/{generic-credentials} [put]
func (s *server) updateSharedGenericCredentials(c *gin.Context) {
	if !s.requireSecretManagement(c) {
		return
	}

	ctx := c.Request.Context()
	name := c.Param("generic-credentials")

	var req updateGenericCredentialsRequest
	if !bindJSONOrError(c, &req) {
		return
	}

	if len(req.Data) == 0 {
		_ = c.Error(libhttp.ErrorStr(
			"cannot update to empty secret",
			http.StatusBadRequest,
		))
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

	applyGenericCredentialsUpdateToK8sSecret(&secret, req)

	if err := s.client.Update(ctx, &secret); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, sanitizeGenericCredentials(secret))
}

func applyGenericCredentialsUpdateToK8sSecret(secret *corev1.Secret, req updateGenericCredentialsRequest) {
	if secret.Annotations == nil {
		secret.Annotations = make(map[string]string)
	}

	// Set or clear the description annotation
	if req.Description != "" {
		secret.Annotations[kargoapi.AnnotationKeyDescription] = req.Description
	} else {
		delete(secret.Annotations, kargoapi.AnnotationKeyDescription)
	}

	// Set or clear the replicate-to annotation
	if req.Replicate {
		secret.Annotations[kargoapi.AnnotationKeyReplicateTo] = kargoapi.AnnotationValueReplicateToAll
	} else {
		delete(secret.Annotations, kargoapi.AnnotationKeyReplicateTo)
	}

	// Delete any keys in the secret that are not in the update
	for key := range secret.Data {
		if _, ok := req.Data[key]; !ok {
			delete(secret.Data, key)
		}
	}

	// Add or update the keys in the secret with the values from the update
	if secret.Data == nil {
		secret.Data = make(map[string][]byte, len(req.Data))
	}
	for key, value := range req.Data {
		if value != "" {
			secret.Data[key] = []byte(value)
		}
	}
}
