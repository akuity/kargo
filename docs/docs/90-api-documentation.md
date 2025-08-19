# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [api/service/v1alpha1/service.proto](#api_service_v1alpha1_service-proto)
    - [AbortPromotionRequest](#akuity-io-kargo-service-v1alpha1-AbortPromotionRequest)
    - [AbortPromotionResponse](#akuity-io-kargo-service-v1alpha1-AbortPromotionResponse)
    - [AbortVerificationRequest](#akuity-io-kargo-service-v1alpha1-AbortVerificationRequest)
    - [AbortVerificationResponse](#akuity-io-kargo-service-v1alpha1-AbortVerificationResponse)
    - [AdminLoginRequest](#akuity-io-kargo-service-v1alpha1-AdminLoginRequest)
    - [AdminLoginResponse](#akuity-io-kargo-service-v1alpha1-AdminLoginResponse)
    - [ApproveFreightRequest](#akuity-io-kargo-service-v1alpha1-ApproveFreightRequest)
    - [ApproveFreightResponse](#akuity-io-kargo-service-v1alpha1-ApproveFreightResponse)
    - [ArgoCDShard](#akuity-io-kargo-service-v1alpha1-ArgoCDShard)
    - [Claims](#akuity-io-kargo-service-v1alpha1-Claims)
    - [ComponentVersions](#akuity-io-kargo-service-v1alpha1-ComponentVersions)
    - [CreateClusterSecretRequest](#akuity-io-kargo-service-v1alpha1-CreateClusterSecretRequest)
    - [CreateClusterSecretRequest.DataEntry](#akuity-io-kargo-service-v1alpha1-CreateClusterSecretRequest-DataEntry)
    - [CreateClusterSecretResponse](#akuity-io-kargo-service-v1alpha1-CreateClusterSecretResponse)
    - [CreateCredentialsRequest](#akuity-io-kargo-service-v1alpha1-CreateCredentialsRequest)
    - [CreateCredentialsResponse](#akuity-io-kargo-service-v1alpha1-CreateCredentialsResponse)
    - [CreateOrUpdateResourceRequest](#akuity-io-kargo-service-v1alpha1-CreateOrUpdateResourceRequest)
    - [CreateOrUpdateResourceResponse](#akuity-io-kargo-service-v1alpha1-CreateOrUpdateResourceResponse)
    - [CreateOrUpdateResourceResult](#akuity-io-kargo-service-v1alpha1-CreateOrUpdateResourceResult)
    - [CreateProjectSecretRequest](#akuity-io-kargo-service-v1alpha1-CreateProjectSecretRequest)
    - [CreateProjectSecretRequest.DataEntry](#akuity-io-kargo-service-v1alpha1-CreateProjectSecretRequest-DataEntry)
    - [CreateProjectSecretResponse](#akuity-io-kargo-service-v1alpha1-CreateProjectSecretResponse)
    - [CreateResourceRequest](#akuity-io-kargo-service-v1alpha1-CreateResourceRequest)
    - [CreateResourceResponse](#akuity-io-kargo-service-v1alpha1-CreateResourceResponse)
    - [CreateResourceResult](#akuity-io-kargo-service-v1alpha1-CreateResourceResult)
    - [CreateRoleRequest](#akuity-io-kargo-service-v1alpha1-CreateRoleRequest)
    - [CreateRoleResponse](#akuity-io-kargo-service-v1alpha1-CreateRoleResponse)
    - [DeleteAnalysisTemplateRequest](#akuity-io-kargo-service-v1alpha1-DeleteAnalysisTemplateRequest)
    - [DeleteAnalysisTemplateResponse](#akuity-io-kargo-service-v1alpha1-DeleteAnalysisTemplateResponse)
    - [DeleteClusterAnalysisTemplateRequest](#akuity-io-kargo-service-v1alpha1-DeleteClusterAnalysisTemplateRequest)
    - [DeleteClusterAnalysisTemplateResponse](#akuity-io-kargo-service-v1alpha1-DeleteClusterAnalysisTemplateResponse)
    - [DeleteClusterConfigRequest](#akuity-io-kargo-service-v1alpha1-DeleteClusterConfigRequest)
    - [DeleteClusterConfigResponse](#akuity-io-kargo-service-v1alpha1-DeleteClusterConfigResponse)
    - [DeleteClusterSecretRequest](#akuity-io-kargo-service-v1alpha1-DeleteClusterSecretRequest)
    - [DeleteClusterSecretResponse](#akuity-io-kargo-service-v1alpha1-DeleteClusterSecretResponse)
    - [DeleteCredentialsRequest](#akuity-io-kargo-service-v1alpha1-DeleteCredentialsRequest)
    - [DeleteCredentialsResponse](#akuity-io-kargo-service-v1alpha1-DeleteCredentialsResponse)
    - [DeleteFreightRequest](#akuity-io-kargo-service-v1alpha1-DeleteFreightRequest)
    - [DeleteFreightResponse](#akuity-io-kargo-service-v1alpha1-DeleteFreightResponse)
    - [DeleteProjectConfigRequest](#akuity-io-kargo-service-v1alpha1-DeleteProjectConfigRequest)
    - [DeleteProjectConfigResponse](#akuity-io-kargo-service-v1alpha1-DeleteProjectConfigResponse)
    - [DeleteProjectRequest](#akuity-io-kargo-service-v1alpha1-DeleteProjectRequest)
    - [DeleteProjectResponse](#akuity-io-kargo-service-v1alpha1-DeleteProjectResponse)
    - [DeleteProjectSecretRequest](#akuity-io-kargo-service-v1alpha1-DeleteProjectSecretRequest)
    - [DeleteProjectSecretResponse](#akuity-io-kargo-service-v1alpha1-DeleteProjectSecretResponse)
    - [DeleteResourceRequest](#akuity-io-kargo-service-v1alpha1-DeleteResourceRequest)
    - [DeleteResourceResponse](#akuity-io-kargo-service-v1alpha1-DeleteResourceResponse)
    - [DeleteResourceResult](#akuity-io-kargo-service-v1alpha1-DeleteResourceResult)
    - [DeleteRoleRequest](#akuity-io-kargo-service-v1alpha1-DeleteRoleRequest)
    - [DeleteRoleResponse](#akuity-io-kargo-service-v1alpha1-DeleteRoleResponse)
    - [DeleteStageRequest](#akuity-io-kargo-service-v1alpha1-DeleteStageRequest)
    - [DeleteStageResponse](#akuity-io-kargo-service-v1alpha1-DeleteStageResponse)
    - [DeleteWarehouseRequest](#akuity-io-kargo-service-v1alpha1-DeleteWarehouseRequest)
    - [DeleteWarehouseResponse](#akuity-io-kargo-service-v1alpha1-DeleteWarehouseResponse)
    - [FreightList](#akuity-io-kargo-service-v1alpha1-FreightList)
    - [GetAnalysisRunLogsRequest](#akuity-io-kargo-service-v1alpha1-GetAnalysisRunLogsRequest)
    - [GetAnalysisRunLogsResponse](#akuity-io-kargo-service-v1alpha1-GetAnalysisRunLogsResponse)
    - [GetAnalysisRunRequest](#akuity-io-kargo-service-v1alpha1-GetAnalysisRunRequest)
    - [GetAnalysisRunResponse](#akuity-io-kargo-service-v1alpha1-GetAnalysisRunResponse)
    - [GetAnalysisTemplateRequest](#akuity-io-kargo-service-v1alpha1-GetAnalysisTemplateRequest)
    - [GetAnalysisTemplateResponse](#akuity-io-kargo-service-v1alpha1-GetAnalysisTemplateResponse)
    - [GetClusterAnalysisTemplateRequest](#akuity-io-kargo-service-v1alpha1-GetClusterAnalysisTemplateRequest)
    - [GetClusterAnalysisTemplateResponse](#akuity-io-kargo-service-v1alpha1-GetClusterAnalysisTemplateResponse)
    - [GetClusterConfigRequest](#akuity-io-kargo-service-v1alpha1-GetClusterConfigRequest)
    - [GetClusterConfigResponse](#akuity-io-kargo-service-v1alpha1-GetClusterConfigResponse)
    - [GetClusterPromotionTaskRequest](#akuity-io-kargo-service-v1alpha1-GetClusterPromotionTaskRequest)
    - [GetClusterPromotionTaskResponse](#akuity-io-kargo-service-v1alpha1-GetClusterPromotionTaskResponse)
    - [GetConfigMapRequest](#akuity-io-kargo-service-v1alpha1-GetConfigMapRequest)
    - [GetConfigMapResponse](#akuity-io-kargo-service-v1alpha1-GetConfigMapResponse)
    - [GetConfigRequest](#akuity-io-kargo-service-v1alpha1-GetConfigRequest)
    - [GetConfigResponse](#akuity-io-kargo-service-v1alpha1-GetConfigResponse)
    - [GetConfigResponse.ArgocdShardsEntry](#akuity-io-kargo-service-v1alpha1-GetConfigResponse-ArgocdShardsEntry)
    - [GetCredentialsRequest](#akuity-io-kargo-service-v1alpha1-GetCredentialsRequest)
    - [GetCredentialsResponse](#akuity-io-kargo-service-v1alpha1-GetCredentialsResponse)
    - [GetFreightRequest](#akuity-io-kargo-service-v1alpha1-GetFreightRequest)
    - [GetFreightResponse](#akuity-io-kargo-service-v1alpha1-GetFreightResponse)
    - [GetProjectConfigRequest](#akuity-io-kargo-service-v1alpha1-GetProjectConfigRequest)
    - [GetProjectConfigResponse](#akuity-io-kargo-service-v1alpha1-GetProjectConfigResponse)
    - [GetProjectRequest](#akuity-io-kargo-service-v1alpha1-GetProjectRequest)
    - [GetProjectResponse](#akuity-io-kargo-service-v1alpha1-GetProjectResponse)
    - [GetPromotionRequest](#akuity-io-kargo-service-v1alpha1-GetPromotionRequest)
    - [GetPromotionResponse](#akuity-io-kargo-service-v1alpha1-GetPromotionResponse)
    - [GetPromotionTaskRequest](#akuity-io-kargo-service-v1alpha1-GetPromotionTaskRequest)
    - [GetPromotionTaskResponse](#akuity-io-kargo-service-v1alpha1-GetPromotionTaskResponse)
    - [GetPublicConfigRequest](#akuity-io-kargo-service-v1alpha1-GetPublicConfigRequest)
    - [GetPublicConfigResponse](#akuity-io-kargo-service-v1alpha1-GetPublicConfigResponse)
    - [GetRoleRequest](#akuity-io-kargo-service-v1alpha1-GetRoleRequest)
    - [GetRoleResponse](#akuity-io-kargo-service-v1alpha1-GetRoleResponse)
    - [GetStageRequest](#akuity-io-kargo-service-v1alpha1-GetStageRequest)
    - [GetStageResponse](#akuity-io-kargo-service-v1alpha1-GetStageResponse)
    - [GetVersionInfoRequest](#akuity-io-kargo-service-v1alpha1-GetVersionInfoRequest)
    - [GetVersionInfoResponse](#akuity-io-kargo-service-v1alpha1-GetVersionInfoResponse)
    - [GetWarehouseRequest](#akuity-io-kargo-service-v1alpha1-GetWarehouseRequest)
    - [GetWarehouseResponse](#akuity-io-kargo-service-v1alpha1-GetWarehouseResponse)
    - [GrantRequest](#akuity-io-kargo-service-v1alpha1-GrantRequest)
    - [GrantResponse](#akuity-io-kargo-service-v1alpha1-GrantResponse)
    - [ImageStageMap](#akuity-io-kargo-service-v1alpha1-ImageStageMap)
    - [ImageStageMap.StagesEntry](#akuity-io-kargo-service-v1alpha1-ImageStageMap-StagesEntry)
    - [ListAnalysisTemplatesRequest](#akuity-io-kargo-service-v1alpha1-ListAnalysisTemplatesRequest)
    - [ListAnalysisTemplatesResponse](#akuity-io-kargo-service-v1alpha1-ListAnalysisTemplatesResponse)
    - [ListClusterAnalysisTemplatesRequest](#akuity-io-kargo-service-v1alpha1-ListClusterAnalysisTemplatesRequest)
    - [ListClusterAnalysisTemplatesResponse](#akuity-io-kargo-service-v1alpha1-ListClusterAnalysisTemplatesResponse)
    - [ListClusterPromotionTasksRequest](#akuity-io-kargo-service-v1alpha1-ListClusterPromotionTasksRequest)
    - [ListClusterPromotionTasksResponse](#akuity-io-kargo-service-v1alpha1-ListClusterPromotionTasksResponse)
    - [ListClusterSecretsRequest](#akuity-io-kargo-service-v1alpha1-ListClusterSecretsRequest)
    - [ListClusterSecretsResponse](#akuity-io-kargo-service-v1alpha1-ListClusterSecretsResponse)
    - [ListConfigMapsRequest](#akuity-io-kargo-service-v1alpha1-ListConfigMapsRequest)
    - [ListConfigMapsResponse](#akuity-io-kargo-service-v1alpha1-ListConfigMapsResponse)
    - [ListCredentialsRequest](#akuity-io-kargo-service-v1alpha1-ListCredentialsRequest)
    - [ListCredentialsResponse](#akuity-io-kargo-service-v1alpha1-ListCredentialsResponse)
    - [ListImagesRequest](#akuity-io-kargo-service-v1alpha1-ListImagesRequest)
    - [ListImagesResponse](#akuity-io-kargo-service-v1alpha1-ListImagesResponse)
    - [ListImagesResponse.ImagesEntry](#akuity-io-kargo-service-v1alpha1-ListImagesResponse-ImagesEntry)
    - [ListProjectEventsRequest](#akuity-io-kargo-service-v1alpha1-ListProjectEventsRequest)
    - [ListProjectEventsResponse](#akuity-io-kargo-service-v1alpha1-ListProjectEventsResponse)
    - [ListProjectSecretsRequest](#akuity-io-kargo-service-v1alpha1-ListProjectSecretsRequest)
    - [ListProjectSecretsResponse](#akuity-io-kargo-service-v1alpha1-ListProjectSecretsResponse)
    - [ListProjectsRequest](#akuity-io-kargo-service-v1alpha1-ListProjectsRequest)
    - [ListProjectsResponse](#akuity-io-kargo-service-v1alpha1-ListProjectsResponse)
    - [ListPromotionTasksRequest](#akuity-io-kargo-service-v1alpha1-ListPromotionTasksRequest)
    - [ListPromotionTasksResponse](#akuity-io-kargo-service-v1alpha1-ListPromotionTasksResponse)
    - [ListPromotionsRequest](#akuity-io-kargo-service-v1alpha1-ListPromotionsRequest)
    - [ListPromotionsResponse](#akuity-io-kargo-service-v1alpha1-ListPromotionsResponse)
    - [ListRolesRequest](#akuity-io-kargo-service-v1alpha1-ListRolesRequest)
    - [ListRolesResponse](#akuity-io-kargo-service-v1alpha1-ListRolesResponse)
    - [ListStagesRequest](#akuity-io-kargo-service-v1alpha1-ListStagesRequest)
    - [ListStagesResponse](#akuity-io-kargo-service-v1alpha1-ListStagesResponse)
    - [ListWarehousesRequest](#akuity-io-kargo-service-v1alpha1-ListWarehousesRequest)
    - [ListWarehousesResponse](#akuity-io-kargo-service-v1alpha1-ListWarehousesResponse)
    - [OIDCConfig](#akuity-io-kargo-service-v1alpha1-OIDCConfig)
    - [PromoteDownstreamRequest](#akuity-io-kargo-service-v1alpha1-PromoteDownstreamRequest)
    - [PromoteDownstreamResponse](#akuity-io-kargo-service-v1alpha1-PromoteDownstreamResponse)
    - [PromoteToStageRequest](#akuity-io-kargo-service-v1alpha1-PromoteToStageRequest)
    - [PromoteToStageResponse](#akuity-io-kargo-service-v1alpha1-PromoteToStageResponse)
    - [QueryFreightRequest](#akuity-io-kargo-service-v1alpha1-QueryFreightRequest)
    - [QueryFreightResponse](#akuity-io-kargo-service-v1alpha1-QueryFreightResponse)
    - [QueryFreightResponse.GroupsEntry](#akuity-io-kargo-service-v1alpha1-QueryFreightResponse-GroupsEntry)
    - [RefreshClusterConfigRequest](#akuity-io-kargo-service-v1alpha1-RefreshClusterConfigRequest)
    - [RefreshClusterConfigResponse](#akuity-io-kargo-service-v1alpha1-RefreshClusterConfigResponse)
    - [RefreshProjectConfigRequest](#akuity-io-kargo-service-v1alpha1-RefreshProjectConfigRequest)
    - [RefreshProjectConfigResponse](#akuity-io-kargo-service-v1alpha1-RefreshProjectConfigResponse)
    - [RefreshStageRequest](#akuity-io-kargo-service-v1alpha1-RefreshStageRequest)
    - [RefreshStageResponse](#akuity-io-kargo-service-v1alpha1-RefreshStageResponse)
    - [RefreshWarehouseRequest](#akuity-io-kargo-service-v1alpha1-RefreshWarehouseRequest)
    - [RefreshWarehouseResponse](#akuity-io-kargo-service-v1alpha1-RefreshWarehouseResponse)
    - [ReverifyRequest](#akuity-io-kargo-service-v1alpha1-ReverifyRequest)
    - [ReverifyResponse](#akuity-io-kargo-service-v1alpha1-ReverifyResponse)
    - [RevokeRequest](#akuity-io-kargo-service-v1alpha1-RevokeRequest)
    - [RevokeResponse](#akuity-io-kargo-service-v1alpha1-RevokeResponse)
    - [TagMap](#akuity-io-kargo-service-v1alpha1-TagMap)
    - [TagMap.TagsEntry](#akuity-io-kargo-service-v1alpha1-TagMap-TagsEntry)
    - [UpdateClusterSecretRequest](#akuity-io-kargo-service-v1alpha1-UpdateClusterSecretRequest)
    - [UpdateClusterSecretRequest.DataEntry](#akuity-io-kargo-service-v1alpha1-UpdateClusterSecretRequest-DataEntry)
    - [UpdateClusterSecretResponse](#akuity-io-kargo-service-v1alpha1-UpdateClusterSecretResponse)
    - [UpdateCredentialsRequest](#akuity-io-kargo-service-v1alpha1-UpdateCredentialsRequest)
    - [UpdateCredentialsResponse](#akuity-io-kargo-service-v1alpha1-UpdateCredentialsResponse)
    - [UpdateFreightAliasRequest](#akuity-io-kargo-service-v1alpha1-UpdateFreightAliasRequest)
    - [UpdateFreightAliasResponse](#akuity-io-kargo-service-v1alpha1-UpdateFreightAliasResponse)
    - [UpdateProjectSecretRequest](#akuity-io-kargo-service-v1alpha1-UpdateProjectSecretRequest)
    - [UpdateProjectSecretRequest.DataEntry](#akuity-io-kargo-service-v1alpha1-UpdateProjectSecretRequest-DataEntry)
    - [UpdateProjectSecretResponse](#akuity-io-kargo-service-v1alpha1-UpdateProjectSecretResponse)
    - [UpdateResourceRequest](#akuity-io-kargo-service-v1alpha1-UpdateResourceRequest)
    - [UpdateResourceResponse](#akuity-io-kargo-service-v1alpha1-UpdateResourceResponse)
    - [UpdateResourceResult](#akuity-io-kargo-service-v1alpha1-UpdateResourceResult)
    - [UpdateRoleRequest](#akuity-io-kargo-service-v1alpha1-UpdateRoleRequest)
    - [UpdateRoleResponse](#akuity-io-kargo-service-v1alpha1-UpdateRoleResponse)
    - [VersionInfo](#akuity-io-kargo-service-v1alpha1-VersionInfo)
    - [WatchClusterConfigRequest](#akuity-io-kargo-service-v1alpha1-WatchClusterConfigRequest)
    - [WatchClusterConfigResponse](#akuity-io-kargo-service-v1alpha1-WatchClusterConfigResponse)
    - [WatchFreightRequest](#akuity-io-kargo-service-v1alpha1-WatchFreightRequest)
    - [WatchFreightResponse](#akuity-io-kargo-service-v1alpha1-WatchFreightResponse)
    - [WatchProjectConfigRequest](#akuity-io-kargo-service-v1alpha1-WatchProjectConfigRequest)
    - [WatchProjectConfigResponse](#akuity-io-kargo-service-v1alpha1-WatchProjectConfigResponse)
    - [WatchPromotionRequest](#akuity-io-kargo-service-v1alpha1-WatchPromotionRequest)
    - [WatchPromotionResponse](#akuity-io-kargo-service-v1alpha1-WatchPromotionResponse)
    - [WatchPromotionsRequest](#akuity-io-kargo-service-v1alpha1-WatchPromotionsRequest)
    - [WatchPromotionsResponse](#akuity-io-kargo-service-v1alpha1-WatchPromotionsResponse)
    - [WatchStagesRequest](#akuity-io-kargo-service-v1alpha1-WatchStagesRequest)
    - [WatchStagesResponse](#akuity-io-kargo-service-v1alpha1-WatchStagesResponse)
    - [WatchWarehousesRequest](#akuity-io-kargo-service-v1alpha1-WatchWarehousesRequest)
    - [WatchWarehousesResponse](#akuity-io-kargo-service-v1alpha1-WatchWarehousesResponse)
  
    - [RawFormat](#akuity-io-kargo-service-v1alpha1-RawFormat)
  
    - [KargoService](#akuity-io-kargo-service-v1alpha1-KargoService)
  
- [Scalar Value Types](#scalar-value-types)



<a name="api_service_v1alpha1_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/service/v1alpha1/service.proto



<a name="akuity-io-kargo-service-v1alpha1-AbortPromotionRequest"></a>

### AbortPromotionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| name | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-AbortPromotionResponse"></a>

### AbortPromotionResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-AbortVerificationRequest"></a>

### AbortVerificationRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| stage | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-AbortVerificationResponse"></a>

### AbortVerificationResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-AdminLoginRequest"></a>

### AdminLoginRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| password | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-AdminLoginResponse"></a>

### AdminLoginResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_token | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-ApproveFreightRequest"></a>

### ApproveFreightRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| name | [string](#string) |  |  |
| alias | [string](#string) |  |  |
| stage | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-ApproveFreightResponse"></a>

### ApproveFreightResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-ArgoCDShard"></a>

### ArgoCDShard



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| url | [string](#string) |  |  |
| namespace | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-Claims"></a>

### Claims



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| claims | [github.com.akuity.kargo.api.rbac.v1alpha1.Claim](#github-com-akuity-kargo-api-rbac-v1alpha1-Claim) | repeated | Note: oneof and repeated do not work together |






<a name="akuity-io-kargo-service-v1alpha1-ComponentVersions"></a>

### ComponentVersions



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| server | [VersionInfo](#akuity-io-kargo-service-v1alpha1-VersionInfo) | optional |  |
| cli | [VersionInfo](#akuity-io-kargo-service-v1alpha1-VersionInfo) | optional |  |






<a name="akuity-io-kargo-service-v1alpha1-CreateClusterSecretRequest"></a>

### CreateClusterSecretRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| data | [CreateClusterSecretRequest.DataEntry](#akuity-io-kargo-service-v1alpha1-CreateClusterSecretRequest-DataEntry) | repeated |  |






<a name="akuity-io-kargo-service-v1alpha1-CreateClusterSecretRequest-DataEntry"></a>

### CreateClusterSecretRequest.DataEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-CreateClusterSecretResponse"></a>

### CreateClusterSecretResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| secret | [k8s.io.api.core.v1.Secret](#k8s-io-api-core-v1-Secret) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-CreateCredentialsRequest"></a>

### CreateCredentialsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| name | [string](#string) |  |  |
| description | [string](#string) |  |  |
| type | [string](#string) |  | type is git, helm, image |
| repo_url | [string](#string) |  |  |
| repo_url_is_regex | [bool](#bool) |  |  |
| username | [string](#string) |  |  |
| password | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-CreateCredentialsResponse"></a>

### CreateCredentialsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| credentials | [k8s.io.api.core.v1.Secret](#k8s-io-api-core-v1-Secret) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-CreateOrUpdateResourceRequest"></a>

### CreateOrUpdateResourceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| manifest | [bytes](#bytes) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-CreateOrUpdateResourceResponse"></a>

### CreateOrUpdateResourceResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| results | [CreateOrUpdateResourceResult](#akuity-io-kargo-service-v1alpha1-CreateOrUpdateResourceResult) | repeated |  |






<a name="akuity-io-kargo-service-v1alpha1-CreateOrUpdateResourceResult"></a>

### CreateOrUpdateResourceResult



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| created_resource_manifest | [bytes](#bytes) |  |  |
| updated_resource_manifest | [bytes](#bytes) |  |  |
| error | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-CreateProjectSecretRequest"></a>

### CreateProjectSecretRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| name | [string](#string) |  |  |
| description | [string](#string) |  |  |
| data | [CreateProjectSecretRequest.DataEntry](#akuity-io-kargo-service-v1alpha1-CreateProjectSecretRequest-DataEntry) | repeated |  |






<a name="akuity-io-kargo-service-v1alpha1-CreateProjectSecretRequest-DataEntry"></a>

### CreateProjectSecretRequest.DataEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-CreateProjectSecretResponse"></a>

### CreateProjectSecretResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| secret | [k8s.io.api.core.v1.Secret](#k8s-io-api-core-v1-Secret) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-CreateResourceRequest"></a>

### CreateResourceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| manifest | [bytes](#bytes) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-CreateResourceResponse"></a>

### CreateResourceResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| results | [CreateResourceResult](#akuity-io-kargo-service-v1alpha1-CreateResourceResult) | repeated |  |






<a name="akuity-io-kargo-service-v1alpha1-CreateResourceResult"></a>

### CreateResourceResult



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| created_resource_manifest | [bytes](#bytes) |  |  |
| error | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-CreateRoleRequest"></a>

### CreateRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [github.com.akuity.kargo.api.rbac.v1alpha1.Role](#github-com-akuity-kargo-api-rbac-v1alpha1-Role) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-CreateRoleResponse"></a>

### CreateRoleResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [github.com.akuity.kargo.api.rbac.v1alpha1.Role](#github-com-akuity-kargo-api-rbac-v1alpha1-Role) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-DeleteAnalysisTemplateRequest"></a>

### DeleteAnalysisTemplateRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| name | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-DeleteAnalysisTemplateResponse"></a>

### DeleteAnalysisTemplateResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-DeleteClusterAnalysisTemplateRequest"></a>

### DeleteClusterAnalysisTemplateRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |






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



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-DeleteClusterSecretResponse"></a>

### DeleteClusterSecretResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-DeleteCredentialsRequest"></a>

### DeleteCredentialsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| name | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-DeleteCredentialsResponse"></a>

### DeleteCredentialsResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-DeleteFreightRequest"></a>

### DeleteFreightRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| name | [string](#string) |  |  |
| alias | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-DeleteFreightResponse"></a>

### DeleteFreightResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-DeleteProjectConfigRequest"></a>

### DeleteProjectConfigRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-DeleteProjectConfigResponse"></a>

### DeleteProjectConfigResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-DeleteProjectRequest"></a>

### DeleteProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-DeleteProjectResponse"></a>

### DeleteProjectResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-DeleteProjectSecretRequest"></a>

### DeleteProjectSecretRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| name | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-DeleteProjectSecretResponse"></a>

### DeleteProjectSecretResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-DeleteResourceRequest"></a>

### DeleteResourceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| manifest | [bytes](#bytes) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-DeleteResourceResponse"></a>

### DeleteResourceResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| results | [DeleteResourceResult](#akuity-io-kargo-service-v1alpha1-DeleteResourceResult) | repeated |  |






<a name="akuity-io-kargo-service-v1alpha1-DeleteResourceResult"></a>

### DeleteResourceResult



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| deleted_resource_manifest | [bytes](#bytes) |  |  |
| error | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-DeleteRoleRequest"></a>

### DeleteRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| name | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-DeleteRoleResponse"></a>

### DeleteRoleResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-DeleteStageRequest"></a>

### DeleteStageRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| name | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-DeleteStageResponse"></a>

### DeleteStageResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-DeleteWarehouseRequest"></a>

### DeleteWarehouseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| name | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-DeleteWarehouseResponse"></a>

### DeleteWarehouseResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-FreightList"></a>

### FreightList



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| freight | [github.com.akuity.kargo.api.v1alpha1.Freight](#github-com-akuity-kargo-api-v1alpha1-Freight) | repeated |  |






<a name="akuity-io-kargo-service-v1alpha1-GetAnalysisRunLogsRequest"></a>

### GetAnalysisRunLogsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| namespace | [string](#string) |  |  |
| name | [string](#string) |  |  |
| metric_name | [string](#string) |  |  |
| container_name | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetAnalysisRunLogsResponse"></a>

### GetAnalysisRunLogsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| chunk | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetAnalysisRunRequest"></a>

### GetAnalysisRunRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| namespace | [string](#string) |  |  |
| name | [string](#string) |  |  |
| format | [RawFormat](#akuity-io-kargo-service-v1alpha1-RawFormat) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetAnalysisRunResponse"></a>

### GetAnalysisRunResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| analysis_run | [github.com.akuity.kargo.api.stubs.rollouts.v1alpha1.AnalysisRun](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisRun) |  |  |
| raw | [bytes](#bytes) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetAnalysisTemplateRequest"></a>

### GetAnalysisTemplateRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| name | [string](#string) |  |  |
| format | [RawFormat](#akuity-io-kargo-service-v1alpha1-RawFormat) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetAnalysisTemplateResponse"></a>

### GetAnalysisTemplateResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| analysis_template | [github.com.akuity.kargo.api.stubs.rollouts.v1alpha1.AnalysisTemplate](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisTemplate) |  |  |
| raw | [bytes](#bytes) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetClusterAnalysisTemplateRequest"></a>

### GetClusterAnalysisTemplateRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| format | [RawFormat](#akuity-io-kargo-service-v1alpha1-RawFormat) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetClusterAnalysisTemplateResponse"></a>

### GetClusterAnalysisTemplateResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| cluster_analysis_template | [github.com.akuity.kargo.api.stubs.rollouts.v1alpha1.ClusterAnalysisTemplate](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-ClusterAnalysisTemplate) |  |  |
| raw | [bytes](#bytes) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetClusterConfigRequest"></a>

### GetClusterConfigRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| format | [RawFormat](#akuity-io-kargo-service-v1alpha1-RawFormat) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetClusterConfigResponse"></a>

### GetClusterConfigResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| cluster_config | [github.com.akuity.kargo.api.v1alpha1.ClusterConfig](#github-com-akuity-kargo-api-v1alpha1-ClusterConfig) |  |  |
| raw | [bytes](#bytes) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetClusterPromotionTaskRequest"></a>

### GetClusterPromotionTaskRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| format | [RawFormat](#akuity-io-kargo-service-v1alpha1-RawFormat) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetClusterPromotionTaskResponse"></a>

### GetClusterPromotionTaskResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| promotion_task | [github.com.akuity.kargo.api.v1alpha1.ClusterPromotionTask](#github-com-akuity-kargo-api-v1alpha1-ClusterPromotionTask) |  |  |
| raw | [bytes](#bytes) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetConfigMapRequest"></a>

### GetConfigMapRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| name | [string](#string) |  |  |
| format | [RawFormat](#akuity-io-kargo-service-v1alpha1-RawFormat) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetConfigMapResponse"></a>

### GetConfigMapResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| config_map | [k8s.io.api.core.v1.ConfigMap](#k8s-io-api-core-v1-ConfigMap) |  |  |
| raw | [bytes](#bytes) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetConfigRequest"></a>

### GetConfigRequest







<a name="akuity-io-kargo-service-v1alpha1-GetConfigResponse"></a>

### GetConfigResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| argocd_shards | [GetConfigResponse.ArgocdShardsEntry](#akuity-io-kargo-service-v1alpha1-GetConfigResponse-ArgocdShardsEntry) | repeated |  |
| secret_management_enabled | [bool](#bool) |  |  |
| cluster_secrets_namespace | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetConfigResponse-ArgocdShardsEntry"></a>

### GetConfigResponse.ArgocdShardsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [ArgoCDShard](#akuity-io-kargo-service-v1alpha1-ArgoCDShard) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetCredentialsRequest"></a>

### GetCredentialsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| name | [string](#string) |  |  |
| format | [RawFormat](#akuity-io-kargo-service-v1alpha1-RawFormat) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetCredentialsResponse"></a>

### GetCredentialsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| credentials | [k8s.io.api.core.v1.Secret](#k8s-io-api-core-v1-Secret) |  |  |
| raw | [bytes](#bytes) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetFreightRequest"></a>

### GetFreightRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| name | [string](#string) |  |  |
| alias | [string](#string) |  |  |
| format | [RawFormat](#akuity-io-kargo-service-v1alpha1-RawFormat) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetFreightResponse"></a>

### GetFreightResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| freight | [github.com.akuity.kargo.api.v1alpha1.Freight](#github-com-akuity-kargo-api-v1alpha1-Freight) |  |  |
| raw | [bytes](#bytes) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetProjectConfigRequest"></a>

### GetProjectConfigRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| format | [RawFormat](#akuity-io-kargo-service-v1alpha1-RawFormat) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetProjectConfigResponse"></a>

### GetProjectConfigResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_config | [github.com.akuity.kargo.api.v1alpha1.ProjectConfig](#github-com-akuity-kargo-api-v1alpha1-ProjectConfig) |  |  |
| raw | [bytes](#bytes) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetProjectRequest"></a>

### GetProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| format | [RawFormat](#akuity-io-kargo-service-v1alpha1-RawFormat) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetProjectResponse"></a>

### GetProjectResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [github.com.akuity.kargo.api.v1alpha1.Project](#github-com-akuity-kargo-api-v1alpha1-Project) |  |  |
| raw | [bytes](#bytes) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetPromotionRequest"></a>

### GetPromotionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| name | [string](#string) |  |  |
| format | [RawFormat](#akuity-io-kargo-service-v1alpha1-RawFormat) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetPromotionResponse"></a>

### GetPromotionResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| promotion | [github.com.akuity.kargo.api.v1alpha1.Promotion](#github-com-akuity-kargo-api-v1alpha1-Promotion) |  |  |
| raw | [bytes](#bytes) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetPromotionTaskRequest"></a>

### GetPromotionTaskRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| name | [string](#string) |  |  |
| format | [RawFormat](#akuity-io-kargo-service-v1alpha1-RawFormat) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetPromotionTaskResponse"></a>

### GetPromotionTaskResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| promotion_task | [github.com.akuity.kargo.api.v1alpha1.PromotionTask](#github-com-akuity-kargo-api-v1alpha1-PromotionTask) |  |  |
| raw | [bytes](#bytes) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetPublicConfigRequest"></a>

### GetPublicConfigRequest







<a name="akuity-io-kargo-service-v1alpha1-GetPublicConfigResponse"></a>

### GetPublicConfigResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| oidc_config | [OIDCConfig](#akuity-io-kargo-service-v1alpha1-OIDCConfig) |  |  |
| admin_account_enabled | [bool](#bool) |  |  |
| skip_auth | [bool](#bool) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetRoleRequest"></a>

### GetRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| name | [string](#string) |  |  |
| as_resources | [bool](#bool) |  |  |
| format | [RawFormat](#akuity-io-kargo-service-v1alpha1-RawFormat) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetRoleResponse"></a>

### GetRoleResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [github.com.akuity.kargo.api.rbac.v1alpha1.Role](#github-com-akuity-kargo-api-rbac-v1alpha1-Role) |  |  |
| resources | [github.com.akuity.kargo.api.rbac.v1alpha1.RoleResources](#github-com-akuity-kargo-api-rbac-v1alpha1-RoleResources) |  |  |
| raw | [bytes](#bytes) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetStageRequest"></a>

### GetStageRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| name | [string](#string) |  |  |
| format | [RawFormat](#akuity-io-kargo-service-v1alpha1-RawFormat) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetStageResponse"></a>

### GetStageResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| stage | [github.com.akuity.kargo.api.v1alpha1.Stage](#github-com-akuity-kargo-api-v1alpha1-Stage) |  |  |
| raw | [bytes](#bytes) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetVersionInfoRequest"></a>

### GetVersionInfoRequest







<a name="akuity-io-kargo-service-v1alpha1-GetVersionInfoResponse"></a>

### GetVersionInfoResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| version_info | [VersionInfo](#akuity-io-kargo-service-v1alpha1-VersionInfo) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetWarehouseRequest"></a>

### GetWarehouseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| name | [string](#string) |  |  |
| format | [RawFormat](#akuity-io-kargo-service-v1alpha1-RawFormat) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GetWarehouseResponse"></a>

### GetWarehouseResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| warehouse | [github.com.akuity.kargo.api.v1alpha1.Warehouse](#github-com-akuity-kargo-api-v1alpha1-Warehouse) |  |  |
| raw | [bytes](#bytes) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GrantRequest"></a>

### GrantRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| role | [string](#string) |  |  |
| user_claims | [Claims](#akuity-io-kargo-service-v1alpha1-Claims) |  |  |
| resource_details | [github.com.akuity.kargo.api.rbac.v1alpha1.ResourceDetails](#github-com-akuity-kargo-api-rbac-v1alpha1-ResourceDetails) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-GrantResponse"></a>

### GrantResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [github.com.akuity.kargo.api.rbac.v1alpha1.Role](#github-com-akuity-kargo-api-rbac-v1alpha1-Role) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-ImageStageMap"></a>

### ImageStageMap



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| stages | [ImageStageMap.StagesEntry](#akuity-io-kargo-service-v1alpha1-ImageStageMap-StagesEntry) | repeated | stages maps stage names to the order which an image was promoted to that stage |






<a name="akuity-io-kargo-service-v1alpha1-ImageStageMap-StagesEntry"></a>

### ImageStageMap.StagesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [int32](#int32) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-ListAnalysisTemplatesRequest"></a>

### ListAnalysisTemplatesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-ListAnalysisTemplatesResponse"></a>

### ListAnalysisTemplatesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| analysis_templates | [github.com.akuity.kargo.api.stubs.rollouts.v1alpha1.AnalysisTemplate](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisTemplate) | repeated |  |






<a name="akuity-io-kargo-service-v1alpha1-ListClusterAnalysisTemplatesRequest"></a>

### ListClusterAnalysisTemplatesRequest







<a name="akuity-io-kargo-service-v1alpha1-ListClusterAnalysisTemplatesResponse"></a>

### ListClusterAnalysisTemplatesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| cluster_analysis_templates | [github.com.akuity.kargo.api.stubs.rollouts.v1alpha1.ClusterAnalysisTemplate](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-ClusterAnalysisTemplate) | repeated |  |






<a name="akuity-io-kargo-service-v1alpha1-ListClusterPromotionTasksRequest"></a>

### ListClusterPromotionTasksRequest







<a name="akuity-io-kargo-service-v1alpha1-ListClusterPromotionTasksResponse"></a>

### ListClusterPromotionTasksResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| cluster_promotion_tasks | [github.com.akuity.kargo.api.v1alpha1.ClusterPromotionTask](#github-com-akuity-kargo-api-v1alpha1-ClusterPromotionTask) | repeated |  |






<a name="akuity-io-kargo-service-v1alpha1-ListClusterSecretsRequest"></a>

### ListClusterSecretsRequest
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-ListClusterSecretsResponse"></a>

### ListClusterSecretsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| secrets | [k8s.io.api.core.v1.Secret](#k8s-io-api-core-v1-Secret) | repeated |  |






<a name="akuity-io-kargo-service-v1alpha1-ListConfigMapsRequest"></a>

### ListConfigMapsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-ListConfigMapsResponse"></a>

### ListConfigMapsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| config_maps | [k8s.io.api.core.v1.ConfigMap](#k8s-io-api-core-v1-ConfigMap) | repeated |  |






<a name="akuity-io-kargo-service-v1alpha1-ListCredentialsRequest"></a>

### ListCredentialsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-ListCredentialsResponse"></a>

### ListCredentialsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| credentials | [k8s.io.api.core.v1.Secret](#k8s-io-api-core-v1-Secret) | repeated |  |






<a name="akuity-io-kargo-service-v1alpha1-ListImagesRequest"></a>

### ListImagesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-ListImagesResponse"></a>

### ListImagesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| images | [ListImagesResponse.ImagesEntry](#akuity-io-kargo-service-v1alpha1-ListImagesResponse-ImagesEntry) | repeated | images maps image repository names to their tags |






<a name="akuity-io-kargo-service-v1alpha1-ListImagesResponse-ImagesEntry"></a>

### ListImagesResponse.ImagesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [TagMap](#akuity-io-kargo-service-v1alpha1-TagMap) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-ListProjectEventsRequest"></a>

### ListProjectEventsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-ListProjectEventsResponse"></a>

### ListProjectEventsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| events | [k8s.io.api.core.v1.Event](#k8s-io-api-core-v1-Event) | repeated |  |






<a name="akuity-io-kargo-service-v1alpha1-ListProjectSecretsRequest"></a>

### ListProjectSecretsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-ListProjectSecretsResponse"></a>

### ListProjectSecretsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| secrets | [k8s.io.api.core.v1.Secret](#k8s-io-api-core-v1-Secret) | repeated |  |






<a name="akuity-io-kargo-service-v1alpha1-ListProjectsRequest"></a>

### ListProjectsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) | optional |  |
| page | [int32](#int32) | optional |  |
| filter | [string](#string) | optional |  |






<a name="akuity-io-kargo-service-v1alpha1-ListProjectsResponse"></a>

### ListProjectsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| projects | [github.com.akuity.kargo.api.v1alpha1.Project](#github-com-akuity-kargo-api-v1alpha1-Project) | repeated |  |
| total | [int32](#int32) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-ListPromotionTasksRequest"></a>

### ListPromotionTasksRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-ListPromotionTasksResponse"></a>

### ListPromotionTasksResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| promotion_tasks | [github.com.akuity.kargo.api.v1alpha1.PromotionTask](#github-com-akuity-kargo-api-v1alpha1-PromotionTask) | repeated |  |






<a name="akuity-io-kargo-service-v1alpha1-ListPromotionsRequest"></a>

### ListPromotionsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| stage | [string](#string) | optional |  |






<a name="akuity-io-kargo-service-v1alpha1-ListPromotionsResponse"></a>

### ListPromotionsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| promotions | [github.com.akuity.kargo.api.v1alpha1.Promotion](#github-com-akuity-kargo-api-v1alpha1-Promotion) | repeated |  |






<a name="akuity-io-kargo-service-v1alpha1-ListRolesRequest"></a>

### ListRolesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| as_resources | [bool](#bool) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-ListRolesResponse"></a>

### ListRolesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| roles | [github.com.akuity.kargo.api.rbac.v1alpha1.Role](#github-com-akuity-kargo-api-rbac-v1alpha1-Role) | repeated | Note: oneof and repeated do not work together |
| resources | [github.com.akuity.kargo.api.rbac.v1alpha1.RoleResources](#github-com-akuity-kargo-api-rbac-v1alpha1-RoleResources) | repeated |  |






<a name="akuity-io-kargo-service-v1alpha1-ListStagesRequest"></a>

### ListStagesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-ListStagesResponse"></a>

### ListStagesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| stages | [github.com.akuity.kargo.api.v1alpha1.Stage](#github-com-akuity-kargo-api-v1alpha1-Stage) | repeated |  |






<a name="akuity-io-kargo-service-v1alpha1-ListWarehousesRequest"></a>

### ListWarehousesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-ListWarehousesResponse"></a>

### ListWarehousesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| warehouses | [github.com.akuity.kargo.api.v1alpha1.Warehouse](#github-com-akuity-kargo-api-v1alpha1-Warehouse) | repeated |  |






<a name="akuity-io-kargo-service-v1alpha1-OIDCConfig"></a>

### OIDCConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| issuer_url | [string](#string) |  |  |
| client_id | [string](#string) |  |  |
| scopes | [string](#string) | repeated |  |
| cli_client_id | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-PromoteDownstreamRequest"></a>

### PromoteDownstreamRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| stage | [string](#string) |  |  |
| freight | [string](#string) |  |  |
| freight_alias | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-PromoteDownstreamResponse"></a>

### PromoteDownstreamResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| promotions | [github.com.akuity.kargo.api.v1alpha1.Promotion](#github-com-akuity-kargo-api-v1alpha1-Promotion) | repeated |  |






<a name="akuity-io-kargo-service-v1alpha1-PromoteToStageRequest"></a>

### PromoteToStageRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| stage | [string](#string) |  |  |
| freight | [string](#string) |  |  |
| freight_alias | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-PromoteToStageResponse"></a>

### PromoteToStageResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| promotion | [github.com.akuity.kargo.api.v1alpha1.Promotion](#github-com-akuity-kargo-api-v1alpha1-Promotion) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-QueryFreightRequest"></a>

### QueryFreightRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| stage | [string](#string) |  |  |
| group_by | [string](#string) |  |  |
| group | [string](#string) |  |  |
| order_by | [string](#string) |  |  |
| reverse | [bool](#bool) |  |  |
| origins | [string](#string) | repeated |  |






<a name="akuity-io-kargo-service-v1alpha1-QueryFreightResponse"></a>

### QueryFreightResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| groups | [QueryFreightResponse.GroupsEntry](#akuity-io-kargo-service-v1alpha1-QueryFreightResponse-GroupsEntry) | repeated |  |






<a name="akuity-io-kargo-service-v1alpha1-QueryFreightResponse-GroupsEntry"></a>

### QueryFreightResponse.GroupsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [FreightList](#akuity-io-kargo-service-v1alpha1-FreightList) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-RefreshClusterConfigRequest"></a>

### RefreshClusterConfigRequest
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-RefreshClusterConfigResponse"></a>

### RefreshClusterConfigResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| cluster_config | [github.com.akuity.kargo.api.v1alpha1.ClusterConfig](#github-com-akuity-kargo-api-v1alpha1-ClusterConfig) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-RefreshProjectConfigRequest"></a>

### RefreshProjectConfigRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-RefreshProjectConfigResponse"></a>

### RefreshProjectConfigResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_config | [github.com.akuity.kargo.api.v1alpha1.ProjectConfig](#github-com-akuity-kargo-api-v1alpha1-ProjectConfig) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-RefreshStageRequest"></a>

### RefreshStageRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| name | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-RefreshStageResponse"></a>

### RefreshStageResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| stage | [github.com.akuity.kargo.api.v1alpha1.Stage](#github-com-akuity-kargo-api-v1alpha1-Stage) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-RefreshWarehouseRequest"></a>

### RefreshWarehouseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| name | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-RefreshWarehouseResponse"></a>

### RefreshWarehouseResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| warehouse | [github.com.akuity.kargo.api.v1alpha1.Warehouse](#github-com-akuity-kargo-api-v1alpha1-Warehouse) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-ReverifyRequest"></a>

### ReverifyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| stage | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-ReverifyResponse"></a>

### ReverifyResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-RevokeRequest"></a>

### RevokeRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| role | [string](#string) |  |  |
| user_claims | [Claims](#akuity-io-kargo-service-v1alpha1-Claims) |  |  |
| resource_details | [github.com.akuity.kargo.api.rbac.v1alpha1.ResourceDetails](#github-com-akuity-kargo-api-rbac-v1alpha1-ResourceDetails) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-RevokeResponse"></a>

### RevokeResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [github.com.akuity.kargo.api.rbac.v1alpha1.Role](#github-com-akuity-kargo-api-rbac-v1alpha1-Role) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-TagMap"></a>

### TagMap



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tags | [TagMap.TagsEntry](#akuity-io-kargo-service-v1alpha1-TagMap-TagsEntry) | repeated | tags maps image tag names to stages which have previously used that tag |






<a name="akuity-io-kargo-service-v1alpha1-TagMap-TagsEntry"></a>

### TagMap.TagsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [ImageStageMap](#akuity-io-kargo-service-v1alpha1-ImageStageMap) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-UpdateClusterSecretRequest"></a>

### UpdateClusterSecretRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| data | [UpdateClusterSecretRequest.DataEntry](#akuity-io-kargo-service-v1alpha1-UpdateClusterSecretRequest-DataEntry) | repeated |  |






<a name="akuity-io-kargo-service-v1alpha1-UpdateClusterSecretRequest-DataEntry"></a>

### UpdateClusterSecretRequest.DataEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-UpdateClusterSecretResponse"></a>

### UpdateClusterSecretResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| secret | [k8s.io.api.core.v1.Secret](#k8s-io-api-core-v1-Secret) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-UpdateCredentialsRequest"></a>

### UpdateCredentialsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| name | [string](#string) |  |  |
| description | [string](#string) |  |  |
| type | [string](#string) |  |  |
| repo_url | [string](#string) |  |  |
| repo_url_is_regex | [bool](#bool) |  |  |
| username | [string](#string) |  |  |
| password | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-UpdateCredentialsResponse"></a>

### UpdateCredentialsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| credentials | [k8s.io.api.core.v1.Secret](#k8s-io-api-core-v1-Secret) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-UpdateFreightAliasRequest"></a>

### UpdateFreightAliasRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| name | [string](#string) |  |  |
| old_alias | [string](#string) |  |  |
| new_alias | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-UpdateFreightAliasResponse"></a>

### UpdateFreightAliasResponse
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-UpdateProjectSecretRequest"></a>

### UpdateProjectSecretRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| name | [string](#string) |  |  |
| description | [string](#string) |  |  |
| data | [UpdateProjectSecretRequest.DataEntry](#akuity-io-kargo-service-v1alpha1-UpdateProjectSecretRequest-DataEntry) | repeated |  |






<a name="akuity-io-kargo-service-v1alpha1-UpdateProjectSecretRequest-DataEntry"></a>

### UpdateProjectSecretRequest.DataEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-UpdateProjectSecretResponse"></a>

### UpdateProjectSecretResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| secret | [k8s.io.api.core.v1.Secret](#k8s-io-api-core-v1-Secret) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-UpdateResourceRequest"></a>

### UpdateResourceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| manifest | [bytes](#bytes) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-UpdateResourceResponse"></a>

### UpdateResourceResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| results | [UpdateResourceResult](#akuity-io-kargo-service-v1alpha1-UpdateResourceResult) | repeated |  |






<a name="akuity-io-kargo-service-v1alpha1-UpdateResourceResult"></a>

### UpdateResourceResult



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| updated_resource_manifest | [bytes](#bytes) |  |  |
| error | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-UpdateRoleRequest"></a>

### UpdateRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [github.com.akuity.kargo.api.rbac.v1alpha1.Role](#github-com-akuity-kargo-api-rbac-v1alpha1-Role) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-UpdateRoleResponse"></a>

### UpdateRoleResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [github.com.akuity.kargo.api.rbac.v1alpha1.Role](#github-com-akuity-kargo-api-rbac-v1alpha1-Role) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-VersionInfo"></a>

### VersionInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| version | [string](#string) |  |  |
| git_commit | [string](#string) |  |  |
| git_tree_dirty | [bool](#bool) |  |  |
| build_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| go_version | [string](#string) |  |  |
| compiler | [string](#string) |  |  |
| platform | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-WatchClusterConfigRequest"></a>

### WatchClusterConfigRequest
explicitly empty






<a name="akuity-io-kargo-service-v1alpha1-WatchClusterConfigResponse"></a>

### WatchClusterConfigResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| cluster_config | [github.com.akuity.kargo.api.v1alpha1.ClusterConfig](#github-com-akuity-kargo-api-v1alpha1-ClusterConfig) |  |  |
| type | [string](#string) |  | ADDED / MODIFIED / DELETED |






<a name="akuity-io-kargo-service-v1alpha1-WatchFreightRequest"></a>

### WatchFreightRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-WatchFreightResponse"></a>

### WatchFreightResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| freight | [github.com.akuity.kargo.api.v1alpha1.Freight](#github-com-akuity-kargo-api-v1alpha1-Freight) |  |  |
| type | [string](#string) |  | ADDED / MODIFIED / DELETED |






<a name="akuity-io-kargo-service-v1alpha1-WatchProjectConfigRequest"></a>

### WatchProjectConfigRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-WatchProjectConfigResponse"></a>

### WatchProjectConfigResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_config | [github.com.akuity.kargo.api.v1alpha1.ProjectConfig](#github-com-akuity-kargo-api-v1alpha1-ProjectConfig) |  |  |
| type | [string](#string) |  | ADDED / MODIFIED / DELETED |






<a name="akuity-io-kargo-service-v1alpha1-WatchPromotionRequest"></a>

### WatchPromotionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| name | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-WatchPromotionResponse"></a>

### WatchPromotionResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| promotion | [github.com.akuity.kargo.api.v1alpha1.Promotion](#github-com-akuity-kargo-api-v1alpha1-Promotion) |  |  |
| type | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-WatchPromotionsRequest"></a>

### WatchPromotionsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| stage | [string](#string) | optional |  |






<a name="akuity-io-kargo-service-v1alpha1-WatchPromotionsResponse"></a>

### WatchPromotionsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| promotion | [github.com.akuity.kargo.api.v1alpha1.Promotion](#github-com-akuity-kargo-api-v1alpha1-Promotion) |  |  |
| type | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-WatchStagesRequest"></a>

### WatchStagesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| name | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-WatchStagesResponse"></a>

### WatchStagesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| stage | [github.com.akuity.kargo.api.v1alpha1.Stage](#github-com-akuity-kargo-api-v1alpha1-Stage) |  |  |
| type | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-WatchWarehousesRequest"></a>

### WatchWarehousesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| name | [string](#string) |  |  |






<a name="akuity-io-kargo-service-v1alpha1-WatchWarehousesResponse"></a>

### WatchWarehousesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| warehouse | [github.com.akuity.kargo.api.v1alpha1.Warehouse](#github-com-akuity-kargo-api-v1alpha1-Warehouse) |  |  |
| type | [string](#string) |  |  |





 


<a name="akuity-io-kargo-service-v1alpha1-RawFormat"></a>

### RawFormat


| Name | Number | Description |
| ---- | ------ | ----------- |
| RAW_FORMAT_UNSPECIFIED | 0 |  |
| RAW_FORMAT_JSON | 1 |  |
| RAW_FORMAT_YAML | 2 |  |


 

 


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

