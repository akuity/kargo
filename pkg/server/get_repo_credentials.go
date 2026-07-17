package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libCreds "github.com/akuity/kargo/pkg/credentials"
)

const redacted = "*** REDACTED ***"

// @id GetProjectRepoCredentials
// @Summary Retrieve project-level repository credentials
// @Description Retrieve project-level repository credentials by name. Returns a
// @Description heavily redacted Kubernetes Secret resource.
// @Tags Credentials, Repo Credentials, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param repo-credentials path string true "Credentials name"
// @Produce json
// @Success 200 {object} corev1.Secret "Secret resource (k8s.io/api/core/v1.Secret)"
// @Router /v1beta1/projects/{project}/repo-credentials/{repo-credentials} [get]
func (s *server) getProjectRepoCredentials(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	name := c.Param("repo-credentials")

	secret := &corev1.Secret{}
	if err := s.client.Get(
		ctx,
		client.ObjectKey{Namespace: project, Name: name},
		secret,
	); err != nil {
		_ = c.Error(err)
		return
	}

	if err := validateRepoCredentialSecret(secret); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, sanitizeCredentialSecret(*secret))
}

// @id GetSharedRepoCredentials
// @Summary Retrieve shared repository credentials
// @Description Retrieve shared repository credentials by name. Returns a
// @Description heavily redacted Kubernetes Secret resource.
// @Tags Credentials, Repo Credentials, Shared
// @Security BearerAuth
// @Param repo-credentials path string true "Credentials name"
// @Produce json
// @Success 200 {object} corev1.Secret "Secret resource (k8s.io/api/core/v1.Secret)"
// @Router /v1beta1/shared/repo-credentials/{repo-credentials} [get]
func (s *server) getSharedRepoCredentials(c *gin.Context) {
	ctx := c.Request.Context()

	name := c.Param("repo-credentials")

	secret := &corev1.Secret{}
	if err := s.client.Get(
		ctx,
		client.ObjectKey{Namespace: s.cfg.SharedResourcesNamespace, Name: name},
		secret,
	); err != nil {
		_ = c.Error(err)
		return
	}

	if err := validateRepoCredentialSecret(secret); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, sanitizeCredentialSecret(*secret))
}

// sanitizeCredentialSecret returns a copy of the secret with all values in the
// stringData map redacted except for those with specific keys that are known to
// represent non-sensitive information when used correctly. The primary
// intention, at present, is only to redact the value associated with the
// "password" key, but this approach prevents accidental exposure of the
// password in the event that it has accidentally been assigned to a
// wrong/unknown key, such as "pass" or "passwd". All annotations are also
// redacted because AT LEAST "last-applied-configuration" is a known vector for
// leaking sensitive information and unknown configuration management tools may
// use other annotations in a manner similar to "last-applied-configuration".
// There is no concern over labels because the constraints on label values rule
// out use in a manner similar to that of the "last-applied-configuration"
// annotation.
func sanitizeCredentialSecret(secret corev1.Secret) *corev1.Secret {
	s := secret.DeepCopy()
	s.StringData = make(map[string]string, len(s.Data))
	for k, v := range s.Annotations {
		switch k {
		case kargoapi.AnnotationKeyDescription:
			s.Annotations[k] = v
		default:
			s.Annotations[k] = redacted
		}
	}
	for k, v := range s.Data {
		switch k {
		case libCreds.FieldRepoURL, libCreds.FieldRepoURLIsRegex, libCreds.FieldUsername:
			s.StringData[k] = string(v)
		default:
			s.StringData[k] = redacted
		}
	}
	s.Data = nil
	return s
}
