package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
)

type genericCredentials struct {
	systemLevel bool
	project     string
	name        string
	description string
	data        map[string]string
}

func (s *server) CreateGenericCredentials(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.CreateGenericCredentialsRequest],
) (*connect.Response[svcv1alpha1.CreateGenericCredentialsResponse], error) {
	// Check if secret management is enabled
	if !s.cfg.SecretManagementEnabled {
		return nil, connect.NewError(connect.CodeUnimplemented, errSecretManagementDisabled)
	}

	if err := s.validateGenericCredentialsRequest(ctx, req.Msg); err != nil {
		return nil, err
	}

	secret := s.genericCredentialsToK8sSecret(
		genericCredentials{
			systemLevel: req.Msg.SystemLevel,
			project:     req.Msg.Project,
			name:        req.Msg.Name,
			data:        req.Msg.Data,
			description: req.Msg.Description,
		},
	)
	if err := s.client.Create(ctx, secret); err != nil {
		return nil, fmt.Errorf("create secret: %w", err)
	}

	return connect.NewResponse(
		&svcv1alpha1.CreateGenericCredentialsResponse{
			Credentials: sanitizeGenericCredentials(*secret),
		},
	), nil
}

func (s *server) validateGenericCredentialsRequest(
	ctx context.Context,
	req *svcv1alpha1.CreateGenericCredentialsRequest,
) error {
	if !req.SystemLevel && req.Project != "" {
		if err := s.validateProjectExists(ctx, req.Project); err != nil {
			return err
		}
	}

	if err := validateFieldNotEmpty("name", req.Name); err != nil {
		return err
	}

	if len(req.Data) == 0 {
		return connect.NewError(connect.CodeInvalidArgument,
			errors.New("cannot create empty secret"))
	}

	return nil
}

// createGenericCredentialsRequest is the request body for creating generic
// credentials.
type createGenericCredentialsRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Data        map[string]string `json:"data,omitempty"`
} // @name CreateGenericCredentialsRequest

// @id CreateProjectGenericCredentials
// @Summary Create project-level generic credentials
// @Description Create project-level generic credentials. Returns a heavily
// @Description redacted Kubernetes Secret resource.
// @Tags Credentials, Generic Credentials, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param body body createGenericCredentialsRequest true "Generic credentials"
// @Success 201 {object} object "Secret resource (k8s.io/api/core/v1.Secret)"
// @Router /v1beta1/projects/{project}/generic-credentials [post]
func (s *server) createProjectGenericCredentials(c *gin.Context) {
	if !s.requireSecretManagement(c) {
		return
	}

	ctx := c.Request.Context()
	project := c.Param("project")

	var req createGenericCredentialsRequest
	if !bindJSONOrError(c, &req) {
		return
	}

	if err := validateRESTCreateGenericCredentialsRequest(req); err != nil {
		_ = c.Error(libhttp.Error(err, http.StatusBadRequest))
		return
	}

	secret := s.genericCredentialsToK8sSecret(
		genericCredentials{
			project:     project,
			name:        req.Name,
			description: req.Description,
			data:        req.Data,
		},
	)
	if err := s.client.Create(ctx, secret); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, sanitizeGenericCredentials(*secret))
}

// @id CreateSystemGenericCredentials
// @Summary Create system-level generic credentials
// @Description Create system-level generic credentials. Returns a heavily
// @Description redacted Kubernetes Secret resource.
// @Tags Credentials, Generic Credentials, System-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body createGenericCredentialsRequest true "Generic credentials"
// @Success 201 {object} object "Secret resource (k8s.io/api/core/v1.Secret)"
// @Router /v1beta1/system/generic-credentials [post]
func (s *server) createSystemGenericCredentials(c *gin.Context) {
	if !s.requireSecretManagement(c) {
		return
	}

	ctx := c.Request.Context()

	var req createGenericCredentialsRequest
	if !bindJSONOrError(c, &req) {
		return
	}

	if err := validateRESTCreateGenericCredentialsRequest(req); err != nil {
		_ = c.Error(libhttp.Error(err, http.StatusBadRequest))
		return
	}

	secret := s.genericCredentialsToK8sSecret(
		genericCredentials{
			systemLevel: true,
			name:        req.Name,
			description: req.Description,
			data:        req.Data,
		},
	)
	if err := s.client.Create(ctx, secret); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, sanitizeGenericCredentials(*secret))
}

// @id CreateSharedGenericCredentials
// @Summary Create shared generic credentials
// @Description Create shared generic credentials referenceable by all
// @Description projects. Returns a heavily redacted Kubernetes Secret resource.
// @Tags Credentials, Generic Credentials, Shared
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body createGenericCredentialsRequest true "Generic credentials"
// @Success 201 {object} object "Secret resource (k8s.io/api/core/v1.Secret)"
// @Router /v1beta1/shared/generic-credentials [post]
func (s *server) createSharedGenericCredentials(c *gin.Context) {
	if !s.requireSecretManagement(c) {
		return
	}

	ctx := c.Request.Context()

	var req createGenericCredentialsRequest
	if !bindJSONOrError(c, &req) {
		return
	}

	if err := validateRESTCreateGenericCredentialsRequest(req); err != nil {
		_ = c.Error(libhttp.Error(err, http.StatusBadRequest))
		return
	}

	secret := s.genericCredentialsToK8sSecret(
		genericCredentials{
			name:        req.Name,
			description: req.Description,
			data:        req.Data,
		},
	)
	if err := s.client.Create(ctx, secret); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, sanitizeGenericCredentials(*secret))
}

func validateRESTCreateGenericCredentialsRequest(
	req createGenericCredentialsRequest,
) error {
	if req.Name == "" {
		return errors.New("name should not be empty")
	}
	if len(req.Data) == 0 {
		return errors.New("cannot create empty secret")
	}
	return nil
}

func (s *server) genericCredentialsToK8sSecret(
	creds genericCredentials,
) *corev1.Secret {
	var namespace string
	if creds.systemLevel {
		namespace = s.cfg.SystemResourcesNamespace
	} else {
		namespace = creds.project
		if namespace == "" {
			namespace = s.cfg.SharedResourcesNamespace
		}
	}

	secretsData := map[string][]byte{}
	for key, value := range creds.data {
		secretsData[key] = []byte(value)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      creds.name,
			Labels: map[string]string{
				kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGeneric,
			},
		},
		Data: secretsData,
	}

	if creds.description != "" {
		secret.Annotations = map[string]string{
			kargoapi.AnnotationKeyDescription: creds.description,
		}
	}

	return secret
}
