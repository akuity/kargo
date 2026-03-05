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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func (s *server) ListRepoCredentials(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListRepoCredentialsRequest],
) (*connect.Response[svcv1alpha1.ListRepoCredentialsResponse], error) {
	// Check if secret management is enabled
	if !s.cfg.SecretManagementEnabled {
		return nil, connect.NewError(connect.CodeUnimplemented, errSecretManagementDisabled)
	}

	var cl client.Client = s.client

	project := req.Msg.GetProject()
	if project != "" {
		if err := s.validateProjectExists(ctx, project); err != nil {
			return nil, err
		}
	}
	namespace := project
	if namespace == "" {
		namespace = s.cfg.SharedResourcesNamespace
		// Note: We're using the internal client here so that all authenticated
		// users can see what shared repo credentials exist without requiring actual
		// permissions to list those Secrets. The Secrets are heavily redacted.
		cl = s.client.InternalClient()
	}

	credsLabelSelector := labels.NewSelector()

	credsLabelSelectorRequirement, err := labels.NewRequirement(
		kargoapi.LabelKeyCredentialType,
		selection.In,
		[]string{
			kargoapi.LabelValueCredentialTypeGit,
			kargoapi.LabelValueCredentialTypeHelm,
			kargoapi.LabelValueCredentialTypeImage,
		})

	if err != nil {
		return nil, err
	}

	credsLabelSelector = credsLabelSelector.Add(*credsLabelSelectorRequirement)

	var secretsList corev1.SecretList
	if err := cl.List(
		ctx,
		&secretsList,
		client.InNamespace(namespace),
		&client.ListOptions{
			LabelSelector: credsLabelSelector,
		},
	); err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}

	// Sort ascending by name
	slices.SortFunc(secretsList.Items, func(lhs, rhs corev1.Secret) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	secrets := make([]*corev1.Secret, len(secretsList.Items))
	for i, secret := range secretsList.Items {
		secrets[i] = sanitizeCredentialSecret(secret)
	}

	return connect.NewResponse(&svcv1alpha1.ListRepoCredentialsResponse{
		Credentials: secrets,
	}), nil
}

// @id ListProjectRepoCredentials
// @Summary List project-level repository credentials
// @Description List project-level repository credentials. Returns a SecretList
// @Description resource containing heavily redacted Secrets.
// @Tags Credentials, Repo Credentials, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Produce json
// @Success 200 {object} object "SecretList resource (k8s.io/api/core/v1.SecretList)"
// @Router /v1beta1/projects/{project}/repo-credentials [get]
func (s *server) listProjectRepoCredentials(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")

	credsLabelSelector := labels.NewSelector()
	credsLabelSelectorRequirement, err := labels.NewRequirement(
		kargoapi.LabelKeyCredentialType,
		selection.In,
		[]string{
			kargoapi.LabelValueCredentialTypeGit,
			kargoapi.LabelValueCredentialTypeHelm,
			kargoapi.LabelValueCredentialTypeImage,
		})
	if err != nil {
		_ = c.Error(err)
		return
	}

	credsLabelSelector = credsLabelSelector.Add(*credsLabelSelectorRequirement)

	list := &corev1.SecretList{}
	if err := s.client.List(
		ctx,
		list,
		client.InNamespace(project),
		&client.ListOptions{LabelSelector: credsLabelSelector},
	); err != nil {
		_ = c.Error(err)
		return
	}

	// Sort ascending by name
	slices.SortFunc(list.Items, func(lhs, rhs corev1.Secret) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	for i := range list.Items {
		list.Items[i] = *sanitizeCredentialSecret(list.Items[i])
	}

	c.JSON(http.StatusOK, list)
}

// @id ListSharedRepoCredentials
// @Summary List shared repository credentials
// @Description List shared repository credentials. Returns a SecretList
// @Description resource containing heavily redacted Secrets.
// @Tags Credentials, Repo Credentials, Shared
// @Security BearerAuth
// @Produce json
// @Success 200 {object} object "SecretList resource (k8s.io/api/core/v1.SecretList)"
// @Router /v1beta1/shared/repo-credentials [get]
func (s *server) listSharedRepoCredentials(c *gin.Context) {
	ctx := c.Request.Context()

	credsLabelSelector := labels.NewSelector()
	credsLabelSelectorRequirement, err := labels.NewRequirement(
		kargoapi.LabelKeyCredentialType,
		selection.In,
		[]string{
			kargoapi.LabelValueCredentialTypeGit,
			kargoapi.LabelValueCredentialTypeHelm,
			kargoapi.LabelValueCredentialTypeImage,
		})
	if err != nil {
		_ = c.Error(err)
		return
	}

	credsLabelSelector = credsLabelSelector.Add(*credsLabelSelectorRequirement)

	// Note: We're using the internal client here so that all authenticated
	// users can see what shared repo credentials exist without requiring actual
	// permissions to list those Secrets. The Secrets are heavily redacted.
	list := &corev1.SecretList{}
	if err := s.client.InternalClient().List(
		ctx,
		list,
		client.InNamespace(s.cfg.SharedResourcesNamespace),
		&client.ListOptions{LabelSelector: credsLabelSelector},
	); err != nil {
		_ = c.Error(err)
		return
	}

	// Sort ascending by name
	slices.SortFunc(list.Items, func(lhs, rhs corev1.Secret) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	for i := range list.Items {
		list.Items[i] = *sanitizeCredentialSecret(list.Items[i])
	}

	c.JSON(http.StatusOK, list)
}
