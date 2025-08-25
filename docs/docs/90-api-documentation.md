# API Documentation
<a name="top"></a>



<a name="api_service_v1alpha1_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/service/v1alpha1/service.proto



<a name="akuity-io-kargo-service-v1alpha1-AbortPromotionRequest"></a>

### AbortPromotionRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| name | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-AbortPromotionResponse"></a>

### AbortPromotionResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-AbortVerificationRequest"></a>

### AbortVerificationRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| stage | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-AbortVerificationResponse"></a>

### AbortVerificationResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-AdminLoginRequest"></a>

### AdminLoginRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| password | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-AdminLoginResponse"></a>

### AdminLoginResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| id_token | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-ApproveFreightRequest"></a>

### ApproveFreightRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| name | [string](#string) |   |
| alias | [string](#string) |   |
| stage | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-ApproveFreightResponse"></a>

### ApproveFreightResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-ArgoCDShard"></a>

### ArgoCDShard



| Field | Type | Description |
| ----- | ---- | ----------- |
| url | [string](#string) |   |
| namespace | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-Claims"></a>

### Claims



| Field | Type | Description |
| ----- | ---- | ----------- |
| claims | [github.com.akuity.kargo.api.rbac.v1alpha1.Claim](#github-com-akuity-kargo-api-rbac-v1alpha1-Claim) |  Note: oneof and repeated do not work together |






<a name="akuity-io-kargo-service-v1alpha1-ComponentVersions"></a>

### ComponentVersions



| Field | Type | Description |
| ----- | ---- | ----------- |
| server | [VersionInfo](#akuity-io-kargo-service-v1alpha1-VersionInfo) |   |
| cli | [VersionInfo](#akuity-io-kargo-service-v1alpha1-VersionInfo) |   |






<a name="akuity-io-kargo-service-v1alpha1-CreateClusterSecretRequest"></a>

### CreateClusterSecretRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| name | [string](#string) |   |
| data | [CreateClusterSecretRequest.DataEntry](#akuity-io-kargo-service-v1alpha1-CreateClusterSecretRequest-DataEntry) |   |






<a name="akuity-io-kargo-service-v1alpha1-CreateClusterSecretRequest-DataEntry"></a>

### CreateClusterSecretRequest.DataEntry



| Field | Type | Description |
| ----- | ---- | ----------- |
| key | [string](#string) |   |
| value | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-CreateClusterSecretResponse"></a>

### CreateClusterSecretResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| secret | k8s.io.api.core.v1.Secret |   |






<a name="akuity-io-kargo-service-v1alpha1-CreateCredentialsRequest"></a>

### CreateCredentialsRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| name | [string](#string) |   |
| description | [string](#string) |   |
| type | [string](#string) |  type is git, helm, image |
| repo_url | [string](#string) |   |
| repo_url_is_regex | [bool](#bool) |   |
| username | [string](#string) |   |
| password | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-CreateCredentialsResponse"></a>

### CreateCredentialsResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| credentials | k8s.io.api.core.v1.Secret |   |






<a name="akuity-io-kargo-service-v1alpha1-CreateOrUpdateResourceRequest"></a>

### CreateOrUpdateResourceRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| manifest | [bytes](#bytes) |   |






<a name="akuity-io-kargo-service-v1alpha1-CreateOrUpdateResourceResponse"></a>

### CreateOrUpdateResourceResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| results | [CreateOrUpdateResourceResult](#akuity-io-kargo-service-v1alpha1-CreateOrUpdateResourceResult) |   |






<a name="akuity-io-kargo-service-v1alpha1-CreateOrUpdateResourceResult"></a>

### CreateOrUpdateResourceResult



| Field | Type | Description |
| ----- | ---- | ----------- |
| created_resource_manifest | [bytes](#bytes) |   |
| updated_resource_manifest | [bytes](#bytes) |   |
| error | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-CreateProjectSecretRequest"></a>

### CreateProjectSecretRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| name | [string](#string) |   |
| description | [string](#string) |   |
| data | [CreateProjectSecretRequest.DataEntry](#akuity-io-kargo-service-v1alpha1-CreateProjectSecretRequest-DataEntry) |   |






<a name="akuity-io-kargo-service-v1alpha1-CreateProjectSecretRequest-DataEntry"></a>

### CreateProjectSecretRequest.DataEntry



| Field | Type | Description |
| ----- | ---- | ----------- |
| key | [string](#string) |   |
| value | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-CreateProjectSecretResponse"></a>

### CreateProjectSecretResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| secret | k8s.io.api.core.v1.Secret |   |






<a name="akuity-io-kargo-service-v1alpha1-CreateResourceRequest"></a>

### CreateResourceRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| manifest | [bytes](#bytes) |   |






<a name="akuity-io-kargo-service-v1alpha1-CreateResourceResponse"></a>

### CreateResourceResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| results | [CreateResourceResult](#akuity-io-kargo-service-v1alpha1-CreateResourceResult) |   |






<a name="akuity-io-kargo-service-v1alpha1-CreateResourceResult"></a>

### CreateResourceResult



| Field | Type | Description |
| ----- | ---- | ----------- |
| created_resource_manifest | [bytes](#bytes) |   |
| error | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-CreateRoleRequest"></a>

### CreateRoleRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| role | [github.com.akuity.kargo.api.rbac.v1alpha1.Role](#github-com-akuity-kargo-api-rbac-v1alpha1-Role) |   |






<a name="akuity-io-kargo-service-v1alpha1-CreateRoleResponse"></a>

### CreateRoleResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| role | [github.com.akuity.kargo.api.rbac.v1alpha1.Role](#github-com-akuity-kargo-api-rbac-v1alpha1-Role) |   |






<a name="akuity-io-kargo-service-v1alpha1-DeleteAnalysisTemplateRequest"></a>

### DeleteAnalysisTemplateRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| name | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-DeleteAnalysisTemplateResponse"></a>

### DeleteAnalysisTemplateResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-DeleteClusterAnalysisTemplateRequest"></a>

### DeleteClusterAnalysisTemplateRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| name | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-DeleteClusterAnalysisTemplateResponse"></a>

### DeleteClusterAnalysisTemplateResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-DeleteClusterConfigRequest"></a>

### DeleteClusterConfigRequest
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-DeleteClusterConfigResponse"></a>

### DeleteClusterConfigResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-DeleteClusterSecretRequest"></a>

### DeleteClusterSecretRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| name | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-DeleteClusterSecretResponse"></a>

### DeleteClusterSecretResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-DeleteCredentialsRequest"></a>

### DeleteCredentialsRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| name | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-DeleteCredentialsResponse"></a>

### DeleteCredentialsResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-DeleteFreightRequest"></a>

### DeleteFreightRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| name | [string](#string) |   |
| alias | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-DeleteFreightResponse"></a>

### DeleteFreightResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-DeleteProjectConfigRequest"></a>

### DeleteProjectConfigRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-DeleteProjectConfigResponse"></a>

### DeleteProjectConfigResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-DeleteProjectRequest"></a>

### DeleteProjectRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| name | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-DeleteProjectResponse"></a>

### DeleteProjectResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-DeleteProjectSecretRequest"></a>

### DeleteProjectSecretRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| name | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-DeleteProjectSecretResponse"></a>

### DeleteProjectSecretResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-DeleteResourceRequest"></a>

### DeleteResourceRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| manifest | [bytes](#bytes) |   |






<a name="akuity-io-kargo-service-v1alpha1-DeleteResourceResponse"></a>

### DeleteResourceResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| results | [DeleteResourceResult](#akuity-io-kargo-service-v1alpha1-DeleteResourceResult) |   |






<a name="akuity-io-kargo-service-v1alpha1-DeleteResourceResult"></a>

### DeleteResourceResult



| Field | Type | Description |
| ----- | ---- | ----------- |
| deleted_resource_manifest | [bytes](#bytes) |   |
| error | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-DeleteRoleRequest"></a>

### DeleteRoleRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| name | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-DeleteRoleResponse"></a>

### DeleteRoleResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-DeleteStageRequest"></a>

### DeleteStageRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| name | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-DeleteStageResponse"></a>

### DeleteStageResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-DeleteWarehouseRequest"></a>

### DeleteWarehouseRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| name | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-DeleteWarehouseResponse"></a>

### DeleteWarehouseResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-FreightList"></a>

### FreightList



| Field | Type | Description |
| ----- | ---- | ----------- |
| freight | [github.com.akuity.kargo.api.v1alpha1.Freight](#github-com-akuity-kargo-api-v1alpha1-Freight) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetAnalysisRunLogsRequest"></a>

### GetAnalysisRunLogsRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| namespace | [string](#string) |   |
| name | [string](#string) |   |
| metric_name | [string](#string) |   |
| container_name | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetAnalysisRunLogsResponse"></a>

### GetAnalysisRunLogsResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| chunk | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetAnalysisRunRequest"></a>

### GetAnalysisRunRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| namespace | [string](#string) |   |
| name | [string](#string) |   |
| format | [RawFormat](#akuity-io-kargo-service-v1alpha1-RawFormat) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetAnalysisRunResponse"></a>

### GetAnalysisRunResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| analysis_run | [github.com.akuity.kargo.api.stubs.rollouts.v1alpha1.AnalysisRun](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisRun) |   |
| raw | [bytes](#bytes) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetAnalysisTemplateRequest"></a>

### GetAnalysisTemplateRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| name | [string](#string) |   |
| format | [RawFormat](#akuity-io-kargo-service-v1alpha1-RawFormat) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetAnalysisTemplateResponse"></a>

### GetAnalysisTemplateResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| analysis_template | [github.com.akuity.kargo.api.stubs.rollouts.v1alpha1.AnalysisTemplate](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisTemplate) |   |
| raw | [bytes](#bytes) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetClusterAnalysisTemplateRequest"></a>

### GetClusterAnalysisTemplateRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| name | [string](#string) |   |
| format | [RawFormat](#akuity-io-kargo-service-v1alpha1-RawFormat) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetClusterAnalysisTemplateResponse"></a>

### GetClusterAnalysisTemplateResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| cluster_analysis_template | [github.com.akuity.kargo.api.stubs.rollouts.v1alpha1.ClusterAnalysisTemplate](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-ClusterAnalysisTemplate) |   |
| raw | [bytes](#bytes) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetClusterConfigRequest"></a>

### GetClusterConfigRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| format | [RawFormat](#akuity-io-kargo-service-v1alpha1-RawFormat) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetClusterConfigResponse"></a>

### GetClusterConfigResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| cluster_config | [github.com.akuity.kargo.api.v1alpha1.ClusterConfig](#github-com-akuity-kargo-api-v1alpha1-ClusterConfig) |   |
| raw | [bytes](#bytes) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetClusterPromotionTaskRequest"></a>

### GetClusterPromotionTaskRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| name | [string](#string) |   |
| format | [RawFormat](#akuity-io-kargo-service-v1alpha1-RawFormat) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetClusterPromotionTaskResponse"></a>

### GetClusterPromotionTaskResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| promotion_task | [github.com.akuity.kargo.api.v1alpha1.ClusterPromotionTask](#github-com-akuity-kargo-api-v1alpha1-ClusterPromotionTask) |   |
| raw | [bytes](#bytes) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetConfigMapRequest"></a>

### GetConfigMapRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| name | [string](#string) |   |
| format | [RawFormat](#akuity-io-kargo-service-v1alpha1-RawFormat) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetConfigMapResponse"></a>

### GetConfigMapResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| config_map | k8s.io.api.core.v1.ConfigMap |   |
| raw | [bytes](#bytes) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetConfigRequest"></a>

### GetConfigRequest







<a name="akuity-io-kargo-service-v1alpha1-GetConfigResponse"></a>

### GetConfigResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| argocd_shards | [GetConfigResponse.ArgocdShardsEntry](#akuity-io-kargo-service-v1alpha1-GetConfigResponse-ArgocdShardsEntry) |   |
| secret_management_enabled | [bool](#bool) |   |
| cluster_secrets_namespace | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetConfigResponse-ArgocdShardsEntry"></a>

### GetConfigResponse.ArgocdShardsEntry



| Field | Type | Description |
| ----- | ---- | ----------- |
| key | [string](#string) |   |
| value | [ArgoCDShard](#akuity-io-kargo-service-v1alpha1-ArgoCDShard) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetCredentialsRequest"></a>

### GetCredentialsRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| name | [string](#string) |   |
| format | [RawFormat](#akuity-io-kargo-service-v1alpha1-RawFormat) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetCredentialsResponse"></a>

### GetCredentialsResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| credentials | k8s.io.api.core.v1.Secret |   |
| raw | [bytes](#bytes) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetFreightRequest"></a>

### GetFreightRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| name | [string](#string) |   |
| alias | [string](#string) |   |
| format | [RawFormat](#akuity-io-kargo-service-v1alpha1-RawFormat) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetFreightResponse"></a>

### GetFreightResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| freight | [github.com.akuity.kargo.api.v1alpha1.Freight](#github-com-akuity-kargo-api-v1alpha1-Freight) |   |
| raw | [bytes](#bytes) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetProjectConfigRequest"></a>

### GetProjectConfigRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| format | [RawFormat](#akuity-io-kargo-service-v1alpha1-RawFormat) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetProjectConfigResponse"></a>

### GetProjectConfigResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| project_config | [github.com.akuity.kargo.api.v1alpha1.ProjectConfig](#github-com-akuity-kargo-api-v1alpha1-ProjectConfig) |   |
| raw | [bytes](#bytes) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetProjectRequest"></a>

### GetProjectRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| name | [string](#string) |   |
| format | [RawFormat](#akuity-io-kargo-service-v1alpha1-RawFormat) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetProjectResponse"></a>

### GetProjectResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [github.com.akuity.kargo.api.v1alpha1.Project](#github-com-akuity-kargo-api-v1alpha1-Project) |   |
| raw | [bytes](#bytes) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetPromotionRequest"></a>

### GetPromotionRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| name | [string](#string) |   |
| format | [RawFormat](#akuity-io-kargo-service-v1alpha1-RawFormat) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetPromotionResponse"></a>

### GetPromotionResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| promotion | [github.com.akuity.kargo.api.v1alpha1.Promotion](#github-com-akuity-kargo-api-v1alpha1-Promotion) |   |
| raw | [bytes](#bytes) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetPromotionTaskRequest"></a>

### GetPromotionTaskRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| name | [string](#string) |   |
| format | [RawFormat](#akuity-io-kargo-service-v1alpha1-RawFormat) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetPromotionTaskResponse"></a>

### GetPromotionTaskResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| promotion_task | [github.com.akuity.kargo.api.v1alpha1.PromotionTask](#github-com-akuity-kargo-api-v1alpha1-PromotionTask) |   |
| raw | [bytes](#bytes) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetPublicConfigRequest"></a>

### GetPublicConfigRequest







<a name="akuity-io-kargo-service-v1alpha1-GetPublicConfigResponse"></a>

### GetPublicConfigResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| oidc_config | [OIDCConfig](#akuity-io-kargo-service-v1alpha1-OIDCConfig) |   |
| admin_account_enabled | [bool](#bool) |   |
| skip_auth | [bool](#bool) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetRoleRequest"></a>

### GetRoleRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| name | [string](#string) |   |
| as_resources | [bool](#bool) |   |
| format | [RawFormat](#akuity-io-kargo-service-v1alpha1-RawFormat) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetRoleResponse"></a>

### GetRoleResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| role | [github.com.akuity.kargo.api.rbac.v1alpha1.Role](#github-com-akuity-kargo-api-rbac-v1alpha1-Role) |   |
| resources | [github.com.akuity.kargo.api.rbac.v1alpha1.RoleResources](#github-com-akuity-kargo-api-rbac-v1alpha1-RoleResources) |   |
| raw | [bytes](#bytes) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetStageRequest"></a>

### GetStageRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| name | [string](#string) |   |
| format | [RawFormat](#akuity-io-kargo-service-v1alpha1-RawFormat) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetStageResponse"></a>

### GetStageResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| stage | [github.com.akuity.kargo.api.v1alpha1.Stage](#github-com-akuity-kargo-api-v1alpha1-Stage) |   |
| raw | [bytes](#bytes) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetVersionInfoRequest"></a>

### GetVersionInfoRequest







<a name="akuity-io-kargo-service-v1alpha1-GetVersionInfoResponse"></a>

### GetVersionInfoResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| version_info | [VersionInfo](#akuity-io-kargo-service-v1alpha1-VersionInfo) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetWarehouseRequest"></a>

### GetWarehouseRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| name | [string](#string) |   |
| format | [RawFormat](#akuity-io-kargo-service-v1alpha1-RawFormat) |   |






<a name="akuity-io-kargo-service-v1alpha1-GetWarehouseResponse"></a>

### GetWarehouseResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| warehouse | [github.com.akuity.kargo.api.v1alpha1.Warehouse](#github-com-akuity-kargo-api-v1alpha1-Warehouse) |   |
| raw | [bytes](#bytes) |   |






<a name="akuity-io-kargo-service-v1alpha1-GrantRequest"></a>

### GrantRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| role | [string](#string) |   |
| user_claims | [Claims](#akuity-io-kargo-service-v1alpha1-Claims) |   |
| resource_details | [github.com.akuity.kargo.api.rbac.v1alpha1.ResourceDetails](#github-com-akuity-kargo-api-rbac-v1alpha1-ResourceDetails) |   |






<a name="akuity-io-kargo-service-v1alpha1-GrantResponse"></a>

### GrantResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| role | [github.com.akuity.kargo.api.rbac.v1alpha1.Role](#github-com-akuity-kargo-api-rbac-v1alpha1-Role) |   |






<a name="akuity-io-kargo-service-v1alpha1-ImageStageMap"></a>

### ImageStageMap



| Field | Type | Description |
| ----- | ---- | ----------- |
| stages | [ImageStageMap.StagesEntry](#akuity-io-kargo-service-v1alpha1-ImageStageMap-StagesEntry) |  stages maps stage names to the order which an image was promoted to that stage |






<a name="akuity-io-kargo-service-v1alpha1-ImageStageMap-StagesEntry"></a>

### ImageStageMap.StagesEntry



| Field | Type | Description |
| ----- | ---- | ----------- |
| key | [string](#string) |   |
| value | [int32](#int32) |   |






<a name="akuity-io-kargo-service-v1alpha1-ListAnalysisTemplatesRequest"></a>

### ListAnalysisTemplatesRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-ListAnalysisTemplatesResponse"></a>

### ListAnalysisTemplatesResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| analysis_templates | [github.com.akuity.kargo.api.stubs.rollouts.v1alpha1.AnalysisTemplate](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisTemplate) |   |






<a name="akuity-io-kargo-service-v1alpha1-ListClusterAnalysisTemplatesRequest"></a>

### ListClusterAnalysisTemplatesRequest







<a name="akuity-io-kargo-service-v1alpha1-ListClusterAnalysisTemplatesResponse"></a>

### ListClusterAnalysisTemplatesResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| cluster_analysis_templates | [github.com.akuity.kargo.api.stubs.rollouts.v1alpha1.ClusterAnalysisTemplate](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-ClusterAnalysisTemplate) |   |






<a name="akuity-io-kargo-service-v1alpha1-ListClusterPromotionTasksRequest"></a>

### ListClusterPromotionTasksRequest







<a name="akuity-io-kargo-service-v1alpha1-ListClusterPromotionTasksResponse"></a>

### ListClusterPromotionTasksResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| cluster_promotion_tasks | [github.com.akuity.kargo.api.v1alpha1.ClusterPromotionTask](#github-com-akuity-kargo-api-v1alpha1-ClusterPromotionTask) |   |






<a name="akuity-io-kargo-service-v1alpha1-ListClusterSecretsRequest"></a>

### ListClusterSecretsRequest
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-ListClusterSecretsResponse"></a>

### ListClusterSecretsResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| secrets | k8s.io.api.core.v1.Secret |   |






<a name="akuity-io-kargo-service-v1alpha1-ListConfigMapsRequest"></a>

### ListConfigMapsRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-ListConfigMapsResponse"></a>

### ListConfigMapsResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| config_maps | k8s.io.api.core.v1.ConfigMap |   |






<a name="akuity-io-kargo-service-v1alpha1-ListCredentialsRequest"></a>

### ListCredentialsRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-ListCredentialsResponse"></a>

### ListCredentialsResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| credentials | k8s.io.api.core.v1.Secret |   |






<a name="akuity-io-kargo-service-v1alpha1-ListImagesRequest"></a>

### ListImagesRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-ListImagesResponse"></a>

### ListImagesResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| images | [ListImagesResponse.ImagesEntry](#akuity-io-kargo-service-v1alpha1-ListImagesResponse-ImagesEntry) |  images maps image repository names to their tags |






<a name="akuity-io-kargo-service-v1alpha1-ListImagesResponse-ImagesEntry"></a>

### ListImagesResponse.ImagesEntry



| Field | Type | Description |
| ----- | ---- | ----------- |
| key | [string](#string) |   |
| value | [TagMap](#akuity-io-kargo-service-v1alpha1-TagMap) |   |






<a name="akuity-io-kargo-service-v1alpha1-ListProjectEventsRequest"></a>

### ListProjectEventsRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-ListProjectEventsResponse"></a>

### ListProjectEventsResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| events | k8s.io.api.core.v1.Event |   |






<a name="akuity-io-kargo-service-v1alpha1-ListProjectSecretsRequest"></a>

### ListProjectSecretsRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-ListProjectSecretsResponse"></a>

### ListProjectSecretsResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| secrets | k8s.io.api.core.v1.Secret |   |






<a name="akuity-io-kargo-service-v1alpha1-ListProjectsRequest"></a>

### ListProjectsRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| page_size | [int32](#int32) |   |
| page | [int32](#int32) |   |
| filter | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-ListProjectsResponse"></a>

### ListProjectsResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| projects | [github.com.akuity.kargo.api.v1alpha1.Project](#github-com-akuity-kargo-api-v1alpha1-Project) |   |
| total | [int32](#int32) |   |






<a name="akuity-io-kargo-service-v1alpha1-ListPromotionTasksRequest"></a>

### ListPromotionTasksRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-ListPromotionTasksResponse"></a>

### ListPromotionTasksResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| promotion_tasks | [github.com.akuity.kargo.api.v1alpha1.PromotionTask](#github-com-akuity-kargo-api-v1alpha1-PromotionTask) |   |






<a name="akuity-io-kargo-service-v1alpha1-ListPromotionsRequest"></a>

### ListPromotionsRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| stage | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-ListPromotionsResponse"></a>

### ListPromotionsResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| promotions | [github.com.akuity.kargo.api.v1alpha1.Promotion](#github-com-akuity-kargo-api-v1alpha1-Promotion) |   |






<a name="akuity-io-kargo-service-v1alpha1-ListRolesRequest"></a>

### ListRolesRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| as_resources | [bool](#bool) |   |






<a name="akuity-io-kargo-service-v1alpha1-ListRolesResponse"></a>

### ListRolesResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| roles | [github.com.akuity.kargo.api.rbac.v1alpha1.Role](#github-com-akuity-kargo-api-rbac-v1alpha1-Role) |  Note: oneof and repeated do not work together |
| resources | [github.com.akuity.kargo.api.rbac.v1alpha1.RoleResources](#github-com-akuity-kargo-api-rbac-v1alpha1-RoleResources) |   |






<a name="akuity-io-kargo-service-v1alpha1-ListStagesRequest"></a>

### ListStagesRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-ListStagesResponse"></a>

### ListStagesResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| stages | [github.com.akuity.kargo.api.v1alpha1.Stage](#github-com-akuity-kargo-api-v1alpha1-Stage) |   |






<a name="akuity-io-kargo-service-v1alpha1-ListWarehousesRequest"></a>

### ListWarehousesRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-ListWarehousesResponse"></a>

### ListWarehousesResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| warehouses | [github.com.akuity.kargo.api.v1alpha1.Warehouse](#github-com-akuity-kargo-api-v1alpha1-Warehouse) |   |






<a name="akuity-io-kargo-service-v1alpha1-OIDCConfig"></a>

### OIDCConfig



| Field | Type | Description |
| ----- | ---- | ----------- |
| issuer_url | [string](#string) |   |
| client_id | [string](#string) |   |
| scopes | [string](#string) |   |
| cli_client_id | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-PromoteDownstreamRequest"></a>

### PromoteDownstreamRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| stage | [string](#string) |   |
| freight | [string](#string) |   |
| freight_alias | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-PromoteDownstreamResponse"></a>

### PromoteDownstreamResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| promotions | [github.com.akuity.kargo.api.v1alpha1.Promotion](#github-com-akuity-kargo-api-v1alpha1-Promotion) |   |






<a name="akuity-io-kargo-service-v1alpha1-PromoteToStageRequest"></a>

### PromoteToStageRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| stage | [string](#string) |   |
| freight | [string](#string) |   |
| freight_alias | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-PromoteToStageResponse"></a>

### PromoteToStageResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| promotion | [github.com.akuity.kargo.api.v1alpha1.Promotion](#github-com-akuity-kargo-api-v1alpha1-Promotion) |   |






<a name="akuity-io-kargo-service-v1alpha1-QueryFreightRequest"></a>

### QueryFreightRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| stage | [string](#string) |   |
| group_by | [string](#string) |   |
| group | [string](#string) |   |
| order_by | [string](#string) |   |
| reverse | [bool](#bool) |   |
| origins | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-QueryFreightResponse"></a>

### QueryFreightResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| groups | [QueryFreightResponse.GroupsEntry](#akuity-io-kargo-service-v1alpha1-QueryFreightResponse-GroupsEntry) |   |






<a name="akuity-io-kargo-service-v1alpha1-QueryFreightResponse-GroupsEntry"></a>

### QueryFreightResponse.GroupsEntry



| Field | Type | Description |
| ----- | ---- | ----------- |
| key | [string](#string) |   |
| value | [FreightList](#akuity-io-kargo-service-v1alpha1-FreightList) |   |






<a name="akuity-io-kargo-service-v1alpha1-RefreshClusterConfigRequest"></a>

### RefreshClusterConfigRequest
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-RefreshClusterConfigResponse"></a>

### RefreshClusterConfigResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| cluster_config | [github.com.akuity.kargo.api.v1alpha1.ClusterConfig](#github-com-akuity-kargo-api-v1alpha1-ClusterConfig) |   |






<a name="akuity-io-kargo-service-v1alpha1-RefreshProjectConfigRequest"></a>

### RefreshProjectConfigRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-RefreshProjectConfigResponse"></a>

### RefreshProjectConfigResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| project_config | [github.com.akuity.kargo.api.v1alpha1.ProjectConfig](#github-com-akuity-kargo-api-v1alpha1-ProjectConfig) |   |






<a name="akuity-io-kargo-service-v1alpha1-RefreshStageRequest"></a>

### RefreshStageRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| name | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-RefreshStageResponse"></a>

### RefreshStageResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| stage | [github.com.akuity.kargo.api.v1alpha1.Stage](#github-com-akuity-kargo-api-v1alpha1-Stage) |   |






<a name="akuity-io-kargo-service-v1alpha1-RefreshWarehouseRequest"></a>

### RefreshWarehouseRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| name | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-RefreshWarehouseResponse"></a>

### RefreshWarehouseResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| warehouse | [github.com.akuity.kargo.api.v1alpha1.Warehouse](#github-com-akuity-kargo-api-v1alpha1-Warehouse) |   |






<a name="akuity-io-kargo-service-v1alpha1-ReverifyRequest"></a>

### ReverifyRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| stage | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-ReverifyResponse"></a>

### ReverifyResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-RevokeRequest"></a>

### RevokeRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| role | [string](#string) |   |
| user_claims | [Claims](#akuity-io-kargo-service-v1alpha1-Claims) |   |
| resource_details | [github.com.akuity.kargo.api.rbac.v1alpha1.ResourceDetails](#github-com-akuity-kargo-api-rbac-v1alpha1-ResourceDetails) |   |






<a name="akuity-io-kargo-service-v1alpha1-RevokeResponse"></a>

### RevokeResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| role | [github.com.akuity.kargo.api.rbac.v1alpha1.Role](#github-com-akuity-kargo-api-rbac-v1alpha1-Role) |   |






<a name="akuity-io-kargo-service-v1alpha1-TagMap"></a>

### TagMap



| Field | Type | Description |
| ----- | ---- | ----------- |
| tags | [TagMap.TagsEntry](#akuity-io-kargo-service-v1alpha1-TagMap-TagsEntry) |  tags maps image tag names to stages which have previously used that tag |






<a name="akuity-io-kargo-service-v1alpha1-TagMap-TagsEntry"></a>

### TagMap.TagsEntry



| Field | Type | Description |
| ----- | ---- | ----------- |
| key | [string](#string) |   |
| value | [ImageStageMap](#akuity-io-kargo-service-v1alpha1-ImageStageMap) |   |






<a name="akuity-io-kargo-service-v1alpha1-UpdateClusterSecretRequest"></a>

### UpdateClusterSecretRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| name | [string](#string) |   |
| data | [UpdateClusterSecretRequest.DataEntry](#akuity-io-kargo-service-v1alpha1-UpdateClusterSecretRequest-DataEntry) |   |






<a name="akuity-io-kargo-service-v1alpha1-UpdateClusterSecretRequest-DataEntry"></a>

### UpdateClusterSecretRequest.DataEntry



| Field | Type | Description |
| ----- | ---- | ----------- |
| key | [string](#string) |   |
| value | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-UpdateClusterSecretResponse"></a>

### UpdateClusterSecretResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| secret | k8s.io.api.core.v1.Secret |   |






<a name="akuity-io-kargo-service-v1alpha1-UpdateCredentialsRequest"></a>

### UpdateCredentialsRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| name | [string](#string) |   |
| description | [string](#string) |   |
| type | [string](#string) |   |
| repo_url | [string](#string) |   |
| repo_url_is_regex | [bool](#bool) |   |
| username | [string](#string) |   |
| password | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-UpdateCredentialsResponse"></a>

### UpdateCredentialsResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| credentials | k8s.io.api.core.v1.Secret |   |






<a name="akuity-io-kargo-service-v1alpha1-UpdateFreightAliasRequest"></a>

### UpdateFreightAliasRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| name | [string](#string) |   |
| old_alias | [string](#string) |   |
| new_alias | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-UpdateFreightAliasResponse"></a>

### UpdateFreightAliasResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-UpdateProjectSecretRequest"></a>

### UpdateProjectSecretRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| name | [string](#string) |   |
| description | [string](#string) |   |
| data | [UpdateProjectSecretRequest.DataEntry](#akuity-io-kargo-service-v1alpha1-UpdateProjectSecretRequest-DataEntry) |   |






<a name="akuity-io-kargo-service-v1alpha1-UpdateProjectSecretRequest-DataEntry"></a>

### UpdateProjectSecretRequest.DataEntry



| Field | Type | Description |
| ----- | ---- | ----------- |
| key | [string](#string) |   |
| value | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-UpdateProjectSecretResponse"></a>

### UpdateProjectSecretResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| secret | k8s.io.api.core.v1.Secret |   |






<a name="akuity-io-kargo-service-v1alpha1-UpdateResourceRequest"></a>

### UpdateResourceRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| manifest | [bytes](#bytes) |   |






<a name="akuity-io-kargo-service-v1alpha1-UpdateResourceResponse"></a>

### UpdateResourceResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| results | [UpdateResourceResult](#akuity-io-kargo-service-v1alpha1-UpdateResourceResult) |   |






<a name="akuity-io-kargo-service-v1alpha1-UpdateResourceResult"></a>

### UpdateResourceResult



| Field | Type | Description |
| ----- | ---- | ----------- |
| updated_resource_manifest | [bytes](#bytes) |   |
| error | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-UpdateRoleRequest"></a>

### UpdateRoleRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| role | [github.com.akuity.kargo.api.rbac.v1alpha1.Role](#github-com-akuity-kargo-api-rbac-v1alpha1-Role) |   |






<a name="akuity-io-kargo-service-v1alpha1-UpdateRoleResponse"></a>

### UpdateRoleResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| role | [github.com.akuity.kargo.api.rbac.v1alpha1.Role](#github-com-akuity-kargo-api-rbac-v1alpha1-Role) |   |






<a name="akuity-io-kargo-service-v1alpha1-VersionInfo"></a>

### VersionInfo



| Field | Type | Description |
| ----- | ---- | ----------- |
| version | [string](#string) |   |
| git_commit | [string](#string) |   |
| git_tree_dirty | [bool](#bool) |   |
| build_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |   |
| go_version | [string](#string) |   |
| compiler | [string](#string) |   |
| platform | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-WatchClusterConfigRequest"></a>

### WatchClusterConfigRequest
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-WatchClusterConfigResponse"></a>

### WatchClusterConfigResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| cluster_config | [github.com.akuity.kargo.api.v1alpha1.ClusterConfig](#github-com-akuity-kargo-api-v1alpha1-ClusterConfig) |   |
| type | [string](#string) |  ADDED / MODIFIED / DELETED |






<a name="akuity-io-kargo-service-v1alpha1-WatchFreightRequest"></a>

### WatchFreightRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-WatchFreightResponse"></a>

### WatchFreightResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| freight | [github.com.akuity.kargo.api.v1alpha1.Freight](#github-com-akuity-kargo-api-v1alpha1-Freight) |   |
| type | [string](#string) |  ADDED / MODIFIED / DELETED |






<a name="akuity-io-kargo-service-v1alpha1-WatchProjectConfigRequest"></a>

### WatchProjectConfigRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-WatchProjectConfigResponse"></a>

### WatchProjectConfigResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| project_config | [github.com.akuity.kargo.api.v1alpha1.ProjectConfig](#github-com-akuity-kargo-api-v1alpha1-ProjectConfig) |   |
| type | [string](#string) |  ADDED / MODIFIED / DELETED |






<a name="akuity-io-kargo-service-v1alpha1-WatchPromotionRequest"></a>

### WatchPromotionRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| name | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-WatchPromotionResponse"></a>

### WatchPromotionResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| promotion | [github.com.akuity.kargo.api.v1alpha1.Promotion](#github-com-akuity-kargo-api-v1alpha1-Promotion) |   |
| type | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-WatchPromotionsRequest"></a>

### WatchPromotionsRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| stage | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-WatchPromotionsResponse"></a>

### WatchPromotionsResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| promotion | [github.com.akuity.kargo.api.v1alpha1.Promotion](#github-com-akuity-kargo-api-v1alpha1-Promotion) |   |
| type | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-WatchStagesRequest"></a>

### WatchStagesRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| name | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-WatchStagesResponse"></a>

### WatchStagesResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| stage | [github.com.akuity.kargo.api.v1alpha1.Stage](#github-com-akuity-kargo-api-v1alpha1-Stage) |   |
| type | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-WatchWarehousesRequest"></a>

### WatchWarehousesRequest



| Field | Type | Description |
| ----- | ---- | ----------- |
| project | [string](#string) |   |
| name | [string](#string) |   |






<a name="akuity-io-kargo-service-v1alpha1-WatchWarehousesResponse"></a>

### WatchWarehousesResponse



| Field | Type | Description |
| ----- | ---- | ----------- |
| warehouse | [github.com.akuity.kargo.api.v1alpha1.Warehouse](#github-com-akuity-kargo-api-v1alpha1-Warehouse) |   |
| type | [string](#string) |   |





 <!-- end messages -->


<a name="akuity-io-kargo-service-v1alpha1-RawFormat"></a>

### RawFormat


| Name | Number | Description |
| ---- | ------ | ----------- |
| RAW_FORMAT_UNSPECIFIED | 0 |  |
| RAW_FORMAT_JSON | 1 |  |
| RAW_FORMAT_YAML | 2 |  |


 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="akuity-io-kargo-service-v1alpha1-KargoService"></a>

### KargoService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetVersionInfo | [GetVersionInfoRequest](#akuity-io-kargo-service-v1alpha1-GetVersionInfoRequest) | [GetVersionInfoResponse](#akuity-io-kargo-service-v1alpha1-GetVersionInfoResponse) |  |
| GetConfig | [GetConfigRequest](#akuity-io-kargo-service-v1alpha1-GetConfigRequest) | [GetConfigResponse](#akuity-io-kargo-service-v1alpha1-GetConfigResponse) |  |
| GetPublicConfig | [GetPublicConfigRequest](#akuity-io-kargo-service-v1alpha1-GetPublicConfigRequest) | [GetPublicConfigResponse](#akuity-io-kargo-service-v1alpha1-GetPublicConfigResponse) |  |
| AdminLogin | [AdminLoginRequest](#akuity-io-kargo-service-v1alpha1-AdminLoginRequest) | [AdminLoginResponse](#akuity-io-kargo-service-v1alpha1-AdminLoginResponse) |  |
| CreateResource | [CreateResourceRequest](#akuity-io-kargo-service-v1alpha1-CreateResourceRequest) | [CreateResourceResponse](#akuity-io-kargo-service-v1alpha1-CreateResourceResponse) | TODO(devholic): Add ApplyResource API rpc ApplyResource(ApplyResourceRequest) returns (ApplyResourceRequest); |
| CreateOrUpdateResource | [CreateOrUpdateResourceRequest](#akuity-io-kargo-service-v1alpha1-CreateOrUpdateResourceRequest) | [CreateOrUpdateResourceResponse](#akuity-io-kargo-service-v1alpha1-CreateOrUpdateResourceResponse) |  |
| UpdateResource | [UpdateResourceRequest](#akuity-io-kargo-service-v1alpha1-UpdateResourceRequest) | [UpdateResourceResponse](#akuity-io-kargo-service-v1alpha1-UpdateResourceResponse) |  |
| DeleteResource | [DeleteResourceRequest](#akuity-io-kargo-service-v1alpha1-DeleteResourceRequest) | [DeleteResourceResponse](#akuity-io-kargo-service-v1alpha1-DeleteResourceResponse) |  |
| ListStages | [ListStagesRequest](#akuity-io-kargo-service-v1alpha1-ListStagesRequest) | [ListStagesResponse](#akuity-io-kargo-service-v1alpha1-ListStagesResponse) |  |
| ListImages | [ListImagesRequest](#akuity-io-kargo-service-v1alpha1-ListImagesRequest) | [ListImagesResponse](#akuity-io-kargo-service-v1alpha1-ListImagesResponse) |  |
| GetStage | [GetStageRequest](#akuity-io-kargo-service-v1alpha1-GetStageRequest) | [GetStageResponse](#akuity-io-kargo-service-v1alpha1-GetStageResponse) |  |
| WatchStages | [WatchStagesRequest](#akuity-io-kargo-service-v1alpha1-WatchStagesRequest) | [WatchStagesResponse](#akuity-io-kargo-service-v1alpha1-WatchStagesResponse) stream |  |
| DeleteStage | [DeleteStageRequest](#akuity-io-kargo-service-v1alpha1-DeleteStageRequest) | [DeleteStageResponse](#akuity-io-kargo-service-v1alpha1-DeleteStageResponse) |  |
| RefreshStage | [RefreshStageRequest](#akuity-io-kargo-service-v1alpha1-RefreshStageRequest) | [RefreshStageResponse](#akuity-io-kargo-service-v1alpha1-RefreshStageResponse) |  |
| GetClusterConfig | [GetClusterConfigRequest](#akuity-io-kargo-service-v1alpha1-GetClusterConfigRequest) | [GetClusterConfigResponse](#akuity-io-kargo-service-v1alpha1-GetClusterConfigResponse) |  |
| DeleteClusterConfig | [DeleteClusterConfigRequest](#akuity-io-kargo-service-v1alpha1-DeleteClusterConfigRequest) | [DeleteClusterConfigResponse](#akuity-io-kargo-service-v1alpha1-DeleteClusterConfigResponse) |  |
| WatchClusterConfig | [WatchClusterConfigRequest](#akuity-io-kargo-service-v1alpha1-WatchClusterConfigRequest) | [WatchClusterConfigResponse](#akuity-io-kargo-service-v1alpha1-WatchClusterConfigResponse) stream |  |
| RefreshClusterConfig | [RefreshClusterConfigRequest](#akuity-io-kargo-service-v1alpha1-RefreshClusterConfigRequest) | [RefreshClusterConfigResponse](#akuity-io-kargo-service-v1alpha1-RefreshClusterConfigResponse) |  |
| ListPromotions | [ListPromotionsRequest](#akuity-io-kargo-service-v1alpha1-ListPromotionsRequest) | [ListPromotionsResponse](#akuity-io-kargo-service-v1alpha1-ListPromotionsResponse) |  |
| WatchPromotions | [WatchPromotionsRequest](#akuity-io-kargo-service-v1alpha1-WatchPromotionsRequest) | [WatchPromotionsResponse](#akuity-io-kargo-service-v1alpha1-WatchPromotionsResponse) stream |  |
| GetPromotion | [GetPromotionRequest](#akuity-io-kargo-service-v1alpha1-GetPromotionRequest) | [GetPromotionResponse](#akuity-io-kargo-service-v1alpha1-GetPromotionResponse) |  |
| WatchPromotion | [WatchPromotionRequest](#akuity-io-kargo-service-v1alpha1-WatchPromotionRequest) | [WatchPromotionResponse](#akuity-io-kargo-service-v1alpha1-WatchPromotionResponse) stream |  |
| AbortPromotion | [AbortPromotionRequest](#akuity-io-kargo-service-v1alpha1-AbortPromotionRequest) | [AbortPromotionResponse](#akuity-io-kargo-service-v1alpha1-AbortPromotionResponse) |  |
| DeleteProject | [DeleteProjectRequest](#akuity-io-kargo-service-v1alpha1-DeleteProjectRequest) | [DeleteProjectResponse](#akuity-io-kargo-service-v1alpha1-DeleteProjectResponse) |  |
| GetProject | [GetProjectRequest](#akuity-io-kargo-service-v1alpha1-GetProjectRequest) | [GetProjectResponse](#akuity-io-kargo-service-v1alpha1-GetProjectResponse) |  |
| ListProjects | [ListProjectsRequest](#akuity-io-kargo-service-v1alpha1-ListProjectsRequest) | [ListProjectsResponse](#akuity-io-kargo-service-v1alpha1-ListProjectsResponse) |  |
| GetProjectConfig | [GetProjectConfigRequest](#akuity-io-kargo-service-v1alpha1-GetProjectConfigRequest) | [GetProjectConfigResponse](#akuity-io-kargo-service-v1alpha1-GetProjectConfigResponse) | ProjectConfig APIs |
| DeleteProjectConfig | [DeleteProjectConfigRequest](#akuity-io-kargo-service-v1alpha1-DeleteProjectConfigRequest) | [DeleteProjectConfigResponse](#akuity-io-kargo-service-v1alpha1-DeleteProjectConfigResponse) |  |
| WatchProjectConfig | [WatchProjectConfigRequest](#akuity-io-kargo-service-v1alpha1-WatchProjectConfigRequest) | [WatchProjectConfigResponse](#akuity-io-kargo-service-v1alpha1-WatchProjectConfigResponse) stream |  |
| RefreshProjectConfig | [RefreshProjectConfigRequest](#akuity-io-kargo-service-v1alpha1-RefreshProjectConfigRequest) | [RefreshProjectConfigResponse](#akuity-io-kargo-service-v1alpha1-RefreshProjectConfigResponse) |  |
| ApproveFreight | [ApproveFreightRequest](#akuity-io-kargo-service-v1alpha1-ApproveFreightRequest) | [ApproveFreightResponse](#akuity-io-kargo-service-v1alpha1-ApproveFreightResponse) |  |
| DeleteFreight | [DeleteFreightRequest](#akuity-io-kargo-service-v1alpha1-DeleteFreightRequest) | [DeleteFreightResponse](#akuity-io-kargo-service-v1alpha1-DeleteFreightResponse) |  |
| GetFreight | [GetFreightRequest](#akuity-io-kargo-service-v1alpha1-GetFreightRequest) | [GetFreightResponse](#akuity-io-kargo-service-v1alpha1-GetFreightResponse) |  |
| WatchFreight | [WatchFreightRequest](#akuity-io-kargo-service-v1alpha1-WatchFreightRequest) | [WatchFreightResponse](#akuity-io-kargo-service-v1alpha1-WatchFreightResponse) stream |  |
| PromoteToStage | [PromoteToStageRequest](#akuity-io-kargo-service-v1alpha1-PromoteToStageRequest) | [PromoteToStageResponse](#akuity-io-kargo-service-v1alpha1-PromoteToStageResponse) |  |
| PromoteDownstream | [PromoteDownstreamRequest](#akuity-io-kargo-service-v1alpha1-PromoteDownstreamRequest) | [PromoteDownstreamResponse](#akuity-io-kargo-service-v1alpha1-PromoteDownstreamResponse) |  |
| QueryFreight | [QueryFreightRequest](#akuity-io-kargo-service-v1alpha1-QueryFreightRequest) | [QueryFreightResponse](#akuity-io-kargo-service-v1alpha1-QueryFreightResponse) |  |
| UpdateFreightAlias | [UpdateFreightAliasRequest](#akuity-io-kargo-service-v1alpha1-UpdateFreightAliasRequest) | [UpdateFreightAliasResponse](#akuity-io-kargo-service-v1alpha1-UpdateFreightAliasResponse) |  |
| Reverify | [ReverifyRequest](#akuity-io-kargo-service-v1alpha1-ReverifyRequest) | [ReverifyResponse](#akuity-io-kargo-service-v1alpha1-ReverifyResponse) |  |
| AbortVerification | [AbortVerificationRequest](#akuity-io-kargo-service-v1alpha1-AbortVerificationRequest) | [AbortVerificationResponse](#akuity-io-kargo-service-v1alpha1-AbortVerificationResponse) |  |
| ListWarehouses | [ListWarehousesRequest](#akuity-io-kargo-service-v1alpha1-ListWarehousesRequest) | [ListWarehousesResponse](#akuity-io-kargo-service-v1alpha1-ListWarehousesResponse) |  |
| GetWarehouse | [GetWarehouseRequest](#akuity-io-kargo-service-v1alpha1-GetWarehouseRequest) | [GetWarehouseResponse](#akuity-io-kargo-service-v1alpha1-GetWarehouseResponse) |  |
| WatchWarehouses | [WatchWarehousesRequest](#akuity-io-kargo-service-v1alpha1-WatchWarehousesRequest) | [WatchWarehousesResponse](#akuity-io-kargo-service-v1alpha1-WatchWarehousesResponse) stream |  |
| DeleteWarehouse | [DeleteWarehouseRequest](#akuity-io-kargo-service-v1alpha1-DeleteWarehouseRequest) | [DeleteWarehouseResponse](#akuity-io-kargo-service-v1alpha1-DeleteWarehouseResponse) |  |
| RefreshWarehouse | [RefreshWarehouseRequest](#akuity-io-kargo-service-v1alpha1-RefreshWarehouseRequest) | [RefreshWarehouseResponse](#akuity-io-kargo-service-v1alpha1-RefreshWarehouseResponse) |  |
| CreateCredentials | [CreateCredentialsRequest](#akuity-io-kargo-service-v1alpha1-CreateCredentialsRequest) | [CreateCredentialsResponse](#akuity-io-kargo-service-v1alpha1-CreateCredentialsResponse) |  |
| DeleteCredentials | [DeleteCredentialsRequest](#akuity-io-kargo-service-v1alpha1-DeleteCredentialsRequest) | [DeleteCredentialsResponse](#akuity-io-kargo-service-v1alpha1-DeleteCredentialsResponse) |  |
| GetCredentials | [GetCredentialsRequest](#akuity-io-kargo-service-v1alpha1-GetCredentialsRequest) | [GetCredentialsResponse](#akuity-io-kargo-service-v1alpha1-GetCredentialsResponse) |  |
| ListCredentials | [ListCredentialsRequest](#akuity-io-kargo-service-v1alpha1-ListCredentialsRequest) | [ListCredentialsResponse](#akuity-io-kargo-service-v1alpha1-ListCredentialsResponse) |  |
| UpdateCredentials | [UpdateCredentialsRequest](#akuity-io-kargo-service-v1alpha1-UpdateCredentialsRequest) | [UpdateCredentialsResponse](#akuity-io-kargo-service-v1alpha1-UpdateCredentialsResponse) |  |
| ListProjectSecrets | [ListProjectSecretsRequest](#akuity-io-kargo-service-v1alpha1-ListProjectSecretsRequest) | [ListProjectSecretsResponse](#akuity-io-kargo-service-v1alpha1-ListProjectSecretsResponse) |  |
| CreateProjectSecret | [CreateProjectSecretRequest](#akuity-io-kargo-service-v1alpha1-CreateProjectSecretRequest) | [CreateProjectSecretResponse](#akuity-io-kargo-service-v1alpha1-CreateProjectSecretResponse) |  |
| UpdateProjectSecret | [UpdateProjectSecretRequest](#akuity-io-kargo-service-v1alpha1-UpdateProjectSecretRequest) | [UpdateProjectSecretResponse](#akuity-io-kargo-service-v1alpha1-UpdateProjectSecretResponse) |  |
| DeleteProjectSecret | [DeleteProjectSecretRequest](#akuity-io-kargo-service-v1alpha1-DeleteProjectSecretRequest) | [DeleteProjectSecretResponse](#akuity-io-kargo-service-v1alpha1-DeleteProjectSecretResponse) |  |
| ListConfigMaps | [ListConfigMapsRequest](#akuity-io-kargo-service-v1alpha1-ListConfigMapsRequest) | [ListConfigMapsResponse](#akuity-io-kargo-service-v1alpha1-ListConfigMapsResponse) |  |
| GetConfigMap | [GetConfigMapRequest](#akuity-io-kargo-service-v1alpha1-GetConfigMapRequest) | [GetConfigMapResponse](#akuity-io-kargo-service-v1alpha1-GetConfigMapResponse) |  |
| ListAnalysisTemplates | [ListAnalysisTemplatesRequest](#akuity-io-kargo-service-v1alpha1-ListAnalysisTemplatesRequest) | [ListAnalysisTemplatesResponse](#akuity-io-kargo-service-v1alpha1-ListAnalysisTemplatesResponse) |  |
| GetAnalysisTemplate | [GetAnalysisTemplateRequest](#akuity-io-kargo-service-v1alpha1-GetAnalysisTemplateRequest) | [GetAnalysisTemplateResponse](#akuity-io-kargo-service-v1alpha1-GetAnalysisTemplateResponse) |  |
| DeleteAnalysisTemplate | [DeleteAnalysisTemplateRequest](#akuity-io-kargo-service-v1alpha1-DeleteAnalysisTemplateRequest) | [DeleteAnalysisTemplateResponse](#akuity-io-kargo-service-v1alpha1-DeleteAnalysisTemplateResponse) |  |
| ListClusterAnalysisTemplates | [ListClusterAnalysisTemplatesRequest](#akuity-io-kargo-service-v1alpha1-ListClusterAnalysisTemplatesRequest) | [ListClusterAnalysisTemplatesResponse](#akuity-io-kargo-service-v1alpha1-ListClusterAnalysisTemplatesResponse) |  |
| GetClusterAnalysisTemplate | [GetClusterAnalysisTemplateRequest](#akuity-io-kargo-service-v1alpha1-GetClusterAnalysisTemplateRequest) | [GetClusterAnalysisTemplateResponse](#akuity-io-kargo-service-v1alpha1-GetClusterAnalysisTemplateResponse) |  |
| DeleteClusterAnalysisTemplate | [DeleteClusterAnalysisTemplateRequest](#akuity-io-kargo-service-v1alpha1-DeleteClusterAnalysisTemplateRequest) | [DeleteClusterAnalysisTemplateResponse](#akuity-io-kargo-service-v1alpha1-DeleteClusterAnalysisTemplateResponse) |  |
| GetAnalysisRun | [GetAnalysisRunRequest](#akuity-io-kargo-service-v1alpha1-GetAnalysisRunRequest) | [GetAnalysisRunResponse](#akuity-io-kargo-service-v1alpha1-GetAnalysisRunResponse) |  |
| GetAnalysisRunLogs | [GetAnalysisRunLogsRequest](#akuity-io-kargo-service-v1alpha1-GetAnalysisRunLogsRequest) | [GetAnalysisRunLogsResponse](#akuity-io-kargo-service-v1alpha1-GetAnalysisRunLogsResponse) stream |  |
| ListProjectEvents | [ListProjectEventsRequest](#akuity-io-kargo-service-v1alpha1-ListProjectEventsRequest) | [ListProjectEventsResponse](#akuity-io-kargo-service-v1alpha1-ListProjectEventsResponse) |  |
| ListPromotionTasks | [ListPromotionTasksRequest](#akuity-io-kargo-service-v1alpha1-ListPromotionTasksRequest) | [ListPromotionTasksResponse](#akuity-io-kargo-service-v1alpha1-ListPromotionTasksResponse) |  |
| ListClusterPromotionTasks | [ListClusterPromotionTasksRequest](#akuity-io-kargo-service-v1alpha1-ListClusterPromotionTasksRequest) | [ListClusterPromotionTasksResponse](#akuity-io-kargo-service-v1alpha1-ListClusterPromotionTasksResponse) |  |
| GetPromotionTask | [GetPromotionTaskRequest](#akuity-io-kargo-service-v1alpha1-GetPromotionTaskRequest) | [GetPromotionTaskResponse](#akuity-io-kargo-service-v1alpha1-GetPromotionTaskResponse) |  |
| GetClusterPromotionTask | [GetClusterPromotionTaskRequest](#akuity-io-kargo-service-v1alpha1-GetClusterPromotionTaskRequest) | [GetClusterPromotionTaskResponse](#akuity-io-kargo-service-v1alpha1-GetClusterPromotionTaskResponse) |  |
| CreateRole | [CreateRoleRequest](#akuity-io-kargo-service-v1alpha1-CreateRoleRequest) | [CreateRoleResponse](#akuity-io-kargo-service-v1alpha1-CreateRoleResponse) |  |
| DeleteRole | [DeleteRoleRequest](#akuity-io-kargo-service-v1alpha1-DeleteRoleRequest) | [DeleteRoleResponse](#akuity-io-kargo-service-v1alpha1-DeleteRoleResponse) |  |
| GetRole | [GetRoleRequest](#akuity-io-kargo-service-v1alpha1-GetRoleRequest) | [GetRoleResponse](#akuity-io-kargo-service-v1alpha1-GetRoleResponse) |  |
| Grant | [GrantRequest](#akuity-io-kargo-service-v1alpha1-GrantRequest) | [GrantResponse](#akuity-io-kargo-service-v1alpha1-GrantResponse) |  |
| ListRoles | [ListRolesRequest](#akuity-io-kargo-service-v1alpha1-ListRolesRequest) | [ListRolesResponse](#akuity-io-kargo-service-v1alpha1-ListRolesResponse) |  |
| Revoke | [RevokeRequest](#akuity-io-kargo-service-v1alpha1-RevokeRequest) | [RevokeResponse](#akuity-io-kargo-service-v1alpha1-RevokeResponse) |  |
| UpdateRole | [UpdateRoleRequest](#akuity-io-kargo-service-v1alpha1-UpdateRoleRequest) | [UpdateRoleResponse](#akuity-io-kargo-service-v1alpha1-UpdateRoleResponse) |  |
| ListClusterSecrets | [ListClusterSecretsRequest](#akuity-io-kargo-service-v1alpha1-ListClusterSecretsRequest) | [ListClusterSecretsResponse](#akuity-io-kargo-service-v1alpha1-ListClusterSecretsResponse) | Cluster Secrets APIs |
| CreateClusterSecret | [CreateClusterSecretRequest](#akuity-io-kargo-service-v1alpha1-CreateClusterSecretRequest) | [CreateClusterSecretResponse](#akuity-io-kargo-service-v1alpha1-CreateClusterSecretResponse) |  |
| UpdateClusterSecret | [UpdateClusterSecretRequest](#akuity-io-kargo-service-v1alpha1-UpdateClusterSecretRequest) | [UpdateClusterSecretResponse](#akuity-io-kargo-service-v1alpha1-UpdateClusterSecretResponse) |  |
| DeleteClusterSecret | [DeleteClusterSecretRequest](#akuity-io-kargo-service-v1alpha1-DeleteClusterSecretRequest) | [DeleteClusterSecretResponse](#akuity-io-kargo-service-v1alpha1-DeleteClusterSecretResponse) |  |

 <!-- end services -->



<a name="api_rbac_v1alpha1_generated-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/rbac/v1alpha1/generated.proto



<a name="github-com-akuity-kargo-api-rbac-v1alpha1-Claim"></a>

### Claim



| Field | Type | Description |
| ----- | ---- | ----------- |
| name | [string](#string) |   |
| values | [string](#string) |   |






<a name="github-com-akuity-kargo-api-rbac-v1alpha1-ResourceDetails"></a>

### ResourceDetails



| Field | Type | Description |
| ----- | ---- | ----------- |
| resourceType | [string](#string) |   |
| resourceName | [string](#string) |   |
| verbs | [string](#string) |   |






<a name="github-com-akuity-kargo-api-rbac-v1alpha1-Role"></a>

### Role
+kubebuilder:object:root=true


| Field | Type | Description |
| ----- | ---- | ----------- |
| metadata | k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta |   |
| kargoManaged | [bool](#bool) |   |
| claims | [Claim](#github-com-akuity-kargo-api-rbac-v1alpha1-Claim) |   |
| rules | k8s.io.api.rbac.v1.PolicyRule |   |






<a name="github-com-akuity-kargo-api-rbac-v1alpha1-RoleResources"></a>

### RoleResources
+kubebuilder:object:root=true


| Field | Type | Description |
| ----- | ---- | ----------- |
| metadata | k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta |   |
| serviceAccount | k8s.io.api.core.v1.ServiceAccount |   |
| roles | k8s.io.api.rbac.v1.Role |   |
| roleBindings | k8s.io.api.rbac.v1.RoleBinding |   |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="api_v1alpha1_generated-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha1/generated.proto



<a name="github-com-akuity-kargo-api-v1alpha1-AnalysisRunArgument"></a>

### AnalysisRunArgument
AnalysisRunArgument represents an argument to be added to an AnalysisRun.


| Field | Type | Description |
| ----- | ---- | ----------- |
| name | [string](#string) |  Name is the name of the argument.  +kubebuilder:validation:Required |
| value | [string](#string) |  Value is the value of the argument.  +kubebuilder:validation:Required |






<a name="github-com-akuity-kargo-api-v1alpha1-AnalysisRunMetadata"></a>

### AnalysisRunMetadata
AnalysisRunMetadata contains optional metadata that should be applied to all
AnalysisRuns.


| Field | Type | Description |
| ----- | ---- | ----------- |
| labels | [AnalysisRunMetadata.LabelsEntry](#github-com-akuity-kargo-api-v1alpha1-AnalysisRunMetadata-LabelsEntry) |  Additional labels to apply to an AnalysisRun. |
| annotations | [AnalysisRunMetadata.AnnotationsEntry](#github-com-akuity-kargo-api-v1alpha1-AnalysisRunMetadata-AnnotationsEntry) |  Additional annotations to apply to an AnalysisRun. |






<a name="github-com-akuity-kargo-api-v1alpha1-AnalysisRunMetadata-AnnotationsEntry"></a>

### AnalysisRunMetadata.AnnotationsEntry



| Field | Type | Description |
| ----- | ---- | ----------- |
| key | [string](#string) |   |
| value | [string](#string) |   |






<a name="github-com-akuity-kargo-api-v1alpha1-AnalysisRunMetadata-LabelsEntry"></a>

### AnalysisRunMetadata.LabelsEntry



| Field | Type | Description |
| ----- | ---- | ----------- |
| key | [string](#string) |   |
| value | [string](#string) |   |






<a name="github-com-akuity-kargo-api-v1alpha1-AnalysisRunReference"></a>

### AnalysisRunReference
AnalysisRunReference is a reference to an AnalysisRun.


| Field | Type | Description |
| ----- | ---- | ----------- |
| namespace | [string](#string) |  Namespace is the namespace of the AnalysisRun. |
| name | [string](#string) |  Name is the name of the AnalysisRun. |
| phase | [string](#string) |  Phase is the last observed phase of the AnalysisRun referenced by Name. |






<a name="github-com-akuity-kargo-api-v1alpha1-AnalysisTemplateReference"></a>

### AnalysisTemplateReference
AnalysisTemplateReference is a reference to an AnalysisTemplate.


| Field | Type | Description |
| ----- | ---- | ----------- |
| name | [string](#string) |  Name is the name of the AnalysisTemplate in the same project/namespace as the Stage.  +kubebuilder:validation:Required |
| kind | [string](#string) |  Kind is the type of the AnalysisTemplate. Can be either AnalysisTemplate or ClusterAnalysisTemplate, default is AnalysisTemplate.  +kubebuilder:validation:Optional +kubebuilder:validation:Enum=AnalysisTemplate;ClusterAnalysisTemplate |






<a name="github-com-akuity-kargo-api-v1alpha1-ApprovedStage"></a>

### ApprovedStage
ApprovedStage describes a Stage for which Freight has been (manually)
approved.


| Field | Type | Description |
| ----- | ---- | ----------- |
| approvedAt | k8s.io.apimachinery.pkg.apis.meta.v1.Time |  ApprovedAt is the time at which the Freight was approved for the Stage. |






<a name="github-com-akuity-kargo-api-v1alpha1-ArgoCDAppHealthStatus"></a>

### ArgoCDAppHealthStatus
ArgoCDAppHealthStatus describes the health of an ArgoCD Application.


| Field | Type | Description |
| ----- | ---- | ----------- |
| status | [string](#string) |   |
| message | [string](#string) |   |






<a name="github-com-akuity-kargo-api-v1alpha1-ArgoCDAppStatus"></a>

### ArgoCDAppStatus
ArgoCDAppStatus describes the current state of a single ArgoCD Application.


| Field | Type | Description |
| ----- | ---- | ----------- |
| namespace | [string](#string) |  Namespace is the namespace of the ArgoCD Application. |
| name | [string](#string) |  Name is the name of the ArgoCD Application. |
| healthStatus | [ArgoCDAppHealthStatus](#github-com-akuity-kargo-api-v1alpha1-ArgoCDAppHealthStatus) |  HealthStatus is the health of the ArgoCD Application. |
| syncStatus | [ArgoCDAppSyncStatus](#github-com-akuity-kargo-api-v1alpha1-ArgoCDAppSyncStatus) |  SyncStatus is the sync status of the ArgoCD Application. |






<a name="github-com-akuity-kargo-api-v1alpha1-ArgoCDAppSyncStatus"></a>

### ArgoCDAppSyncStatus
ArgoCDAppSyncStatus describes the sync status of an ArgoCD Application.


| Field | Type | Description |
| ----- | ---- | ----------- |
| status | [string](#string) |   |
| revision | [string](#string) |   |
| revisions | [string](#string) |   |






<a name="github-com-akuity-kargo-api-v1alpha1-ArtifactoryWebhookReceiverConfig"></a>

### ArtifactoryWebhookReceiverConfig
ArtifactoryWebhookReceiverConfig describes a webhook receiver that is
compatible with JFrog Artifactory payloads.


| Field | Type | Description |
| ----- | ---- | ----------- |
| secretRef | k8s.io.api.core.v1.LocalObjectReference |  SecretRef contains a reference to a Secret. For Project-scoped webhook receivers, the referenced Secret must be in the same namespace as the ProjectConfig.  For cluster-scoped webhook receivers, the referenced Secret must be in the designated "cluster Secrets" namespace.  The Secret's data map is expected to contain a `secret-token` key whose value is the shared secret used to authenticate the webhook requests sent by JFrog Artifactory. For more information please refer to the JFrog Artifactory documentation:   https://jfrog.com/help/r/jfrog-platform-administration-documentation/webhooks  +kubebuilder:validation:Required |






<a name="github-com-akuity-kargo-api-v1alpha1-AzureWebhookReceiverConfig"></a>

### AzureWebhookReceiverConfig
AzureWebhookReceiverConfig describes a webhook receiver that is compatible
with Azure Container Registry (ACR) and Azure DevOps payloads.


| Field | Type | Description |
| ----- | ---- | ----------- |
| secretRef | k8s.io.api.core.v1.LocalObjectReference |  SecretRef contains a reference to a Secret. For Project-scoped webhook receivers, the referenced Secret must be in the same namespace as the ProjectConfig.  For cluster-scoped webhook receivers, the referenced Secret must be in the designated "cluster Secrets" namespace.  The Secret's data map is expected to contain a `secret` key whose value does NOT need to be shared directly with Azure when registering a webhook. It is used only by Kargo to create a complex, hard-to-guess URL, which implicitly serves as a shared secret. For more information about Azure webhooks, please refer to the Azure documentation:   Azure Container Registry: 	https://learn.microsoft.com/en-us/azure/container-registry/container-registry-repositories   Azure DevOps: 	http://learn.microsoft.com/en-us/azure/devops/service-hooks/services/webhooks?view=azure-devops  +kubebuilder:validation:Required |






<a name="github-com-akuity-kargo-api-v1alpha1-BitbucketWebhookReceiverConfig"></a>

### BitbucketWebhookReceiverConfig
BitbucketWebhookReceiverConfig describes a webhook receiver that is
compatible with Bitbucket payloads.


| Field | Type | Description |
| ----- | ---- | ----------- |
| secretRef | k8s.io.api.core.v1.LocalObjectReference |  SecretRef contains a reference to a Secret. For Project-scoped webhook receivers, the referenced Secret must be in the same namespace as the ProjectConfig.  For cluster-scoped webhook receivers, the referenced Secret must be in the designated "cluster Secrets" namespace.  The Secret's data map is expected to contain a `secret` key whose value is the shared secret used to authenticate the webhook requests sent by Bitbucket. For more information please refer to the Bitbucket documentation:   https://support.atlassian.com/bitbucket-cloud/docs/manage-webhooks/  +kubebuilder:validation:Required |






<a name="github-com-akuity-kargo-api-v1alpha1-Chart"></a>

### Chart
Chart describes a specific version of a Helm chart.


| Field | Type | Description |
| ----- | ---- | ----------- |
| repoURL | [string](#string) |  RepoURL specifies the URL of a Helm chart repository. Classic chart repositories (using HTTP/S) can contain differently named charts. When this field points to such a repository, the Name field will specify the name of the chart within the repository. In the case of a repository within an OCI registry, the URL implicitly points to a specific chart and the Name field will be empty. |
| name | [string](#string) |  Name specifies the name of the chart. |
| version | [string](#string) |  Version specifies a particular version of the chart. |






<a name="github-com-akuity-kargo-api-v1alpha1-ChartDiscoveryResult"></a>

### ChartDiscoveryResult
ChartDiscoveryResult represents the result of a chart discovery operation for
a ChartSubscription.


| Field | Type | Description |
| ----- | ---- | ----------- |
| repoURL | [string](#string) |  RepoURL is the repository URL of the Helm chart, as specified in the ChartSubscription.  +kubebuilder:validation:MinLength=1 |
| name | [string](#string) |  Name is the name of the Helm chart, as specified in the ChartSubscription. |
| semverConstraint | [string](#string) |  SemverConstraint is the constraint for which versions were discovered. This field is optional, and only populated if the ChartSubscription specifies a SemverConstraint. |
| versions | [string](#string) |  Versions is a list of versions discovered by the Warehouse for the ChartSubscription. An empty list indicates that the discovery operation was successful, but no versions matching the ChartSubscription criteria were found.  +optional |






<a name="github-com-akuity-kargo-api-v1alpha1-ChartSubscription"></a>

### ChartSubscription
ChartSubscription defines a subscription to a Helm chart repository.


| Field | Type | Description |
| ----- | ---- | ----------- |
| repoURL | [string](#string) |  RepoURL specifies the URL of a Helm chart repository. It may be a classic chart repository (using HTTP/S) OR a repository within an OCI registry. Classic chart repositories can contain differently named charts. When this field points to such a repository, the Name field MUST also be used to specify the name of the desired chart within that repository. In the case of a repository within an OCI registry, the URL implicitly points to a specific chart and the Name field MUST NOT be used. The RepoURL field is required.  +kubebuilder:validation:MinLength=1 +kubebuilder:validation:Pattern=`^(((https?)|(oci))://)(\w\d\.\-+)(:\d+)?(/.*)*$` +akuity:test-kubebuilder-pattern=HelmRepoURL |
| name | [string](#string) |  Name specifies the name of a Helm chart to subscribe to within a classic chart repository specified by the RepoURL field. This field is required when the RepoURL field points to a classic chart repository and MUST otherwise be empty. |
| semverConstraint | [string](#string) |  SemverConstraint specifies constraints on what new chart versions are permissible. This field is optional. When left unspecified, there will be no constraints, which means the latest version of the chart will always be used. Care should be taken with leaving this field unspecified, as it can lead to the unanticipated rollout of breaking changes. More info: https://github.com/masterminds/semver#checking-version-constraints  +kubebuilder:validation:Optional |
| discoveryLimit | [int32](#int32) |  DiscoveryLimit is an optional limit on the number of chart versions that can be discovered for this subscription. The limit is applied after filtering charts based on the SemverConstraint field. When left unspecified, the field is implicitly treated as if its value were "20". The upper limit for this field is 100.  +kubebuilder:validation:Minimum=1 +kubebuilder:validation:Maximum=100 +kubebuilder:default=20 |






<a name="github-com-akuity-kargo-api-v1alpha1-ClusterConfig"></a>

### ClusterConfig
ClusterConfig is a resource type that describes cluster-level Kargo
configuration.


| Field | Type | Description |
| ----- | ---- | ----------- |
| metadata | k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta |   |
| spec | [ClusterConfigSpec](#github-com-akuity-kargo-api-v1alpha1-ClusterConfigSpec) |  Spec describes the configuration of a cluster. |
| status | [ClusterConfigStatus](#github-com-akuity-kargo-api-v1alpha1-ClusterConfigStatus) |  Status describes the current status of a ClusterConfig. |






<a name="github-com-akuity-kargo-api-v1alpha1-ClusterConfigList"></a>

### ClusterConfigList
ClusterConfigList contains a list of ClusterConfigs.


| Field | Type | Description |
| ----- | ---- | ----------- |
| metadata | k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta |   |
| items | [ClusterConfig](#github-com-akuity-kargo-api-v1alpha1-ClusterConfig) |   |






<a name="github-com-akuity-kargo-api-v1alpha1-ClusterConfigSpec"></a>

### ClusterConfigSpec
ClusterConfigSpec describes cluster-level Kargo configuration.


| Field | Type | Description |
| ----- | ---- | ----------- |
| webhookReceivers | [WebhookReceiverConfig](#github-com-akuity-kargo-api-v1alpha1-WebhookReceiverConfig) |  WebhookReceivers describes cluster-scoped webhook receivers used for processing events from various external platforms |






<a name="github-com-akuity-kargo-api-v1alpha1-ClusterConfigStatus"></a>

### ClusterConfigStatus
ClusterConfigStatus describes the current status of a ClusterConfig.


| Field | Type | Description |
| ----- | ---- | ----------- |
| conditions | k8s.io.apimachinery.pkg.apis.meta.v1.Condition |  Conditions contains the last observations of the ClusterConfig's current state.  +patchMergeKey=type +patchStrategy=merge +listType=map +listMapKey=type |
| observedGeneration | [int64](#int64) |  ObservedGeneration represents the .metadata.generation that this ClusterConfig was reconciled against. |
| lastHandledRefresh | [string](#string) |  LastHandledRefresh holds the value of the most recent AnnotationKeyRefresh annotation that was handled by the controller. This field can be used to determine whether the request to refresh the resource has been handled. +optional |
| webhookReceivers | [WebhookReceiverDetails](#github-com-akuity-kargo-api-v1alpha1-WebhookReceiverDetails) |  WebhookReceivers describes the status of cluster-scoped webhook receivers. |






<a name="github-com-akuity-kargo-api-v1alpha1-ClusterPromotionTask"></a>

### ClusterPromotionTask



| Field | Type | Description |
| ----- | ---- | ----------- |
| metadata | k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta |   |
| spec | [PromotionTaskSpec](#github-com-akuity-kargo-api-v1alpha1-PromotionTaskSpec) |  Spec describes the desired transition of a specific Stage into a specific Freight.  +kubebuilder:validation:Required |






<a name="github-com-akuity-kargo-api-v1alpha1-ClusterPromotionTaskList"></a>

### ClusterPromotionTaskList
ClusterPromotionTaskList contains a list of PromotionTasks.


| Field | Type | Description |
| ----- | ---- | ----------- |
| metadata | k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta |   |
| items | [ClusterPromotionTask](#github-com-akuity-kargo-api-v1alpha1-ClusterPromotionTask) |   |






<a name="github-com-akuity-kargo-api-v1alpha1-CurrentStage"></a>

### CurrentStage
CurrentStage reflects a Stage's current use of Freight.


| Field | Type | Description |
| ----- | ---- | ----------- |
| since | k8s.io.apimachinery.pkg.apis.meta.v1.Time |  Since is the time at which the Stage most recently started using the Freight. This can be used to calculate how long the Freight has been in use by the Stage. |






<a name="github-com-akuity-kargo-api-v1alpha1-DiscoveredArtifacts"></a>

### DiscoveredArtifacts
DiscoveredArtifacts holds the artifacts discovered by the Warehouse for its
subscriptions.


| Field | Type | Description |
| ----- | ---- | ----------- |
| discoveredAt | k8s.io.apimachinery.pkg.apis.meta.v1.Time |  DiscoveredAt is the time at which the Warehouse discovered the artifacts.  +optional |
| git | [GitDiscoveryResult](#github-com-akuity-kargo-api-v1alpha1-GitDiscoveryResult) |  Git holds the commits discovered by the Warehouse for the Git subscriptions.  +optional |
| images | [ImageDiscoveryResult](#github-com-akuity-kargo-api-v1alpha1-ImageDiscoveryResult) |  Images holds the image references discovered by the Warehouse for the image subscriptions.  +optional |
| charts | [ChartDiscoveryResult](#github-com-akuity-kargo-api-v1alpha1-ChartDiscoveryResult) |  Charts holds the charts discovered by the Warehouse for the chart subscriptions.  +optional |






<a name="github-com-akuity-kargo-api-v1alpha1-DiscoveredCommit"></a>

### DiscoveredCommit
DiscoveredCommit represents a commit discovered by a Warehouse for a
GitSubscription.


| Field | Type | Description |
| ----- | ---- | ----------- |
| id | [string](#string) |  ID is the identifier of the commit. This typically is a SHA-1 hash.  +kubebuilder:validation:MinLength=1 |
| branch | [string](#string) |  Branch is the branch in which the commit was found. This field is optional, and populated based on the CommitSelectionStrategy of the GitSubscription. |
| tag | [string](#string) |  Tag is the tag that resolved to this commit. This field is optional, and populated based on the CommitSelectionStrategy of the GitSubscription. |
| subject | [string](#string) |  Subject is the subject of the commit (i.e. the first line of the commit message). |
| author | [string](#string) |  Author is the author of the commit. |
| committer | [string](#string) |  Committer is the person who committed the commit. |
| creatorDate | k8s.io.apimachinery.pkg.apis.meta.v1.Time |  CreatorDate is the commit creation date as specified by the commit, or the tagger date if the commit belongs to an annotated tag. |






<a name="github-com-akuity-kargo-api-v1alpha1-DiscoveredImageReference"></a>

### DiscoveredImageReference
DiscoveredImageReference represents an image reference discovered by a
Warehouse for an ImageSubscription.


| Field | Type | Description |
| ----- | ---- | ----------- |
| tag | [string](#string) |  Tag is the tag of the image.  +kubebuilder:validation:MinLength=1 +kubebuilder:validation:MaxLength=128 +kubebuilder:validation:Pattern=`^\w.\-\_+$` +akuity:test-kubebuilder-pattern=Tag |
| digest | [string](#string) |  Digest is the digest of the image.  +kubebuilder:validation:MinLength=1 +kubebuilder:validation:Pattern=`^a-z0-9+:a-f0-9+$` +akuity:test-kubebuilder-pattern=Digest |
| annotations | [DiscoveredImageReference.AnnotationsEntry](#github-com-akuity-kargo-api-v1alpha1-DiscoveredImageReference-AnnotationsEntry) |  Annotations is a map of key-value pairs that provide additional information about the image. |
| createdAt | k8s.io.apimachinery.pkg.apis.meta.v1.Time |  CreatedAt is the time the image was created. This field is optional, and not populated for every ImageSelectionStrategy. |






<a name="github-com-akuity-kargo-api-v1alpha1-DiscoveredImageReference-AnnotationsEntry"></a>

### DiscoveredImageReference.AnnotationsEntry



| Field | Type | Description |
| ----- | ---- | ----------- |
| key | [string](#string) |   |
| value | [string](#string) |   |






<a name="github-com-akuity-kargo-api-v1alpha1-DockerHubWebhookReceiverConfig"></a>

### DockerHubWebhookReceiverConfig
DockerHubWebhookReceiverConfig describes a webhook receiver that is
compatible with Docker Hub payloads.


| Field | Type | Description |
| ----- | ---- | ----------- |
| secretRef | k8s.io.api.core.v1.LocalObjectReference |  SecretRef contains a reference to a Secret. For Project-scoped webhook receivers, the referenced Secret must be in the same namespace as the ProjectConfig.  The Secret's data map is expected to contain a `secret` key whose value does NOT need to be shared directly with Docker Hub when registering a webhook. It is used only by Kargo to create a complex, hard-to-guess URL, which implicitly serves as a shared secret. For more information about Docker Hub webhooks, please refer to the Docker documentation:   https://docs.docker.com/docker-hub/webhooks/  +kubebuilder:validation:Required |






<a name="github-com-akuity-kargo-api-v1alpha1-ExpressionVariable"></a>

### ExpressionVariable
ExpressionVariable describes a single variable that may be referenced by
expressions in the context of a ClusterPromotionTask, PromotionTask,
Promotion, AnalysisRun arguments, or other objects that support expressions.

It is used to pass information to the expression evaluation engine, and to
allow for dynamic evaluation of expressions based on the variable values.


| Field | Type | Description |
| ----- | ---- | ----------- |
| name | [string](#string) |  Name is the name of the variable.  +kubebuilder:validation:MinLength=1 +kubebuilder:validation:Pattern=^a-zA-Z_\w*$ |
| value | [string](#string) |  Value is the value of the variable. It is allowed to utilize expressions in the value. See https://docs.kargo.io/user-guide/reference-docs/expressions for details. |






<a name="github-com-akuity-kargo-api-v1alpha1-Freight"></a>

### Freight
Freight represents a collection of versioned artifacts.


| Field | Type | Description |
| ----- | ---- | ----------- |
| metadata | k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta |   |
| alias | [string](#string) |  Alias is a human-friendly alias for a piece of Freight. This is an optional field. A defaulting webhook will sync this field with the value of the kargo.akuity.io/alias label. When the alias label is not present or differs from the value of this field, the defaulting webhook will set the label to the value of this field. If the alias label is present and this field is empty, the defaulting webhook will set the value of this field to the value of the alias label. If this field is empty and the alias label is not present, the defaulting webhook will choose an available alias and assign it to both the field and label. |
| origin | [FreightOrigin](#github-com-akuity-kargo-api-v1alpha1-FreightOrigin) |  Origin describes a kind of Freight in terms of its origin.  +kubebuilder:validation:Required |
| commits | [GitCommit](#github-com-akuity-kargo-api-v1alpha1-GitCommit) |  Commits describes specific Git repository commits. |
| images | [Image](#github-com-akuity-kargo-api-v1alpha1-Image) |  Images describes specific versions of specific container images. |
| charts | [Chart](#github-com-akuity-kargo-api-v1alpha1-Chart) |  Charts describes specific versions of specific Helm charts. |
| status | [FreightStatus](#github-com-akuity-kargo-api-v1alpha1-FreightStatus) |  Status describes the current status of this Freight. |






<a name="github-com-akuity-kargo-api-v1alpha1-FreightCollection"></a>

### FreightCollection
FreightCollection is a collection of FreightReferences, each of which
represents a piece of Freight that has been selected for deployment to a
Stage.


| Field | Type | Description |
| ----- | ---- | ----------- |
| id | [string](#string) |  ID is a unique and deterministically calculated identifier for the FreightCollection. It is updated on each use of the UpdateOrPush method. |
| items | [FreightCollection.ItemsEntry](#github-com-akuity-kargo-api-v1alpha1-FreightCollection-ItemsEntry) |  Freight is a map of FreightReference objects, indexed by their Warehouse origin. |
| verificationHistory | [VerificationInfo](#github-com-akuity-kargo-api-v1alpha1-VerificationInfo) |  VerificationHistory is a stack of recent VerificationInfo. By default, the last ten VerificationInfo are stored. |






<a name="github-com-akuity-kargo-api-v1alpha1-FreightCollection-ItemsEntry"></a>

### FreightCollection.ItemsEntry



| Field | Type | Description |
| ----- | ---- | ----------- |
| key | [string](#string) |   |
| value | [FreightReference](#github-com-akuity-kargo-api-v1alpha1-FreightReference) |   |






<a name="github-com-akuity-kargo-api-v1alpha1-FreightList"></a>

### FreightList
FreightList is a list of Freight resources.


| Field | Type | Description |
| ----- | ---- | ----------- |
| metadata | k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta |   |
| items | [Freight](#github-com-akuity-kargo-api-v1alpha1-Freight) |   |






<a name="github-com-akuity-kargo-api-v1alpha1-FreightOrigin"></a>

### FreightOrigin
FreightOrigin describes a kind of Freight in terms of where it may have
originated.

+protobuf.options.(gogoproto.goproto_stringer)=false


| Field | Type | Description |
| ----- | ---- | ----------- |
| kind | [string](#string) |  Kind is the kind of resource from which Freight may have originated. At present, this can only be "Warehouse".  +kubebuilder:validation:Required |
| name | [string](#string) |  Name is the name of the resource of the kind indicated by the Kind field from which Freight may originate.  +kubebuilder:validation:Required |






<a name="github-com-akuity-kargo-api-v1alpha1-FreightReference"></a>

### FreightReference
FreightReference is a simplified representation of a piece of Freight -- not
a root resource type.


| Field | Type | Description |
| ----- | ---- | ----------- |
| name | [string](#string) |  Name is a system-assigned identifier derived deterministically from the contents of the Freight. I.e., two pieces of Freight can be compared for equality by comparing their Names. |
| origin | [FreightOrigin](#github-com-akuity-kargo-api-v1alpha1-FreightOrigin) |  Origin describes a kind of Freight in terms of its origin. |
| commits | [GitCommit](#github-com-akuity-kargo-api-v1alpha1-GitCommit) |  Commits describes specific Git repository commits. |
| images | [Image](#github-com-akuity-kargo-api-v1alpha1-Image) |  Images describes specific versions of specific container images. |
| charts | [Chart](#github-com-akuity-kargo-api-v1alpha1-Chart) |  Charts describes specific versions of specific Helm charts. |






<a name="github-com-akuity-kargo-api-v1alpha1-FreightRequest"></a>

### FreightRequest
FreightRequest expresses a Stage's need for Freight having originated from a
particular Warehouse.


| Field | Type | Description |
| ----- | ---- | ----------- |
| origin | [FreightOrigin](#github-com-akuity-kargo-api-v1alpha1-FreightOrigin) |  Origin specifies from where the requested Freight must have originated. This is a required field.  +kubebuilder:validation:Required |
| sources | [FreightSources](#github-com-akuity-kargo-api-v1alpha1-FreightSources) |  Sources describes where the requested Freight may be obtained from. This is a required field. |






<a name="github-com-akuity-kargo-api-v1alpha1-FreightSources"></a>

### FreightSources



| Field | Type | Description |
| ----- | ---- | ----------- |
| direct | [bool](#bool) |  Direct indicates the requested Freight may be obtained directly from the Warehouse from which it originated. If this field's value is false, then the value of the Stages field must be non-empty. i.e. Between the two fields, at least one source must be specified. |
| stages | [string](#string) |  Stages identifies other "upstream" Stages as potential sources of the requested Freight. If this field's value is empty, then the value of the Direct field must be true. i.e. Between the two fields, at least on source must be specified. |
| requiredSoakTime | k8s.io.apimachinery.pkg.apis.meta.v1.Duration |  RequiredSoakTime specifies a minimum duration for which the requested Freight must have continuously occupied ("soaked in") in an upstream Stage before becoming available for promotion to this Stage. This is an optional field. If nil or zero, no soak time is required. Any soak time requirement is in ADDITION to the requirement that Freight be verified in an upstream Stage to become available for promotion to this Stage, although a manual approval for promotion to this Stage will supersede any soak time requirement.  +kubebuilder:validation:Type=string +kubebuilder:validation:Pattern=`^(0-9+(\.0-9+)?(s|m|h))+$` +akuity:test-kubebuilder-pattern=Duration |
| availabilityStrategy | [string](#string) |  AvailabilityStrategy specifies the semantics for how requested Freight is made available to the Stage. This field is optional. When left unspecified, the field is implicitly treated as if its value were "OneOf".  Accepted Values:  - "All": Freight must be verified and, if applicable, soaked in all   upstream Stages to be considered available for promotion. - "OneOf": Freight must be verified and, if applicable, soaked in at least    one upstream Stage to be considered available for promotion. - "": Treated the same as "OneOf".  +kubebuilder:validation:Optional |






<a name="github-com-akuity-kargo-api-v1alpha1-FreightStatus"></a>

### FreightStatus
FreightStatus describes a piece of Freight's most recently observed state.


| Field | Type | Description |
| ----- | ---- | ----------- |
| currentlyIn | [FreightStatus.CurrentlyInEntry](#github-com-akuity-kargo-api-v1alpha1-FreightStatus-CurrentlyInEntry) |  CurrentlyIn describes the Stages in which this Freight is currently in use. |
| verifiedIn | [FreightStatus.VerifiedInEntry](#github-com-akuity-kargo-api-v1alpha1-FreightStatus-VerifiedInEntry) |  VerifiedIn describes the Stages in which this Freight has been verified through promotion and subsequent health checks. |
| approvedFor | [FreightStatus.ApprovedForEntry](#github-com-akuity-kargo-api-v1alpha1-FreightStatus-ApprovedForEntry) |  ApprovedFor describes the Stages for which this Freight has been approved preemptively/manually by a user. This is useful for hotfixes, where one might wish to promote a piece of Freight to a given Stage without transiting the entire pipeline. |
| metadata | [FreightStatus.MetadataEntry](#github-com-akuity-kargo-api-v1alpha1-FreightStatus-MetadataEntry) |  Metadata is a map of arbitrary metadata associated with the Freight. This is useful for storing additional information about the Freight or Promotion that can be shared across steps or stages. |






<a name="github-com-akuity-kargo-api-v1alpha1-FreightStatus-ApprovedForEntry"></a>

### FreightStatus.ApprovedForEntry



| Field | Type | Description |
| ----- | ---- | ----------- |
| key | [string](#string) |   |
| value | [ApprovedStage](#github-com-akuity-kargo-api-v1alpha1-ApprovedStage) |   |






<a name="github-com-akuity-kargo-api-v1alpha1-FreightStatus-CurrentlyInEntry"></a>

### FreightStatus.CurrentlyInEntry



| Field | Type | Description |
| ----- | ---- | ----------- |
| key | [string](#string) |   |
| value | [CurrentStage](#github-com-akuity-kargo-api-v1alpha1-CurrentStage) |   |






<a name="github-com-akuity-kargo-api-v1alpha1-FreightStatus-MetadataEntry"></a>

### FreightStatus.MetadataEntry



| Field | Type | Description |
| ----- | ---- | ----------- |
| key | [string](#string) |   |
| value | k8s.io.apiextensions_apiserver.pkg.apis.apiextensions.v1.JSON |   |






<a name="github-com-akuity-kargo-api-v1alpha1-FreightStatus-VerifiedInEntry"></a>

### FreightStatus.VerifiedInEntry



| Field | Type | Description |
| ----- | ---- | ----------- |
| key | [string](#string) |   |
| value | [VerifiedStage](#github-com-akuity-kargo-api-v1alpha1-VerifiedStage) |   |






<a name="github-com-akuity-kargo-api-v1alpha1-GitCommit"></a>

### GitCommit
GitCommit describes a specific commit from a specific Git repository.


| Field | Type | Description |
| ----- | ---- | ----------- |
| repoURL | [string](#string) |  RepoURL is the URL of a Git repository. |
| id | [string](#string) |  ID is the ID of a specific commit in the Git repository specified by RepoURL. |
| branch | [string](#string) |  Branch denotes the branch of the repository where this commit was found. |
| tag | [string](#string) |  Tag denotes a tag in the repository that matched selection criteria and resolved to this commit. |
| message | [string](#string) |  Message is the message associated with the commit. At present, this only contains the first line (subject) of the commit message. |
| author | [string](#string) |  Author is the author of the commit. |
| committer | [string](#string) |  Committer is the person who committed the commit. |






<a name="github-com-akuity-kargo-api-v1alpha1-GitDiscoveryResult"></a>

### GitDiscoveryResult
GitDiscoveryResult represents the result of a Git discovery operation for a
GitSubscription.


| Field | Type | Description |
| ----- | ---- | ----------- |
| repoURL | [string](#string) |  RepoURL is the repository URL of the GitSubscription.  +kubebuilder:validation:MinLength=1 +kubebuilder:validation:Pattern=`(?:^(ssh|https?)://(?:(\w-+)(:(.+))?@)?(\w-+(?:\.\w-+)*)(?::(\d{1,5}))?(/.*)$)|(?:^(\w-+)@(\w++(?:\.\w-+)*):(/?.*))` +akuity:test-kubebuilder-pattern=GitRepoURLPattern |
| commits | [DiscoveredCommit](#github-com-akuity-kargo-api-v1alpha1-DiscoveredCommit) |  Commits is a list of commits discovered by the Warehouse for the GitSubscription. An empty list indicates that the discovery operation was successful, but no commits matching the GitSubscription criteria were found.  +optional |






<a name="github-com-akuity-kargo-api-v1alpha1-GitHubWebhookReceiverConfig"></a>

### GitHubWebhookReceiverConfig
GitHubWebhookReceiverConfig describes a webhook receiver that is compatible
with GitHub payloads.


| Field | Type | Description |
| ----- | ---- | ----------- |
| secretRef | k8s.io.api.core.v1.LocalObjectReference |  SecretRef contains a reference to a Secret. For Project-scoped webhook receivers, the referenced Secret must be in the same namespace as the ProjectConfig.  For cluster-scoped webhook receivers, the referenced Secret must be in the designated "cluster Secrets" namespace.  The Secret's data map is expected to contain a `secret` key whose value is the shared secret used to authenticate the webhook requests sent by GitHub. For more information please refer to GitHub documentation:   https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries  +kubebuilder:validation:Required |






<a name="github-com-akuity-kargo-api-v1alpha1-GitLabWebhookReceiverConfig"></a>

### GitLabWebhookReceiverConfig
GitLabWebhookReceiverConfig describes a webhook receiver that is compatible
with GitLab payloads.


| Field | Type | Description |
| ----- | ---- | ----------- |
| secretRef | k8s.io.api.core.v1.LocalObjectReference |  SecretRef contains a reference to a Secret. For Project-scoped webhook receivers, the referenced Secret must be in the same namespace as the ProjectConfig.  For cluster-scoped webhook receivers, the referenced Secret must be in the designated "cluster Secrets" namespace.  The secret is expected to contain a `secret-token` key containing the shared secret specified when registering the webhook in GitLab. For more information about this token, please refer to the GitLab documentation:   https://docs.gitlab.com/user/project/integrations/webhooks/  +kubebuilder:validation:Required |






<a name="github-com-akuity-kargo-api-v1alpha1-GitSubscription"></a>

### GitSubscription
GitSubscription defines a subscription to a Git repository.


| Field | Type | Description |
| ----- | ---- | ----------- |
| repoURL | [string](#string) |  URL is the repository's URL. This is a required field.  +kubebuilder:validation:MinLength=1 +kubebuilder:validation:Pattern=`(?:^(ssh|https?)://(?:(\w-+)(:(.+))?@)?(\w-+(?:\.\w-+)*)(?::(\d{1,5}))?(/.*)$)|(?:^(\w-+)@(\w++(?:\.\w-+)*):(/?.*))` +akuity:test-kubebuilder-pattern=GitRepoURLPattern |
| commitSelectionStrategy | [string](#string) |  CommitSelectionStrategy specifies the rules for how to identify the newest commit of interest in the repository specified by the RepoURL field. This field is optional. When left unspecified, the field is implicitly treated as if its value were "NewestFromBranch".  Accepted values:  - "NewestFromBranch": Selects the latest commit on the branch specified   by the Branch field or the default branch if none is specified. This is   the default strategy.  - "SemVer": Selects the commit referenced by the semantically greatest   tag. The SemverConstraint field can optionally be used to narrow the set   of tags eligible for selection.  - "Lexical": Selects the commit referenced by the lexicographically   greatest tag. Useful when tags embed a _leading_ date or timestamp. The   AllowTags and IgnoreTags fields can optionally be used to narrow the set   of tags eligible for selection.  - "NewestTag": Selects the commit referenced by the most recently created   tag. The AllowTags and IgnoreTags fields can optionally be used to   narrow the set of tags eligible for selection.  +kubebuilder:default=NewestFromBranch |
| branch | [string](#string) |  Branch references a particular branch of the repository. The value in this field only has any effect when the CommitSelectionStrategy is NewestFromBranch or left unspecified (which is implicitly the same as NewestFromBranch). This field is optional. When left unspecified, (and the CommitSelectionStrategy is NewestFromBranch or unspecified), the subscription is implicitly to the repository's default branch.  +kubebuilder:validation:MinLength=1 +kubebuilder:validation:MaxLength=255 +kubebuilder:validation:Pattern=`^a-zA-Z0-9(a-zA-Z0-9._\/-*a-zA-Z0-9_-)?$` +akuity:test-kubebuilder-pattern=Branch |
| strictSemvers | [bool](#bool) |  StrictSemvers specifies whether only "strict" semver tags should be considered. A "strict" semver tag is one containing ALL of major, minor, and patch version components. This is enabled by default, but only has any effect when the CommitSelectionStrategy is SemVer. This should be disabled cautiously, as it creates the potential for any tag containing numeric characters only to be mistaken for a semver string containing the major version number only.  +kubebuilder:default=true |
| semverConstraint | [string](#string) |  SemverConstraint specifies constraints on what new tagged commits are considered in determining the newest commit of interest. The value in this field only has any effect when the CommitSelectionStrategy is SemVer. This field is optional. When left unspecified, there will be no constraints, which means the latest semantically tagged commit will always be used. Care should be taken with leaving this field unspecified, as it can lead to the unanticipated rollout of breaking changes.  +kubebuilder:validation:Optional |
| allowTags | [string](#string) |  AllowTags is a regular expression that can optionally be used to limit the tags that are considered in determining the newest commit of interest. The value in this field only has any effect when the CommitSelectionStrategy is Lexical, NewestTag, or SemVer. This field is optional.  +kubebuilder:validation:Optional |
| ignoreTags | [string](#string) |  IgnoreTags is a list of tags that must be ignored when determining the newest commit of interest. No regular expressions or glob patterns are supported yet. The value in this field only has any effect when the CommitSelectionStrategy is Lexical, NewestTag, or SemVer. This field is optional.  +kubebuilder:validation:Optional |
| expressionFilter | [string](#string) |  ExpressionFilter is an expression that can optionally be used to limit the commits or tags that are considered in determining the newest commit of interest based on their metadata.  For commit-based strategies (NewestFromBranch), the filter applies to commits and has access to commit metadata variables. For tag-based strategies (Lexical, NewestTag, SemVer), the filter applies to tags and has access to tag metadata variables. The filter is applied after AllowTags, IgnoreTags, and SemverConstraint fields.  The expression should be a valid expr-lang expression that evaluates to true or false. When the expression evaluates to true, the commit/tag is included in the set that is considered. When the expression evaluates to false, the commit/tag is excluded.  Available variables depend on the CommitSelectionStrategy:  For NewestFromBranch (commit filtering):   - `id`: The ID (sha) of the commit.   - `commitDate`: The commit date of the commit.   - `author`: The author of the commit message, in the format "Name email".   - `committer`: The person who committed the commit, in the format 	   "Name email".   - `subject`: The subject (first line) of the commit message.  For Lexical, NewestTag, SemVer (tag filtering):   - `tag`: The name of the tag.   - `id`: The ID (sha) of the commit associated with the tag.   - `creatorDate`: The creation date of an annotated tag, or the commit 		date of a lightweight tag.   - `author`: The author of the commit message associated with the tag, 	   in the format "Name email".   - `committer`: The person who committed the commit associated with the 	   tag, in the format "Name email".   - `subject`: The subject (first line) of the commit message associated 	   with the tag. 	 - `tagger`: The person who created the tag, in the format "Name email". 	   Only available for annotated tags. 	 - `annotation`: The subject (first line) of the tag annotation. Only 	   available for annotated tags.  Refer to the expr-lang documentation for more details on syntax and capabilities of the expression language: https://expr-lang.org.  +kubebuilder:validation:Optional |
| insecureSkipTLSVerify | [bool](#bool) |  InsecureSkipTLSVerify specifies whether certificate verification errors should be ignored when connecting to the repository. This should be enabled only with great caution. |
| includePaths | [string](#string) |  IncludePaths is a list of selectors that designate paths in the repository that should trigger the production of new Freight when changes are detected therein. When specified, only changes in the identified paths will trigger Freight production. When not specified, changes in any path will trigger Freight production. Selectors may be defined using:   1. Exact paths to files or directories (ex. "charts/foo")   2. Glob patterns (prefix the pattern with "glob:"; ex. "glob:*.yaml")   3. Regular expressions (prefix the pattern with "regex:" or "regexp:";      ex. "regexp:^.*\.yaml$")  Paths selected by IncludePaths may be unselected by ExcludePaths. This is a useful method for including a broad set of paths and then excluding a subset of them. +kubebuilder:validation:Optional |
| excludePaths | [string](#string) |  ExcludePaths is a list of selectors that designate paths in the repository that should NOT trigger the production of new Freight when changes are detected therein. When specified, changes in the identified paths will not trigger Freight production. When not specified, paths that should trigger Freight production will be defined solely by IncludePaths. Selectors may be defined using:   1. Exact paths to files or directories (ex. "charts/foo")   2. Glob patterns (prefix the pattern with "glob:"; ex. "glob:*.yaml")   3. Regular expressions (prefix the pattern with "regex:" or "regexp:";      ex. "regexp:^.*\.yaml$") Paths selected by IncludePaths may be unselected by ExcludePaths. This is a useful method for including a broad set of paths and then excluding a subset of them. +kubebuilder:validation:Optional |
| discoveryLimit | [int32](#int32) |  DiscoveryLimit is an optional limit on the number of commits that can be discovered for this subscription. The limit is applied after filtering commits based on the AllowTags and IgnoreTags fields. When left unspecified, the field is implicitly treated as if its value were "20". The upper limit for this field is 100.  +kubebuilder:validation:Minimum=1 +kubebuilder:validation:Maximum=100 +kubebuilder:default=20 |






<a name="github-com-akuity-kargo-api-v1alpha1-GiteaWebhookReceiverConfig"></a>

### GiteaWebhookReceiverConfig
GiteaWebhookReceiverConfig describes a webhook receiver that is compatible
with Gitea payloads.


| Field | Type | Description |
| ----- | ---- | ----------- |
| secretRef | k8s.io.api.core.v1.LocalObjectReference |  SecretRef contains a reference to a Secret. For Project-scoped webhook receivers, the referenced Secret must be in the same namespace as the ProjectConfig.  For cluster-scoped webhook receivers, the referenced Secret must be in the designated "cluster Secrets" namespace.  The Secret's data map is expected to contain a `secret` key whose value is the shared secret used to authenticate the webhook requests sent by Gitea. For more information please refer to the Gitea documentation:   https://docs.gitea.io/en-us/webhooks/  +kubebuilder:validation:Required |






<a name="github-com-akuity-kargo-api-v1alpha1-HarborWebhookReceiverConfig"></a>

### HarborWebhookReceiverConfig
HarborWebhookReceiverConfig describes a webhook receiver that is compatible
with Harbor payloads.


| Field | Type | Description |
| ----- | ---- | ----------- |
| secretRef | k8s.io.api.core.v1.LocalObjectReference |  SecretRef contains a reference to a Secret. For Project-scoped webhook receivers, the referenced Secret must be in the same namespace as the ProjectConfig.  For cluster-scoped webhook receivers, the referenced Secret must be in the designated "cluster Secrets" namespace.  The secret is expected to contain an `auth-header` key containing the "auth header" specified when registering the webhook in Harbor. For more information, please refer to the Harbor documentation:   https://goharbor.io/docs/main/working-with-projects/project-configuration/configure-webhooks/  +kubebuilder:validation:Required |






<a name="github-com-akuity-kargo-api-v1alpha1-Health"></a>

### Health
Health describes the health of a Stage.


| Field | Type | Description |
| ----- | ---- | ----------- |
| status | [string](#string) |  Status describes the health of the Stage. |
| issues | [string](#string) |  Issues clarifies why a Stage in any state other than Healthy is in that state. This field will always be the empty when a Stage is Healthy. |
| config | k8s.io.apiextensions_apiserver.pkg.apis.apiextensions.v1.JSON |  Config is the opaque configuration of all health checks performed on this Stage. |
| output | k8s.io.apiextensions_apiserver.pkg.apis.apiextensions.v1.JSON |  Output is the opaque output of all health checks performed on this Stage. |






<a name="github-com-akuity-kargo-api-v1alpha1-HealthCheckStep"></a>

### HealthCheckStep
HealthCheckStep describes a health check directive which can be executed by
a Stage to verify the health of a Promotion result.


| Field | Type | Description |
| ----- | ---- | ----------- |
| uses | [string](#string) |  Uses identifies a runner that can execute this step.  +kubebuilder:validation:MinLength=1 |
| config | k8s.io.apiextensions_apiserver.pkg.apis.apiextensions.v1.JSON |  Config is the configuration for the directive. |






<a name="github-com-akuity-kargo-api-v1alpha1-HealthStats"></a>

### HealthStats
HealthStats contains a summary of the collective health of some resource
type.


| Field | Type | Description |
| ----- | ---- | ----------- |
| healthy | [int64](#int64) |  Healthy contains the number of resources that are explicitly healthy. |






<a name="github-com-akuity-kargo-api-v1alpha1-Image"></a>

### Image
Image describes a specific version of a container image.


| Field | Type | Description |
| ----- | ---- | ----------- |
| repoURL | [string](#string) |  RepoURL describes the repository in which the image can be found. |
| tag | [string](#string) |  Tag identifies a specific version of the image in the repository specified by RepoURL. |
| digest | [string](#string) |  Digest identifies a specific version of the image in the repository specified by RepoURL. This is a more precise identifier than Tag. |
| annotations | [Image.AnnotationsEntry](#github-com-akuity-kargo-api-v1alpha1-Image-AnnotationsEntry) |  Annotations is a map of arbitrary metadata for the image. |






<a name="github-com-akuity-kargo-api-v1alpha1-Image-AnnotationsEntry"></a>

### Image.AnnotationsEntry



| Field | Type | Description |
| ----- | ---- | ----------- |
| key | [string](#string) |   |
| value | [string](#string) |   |






<a name="github-com-akuity-kargo-api-v1alpha1-ImageDiscoveryResult"></a>

### ImageDiscoveryResult
ImageDiscoveryResult represents the result of an image discovery operation
for an ImageSubscription.


| Field | Type | Description |
| ----- | ---- | ----------- |
| repoURL | [string](#string) |  RepoURL is the repository URL of the image, as specified in the ImageSubscription.  +kubebuilder:validation:MinLength=1 |
| platform | [string](#string) |  Platform is the target platform constraint of the ImageSubscription for which references were discovered. This field is optional, and only populated if the ImageSubscription specifies a Platform. |
| references | [DiscoveredImageReference](#github-com-akuity-kargo-api-v1alpha1-DiscoveredImageReference) |  References is a list of image references discovered by the Warehouse for the ImageSubscription. An empty list indicates that the discovery operation was successful, but no images matching the ImageSubscription criteria were found.  +optional |






<a name="github-com-akuity-kargo-api-v1alpha1-ImageSubscription"></a>

### ImageSubscription
ImageSubscription defines a subscription to an image repository.

+kubebuilder:validation:XValidation:message="semverConstraint and constraint fields are mutually exclusive",rule="!(has(self.semverConstraint) && has(self.constraint))"
+kubebuilder:validation:XValidation:message="If imageSelectionStrategy is Digest, either constraint or semverConstraint must be set",rule="!(self.imageSelectionStrategy == 'Digest') || has(self.constraint) || has(self.semverConstraint)"


| Field | Type | Description |
| ----- | ---- | ----------- |
| repoURL | [string](#string) |  RepoURL specifies the URL of the image repository to subscribe to. The value in this field MUST NOT include an image tag. This field is required.  +kubebuilder:validation:MinLength=1 +kubebuilder:validation:Pattern=`^(\w+(\.-\w+)*(:\d+)?/)?(\w+(\.-\w+)*)(/\w+(\.-\w+)*)*$` +akuity:test-kubebuilder-pattern=ImageRepoURL |
| imageSelectionStrategy | [string](#string) |  ImageSelectionStrategy specifies the rules for how to identify the newest version of the image specified by the RepoURL field. This field is optional. When left unspecified, the field is implicitly treated as if its value were "SemVer".  Accepted values:  - "Digest": Selects the image currently referenced by the tag specified   (unintuitively) by the SemverConstraint field.  - "Lexical": Selects the image referenced by the lexicographically greatest   tag. Useful when tags embed a leading date or timestamp. The AllowTags   and IgnoreTags fields can optionally be used to narrow the set of tags   eligible for selection.  - "NewestBuild": Selects the image that was most recently pushed to the   repository. The AllowTags and IgnoreTags fields can optionally be used   to narrow the set of tags eligible for selection. This is the least   efficient and is likely to cause rate limiting affecting this Warehouse   and possibly others. This strategy should be avoided.  - "SemVer": Selects the image with the semantically greatest tag. The   AllowTags and IgnoreTags fields can optionally be used to narrow the set   of tags eligible for selection.  +kubebuilder:default=SemVer |
| strictSemvers | [bool](#bool) |  StrictSemvers specifies whether only "strict" semver tags should be considered. A "strict" semver tag is one containing ALL of major, minor, and patch version components. This is enabled by default, but only has any effect when the ImageSelectionStrategy is SemVer. This should be disabled cautiously, as it is not uncommon to tag container images with short Git commit hashes, which have the potential to contain numeric characters only and could be mistaken for a semver string containing the major version number only.  +kubebuilder:default=true |
| semverConstraint | [string](#string) |  SemverConstraint specifies constraints on what new image versions are permissible. The value in this field only has any effect when the ImageSelectionStrategy is SemVer or left unspecified (which is implicitly the same as SemVer). This field is also optional. When left unspecified, (and the ImageSelectionStrategy is SemVer or unspecified), there will be no constraints, which means the latest semantically tagged version of an image will always be used. Care should be taken with leaving this field unspecified, as it can lead to the unanticipated rollout of breaking changes. More info: https://github.com/masterminds/semver#checking-version-constraints  Deprecated: Use Constraint instead. This field will be removed in v1.9.0  +kubebuilder:validation:Optional |
| constraint | [string](#string) |  Constraint specifies constraints on what new image versions are permissible. Acceptable values for this field vary contextually by ImageSelectionStrategy. The field is optional and is ignored by some strategies. When non-empty, the value in this field takes precedence over the value of the deprecated SemverConstraint field.  +kubebuilder:validation:Optional |
| allowTags | [string](#string) |  AllowTags is a regular expression that can optionally be used to limit the image tags that are considered in determining the newest version of an image. This field is optional.  +kubebuilder:validation:Optional |
| ignoreTags | [string](#string) |  IgnoreTags is a list of tags that must be ignored when determining the newest version of an image. No regular expressions or glob patterns are supported yet. This field is optional.  +kubebuilder:validation:Optional |
| platform | [string](#string) |  Platform is a string of the form os/arch that limits the tags that can be considered when searching for new versions of an image. This field is optional. When left unspecified, it is implicitly equivalent to the OS/architecture of the Kargo controller. Care should be taken to set this value correctly in cases where the image referenced by this ImageRepositorySubscription will run on a Kubernetes node with a different OS/architecture than the Kargo controller. At present this is uncommon, but not unheard of.  +kubebuilder:validation:Optional |
| insecureSkipTLSVerify | [bool](#bool) |  InsecureSkipTLSVerify specifies whether certificate verification errors should be ignored when connecting to the repository. This should be enabled only with great caution. |
| discoveryLimit | [int32](#int32) |  DiscoveryLimit is an optional limit on the number of image references that can be discovered for this subscription. The limit is applied after filtering images based on the AllowTags and IgnoreTags fields. When left unspecified, the field is implicitly treated as if its value were "20". The upper limit for this field is 100.  +kubebuilder:validation:Minimum=1 +kubebuilder:validation:Maximum=100 +kubebuilder:default=20 |






<a name="github-com-akuity-kargo-api-v1alpha1-Project"></a>

### Project
Project is a resource type that reconciles to a specially labeled namespace
and other TODO: TBD project-level resources.


| Field | Type | Description |
| ----- | ---- | ----------- |
| metadata | k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta |   |
| status | [ProjectStatus](#github-com-akuity-kargo-api-v1alpha1-ProjectStatus) |  Status describes the Project's current status. |






<a name="github-com-akuity-kargo-api-v1alpha1-ProjectConfig"></a>

### ProjectConfig
ProjectConfig is a resource type that describes the configuration of a
Project.


| Field | Type | Description |
| ----- | ---- | ----------- |
| metadata | k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta |   |
| spec | [ProjectConfigSpec](#github-com-akuity-kargo-api-v1alpha1-ProjectConfigSpec) |  Spec describes the configuration of a Project. |
| status | [ProjectConfigStatus](#github-com-akuity-kargo-api-v1alpha1-ProjectConfigStatus) |  Status describes the current status of a ProjectConfig. |






<a name="github-com-akuity-kargo-api-v1alpha1-ProjectConfigList"></a>

### ProjectConfigList
ProjectConfigList is a list of ProjectConfig resources.


| Field | Type | Description |
| ----- | ---- | ----------- |
| metadata | k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta |   |
| items | [ProjectConfig](#github-com-akuity-kargo-api-v1alpha1-ProjectConfig) |   |






<a name="github-com-akuity-kargo-api-v1alpha1-ProjectConfigSpec"></a>

### ProjectConfigSpec
ProjectConfigSpec describes the configuration of a Project.


| Field | Type | Description |
| ----- | ---- | ----------- |
| promotionPolicies | [PromotionPolicy](#github-com-akuity-kargo-api-v1alpha1-PromotionPolicy) |  PromotionPolicies defines policies governing the promotion of Freight to specific Stages within the Project. |
| webhookReceivers | [WebhookReceiverConfig](#github-com-akuity-kargo-api-v1alpha1-WebhookReceiverConfig) |  WebhookReceivers describes Project-specific webhook receivers used for processing events from various external platforms |






<a name="github-com-akuity-kargo-api-v1alpha1-ProjectConfigStatus"></a>

### ProjectConfigStatus
ProjectConfigStatus describes the current status of a ProjectConfig.


| Field | Type | Description |
| ----- | ---- | ----------- |
| conditions | k8s.io.apimachinery.pkg.apis.meta.v1.Condition |  Conditions contains the last observations of the Project Config's current state.  +patchMergeKey=type +patchStrategy=merge +listType=map +listMapKey=type |
| observedGeneration | [int64](#int64) |  ObservedGeneration represents the .metadata.generation that this ProjectConfig was reconciled against. |
| lastHandledRefresh | [string](#string) |  LastHandledRefresh holds the value of the most recent AnnotationKeyRefresh annotation that was handled by the controller. This field can be used to determine whether the request to refresh the resource has been handled. +optional |
| webhookReceivers | [WebhookReceiverDetails](#github-com-akuity-kargo-api-v1alpha1-WebhookReceiverDetails) |  WebhookReceivers describes the status of Project-specific webhook receivers. |






<a name="github-com-akuity-kargo-api-v1alpha1-ProjectList"></a>

### ProjectList
ProjectList is a list of Project resources.


| Field | Type | Description |
| ----- | ---- | ----------- |
| metadata | k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta |   |
| items | [Project](#github-com-akuity-kargo-api-v1alpha1-Project) |   |






<a name="github-com-akuity-kargo-api-v1alpha1-ProjectStats"></a>

### ProjectStats
ProjectStats contains a summary of the collective state of a Project's
constituent resources.


| Field | Type | Description |
| ----- | ---- | ----------- |
| warehouses | [WarehouseStats](#github-com-akuity-kargo-api-v1alpha1-WarehouseStats) |  Warehouses contains a summary of the collective state of the Project's Warehouses. |
| stages | [StageStats](#github-com-akuity-kargo-api-v1alpha1-StageStats) |  Stages contains a summary of the collective state of the Project's Stages. |






<a name="github-com-akuity-kargo-api-v1alpha1-ProjectStatus"></a>

### ProjectStatus
ProjectStatus describes a Project's current status.


| Field | Type | Description |
| ----- | ---- | ----------- |
| conditions | k8s.io.apimachinery.pkg.apis.meta.v1.Condition |  Conditions contains the last observations of the Project's current state. +patchMergeKey=type +patchStrategy=merge +listType=map +listMapKey=type |
| stats | [ProjectStats](#github-com-akuity-kargo-api-v1alpha1-ProjectStats) |  Stats contains a summary of the collective state of a Project's constituent resources. |






<a name="github-com-akuity-kargo-api-v1alpha1-Promotion"></a>

### Promotion
Promotion represents a request to transition a particular Stage into a
particular Freight.


| Field | Type | Description |
| ----- | ---- | ----------- |
| metadata | k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta |   |
| spec | [PromotionSpec](#github-com-akuity-kargo-api-v1alpha1-PromotionSpec) |  Spec describes the desired transition of a specific Stage into a specific Freight.  +kubebuilder:validation:Required |
| status | [PromotionStatus](#github-com-akuity-kargo-api-v1alpha1-PromotionStatus) |  Status describes the current state of the transition represented by this Promotion. |






<a name="github-com-akuity-kargo-api-v1alpha1-PromotionList"></a>

### PromotionList
PromotionList contains a list of Promotion


| Field | Type | Description |
| ----- | ---- | ----------- |
| metadata | k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta |   |
| items | [Promotion](#github-com-akuity-kargo-api-v1alpha1-Promotion) |   |






<a name="github-com-akuity-kargo-api-v1alpha1-PromotionPolicy"></a>

### PromotionPolicy
PromotionPolicy defines policies governing the promotion of Freight to a
specific Stage.

+kubebuilder:validation:XValidation:message="PromotionPolicy must have exactly one of stage or stageSelector set",rule="has(self.stage) ? !has(self.stageSelector) : has(self.stageSelector)"


| Field | Type | Description |
| ----- | ---- | ----------- |
| stage | [string](#string) |  Stage is the name of the Stage to which this policy applies.  Deprecated: Use StageSelector instead.  +kubebuilder:validation:Pattern=^a-z0-9(-a-z0-9*a-z0-9)?(\.a-z0-9(-a-z0-9*a-z0-9)?)*$ |
| stageSelector | [PromotionPolicySelector](#github-com-akuity-kargo-api-v1alpha1-PromotionPolicySelector) |  StageSelector is a selector that matches the Stage resource to which this policy applies. |
| autoPromotionEnabled | [bool](#bool) |  AutoPromotionEnabled indicates whether new Freight can automatically be promoted into the Stage referenced by the Stage field. Note: There are may be other conditions also required for an auto-promotion to occur. This field defaults to false, but is commonly set to true for Stages that subscribe to Warehouses instead of other, upstream Stages. This allows users to define Stages that are automatically updated as soon as new artifacts are detected. |






<a name="github-com-akuity-kargo-api-v1alpha1-PromotionPolicySelector"></a>

### PromotionPolicySelector
PromotionPolicySelector is a selector that matches the resource to which
this policy applies. It can be used to match a specific resource by name or
to match a set of resources by label.


| Field | Type | Description |
| ----- | ---- | ----------- |
| name | [string](#string) |  Name is the name of the resource to which this policy applies.  It can be an exact name, a regex pattern (with prefix "regex:"), or a glob pattern (with prefix "glob:").  When both Name and LabelSelector are specified, the Name is ANDed with the LabelSelector. I.e., the resource must match both the Name and LabelSelector to be selected by this policy.  NOTE: Using a specific exact name is the most secure option. Pattern matching via regex or glob can be exploited by users with permissions to match promotion policies that weren't intended to apply to their resources. For example, a user could create a resource with a name deliberately crafted to match the pattern, potentially bypassing intended promotion controls.  +optional |
| labelSelector | k8s.io.apimachinery.pkg.apis.meta.v1.LabelSelector |  LabelSelector is a selector that matches the resource to which this policy applies.  When both Name and LabelSelector are specified, the Name is ANDed with the LabelSelector. I.e., the resource must match both the Name and LabelSelector to be selected by this policy.  NOTE: Using label selectors introduces security risks as users with appropriate permissions could create new resources with labels that match the selector, potentially enabling unauthorized auto-promotion. For sensitive environments, exact Name matching provides tighter control. |






<a name="github-com-akuity-kargo-api-v1alpha1-PromotionReference"></a>

### PromotionReference
PromotionReference contains the relevant information about a Promotion
as observed by a Stage.


| Field | Type | Description |
| ----- | ---- | ----------- |
| name | [string](#string) |  Name is the name of the Promotion. |
| freight | [FreightReference](#github-com-akuity-kargo-api-v1alpha1-FreightReference) |  Freight is the freight being promoted. |
| status | [PromotionStatus](#github-com-akuity-kargo-api-v1alpha1-PromotionStatus) |  Status is the (optional) status of the Promotion. |
| finishedAt | k8s.io.apimachinery.pkg.apis.meta.v1.Time |  FinishedAt is the time at which the Promotion was completed. |






<a name="github-com-akuity-kargo-api-v1alpha1-PromotionSpec"></a>

### PromotionSpec
PromotionSpec describes the desired transition of a specific Stage into a
specific Freight.


| Field | Type | Description |
| ----- | ---- | ----------- |
| stage | [string](#string) |  Stage specifies the name of the Stage to which this Promotion applies. The Stage referenced by this field MUST be in the same namespace as the Promotion.  +kubebuilder:validation:Required +kubebuilder:validation:MinLength=1 +kubebuilder:validation:MaxLength=253 +kubebuilder:validation:Pattern=`^a-z0-9(-a-z0-9*a-z0-9)?(\.a-z0-9(-a-z0-9*a-z0-9)?)*$` +akuity:test-kubebuilder-pattern=KubernetesName |
| freight | [string](#string) |  Freight specifies the piece of Freight to be promoted into the Stage referenced by the Stage field.  +kubebuilder:validation:Required +kubebuilder:validation:MinLength=1 +kubebuilder:validation:MaxLength=253 +kubebuilder:validation:Pattern=`^a-z0-9(-a-z0-9*a-z0-9)?(\.a-z0-9(-a-z0-9*a-z0-9)?)*$` +akuity:test-kubebuilder-pattern=KubernetesName |
| vars | [ExpressionVariable](#github-com-akuity-kargo-api-v1alpha1-ExpressionVariable) |  Vars is a list of variables that can be referenced by expressions in promotion steps. |
| steps | [PromotionStep](#github-com-akuity-kargo-api-v1alpha1-PromotionStep) |  Steps specifies the directives to be executed as part of this Promotion. The order in which the directives are executed is the order in which they are listed in this field.  +kubebuilder:validation:Required +kubebuilder:validation:MinItems=1 +kubebuilder:validation:items:XValidation:message="Promotion step must have uses set and must not reference a task",rule="has(self.uses) && !has(self.task)" |






<a name="github-com-akuity-kargo-api-v1alpha1-PromotionStatus"></a>

### PromotionStatus
PromotionStatus describes the current state of the transition represented by
a Promotion.


| Field | Type | Description |
| ----- | ---- | ----------- |
| lastHandledRefresh | [string](#string) |  LastHandledRefresh holds the value of the most recent AnnotationKeyRefresh annotation that was handled by the controller. This field can be used to determine whether the request to refresh the resource has been handled. +optional |
| phase | [string](#string) |  Phase describes where the Promotion currently is in its lifecycle. |
| message | [string](#string) |  Message is a display message about the promotion, including any errors preventing the Promotion controller from executing this Promotion. i.e. If the Phase field has a value of Failed, this field can be expected to explain why. |
| freight | [FreightReference](#github-com-akuity-kargo-api-v1alpha1-FreightReference) |  Freight is the detail of the piece of freight that was referenced by this promotion. |
| freightCollection | [FreightCollection](#github-com-akuity-kargo-api-v1alpha1-FreightCollection) |  FreightCollection contains the details of the piece of Freight referenced by this Promotion as well as any additional Freight that is carried over from the target Stage's current state. |
| healthChecks | [HealthCheckStep](#github-com-akuity-kargo-api-v1alpha1-HealthCheckStep) |  HealthChecks contains the health check directives to be executed after the Promotion has completed. |
| startedAt | k8s.io.apimachinery.pkg.apis.meta.v1.Time |  StartedAt is the time when the promotion started. |
| finishedAt | k8s.io.apimachinery.pkg.apis.meta.v1.Time |  FinishedAt is the time when the promotion was completed. |
| currentStep | [int64](#int64) |  CurrentStep is the index of the current promotion step being executed. This permits steps that have already run successfully to be skipped on subsequent reconciliations attempts. |
| stepExecutionMetadata | [StepExecutionMetadata](#github-com-akuity-kargo-api-v1alpha1-StepExecutionMetadata) |  StepExecutionMetadata tracks metadata pertaining to the execution of individual promotion steps. |
| state | k8s.io.apiextensions_apiserver.pkg.apis.apiextensions.v1.JSON |  State stores the state of the promotion process between reconciliation attempts. |






<a name="github-com-akuity-kargo-api-v1alpha1-PromotionStep"></a>

### PromotionStep
PromotionStep describes a directive to be executed as part of a Promotion.


| Field | Type | Description |
| ----- | ---- | ----------- |
| uses | [string](#string) |  Uses identifies a runner that can execute this step.  +kubebuilder:validation:Optional +kubebuilder:validation:MinLength=1 |
| task | [PromotionTaskReference](#github-com-akuity-kargo-api-v1alpha1-PromotionTaskReference) |  Task is a reference to a PromotionTask that should be inflated into a Promotion when it is built from a PromotionTemplate. |
| as | [string](#string) |  As is the alias this step can be referred to as. |
| if | [string](#string) |  If is an optional expression that, if present, must evaluate to a boolean value. If the expression evaluates to false, the step will be skipped. If the expression does not evaluate to a boolean value, the step will be considered to have failed. |
| continueOnError | [bool](#bool) |  ContinueOnError is a boolean value that, if set to true, will cause the Promotion to continue executing the next step even if this step fails. It also will not permit this failure to impact the overall status of the Promotion. |
| retry | [PromotionStepRetry](#github-com-akuity-kargo-api-v1alpha1-PromotionStepRetry) |  Retry is the retry policy for this step. |
| vars | [ExpressionVariable](#github-com-akuity-kargo-api-v1alpha1-ExpressionVariable) |  Vars is a list of variables that can be referenced by expressions in the step's Config. The values override the values specified in the PromotionSpec. |
| config | k8s.io.apiextensions_apiserver.pkg.apis.apiextensions.v1.JSON |  Config is opaque configuration for the PromotionStep that is understood only by each PromotionStep's implementation. It is legal to utilize expressions in defining values at any level of this block. See https://docs.kargo.io/user-guide/reference-docs/expressions for details. |






<a name="github-com-akuity-kargo-api-v1alpha1-PromotionStepRetry"></a>

### PromotionStepRetry
PromotionStepRetry describes the retry policy for a PromotionStep.


| Field | Type | Description |
| ----- | ---- | ----------- |
| timeout | k8s.io.apimachinery.pkg.apis.meta.v1.Duration |  Timeout is the soft maximum interval in which a step that returns a Running status (which typically indicates it's waiting for something to happen) may be retried.  The maximum is a soft one because the check for whether the interval has elapsed occurs AFTER the step has run. This effectively means a step may run ONCE beyond the close of the interval.  If this field is set to nil, the effective default will be a step-specific one. If no step-specific default exists (i.e. is also nil), the effective default will be the system-wide default of 0.  A value of 0 will cause the step to be retried indefinitely unless the ErrorThreshold is reached. |
| errorThreshold | [uint32](#uint32) |  ErrorThreshold is the number of consecutive times the step must fail (for any reason) before retries are abandoned and the entire Promotion is marked as failed.  If this field is set to 0, the effective default will be a step-specific one. If no step-specific default exists (i.e. is also 0), the effective default will be the system-wide default of 1.  A value of 1 will cause the Promotion to be marked as failed after just a single failure; i.e. no retries will be attempted.  There is no option to specify an infinite number of retries using a value such as -1.  In a future release, Kargo is likely to become capable of distinguishing between recoverable and non-recoverable step failures. At that time, it is planned that unrecoverable failures will not be subject to this threshold and will immediately cause the Promotion to be marked as failed without further condition. |






<a name="github-com-akuity-kargo-api-v1alpha1-PromotionTask"></a>

### PromotionTask



| Field | Type | Description |
| ----- | ---- | ----------- |
| metadata | k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta |   |
| spec | [PromotionTaskSpec](#github-com-akuity-kargo-api-v1alpha1-PromotionTaskSpec) |  Spec describes the composition of a PromotionTask, including the variables available to the task and the steps.  +kubebuilder:validation:Required |






<a name="github-com-akuity-kargo-api-v1alpha1-PromotionTaskList"></a>

### PromotionTaskList
PromotionTaskList contains a list of PromotionTasks.


| Field | Type | Description |
| ----- | ---- | ----------- |
| metadata | k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta |   |
| items | [PromotionTask](#github-com-akuity-kargo-api-v1alpha1-PromotionTask) |   |






<a name="github-com-akuity-kargo-api-v1alpha1-PromotionTaskReference"></a>

### PromotionTaskReference
PromotionTaskReference describes a reference to a PromotionTask.


| Field | Type | Description |
| ----- | ---- | ----------- |
| name | [string](#string) |  Name is the name of the (Cluster)PromotionTask.  +kubebuilder:validation:Required +kubebuilder:validation:MinLength=1 +kubebuilder:validation:MaxLength=253 +kubebuilder:validation:Pattern=`^a-z0-9(-a-z0-9*a-z0-9)?(\.a-z0-9(-a-z0-9*a-z0-9)?)*$` +akuity:test-kubebuilder-pattern=KubernetesName |
| kind | [string](#string) |  Kind is the type of the PromotionTask. Can be either PromotionTask or ClusterPromotionTask, default is PromotionTask.  +kubebuilder:validation:Optional +kubebuilder:validation:Enum=PromotionTask;ClusterPromotionTask |






<a name="github-com-akuity-kargo-api-v1alpha1-PromotionTaskSpec"></a>

### PromotionTaskSpec



| Field | Type | Description |
| ----- | ---- | ----------- |
| vars | [ExpressionVariable](#github-com-akuity-kargo-api-v1alpha1-ExpressionVariable) |  Vars specifies the variables available to the PromotionTask. The values of these variables are the default values that can be overridden by the step referencing the task. |
| steps | [PromotionStep](#github-com-akuity-kargo-api-v1alpha1-PromotionStep) |  Steps specifies the directives to be executed as part of this PromotionTask. The steps as defined here are inflated into a Promotion when it is built from a PromotionTemplate.  +kubebuilder:validation:Required +kubebuilder:validation:MinItems=1 +kubebuilder:validation:items:XValidation:message="PromotionTask step must have uses set and must not reference another task",rule="has(self.uses) && !has(self.task)" |






<a name="github-com-akuity-kargo-api-v1alpha1-PromotionTemplate"></a>

### PromotionTemplate
PromotionTemplate defines a template for a Promotion that can be used to
incorporate Freight into a Stage.


| Field | Type | Description |
| ----- | ---- | ----------- |
| spec | [PromotionTemplateSpec](#github-com-akuity-kargo-api-v1alpha1-PromotionTemplateSpec) |   |






<a name="github-com-akuity-kargo-api-v1alpha1-PromotionTemplateSpec"></a>

### PromotionTemplateSpec
PromotionTemplateSpec describes the (partial) specification of a Promotion
for a Stage. This is a template that can be used to create a Promotion for a
Stage.


| Field | Type | Description |
| ----- | ---- | ----------- |
| vars | [ExpressionVariable](#github-com-akuity-kargo-api-v1alpha1-ExpressionVariable) |  Vars is a list of variables that can be referenced by expressions in promotion steps. |
| steps | [PromotionStep](#github-com-akuity-kargo-api-v1alpha1-PromotionStep) |  Steps specifies the directives to be executed as part of a Promotion. The order in which the directives are executed is the order in which they are listed in this field.  +kubebuilder:validation:MinItems=1 +kubebuilder:validation:items:XValidation:message="PromotionTemplate step must have exactly one of uses or task set",rule="(has(self.uses) ? !has(self.task) : has(self.task))" +kubebuilder:validation:items:XValidation:message="PromotionTemplate step referencing a task cannot set continueOnError",rule="!has(self.task) || !has(self.continueOnError)" +kubebuilder:validation:items:XValidation:message="PromotionTemplate step referencing a task cannot set retry",rule="!has(self.task) || !has(self.retry)" |






<a name="github-com-akuity-kargo-api-v1alpha1-QuayWebhookReceiverConfig"></a>

### QuayWebhookReceiverConfig
QuayWebhookReceiverConfig describes a webhook receiver that is compatible
with Quay.io payloads.


| Field | Type | Description |
| ----- | ---- | ----------- |
| secretRef | k8s.io.api.core.v1.LocalObjectReference |  SecretRef contains a reference to a Secret. For Project-scoped webhook receivers, the referenced Secret must be in the same namespace as the ProjectConfig.  For cluster-scoped webhook receivers, the referenced Secret must be in the designated "cluster Secrets" namespace.  The Secret's data map is expected to contain a `secret` key whose value does NOT need to be shared directly with Quay when registering a webhook. It is used only by Kargo to create a complex, hard-to-guess URL, which implicitly serves as a shared secret. For more information about Quay webhooks, please refer to the Quay documentation:   https://docs.quay.io/guides/notifications.html  +kubebuilder:validation:Required |






<a name="github-com-akuity-kargo-api-v1alpha1-RepoSubscription"></a>

### RepoSubscription
RepoSubscription describes a subscription to ONE OF a Git repository, a
container image repository, or a Helm chart repository.


| Field | Type | Description |
| ----- | ---- | ----------- |
| git | [GitSubscription](#github-com-akuity-kargo-api-v1alpha1-GitSubscription) |  Git describes a subscriptions to a Git repository. |
| image | [ImageSubscription](#github-com-akuity-kargo-api-v1alpha1-ImageSubscription) |  Image describes a subscription to container image repository. |
| chart | [ChartSubscription](#github-com-akuity-kargo-api-v1alpha1-ChartSubscription) |  Chart describes a subscription to a Helm chart repository. |






<a name="github-com-akuity-kargo-api-v1alpha1-Stage"></a>

### Stage
Stage is the Kargo API's main type.


| Field | Type | Description |
| ----- | ---- | ----------- |
| metadata | k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta |   |
| spec | [StageSpec](#github-com-akuity-kargo-api-v1alpha1-StageSpec) |  Spec describes sources of Freight used by the Stage and how to incorporate Freight into the Stage.  +kubebuilder:validation:Required |
| status | [StageStatus](#github-com-akuity-kargo-api-v1alpha1-StageStatus) |  Status describes the Stage's current and recent Freight, health, and more. |






<a name="github-com-akuity-kargo-api-v1alpha1-StageList"></a>

### StageList
StageList is a list of Stage resources.


| Field | Type | Description |
| ----- | ---- | ----------- |
| metadata | k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta |   |
| items | [Stage](#github-com-akuity-kargo-api-v1alpha1-Stage) |   |






<a name="github-com-akuity-kargo-api-v1alpha1-StageSpec"></a>

### StageSpec
StageSpec describes the sources of Freight used by a Stage and how to
incorporate Freight into the Stage.


| Field | Type | Description |
| ----- | ---- | ----------- |
| shard | [string](#string) |  Shard is the name of the shard that this Stage belongs to. This is an optional field. If not specified, the Stage will belong to the default shard. A defaulting webhook will sync the value of the kargo.akuity.io/shard label with the value of this field. When this field is empty, the webhook will ensure that label is absent. |
| vars | [ExpressionVariable](#github-com-akuity-kargo-api-v1alpha1-ExpressionVariable) |  Vars is a list of variables that can be referenced anywhere in the StageSpec that supports expressions. For example, the PromotionTemplate and arguments of the Verification. |
| requestedFreight | [FreightRequest](#github-com-akuity-kargo-api-v1alpha1-FreightRequest) |  RequestedFreight expresses the Stage's need for certain pieces of Freight, each having originated from a particular Warehouse. This list must be non-empty. In the common case, a Stage will request Freight having originated from just one specific Warehouse. In advanced cases, requesting Freight from multiple Warehouses provides a method of advancing new artifacts of different types through parallel pipelines at different speeds. This can be useful, for instance, if a Stage is home to multiple microservices that are independently versioned.  +kubebuilder:validation:MinItems=1 |
| promotionTemplate | [PromotionTemplate](#github-com-akuity-kargo-api-v1alpha1-PromotionTemplate) |  PromotionTemplate describes how to incorporate Freight into the Stage using a Promotion. |
| verification | [Verification](#github-com-akuity-kargo-api-v1alpha1-Verification) |  Verification describes how to verify a Stage's current Freight is fit for promotion downstream. |






<a name="github-com-akuity-kargo-api-v1alpha1-StageStats"></a>

### StageStats
StageStats contains a summary of the collective state of a Project's
Stages.


| Field | Type | Description |
| ----- | ---- | ----------- |
| count | [int64](#int64) |  Count contains the total number of Stages in the Project. |
| health | [HealthStats](#github-com-akuity-kargo-api-v1alpha1-HealthStats) |  Health contains a summary of the collective health of a Project's Stages. |






<a name="github-com-akuity-kargo-api-v1alpha1-StageStatus"></a>

### StageStatus
StageStatus describes a Stages's current and recent Freight, health, and
more.


| Field | Type | Description |
| ----- | ---- | ----------- |
| conditions | k8s.io.apimachinery.pkg.apis.meta.v1.Condition |  Conditions contains the last observations of the Stage's current state. +patchMergeKey=type +patchStrategy=merge +listType=map +listMapKey=type |
| lastHandledRefresh | [string](#string) |  LastHandledRefresh holds the value of the most recent AnnotationKeyRefresh annotation that was handled by the controller. This field can be used to determine whether the request to refresh the resource has been handled. +optional |
| freightHistory | [FreightCollection](#github-com-akuity-kargo-api-v1alpha1-FreightCollection) |  FreightHistory is a list of recent Freight selections that were deployed to the Stage. By default, the last ten Freight selections are stored. The first item in the list is the most recent Freight selection and currently deployed to the Stage, subsequent items are older selections. |
| freightSummary | [string](#string) |  FreightSummary is human-readable text maintained by the controller that summarizes what Freight is currently deployed to the Stage. For Stages that request a single piece of Freight AND the request has been fulfilled, this field will simply contain the name of the Freight. For Stages that request a single piece of Freight AND the request has NOT been fulfilled, or for Stages that request multiple pieces of Freight, this field will contain a summary of fulfilled/requested Freight. The existence of this field is a workaround for kubectl limitations so that this complex but valuable information can be displayed in a column in response to `kubectl get stages`. |
| health | [Health](#github-com-akuity-kargo-api-v1alpha1-Health) |  Health is the Stage's last observed health. |
| observedGeneration | [int64](#int64) |  ObservedGeneration represents the .metadata.generation that this Stage status was reconciled against. |
| currentPromotion | [PromotionReference](#github-com-akuity-kargo-api-v1alpha1-PromotionReference) |  CurrentPromotion is a reference to the currently Running promotion. |
| lastPromotion | [PromotionReference](#github-com-akuity-kargo-api-v1alpha1-PromotionReference) |  LastPromotion is a reference to the last completed promotion. |
| autoPromotionEnabled | [bool](#bool) |  AutoPromotionEnabled indicates whether automatic promotion is enabled for the Stage based on the ProjectConfig. |






<a name="github-com-akuity-kargo-api-v1alpha1-StepExecutionMetadata"></a>

### StepExecutionMetadata
StepExecutionMetadata tracks metadata pertaining to the execution of
a promotion step.


| Field | Type | Description |
| ----- | ---- | ----------- |
| alias | [string](#string) |  Alias is the alias of the step. |
| startedAt | k8s.io.apimachinery.pkg.apis.meta.v1.Time |  StartedAt is the time at which the first attempt to execute the step began. |
| finishedAt | k8s.io.apimachinery.pkg.apis.meta.v1.Time |  FinishedAt is the time at which the final attempt to execute the step completed. |
| errorCount | [uint32](#uint32) |  ErrorCount tracks consecutive failed attempts to execute the step. |
| status | [string](#string) |  Status is the high-level outcome of the step. |
| message | [string](#string) |  Message is a display message about the step, including any errors. |
| continueOnError | [bool](#bool) |  ContinueOnError is a boolean value that, if set to true, will cause the Promotion to continue executing the next step even if this step fails. It also will not permit this failure to impact the overall status of the Promotion. |






<a name="github-com-akuity-kargo-api-v1alpha1-Verification"></a>

### Verification
Verification describes how to verify that a Promotion has been successful
using Argo Rollouts AnalysisTemplates.


| Field | Type | Description |
| ----- | ---- | ----------- |
| analysisTemplates | [AnalysisTemplateReference](#github-com-akuity-kargo-api-v1alpha1-AnalysisTemplateReference) |  AnalysisTemplates is a list of AnalysisTemplates from which AnalysisRuns should be created to verify a Stage's current Freight is fit to be promoted downstream. |
| analysisRunMetadata | [AnalysisRunMetadata](#github-com-akuity-kargo-api-v1alpha1-AnalysisRunMetadata) |  AnalysisRunMetadata contains optional metadata that should be applied to all AnalysisRuns. |
| args | [AnalysisRunArgument](#github-com-akuity-kargo-api-v1alpha1-AnalysisRunArgument) |  Args lists arguments that should be added to all AnalysisRuns. |






<a name="github-com-akuity-kargo-api-v1alpha1-VerificationInfo"></a>

### VerificationInfo
VerificationInfo contains the details of an instance of a Verification
process.


| Field | Type | Description |
| ----- | ---- | ----------- |
| id | [string](#string) |  ID is the identifier of the Verification process. |
| actor | [string](#string) |  Actor is the name of the entity that initiated or aborted the Verification process. |
| startTime | k8s.io.apimachinery.pkg.apis.meta.v1.Time |  StartTime is the time at which the Verification process was started. |
| phase | [string](#string) |  Phase describes the current phase of the Verification process. Generally, this will be a reflection of the underlying AnalysisRun's phase, however, there are exceptions to this, such as in the case where an AnalysisRun cannot be launched successfully. |
| message | [string](#string) |  Message may contain additional information about why the verification process is in its current phase. |
| analysisRun | [AnalysisRunReference](#github-com-akuity-kargo-api-v1alpha1-AnalysisRunReference) |  AnalysisRun is a reference to the Argo Rollouts AnalysisRun that implements the Verification process. |
| finishTime | k8s.io.apimachinery.pkg.apis.meta.v1.Time |  FinishTime is the time at which the Verification process finished. |






<a name="github-com-akuity-kargo-api-v1alpha1-VerifiedStage"></a>

### VerifiedStage
VerifiedStage describes a Stage in which Freight has been verified.


| Field | Type | Description |
| ----- | ---- | ----------- |
| verifiedAt | k8s.io.apimachinery.pkg.apis.meta.v1.Time |  VerifiedAt is the time at which the Freight was verified in the Stage. |
| longestSoak | k8s.io.apimachinery.pkg.apis.meta.v1.Duration |  LongestCompletedSoak represents the longest definite time interval wherein the Freight was in CONTINUOUS use by the Stage. This value is updated as Freight EXITS the Stage. If the Freight is currently in use by the Stage, the time elapsed since the Freight ENTERED the Stage is its current soak time, which may exceed the value of this field. |






<a name="github-com-akuity-kargo-api-v1alpha1-Warehouse"></a>

### Warehouse
Warehouse is a source of Freight.


| Field | Type | Description |
| ----- | ---- | ----------- |
| metadata | k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta |   |
| spec | [WarehouseSpec](#github-com-akuity-kargo-api-v1alpha1-WarehouseSpec) |  Spec describes sources of artifacts.  +kubebuilder:validation:Required |
| status | [WarehouseStatus](#github-com-akuity-kargo-api-v1alpha1-WarehouseStatus) |  Status describes the Warehouse's most recently observed state. |






<a name="github-com-akuity-kargo-api-v1alpha1-WarehouseList"></a>

### WarehouseList
WarehouseList is a list of Warehouse resources.


| Field | Type | Description |
| ----- | ---- | ----------- |
| metadata | k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta |   |
| items | [Warehouse](#github-com-akuity-kargo-api-v1alpha1-Warehouse) |   |






<a name="github-com-akuity-kargo-api-v1alpha1-WarehouseSpec"></a>

### WarehouseSpec
WarehouseSpec describes sources of versioned artifacts to be included in
Freight produced by this Warehouse.


| Field | Type | Description |
| ----- | ---- | ----------- |
| shard | [string](#string) |  Shard is the name of the shard that this Warehouse belongs to. This is an optional field. If not specified, the Warehouse will belong to the default shard. A defaulting webhook will sync this field with the value of the kargo.akuity.io/shard label. When the shard label is not present or differs from the value of this field, the defaulting webhook will set the label to the value of this field. If the shard label is present and this field is empty, the defaulting webhook will set the value of this field to the value of the shard label. |
| interval | k8s.io.apimachinery.pkg.apis.meta.v1.Duration |  Interval is the reconciliation interval for this Warehouse. On each reconciliation, the Warehouse will discover new artifacts and optionally produce new Freight. This field is optional. When left unspecified, the field is implicitly treated as if its value were "5m0s".  +kubebuilder:validation:Type=string +kubebuilder:validation:Pattern=`^(0-9+(\.0-9+)?(s|m|h))+$` +kubebuilder:default="5m0s" +akuity:test-kubebuilder-pattern=Duration |
| freightCreationPolicy | [string](#string) |  FreightCreationPolicy describes how Freight is created by this Warehouse. This field is optional. When left unspecified, the field is implicitly treated as if its value were "Automatic".  Accepted values:  - "Automatic": New Freight is created automatically when any new artifact   is discovered. - "Manual": New Freight is never created automatically.  +kubebuilder:default=Automatic +kubebuilder:validation:Optional |
| subscriptions | [RepoSubscription](#github-com-akuity-kargo-api-v1alpha1-RepoSubscription) |  Subscriptions describes sources of artifacts to be included in Freight produced by this Warehouse.  +kubebuilder:validation:MinItems=1 |






<a name="github-com-akuity-kargo-api-v1alpha1-WarehouseStats"></a>

### WarehouseStats
WarehouseStats contains a summary of the collective state of a Project's
Warehouses.


| Field | Type | Description |
| ----- | ---- | ----------- |
| count | [int64](#int64) |  Count contains the total number of Warehouses in the Project. |
| health | [HealthStats](#github-com-akuity-kargo-api-v1alpha1-HealthStats) |  Health contains a summary of the collective health of a Project's Warehouses. |






<a name="github-com-akuity-kargo-api-v1alpha1-WarehouseStatus"></a>

### WarehouseStatus
WarehouseStatus describes a Warehouse's most recently observed state.


| Field | Type | Description |
| ----- | ---- | ----------- |
| conditions | k8s.io.apimachinery.pkg.apis.meta.v1.Condition |  Conditions contains the last observations of the Warehouse's current state. +patchMergeKey=type +patchStrategy=merge +listType=map +listMapKey=type |
| lastHandledRefresh | [string](#string) |  LastHandledRefresh holds the value of the most recent AnnotationKeyRefresh annotation that was handled by the controller. This field can be used to determine whether the request to refresh the resource has been handled. +optional |
| observedGeneration | [int64](#int64) |  ObservedGeneration represents the .metadata.generation that this Warehouse was reconciled against. |
| lastFreightID | [string](#string) |  LastFreightID is a reference to the system-assigned identifier (name) of the most recent Freight produced by the Warehouse. |
| discoveredArtifacts | [DiscoveredArtifacts](#github-com-akuity-kargo-api-v1alpha1-DiscoveredArtifacts) |  DiscoveredArtifacts holds the artifacts discovered by the Warehouse. |






<a name="github-com-akuity-kargo-api-v1alpha1-WebhookReceiverConfig"></a>

### WebhookReceiverConfig
WebhookReceiverConfig describes the configuration for a single webhook
receiver.


| Field | Type | Description |
| ----- | ---- | ----------- |
| name | [string](#string) |  Name is the name of the webhook receiver.  +kubebuilder:validation:Required +kubebuilder:validation:MinLength=1 +kubebuilder:validation:MaxLength=253 +kubebuilder:validation:Pattern=`^a-z0-9(-a-z0-9*a-z0-9)?(\.a-z0-9(-a-z0-9*a-z0-9)?)*$` +akuity:test-kubebuilder-pattern=KubernetesName |
| bitbucket | [BitbucketWebhookReceiverConfig](#github-com-akuity-kargo-api-v1alpha1-BitbucketWebhookReceiverConfig) |  Bitbucket contains the configuration for a webhook receiver that is compatible with Bitbucket payloads. |
| dockerhub | [DockerHubWebhookReceiverConfig](#github-com-akuity-kargo-api-v1alpha1-DockerHubWebhookReceiverConfig) |  DockerHub contains the configuration for a webhook receiver that is compatible with DockerHub payloads. |
| github | [GitHubWebhookReceiverConfig](#github-com-akuity-kargo-api-v1alpha1-GitHubWebhookReceiverConfig) |  GitHub contains the configuration for a webhook receiver that is compatible with GitHub payloads. |
| gitlab | [GitLabWebhookReceiverConfig](#github-com-akuity-kargo-api-v1alpha1-GitLabWebhookReceiverConfig) |  GitLab contains the configuration for a webhook receiver that is compatible with GitLab payloads. |
| harbor | [HarborWebhookReceiverConfig](#github-com-akuity-kargo-api-v1alpha1-HarborWebhookReceiverConfig) |  Harbor contains the configuration for a webhook receiver that is compatible with Harbor payloads. |
| quay | [QuayWebhookReceiverConfig](#github-com-akuity-kargo-api-v1alpha1-QuayWebhookReceiverConfig) |  Quay contains the configuration for a webhook receiver that is compatible with Quay payloads. |
| artifactory | [ArtifactoryWebhookReceiverConfig](#github-com-akuity-kargo-api-v1alpha1-ArtifactoryWebhookReceiverConfig) |  Artifactory contains the configuration for a webhook receiver that is compatible with JFrog Artifactory payloads. |
| azure | [AzureWebhookReceiverConfig](#github-com-akuity-kargo-api-v1alpha1-AzureWebhookReceiverConfig) |  Azure contains the configuration for a webhook receiver that is compatible with Azure Container Registry (ACR) and Azure DevOps payloads. |
| gitea | [GiteaWebhookReceiverConfig](#github-com-akuity-kargo-api-v1alpha1-GiteaWebhookReceiverConfig) |  Gitea contains the configuration for a webhook receiver that is compatible with Gitea payloads. |






<a name="github-com-akuity-kargo-api-v1alpha1-WebhookReceiverDetails"></a>

### WebhookReceiverDetails
WebhookReceiverDetails encapsulates the details of a webhook receiver.


| Field | Type | Description |
| ----- | ---- | ----------- |
| name | [string](#string) |  Name is the name of the webhook receiver. |
| path | [string](#string) |  Path is the path to the receiver's webhook endpoint. |
| url | [string](#string) |  URL includes the full address of the receiver's webhook endpoint. |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="api_stubs_rollouts_v1alpha1_generated-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/stubs/rollouts/v1alpha1/generated.proto



<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisRun"></a>

### AnalysisRun



| Field | Type | Description |
| ----- | ---- | ----------- |
| metadata | k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta |   |
| spec | [AnalysisRunSpec](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisRunSpec) |   |
| status | [AnalysisRunStatus](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisRunStatus) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisRunList"></a>

### AnalysisRunList



| Field | Type | Description |
| ----- | ---- | ----------- |
| metadata | k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta |   |
| items | [AnalysisRun](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisRun) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisRunSpec"></a>

### AnalysisRunSpec



| Field | Type | Description |
| ----- | ---- | ----------- |
| metrics | [Metric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Metric) |   |
| args | [Argument](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Argument) |   |
| terminate | [bool](#bool) |   |
| dryRun | [DryRun](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-DryRun) |   |
| measurementRetention | [MeasurementRetention](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-MeasurementRetention) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisRunStatus"></a>

### AnalysisRunStatus



| Field | Type | Description |
| ----- | ---- | ----------- |
| phase | [string](#string) |   |
| message | [string](#string) |   |
| metricResults | [MetricResult](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-MetricResult) |   |
| startedAt | k8s.io.apimachinery.pkg.apis.meta.v1.Time |   |
| runSummary | [RunSummary](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-RunSummary) |   |
| dryRunSummary | [RunSummary](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-RunSummary) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisTemplate"></a>

### AnalysisTemplate



| Field | Type | Description |
| ----- | ---- | ----------- |
| metadata | k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta |   |
| spec | [AnalysisTemplateSpec](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisTemplateSpec) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisTemplateList"></a>

### AnalysisTemplateList



| Field | Type | Description |
| ----- | ---- | ----------- |
| metadata | k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta |   |
| items | [AnalysisTemplate](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisTemplate) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisTemplateSpec"></a>

### AnalysisTemplateSpec



| Field | Type | Description |
| ----- | ---- | ----------- |
| metrics | [Metric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Metric) |   |
| args | [Argument](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Argument) |   |
| dryRun | [DryRun](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-DryRun) |   |
| measurementRetention | [MeasurementRetention](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-MeasurementRetention) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Argument"></a>

### Argument



| Field | Type | Description |
| ----- | ---- | ----------- |
| name | [string](#string) |   |
| value | [string](#string) |   |
| valueFrom | [ValueFrom](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-ValueFrom) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Authentication"></a>

### Authentication



| Field | Type | Description |
| ----- | ---- | ----------- |
| sigv4 | [Sigv4Config](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Sigv4Config) |   |
| oauth2 | [OAuth2Config](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-OAuth2Config) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-CloudWatchMetric"></a>

### CloudWatchMetric



| Field | Type | Description |
| ----- | ---- | ----------- |
| interval | [string](#string) |   |
| metricDataQueries | [CloudWatchMetricDataQuery](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-CloudWatchMetricDataQuery) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-CloudWatchMetricDataQuery"></a>

### CloudWatchMetricDataQuery



| Field | Type | Description |
| ----- | ---- | ----------- |
| id | [string](#string) |   |
| expression | [string](#string) |   |
| label | [string](#string) |   |
| metricStat | [CloudWatchMetricStat](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-CloudWatchMetricStat) |   |
| period | k8s.io.apimachinery.pkg.util.intstr.IntOrString |   |
| returnData | [bool](#bool) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-CloudWatchMetricStat"></a>

### CloudWatchMetricStat



| Field | Type | Description |
| ----- | ---- | ----------- |
| metric | [CloudWatchMetricStatMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-CloudWatchMetricStatMetric) |   |
| period | k8s.io.apimachinery.pkg.util.intstr.IntOrString |   |
| stat | [string](#string) |   |
| unit | [string](#string) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-CloudWatchMetricStatMetric"></a>

### CloudWatchMetricStatMetric



| Field | Type | Description |
| ----- | ---- | ----------- |
| dimensions | [CloudWatchMetricStatMetricDimension](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-CloudWatchMetricStatMetricDimension) |   |
| metricName | [string](#string) |   |
| namespace | [string](#string) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-CloudWatchMetricStatMetricDimension"></a>

### CloudWatchMetricStatMetricDimension



| Field | Type | Description |
| ----- | ---- | ----------- |
| name | [string](#string) |   |
| value | [string](#string) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-ClusterAnalysisTemplate"></a>

### ClusterAnalysisTemplate



| Field | Type | Description |
| ----- | ---- | ----------- |
| metadata | k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta |   |
| spec | [AnalysisTemplateSpec](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisTemplateSpec) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-ClusterAnalysisTemplateList"></a>

### ClusterAnalysisTemplateList



| Field | Type | Description |
| ----- | ---- | ----------- |
| metadata | k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta |   |
| items | [ClusterAnalysisTemplate](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-ClusterAnalysisTemplate) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-DatadogMetric"></a>

### DatadogMetric



| Field | Type | Description |
| ----- | ---- | ----------- |
| interval | [string](#string) |   |
| query | [string](#string) |   |
| queries | [DatadogMetric.QueriesEntry](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-DatadogMetric-QueriesEntry) |   |
| formula | [string](#string) |   |
| apiVersion | [string](#string) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-DatadogMetric-QueriesEntry"></a>

### DatadogMetric.QueriesEntry



| Field | Type | Description |
| ----- | ---- | ----------- |
| key | [string](#string) |   |
| value | [string](#string) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-DryRun"></a>

### DryRun



| Field | Type | Description |
| ----- | ---- | ----------- |
| metricName | [string](#string) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-FieldRef"></a>

### FieldRef



| Field | Type | Description |
| ----- | ---- | ----------- |
| fieldPath | [string](#string) |  Required: Path of the field to select in the specified API version |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-GraphiteMetric"></a>

### GraphiteMetric



| Field | Type | Description |
| ----- | ---- | ----------- |
| address | [string](#string) |   |
| query | [string](#string) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-InfluxdbMetric"></a>

### InfluxdbMetric



| Field | Type | Description |
| ----- | ---- | ----------- |
| profile | [string](#string) |   |
| query | [string](#string) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-JobMetric"></a>

### JobMetric



| Field | Type | Description |
| ----- | ---- | ----------- |
| metadata | k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta |   |
| spec | k8s.io.api.batch.v1.JobSpec |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-KayentaMetric"></a>

### KayentaMetric



| Field | Type | Description |
| ----- | ---- | ----------- |
| address | [string](#string) |   |
| application | [string](#string) |   |
| canaryConfigName | [string](#string) |   |
| metricsAccountName | [string](#string) |   |
| configurationAccountName | [string](#string) |   |
| storageAccountName | [string](#string) |   |
| threshold | [KayentaThreshold](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-KayentaThreshold) |   |
| scopes | [KayentaScope](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-KayentaScope) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-KayentaScope"></a>

### KayentaScope



| Field | Type | Description |
| ----- | ---- | ----------- |
| name | [string](#string) |   |
| controlScope | [ScopeDetail](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-ScopeDetail) |   |
| experimentScope | [ScopeDetail](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-ScopeDetail) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-KayentaThreshold"></a>

### KayentaThreshold



| Field | Type | Description |
| ----- | ---- | ----------- |
| pass | [int64](#int64) |   |
| marginal | [int64](#int64) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Measurement"></a>

### Measurement



| Field | Type | Description |
| ----- | ---- | ----------- |
| phase | [string](#string) |   |
| message | [string](#string) |   |
| startedAt | k8s.io.apimachinery.pkg.apis.meta.v1.Time |   |
| finishedAt | k8s.io.apimachinery.pkg.apis.meta.v1.Time |   |
| value | [string](#string) |   |
| metadata | [Measurement.MetadataEntry](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Measurement-MetadataEntry) |   |
| resumeAt | k8s.io.apimachinery.pkg.apis.meta.v1.Time |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Measurement-MetadataEntry"></a>

### Measurement.MetadataEntry



| Field | Type | Description |
| ----- | ---- | ----------- |
| key | [string](#string) |   |
| value | [string](#string) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-MeasurementRetention"></a>

### MeasurementRetention



| Field | Type | Description |
| ----- | ---- | ----------- |
| metricName | [string](#string) |   |
| limit | [int32](#int32) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Metric"></a>

### Metric



| Field | Type | Description |
| ----- | ---- | ----------- |
| name | [string](#string) |   |
| interval | [string](#string) |   |
| initialDelay | [string](#string) |   |
| count | k8s.io.apimachinery.pkg.util.intstr.IntOrString |   |
| successCondition | [string](#string) |   |
| failureCondition | [string](#string) |   |
| failureLimit | k8s.io.apimachinery.pkg.util.intstr.IntOrString |   |
| inconclusiveLimit | k8s.io.apimachinery.pkg.util.intstr.IntOrString |   |
| consecutiveErrorLimit | k8s.io.apimachinery.pkg.util.intstr.IntOrString |   |
| provider | [MetricProvider](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-MetricProvider) |   |
| consecutiveSuccessLimit | k8s.io.apimachinery.pkg.util.intstr.IntOrString |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-MetricProvider"></a>

### MetricProvider



| Field | Type | Description |
| ----- | ---- | ----------- |
| prometheus | [PrometheusMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-PrometheusMetric) |   |
| kayenta | [KayentaMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-KayentaMetric) |   |
| web | [WebMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-WebMetric) |   |
| datadog | [DatadogMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-DatadogMetric) |   |
| wavefront | [WavefrontMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-WavefrontMetric) |   |
| newRelic | [NewRelicMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-NewRelicMetric) |   |
| job | [JobMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-JobMetric) |   |
| cloudWatch | [CloudWatchMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-CloudWatchMetric) |   |
| graphite | [GraphiteMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-GraphiteMetric) |   |
| influxdb | [InfluxdbMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-InfluxdbMetric) |   |
| skywalking | [SkyWalkingMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-SkyWalkingMetric) |   |
| plugin | [MetricProvider.PluginEntry](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-MetricProvider-PluginEntry) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-MetricProvider-PluginEntry"></a>

### MetricProvider.PluginEntry



| Field | Type | Description |
| ----- | ---- | ----------- |
| key | [string](#string) |   |
| value | [bytes](#bytes) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-MetricResult"></a>

### MetricResult



| Field | Type | Description |
| ----- | ---- | ----------- |
| name | [string](#string) |   |
| phase | [string](#string) |   |
| measurements | [Measurement](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Measurement) |   |
| message | [string](#string) |   |
| count | [int32](#int32) |   |
| successful | [int32](#int32) |   |
| failed | [int32](#int32) |   |
| inconclusive | [int32](#int32) |   |
| error | [int32](#int32) |   |
| consecutiveError | [int32](#int32) |   |
| dryRun | [bool](#bool) |   |
| metadata | [MetricResult.MetadataEntry](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-MetricResult-MetadataEntry) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-MetricResult-MetadataEntry"></a>

### MetricResult.MetadataEntry



| Field | Type | Description |
| ----- | ---- | ----------- |
| key | [string](#string) |   |
| value | [string](#string) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-NewRelicMetric"></a>

### NewRelicMetric



| Field | Type | Description |
| ----- | ---- | ----------- |
| profile | [string](#string) |   |
| query | [string](#string) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-OAuth2Config"></a>

### OAuth2Config



| Field | Type | Description |
| ----- | ---- | ----------- |
| tokenUrl | [string](#string) |   |
| clientId | [string](#string) |   |
| clientSecret | [string](#string) |   |
| scopes | [string](#string) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-PrometheusMetric"></a>

### PrometheusMetric



| Field | Type | Description |
| ----- | ---- | ----------- |
| address | [string](#string) |   |
| query | [string](#string) |   |
| authentication | [Authentication](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Authentication) |   |
| timeout | [int64](#int64) |   |
| insecure | [bool](#bool) |   |
| headers | [WebMetricHeader](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-WebMetricHeader) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-RunSummary"></a>

### RunSummary



| Field | Type | Description |
| ----- | ---- | ----------- |
| count | [int32](#int32) |   |
| successful | [int32](#int32) |   |
| failed | [int32](#int32) |   |
| inconclusive | [int32](#int32) |   |
| error | [int32](#int32) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-ScopeDetail"></a>

### ScopeDetail



| Field | Type | Description |
| ----- | ---- | ----------- |
| scope | [string](#string) |   |
| region | [string](#string) |   |
| step | [int64](#int64) |   |
| start | [string](#string) |   |
| end | [string](#string) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-SecretKeyRef"></a>

### SecretKeyRef



| Field | Type | Description |
| ----- | ---- | ----------- |
| name | [string](#string) |   |
| key | [string](#string) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Sigv4Config"></a>

### Sigv4Config



| Field | Type | Description |
| ----- | ---- | ----------- |
| region | [string](#string) |   |
| profile | [string](#string) |   |
| roleArn | [string](#string) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-SkyWalkingMetric"></a>

### SkyWalkingMetric



| Field | Type | Description |
| ----- | ---- | ----------- |
| address | [string](#string) |   |
| query | [string](#string) |   |
| interval | [string](#string) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-ValueFrom"></a>

### ValueFrom



| Field | Type | Description |
| ----- | ---- | ----------- |
| secretKeyRef | [SecretKeyRef](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-SecretKeyRef) |   |
| fieldRef | [FieldRef](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-FieldRef) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-WavefrontMetric"></a>

### WavefrontMetric



| Field | Type | Description |
| ----- | ---- | ----------- |
| address | [string](#string) |   |
| query | [string](#string) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-WebMetric"></a>

### WebMetric



| Field | Type | Description |
| ----- | ---- | ----------- |
| method | [string](#string) |   |
| url | [string](#string) |  URL is the address of the web metric |
| headers | [WebMetricHeader](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-WebMetricHeader) |   |
| body | [string](#string) |   |
| timeoutSeconds | [int64](#int64) |   |
| jsonPath | [string](#string) |   |
| insecure | [bool](#bool) |   |
| jsonBody | [bytes](#bytes) |   |
| authentication | [Authentication](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Authentication) |   |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-WebMetricHeader"></a>

### WebMetricHeader



| Field | Type | Description |
| ----- | ---- | ----------- |
| key | [string](#string) |   |
| value | [string](#string) |   |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



## Scalar Value Types

| .proto Type | Notes | C++ | Java | Python | Go | C# | PHP | Ruby |
| ----------- | ----- | --- | ---- | ------ | -- | -- | --- | ---- |
| <a name="double" /> double |  | double | double | float | float64 | double | float | Float |
| <a name="float" /> float |  | float | float | float | float32 | float | float | Float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum or Fixnum (as required) |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="bool" /> bool |  | bool | boolean | boolean | bool | bool | boolean | TrueClass/FalseClass |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode | string | string | string | String (UTF-8) |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str | []byte | ByteString | string | String (ASCII-8BIT) |

