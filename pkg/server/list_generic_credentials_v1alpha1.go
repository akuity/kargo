package server

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func (s *server) ListGenericCredentials(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListGenericCredentialsRequest],
) (*connect.Response[svcv1alpha1.ListGenericCredentialsResponse], error) {
	// Check if secret management is enabled
	if !s.cfg.SecretManagementEnabled {
		return nil, connect.NewError(connect.CodeUnimplemented, errSecretManagementDisabled)
	}

	var cl client.Client = s.client

	var namespace string
	if req.Msg.SystemLevel {
		namespace = s.cfg.SystemResourcesNamespace
	} else {
		project := req.Msg.Project
		if project != "" {
			if err := s.validateProjectExists(ctx, project); err != nil {
				return nil, err
			}
		}
		namespace = project
		if namespace == "" {
			namespace = s.cfg.SharedResourcesNamespace
			// Note: We're using the internal client here so that all authenticated
			// users can see what shared generic credentials exist without requiring
			// actual permissions to list those Secrets. The Secrets are heavily
			// redacted.
			cl = s.client.InternalClient()
		}
	}

	// List secrets having the label that indicates this is a generic secret.
	var secretsList corev1.SecretList
	if err := cl.List(
		ctx,
		&secretsList,
		client.InNamespace(namespace),
		client.MatchingLabels{
			kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGeneric,
		},
	); err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}

	// Sort the secrets by name
	secrets := secretsList.Items
	slices.SortFunc(secrets, func(lhs, rhs corev1.Secret) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	sanitizedSecrets := make([]*corev1.Secret, len(secrets))
	for i, secret := range secrets {
		sanitizedSecrets[i] = sanitizeGenericCredentials(secret)
	}

	return connect.NewResponse(&svcv1alpha1.ListGenericCredentialsResponse{
		Credentials: sanitizedSecrets,
	}), nil
}

// @id ListProjectGenericCredentials
// @Summary List project-level generic credentials
// @Description List project-level generic credentials. Returns a Kubernetes
// @Description SecretList resource containing heavily redacted Secrets.
// @Tags Credentials, Generic Credentials, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Produce json
// @Success 200 {object} object "SecretList resource (k8s.io/api/core/v1.SecretList)"
// @Router /v1beta1/projects/{project}/generic-credentials [get]
func (s *server) listProjectGenericCredentials(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")

	list := &corev1.SecretList{}
	if err := s.client.List(
		ctx,
		list,
		client.InNamespace(project),
		client.MatchingLabels{
			kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGeneric,
		},
	); err != nil {
		_ = c.Error(err)
		return
	}

	// Sort ascending by name
	slices.SortFunc(list.Items, func(lhs, rhs corev1.Secret) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	for i := range list.Items {
		list.Items[i] = *sanitizeGenericCredentials(list.Items[i])
	}

	c.JSON(http.StatusOK, list)
}

// @id ListSystemGenericCredentials
// @Summary List system-level generic credentials
// @Description List system-level generic credentials. Returns a Kubernetes
// @Description SecretList resource containing heavily redacted Secrets.
// @Tags Credentials, Generic Credentials, System-Level
// @Security BearerAuth
// @Produce json
// @Success 200 {object} object "SecretList resource (k8s.io/api/core/v1.SecretList)"
// @Router /v1beta1/system/generic-credentials [get]
func (s *server) listSystemGenericCredentials(c *gin.Context) {
	ctx := c.Request.Context()

	list := &corev1.SecretList{}
	if err := s.client.List(
		ctx,
		list,
		client.InNamespace(s.cfg.SystemResourcesNamespace),
		client.MatchingLabels{
			kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGeneric,
		},
	); err != nil {
		_ = c.Error(err)
		return
	}

	// Sort ascending by name
	slices.SortFunc(list.Items, func(lhs, rhs corev1.Secret) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	for i := range list.Items {
		list.Items[i] = *sanitizeGenericCredentials(list.Items[i])
	}

	c.JSON(http.StatusOK, list)
}

// @id ListSharedGenericCredentials
// @Summary List shared generic credentials
// @Description List shared generic credentials. Returns a Kubernetes SecretList
// @Description resource containing heavily redacted Secrets.
// @Tags Credentials, Generic Credentials, Shared
// @Security BearerAuth
// @Produce json
// @Success 200 {object} object "SecretList resource (k8s.io/api/core/v1.SecretList)"
// @Router /v1beta1/shared/generic-credentials [get]
func (s *server) listSharedGenericCredentials(c *gin.Context) {
	ctx := c.Request.Context()

	// Note: We're using the internal client here so that all authenticated
	// users can see what shared generic credentials exist without requiring
	// actual permissions to list those Secrets. The Secrets are heavily
	// redacted.
	list := &corev1.SecretList{}
	if err := s.client.InternalClient().List(
		ctx,
		list,
		client.InNamespace(s.cfg.SharedResourcesNamespace),
		client.MatchingLabels{
			kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGeneric,
		},
	); err != nil {
		_ = c.Error(err)
		return
	}

	// Sort ascending by name
	slices.SortFunc(list.Items, func(lhs, rhs corev1.Secret) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	for i := range list.Items {
		list.Items[i] = *sanitizeGenericCredentials(list.Items[i])
	}

	c.JSON(http.StatusOK, list)
}

// sanitizeGenericCredentials returns a copy of the secret with all values in the
// stringData map redacted. All annotations are also redacted because AT LEAST
// "last-applied-configuration" is a known vector for leaking sensitive
// information and unknown configuration management tools may use other
// annotations in a manner similar to "last-applied-configuration". There is no
// concern over labels because the constraints on label values rule out use in a
// manner similar to that of the "last-applied-configuration" annotation.
func sanitizeGenericCredentials(secret corev1.Secret) *corev1.Secret {
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
	for k := range s.Data {
		s.StringData[k] = redacted
	}
	s.Data = nil
	return s
}
