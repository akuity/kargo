package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/logging"
)

// @title Kargo API
// @version v1alpha1
// @description REST API for Kargo
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Bearer token authentication. Obtain token via OIDC/PKCE flow with your identity provider.
func (s *server) setupRESTRouter(ctx context.Context) *gin.Engine {
	router := gin.Default()

	// Error handling middleware
	router.Use(s.handleError)

	// Authentication middleware (only if auth is configured)
	if s.cfg.AdminConfig != nil || s.cfg.OIDCConfig != nil {
		router.Use(newAuthMiddleware(ctx, s.cfg, s.client.InternalClient()))
	}

	v1beta1 := router.Group("/v1beta1")
	{
		// =====================================================================
		// Authentication
		// =====================================================================
		v1beta1.POST("/login", bodyLimitMiddleware(1*1024*1024), s.adminLogin)

		// =====================================================================
		// Generic Resources (CRUD via group/version/kind/namespace/name)
		// These endpoints accept YAML/JSON manifests and need a larger limit (4MB).
		// =====================================================================
		resourceLimit := bodyLimitMiddleware(4 * 1024 * 1024)
		v1beta1.POST("/resources", resourceLimit, s.createResources)
		v1beta1.PUT("/resources", resourceLimit, s.updateResources)
		v1beta1.DELETE("/resources", resourceLimit, s.deleteResources)

		// All other endpoints use a 1MB limit
		defaultLimit := bodyLimitMiddleware(1 * 1024 * 1024)

		// =====================================================================
		// System-Level Endpoints (/v1beta1/system/*)
		// =====================================================================
		system := v1beta1.Group("/system")
		system.Use(defaultLimit)
		{
			// Configuration
			system.GET("/server-version", s.getVersionInfo)
			system.GET("/server-config", s.getConfig)
			system.GET("/public-server-config", s.getPublicConfig)
			system.GET("/cluster-config", s.getClusterConfig)
			system.POST("/cluster-config/refresh", s.refreshClusterConfig)
			system.DELETE("/cluster-config", s.deleteClusterConfig)

			// Roles
			system.GET("/roles", s.listSystemRoles)
			system.GET("/roles/:role", s.getSystemRole)
			system.POST("/roles/:role/api-tokens", s.createSystemAPIToken)

			// API Tokens
			system.GET("/api-tokens", s.listSystemAPITokens)
			system.GET("/api-tokens/:apitoken", s.getSystemAPIToken)
			system.DELETE("/api-tokens/:apitoken", s.deleteSystemAPIToken)

			// Generic Credentials
			system.GET("/generic-credentials", s.listSystemGenericCredentials)
			system.POST("/generic-credentials", s.createSystemGenericCredentials)
			system.GET("/generic-credentials/:generic-credentials", s.getSystemGenericCredentials)
			system.PUT("/generic-credentials/:generic-credentials", s.updateSystemGenericCredentials)
			system.PATCH("/generic-credentials/:generic-credentials", s.patchSystemGenericCredentials)
			system.DELETE("/generic-credentials/:generic-credentials", s.deleteSystemGenericCredentials)

			// ConfigMaps
			system.GET("/configmaps", s.listSystemConfigMaps)
			system.POST("/configmaps", s.createSystemConfigMap)
			system.GET("/configmaps/:configmap", s.getSystemConfigMap)
			system.PUT("/configmaps/:configmap", s.updateSystemConfigMap)
			system.PATCH("/configmaps/:configmap", s.patchSystemConfigMap)
			system.DELETE("/configmaps/:configmap", s.deleteSystemConfigMap)
		}

		// =====================================================================
		// Shared Resources (/v1beta1/shared/*)
		// =====================================================================
		shared := v1beta1.Group("/shared")
		shared.Use(defaultLimit)
		{
			// Cluster Analysis Templates (Argo Rollouts)
			shared.GET("/cluster-analysis-templates", s.listClusterAnalysisTemplates)
			shared.GET("/cluster-analysis-templates/:cluster-analysis-template", s.getClusterAnalysisTemplate)
			shared.DELETE("/cluster-analysis-templates/:cluster-analysis-template", s.deleteClusterAnalysisTemplate)

			// Cluster Promotion Tasks
			shared.GET("/cluster-promotion-tasks", s.listClusterPromotionTasks)
			shared.GET("/cluster-promotion-tasks/:cluster-promotion-task", s.getClusterPromotionTask)

			// Repo Credentials
			shared.GET("/repo-credentials", s.listSharedRepoCredentials)
			shared.POST("/repo-credentials", s.createSharedRepoCredentials)
			shared.GET("/repo-credentials/:repo-credentials", s.getSharedRepoCredentials)
			shared.PUT("/repo-credentials/:repo-credentials", s.updateSharedRepoCredentials)
			shared.PATCH("/repo-credentials/:repo-credentials", s.patchSharedRepoCredentials)
			shared.DELETE("/repo-credentials/:repo-credentials", s.deleteSharedRepoCredentials)

			// Generic Credentials
			shared.GET("/generic-credentials", s.listSharedGenericCredentials)
			shared.POST("/generic-credentials", s.createSharedGenericCredentials)
			shared.GET("/generic-credentials/:generic-credentials", s.getSharedGenericCredentials)
			shared.PUT("/generic-credentials/:generic-credentials", s.updateSharedGenericCredentials)
			shared.PATCH("/generic-credentials/:generic-credentials", s.patchSharedGenericCredentials)
			shared.DELETE("/generic-credentials/:generic-credentials", s.deleteSharedGenericCredentials)

			// ConfigMaps
			shared.GET("/configmaps", s.listSharedConfigMaps)
			shared.POST("/configmaps", s.createSharedConfigMap)
			shared.GET("/configmaps/:configmap", s.getSharedConfigMap)
			shared.PUT("/configmaps/:configmap", s.updateSharedConfigMap)
			shared.PATCH("/configmaps/:configmap", s.patchSharedConfigMap)
			shared.DELETE("/configmaps/:configmap", s.deleteSharedConfigMap)
		}

		// =====================================================================
		// Projects (/v1beta1/projects)
		// =====================================================================
		v1beta1.GET("/projects", defaultLimit, s.listProjects)
		project := v1beta1.Group("/projects/:project")
		project.Use(defaultLimit)
		project.Use(s.projectExistsMiddleware())
		{
			// Project CRUD
			project.GET("", s.getProject)
			project.DELETE("", s.deleteProject)

			// Project Configuration
			project.GET("/config", s.getProjectConfig)
			project.POST("/config/refresh", s.refreshProjectConfig)
			project.DELETE("/config", s.deleteProjectConfig)

			// Events
			project.GET("/events", s.listProjectEvents)

			// -----------------------------------------------------------------
			// Core Resources
			// -----------------------------------------------------------------

			// Stages
			project.GET("/stages", s.listStages)
			project.GET("/stages/:stage", s.getStage)
			project.POST("/stages/:stage/refresh", s.refreshStage)
			project.DELETE("/stages/:stage", s.deleteStage)
			// Stage Promotions
			project.POST("/stages/:stage/promotions", s.promoteToStage)
			project.POST("/stages/:stage/promotions/downstream", s.promoteDownstream)
			// Stage Verification
			project.POST("/stages/:stage/verification", s.reverify)
			project.POST("/stages/:stage/verification/abort", s.abortVerification)

			// Warehouses
			project.GET("/warehouses", s.listWarehouses)
			project.GET("/warehouses/:warehouse", s.getWarehouse)
			project.POST("/warehouses/:warehouse/refresh", s.refreshWarehouse)
			project.DELETE("/warehouses/:warehouse", s.deleteWarehouse)

			// Freight
			project.GET("/freight", s.queryFreight)
			project.GET("/freight/:freight-name-or-alias", s.getFreight)
			project.POST("/freight/:freight-name-or-alias/approve", s.approveFreight)
			project.PATCH("/freight/:freight-name-or-alias/alias", s.patchFreightAliasHandler)
			project.DELETE("/freight/:freight-name-or-alias", s.deleteFreight)

			// Promotions
			project.GET("/promotions", s.listPromotions)
			project.GET("/promotions/:promotion", s.getPromotion)
			project.POST("/promotions/:promotion/refresh", s.refreshPromotion)
			project.POST("/promotions/:promotion/abort", s.abortPromotion)

			// Promotion Tasks
			project.GET("/promotion-tasks", s.listPromotionTasks)
			project.GET("/promotion-tasks/:promotion-task", s.getPromotionTask)

			// -----------------------------------------------------------------
			// Verifications (Argo Rollouts Integration)
			// -----------------------------------------------------------------

			// Analysis Templates
			project.GET("/analysis-templates", s.listAnalysisTemplates)
			project.GET("/analysis-templates/:analysis-template", s.getAnalysisTemplate)
			project.DELETE("/analysis-templates/:analysis-template", s.deleteAnalysisTemplate)

			// Analysis Runs
			project.GET("/analysis-runs/:analysis-run", s.getAnalysisRun)
			project.GET("/analysis-runs/:analysis-run/logs", s.getAnalysisRunLogs)

			// -----------------------------------------------------------------
			// Generic Config
			// -----------------------------------------------------------------

			// ConfigMaps
			project.GET("/configmaps", s.listProjectConfigMaps)
			project.POST("/configmaps", s.createProjectConfigMap)
			project.GET("/configmaps/:configmap", s.getProjectConfigMap)
			project.PUT("/configmaps/:configmap", s.updateProjectConfigMap)
			project.PATCH("/configmaps/:configmap", s.patchProjectConfigMap)
			project.DELETE("/configmaps/:configmap", s.deleteProjectConfigMap)

			// Images
			project.GET("/images", s.listImages)

			// -----------------------------------------------------------------
			// Credentials
			// -----------------------------------------------------------------

			// Repo Credentials
			project.GET("/repo-credentials", s.listProjectRepoCredentials)
			project.POST("/repo-credentials", s.createProjectRepoCredentials)
			project.GET("/repo-credentials/:repo-credentials", s.getProjectRepoCredentials)
			project.PUT("/repo-credentials/:repo-credentials", s.updateProjectRepoCredentials)
			project.PATCH("/repo-credentials/:repo-credentials", s.patchProjectRepoCredentials)
			project.DELETE("/repo-credentials/:repo-credentials", s.deleteProjectRepoCredentials)

			// Generic Credentials
			project.GET("/generic-credentials", s.listProjectGenericCredentials)
			project.POST("/generic-credentials", s.createProjectGenericCredentials)
			project.GET("/generic-credentials/:generic-credentials", s.getProjectGenericCredentials)
			project.PUT("/generic-credentials/:generic-credentials", s.updateProjectGenericCredentials)
			project.PATCH("/generic-credentials/:generic-credentials", s.patchProjectGenericCredentials)
			project.DELETE("/generic-credentials/:generic-credentials", s.deleteProjectGenericCredentials)

			// -----------------------------------------------------------------
			// RBAC
			// -----------------------------------------------------------------

			// Roles
			project.GET("/roles", s.listProjectRoles)
			project.POST("/roles", s.createProjectRole)
			project.GET("/roles/:role", s.getProjectRole)
			project.PUT("/roles/:role", s.updateRole)
			project.DELETE("/roles/:role", s.deleteProjectRole)
			project.POST("/roles/:role/api-tokens", s.createProjectAPIToken)

			// Role Grants/Revocations
			project.POST("/roles/grants", s.grant)
			project.POST("/roles/revocations", s.revoke)

			// API Tokens
			project.GET("/api-tokens", s.listProjectAPITokens)
			project.GET("/api-tokens/:apitoken", s.getProjectAPIToken)
			project.DELETE("/api-tokens/:apitoken", s.deleteProjectAPIToken)
		}
	}

	return router
}

func (s *server) handleError(c *gin.Context) {
	c.Next()
	if len(c.Errors) > 0 {
		err := c.Errors.Last().Err

		// Check for MaxBytesError (body too large)
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request body too large"})
			return
		}

		var httpErr *libhttp.HTTPError
		if ok := errors.As(err, &httpErr); ok {
			if code := httpErr.Code(); code == http.StatusInternalServerError {
				logging.LoggerFromContext(c.Request.Context()).
					Error(err, "internal server error")
				c.JSON(
					http.StatusInternalServerError,
					gin.H{"error": "internal server error"},
				)
			}
			c.JSON(httpErr.Code(), gin.H{"error": httpErr.Error()})
			return
		}
		var statusErr *apierrors.StatusError
		if ok := errors.As(err, &statusErr); ok {
			c.JSON(int(statusErr.Status().Code), gin.H{"error": err.Error()})
			return
		}
		_ = c.Error(libhttp.Error(
			errors.New("internal server error"),
			http.StatusInternalServerError,
		))
	}
}
