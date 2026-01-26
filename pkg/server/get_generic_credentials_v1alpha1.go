package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// @id GetProjectGenericCredentials
// @Summary Retrieve project-level generic credentials
// @Description Retrieve project-level generic credentials by name. Returns a
// @Description heavily redacted Kubernetes Secret resource.
// @Tags Credentials, Generic Credentials, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param generic-credentials path string true "Credentials name"
// @Produce json
// @Success 200 {object} object "Secret resource (k8s.io/api/core/v1.Secret)"
// @Router /v1beta1/projects/{project}/generic-credentials/{generic-credentials} [get]
func (s *server) getProjectGenericCredentials(c *gin.Context) {
	if !s.requireSecretManagement(c) {
		return
	}

	ctx := c.Request.Context()

	project := c.Param("project")
	name := c.Param("generic-credentials")

	secret := &corev1.Secret{}
	if err := s.client.Get(
		ctx,
		client.ObjectKey{Namespace: project, Name: name},
		secret,
	); err != nil {
		_ = c.Error(err)
		return
	}

	if err := validateGenericCredentialSecret(secret); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, sanitizeGenericCredentials(*secret))
}

// @id GetSystemGenericCredentials
// @Summary Retrieve system-level generic credentials
// @Description Retrieve system-level generic credentials by name. Returns a
// @Description heavily redacted Kubernetes Secret resource.
// @Tags Credentials, Generic Credentials, System-Level
// @Security BearerAuth
// @Param generic-credentials path string true "Credentials name"
// @Produce json
// @Success 200 {object} object "Secret resource (k8s.io/api/core/v1.Secret)"
// @Router /v1beta1/system/generic-credentials/{generic-credentials} [get]
func (s *server) getSystemGenericCredentials(c *gin.Context) {
	if !s.requireSecretManagement(c) {
		return
	}

	ctx := c.Request.Context()

	name := c.Param("generic-credentials")

	secret := &corev1.Secret{}
	if err := s.client.Get(
		ctx,
		client.ObjectKey{Namespace: s.cfg.SystemResourcesNamespace, Name: name},
		secret,
	); err != nil {
		_ = c.Error(err)
		return
	}

	if err := validateGenericCredentialSecret(secret); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, sanitizeGenericCredentials(*secret))
}

// @id GetSharedGenericCredentials
// @Summary Retrieve shared generic credentials
// @Description Retrieve shared generic credentials by name. Returns a
// @Description heavily redacted Kubernetes Secret resource.
// @Tags Credentials, Generic Credentials, Shared
// @Security BearerAuth
// @Param generic-credentials path string true "Credentials name"
// @Produce json
// @Success 200 {object} object "Secret resource (k8s.io/api/core/v1.Secret)"
// @Router /v1beta1/shared/generic-credentials/{generic-credentials} [get]
func (s *server) getSharedGenericCredentials(c *gin.Context) {
	if !s.requireSecretManagement(c) {
		return
	}

	ctx := c.Request.Context()

	name := c.Param("generic-credentials")

	secret := &corev1.Secret{}
	if err := s.client.Get(
		ctx,
		client.ObjectKey{Namespace: s.cfg.SharedResourcesNamespace, Name: name},
		secret,
	); err != nil {
		_ = c.Error(err)
		return
	}

	if err := validateGenericCredentialSecret(secret); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, sanitizeGenericCredentials(*secret))
}
