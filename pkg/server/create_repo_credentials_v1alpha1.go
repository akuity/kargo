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
	libCreds "github.com/akuity/kargo/pkg/credentials"
	libhttp "github.com/akuity/kargo/pkg/http"
)

type repoCredentials struct {
	project        string
	name           string
	credType       string
	repoURL        string
	repoURLIsRegex bool
	username       string
	password       string
	description    string
}

func (s *server) CreateRepoCredentials(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.CreateRepoCredentialsRequest],
) (*connect.Response[svcv1alpha1.CreateRepoCredentialsResponse], error) {
	// Check if secret management is enabled
	if !s.cfg.SecretManagementEnabled {
		return nil, connect.NewError(
			connect.CodeUnimplemented,
			errSecretManagementDisabled,
		)
	}

	if err := s.validateCreateRepoCredentialsRequest(req.Msg); err != nil {
		return nil, err
	}

	secret := s.repoCredentialsToK8sSecret(
		repoCredentials{
			project:        req.Msg.GetProject(),
			name:           req.Msg.GetName(),
			description:    req.Msg.GetDescription(),
			credType:       req.Msg.GetType(),
			repoURL:        req.Msg.GetRepoUrl(),
			repoURLIsRegex: req.Msg.GetRepoUrlIsRegex(),
			username:       req.Msg.GetUsername(),
			password:       req.Msg.GetPassword(),
		},
	)
	if err := s.client.Create(ctx, secret); err != nil {
		return nil, fmt.Errorf("create secret: %w", err)
	}

	return connect.NewResponse(
		&svcv1alpha1.CreateRepoCredentialsResponse{
			Credentials: sanitizeCredentialSecret(*secret),
		},
	), nil
}

func (s *server) validateCreateRepoCredentialsRequest(
	req *svcv1alpha1.CreateRepoCredentialsRequest,
) error {
	if err := validateFieldNotEmpty("name", req.GetName()); err != nil {
		return err
	}
	if err := validateFieldNotEmpty("type", req.GetType()); err != nil {
		return err
	}
	switch req.GetType() {
	case kargoapi.LabelValueCredentialTypeGit,
		kargoapi.LabelValueCredentialTypeHelm,
		kargoapi.LabelValueCredentialTypeImage:
	default:
		return connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("type should be one of git, helm, or image"),
		)
	}
	if req.GetRepoUrl() == "" {
		return connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("repoURL should not be empty"),
		)
	}
	if err := validateFieldNotEmpty("username", req.GetUsername()); err != nil {
		return err
	}
	return validateFieldNotEmpty("password", req.GetPassword())
}

// createRepoCredentialsRequest is the request body for creating repository
// credentials.
type createRepoCredentialsRequest struct {
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
	Type           string `json:"type"`
	RepoURL        string `json:"repoUrl"`
	RepoURLIsRegex bool   `json:"repoUrlIsRegex,omitempty"`
	Username       string `json:"username"`
	Password       string `json:"password"`
} // @name CreateRepoCredentialsRequest

// @id CreateProjectRepoCredentials
// @Summary Create project-level repository credentials
// @Description Create project-level repository credentials. Returns a heavily
// @Description redacted Kubernetes Secret resource.
// @Tags Credentials, Repo Credentials, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param body body createRepoCredentialsRequest true "Credentials"
// @Success 201 {object} object "Secret resource (k8s.io/api/core/v1.Secret)"
// @Router /v1beta1/projects/{project}/repo-credentials [post]
func (s *server) createProjectRepoCredentials(c *gin.Context) {
	if !s.requireSecretManagement(c) {
		return
	}

	ctx := c.Request.Context()
	project := c.Param("project")

	var req createRepoCredentialsRequest
	if !bindJSONOrError(c, &req) {
		return
	}

	if err := s.validateRESTCreateRepoCredentialsRequest(req); err != nil {
		_ = c.Error(libhttp.Error(err, http.StatusBadRequest))
		return
	}

	secret := s.repoCredentialsToK8sSecret(
		repoCredentials{
			project:        project,
			name:           req.Name,
			description:    req.Description,
			credType:       req.Type,
			repoURL:        req.RepoURL,
			repoURLIsRegex: req.RepoURLIsRegex,
			username:       req.Username,
			password:       req.Password,
		},
	)
	if err := s.client.Create(ctx, secret); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, sanitizeCredentialSecret(*secret))
}

// @id CreateSharedRepoCredentials
// @Summary Create shared repository credentials
// @Description Create shared repository credentials. Returns a heavily
// @Description redacted Kubernetes Secret resource.
// @Tags Credentials, Repo Credentials, Shared
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body createRepoCredentialsRequest true "Credentials"
// @Success 201 {object} object "Secret resource (k8s.io/api/core/v1.Secret)"
// @Router /v1beta1/shared/repo-credentials [post]
func (s *server) createSharedRepoCredentials(c *gin.Context) {
	if !s.requireSecretManagement(c) {
		return
	}

	ctx := c.Request.Context()

	var req createRepoCredentialsRequest
	if !bindJSONOrError(c, &req) {
		return
	}

	if err := s.validateRESTCreateRepoCredentialsRequest(req); err != nil {
		_ = c.Error(libhttp.Error(err, http.StatusBadRequest))
		return
	}

	secret := s.repoCredentialsToK8sSecret(
		repoCredentials{
			name:           req.Name,
			description:    req.Description,
			credType:       req.Type,
			repoURL:        req.RepoURL,
			repoURLIsRegex: req.RepoURLIsRegex,
			username:       req.Username,
			password:       req.Password,
		},
	)
	if err := s.client.Create(ctx, secret); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, sanitizeCredentialSecret(*secret))
}

func (s *server) validateRESTCreateRepoCredentialsRequest(
	req createRepoCredentialsRequest,
) error {
	if req.Name == "" {
		return errors.New("name should not be empty")
	}
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
		return errors.New("repoURL should not be empty")
	}
	if req.Username == "" {
		return errors.New("username should not be empty")
	}
	if req.Password == "" {
		return errors.New("password should not be empty")
	}
	return nil
}

func (s *server) repoCredentialsToK8sSecret(
	creds repoCredentials,
) *corev1.Secret {
	namespace := creds.project
	if namespace == "" {
		namespace = s.cfg.SharedResourcesNamespace
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      creds.name,
			Labels: map[string]string{
				kargoapi.LabelKeyCredentialType: creds.credType,
			},
		},
		Data: map[string][]byte{
			libCreds.FieldRepoURL:  []byte(creds.repoURL),
			libCreds.FieldUsername: []byte(creds.username),
			libCreds.FieldPassword: []byte(creds.password),
		},
	}
	if creds.description != "" {
		secret.Annotations = map[string]string{
			kargoapi.AnnotationKeyDescription: creds.description,
		}
	}
	if creds.repoURLIsRegex {
		secret.Data[libCreds.FieldRepoURLIsRegex] = []byte("true")
	}
	return secret
}
