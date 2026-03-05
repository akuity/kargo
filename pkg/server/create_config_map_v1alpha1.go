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

type configMap struct {
	systemLevel bool
	project     string
	name        string
	description string
	data        map[string]string
}

func (s *server) CreateConfigMap(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.CreateConfigMapRequest],
) (*connect.Response[svcv1alpha1.CreateConfigMapResponse], error) {
	if err := s.validateCreateConfigMapRequest(ctx, req.Msg); err != nil {
		return nil, err
	}

	configMap := s.configMapToK8sConfigMap(configMap{
		systemLevel: req.Msg.SystemLevel,
		project:     req.Msg.Project,
		name:        req.Msg.Name,
		data:        req.Msg.Data,
		description: req.Msg.Description,
	})

	if err := s.client.Create(ctx, configMap); err != nil {
		return nil, fmt.Errorf("create configmap: %w", err)
	}

	return connect.NewResponse(&svcv1alpha1.CreateConfigMapResponse{
		ConfigMap: configMap,
	}), nil
}

func (s *server) validateCreateConfigMapRequest(
	ctx context.Context,
	req *svcv1alpha1.CreateConfigMapRequest,
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
			errors.New("ConfigMap data cannot be empty"))
	}

	return nil
}

func (s *server) configMapToK8sConfigMap(cm configMap) *corev1.ConfigMap {
	var namespace string
	if cm.systemLevel {
		namespace = s.cfg.SystemResourcesNamespace
	} else {
		namespace = cm.project
		if namespace == "" {
			namespace = s.cfg.SharedResourcesNamespace
		}
	}
	configMapObj := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cm.name,
			Namespace: namespace,
		},
		Data: cm.data,
	}
	if cm.description != "" {
		configMapObj.Annotations = map[string]string{
			kargoapi.AnnotationKeyDescription: cm.description,
		}
	}
	return configMapObj
}

// createConfigMapRequest is the request body for creating a ConfigMap.
type createConfigMapRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Data        map[string]string `json:"data"`
} // @name CreateConfigMapRequest

// @id CreateProjectConfigMap
// @Summary Create a project-level ConfigMap
// @Description Create a ConfigMap in a project's namespace. Returns the created
// @Description Kubernetes ConfigMap resource.
// @Tags Core, Generic Config, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param body body createConfigMapRequest true "ConfigMap"
// @Success 201 {object} object "ConfigMap resource (k8s.io/api/core/v1.ConfigMap)"
// @Router /v1beta1/projects/{project}/configmaps [post]
func (s *server) createProjectConfigMap(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")

	var req createConfigMapRequest
	if !bindJSONOrError(c, &req) {
		return
	}

	if err := validateRESTCreateConfigMapRequest(req); err != nil {
		_ = c.Error(libhttp.Error(err, http.StatusBadRequest))
		return
	}

	configMapObj := s.configMapToK8sConfigMap(configMap{
		project:     project,
		name:        req.Name,
		description: req.Description,
		data:        req.Data,
	})

	if err := s.client.Create(ctx, configMapObj); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, configMapObj)
}

// @id CreateSystemConfigMap
// @Summary Create a system-level ConfigMap
// @Description Create a system-level ConfigMap. Returns the created Kubernetes
// @Description ConfigMap resource.
// @Tags Core, Generic Config, System-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body createConfigMapRequest true "ConfigMap"
// @Success 201 {object} object "ConfigMap resource (k8s.io/api/core/v1.ConfigMap)"
// @Router /v1beta1/system/configmaps [post]
func (s *server) createSystemConfigMap(c *gin.Context) {
	ctx := c.Request.Context()

	var req createConfigMapRequest
	if !bindJSONOrError(c, &req) {
		return
	}

	if err := validateRESTCreateConfigMapRequest(req); err != nil {
		_ = c.Error(libhttp.Error(err, http.StatusBadRequest))
		return
	}

	configMapObj := s.configMapToK8sConfigMap(configMap{
		systemLevel: true,
		name:        req.Name,
		description: req.Description,
		data:        req.Data,
	})

	if err := s.client.Create(ctx, configMapObj); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, configMapObj)
}

// @id CreateSharedConfigMap
// @Summary Create a shared ConfigMap
// @Description Create a shared ConfigMap referenceable by all projects. Returns
// @Description the created Kubernetes ConfigMap resource.
// @Tags Core, Generic Config, Shared
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body createConfigMapRequest true "ConfigMap"
// @Success 201 {object} object "ConfigMap resource (k8s.io/api/core/v1.ConfigMap)"
// @Router /v1beta1/shared/configmaps [post]
func (s *server) createSharedConfigMap(c *gin.Context) {
	ctx := c.Request.Context()

	var req createConfigMapRequest
	if !bindJSONOrError(c, &req) {
		return
	}

	if err := validateRESTCreateConfigMapRequest(req); err != nil {
		_ = c.Error(libhttp.Error(err, http.StatusBadRequest))
		return
	}

	configMapObj := s.configMapToK8sConfigMap(configMap{
		name:        req.Name,
		description: req.Description,
		data:        req.Data,
	})

	if err := s.client.Create(ctx, configMapObj); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, configMapObj)
}

func validateRESTCreateConfigMapRequest(req createConfigMapRequest) error {
	if req.Name == "" {
		return errors.New("name should not be empty")
	}
	if len(req.Data) == 0 {
		return errors.New("ConfigMap data cannot be empty")
	}
	return nil
}
