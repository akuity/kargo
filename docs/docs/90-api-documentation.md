# API Documentation
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
  
- [api/rbac/v1alpha1/generated.proto](#api_rbac_v1alpha1_generated-proto)
    - [Claim](#github-com-akuity-kargo-api-rbac-v1alpha1-Claim)
    - [ResourceDetails](#github-com-akuity-kargo-api-rbac-v1alpha1-ResourceDetails)
    - [Role](#github-com-akuity-kargo-api-rbac-v1alpha1-Role)
    - [RoleResources](#github-com-akuity-kargo-api-rbac-v1alpha1-RoleResources)
  
- [api/v1alpha1/generated.proto](#api_v1alpha1_generated-proto)
    - [AnalysisRunArgument](#github-com-akuity-kargo-api-v1alpha1-AnalysisRunArgument)
    - [AnalysisRunMetadata](#github-com-akuity-kargo-api-v1alpha1-AnalysisRunMetadata)
    - [AnalysisRunMetadata.AnnotationsEntry](#github-com-akuity-kargo-api-v1alpha1-AnalysisRunMetadata-AnnotationsEntry)
    - [AnalysisRunMetadata.LabelsEntry](#github-com-akuity-kargo-api-v1alpha1-AnalysisRunMetadata-LabelsEntry)
    - [AnalysisRunReference](#github-com-akuity-kargo-api-v1alpha1-AnalysisRunReference)
    - [AnalysisTemplateReference](#github-com-akuity-kargo-api-v1alpha1-AnalysisTemplateReference)
    - [ApprovedStage](#github-com-akuity-kargo-api-v1alpha1-ApprovedStage)
    - [ArgoCDAppHealthStatus](#github-com-akuity-kargo-api-v1alpha1-ArgoCDAppHealthStatus)
    - [ArgoCDAppStatus](#github-com-akuity-kargo-api-v1alpha1-ArgoCDAppStatus)
    - [ArgoCDAppSyncStatus](#github-com-akuity-kargo-api-v1alpha1-ArgoCDAppSyncStatus)
    - [ArtifactoryWebhookReceiverConfig](#github-com-akuity-kargo-api-v1alpha1-ArtifactoryWebhookReceiverConfig)
    - [AzureWebhookReceiverConfig](#github-com-akuity-kargo-api-v1alpha1-AzureWebhookReceiverConfig)
    - [BitbucketWebhookReceiverConfig](#github-com-akuity-kargo-api-v1alpha1-BitbucketWebhookReceiverConfig)
    - [Chart](#github-com-akuity-kargo-api-v1alpha1-Chart)
    - [ChartDiscoveryResult](#github-com-akuity-kargo-api-v1alpha1-ChartDiscoveryResult)
    - [ChartSubscription](#github-com-akuity-kargo-api-v1alpha1-ChartSubscription)
    - [ClusterConfig](#github-com-akuity-kargo-api-v1alpha1-ClusterConfig)
    - [ClusterConfigList](#github-com-akuity-kargo-api-v1alpha1-ClusterConfigList)
    - [ClusterConfigSpec](#github-com-akuity-kargo-api-v1alpha1-ClusterConfigSpec)
    - [ClusterConfigStatus](#github-com-akuity-kargo-api-v1alpha1-ClusterConfigStatus)
    - [ClusterPromotionTask](#github-com-akuity-kargo-api-v1alpha1-ClusterPromotionTask)
    - [ClusterPromotionTaskList](#github-com-akuity-kargo-api-v1alpha1-ClusterPromotionTaskList)
    - [CurrentStage](#github-com-akuity-kargo-api-v1alpha1-CurrentStage)
    - [DiscoveredArtifacts](#github-com-akuity-kargo-api-v1alpha1-DiscoveredArtifacts)
    - [DiscoveredCommit](#github-com-akuity-kargo-api-v1alpha1-DiscoveredCommit)
    - [DiscoveredImageReference](#github-com-akuity-kargo-api-v1alpha1-DiscoveredImageReference)
    - [DiscoveredImageReference.AnnotationsEntry](#github-com-akuity-kargo-api-v1alpha1-DiscoveredImageReference-AnnotationsEntry)
    - [DockerHubWebhookReceiverConfig](#github-com-akuity-kargo-api-v1alpha1-DockerHubWebhookReceiverConfig)
    - [ExpressionVariable](#github-com-akuity-kargo-api-v1alpha1-ExpressionVariable)
    - [Freight](#github-com-akuity-kargo-api-v1alpha1-Freight)
    - [FreightCollection](#github-com-akuity-kargo-api-v1alpha1-FreightCollection)
    - [FreightCollection.ItemsEntry](#github-com-akuity-kargo-api-v1alpha1-FreightCollection-ItemsEntry)
    - [FreightList](#github-com-akuity-kargo-api-v1alpha1-FreightList)
    - [FreightOrigin](#github-com-akuity-kargo-api-v1alpha1-FreightOrigin)
    - [FreightReference](#github-com-akuity-kargo-api-v1alpha1-FreightReference)
    - [FreightRequest](#github-com-akuity-kargo-api-v1alpha1-FreightRequest)
    - [FreightSources](#github-com-akuity-kargo-api-v1alpha1-FreightSources)
    - [FreightStatus](#github-com-akuity-kargo-api-v1alpha1-FreightStatus)
    - [FreightStatus.ApprovedForEntry](#github-com-akuity-kargo-api-v1alpha1-FreightStatus-ApprovedForEntry)
    - [FreightStatus.CurrentlyInEntry](#github-com-akuity-kargo-api-v1alpha1-FreightStatus-CurrentlyInEntry)
    - [FreightStatus.MetadataEntry](#github-com-akuity-kargo-api-v1alpha1-FreightStatus-MetadataEntry)
    - [FreightStatus.VerifiedInEntry](#github-com-akuity-kargo-api-v1alpha1-FreightStatus-VerifiedInEntry)
    - [GitCommit](#github-com-akuity-kargo-api-v1alpha1-GitCommit)
    - [GitDiscoveryResult](#github-com-akuity-kargo-api-v1alpha1-GitDiscoveryResult)
    - [GitHubWebhookReceiverConfig](#github-com-akuity-kargo-api-v1alpha1-GitHubWebhookReceiverConfig)
    - [GitLabWebhookReceiverConfig](#github-com-akuity-kargo-api-v1alpha1-GitLabWebhookReceiverConfig)
    - [GitSubscription](#github-com-akuity-kargo-api-v1alpha1-GitSubscription)
    - [GiteaWebhookReceiverConfig](#github-com-akuity-kargo-api-v1alpha1-GiteaWebhookReceiverConfig)
    - [Health](#github-com-akuity-kargo-api-v1alpha1-Health)
    - [HealthCheckStep](#github-com-akuity-kargo-api-v1alpha1-HealthCheckStep)
    - [HealthStats](#github-com-akuity-kargo-api-v1alpha1-HealthStats)
    - [Image](#github-com-akuity-kargo-api-v1alpha1-Image)
    - [Image.AnnotationsEntry](#github-com-akuity-kargo-api-v1alpha1-Image-AnnotationsEntry)
    - [ImageDiscoveryResult](#github-com-akuity-kargo-api-v1alpha1-ImageDiscoveryResult)
    - [ImageSubscription](#github-com-akuity-kargo-api-v1alpha1-ImageSubscription)
    - [Project](#github-com-akuity-kargo-api-v1alpha1-Project)
    - [ProjectConfig](#github-com-akuity-kargo-api-v1alpha1-ProjectConfig)
    - [ProjectConfigList](#github-com-akuity-kargo-api-v1alpha1-ProjectConfigList)
    - [ProjectConfigSpec](#github-com-akuity-kargo-api-v1alpha1-ProjectConfigSpec)
    - [ProjectConfigStatus](#github-com-akuity-kargo-api-v1alpha1-ProjectConfigStatus)
    - [ProjectList](#github-com-akuity-kargo-api-v1alpha1-ProjectList)
    - [ProjectStats](#github-com-akuity-kargo-api-v1alpha1-ProjectStats)
    - [ProjectStatus](#github-com-akuity-kargo-api-v1alpha1-ProjectStatus)
    - [Promotion](#github-com-akuity-kargo-api-v1alpha1-Promotion)
    - [PromotionList](#github-com-akuity-kargo-api-v1alpha1-PromotionList)
    - [PromotionPolicy](#github-com-akuity-kargo-api-v1alpha1-PromotionPolicy)
    - [PromotionPolicySelector](#github-com-akuity-kargo-api-v1alpha1-PromotionPolicySelector)
    - [PromotionReference](#github-com-akuity-kargo-api-v1alpha1-PromotionReference)
    - [PromotionSpec](#github-com-akuity-kargo-api-v1alpha1-PromotionSpec)
    - [PromotionStatus](#github-com-akuity-kargo-api-v1alpha1-PromotionStatus)
    - [PromotionStep](#github-com-akuity-kargo-api-v1alpha1-PromotionStep)
    - [PromotionStepRetry](#github-com-akuity-kargo-api-v1alpha1-PromotionStepRetry)
    - [PromotionTask](#github-com-akuity-kargo-api-v1alpha1-PromotionTask)
    - [PromotionTaskList](#github-com-akuity-kargo-api-v1alpha1-PromotionTaskList)
    - [PromotionTaskReference](#github-com-akuity-kargo-api-v1alpha1-PromotionTaskReference)
    - [PromotionTaskSpec](#github-com-akuity-kargo-api-v1alpha1-PromotionTaskSpec)
    - [PromotionTemplate](#github-com-akuity-kargo-api-v1alpha1-PromotionTemplate)
    - [PromotionTemplateSpec](#github-com-akuity-kargo-api-v1alpha1-PromotionTemplateSpec)
    - [QuayWebhookReceiverConfig](#github-com-akuity-kargo-api-v1alpha1-QuayWebhookReceiverConfig)
    - [RepoSubscription](#github-com-akuity-kargo-api-v1alpha1-RepoSubscription)
    - [Stage](#github-com-akuity-kargo-api-v1alpha1-Stage)
    - [StageList](#github-com-akuity-kargo-api-v1alpha1-StageList)
    - [StageSpec](#github-com-akuity-kargo-api-v1alpha1-StageSpec)
    - [StageStats](#github-com-akuity-kargo-api-v1alpha1-StageStats)
    - [StageStatus](#github-com-akuity-kargo-api-v1alpha1-StageStatus)
    - [StepExecutionMetadata](#github-com-akuity-kargo-api-v1alpha1-StepExecutionMetadata)
    - [Verification](#github-com-akuity-kargo-api-v1alpha1-Verification)
    - [VerificationInfo](#github-com-akuity-kargo-api-v1alpha1-VerificationInfo)
    - [VerifiedStage](#github-com-akuity-kargo-api-v1alpha1-VerifiedStage)
    - [Warehouse](#github-com-akuity-kargo-api-v1alpha1-Warehouse)
    - [WarehouseList](#github-com-akuity-kargo-api-v1alpha1-WarehouseList)
    - [WarehouseSpec](#github-com-akuity-kargo-api-v1alpha1-WarehouseSpec)
    - [WarehouseStats](#github-com-akuity-kargo-api-v1alpha1-WarehouseStats)
    - [WarehouseStatus](#github-com-akuity-kargo-api-v1alpha1-WarehouseStatus)
    - [WebhookReceiverConfig](#github-com-akuity-kargo-api-v1alpha1-WebhookReceiverConfig)
    - [WebhookReceiverDetails](#github-com-akuity-kargo-api-v1alpha1-WebhookReceiverDetails)
  
- [api/stubs/rollouts/v1alpha1/generated.proto](#api_stubs_rollouts_v1alpha1_generated-proto)
    - [AnalysisRun](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisRun)
    - [AnalysisRunList](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisRunList)
    - [AnalysisRunSpec](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisRunSpec)
    - [AnalysisRunStatus](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisRunStatus)
    - [AnalysisTemplate](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisTemplate)
    - [AnalysisTemplateList](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisTemplateList)
    - [AnalysisTemplateSpec](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisTemplateSpec)
    - [Argument](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Argument)
    - [Authentication](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Authentication)
    - [CloudWatchMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-CloudWatchMetric)
    - [CloudWatchMetricDataQuery](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-CloudWatchMetricDataQuery)
    - [CloudWatchMetricStat](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-CloudWatchMetricStat)
    - [CloudWatchMetricStatMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-CloudWatchMetricStatMetric)
    - [CloudWatchMetricStatMetricDimension](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-CloudWatchMetricStatMetricDimension)
    - [ClusterAnalysisTemplate](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-ClusterAnalysisTemplate)
    - [ClusterAnalysisTemplateList](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-ClusterAnalysisTemplateList)
    - [DatadogMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-DatadogMetric)
    - [DatadogMetric.QueriesEntry](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-DatadogMetric-QueriesEntry)
    - [DryRun](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-DryRun)
    - [FieldRef](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-FieldRef)
    - [GraphiteMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-GraphiteMetric)
    - [InfluxdbMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-InfluxdbMetric)
    - [JobMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-JobMetric)
    - [KayentaMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-KayentaMetric)
    - [KayentaScope](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-KayentaScope)
    - [KayentaThreshold](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-KayentaThreshold)
    - [Measurement](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Measurement)
    - [Measurement.MetadataEntry](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Measurement-MetadataEntry)
    - [MeasurementRetention](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-MeasurementRetention)
    - [Metric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Metric)
    - [MetricProvider](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-MetricProvider)
    - [MetricProvider.PluginEntry](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-MetricProvider-PluginEntry)
    - [MetricResult](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-MetricResult)
    - [MetricResult.MetadataEntry](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-MetricResult-MetadataEntry)
    - [NewRelicMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-NewRelicMetric)
    - [OAuth2Config](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-OAuth2Config)
    - [PrometheusMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-PrometheusMetric)
    - [RunSummary](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-RunSummary)
    - [ScopeDetail](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-ScopeDetail)
    - [SecretKeyRef](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-SecretKeyRef)
    - [Sigv4Config](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Sigv4Config)
    - [SkyWalkingMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-SkyWalkingMetric)
    - [ValueFrom](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-ValueFrom)
    - [WavefrontMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-WavefrontMetric)
    - [WebMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-WebMetric)
    - [WebMetricHeader](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-WebMetricHeader)
  
- [k8s.io/api/core/v1/generated.proto](#k8s-io_api_core_v1_generated-proto)
    - [AWSElasticBlockStoreVolumeSource](#k8s-io-api-core-v1-AWSElasticBlockStoreVolumeSource)
    - [Affinity](#k8s-io-api-core-v1-Affinity)
    - [AppArmorProfile](#k8s-io-api-core-v1-AppArmorProfile)
    - [AttachedVolume](#k8s-io-api-core-v1-AttachedVolume)
    - [AvoidPods](#k8s-io-api-core-v1-AvoidPods)
    - [AzureDiskVolumeSource](#k8s-io-api-core-v1-AzureDiskVolumeSource)
    - [AzureFilePersistentVolumeSource](#k8s-io-api-core-v1-AzureFilePersistentVolumeSource)
    - [AzureFileVolumeSource](#k8s-io-api-core-v1-AzureFileVolumeSource)
    - [Binding](#k8s-io-api-core-v1-Binding)
    - [CSIPersistentVolumeSource](#k8s-io-api-core-v1-CSIPersistentVolumeSource)
    - [CSIPersistentVolumeSource.VolumeAttributesEntry](#k8s-io-api-core-v1-CSIPersistentVolumeSource-VolumeAttributesEntry)
    - [CSIVolumeSource](#k8s-io-api-core-v1-CSIVolumeSource)
    - [CSIVolumeSource.VolumeAttributesEntry](#k8s-io-api-core-v1-CSIVolumeSource-VolumeAttributesEntry)
    - [Capabilities](#k8s-io-api-core-v1-Capabilities)
    - [CephFSPersistentVolumeSource](#k8s-io-api-core-v1-CephFSPersistentVolumeSource)
    - [CephFSVolumeSource](#k8s-io-api-core-v1-CephFSVolumeSource)
    - [CinderPersistentVolumeSource](#k8s-io-api-core-v1-CinderPersistentVolumeSource)
    - [CinderVolumeSource](#k8s-io-api-core-v1-CinderVolumeSource)
    - [ClientIPConfig](#k8s-io-api-core-v1-ClientIPConfig)
    - [ClusterTrustBundleProjection](#k8s-io-api-core-v1-ClusterTrustBundleProjection)
    - [ComponentCondition](#k8s-io-api-core-v1-ComponentCondition)
    - [ComponentStatus](#k8s-io-api-core-v1-ComponentStatus)
    - [ComponentStatusList](#k8s-io-api-core-v1-ComponentStatusList)
    - [ConfigMap](#k8s-io-api-core-v1-ConfigMap)
    - [ConfigMap.BinaryDataEntry](#k8s-io-api-core-v1-ConfigMap-BinaryDataEntry)
    - [ConfigMap.DataEntry](#k8s-io-api-core-v1-ConfigMap-DataEntry)
    - [ConfigMapEnvSource](#k8s-io-api-core-v1-ConfigMapEnvSource)
    - [ConfigMapKeySelector](#k8s-io-api-core-v1-ConfigMapKeySelector)
    - [ConfigMapList](#k8s-io-api-core-v1-ConfigMapList)
    - [ConfigMapNodeConfigSource](#k8s-io-api-core-v1-ConfigMapNodeConfigSource)
    - [ConfigMapProjection](#k8s-io-api-core-v1-ConfigMapProjection)
    - [ConfigMapVolumeSource](#k8s-io-api-core-v1-ConfigMapVolumeSource)
    - [Container](#k8s-io-api-core-v1-Container)
    - [ContainerImage](#k8s-io-api-core-v1-ContainerImage)
    - [ContainerPort](#k8s-io-api-core-v1-ContainerPort)
    - [ContainerResizePolicy](#k8s-io-api-core-v1-ContainerResizePolicy)
    - [ContainerState](#k8s-io-api-core-v1-ContainerState)
    - [ContainerStateRunning](#k8s-io-api-core-v1-ContainerStateRunning)
    - [ContainerStateTerminated](#k8s-io-api-core-v1-ContainerStateTerminated)
    - [ContainerStateWaiting](#k8s-io-api-core-v1-ContainerStateWaiting)
    - [ContainerStatus](#k8s-io-api-core-v1-ContainerStatus)
    - [ContainerStatus.AllocatedResourcesEntry](#k8s-io-api-core-v1-ContainerStatus-AllocatedResourcesEntry)
    - [ContainerUser](#k8s-io-api-core-v1-ContainerUser)
    - [DaemonEndpoint](#k8s-io-api-core-v1-DaemonEndpoint)
    - [DownwardAPIProjection](#k8s-io-api-core-v1-DownwardAPIProjection)
    - [DownwardAPIVolumeFile](#k8s-io-api-core-v1-DownwardAPIVolumeFile)
    - [DownwardAPIVolumeSource](#k8s-io-api-core-v1-DownwardAPIVolumeSource)
    - [EmptyDirVolumeSource](#k8s-io-api-core-v1-EmptyDirVolumeSource)
    - [EndpointAddress](#k8s-io-api-core-v1-EndpointAddress)
    - [EndpointPort](#k8s-io-api-core-v1-EndpointPort)
    - [EndpointSubset](#k8s-io-api-core-v1-EndpointSubset)
    - [Endpoints](#k8s-io-api-core-v1-Endpoints)
    - [EndpointsList](#k8s-io-api-core-v1-EndpointsList)
    - [EnvFromSource](#k8s-io-api-core-v1-EnvFromSource)
    - [EnvVar](#k8s-io-api-core-v1-EnvVar)
    - [EnvVarSource](#k8s-io-api-core-v1-EnvVarSource)
    - [EphemeralContainer](#k8s-io-api-core-v1-EphemeralContainer)
    - [EphemeralContainerCommon](#k8s-io-api-core-v1-EphemeralContainerCommon)
    - [EphemeralVolumeSource](#k8s-io-api-core-v1-EphemeralVolumeSource)
    - [Event](#k8s-io-api-core-v1-Event)
    - [EventList](#k8s-io-api-core-v1-EventList)
    - [EventSeries](#k8s-io-api-core-v1-EventSeries)
    - [EventSource](#k8s-io-api-core-v1-EventSource)
    - [ExecAction](#k8s-io-api-core-v1-ExecAction)
    - [FCVolumeSource](#k8s-io-api-core-v1-FCVolumeSource)
    - [FlexPersistentVolumeSource](#k8s-io-api-core-v1-FlexPersistentVolumeSource)
    - [FlexPersistentVolumeSource.OptionsEntry](#k8s-io-api-core-v1-FlexPersistentVolumeSource-OptionsEntry)
    - [FlexVolumeSource](#k8s-io-api-core-v1-FlexVolumeSource)
    - [FlexVolumeSource.OptionsEntry](#k8s-io-api-core-v1-FlexVolumeSource-OptionsEntry)
    - [FlockerVolumeSource](#k8s-io-api-core-v1-FlockerVolumeSource)
    - [GCEPersistentDiskVolumeSource](#k8s-io-api-core-v1-GCEPersistentDiskVolumeSource)
    - [GRPCAction](#k8s-io-api-core-v1-GRPCAction)
    - [GitRepoVolumeSource](#k8s-io-api-core-v1-GitRepoVolumeSource)
    - [GlusterfsPersistentVolumeSource](#k8s-io-api-core-v1-GlusterfsPersistentVolumeSource)
    - [GlusterfsVolumeSource](#k8s-io-api-core-v1-GlusterfsVolumeSource)
    - [HTTPGetAction](#k8s-io-api-core-v1-HTTPGetAction)
    - [HTTPHeader](#k8s-io-api-core-v1-HTTPHeader)
    - [HostAlias](#k8s-io-api-core-v1-HostAlias)
    - [HostIP](#k8s-io-api-core-v1-HostIP)
    - [HostPathVolumeSource](#k8s-io-api-core-v1-HostPathVolumeSource)
    - [ISCSIPersistentVolumeSource](#k8s-io-api-core-v1-ISCSIPersistentVolumeSource)
    - [ISCSIVolumeSource](#k8s-io-api-core-v1-ISCSIVolumeSource)
    - [ImageVolumeSource](#k8s-io-api-core-v1-ImageVolumeSource)
    - [KeyToPath](#k8s-io-api-core-v1-KeyToPath)
    - [Lifecycle](#k8s-io-api-core-v1-Lifecycle)
    - [LifecycleHandler](#k8s-io-api-core-v1-LifecycleHandler)
    - [LimitRange](#k8s-io-api-core-v1-LimitRange)
    - [LimitRangeItem](#k8s-io-api-core-v1-LimitRangeItem)
    - [LimitRangeItem.DefaultEntry](#k8s-io-api-core-v1-LimitRangeItem-DefaultEntry)
    - [LimitRangeItem.DefaultRequestEntry](#k8s-io-api-core-v1-LimitRangeItem-DefaultRequestEntry)
    - [LimitRangeItem.MaxEntry](#k8s-io-api-core-v1-LimitRangeItem-MaxEntry)
    - [LimitRangeItem.MaxLimitRequestRatioEntry](#k8s-io-api-core-v1-LimitRangeItem-MaxLimitRequestRatioEntry)
    - [LimitRangeItem.MinEntry](#k8s-io-api-core-v1-LimitRangeItem-MinEntry)
    - [LimitRangeList](#k8s-io-api-core-v1-LimitRangeList)
    - [LimitRangeSpec](#k8s-io-api-core-v1-LimitRangeSpec)
    - [LinuxContainerUser](#k8s-io-api-core-v1-LinuxContainerUser)
    - [List](#k8s-io-api-core-v1-List)
    - [LoadBalancerIngress](#k8s-io-api-core-v1-LoadBalancerIngress)
    - [LoadBalancerStatus](#k8s-io-api-core-v1-LoadBalancerStatus)
    - [LocalObjectReference](#k8s-io-api-core-v1-LocalObjectReference)
    - [LocalVolumeSource](#k8s-io-api-core-v1-LocalVolumeSource)
    - [ModifyVolumeStatus](#k8s-io-api-core-v1-ModifyVolumeStatus)
    - [NFSVolumeSource](#k8s-io-api-core-v1-NFSVolumeSource)
    - [Namespace](#k8s-io-api-core-v1-Namespace)
    - [NamespaceCondition](#k8s-io-api-core-v1-NamespaceCondition)
    - [NamespaceList](#k8s-io-api-core-v1-NamespaceList)
    - [NamespaceSpec](#k8s-io-api-core-v1-NamespaceSpec)
    - [NamespaceStatus](#k8s-io-api-core-v1-NamespaceStatus)
    - [Node](#k8s-io-api-core-v1-Node)
    - [NodeAddress](#k8s-io-api-core-v1-NodeAddress)
    - [NodeAffinity](#k8s-io-api-core-v1-NodeAffinity)
    - [NodeCondition](#k8s-io-api-core-v1-NodeCondition)
    - [NodeConfigSource](#k8s-io-api-core-v1-NodeConfigSource)
    - [NodeConfigStatus](#k8s-io-api-core-v1-NodeConfigStatus)
    - [NodeDaemonEndpoints](#k8s-io-api-core-v1-NodeDaemonEndpoints)
    - [NodeFeatures](#k8s-io-api-core-v1-NodeFeatures)
    - [NodeList](#k8s-io-api-core-v1-NodeList)
    - [NodeProxyOptions](#k8s-io-api-core-v1-NodeProxyOptions)
    - [NodeRuntimeHandler](#k8s-io-api-core-v1-NodeRuntimeHandler)
    - [NodeRuntimeHandlerFeatures](#k8s-io-api-core-v1-NodeRuntimeHandlerFeatures)
    - [NodeSelector](#k8s-io-api-core-v1-NodeSelector)
    - [NodeSelectorRequirement](#k8s-io-api-core-v1-NodeSelectorRequirement)
    - [NodeSelectorTerm](#k8s-io-api-core-v1-NodeSelectorTerm)
    - [NodeSpec](#k8s-io-api-core-v1-NodeSpec)
    - [NodeStatus](#k8s-io-api-core-v1-NodeStatus)
    - [NodeStatus.AllocatableEntry](#k8s-io-api-core-v1-NodeStatus-AllocatableEntry)
    - [NodeStatus.CapacityEntry](#k8s-io-api-core-v1-NodeStatus-CapacityEntry)
    - [NodeSwapStatus](#k8s-io-api-core-v1-NodeSwapStatus)
    - [NodeSystemInfo](#k8s-io-api-core-v1-NodeSystemInfo)
    - [ObjectFieldSelector](#k8s-io-api-core-v1-ObjectFieldSelector)
    - [ObjectReference](#k8s-io-api-core-v1-ObjectReference)
    - [PersistentVolume](#k8s-io-api-core-v1-PersistentVolume)
    - [PersistentVolumeClaim](#k8s-io-api-core-v1-PersistentVolumeClaim)
    - [PersistentVolumeClaimCondition](#k8s-io-api-core-v1-PersistentVolumeClaimCondition)
    - [PersistentVolumeClaimList](#k8s-io-api-core-v1-PersistentVolumeClaimList)
    - [PersistentVolumeClaimSpec](#k8s-io-api-core-v1-PersistentVolumeClaimSpec)
    - [PersistentVolumeClaimStatus](#k8s-io-api-core-v1-PersistentVolumeClaimStatus)
    - [PersistentVolumeClaimStatus.AllocatedResourceStatusesEntry](#k8s-io-api-core-v1-PersistentVolumeClaimStatus-AllocatedResourceStatusesEntry)
    - [PersistentVolumeClaimStatus.AllocatedResourcesEntry](#k8s-io-api-core-v1-PersistentVolumeClaimStatus-AllocatedResourcesEntry)
    - [PersistentVolumeClaimStatus.CapacityEntry](#k8s-io-api-core-v1-PersistentVolumeClaimStatus-CapacityEntry)
    - [PersistentVolumeClaimTemplate](#k8s-io-api-core-v1-PersistentVolumeClaimTemplate)
    - [PersistentVolumeClaimVolumeSource](#k8s-io-api-core-v1-PersistentVolumeClaimVolumeSource)
    - [PersistentVolumeList](#k8s-io-api-core-v1-PersistentVolumeList)
    - [PersistentVolumeSource](#k8s-io-api-core-v1-PersistentVolumeSource)
    - [PersistentVolumeSpec](#k8s-io-api-core-v1-PersistentVolumeSpec)
    - [PersistentVolumeSpec.CapacityEntry](#k8s-io-api-core-v1-PersistentVolumeSpec-CapacityEntry)
    - [PersistentVolumeStatus](#k8s-io-api-core-v1-PersistentVolumeStatus)
    - [PhotonPersistentDiskVolumeSource](#k8s-io-api-core-v1-PhotonPersistentDiskVolumeSource)
    - [Pod](#k8s-io-api-core-v1-Pod)
    - [PodAffinity](#k8s-io-api-core-v1-PodAffinity)
    - [PodAffinityTerm](#k8s-io-api-core-v1-PodAffinityTerm)
    - [PodAntiAffinity](#k8s-io-api-core-v1-PodAntiAffinity)
    - [PodAttachOptions](#k8s-io-api-core-v1-PodAttachOptions)
    - [PodCondition](#k8s-io-api-core-v1-PodCondition)
    - [PodDNSConfig](#k8s-io-api-core-v1-PodDNSConfig)
    - [PodDNSConfigOption](#k8s-io-api-core-v1-PodDNSConfigOption)
    - [PodExecOptions](#k8s-io-api-core-v1-PodExecOptions)
    - [PodIP](#k8s-io-api-core-v1-PodIP)
    - [PodList](#k8s-io-api-core-v1-PodList)
    - [PodLogOptions](#k8s-io-api-core-v1-PodLogOptions)
    - [PodOS](#k8s-io-api-core-v1-PodOS)
    - [PodPortForwardOptions](#k8s-io-api-core-v1-PodPortForwardOptions)
    - [PodProxyOptions](#k8s-io-api-core-v1-PodProxyOptions)
    - [PodReadinessGate](#k8s-io-api-core-v1-PodReadinessGate)
    - [PodResourceClaim](#k8s-io-api-core-v1-PodResourceClaim)
    - [PodResourceClaimStatus](#k8s-io-api-core-v1-PodResourceClaimStatus)
    - [PodSchedulingGate](#k8s-io-api-core-v1-PodSchedulingGate)
    - [PodSecurityContext](#k8s-io-api-core-v1-PodSecurityContext)
    - [PodSignature](#k8s-io-api-core-v1-PodSignature)
    - [PodSpec](#k8s-io-api-core-v1-PodSpec)
    - [PodSpec.NodeSelectorEntry](#k8s-io-api-core-v1-PodSpec-NodeSelectorEntry)
    - [PodSpec.OverheadEntry](#k8s-io-api-core-v1-PodSpec-OverheadEntry)
    - [PodStatus](#k8s-io-api-core-v1-PodStatus)
    - [PodStatusResult](#k8s-io-api-core-v1-PodStatusResult)
    - [PodTemplate](#k8s-io-api-core-v1-PodTemplate)
    - [PodTemplateList](#k8s-io-api-core-v1-PodTemplateList)
    - [PodTemplateSpec](#k8s-io-api-core-v1-PodTemplateSpec)
    - [PortStatus](#k8s-io-api-core-v1-PortStatus)
    - [PortworxVolumeSource](#k8s-io-api-core-v1-PortworxVolumeSource)
    - [Preconditions](#k8s-io-api-core-v1-Preconditions)
    - [PreferAvoidPodsEntry](#k8s-io-api-core-v1-PreferAvoidPodsEntry)
    - [PreferredSchedulingTerm](#k8s-io-api-core-v1-PreferredSchedulingTerm)
    - [Probe](#k8s-io-api-core-v1-Probe)
    - [ProbeHandler](#k8s-io-api-core-v1-ProbeHandler)
    - [ProjectedVolumeSource](#k8s-io-api-core-v1-ProjectedVolumeSource)
    - [QuobyteVolumeSource](#k8s-io-api-core-v1-QuobyteVolumeSource)
    - [RBDPersistentVolumeSource](#k8s-io-api-core-v1-RBDPersistentVolumeSource)
    - [RBDVolumeSource](#k8s-io-api-core-v1-RBDVolumeSource)
    - [RangeAllocation](#k8s-io-api-core-v1-RangeAllocation)
    - [ReplicationController](#k8s-io-api-core-v1-ReplicationController)
    - [ReplicationControllerCondition](#k8s-io-api-core-v1-ReplicationControllerCondition)
    - [ReplicationControllerList](#k8s-io-api-core-v1-ReplicationControllerList)
    - [ReplicationControllerSpec](#k8s-io-api-core-v1-ReplicationControllerSpec)
    - [ReplicationControllerSpec.SelectorEntry](#k8s-io-api-core-v1-ReplicationControllerSpec-SelectorEntry)
    - [ReplicationControllerStatus](#k8s-io-api-core-v1-ReplicationControllerStatus)
    - [ResourceClaim](#k8s-io-api-core-v1-ResourceClaim)
    - [ResourceFieldSelector](#k8s-io-api-core-v1-ResourceFieldSelector)
    - [ResourceHealth](#k8s-io-api-core-v1-ResourceHealth)
    - [ResourceQuota](#k8s-io-api-core-v1-ResourceQuota)
    - [ResourceQuotaList](#k8s-io-api-core-v1-ResourceQuotaList)
    - [ResourceQuotaSpec](#k8s-io-api-core-v1-ResourceQuotaSpec)
    - [ResourceQuotaSpec.HardEntry](#k8s-io-api-core-v1-ResourceQuotaSpec-HardEntry)
    - [ResourceQuotaStatus](#k8s-io-api-core-v1-ResourceQuotaStatus)
    - [ResourceQuotaStatus.HardEntry](#k8s-io-api-core-v1-ResourceQuotaStatus-HardEntry)
    - [ResourceQuotaStatus.UsedEntry](#k8s-io-api-core-v1-ResourceQuotaStatus-UsedEntry)
    - [ResourceRequirements](#k8s-io-api-core-v1-ResourceRequirements)
    - [ResourceRequirements.LimitsEntry](#k8s-io-api-core-v1-ResourceRequirements-LimitsEntry)
    - [ResourceRequirements.RequestsEntry](#k8s-io-api-core-v1-ResourceRequirements-RequestsEntry)
    - [ResourceStatus](#k8s-io-api-core-v1-ResourceStatus)
    - [SELinuxOptions](#k8s-io-api-core-v1-SELinuxOptions)
    - [ScaleIOPersistentVolumeSource](#k8s-io-api-core-v1-ScaleIOPersistentVolumeSource)
    - [ScaleIOVolumeSource](#k8s-io-api-core-v1-ScaleIOVolumeSource)
    - [ScopeSelector](#k8s-io-api-core-v1-ScopeSelector)
    - [ScopedResourceSelectorRequirement](#k8s-io-api-core-v1-ScopedResourceSelectorRequirement)
    - [SeccompProfile](#k8s-io-api-core-v1-SeccompProfile)
    - [Secret](#k8s-io-api-core-v1-Secret)
    - [Secret.DataEntry](#k8s-io-api-core-v1-Secret-DataEntry)
    - [Secret.StringDataEntry](#k8s-io-api-core-v1-Secret-StringDataEntry)
    - [SecretEnvSource](#k8s-io-api-core-v1-SecretEnvSource)
    - [SecretKeySelector](#k8s-io-api-core-v1-SecretKeySelector)
    - [SecretList](#k8s-io-api-core-v1-SecretList)
    - [SecretProjection](#k8s-io-api-core-v1-SecretProjection)
    - [SecretReference](#k8s-io-api-core-v1-SecretReference)
    - [SecretVolumeSource](#k8s-io-api-core-v1-SecretVolumeSource)
    - [SecurityContext](#k8s-io-api-core-v1-SecurityContext)
    - [SerializedReference](#k8s-io-api-core-v1-SerializedReference)
    - [Service](#k8s-io-api-core-v1-Service)
    - [ServiceAccount](#k8s-io-api-core-v1-ServiceAccount)
    - [ServiceAccountList](#k8s-io-api-core-v1-ServiceAccountList)
    - [ServiceAccountTokenProjection](#k8s-io-api-core-v1-ServiceAccountTokenProjection)
    - [ServiceList](#k8s-io-api-core-v1-ServiceList)
    - [ServicePort](#k8s-io-api-core-v1-ServicePort)
    - [ServiceProxyOptions](#k8s-io-api-core-v1-ServiceProxyOptions)
    - [ServiceSpec](#k8s-io-api-core-v1-ServiceSpec)
    - [ServiceSpec.SelectorEntry](#k8s-io-api-core-v1-ServiceSpec-SelectorEntry)
    - [ServiceStatus](#k8s-io-api-core-v1-ServiceStatus)
    - [SessionAffinityConfig](#k8s-io-api-core-v1-SessionAffinityConfig)
    - [SleepAction](#k8s-io-api-core-v1-SleepAction)
    - [StorageOSPersistentVolumeSource](#k8s-io-api-core-v1-StorageOSPersistentVolumeSource)
    - [StorageOSVolumeSource](#k8s-io-api-core-v1-StorageOSVolumeSource)
    - [Sysctl](#k8s-io-api-core-v1-Sysctl)
    - [TCPSocketAction](#k8s-io-api-core-v1-TCPSocketAction)
    - [Taint](#k8s-io-api-core-v1-Taint)
    - [Toleration](#k8s-io-api-core-v1-Toleration)
    - [TopologySelectorLabelRequirement](#k8s-io-api-core-v1-TopologySelectorLabelRequirement)
    - [TopologySelectorTerm](#k8s-io-api-core-v1-TopologySelectorTerm)
    - [TopologySpreadConstraint](#k8s-io-api-core-v1-TopologySpreadConstraint)
    - [TypedLocalObjectReference](#k8s-io-api-core-v1-TypedLocalObjectReference)
    - [TypedObjectReference](#k8s-io-api-core-v1-TypedObjectReference)
    - [Volume](#k8s-io-api-core-v1-Volume)
    - [VolumeDevice](#k8s-io-api-core-v1-VolumeDevice)
    - [VolumeMount](#k8s-io-api-core-v1-VolumeMount)
    - [VolumeMountStatus](#k8s-io-api-core-v1-VolumeMountStatus)
    - [VolumeNodeAffinity](#k8s-io-api-core-v1-VolumeNodeAffinity)
    - [VolumeProjection](#k8s-io-api-core-v1-VolumeProjection)
    - [VolumeResourceRequirements](#k8s-io-api-core-v1-VolumeResourceRequirements)
    - [VolumeResourceRequirements.LimitsEntry](#k8s-io-api-core-v1-VolumeResourceRequirements-LimitsEntry)
    - [VolumeResourceRequirements.RequestsEntry](#k8s-io-api-core-v1-VolumeResourceRequirements-RequestsEntry)
    - [VolumeSource](#k8s-io-api-core-v1-VolumeSource)
    - [VsphereVirtualDiskVolumeSource](#k8s-io-api-core-v1-VsphereVirtualDiskVolumeSource)
    - [WeightedPodAffinityTerm](#k8s-io-api-core-v1-WeightedPodAffinityTerm)
    - [WindowsSecurityContextOptions](#k8s-io-api-core-v1-WindowsSecurityContextOptions)
  
- [k8s.io/api/batch/v1/generated.proto](#k8s-io_api_batch_v1_generated-proto)
    - [CronJob](#k8s-io-api-batch-v1-CronJob)
    - [CronJobList](#k8s-io-api-batch-v1-CronJobList)
    - [CronJobSpec](#k8s-io-api-batch-v1-CronJobSpec)
    - [CronJobStatus](#k8s-io-api-batch-v1-CronJobStatus)
    - [Job](#k8s-io-api-batch-v1-Job)
    - [JobCondition](#k8s-io-api-batch-v1-JobCondition)
    - [JobList](#k8s-io-api-batch-v1-JobList)
    - [JobSpec](#k8s-io-api-batch-v1-JobSpec)
    - [JobStatus](#k8s-io-api-batch-v1-JobStatus)
    - [JobTemplateSpec](#k8s-io-api-batch-v1-JobTemplateSpec)
    - [PodFailurePolicy](#k8s-io-api-batch-v1-PodFailurePolicy)
    - [PodFailurePolicyOnExitCodesRequirement](#k8s-io-api-batch-v1-PodFailurePolicyOnExitCodesRequirement)
    - [PodFailurePolicyOnPodConditionsPattern](#k8s-io-api-batch-v1-PodFailurePolicyOnPodConditionsPattern)
    - [PodFailurePolicyRule](#k8s-io-api-batch-v1-PodFailurePolicyRule)
    - [SuccessPolicy](#k8s-io-api-batch-v1-SuccessPolicy)
    - [SuccessPolicyRule](#k8s-io-api-batch-v1-SuccessPolicyRule)
    - [UncountedTerminatedPods](#k8s-io-api-batch-v1-UncountedTerminatedPods)
  
- [k8s.io/api/rbac/v1/generated.proto](#k8s-io_api_rbac_v1_generated-proto)
    - [AggregationRule](#k8s-io-api-rbac-v1-AggregationRule)
    - [ClusterRole](#k8s-io-api-rbac-v1-ClusterRole)
    - [ClusterRoleBinding](#k8s-io-api-rbac-v1-ClusterRoleBinding)
    - [ClusterRoleBindingList](#k8s-io-api-rbac-v1-ClusterRoleBindingList)
    - [ClusterRoleList](#k8s-io-api-rbac-v1-ClusterRoleList)
    - [PolicyRule](#k8s-io-api-rbac-v1-PolicyRule)
    - [Role](#k8s-io-api-rbac-v1-Role)
    - [RoleBinding](#k8s-io-api-rbac-v1-RoleBinding)
    - [RoleBindingList](#k8s-io-api-rbac-v1-RoleBindingList)
    - [RoleList](#k8s-io-api-rbac-v1-RoleList)
    - [RoleRef](#k8s-io-api-rbac-v1-RoleRef)
    - [Subject](#k8s-io-api-rbac-v1-Subject)
  
- [k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1/generated.proto](#k8s-io_apiextensions-apiserver_pkg_apis_apiextensions_v1_generated-proto)
    - [ConversionRequest](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-ConversionRequest)
    - [ConversionResponse](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-ConversionResponse)
    - [ConversionReview](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-ConversionReview)
    - [CustomResourceColumnDefinition](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceColumnDefinition)
    - [CustomResourceConversion](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceConversion)
    - [CustomResourceDefinition](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceDefinition)
    - [CustomResourceDefinitionCondition](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceDefinitionCondition)
    - [CustomResourceDefinitionList](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceDefinitionList)
    - [CustomResourceDefinitionNames](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceDefinitionNames)
    - [CustomResourceDefinitionSpec](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceDefinitionSpec)
    - [CustomResourceDefinitionStatus](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceDefinitionStatus)
    - [CustomResourceDefinitionVersion](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceDefinitionVersion)
    - [CustomResourceSubresourceScale](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceSubresourceScale)
    - [CustomResourceSubresourceStatus](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceSubresourceStatus)
    - [CustomResourceSubresources](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceSubresources)
    - [CustomResourceValidation](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceValidation)
    - [ExternalDocumentation](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-ExternalDocumentation)
    - [JSON](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSON)
    - [JSONSchemaProps](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaProps)
    - [JSONSchemaProps.DefinitionsEntry](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaProps-DefinitionsEntry)
    - [JSONSchemaProps.DependenciesEntry](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaProps-DependenciesEntry)
    - [JSONSchemaProps.PatternPropertiesEntry](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaProps-PatternPropertiesEntry)
    - [JSONSchemaProps.PropertiesEntry](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaProps-PropertiesEntry)
    - [JSONSchemaPropsOrArray](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaPropsOrArray)
    - [JSONSchemaPropsOrBool](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaPropsOrBool)
    - [JSONSchemaPropsOrStringArray](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaPropsOrStringArray)
    - [SelectableField](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-SelectableField)
    - [ServiceReference](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-ServiceReference)
    - [ValidationRule](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-ValidationRule)
    - [WebhookClientConfig](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-WebhookClientConfig)
    - [WebhookConversion](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-WebhookConversion)
  
- [k8s.io/apimachinery/pkg/util/intstr/generated.proto](#k8s-io_apimachinery_pkg_util_intstr_generated-proto)
    - [IntOrString](#k8s-io-apimachinery-pkg-util-intstr-IntOrString)
  
- [k8s.io/apimachinery/pkg/api/resource/generated.proto](#k8s-io_apimachinery_pkg_api_resource_generated-proto)
    - [Quantity](#k8s-io-apimachinery-pkg-api-resource-Quantity)
    - [QuantityValue](#k8s-io-apimachinery-pkg-api-resource-QuantityValue)
  
- [k8s.io/apimachinery/pkg/runtime/schema/generated.proto](#k8s-io_apimachinery_pkg_runtime_schema_generated-proto)
- [k8s.io/apimachinery/pkg/runtime/generated.proto](#k8s-io_apimachinery_pkg_runtime_generated-proto)
    - [RawExtension](#k8s-io-apimachinery-pkg-runtime-RawExtension)
    - [TypeMeta](#k8s-io-apimachinery-pkg-runtime-TypeMeta)
    - [Unknown](#k8s-io-apimachinery-pkg-runtime-Unknown)
  
- [k8s.io/apimachinery/pkg/apis/meta/v1/generated.proto](#k8s-io_apimachinery_pkg_apis_meta_v1_generated-proto)
    - [APIGroup](#k8s-io-apimachinery-pkg-apis-meta-v1-APIGroup)
    - [APIGroupList](#k8s-io-apimachinery-pkg-apis-meta-v1-APIGroupList)
    - [APIResource](#k8s-io-apimachinery-pkg-apis-meta-v1-APIResource)
    - [APIResourceList](#k8s-io-apimachinery-pkg-apis-meta-v1-APIResourceList)
    - [APIVersions](#k8s-io-apimachinery-pkg-apis-meta-v1-APIVersions)
    - [ApplyOptions](#k8s-io-apimachinery-pkg-apis-meta-v1-ApplyOptions)
    - [Condition](#k8s-io-apimachinery-pkg-apis-meta-v1-Condition)
    - [CreateOptions](#k8s-io-apimachinery-pkg-apis-meta-v1-CreateOptions)
    - [DeleteOptions](#k8s-io-apimachinery-pkg-apis-meta-v1-DeleteOptions)
    - [Duration](#k8s-io-apimachinery-pkg-apis-meta-v1-Duration)
    - [FieldSelectorRequirement](#k8s-io-apimachinery-pkg-apis-meta-v1-FieldSelectorRequirement)
    - [FieldsV1](#k8s-io-apimachinery-pkg-apis-meta-v1-FieldsV1)
    - [GetOptions](#k8s-io-apimachinery-pkg-apis-meta-v1-GetOptions)
    - [GroupKind](#k8s-io-apimachinery-pkg-apis-meta-v1-GroupKind)
    - [GroupResource](#k8s-io-apimachinery-pkg-apis-meta-v1-GroupResource)
    - [GroupVersion](#k8s-io-apimachinery-pkg-apis-meta-v1-GroupVersion)
    - [GroupVersionForDiscovery](#k8s-io-apimachinery-pkg-apis-meta-v1-GroupVersionForDiscovery)
    - [GroupVersionKind](#k8s-io-apimachinery-pkg-apis-meta-v1-GroupVersionKind)
    - [GroupVersionResource](#k8s-io-apimachinery-pkg-apis-meta-v1-GroupVersionResource)
    - [LabelSelector](#k8s-io-apimachinery-pkg-apis-meta-v1-LabelSelector)
    - [LabelSelector.MatchLabelsEntry](#k8s-io-apimachinery-pkg-apis-meta-v1-LabelSelector-MatchLabelsEntry)
    - [LabelSelectorRequirement](#k8s-io-apimachinery-pkg-apis-meta-v1-LabelSelectorRequirement)
    - [List](#k8s-io-apimachinery-pkg-apis-meta-v1-List)
    - [ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta)
    - [ListOptions](#k8s-io-apimachinery-pkg-apis-meta-v1-ListOptions)
    - [ManagedFieldsEntry](#k8s-io-apimachinery-pkg-apis-meta-v1-ManagedFieldsEntry)
    - [MicroTime](#k8s-io-apimachinery-pkg-apis-meta-v1-MicroTime)
    - [ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta)
    - [ObjectMeta.AnnotationsEntry](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta-AnnotationsEntry)
    - [ObjectMeta.LabelsEntry](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta-LabelsEntry)
    - [OwnerReference](#k8s-io-apimachinery-pkg-apis-meta-v1-OwnerReference)
    - [PartialObjectMetadata](#k8s-io-apimachinery-pkg-apis-meta-v1-PartialObjectMetadata)
    - [PartialObjectMetadataList](#k8s-io-apimachinery-pkg-apis-meta-v1-PartialObjectMetadataList)
    - [Patch](#k8s-io-apimachinery-pkg-apis-meta-v1-Patch)
    - [PatchOptions](#k8s-io-apimachinery-pkg-apis-meta-v1-PatchOptions)
    - [Preconditions](#k8s-io-apimachinery-pkg-apis-meta-v1-Preconditions)
    - [RootPaths](#k8s-io-apimachinery-pkg-apis-meta-v1-RootPaths)
    - [ServerAddressByClientCIDR](#k8s-io-apimachinery-pkg-apis-meta-v1-ServerAddressByClientCIDR)
    - [Status](#k8s-io-apimachinery-pkg-apis-meta-v1-Status)
    - [StatusCause](#k8s-io-apimachinery-pkg-apis-meta-v1-StatusCause)
    - [StatusDetails](#k8s-io-apimachinery-pkg-apis-meta-v1-StatusDetails)
    - [TableOptions](#k8s-io-apimachinery-pkg-apis-meta-v1-TableOptions)
    - [Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time)
    - [Timestamp](#k8s-io-apimachinery-pkg-apis-meta-v1-Timestamp)
    - [TypeMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-TypeMeta)
    - [UpdateOptions](#k8s-io-apimachinery-pkg-apis-meta-v1-UpdateOptions)
    - [Verbs](#k8s-io-apimachinery-pkg-apis-meta-v1-Verbs)
    - [WatchEvent](#k8s-io-apimachinery-pkg-apis-meta-v1-WatchEvent)
  
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



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional |  |
| values | [string](#string) | repeated |  |






<a name="github-com-akuity-kargo-api-rbac-v1alpha1-ResourceDetails"></a>

### ResourceDetails



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resourceType | [string](#string) | optional |  |
| resourceName | [string](#string) | optional |  |
| verbs | [string](#string) | repeated |  |






<a name="github-com-akuity-kargo-api-rbac-v1alpha1-Role"></a>

### Role
+kubebuilder:object:root=true


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional |  |
| kargoManaged | [bool](#bool) | optional |  |
| claims | [Claim](#github-com-akuity-kargo-api-rbac-v1alpha1-Claim) | repeated |  |
| rules | [k8s.io.api.rbac.v1.PolicyRule](#k8s-io-api-rbac-v1-PolicyRule) | repeated |  |






<a name="github-com-akuity-kargo-api-rbac-v1alpha1-RoleResources"></a>

### RoleResources
+kubebuilder:object:root=true


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional |  |
| serviceAccount | [k8s.io.api.core.v1.ServiceAccount](#k8s-io-api-core-v1-ServiceAccount) | optional |  |
| roles | [k8s.io.api.rbac.v1.Role](#k8s-io-api-rbac-v1-Role) | repeated |  |
| roleBindings | [k8s.io.api.rbac.v1.RoleBinding](#k8s-io-api-rbac-v1-RoleBinding) | repeated |  |





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


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name is the name of the argument.

+kubebuilder:validation:Required |
| value | [string](#string) | optional | Value is the value of the argument.

+kubebuilder:validation:Required |






<a name="github-com-akuity-kargo-api-v1alpha1-AnalysisRunMetadata"></a>

### AnalysisRunMetadata
AnalysisRunMetadata contains optional metadata that should be applied to all
AnalysisRuns.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| labels | [AnalysisRunMetadata.LabelsEntry](#github-com-akuity-kargo-api-v1alpha1-AnalysisRunMetadata-LabelsEntry) | repeated | Additional labels to apply to an AnalysisRun. |
| annotations | [AnalysisRunMetadata.AnnotationsEntry](#github-com-akuity-kargo-api-v1alpha1-AnalysisRunMetadata-AnnotationsEntry) | repeated | Additional annotations to apply to an AnalysisRun. |






<a name="github-com-akuity-kargo-api-v1alpha1-AnalysisRunMetadata-AnnotationsEntry"></a>

### AnalysisRunMetadata.AnnotationsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [string](#string) | optional |  |






<a name="github-com-akuity-kargo-api-v1alpha1-AnalysisRunMetadata-LabelsEntry"></a>

### AnalysisRunMetadata.LabelsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [string](#string) | optional |  |






<a name="github-com-akuity-kargo-api-v1alpha1-AnalysisRunReference"></a>

### AnalysisRunReference
AnalysisRunReference is a reference to an AnalysisRun.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| namespace | [string](#string) | optional | Namespace is the namespace of the AnalysisRun. |
| name | [string](#string) | optional | Name is the name of the AnalysisRun. |
| phase | [string](#string) | optional | Phase is the last observed phase of the AnalysisRun referenced by Name. |






<a name="github-com-akuity-kargo-api-v1alpha1-AnalysisTemplateReference"></a>

### AnalysisTemplateReference
AnalysisTemplateReference is a reference to an AnalysisTemplate.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name is the name of the AnalysisTemplate in the same project/namespace as the Stage.

+kubebuilder:validation:Required |
| kind | [string](#string) | optional | Kind is the type of the AnalysisTemplate. Can be either AnalysisTemplate or ClusterAnalysisTemplate, default is AnalysisTemplate.

+kubebuilder:validation:Optional +kubebuilder:validation:Enum=AnalysisTemplate;ClusterAnalysisTemplate |






<a name="github-com-akuity-kargo-api-v1alpha1-ApprovedStage"></a>

### ApprovedStage
ApprovedStage describes a Stage for which Freight has been (manually)
approved.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| approvedAt | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | ApprovedAt is the time at which the Freight was approved for the Stage. |






<a name="github-com-akuity-kargo-api-v1alpha1-ArgoCDAppHealthStatus"></a>

### ArgoCDAppHealthStatus
ArgoCDAppHealthStatus describes the health of an ArgoCD Application.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) | optional |  |
| message | [string](#string) | optional |  |






<a name="github-com-akuity-kargo-api-v1alpha1-ArgoCDAppStatus"></a>

### ArgoCDAppStatus
ArgoCDAppStatus describes the current state of a single ArgoCD Application.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| namespace | [string](#string) | optional | Namespace is the namespace of the ArgoCD Application. |
| name | [string](#string) | optional | Name is the name of the ArgoCD Application. |
| healthStatus | [ArgoCDAppHealthStatus](#github-com-akuity-kargo-api-v1alpha1-ArgoCDAppHealthStatus) | optional | HealthStatus is the health of the ArgoCD Application. |
| syncStatus | [ArgoCDAppSyncStatus](#github-com-akuity-kargo-api-v1alpha1-ArgoCDAppSyncStatus) | optional | SyncStatus is the sync status of the ArgoCD Application. |






<a name="github-com-akuity-kargo-api-v1alpha1-ArgoCDAppSyncStatus"></a>

### ArgoCDAppSyncStatus
ArgoCDAppSyncStatus describes the sync status of an ArgoCD Application.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) | optional |  |
| revision | [string](#string) | optional |  |
| revisions | [string](#string) | repeated |  |






<a name="github-com-akuity-kargo-api-v1alpha1-ArtifactoryWebhookReceiverConfig"></a>

### ArtifactoryWebhookReceiverConfig
ArtifactoryWebhookReceiverConfig describes a webhook receiver that is
compatible with JFrog Artifactory payloads.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| secretRef | [k8s.io.api.core.v1.LocalObjectReference](#k8s-io-api-core-v1-LocalObjectReference) | optional | SecretRef contains a reference to a Secret. For Project-scoped webhook receivers, the referenced Secret must be in the same namespace as the ProjectConfig.

For cluster-scoped webhook receivers, the referenced Secret must be in the designated "cluster Secrets" namespace.

The Secret's data map is expected to contain a `secret-token` key whose value is the shared secret used to authenticate the webhook requests sent by JFrog Artifactory. For more information please refer to the JFrog Artifactory documentation: https://jfrog.com/help/r/jfrog-platform-administration-documentation/webhooks

+kubebuilder:validation:Required |






<a name="github-com-akuity-kargo-api-v1alpha1-AzureWebhookReceiverConfig"></a>

### AzureWebhookReceiverConfig
AzureWebhookReceiverConfig describes a webhook receiver that is compatible
with Azure Container Registry (ACR) and Azure DevOps payloads.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| secretRef | [k8s.io.api.core.v1.LocalObjectReference](#k8s-io-api-core-v1-LocalObjectReference) | optional | SecretRef contains a reference to a Secret. For Project-scoped webhook receivers, the referenced Secret must be in the same namespace as the ProjectConfig.

For cluster-scoped webhook receivers, the referenced Secret must be in the designated "cluster Secrets" namespace.

The Secret's data map is expected to contain a `secret` key whose value does NOT need to be shared directly with Azure when registering a webhook. It is used only by Kargo to create a complex, hard-to-guess URL, which implicitly serves as a shared secret. For more information about Azure webhooks, please refer to the Azure documentation:

 Azure Container Registry: 	https://learn.microsoft.com/en-us/azure/container-registry/container-registry-repositories

 Azure DevOps: 	http://learn.microsoft.com/en-us/azure/devops/service-hooks/services/webhooks?view=azure-devops

+kubebuilder:validation:Required |






<a name="github-com-akuity-kargo-api-v1alpha1-BitbucketWebhookReceiverConfig"></a>

### BitbucketWebhookReceiverConfig
BitbucketWebhookReceiverConfig describes a webhook receiver that is
compatible with Bitbucket payloads.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| secretRef | [k8s.io.api.core.v1.LocalObjectReference](#k8s-io-api-core-v1-LocalObjectReference) | optional | SecretRef contains a reference to a Secret. For Project-scoped webhook receivers, the referenced Secret must be in the same namespace as the ProjectConfig.

For cluster-scoped webhook receivers, the referenced Secret must be in the designated "cluster Secrets" namespace.

The Secret's data map is expected to contain a `secret` key whose value is the shared secret used to authenticate the webhook requests sent by Bitbucket. For more information please refer to the Bitbucket documentation: https://support.atlassian.com/bitbucket-cloud/docs/manage-webhooks/

+kubebuilder:validation:Required |






<a name="github-com-akuity-kargo-api-v1alpha1-Chart"></a>

### Chart
Chart describes a specific version of a Helm chart.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repoURL | [string](#string) | optional | RepoURL specifies the URL of a Helm chart repository. Classic chart repositories (using HTTP/S) can contain differently named charts. When this field points to such a repository, the Name field will specify the name of the chart within the repository. In the case of a repository within an OCI registry, the URL implicitly points to a specific chart and the Name field will be empty. |
| name | [string](#string) | optional | Name specifies the name of the chart. |
| version | [string](#string) | optional | Version specifies a particular version of the chart. |






<a name="github-com-akuity-kargo-api-v1alpha1-ChartDiscoveryResult"></a>

### ChartDiscoveryResult
ChartDiscoveryResult represents the result of a chart discovery operation for
a ChartSubscription.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repoURL | [string](#string) | optional | RepoURL is the repository URL of the Helm chart, as specified in the ChartSubscription.

+kubebuilder:validation:MinLength=1 |
| name | [string](#string) | optional | Name is the name of the Helm chart, as specified in the ChartSubscription. |
| semverConstraint | [string](#string) | optional | SemverConstraint is the constraint for which versions were discovered. This field is optional, and only populated if the ChartSubscription specifies a SemverConstraint. |
| versions | [string](#string) | repeated | Versions is a list of versions discovered by the Warehouse for the ChartSubscription. An empty list indicates that the discovery operation was successful, but no versions matching the ChartSubscription criteria were found.

+optional |






<a name="github-com-akuity-kargo-api-v1alpha1-ChartSubscription"></a>

### ChartSubscription
ChartSubscription defines a subscription to a Helm chart repository.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repoURL | [string](#string) | optional | RepoURL specifies the URL of a Helm chart repository. It may be a classic chart repository (using HTTP/S) OR a repository within an OCI registry. Classic chart repositories can contain differently named charts. When this field points to such a repository, the Name field MUST also be used to specify the name of the desired chart within that repository. In the case of a repository within an OCI registry, the URL implicitly points to a specific chart and the Name field MUST NOT be used. The RepoURL field is required.

+kubebuilder:validation:MinLength=1 +kubebuilder:validation:Pattern=`^(((https?)|(oci))://)([\w\d\.\-]+)(:[\d]+)?(/.*)*$` +akuity:test-kubebuilder-pattern=HelmRepoURL |
| name | [string](#string) | optional | Name specifies the name of a Helm chart to subscribe to within a classic chart repository specified by the RepoURL field. This field is required when the RepoURL field points to a classic chart repository and MUST otherwise be empty. |
| semverConstraint | [string](#string) | optional | SemverConstraint specifies constraints on what new chart versions are permissible. This field is optional. When left unspecified, there will be no constraints, which means the latest version of the chart will always be used. Care should be taken with leaving this field unspecified, as it can lead to the unanticipated rollout of breaking changes. More info: https://github.com/masterminds/semver#checking-version-constraints

+kubebuilder:validation:Optional |
| discoveryLimit | [int32](#int32) | optional | DiscoveryLimit is an optional limit on the number of chart versions that can be discovered for this subscription. The limit is applied after filtering charts based on the SemverConstraint field. When left unspecified, the field is implicitly treated as if its value were "20". The upper limit for this field is 100.

+kubebuilder:validation:Minimum=1 +kubebuilder:validation:Maximum=100 +kubebuilder:default=20 |






<a name="github-com-akuity-kargo-api-v1alpha1-ClusterConfig"></a>

### ClusterConfig
ClusterConfig is a resource type that describes cluster-level Kargo
configuration.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional |  |
| spec | [ClusterConfigSpec](#github-com-akuity-kargo-api-v1alpha1-ClusterConfigSpec) | optional | Spec describes the configuration of a cluster. |
| status | [ClusterConfigStatus](#github-com-akuity-kargo-api-v1alpha1-ClusterConfigStatus) | optional | Status describes the current status of a ClusterConfig. |






<a name="github-com-akuity-kargo-api-v1alpha1-ClusterConfigList"></a>

### ClusterConfigList
ClusterConfigList contains a list of ClusterConfigs.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional |  |
| items | [ClusterConfig](#github-com-akuity-kargo-api-v1alpha1-ClusterConfig) | repeated |  |






<a name="github-com-akuity-kargo-api-v1alpha1-ClusterConfigSpec"></a>

### ClusterConfigSpec
ClusterConfigSpec describes cluster-level Kargo configuration.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| webhookReceivers | [WebhookReceiverConfig](#github-com-akuity-kargo-api-v1alpha1-WebhookReceiverConfig) | repeated | WebhookReceivers describes cluster-scoped webhook receivers used for processing events from various external platforms |






<a name="github-com-akuity-kargo-api-v1alpha1-ClusterConfigStatus"></a>

### ClusterConfigStatus
ClusterConfigStatus describes the current status of a ClusterConfig.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| conditions | [k8s.io.apimachinery.pkg.apis.meta.v1.Condition](#k8s-io-apimachinery-pkg-apis-meta-v1-Condition) | repeated | Conditions contains the last observations of the ClusterConfig's current state.

+patchMergeKey=type +patchStrategy=merge +listType=map +listMapKey=type |
| observedGeneration | [int64](#int64) | optional | ObservedGeneration represents the .metadata.generation that this ClusterConfig was reconciled against. |
| lastHandledRefresh | [string](#string) | optional | LastHandledRefresh holds the value of the most recent AnnotationKeyRefresh annotation that was handled by the controller. This field can be used to determine whether the request to refresh the resource has been handled. +optional |
| webhookReceivers | [WebhookReceiverDetails](#github-com-akuity-kargo-api-v1alpha1-WebhookReceiverDetails) | repeated | WebhookReceivers describes the status of cluster-scoped webhook receivers. |






<a name="github-com-akuity-kargo-api-v1alpha1-ClusterPromotionTask"></a>

### ClusterPromotionTask



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional |  |
| spec | [PromotionTaskSpec](#github-com-akuity-kargo-api-v1alpha1-PromotionTaskSpec) | optional | Spec describes the desired transition of a specific Stage into a specific Freight.

+kubebuilder:validation:Required |






<a name="github-com-akuity-kargo-api-v1alpha1-ClusterPromotionTaskList"></a>

### ClusterPromotionTaskList
ClusterPromotionTaskList contains a list of PromotionTasks.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional |  |
| items | [ClusterPromotionTask](#github-com-akuity-kargo-api-v1alpha1-ClusterPromotionTask) | repeated |  |






<a name="github-com-akuity-kargo-api-v1alpha1-CurrentStage"></a>

### CurrentStage
CurrentStage reflects a Stage's current use of Freight.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| since | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | Since is the time at which the Stage most recently started using the Freight. This can be used to calculate how long the Freight has been in use by the Stage. |






<a name="github-com-akuity-kargo-api-v1alpha1-DiscoveredArtifacts"></a>

### DiscoveredArtifacts
DiscoveredArtifacts holds the artifacts discovered by the Warehouse for its
subscriptions.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| discoveredAt | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | DiscoveredAt is the time at which the Warehouse discovered the artifacts.

+optional |
| git | [GitDiscoveryResult](#github-com-akuity-kargo-api-v1alpha1-GitDiscoveryResult) | repeated | Git holds the commits discovered by the Warehouse for the Git subscriptions.

+optional |
| images | [ImageDiscoveryResult](#github-com-akuity-kargo-api-v1alpha1-ImageDiscoveryResult) | repeated | Images holds the image references discovered by the Warehouse for the image subscriptions.

+optional |
| charts | [ChartDiscoveryResult](#github-com-akuity-kargo-api-v1alpha1-ChartDiscoveryResult) | repeated | Charts holds the charts discovered by the Warehouse for the chart subscriptions.

+optional |






<a name="github-com-akuity-kargo-api-v1alpha1-DiscoveredCommit"></a>

### DiscoveredCommit
DiscoveredCommit represents a commit discovered by a Warehouse for a
GitSubscription.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) | optional | ID is the identifier of the commit. This typically is a SHA-1 hash.

+kubebuilder:validation:MinLength=1 |
| branch | [string](#string) | optional | Branch is the branch in which the commit was found. This field is optional, and populated based on the CommitSelectionStrategy of the GitSubscription. |
| tag | [string](#string) | optional | Tag is the tag that resolved to this commit. This field is optional, and populated based on the CommitSelectionStrategy of the GitSubscription. |
| subject | [string](#string) | optional | Subject is the subject of the commit (i.e. the first line of the commit message). |
| author | [string](#string) | optional | Author is the author of the commit. |
| committer | [string](#string) | optional | Committer is the person who committed the commit. |
| creatorDate | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | CreatorDate is the commit creation date as specified by the commit, or the tagger date if the commit belongs to an annotated tag. |






<a name="github-com-akuity-kargo-api-v1alpha1-DiscoveredImageReference"></a>

### DiscoveredImageReference
DiscoveredImageReference represents an image reference discovered by a
Warehouse for an ImageSubscription.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tag | [string](#string) | optional | Tag is the tag of the image.

+kubebuilder:validation:MinLength=1 +kubebuilder:validation:MaxLength=128 +kubebuilder:validation:Pattern=`^[\w.\-\_]+$` +akuity:test-kubebuilder-pattern=Tag |
| digest | [string](#string) | optional | Digest is the digest of the image.

+kubebuilder:validation:MinLength=1 +kubebuilder:validation:Pattern=`^[a-z0-9]+:[a-f0-9]+$` +akuity:test-kubebuilder-pattern=Digest |
| annotations | [DiscoveredImageReference.AnnotationsEntry](#github-com-akuity-kargo-api-v1alpha1-DiscoveredImageReference-AnnotationsEntry) | repeated | Annotations is a map of key-value pairs that provide additional information about the image. |
| createdAt | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | CreatedAt is the time the image was created. This field is optional, and not populated for every ImageSelectionStrategy. |






<a name="github-com-akuity-kargo-api-v1alpha1-DiscoveredImageReference-AnnotationsEntry"></a>

### DiscoveredImageReference.AnnotationsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [string](#string) | optional |  |






<a name="github-com-akuity-kargo-api-v1alpha1-DockerHubWebhookReceiverConfig"></a>

### DockerHubWebhookReceiverConfig
DockerHubWebhookReceiverConfig describes a webhook receiver that is
compatible with Docker Hub payloads.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| secretRef | [k8s.io.api.core.v1.LocalObjectReference](#k8s-io-api-core-v1-LocalObjectReference) | optional | SecretRef contains a reference to a Secret. For Project-scoped webhook receivers, the referenced Secret must be in the same namespace as the ProjectConfig.

The Secret's data map is expected to contain a `secret` key whose value does NOT need to be shared directly with Docker Hub when registering a webhook. It is used only by Kargo to create a complex, hard-to-guess URL, which implicitly serves as a shared secret. For more information about Docker Hub webhooks, please refer to the Docker documentation: https://docs.docker.com/docker-hub/webhooks/

+kubebuilder:validation:Required |






<a name="github-com-akuity-kargo-api-v1alpha1-ExpressionVariable"></a>

### ExpressionVariable
ExpressionVariable describes a single variable that may be referenced by
expressions in the context of a ClusterPromotionTask, PromotionTask,
Promotion, AnalysisRun arguments, or other objects that support expressions.

It is used to pass information to the expression evaluation engine, and to
allow for dynamic evaluation of expressions based on the variable values.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name is the name of the variable.

+kubebuilder:validation:MinLength=1 +kubebuilder:validation:Pattern=^[a-zA-Z_]\w*$ |
| value | [string](#string) | optional | Value is the value of the variable. It is allowed to utilize expressions in the value. See https://docs.kargo.io/user-guide/reference-docs/expressions for details. |






<a name="github-com-akuity-kargo-api-v1alpha1-Freight"></a>

### Freight
Freight represents a collection of versioned artifacts.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional |  |
| alias | [string](#string) | optional | Alias is a human-friendly alias for a piece of Freight. This is an optional field. A defaulting webhook will sync this field with the value of the kargo.akuity.io/alias label. When the alias label is not present or differs from the value of this field, the defaulting webhook will set the label to the value of this field. If the alias label is present and this field is empty, the defaulting webhook will set the value of this field to the value of the alias label. If this field is empty and the alias label is not present, the defaulting webhook will choose an available alias and assign it to both the field and label. |
| origin | [FreightOrigin](#github-com-akuity-kargo-api-v1alpha1-FreightOrigin) | optional | Origin describes a kind of Freight in terms of its origin.

+kubebuilder:validation:Required |
| commits | [GitCommit](#github-com-akuity-kargo-api-v1alpha1-GitCommit) | repeated | Commits describes specific Git repository commits. |
| images | [Image](#github-com-akuity-kargo-api-v1alpha1-Image) | repeated | Images describes specific versions of specific container images. |
| charts | [Chart](#github-com-akuity-kargo-api-v1alpha1-Chart) | repeated | Charts describes specific versions of specific Helm charts. |
| status | [FreightStatus](#github-com-akuity-kargo-api-v1alpha1-FreightStatus) | optional | Status describes the current status of this Freight. |






<a name="github-com-akuity-kargo-api-v1alpha1-FreightCollection"></a>

### FreightCollection
FreightCollection is a collection of FreightReferences, each of which
represents a piece of Freight that has been selected for deployment to a
Stage.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) | optional | ID is a unique and deterministically calculated identifier for the FreightCollection. It is updated on each use of the UpdateOrPush method. |
| items | [FreightCollection.ItemsEntry](#github-com-akuity-kargo-api-v1alpha1-FreightCollection-ItemsEntry) | repeated | Freight is a map of FreightReference objects, indexed by their Warehouse origin. |
| verificationHistory | [VerificationInfo](#github-com-akuity-kargo-api-v1alpha1-VerificationInfo) | repeated | VerificationHistory is a stack of recent VerificationInfo. By default, the last ten VerificationInfo are stored. |






<a name="github-com-akuity-kargo-api-v1alpha1-FreightCollection-ItemsEntry"></a>

### FreightCollection.ItemsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [FreightReference](#github-com-akuity-kargo-api-v1alpha1-FreightReference) | optional |  |






<a name="github-com-akuity-kargo-api-v1alpha1-FreightList"></a>

### FreightList
FreightList is a list of Freight resources.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional |  |
| items | [Freight](#github-com-akuity-kargo-api-v1alpha1-Freight) | repeated |  |






<a name="github-com-akuity-kargo-api-v1alpha1-FreightOrigin"></a>

### FreightOrigin
FreightOrigin describes a kind of Freight in terms of where it may have
originated.

+protobuf.options.(gogoproto.goproto_stringer)=false


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| kind | [string](#string) | optional | Kind is the kind of resource from which Freight may have originated. At present, this can only be "Warehouse".

+kubebuilder:validation:Required |
| name | [string](#string) | optional | Name is the name of the resource of the kind indicated by the Kind field from which Freight may originate.

+kubebuilder:validation:Required |






<a name="github-com-akuity-kargo-api-v1alpha1-FreightReference"></a>

### FreightReference
FreightReference is a simplified representation of a piece of Freight -- not
a root resource type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name is a system-assigned identifier derived deterministically from the contents of the Freight. I.e., two pieces of Freight can be compared for equality by comparing their Names. |
| origin | [FreightOrigin](#github-com-akuity-kargo-api-v1alpha1-FreightOrigin) | optional | Origin describes a kind of Freight in terms of its origin. |
| commits | [GitCommit](#github-com-akuity-kargo-api-v1alpha1-GitCommit) | repeated | Commits describes specific Git repository commits. |
| images | [Image](#github-com-akuity-kargo-api-v1alpha1-Image) | repeated | Images describes specific versions of specific container images. |
| charts | [Chart](#github-com-akuity-kargo-api-v1alpha1-Chart) | repeated | Charts describes specific versions of specific Helm charts. |






<a name="github-com-akuity-kargo-api-v1alpha1-FreightRequest"></a>

### FreightRequest
FreightRequest expresses a Stage's need for Freight having originated from a
particular Warehouse.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| origin | [FreightOrigin](#github-com-akuity-kargo-api-v1alpha1-FreightOrigin) | optional | Origin specifies from where the requested Freight must have originated. This is a required field.

+kubebuilder:validation:Required |
| sources | [FreightSources](#github-com-akuity-kargo-api-v1alpha1-FreightSources) | optional | Sources describes where the requested Freight may be obtained from. This is a required field. |






<a name="github-com-akuity-kargo-api-v1alpha1-FreightSources"></a>

### FreightSources



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| direct | [bool](#bool) | optional | Direct indicates the requested Freight may be obtained directly from the Warehouse from which it originated. If this field's value is false, then the value of the Stages field must be non-empty. i.e. Between the two fields, at least one source must be specified. |
| stages | [string](#string) | repeated | Stages identifies other "upstream" Stages as potential sources of the requested Freight. If this field's value is empty, then the value of the Direct field must be true. i.e. Between the two fields, at least on source must be specified. |
| requiredSoakTime | [k8s.io.apimachinery.pkg.apis.meta.v1.Duration](#k8s-io-apimachinery-pkg-apis-meta-v1-Duration) | optional | RequiredSoakTime specifies a minimum duration for which the requested Freight must have continuously occupied ("soaked in") in an upstream Stage before becoming available for promotion to this Stage. This is an optional field. If nil or zero, no soak time is required. Any soak time requirement is in ADDITION to the requirement that Freight be verified in an upstream Stage to become available for promotion to this Stage, although a manual approval for promotion to this Stage will supersede any soak time requirement.

+kubebuilder:validation:Type=string +kubebuilder:validation:Pattern=`^([0-9]+(\.[0-9]+)?(s|m|h))+$` +akuity:test-kubebuilder-pattern=Duration |
| availabilityStrategy | [string](#string) | optional | AvailabilityStrategy specifies the semantics for how requested Freight is made available to the Stage. This field is optional. When left unspecified, the field is implicitly treated as if its value were "OneOf".

Accepted Values:

- "All": Freight must be verified and, if applicable, soaked in all upstream Stages to be considered available for promotion. - "OneOf": Freight must be verified and, if applicable, soaked in at least one upstream Stage to be considered available for promotion. - "": Treated the same as "OneOf".

+kubebuilder:validation:Optional |






<a name="github-com-akuity-kargo-api-v1alpha1-FreightStatus"></a>

### FreightStatus
FreightStatus describes a piece of Freight's most recently observed state.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| currentlyIn | [FreightStatus.CurrentlyInEntry](#github-com-akuity-kargo-api-v1alpha1-FreightStatus-CurrentlyInEntry) | repeated | CurrentlyIn describes the Stages in which this Freight is currently in use. |
| verifiedIn | [FreightStatus.VerifiedInEntry](#github-com-akuity-kargo-api-v1alpha1-FreightStatus-VerifiedInEntry) | repeated | VerifiedIn describes the Stages in which this Freight has been verified through promotion and subsequent health checks. |
| approvedFor | [FreightStatus.ApprovedForEntry](#github-com-akuity-kargo-api-v1alpha1-FreightStatus-ApprovedForEntry) | repeated | ApprovedFor describes the Stages for which this Freight has been approved preemptively/manually by a user. This is useful for hotfixes, where one might wish to promote a piece of Freight to a given Stage without transiting the entire pipeline. |
| metadata | [FreightStatus.MetadataEntry](#github-com-akuity-kargo-api-v1alpha1-FreightStatus-MetadataEntry) | repeated | Metadata is a map of arbitrary metadata associated with the Freight. This is useful for storing additional information about the Freight or Promotion that can be shared across steps or stages. |






<a name="github-com-akuity-kargo-api-v1alpha1-FreightStatus-ApprovedForEntry"></a>

### FreightStatus.ApprovedForEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [ApprovedStage](#github-com-akuity-kargo-api-v1alpha1-ApprovedStage) | optional |  |






<a name="github-com-akuity-kargo-api-v1alpha1-FreightStatus-CurrentlyInEntry"></a>

### FreightStatus.CurrentlyInEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [CurrentStage](#github-com-akuity-kargo-api-v1alpha1-CurrentStage) | optional |  |






<a name="github-com-akuity-kargo-api-v1alpha1-FreightStatus-MetadataEntry"></a>

### FreightStatus.MetadataEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [k8s.io.apiextensions_apiserver.pkg.apis.apiextensions.v1.JSON](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSON) | optional |  |






<a name="github-com-akuity-kargo-api-v1alpha1-FreightStatus-VerifiedInEntry"></a>

### FreightStatus.VerifiedInEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [VerifiedStage](#github-com-akuity-kargo-api-v1alpha1-VerifiedStage) | optional |  |






<a name="github-com-akuity-kargo-api-v1alpha1-GitCommit"></a>

### GitCommit
GitCommit describes a specific commit from a specific Git repository.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repoURL | [string](#string) | optional | RepoURL is the URL of a Git repository. |
| id | [string](#string) | optional | ID is the ID of a specific commit in the Git repository specified by RepoURL. |
| branch | [string](#string) | optional | Branch denotes the branch of the repository where this commit was found. |
| tag | [string](#string) | optional | Tag denotes a tag in the repository that matched selection criteria and resolved to this commit. |
| message | [string](#string) | optional | Message is the message associated with the commit. At present, this only contains the first line (subject) of the commit message. |
| author | [string](#string) | optional | Author is the author of the commit. |
| committer | [string](#string) | optional | Committer is the person who committed the commit. |






<a name="github-com-akuity-kargo-api-v1alpha1-GitDiscoveryResult"></a>

### GitDiscoveryResult
GitDiscoveryResult represents the result of a Git discovery operation for a
GitSubscription.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repoURL | [string](#string) | optional | RepoURL is the repository URL of the GitSubscription.

+kubebuilder:validation:MinLength=1 +kubebuilder:validation:Pattern=`(?:^(ssh|https?)://(?:([\w-]+)(:(.+))?@)?([\w-]+(?:\.[\w-]+)*)(?::(\d{1,5}))?(/.*)$)|(?:^([\w-]+)@([\w+]+(?:\.[\w-]+)*):(/?.*))` +akuity:test-kubebuilder-pattern=GitRepoURLPattern |
| commits | [DiscoveredCommit](#github-com-akuity-kargo-api-v1alpha1-DiscoveredCommit) | repeated | Commits is a list of commits discovered by the Warehouse for the GitSubscription. An empty list indicates that the discovery operation was successful, but no commits matching the GitSubscription criteria were found.

+optional |






<a name="github-com-akuity-kargo-api-v1alpha1-GitHubWebhookReceiverConfig"></a>

### GitHubWebhookReceiverConfig
GitHubWebhookReceiverConfig describes a webhook receiver that is compatible
with GitHub payloads.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| secretRef | [k8s.io.api.core.v1.LocalObjectReference](#k8s-io-api-core-v1-LocalObjectReference) | optional | SecretRef contains a reference to a Secret. For Project-scoped webhook receivers, the referenced Secret must be in the same namespace as the ProjectConfig.

For cluster-scoped webhook receivers, the referenced Secret must be in the designated "cluster Secrets" namespace.

The Secret's data map is expected to contain a `secret` key whose value is the shared secret used to authenticate the webhook requests sent by GitHub. For more information please refer to GitHub documentation: https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries

+kubebuilder:validation:Required |






<a name="github-com-akuity-kargo-api-v1alpha1-GitLabWebhookReceiverConfig"></a>

### GitLabWebhookReceiverConfig
GitLabWebhookReceiverConfig describes a webhook receiver that is compatible
with GitLab payloads.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| secretRef | [k8s.io.api.core.v1.LocalObjectReference](#k8s-io-api-core-v1-LocalObjectReference) | optional | SecretRef contains a reference to a Secret. For Project-scoped webhook receivers, the referenced Secret must be in the same namespace as the ProjectConfig.

For cluster-scoped webhook receivers, the referenced Secret must be in the designated "cluster Secrets" namespace.

The secret is expected to contain a `secret-token` key containing the shared secret specified when registering the webhook in GitLab. For more information about this token, please refer to the GitLab documentation: https://docs.gitlab.com/user/project/integrations/webhooks/

+kubebuilder:validation:Required |






<a name="github-com-akuity-kargo-api-v1alpha1-GitSubscription"></a>

### GitSubscription
GitSubscription defines a subscription to a Git repository.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repoURL | [string](#string) | optional | URL is the repository's URL. This is a required field.

+kubebuilder:validation:MinLength=1 +kubebuilder:validation:Pattern=`(?:^(ssh|https?)://(?:([\w-]+)(:(.+))?@)?([\w-]+(?:\.[\w-]+)*)(?::(\d{1,5}))?(/.*)$)|(?:^([\w-]+)@([\w+]+(?:\.[\w-]+)*):(/?.*))` +akuity:test-kubebuilder-pattern=GitRepoURLPattern |
| commitSelectionStrategy | [string](#string) | optional | CommitSelectionStrategy specifies the rules for how to identify the newest commit of interest in the repository specified by the RepoURL field. This field is optional. When left unspecified, the field is implicitly treated as if its value were "NewestFromBranch".

Accepted values:

- "NewestFromBranch": Selects the latest commit on the branch specified by the Branch field or the default branch if none is specified. This is the default strategy.

- "SemVer": Selects the commit referenced by the semantically greatest tag. The SemverConstraint field can optionally be used to narrow the set of tags eligible for selection.

- "Lexical": Selects the commit referenced by the lexicographically greatest tag. Useful when tags embed a _leading_ date or timestamp. The AllowTags and IgnoreTags fields can optionally be used to narrow the set of tags eligible for selection.

- "NewestTag": Selects the commit referenced by the most recently created tag. The AllowTags and IgnoreTags fields can optionally be used to narrow the set of tags eligible for selection.

+kubebuilder:default=NewestFromBranch |
| branch | [string](#string) | optional | Branch references a particular branch of the repository. The value in this field only has any effect when the CommitSelectionStrategy is NewestFromBranch or left unspecified (which is implicitly the same as NewestFromBranch). This field is optional. When left unspecified, (and the CommitSelectionStrategy is NewestFromBranch or unspecified), the subscription is implicitly to the repository's default branch.

+kubebuilder:validation:MinLength=1 +kubebuilder:validation:MaxLength=255 +kubebuilder:validation:Pattern=`^[a-zA-Z0-9]([a-zA-Z0-9._\/-]*[a-zA-Z0-9_-])?$` +akuity:test-kubebuilder-pattern=Branch |
| strictSemvers | [bool](#bool) | optional | StrictSemvers specifies whether only "strict" semver tags should be considered. A "strict" semver tag is one containing ALL of major, minor, and patch version components. This is enabled by default, but only has any effect when the CommitSelectionStrategy is SemVer. This should be disabled cautiously, as it creates the potential for any tag containing numeric characters only to be mistaken for a semver string containing the major version number only.

+kubebuilder:default=true |
| semverConstraint | [string](#string) | optional | SemverConstraint specifies constraints on what new tagged commits are considered in determining the newest commit of interest. The value in this field only has any effect when the CommitSelectionStrategy is SemVer. This field is optional. When left unspecified, there will be no constraints, which means the latest semantically tagged commit will always be used. Care should be taken with leaving this field unspecified, as it can lead to the unanticipated rollout of breaking changes.

+kubebuilder:validation:Optional |
| allowTags | [string](#string) | optional | AllowTags is a regular expression that can optionally be used to limit the tags that are considered in determining the newest commit of interest. The value in this field only has any effect when the CommitSelectionStrategy is Lexical, NewestTag, or SemVer. This field is optional.

+kubebuilder:validation:Optional |
| ignoreTags | [string](#string) | repeated | IgnoreTags is a list of tags that must be ignored when determining the newest commit of interest. No regular expressions or glob patterns are supported yet. The value in this field only has any effect when the CommitSelectionStrategy is Lexical, NewestTag, or SemVer. This field is optional.

+kubebuilder:validation:Optional |
| expressionFilter | [string](#string) | optional | ExpressionFilter is an expression that can optionally be used to limit the commits or tags that are considered in determining the newest commit of interest based on their metadata.

For commit-based strategies (NewestFromBranch), the filter applies to commits and has access to commit metadata variables. For tag-based strategies (Lexical, NewestTag, SemVer), the filter applies to tags and has access to tag metadata variables. The filter is applied after AllowTags, IgnoreTags, and SemverConstraint fields.

The expression should be a valid expr-lang expression that evaluates to true or false. When the expression evaluates to true, the commit/tag is included in the set that is considered. When the expression evaluates to false, the commit/tag is excluded.

Available variables depend on the CommitSelectionStrategy:

For NewestFromBranch (commit filtering): - `id`: The ID (sha) of the commit. - `commitDate`: The commit date of the commit. - `author`: The author of the commit message, in the format "Name <email>". - `committer`: The person who committed the commit, in the format 	 "Name <email>". - `subject`: The subject (first line) of the commit message.

For Lexical, NewestTag, SemVer (tag filtering): - `tag`: The name of the tag. - `id`: The ID (sha) of the commit associated with the tag. - `creatorDate`: The creation date of an annotated tag, or the commit 		date of a lightweight tag. - `author`: The author of the commit message associated with the tag, 	 in the format "Name <email>". - `committer`: The person who committed the commit associated with the 	 tag, in the format "Name <email>". - `subject`: The subject (first line) of the commit message associated 	 with the tag. 	 - `tagger`: The person who created the tag, in the format "Name <email>". 	 Only available for annotated tags. 	 - `annotation`: The subject (first line) of the tag annotation. Only 	 available for annotated tags.

Refer to the expr-lang documentation for more details on syntax and capabilities of the expression language: https://expr-lang.org.

+kubebuilder:validation:Optional |
| insecureSkipTLSVerify | [bool](#bool) | optional | InsecureSkipTLSVerify specifies whether certificate verification errors should be ignored when connecting to the repository. This should be enabled only with great caution. |
| includePaths | [string](#string) | repeated | IncludePaths is a list of selectors that designate paths in the repository that should trigger the production of new Freight when changes are detected therein. When specified, only changes in the identified paths will trigger Freight production. When not specified, changes in any path will trigger Freight production. Selectors may be defined using: 1. Exact paths to files or directories (ex. "charts/foo") 2. Glob patterns (prefix the pattern with "glob:"; ex. "glob:*.yaml") 3. Regular expressions (prefix the pattern with "regex:" or "regexp:"; ex. "regexp:^.*\.yaml$")

Paths selected by IncludePaths may be unselected by ExcludePaths. This is a useful method for including a broad set of paths and then excluding a subset of them. +kubebuilder:validation:Optional |
| excludePaths | [string](#string) | repeated | ExcludePaths is a list of selectors that designate paths in the repository that should NOT trigger the production of new Freight when changes are detected therein. When specified, changes in the identified paths will not trigger Freight production. When not specified, paths that should trigger Freight production will be defined solely by IncludePaths. Selectors may be defined using: 1. Exact paths to files or directories (ex. "charts/foo") 2. Glob patterns (prefix the pattern with "glob:"; ex. "glob:*.yaml") 3. Regular expressions (prefix the pattern with "regex:" or "regexp:"; ex. "regexp:^.*\.yaml$") Paths selected by IncludePaths may be unselected by ExcludePaths. This is a useful method for including a broad set of paths and then excluding a subset of them. +kubebuilder:validation:Optional |
| discoveryLimit | [int32](#int32) | optional | DiscoveryLimit is an optional limit on the number of commits that can be discovered for this subscription. The limit is applied after filtering commits based on the AllowTags and IgnoreTags fields. When left unspecified, the field is implicitly treated as if its value were "20". The upper limit for this field is 100.

+kubebuilder:validation:Minimum=1 +kubebuilder:validation:Maximum=100 +kubebuilder:default=20 |






<a name="github-com-akuity-kargo-api-v1alpha1-GiteaWebhookReceiverConfig"></a>

### GiteaWebhookReceiverConfig
GiteaWebhookReceiverConfig describes a webhook receiver that is compatible
with Gitea payloads.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| secretRef | [k8s.io.api.core.v1.LocalObjectReference](#k8s-io-api-core-v1-LocalObjectReference) | optional | SecretRef contains a reference to a Secret. For Project-scoped webhook receivers, the referenced Secret must be in the same namespace as the ProjectConfig.

For cluster-scoped webhook receivers, the referenced Secret must be in the designated "cluster Secrets" namespace.

The Secret's data map is expected to contain a `secret` key whose value is the shared secret used to authenticate the webhook requests sent by Gitea. For more information please refer to the Gitea documentation: https://docs.gitea.io/en-us/webhooks/

+kubebuilder:validation:Required |






<a name="github-com-akuity-kargo-api-v1alpha1-Health"></a>

### Health
Health describes the health of a Stage.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) | optional | Status describes the health of the Stage. |
| issues | [string](#string) | repeated | Issues clarifies why a Stage in any state other than Healthy is in that state. This field will always be the empty when a Stage is Healthy. |
| config | [k8s.io.apiextensions_apiserver.pkg.apis.apiextensions.v1.JSON](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSON) | optional | Config is the opaque configuration of all health checks performed on this Stage. |
| output | [k8s.io.apiextensions_apiserver.pkg.apis.apiextensions.v1.JSON](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSON) | optional | Output is the opaque output of all health checks performed on this Stage. |






<a name="github-com-akuity-kargo-api-v1alpha1-HealthCheckStep"></a>

### HealthCheckStep
HealthCheckStep describes a health check directive which can be executed by
a Stage to verify the health of a Promotion result.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| uses | [string](#string) | optional | Uses identifies a runner that can execute this step.

+kubebuilder:validation:MinLength=1 |
| config | [k8s.io.apiextensions_apiserver.pkg.apis.apiextensions.v1.JSON](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSON) | optional | Config is the configuration for the directive. |






<a name="github-com-akuity-kargo-api-v1alpha1-HealthStats"></a>

### HealthStats
HealthStats contains a summary of the collective health of some resource
type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| healthy | [int64](#int64) | optional | Healthy contains the number of resources that are explicitly healthy. |






<a name="github-com-akuity-kargo-api-v1alpha1-Image"></a>

### Image
Image describes a specific version of a container image.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repoURL | [string](#string) | optional | RepoURL describes the repository in which the image can be found. |
| tag | [string](#string) | optional | Tag identifies a specific version of the image in the repository specified by RepoURL. |
| digest | [string](#string) | optional | Digest identifies a specific version of the image in the repository specified by RepoURL. This is a more precise identifier than Tag. |
| annotations | [Image.AnnotationsEntry](#github-com-akuity-kargo-api-v1alpha1-Image-AnnotationsEntry) | repeated | Annotations is a map of arbitrary metadata for the image. |






<a name="github-com-akuity-kargo-api-v1alpha1-Image-AnnotationsEntry"></a>

### Image.AnnotationsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [string](#string) | optional |  |






<a name="github-com-akuity-kargo-api-v1alpha1-ImageDiscoveryResult"></a>

### ImageDiscoveryResult
ImageDiscoveryResult represents the result of an image discovery operation
for an ImageSubscription.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repoURL | [string](#string) | optional | RepoURL is the repository URL of the image, as specified in the ImageSubscription.

+kubebuilder:validation:MinLength=1 |
| platform | [string](#string) | optional | Platform is the target platform constraint of the ImageSubscription for which references were discovered. This field is optional, and only populated if the ImageSubscription specifies a Platform. |
| references | [DiscoveredImageReference](#github-com-akuity-kargo-api-v1alpha1-DiscoveredImageReference) | repeated | References is a list of image references discovered by the Warehouse for the ImageSubscription. An empty list indicates that the discovery operation was successful, but no images matching the ImageSubscription criteria were found.

+optional |






<a name="github-com-akuity-kargo-api-v1alpha1-ImageSubscription"></a>

### ImageSubscription
ImageSubscription defines a subscription to an image repository.

+kubebuilder:validation:XValidation:message="semverConstraint and constraint fields are mutually exclusive",rule="!(has(self.semverConstraint) && has(self.constraint))"
+kubebuilder:validation:XValidation:message="If imageSelectionStrategy is Digest, either constraint or semverConstraint must be set",rule="!(self.imageSelectionStrategy == 'Digest') || has(self.constraint) || has(self.semverConstraint)"


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repoURL | [string](#string) | optional | RepoURL specifies the URL of the image repository to subscribe to. The value in this field MUST NOT include an image tag. This field is required.

+kubebuilder:validation:MinLength=1 +kubebuilder:validation:Pattern=`^(\w+([\.-]\w+)*(:[\d]+)?/)?(\w+([\.-]\w+)*)(/\w+([\.-]\w+)*)*$` +akuity:test-kubebuilder-pattern=ImageRepoURL |
| imageSelectionStrategy | [string](#string) | optional | ImageSelectionStrategy specifies the rules for how to identify the newest version of the image specified by the RepoURL field. This field is optional. When left unspecified, the field is implicitly treated as if its value were "SemVer".

Accepted values:

- "Digest": Selects the image currently referenced by the tag specified (unintuitively) by the SemverConstraint field.

- "Lexical": Selects the image referenced by the lexicographically greatest tag. Useful when tags embed a leading date or timestamp. The AllowTags and IgnoreTags fields can optionally be used to narrow the set of tags eligible for selection.

- "NewestBuild": Selects the image that was most recently pushed to the repository. The AllowTags and IgnoreTags fields can optionally be used to narrow the set of tags eligible for selection. This is the least efficient and is likely to cause rate limiting affecting this Warehouse and possibly others. This strategy should be avoided.

- "SemVer": Selects the image with the semantically greatest tag. The AllowTags and IgnoreTags fields can optionally be used to narrow the set of tags eligible for selection.

+kubebuilder:default=SemVer |
| strictSemvers | [bool](#bool) | optional | StrictSemvers specifies whether only "strict" semver tags should be considered. A "strict" semver tag is one containing ALL of major, minor, and patch version components. This is enabled by default, but only has any effect when the ImageSelectionStrategy is SemVer. This should be disabled cautiously, as it is not uncommon to tag container images with short Git commit hashes, which have the potential to contain numeric characters only and could be mistaken for a semver string containing the major version number only.

+kubebuilder:default=true |
| semverConstraint | [string](#string) | optional | SemverConstraint specifies constraints on what new image versions are permissible. The value in this field only has any effect when the ImageSelectionStrategy is SemVer or left unspecified (which is implicitly the same as SemVer). This field is also optional. When left unspecified, (and the ImageSelectionStrategy is SemVer or unspecified), there will be no constraints, which means the latest semantically tagged version of an image will always be used. Care should be taken with leaving this field unspecified, as it can lead to the unanticipated rollout of breaking changes. More info: https://github.com/masterminds/semver#checking-version-constraints

Deprecated: Use Constraint instead. This field will be removed in v1.9.0

+kubebuilder:validation:Optional |
| constraint | [string](#string) | optional | Constraint specifies constraints on what new image versions are permissible. Acceptable values for this field vary contextually by ImageSelectionStrategy. The field is optional and is ignored by some strategies. When non-empty, the value in this field takes precedence over the value of the deprecated SemverConstraint field.

+kubebuilder:validation:Optional |
| allowTags | [string](#string) | optional | AllowTags is a regular expression that can optionally be used to limit the image tags that are considered in determining the newest version of an image. This field is optional.

+kubebuilder:validation:Optional |
| ignoreTags | [string](#string) | repeated | IgnoreTags is a list of tags that must be ignored when determining the newest version of an image. No regular expressions or glob patterns are supported yet. This field is optional.

+kubebuilder:validation:Optional |
| platform | [string](#string) | optional | Platform is a string of the form <os>/<arch> that limits the tags that can be considered when searching for new versions of an image. This field is optional. When left unspecified, it is implicitly equivalent to the OS/architecture of the Kargo controller. Care should be taken to set this value correctly in cases where the image referenced by this ImageRepositorySubscription will run on a Kubernetes node with a different OS/architecture than the Kargo controller. At present this is uncommon, but not unheard of.

+kubebuilder:validation:Optional |
| insecureSkipTLSVerify | [bool](#bool) | optional | InsecureSkipTLSVerify specifies whether certificate verification errors should be ignored when connecting to the repository. This should be enabled only with great caution. |
| discoveryLimit | [int32](#int32) | optional | DiscoveryLimit is an optional limit on the number of image references that can be discovered for this subscription. The limit is applied after filtering images based on the AllowTags and IgnoreTags fields. When left unspecified, the field is implicitly treated as if its value were "20". The upper limit for this field is 100.

+kubebuilder:validation:Minimum=1 +kubebuilder:validation:Maximum=100 +kubebuilder:default=20 |






<a name="github-com-akuity-kargo-api-v1alpha1-Project"></a>

### Project
Project is a resource type that reconciles to a specially labeled namespace
and other TODO: TBD project-level resources.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional |  |
| status | [ProjectStatus](#github-com-akuity-kargo-api-v1alpha1-ProjectStatus) | optional | Status describes the Project's current status. |






<a name="github-com-akuity-kargo-api-v1alpha1-ProjectConfig"></a>

### ProjectConfig
ProjectConfig is a resource type that describes the configuration of a
Project.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional |  |
| spec | [ProjectConfigSpec](#github-com-akuity-kargo-api-v1alpha1-ProjectConfigSpec) | optional | Spec describes the configuration of a Project. |
| status | [ProjectConfigStatus](#github-com-akuity-kargo-api-v1alpha1-ProjectConfigStatus) | optional | Status describes the current status of a ProjectConfig. |






<a name="github-com-akuity-kargo-api-v1alpha1-ProjectConfigList"></a>

### ProjectConfigList
ProjectConfigList is a list of ProjectConfig resources.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional |  |
| items | [ProjectConfig](#github-com-akuity-kargo-api-v1alpha1-ProjectConfig) | repeated |  |






<a name="github-com-akuity-kargo-api-v1alpha1-ProjectConfigSpec"></a>

### ProjectConfigSpec
ProjectConfigSpec describes the configuration of a Project.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| promotionPolicies | [PromotionPolicy](#github-com-akuity-kargo-api-v1alpha1-PromotionPolicy) | repeated | PromotionPolicies defines policies governing the promotion of Freight to specific Stages within the Project. |
| webhookReceivers | [WebhookReceiverConfig](#github-com-akuity-kargo-api-v1alpha1-WebhookReceiverConfig) | repeated | WebhookReceivers describes Project-specific webhook receivers used for processing events from various external platforms |






<a name="github-com-akuity-kargo-api-v1alpha1-ProjectConfigStatus"></a>

### ProjectConfigStatus
ProjectConfigStatus describes the current status of a ProjectConfig.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| conditions | [k8s.io.apimachinery.pkg.apis.meta.v1.Condition](#k8s-io-apimachinery-pkg-apis-meta-v1-Condition) | repeated | Conditions contains the last observations of the Project Config's current state.

+patchMergeKey=type +patchStrategy=merge +listType=map +listMapKey=type |
| observedGeneration | [int64](#int64) | optional | ObservedGeneration represents the .metadata.generation that this ProjectConfig was reconciled against. |
| lastHandledRefresh | [string](#string) | optional | LastHandledRefresh holds the value of the most recent AnnotationKeyRefresh annotation that was handled by the controller. This field can be used to determine whether the request to refresh the resource has been handled. +optional |
| webhookReceivers | [WebhookReceiverDetails](#github-com-akuity-kargo-api-v1alpha1-WebhookReceiverDetails) | repeated | WebhookReceivers describes the status of Project-specific webhook receivers. |






<a name="github-com-akuity-kargo-api-v1alpha1-ProjectList"></a>

### ProjectList
ProjectList is a list of Project resources.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional |  |
| items | [Project](#github-com-akuity-kargo-api-v1alpha1-Project) | repeated |  |






<a name="github-com-akuity-kargo-api-v1alpha1-ProjectStats"></a>

### ProjectStats
ProjectStats contains a summary of the collective state of a Project's
constituent resources.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| warehouses | [WarehouseStats](#github-com-akuity-kargo-api-v1alpha1-WarehouseStats) | optional | Warehouses contains a summary of the collective state of the Project's Warehouses. |
| stages | [StageStats](#github-com-akuity-kargo-api-v1alpha1-StageStats) | optional | Stages contains a summary of the collective state of the Project's Stages. |






<a name="github-com-akuity-kargo-api-v1alpha1-ProjectStatus"></a>

### ProjectStatus
ProjectStatus describes a Project's current status.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| conditions | [k8s.io.apimachinery.pkg.apis.meta.v1.Condition](#k8s-io-apimachinery-pkg-apis-meta-v1-Condition) | repeated | Conditions contains the last observations of the Project's current state. +patchMergeKey=type +patchStrategy=merge +listType=map +listMapKey=type |
| stats | [ProjectStats](#github-com-akuity-kargo-api-v1alpha1-ProjectStats) | optional | Stats contains a summary of the collective state of a Project's constituent resources. |






<a name="github-com-akuity-kargo-api-v1alpha1-Promotion"></a>

### Promotion
Promotion represents a request to transition a particular Stage into a
particular Freight.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional |  |
| spec | [PromotionSpec](#github-com-akuity-kargo-api-v1alpha1-PromotionSpec) | optional | Spec describes the desired transition of a specific Stage into a specific Freight.

+kubebuilder:validation:Required |
| status | [PromotionStatus](#github-com-akuity-kargo-api-v1alpha1-PromotionStatus) | optional | Status describes the current state of the transition represented by this Promotion. |






<a name="github-com-akuity-kargo-api-v1alpha1-PromotionList"></a>

### PromotionList
PromotionList contains a list of Promotion


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional |  |
| items | [Promotion](#github-com-akuity-kargo-api-v1alpha1-Promotion) | repeated |  |






<a name="github-com-akuity-kargo-api-v1alpha1-PromotionPolicy"></a>

### PromotionPolicy
PromotionPolicy defines policies governing the promotion of Freight to a
specific Stage.

+kubebuilder:validation:XValidation:message="PromotionPolicy must have exactly one of stage or stageSelector set",rule="has(self.stage) ? !has(self.stageSelector) : has(self.stageSelector)"


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| stage | [string](#string) | optional | Stage is the name of the Stage to which this policy applies.

Deprecated: Use StageSelector instead.

+kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$ |
| stageSelector | [PromotionPolicySelector](#github-com-akuity-kargo-api-v1alpha1-PromotionPolicySelector) | optional | StageSelector is a selector that matches the Stage resource to which this policy applies. |
| autoPromotionEnabled | [bool](#bool) | optional | AutoPromotionEnabled indicates whether new Freight can automatically be promoted into the Stage referenced by the Stage field. Note: There are may be other conditions also required for an auto-promotion to occur. This field defaults to false, but is commonly set to true for Stages that subscribe to Warehouses instead of other, upstream Stages. This allows users to define Stages that are automatically updated as soon as new artifacts are detected. |






<a name="github-com-akuity-kargo-api-v1alpha1-PromotionPolicySelector"></a>

### PromotionPolicySelector
PromotionPolicySelector is a selector that matches the resource to which
this policy applies. It can be used to match a specific resource by name or
to match a set of resources by label.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name is the name of the resource to which this policy applies.

It can be an exact name, a regex pattern (with prefix "regex:"), or a glob pattern (with prefix "glob:").

When both Name and LabelSelector are specified, the Name is ANDed with the LabelSelector. I.e., the resource must match both the Name and LabelSelector to be selected by this policy.

NOTE: Using a specific exact name is the most secure option. Pattern matching via regex or glob can be exploited by users with permissions to match promotion policies that weren't intended to apply to their resources. For example, a user could create a resource with a name deliberately crafted to match the pattern, potentially bypassing intended promotion controls.

+optional |
| labelSelector | [k8s.io.apimachinery.pkg.apis.meta.v1.LabelSelector](#k8s-io-apimachinery-pkg-apis-meta-v1-LabelSelector) | optional | LabelSelector is a selector that matches the resource to which this policy applies.

When both Name and LabelSelector are specified, the Name is ANDed with the LabelSelector. I.e., the resource must match both the Name and LabelSelector to be selected by this policy.

NOTE: Using label selectors introduces security risks as users with appropriate permissions could create new resources with labels that match the selector, potentially enabling unauthorized auto-promotion. For sensitive environments, exact Name matching provides tighter control. |






<a name="github-com-akuity-kargo-api-v1alpha1-PromotionReference"></a>

### PromotionReference
PromotionReference contains the relevant information about a Promotion
as observed by a Stage.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name is the name of the Promotion. |
| freight | [FreightReference](#github-com-akuity-kargo-api-v1alpha1-FreightReference) | optional | Freight is the freight being promoted. |
| status | [PromotionStatus](#github-com-akuity-kargo-api-v1alpha1-PromotionStatus) | optional | Status is the (optional) status of the Promotion. |
| finishedAt | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | FinishedAt is the time at which the Promotion was completed. |






<a name="github-com-akuity-kargo-api-v1alpha1-PromotionSpec"></a>

### PromotionSpec
PromotionSpec describes the desired transition of a specific Stage into a
specific Freight.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| stage | [string](#string) | optional | Stage specifies the name of the Stage to which this Promotion applies. The Stage referenced by this field MUST be in the same namespace as the Promotion.

+kubebuilder:validation:Required +kubebuilder:validation:MinLength=1 +kubebuilder:validation:MaxLength=253 +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$` +akuity:test-kubebuilder-pattern=KubernetesName |
| freight | [string](#string) | optional | Freight specifies the piece of Freight to be promoted into the Stage referenced by the Stage field.

+kubebuilder:validation:Required +kubebuilder:validation:MinLength=1 +kubebuilder:validation:MaxLength=253 +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$` +akuity:test-kubebuilder-pattern=KubernetesName |
| vars | [ExpressionVariable](#github-com-akuity-kargo-api-v1alpha1-ExpressionVariable) | repeated | Vars is a list of variables that can be referenced by expressions in promotion steps. |
| steps | [PromotionStep](#github-com-akuity-kargo-api-v1alpha1-PromotionStep) | repeated | Steps specifies the directives to be executed as part of this Promotion. The order in which the directives are executed is the order in which they are listed in this field.

+kubebuilder:validation:Required +kubebuilder:validation:MinItems=1 +kubebuilder:validation:items:XValidation:message="Promotion step must have uses set and must not reference a task",rule="has(self.uses) && !has(self.task)" |






<a name="github-com-akuity-kargo-api-v1alpha1-PromotionStatus"></a>

### PromotionStatus
PromotionStatus describes the current state of the transition represented by
a Promotion.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| lastHandledRefresh | [string](#string) | optional | LastHandledRefresh holds the value of the most recent AnnotationKeyRefresh annotation that was handled by the controller. This field can be used to determine whether the request to refresh the resource has been handled. +optional |
| phase | [string](#string) | optional | Phase describes where the Promotion currently is in its lifecycle. |
| message | [string](#string) | optional | Message is a display message about the promotion, including any errors preventing the Promotion controller from executing this Promotion. i.e. If the Phase field has a value of Failed, this field can be expected to explain why. |
| freight | [FreightReference](#github-com-akuity-kargo-api-v1alpha1-FreightReference) | optional | Freight is the detail of the piece of freight that was referenced by this promotion. |
| freightCollection | [FreightCollection](#github-com-akuity-kargo-api-v1alpha1-FreightCollection) | optional | FreightCollection contains the details of the piece of Freight referenced by this Promotion as well as any additional Freight that is carried over from the target Stage's current state. |
| healthChecks | [HealthCheckStep](#github-com-akuity-kargo-api-v1alpha1-HealthCheckStep) | repeated | HealthChecks contains the health check directives to be executed after the Promotion has completed. |
| startedAt | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | StartedAt is the time when the promotion started. |
| finishedAt | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | FinishedAt is the time when the promotion was completed. |
| currentStep | [int64](#int64) | optional | CurrentStep is the index of the current promotion step being executed. This permits steps that have already run successfully to be skipped on subsequent reconciliations attempts. |
| stepExecutionMetadata | [StepExecutionMetadata](#github-com-akuity-kargo-api-v1alpha1-StepExecutionMetadata) | repeated | StepExecutionMetadata tracks metadata pertaining to the execution of individual promotion steps. |
| state | [k8s.io.apiextensions_apiserver.pkg.apis.apiextensions.v1.JSON](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSON) | optional | State stores the state of the promotion process between reconciliation attempts. |






<a name="github-com-akuity-kargo-api-v1alpha1-PromotionStep"></a>

### PromotionStep
PromotionStep describes a directive to be executed as part of a Promotion.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| uses | [string](#string) | optional | Uses identifies a runner that can execute this step.

+kubebuilder:validation:Optional +kubebuilder:validation:MinLength=1 |
| task | [PromotionTaskReference](#github-com-akuity-kargo-api-v1alpha1-PromotionTaskReference) | optional | Task is a reference to a PromotionTask that should be inflated into a Promotion when it is built from a PromotionTemplate. |
| as | [string](#string) | optional | As is the alias this step can be referred to as. |
| if | [string](#string) | optional | If is an optional expression that, if present, must evaluate to a boolean value. If the expression evaluates to false, the step will be skipped. If the expression does not evaluate to a boolean value, the step will be considered to have failed. |
| continueOnError | [bool](#bool) | optional | ContinueOnError is a boolean value that, if set to true, will cause the Promotion to continue executing the next step even if this step fails. It also will not permit this failure to impact the overall status of the Promotion. |
| retry | [PromotionStepRetry](#github-com-akuity-kargo-api-v1alpha1-PromotionStepRetry) | optional | Retry is the retry policy for this step. |
| vars | [ExpressionVariable](#github-com-akuity-kargo-api-v1alpha1-ExpressionVariable) | repeated | Vars is a list of variables that can be referenced by expressions in the step's Config. The values override the values specified in the PromotionSpec. |
| config | [k8s.io.apiextensions_apiserver.pkg.apis.apiextensions.v1.JSON](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSON) | optional | Config is opaque configuration for the PromotionStep that is understood only by each PromotionStep's implementation. It is legal to utilize expressions in defining values at any level of this block. See https://docs.kargo.io/user-guide/reference-docs/expressions for details. |






<a name="github-com-akuity-kargo-api-v1alpha1-PromotionStepRetry"></a>

### PromotionStepRetry
PromotionStepRetry describes the retry policy for a PromotionStep.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| timeout | [k8s.io.apimachinery.pkg.apis.meta.v1.Duration](#k8s-io-apimachinery-pkg-apis-meta-v1-Duration) | optional | Timeout is the soft maximum interval in which a step that returns a Running status (which typically indicates it's waiting for something to happen) may be retried.

The maximum is a soft one because the check for whether the interval has elapsed occurs AFTER the step has run. This effectively means a step may run ONCE beyond the close of the interval.

If this field is set to nil, the effective default will be a step-specific one. If no step-specific default exists (i.e. is also nil), the effective default will be the system-wide default of 0.

A value of 0 will cause the step to be retried indefinitely unless the ErrorThreshold is reached. |
| errorThreshold | [uint32](#uint32) | optional | ErrorThreshold is the number of consecutive times the step must fail (for any reason) before retries are abandoned and the entire Promotion is marked as failed.

If this field is set to 0, the effective default will be a step-specific one. If no step-specific default exists (i.e. is also 0), the effective default will be the system-wide default of 1.

A value of 1 will cause the Promotion to be marked as failed after just a single failure; i.e. no retries will be attempted.

There is no option to specify an infinite number of retries using a value such as -1.

In a future release, Kargo is likely to become capable of distinguishing between recoverable and non-recoverable step failures. At that time, it is planned that unrecoverable failures will not be subject to this threshold and will immediately cause the Promotion to be marked as failed without further condition. |






<a name="github-com-akuity-kargo-api-v1alpha1-PromotionTask"></a>

### PromotionTask



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional |  |
| spec | [PromotionTaskSpec](#github-com-akuity-kargo-api-v1alpha1-PromotionTaskSpec) | optional | Spec describes the composition of a PromotionTask, including the variables available to the task and the steps.

+kubebuilder:validation:Required |






<a name="github-com-akuity-kargo-api-v1alpha1-PromotionTaskList"></a>

### PromotionTaskList
PromotionTaskList contains a list of PromotionTasks.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional |  |
| items | [PromotionTask](#github-com-akuity-kargo-api-v1alpha1-PromotionTask) | repeated |  |






<a name="github-com-akuity-kargo-api-v1alpha1-PromotionTaskReference"></a>

### PromotionTaskReference
PromotionTaskReference describes a reference to a PromotionTask.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name is the name of the (Cluster)PromotionTask.

+kubebuilder:validation:Required +kubebuilder:validation:MinLength=1 +kubebuilder:validation:MaxLength=253 +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$` +akuity:test-kubebuilder-pattern=KubernetesName |
| kind | [string](#string) | optional | Kind is the type of the PromotionTask. Can be either PromotionTask or ClusterPromotionTask, default is PromotionTask.

+kubebuilder:validation:Optional +kubebuilder:validation:Enum=PromotionTask;ClusterPromotionTask |






<a name="github-com-akuity-kargo-api-v1alpha1-PromotionTaskSpec"></a>

### PromotionTaskSpec



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vars | [ExpressionVariable](#github-com-akuity-kargo-api-v1alpha1-ExpressionVariable) | repeated | Vars specifies the variables available to the PromotionTask. The values of these variables are the default values that can be overridden by the step referencing the task. |
| steps | [PromotionStep](#github-com-akuity-kargo-api-v1alpha1-PromotionStep) | repeated | Steps specifies the directives to be executed as part of this PromotionTask. The steps as defined here are inflated into a Promotion when it is built from a PromotionTemplate.

+kubebuilder:validation:Required +kubebuilder:validation:MinItems=1 +kubebuilder:validation:items:XValidation:message="PromotionTask step must have uses set and must not reference another task",rule="has(self.uses) && !has(self.task)" |






<a name="github-com-akuity-kargo-api-v1alpha1-PromotionTemplate"></a>

### PromotionTemplate
PromotionTemplate defines a template for a Promotion that can be used to
incorporate Freight into a Stage.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| spec | [PromotionTemplateSpec](#github-com-akuity-kargo-api-v1alpha1-PromotionTemplateSpec) | optional |  |






<a name="github-com-akuity-kargo-api-v1alpha1-PromotionTemplateSpec"></a>

### PromotionTemplateSpec
PromotionTemplateSpec describes the (partial) specification of a Promotion
for a Stage. This is a template that can be used to create a Promotion for a
Stage.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vars | [ExpressionVariable](#github-com-akuity-kargo-api-v1alpha1-ExpressionVariable) | repeated | Vars is a list of variables that can be referenced by expressions in promotion steps. |
| steps | [PromotionStep](#github-com-akuity-kargo-api-v1alpha1-PromotionStep) | repeated | Steps specifies the directives to be executed as part of a Promotion. The order in which the directives are executed is the order in which they are listed in this field.

+kubebuilder:validation:MinItems=1 +kubebuilder:validation:items:XValidation:message="PromotionTemplate step must have exactly one of uses or task set",rule="(has(self.uses) ? !has(self.task) : has(self.task))" +kubebuilder:validation:items:XValidation:message="PromotionTemplate step referencing a task cannot set continueOnError",rule="!has(self.task) || !has(self.continueOnError)" +kubebuilder:validation:items:XValidation:message="PromotionTemplate step referencing a task cannot set retry",rule="!has(self.task) || !has(self.retry)" |






<a name="github-com-akuity-kargo-api-v1alpha1-QuayWebhookReceiverConfig"></a>

### QuayWebhookReceiverConfig
QuayWebhookReceiverConfig describes a webhook receiver that is compatible
with Quay.io payloads.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| secretRef | [k8s.io.api.core.v1.LocalObjectReference](#k8s-io-api-core-v1-LocalObjectReference) | optional | SecretRef contains a reference to a Secret. For Project-scoped webhook receivers, the referenced Secret must be in the same namespace as the ProjectConfig.

For cluster-scoped webhook receivers, the referenced Secret must be in the designated "cluster Secrets" namespace.

The Secret's data map is expected to contain a `secret` key whose value does NOT need to be shared directly with Quay when registering a webhook. It is used only by Kargo to create a complex, hard-to-guess URL, which implicitly serves as a shared secret. For more information about Quay webhooks, please refer to the Quay documentation: https://docs.quay.io/guides/notifications.html

+kubebuilder:validation:Required |






<a name="github-com-akuity-kargo-api-v1alpha1-RepoSubscription"></a>

### RepoSubscription
RepoSubscription describes a subscription to ONE OF a Git repository, a
container image repository, or a Helm chart repository.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| git | [GitSubscription](#github-com-akuity-kargo-api-v1alpha1-GitSubscription) | optional | Git describes a subscriptions to a Git repository. |
| image | [ImageSubscription](#github-com-akuity-kargo-api-v1alpha1-ImageSubscription) | optional | Image describes a subscription to container image repository. |
| chart | [ChartSubscription](#github-com-akuity-kargo-api-v1alpha1-ChartSubscription) | optional | Chart describes a subscription to a Helm chart repository. |






<a name="github-com-akuity-kargo-api-v1alpha1-Stage"></a>

### Stage
Stage is the Kargo API's main type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional |  |
| spec | [StageSpec](#github-com-akuity-kargo-api-v1alpha1-StageSpec) | optional | Spec describes sources of Freight used by the Stage and how to incorporate Freight into the Stage.

+kubebuilder:validation:Required |
| status | [StageStatus](#github-com-akuity-kargo-api-v1alpha1-StageStatus) | optional | Status describes the Stage's current and recent Freight, health, and more. |






<a name="github-com-akuity-kargo-api-v1alpha1-StageList"></a>

### StageList
StageList is a list of Stage resources.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional |  |
| items | [Stage](#github-com-akuity-kargo-api-v1alpha1-Stage) | repeated |  |






<a name="github-com-akuity-kargo-api-v1alpha1-StageSpec"></a>

### StageSpec
StageSpec describes the sources of Freight used by a Stage and how to
incorporate Freight into the Stage.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| shard | [string](#string) | optional | Shard is the name of the shard that this Stage belongs to. This is an optional field. If not specified, the Stage will belong to the default shard. A defaulting webhook will sync the value of the kargo.akuity.io/shard label with the value of this field. When this field is empty, the webhook will ensure that label is absent. |
| vars | [ExpressionVariable](#github-com-akuity-kargo-api-v1alpha1-ExpressionVariable) | repeated | Vars is a list of variables that can be referenced anywhere in the StageSpec that supports expressions. For example, the PromotionTemplate and arguments of the Verification. |
| requestedFreight | [FreightRequest](#github-com-akuity-kargo-api-v1alpha1-FreightRequest) | repeated | RequestedFreight expresses the Stage's need for certain pieces of Freight, each having originated from a particular Warehouse. This list must be non-empty. In the common case, a Stage will request Freight having originated from just one specific Warehouse. In advanced cases, requesting Freight from multiple Warehouses provides a method of advancing new artifacts of different types through parallel pipelines at different speeds. This can be useful, for instance, if a Stage is home to multiple microservices that are independently versioned.

+kubebuilder:validation:MinItems=1 |
| promotionTemplate | [PromotionTemplate](#github-com-akuity-kargo-api-v1alpha1-PromotionTemplate) | optional | PromotionTemplate describes how to incorporate Freight into the Stage using a Promotion. |
| verification | [Verification](#github-com-akuity-kargo-api-v1alpha1-Verification) | optional | Verification describes how to verify a Stage's current Freight is fit for promotion downstream. |






<a name="github-com-akuity-kargo-api-v1alpha1-StageStats"></a>

### StageStats
StageStats contains a summary of the collective state of a Project's
Stages.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| count | [int64](#int64) | optional | Count contains the total number of Stages in the Project. |
| health | [HealthStats](#github-com-akuity-kargo-api-v1alpha1-HealthStats) | optional | Health contains a summary of the collective health of a Project's Stages. |






<a name="github-com-akuity-kargo-api-v1alpha1-StageStatus"></a>

### StageStatus
StageStatus describes a Stages's current and recent Freight, health, and
more.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| conditions | [k8s.io.apimachinery.pkg.apis.meta.v1.Condition](#k8s-io-apimachinery-pkg-apis-meta-v1-Condition) | repeated | Conditions contains the last observations of the Stage's current state. +patchMergeKey=type +patchStrategy=merge +listType=map +listMapKey=type |
| lastHandledRefresh | [string](#string) | optional | LastHandledRefresh holds the value of the most recent AnnotationKeyRefresh annotation that was handled by the controller. This field can be used to determine whether the request to refresh the resource has been handled. +optional |
| freightHistory | [FreightCollection](#github-com-akuity-kargo-api-v1alpha1-FreightCollection) | repeated | FreightHistory is a list of recent Freight selections that were deployed to the Stage. By default, the last ten Freight selections are stored. The first item in the list is the most recent Freight selection and currently deployed to the Stage, subsequent items are older selections. |
| freightSummary | [string](#string) | optional | FreightSummary is human-readable text maintained by the controller that summarizes what Freight is currently deployed to the Stage. For Stages that request a single piece of Freight AND the request has been fulfilled, this field will simply contain the name of the Freight. For Stages that request a single piece of Freight AND the request has NOT been fulfilled, or for Stages that request multiple pieces of Freight, this field will contain a summary of fulfilled/requested Freight. The existence of this field is a workaround for kubectl limitations so that this complex but valuable information can be displayed in a column in response to `kubectl get stages`. |
| health | [Health](#github-com-akuity-kargo-api-v1alpha1-Health) | optional | Health is the Stage's last observed health. |
| observedGeneration | [int64](#int64) | optional | ObservedGeneration represents the .metadata.generation that this Stage status was reconciled against. |
| currentPromotion | [PromotionReference](#github-com-akuity-kargo-api-v1alpha1-PromotionReference) | optional | CurrentPromotion is a reference to the currently Running promotion. |
| lastPromotion | [PromotionReference](#github-com-akuity-kargo-api-v1alpha1-PromotionReference) | optional | LastPromotion is a reference to the last completed promotion. |
| autoPromotionEnabled | [bool](#bool) | optional | AutoPromotionEnabled indicates whether automatic promotion is enabled for the Stage based on the ProjectConfig. |






<a name="github-com-akuity-kargo-api-v1alpha1-StepExecutionMetadata"></a>

### StepExecutionMetadata
StepExecutionMetadata tracks metadata pertaining to the execution of
a promotion step.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| alias | [string](#string) | optional | Alias is the alias of the step. |
| startedAt | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | StartedAt is the time at which the first attempt to execute the step began. |
| finishedAt | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | FinishedAt is the time at which the final attempt to execute the step completed. |
| errorCount | [uint32](#uint32) | optional | ErrorCount tracks consecutive failed attempts to execute the step. |
| status | [string](#string) | optional | Status is the high-level outcome of the step. |
| message | [string](#string) | optional | Message is a display message about the step, including any errors. |
| continueOnError | [bool](#bool) | optional | ContinueOnError is a boolean value that, if set to true, will cause the Promotion to continue executing the next step even if this step fails. It also will not permit this failure to impact the overall status of the Promotion. |






<a name="github-com-akuity-kargo-api-v1alpha1-Verification"></a>

### Verification
Verification describes how to verify that a Promotion has been successful
using Argo Rollouts AnalysisTemplates.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| analysisTemplates | [AnalysisTemplateReference](#github-com-akuity-kargo-api-v1alpha1-AnalysisTemplateReference) | repeated | AnalysisTemplates is a list of AnalysisTemplates from which AnalysisRuns should be created to verify a Stage's current Freight is fit to be promoted downstream. |
| analysisRunMetadata | [AnalysisRunMetadata](#github-com-akuity-kargo-api-v1alpha1-AnalysisRunMetadata) | optional | AnalysisRunMetadata contains optional metadata that should be applied to all AnalysisRuns. |
| args | [AnalysisRunArgument](#github-com-akuity-kargo-api-v1alpha1-AnalysisRunArgument) | repeated | Args lists arguments that should be added to all AnalysisRuns. |






<a name="github-com-akuity-kargo-api-v1alpha1-VerificationInfo"></a>

### VerificationInfo
VerificationInfo contains the details of an instance of a Verification
process.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) | optional | ID is the identifier of the Verification process. |
| actor | [string](#string) | optional | Actor is the name of the entity that initiated or aborted the Verification process. |
| startTime | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | StartTime is the time at which the Verification process was started. |
| phase | [string](#string) | optional | Phase describes the current phase of the Verification process. Generally, this will be a reflection of the underlying AnalysisRun's phase, however, there are exceptions to this, such as in the case where an AnalysisRun cannot be launched successfully. |
| message | [string](#string) | optional | Message may contain additional information about why the verification process is in its current phase. |
| analysisRun | [AnalysisRunReference](#github-com-akuity-kargo-api-v1alpha1-AnalysisRunReference) | optional | AnalysisRun is a reference to the Argo Rollouts AnalysisRun that implements the Verification process. |
| finishTime | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | FinishTime is the time at which the Verification process finished. |






<a name="github-com-akuity-kargo-api-v1alpha1-VerifiedStage"></a>

### VerifiedStage
VerifiedStage describes a Stage in which Freight has been verified.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| verifiedAt | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | VerifiedAt is the time at which the Freight was verified in the Stage. |
| longestSoak | [k8s.io.apimachinery.pkg.apis.meta.v1.Duration](#k8s-io-apimachinery-pkg-apis-meta-v1-Duration) | optional | LongestCompletedSoak represents the longest definite time interval wherein the Freight was in CONTINUOUS use by the Stage. This value is updated as Freight EXITS the Stage. If the Freight is currently in use by the Stage, the time elapsed since the Freight ENTERED the Stage is its current soak time, which may exceed the value of this field. |






<a name="github-com-akuity-kargo-api-v1alpha1-Warehouse"></a>

### Warehouse
Warehouse is a source of Freight.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional |  |
| spec | [WarehouseSpec](#github-com-akuity-kargo-api-v1alpha1-WarehouseSpec) | optional | Spec describes sources of artifacts.

+kubebuilder:validation:Required |
| status | [WarehouseStatus](#github-com-akuity-kargo-api-v1alpha1-WarehouseStatus) | optional | Status describes the Warehouse's most recently observed state. |






<a name="github-com-akuity-kargo-api-v1alpha1-WarehouseList"></a>

### WarehouseList
WarehouseList is a list of Warehouse resources.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional |  |
| items | [Warehouse](#github-com-akuity-kargo-api-v1alpha1-Warehouse) | repeated |  |






<a name="github-com-akuity-kargo-api-v1alpha1-WarehouseSpec"></a>

### WarehouseSpec
WarehouseSpec describes sources of versioned artifacts to be included in
Freight produced by this Warehouse.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| shard | [string](#string) | optional | Shard is the name of the shard that this Warehouse belongs to. This is an optional field. If not specified, the Warehouse will belong to the default shard. A defaulting webhook will sync this field with the value of the kargo.akuity.io/shard label. When the shard label is not present or differs from the value of this field, the defaulting webhook will set the label to the value of this field. If the shard label is present and this field is empty, the defaulting webhook will set the value of this field to the value of the shard label. |
| interval | [k8s.io.apimachinery.pkg.apis.meta.v1.Duration](#k8s-io-apimachinery-pkg-apis-meta-v1-Duration) | optional | Interval is the reconciliation interval for this Warehouse. On each reconciliation, the Warehouse will discover new artifacts and optionally produce new Freight. This field is optional. When left unspecified, the field is implicitly treated as if its value were "5m0s".

+kubebuilder:validation:Type=string +kubebuilder:validation:Pattern=`^([0-9]+(\.[0-9]+)?(s|m|h))+$` +kubebuilder:default="5m0s" +akuity:test-kubebuilder-pattern=Duration |
| freightCreationPolicy | [string](#string) | optional | FreightCreationPolicy describes how Freight is created by this Warehouse. This field is optional. When left unspecified, the field is implicitly treated as if its value were "Automatic".

Accepted values:

- "Automatic": New Freight is created automatically when any new artifact is discovered. - "Manual": New Freight is never created automatically.

+kubebuilder:default=Automatic +kubebuilder:validation:Optional |
| subscriptions | [RepoSubscription](#github-com-akuity-kargo-api-v1alpha1-RepoSubscription) | repeated | Subscriptions describes sources of artifacts to be included in Freight produced by this Warehouse.

+kubebuilder:validation:MinItems=1 |






<a name="github-com-akuity-kargo-api-v1alpha1-WarehouseStats"></a>

### WarehouseStats
WarehouseStats contains a summary of the collective state of a Project's
Warehouses.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| count | [int64](#int64) | optional | Count contains the total number of Warehouses in the Project. |
| health | [HealthStats](#github-com-akuity-kargo-api-v1alpha1-HealthStats) | optional | Health contains a summary of the collective health of a Project's Warehouses. |






<a name="github-com-akuity-kargo-api-v1alpha1-WarehouseStatus"></a>

### WarehouseStatus
WarehouseStatus describes a Warehouse's most recently observed state.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| conditions | [k8s.io.apimachinery.pkg.apis.meta.v1.Condition](#k8s-io-apimachinery-pkg-apis-meta-v1-Condition) | repeated | Conditions contains the last observations of the Warehouse's current state. +patchMergeKey=type +patchStrategy=merge +listType=map +listMapKey=type |
| lastHandledRefresh | [string](#string) | optional | LastHandledRefresh holds the value of the most recent AnnotationKeyRefresh annotation that was handled by the controller. This field can be used to determine whether the request to refresh the resource has been handled. +optional |
| observedGeneration | [int64](#int64) | optional | ObservedGeneration represents the .metadata.generation that this Warehouse was reconciled against. |
| lastFreightID | [string](#string) | optional | LastFreightID is a reference to the system-assigned identifier (name) of the most recent Freight produced by the Warehouse. |
| discoveredArtifacts | [DiscoveredArtifacts](#github-com-akuity-kargo-api-v1alpha1-DiscoveredArtifacts) | optional | DiscoveredArtifacts holds the artifacts discovered by the Warehouse. |






<a name="github-com-akuity-kargo-api-v1alpha1-WebhookReceiverConfig"></a>

### WebhookReceiverConfig
WebhookReceiverConfig describes the configuration for a single webhook
receiver.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name is the name of the webhook receiver.

+kubebuilder:validation:Required +kubebuilder:validation:MinLength=1 +kubebuilder:validation:MaxLength=253 +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$` +akuity:test-kubebuilder-pattern=KubernetesName |
| bitbucket | [BitbucketWebhookReceiverConfig](#github-com-akuity-kargo-api-v1alpha1-BitbucketWebhookReceiverConfig) | optional | Bitbucket contains the configuration for a webhook receiver that is compatible with Bitbucket payloads. |
| dockerhub | [DockerHubWebhookReceiverConfig](#github-com-akuity-kargo-api-v1alpha1-DockerHubWebhookReceiverConfig) | optional | DockerHub contains the configuration for a webhook receiver that is compatible with DockerHub payloads. |
| github | [GitHubWebhookReceiverConfig](#github-com-akuity-kargo-api-v1alpha1-GitHubWebhookReceiverConfig) | optional | GitHub contains the configuration for a webhook receiver that is compatible with GitHub payloads. |
| gitlab | [GitLabWebhookReceiverConfig](#github-com-akuity-kargo-api-v1alpha1-GitLabWebhookReceiverConfig) | optional | GitLab contains the configuration for a webhook receiver that is compatible with GitLab payloads. |
| quay | [QuayWebhookReceiverConfig](#github-com-akuity-kargo-api-v1alpha1-QuayWebhookReceiverConfig) | optional | Quay contains the configuration for a webhook receiver that is compatible with Quay payloads. |
| artifactory | [ArtifactoryWebhookReceiverConfig](#github-com-akuity-kargo-api-v1alpha1-ArtifactoryWebhookReceiverConfig) | optional | Artifactory contains the configuration for a webhook receiver that is compatible with JFrog Artifactory payloads. |
| azure | [AzureWebhookReceiverConfig](#github-com-akuity-kargo-api-v1alpha1-AzureWebhookReceiverConfig) | optional | Azure contains the configuration for a webhook receiver that is compatible with Azure Container Registry (ACR) and Azure DevOps payloads. |
| gitea | [GiteaWebhookReceiverConfig](#github-com-akuity-kargo-api-v1alpha1-GiteaWebhookReceiverConfig) | optional | Gitea contains the configuration for a webhook receiver that is compatible with Gitea payloads. |






<a name="github-com-akuity-kargo-api-v1alpha1-WebhookReceiverDetails"></a>

### WebhookReceiverDetails
WebhookReceiverDetails encapsulates the details of a webhook receiver.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name is the name of the webhook receiver. |
| path | [string](#string) | optional | Path is the path to the receiver's webhook endpoint. |
| url | [string](#string) | optional | URL includes the full address of the receiver's webhook endpoint. |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="api_stubs_rollouts_v1alpha1_generated-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/stubs/rollouts/v1alpha1/generated.proto



<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisRun"></a>

### AnalysisRun



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional |  |
| spec | [AnalysisRunSpec](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisRunSpec) | optional |  |
| status | [AnalysisRunStatus](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisRunStatus) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisRunList"></a>

### AnalysisRunList



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional |  |
| items | [AnalysisRun](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisRun) | repeated |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisRunSpec"></a>

### AnalysisRunSpec



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metrics | [Metric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Metric) | repeated |  |
| args | [Argument](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Argument) | repeated |  |
| terminate | [bool](#bool) | optional |  |
| dryRun | [DryRun](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-DryRun) | repeated |  |
| measurementRetention | [MeasurementRetention](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-MeasurementRetention) | repeated |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisRunStatus"></a>

### AnalysisRunStatus



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| phase | [string](#string) | optional |  |
| message | [string](#string) | optional |  |
| metricResults | [MetricResult](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-MetricResult) | repeated |  |
| startedAt | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional |  |
| runSummary | [RunSummary](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-RunSummary) | optional |  |
| dryRunSummary | [RunSummary](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-RunSummary) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisTemplate"></a>

### AnalysisTemplate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional |  |
| spec | [AnalysisTemplateSpec](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisTemplateSpec) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisTemplateList"></a>

### AnalysisTemplateList



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional |  |
| items | [AnalysisTemplate](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisTemplate) | repeated |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisTemplateSpec"></a>

### AnalysisTemplateSpec



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metrics | [Metric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Metric) | repeated |  |
| args | [Argument](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Argument) | repeated |  |
| dryRun | [DryRun](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-DryRun) | repeated |  |
| measurementRetention | [MeasurementRetention](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-MeasurementRetention) | repeated |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Argument"></a>

### Argument



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional |  |
| value | [string](#string) | optional |  |
| valueFrom | [ValueFrom](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-ValueFrom) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Authentication"></a>

### Authentication



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sigv4 | [Sigv4Config](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Sigv4Config) | optional |  |
| oauth2 | [OAuth2Config](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-OAuth2Config) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-CloudWatchMetric"></a>

### CloudWatchMetric



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| interval | [string](#string) | optional |  |
| metricDataQueries | [CloudWatchMetricDataQuery](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-CloudWatchMetricDataQuery) | repeated |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-CloudWatchMetricDataQuery"></a>

### CloudWatchMetricDataQuery



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) | optional |  |
| expression | [string](#string) | optional |  |
| label | [string](#string) | optional |  |
| metricStat | [CloudWatchMetricStat](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-CloudWatchMetricStat) | optional |  |
| period | [k8s.io.apimachinery.pkg.util.intstr.IntOrString](#k8s-io-apimachinery-pkg-util-intstr-IntOrString) | optional |  |
| returnData | [bool](#bool) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-CloudWatchMetricStat"></a>

### CloudWatchMetricStat



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metric | [CloudWatchMetricStatMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-CloudWatchMetricStatMetric) | optional |  |
| period | [k8s.io.apimachinery.pkg.util.intstr.IntOrString](#k8s-io-apimachinery-pkg-util-intstr-IntOrString) | optional |  |
| stat | [string](#string) | optional |  |
| unit | [string](#string) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-CloudWatchMetricStatMetric"></a>

### CloudWatchMetricStatMetric



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| dimensions | [CloudWatchMetricStatMetricDimension](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-CloudWatchMetricStatMetricDimension) | repeated |  |
| metricName | [string](#string) | optional |  |
| namespace | [string](#string) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-CloudWatchMetricStatMetricDimension"></a>

### CloudWatchMetricStatMetricDimension



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional |  |
| value | [string](#string) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-ClusterAnalysisTemplate"></a>

### ClusterAnalysisTemplate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional |  |
| spec | [AnalysisTemplateSpec](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-AnalysisTemplateSpec) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-ClusterAnalysisTemplateList"></a>

### ClusterAnalysisTemplateList



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional |  |
| items | [ClusterAnalysisTemplate](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-ClusterAnalysisTemplate) | repeated |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-DatadogMetric"></a>

### DatadogMetric



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| interval | [string](#string) | optional |  |
| query | [string](#string) | optional |  |
| queries | [DatadogMetric.QueriesEntry](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-DatadogMetric-QueriesEntry) | repeated |  |
| formula | [string](#string) | optional |  |
| apiVersion | [string](#string) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-DatadogMetric-QueriesEntry"></a>

### DatadogMetric.QueriesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [string](#string) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-DryRun"></a>

### DryRun



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metricName | [string](#string) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-FieldRef"></a>

### FieldRef



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| fieldPath | [string](#string) | optional | Required: Path of the field to select in the specified API version |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-GraphiteMetric"></a>

### GraphiteMetric



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| address | [string](#string) | optional |  |
| query | [string](#string) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-InfluxdbMetric"></a>

### InfluxdbMetric



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile | [string](#string) | optional |  |
| query | [string](#string) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-JobMetric"></a>

### JobMetric



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional |  |
| spec | [k8s.io.api.batch.v1.JobSpec](#k8s-io-api-batch-v1-JobSpec) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-KayentaMetric"></a>

### KayentaMetric



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| address | [string](#string) | optional |  |
| application | [string](#string) | optional |  |
| canaryConfigName | [string](#string) | optional |  |
| metricsAccountName | [string](#string) | optional |  |
| configurationAccountName | [string](#string) | optional |  |
| storageAccountName | [string](#string) | optional |  |
| threshold | [KayentaThreshold](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-KayentaThreshold) | optional |  |
| scopes | [KayentaScope](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-KayentaScope) | repeated |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-KayentaScope"></a>

### KayentaScope



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional |  |
| controlScope | [ScopeDetail](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-ScopeDetail) | optional |  |
| experimentScope | [ScopeDetail](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-ScopeDetail) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-KayentaThreshold"></a>

### KayentaThreshold



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| pass | [int64](#int64) | optional |  |
| marginal | [int64](#int64) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Measurement"></a>

### Measurement



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| phase | [string](#string) | optional |  |
| message | [string](#string) | optional |  |
| startedAt | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional |  |
| finishedAt | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional |  |
| value | [string](#string) | optional |  |
| metadata | [Measurement.MetadataEntry](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Measurement-MetadataEntry) | repeated |  |
| resumeAt | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Measurement-MetadataEntry"></a>

### Measurement.MetadataEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [string](#string) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-MeasurementRetention"></a>

### MeasurementRetention



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metricName | [string](#string) | optional |  |
| limit | [int32](#int32) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Metric"></a>

### Metric



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional |  |
| interval | [string](#string) | optional |  |
| initialDelay | [string](#string) | optional |  |
| count | [k8s.io.apimachinery.pkg.util.intstr.IntOrString](#k8s-io-apimachinery-pkg-util-intstr-IntOrString) | optional |  |
| successCondition | [string](#string) | optional |  |
| failureCondition | [string](#string) | optional |  |
| failureLimit | [k8s.io.apimachinery.pkg.util.intstr.IntOrString](#k8s-io-apimachinery-pkg-util-intstr-IntOrString) | optional |  |
| inconclusiveLimit | [k8s.io.apimachinery.pkg.util.intstr.IntOrString](#k8s-io-apimachinery-pkg-util-intstr-IntOrString) | optional |  |
| consecutiveErrorLimit | [k8s.io.apimachinery.pkg.util.intstr.IntOrString](#k8s-io-apimachinery-pkg-util-intstr-IntOrString) | optional |  |
| provider | [MetricProvider](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-MetricProvider) | optional |  |
| consecutiveSuccessLimit | [k8s.io.apimachinery.pkg.util.intstr.IntOrString](#k8s-io-apimachinery-pkg-util-intstr-IntOrString) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-MetricProvider"></a>

### MetricProvider



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| prometheus | [PrometheusMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-PrometheusMetric) | optional |  |
| kayenta | [KayentaMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-KayentaMetric) | optional |  |
| web | [WebMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-WebMetric) | optional |  |
| datadog | [DatadogMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-DatadogMetric) | optional |  |
| wavefront | [WavefrontMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-WavefrontMetric) | optional |  |
| newRelic | [NewRelicMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-NewRelicMetric) | optional |  |
| job | [JobMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-JobMetric) | optional |  |
| cloudWatch | [CloudWatchMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-CloudWatchMetric) | optional |  |
| graphite | [GraphiteMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-GraphiteMetric) | optional |  |
| influxdb | [InfluxdbMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-InfluxdbMetric) | optional |  |
| skywalking | [SkyWalkingMetric](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-SkyWalkingMetric) | optional |  |
| plugin | [MetricProvider.PluginEntry](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-MetricProvider-PluginEntry) | repeated |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-MetricProvider-PluginEntry"></a>

### MetricProvider.PluginEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [bytes](#bytes) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-MetricResult"></a>

### MetricResult



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional |  |
| phase | [string](#string) | optional |  |
| measurements | [Measurement](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Measurement) | repeated |  |
| message | [string](#string) | optional |  |
| count | [int32](#int32) | optional |  |
| successful | [int32](#int32) | optional |  |
| failed | [int32](#int32) | optional |  |
| inconclusive | [int32](#int32) | optional |  |
| error | [int32](#int32) | optional |  |
| consecutiveError | [int32](#int32) | optional |  |
| dryRun | [bool](#bool) | optional |  |
| metadata | [MetricResult.MetadataEntry](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-MetricResult-MetadataEntry) | repeated |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-MetricResult-MetadataEntry"></a>

### MetricResult.MetadataEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [string](#string) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-NewRelicMetric"></a>

### NewRelicMetric



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile | [string](#string) | optional |  |
| query | [string](#string) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-OAuth2Config"></a>

### OAuth2Config



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tokenUrl | [string](#string) | optional |  |
| clientId | [string](#string) | optional |  |
| clientSecret | [string](#string) | optional |  |
| scopes | [string](#string) | repeated |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-PrometheusMetric"></a>

### PrometheusMetric



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| address | [string](#string) | optional |  |
| query | [string](#string) | optional |  |
| authentication | [Authentication](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Authentication) | optional |  |
| timeout | [int64](#int64) | optional |  |
| insecure | [bool](#bool) | optional |  |
| headers | [WebMetricHeader](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-WebMetricHeader) | repeated |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-RunSummary"></a>

### RunSummary



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| count | [int32](#int32) | optional |  |
| successful | [int32](#int32) | optional |  |
| failed | [int32](#int32) | optional |  |
| inconclusive | [int32](#int32) | optional |  |
| error | [int32](#int32) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-ScopeDetail"></a>

### ScopeDetail



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| scope | [string](#string) | optional |  |
| region | [string](#string) | optional |  |
| step | [int64](#int64) | optional |  |
| start | [string](#string) | optional |  |
| end | [string](#string) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-SecretKeyRef"></a>

### SecretKeyRef



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional |  |
| key | [string](#string) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Sigv4Config"></a>

### Sigv4Config



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| region | [string](#string) | optional |  |
| profile | [string](#string) | optional |  |
| roleArn | [string](#string) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-SkyWalkingMetric"></a>

### SkyWalkingMetric



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| address | [string](#string) | optional |  |
| query | [string](#string) | optional |  |
| interval | [string](#string) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-ValueFrom"></a>

### ValueFrom



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| secretKeyRef | [SecretKeyRef](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-SecretKeyRef) | optional |  |
| fieldRef | [FieldRef](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-FieldRef) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-WavefrontMetric"></a>

### WavefrontMetric



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| address | [string](#string) | optional |  |
| query | [string](#string) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-WebMetric"></a>

### WebMetric



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| method | [string](#string) | optional |  |
| url | [string](#string) | optional | URL is the address of the web metric |
| headers | [WebMetricHeader](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-WebMetricHeader) | repeated |  |
| body | [string](#string) | optional |  |
| timeoutSeconds | [int64](#int64) | optional |  |
| jsonPath | [string](#string) | optional |  |
| insecure | [bool](#bool) | optional |  |
| jsonBody | [bytes](#bytes) | optional |  |
| authentication | [Authentication](#github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-Authentication) | optional |  |






<a name="github-com-akuity-kargo-api-stubs-rollouts-v1alpha1-WebMetricHeader"></a>

### WebMetricHeader



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [string](#string) | optional |  |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="k8s-io_api_core_v1_generated-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## k8s.io/api/core/v1/generated.proto



<a name="k8s-io-api-core-v1-AWSElasticBlockStoreVolumeSource"></a>

### AWSElasticBlockStoreVolumeSource
Represents a Persistent Disk resource in AWS.

An AWS EBS disk must exist before mounting to a container. The disk
must also be in the same AWS zone as the kubelet. An AWS EBS disk
can only be mounted as read/write once. AWS EBS volumes support
ownership management and SELinux relabeling.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| volumeID | [string](#string) | optional | volumeID is unique ID of the persistent disk resource in AWS (Amazon EBS volume). More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore |
| fsType | [string](#string) | optional | fsType is the filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore TODO: how do we prevent errors in the filesystem from compromising the machine +optional |
| partition | [int32](#int32) | optional | partition is the partition in the volume that you want to mount. If omitted, the default is to mount by volume name. Examples: For volume /dev/sda1, you specify the partition as "1". Similarly, the volume partition for /dev/sda is "0" (or you can leave the property empty). +optional |
| readOnly | [bool](#bool) | optional | readOnly value true will force the readOnly setting in VolumeMounts. More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore +optional |






<a name="k8s-io-api-core-v1-Affinity"></a>

### Affinity
Affinity is a group of affinity scheduling rules.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| nodeAffinity | [NodeAffinity](#k8s-io-api-core-v1-NodeAffinity) | optional | Describes node affinity scheduling rules for the pod. +optional |
| podAffinity | [PodAffinity](#k8s-io-api-core-v1-PodAffinity) | optional | Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)). +optional |
| podAntiAffinity | [PodAntiAffinity](#k8s-io-api-core-v1-PodAntiAffinity) | optional | Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)). +optional |






<a name="k8s-io-api-core-v1-AppArmorProfile"></a>

### AppArmorProfile
AppArmorProfile defines a pod or container's AppArmor settings.
+union


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) | optional | type indicates which kind of AppArmor profile will be applied. Valid options are: Localhost - a profile pre-loaded on the node. RuntimeDefault - the container runtime's default profile. Unconfined - no AppArmor enforcement. +unionDiscriminator |
| localhostProfile | [string](#string) | optional | localhostProfile indicates a profile loaded on the node that should be used. The profile must be preconfigured on the node to work. Must match the loaded name of the profile. Must be set if and only if type is "Localhost". +optional |






<a name="k8s-io-api-core-v1-AttachedVolume"></a>

### AttachedVolume
AttachedVolume describes a volume attached to a node


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name of the attached volume |
| devicePath | [string](#string) | optional | DevicePath represents the device path where the volume should be available |






<a name="k8s-io-api-core-v1-AvoidPods"></a>

### AvoidPods
AvoidPods describes pods that should avoid this node. This is the value for a
Node annotation with key scheduler.alpha.kubernetes.io/preferAvoidPods and
will eventually become a field of NodeStatus.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| preferAvoidPods | [PreferAvoidPodsEntry](#k8s-io-api-core-v1-PreferAvoidPodsEntry) | repeated | Bounded-sized list of signatures of pods that should avoid this node, sorted in timestamp order from oldest to newest. Size of the slice is unspecified. +optional +listType=atomic |






<a name="k8s-io-api-core-v1-AzureDiskVolumeSource"></a>

### AzureDiskVolumeSource
AzureDisk represents an Azure Data Disk mount on the host and bind mount to the pod.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| diskName | [string](#string) | optional | diskName is the Name of the data disk in the blob storage |
| diskURI | [string](#string) | optional | diskURI is the URI of data disk in the blob storage |
| cachingMode | [string](#string) | optional | cachingMode is the Host Caching mode: None, Read Only, Read Write. +optional +default=ref(AzureDataDiskCachingReadWrite) |
| fsType | [string](#string) | optional | fsType is Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. +optional +default="ext4" |
| readOnly | [bool](#bool) | optional | readOnly Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. +optional +default=false |
| kind | [string](#string) | optional | kind expected values are Shared: multiple blob disks per storage account Dedicated: single blob disk per storage account Managed: azure managed data disk (only in managed availability set). defaults to shared +default=ref(AzureSharedBlobDisk) |






<a name="k8s-io-api-core-v1-AzureFilePersistentVolumeSource"></a>

### AzureFilePersistentVolumeSource
AzureFile represents an Azure File Service mount on the host and bind mount to the pod.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| secretName | [string](#string) | optional | secretName is the name of secret that contains Azure Storage Account Name and Key |
| shareName | [string](#string) | optional | shareName is the azure Share Name |
| readOnly | [bool](#bool) | optional | readOnly defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. +optional |
| secretNamespace | [string](#string) | optional | secretNamespace is the namespace of the secret that contains Azure Storage Account Name and Key default is the same as the Pod +optional |






<a name="k8s-io-api-core-v1-AzureFileVolumeSource"></a>

### AzureFileVolumeSource
AzureFile represents an Azure File Service mount on the host and bind mount to the pod.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| secretName | [string](#string) | optional | secretName is the name of secret that contains Azure Storage Account Name and Key |
| shareName | [string](#string) | optional | shareName is the azure share Name |
| readOnly | [bool](#bool) | optional | readOnly defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. +optional |






<a name="k8s-io-api-core-v1-Binding"></a>

### Binding
Binding ties one object to another; for example, a pod is bound to a node by a scheduler.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional | Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional |
| target | [ObjectReference](#k8s-io-api-core-v1-ObjectReference) | optional | The target object that you want to bind to the standard object. |






<a name="k8s-io-api-core-v1-CSIPersistentVolumeSource"></a>

### CSIPersistentVolumeSource
Represents storage that is managed by an external CSI volume driver


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| driver | [string](#string) | optional | driver is the name of the driver to use for this volume. Required. |
| volumeHandle | [string](#string) | optional | volumeHandle is the unique volume name returned by the CSI volume plugin’s CreateVolume to refer to the volume on all subsequent calls. Required. |
| readOnly | [bool](#bool) | optional | readOnly value to pass to ControllerPublishVolumeRequest. Defaults to false (read/write). +optional |
| fsType | [string](#string) | optional | fsType to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". +optional |
| volumeAttributes | [CSIPersistentVolumeSource.VolumeAttributesEntry](#k8s-io-api-core-v1-CSIPersistentVolumeSource-VolumeAttributesEntry) | repeated | volumeAttributes of the volume to publish. +optional |
| controllerPublishSecretRef | [SecretReference](#k8s-io-api-core-v1-SecretReference) | optional | controllerPublishSecretRef is a reference to the secret object containing sensitive information to pass to the CSI driver to complete the CSI ControllerPublishVolume and ControllerUnpublishVolume calls. This field is optional, and may be empty if no secret is required. If the secret object contains more than one secret, all secrets are passed. +optional |
| nodeStageSecretRef | [SecretReference](#k8s-io-api-core-v1-SecretReference) | optional | nodeStageSecretRef is a reference to the secret object containing sensitive information to pass to the CSI driver to complete the CSI NodeStageVolume and NodeStageVolume and NodeUnstageVolume calls. This field is optional, and may be empty if no secret is required. If the secret object contains more than one secret, all secrets are passed. +optional |
| nodePublishSecretRef | [SecretReference](#k8s-io-api-core-v1-SecretReference) | optional | nodePublishSecretRef is a reference to the secret object containing sensitive information to pass to the CSI driver to complete the CSI NodePublishVolume and NodeUnpublishVolume calls. This field is optional, and may be empty if no secret is required. If the secret object contains more than one secret, all secrets are passed. +optional |
| controllerExpandSecretRef | [SecretReference](#k8s-io-api-core-v1-SecretReference) | optional | controllerExpandSecretRef is a reference to the secret object containing sensitive information to pass to the CSI driver to complete the CSI ControllerExpandVolume call. This field is optional, and may be empty if no secret is required. If the secret object contains more than one secret, all secrets are passed. +optional |
| nodeExpandSecretRef | [SecretReference](#k8s-io-api-core-v1-SecretReference) | optional | nodeExpandSecretRef is a reference to the secret object containing sensitive information to pass to the CSI driver to complete the CSI NodeExpandVolume call. This field is optional, may be omitted if no secret is required. If the secret object contains more than one secret, all secrets are passed. +optional |






<a name="k8s-io-api-core-v1-CSIPersistentVolumeSource-VolumeAttributesEntry"></a>

### CSIPersistentVolumeSource.VolumeAttributesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [string](#string) | optional |  |






<a name="k8s-io-api-core-v1-CSIVolumeSource"></a>

### CSIVolumeSource
Represents a source location of a volume to mount, managed by an external CSI driver


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| driver | [string](#string) | optional | driver is the name of the CSI driver that handles this volume. Consult with your admin for the correct name as registered in the cluster. |
| readOnly | [bool](#bool) | optional | readOnly specifies a read-only configuration for the volume. Defaults to false (read/write). +optional |
| fsType | [string](#string) | optional | fsType to mount. Ex. "ext4", "xfs", "ntfs". If not provided, the empty value is passed to the associated CSI driver which will determine the default filesystem to apply. +optional |
| volumeAttributes | [CSIVolumeSource.VolumeAttributesEntry](#k8s-io-api-core-v1-CSIVolumeSource-VolumeAttributesEntry) | repeated | volumeAttributes stores driver-specific properties that are passed to the CSI driver. Consult your driver's documentation for supported values. +optional |
| nodePublishSecretRef | [LocalObjectReference](#k8s-io-api-core-v1-LocalObjectReference) | optional | nodePublishSecretRef is a reference to the secret object containing sensitive information to pass to the CSI driver to complete the CSI NodePublishVolume and NodeUnpublishVolume calls. This field is optional, and may be empty if no secret is required. If the secret object contains more than one secret, all secret references are passed. +optional |






<a name="k8s-io-api-core-v1-CSIVolumeSource-VolumeAttributesEntry"></a>

### CSIVolumeSource.VolumeAttributesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [string](#string) | optional |  |






<a name="k8s-io-api-core-v1-Capabilities"></a>

### Capabilities
Adds and removes POSIX capabilities from running containers.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| add | [string](#string) | repeated | Added capabilities +optional +listType=atomic |
| drop | [string](#string) | repeated | Removed capabilities +optional +listType=atomic |






<a name="k8s-io-api-core-v1-CephFSPersistentVolumeSource"></a>

### CephFSPersistentVolumeSource
Represents a Ceph Filesystem mount that lasts the lifetime of a pod
Cephfs volumes do not support ownership management or SELinux relabeling.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| monitors | [string](#string) | repeated | monitors is Required: Monitors is a collection of Ceph monitors More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it +listType=atomic |
| path | [string](#string) | optional | path is Optional: Used as the mounted root, rather than the full Ceph tree, default is / +optional |
| user | [string](#string) | optional | user is Optional: User is the rados user name, default is admin More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it +optional |
| secretFile | [string](#string) | optional | secretFile is Optional: SecretFile is the path to key ring for User, default is /etc/ceph/user.secret More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it +optional |
| secretRef | [SecretReference](#k8s-io-api-core-v1-SecretReference) | optional | secretRef is Optional: SecretRef is reference to the authentication secret for User, default is empty. More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it +optional |
| readOnly | [bool](#bool) | optional | readOnly is Optional: Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it +optional |






<a name="k8s-io-api-core-v1-CephFSVolumeSource"></a>

### CephFSVolumeSource
Represents a Ceph Filesystem mount that lasts the lifetime of a pod
Cephfs volumes do not support ownership management or SELinux relabeling.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| monitors | [string](#string) | repeated | monitors is Required: Monitors is a collection of Ceph monitors More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it +listType=atomic |
| path | [string](#string) | optional | path is Optional: Used as the mounted root, rather than the full Ceph tree, default is / +optional |
| user | [string](#string) | optional | user is optional: User is the rados user name, default is admin More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it +optional |
| secretFile | [string](#string) | optional | secretFile is Optional: SecretFile is the path to key ring for User, default is /etc/ceph/user.secret More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it +optional |
| secretRef | [LocalObjectReference](#k8s-io-api-core-v1-LocalObjectReference) | optional | secretRef is Optional: SecretRef is reference to the authentication secret for User, default is empty. More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it +optional |
| readOnly | [bool](#bool) | optional | readOnly is Optional: Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it +optional |






<a name="k8s-io-api-core-v1-CinderPersistentVolumeSource"></a>

### CinderPersistentVolumeSource
Represents a cinder volume resource in Openstack.
A Cinder volume must exist before mounting to a container.
The volume must also be in the same region as the kubelet.
Cinder volumes support ownership management and SELinux relabeling.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| volumeID | [string](#string) | optional | volumeID used to identify the volume in cinder. More info: https://examples.k8s.io/mysql-cinder-pd/README.md |
| fsType | [string](#string) | optional | fsType Filesystem type to mount. Must be a filesystem type supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. More info: https://examples.k8s.io/mysql-cinder-pd/README.md +optional |
| readOnly | [bool](#bool) | optional | readOnly is Optional: Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. More info: https://examples.k8s.io/mysql-cinder-pd/README.md +optional |
| secretRef | [SecretReference](#k8s-io-api-core-v1-SecretReference) | optional | secretRef is Optional: points to a secret object containing parameters used to connect to OpenStack. +optional |






<a name="k8s-io-api-core-v1-CinderVolumeSource"></a>

### CinderVolumeSource
Represents a cinder volume resource in Openstack.
A Cinder volume must exist before mounting to a container.
The volume must also be in the same region as the kubelet.
Cinder volumes support ownership management and SELinux relabeling.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| volumeID | [string](#string) | optional | volumeID used to identify the volume in cinder. More info: https://examples.k8s.io/mysql-cinder-pd/README.md |
| fsType | [string](#string) | optional | fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. More info: https://examples.k8s.io/mysql-cinder-pd/README.md +optional |
| readOnly | [bool](#bool) | optional | readOnly defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. More info: https://examples.k8s.io/mysql-cinder-pd/README.md +optional |
| secretRef | [LocalObjectReference](#k8s-io-api-core-v1-LocalObjectReference) | optional | secretRef is optional: points to a secret object containing parameters used to connect to OpenStack. +optional |






<a name="k8s-io-api-core-v1-ClientIPConfig"></a>

### ClientIPConfig
ClientIPConfig represents the configurations of Client IP based session affinity.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| timeoutSeconds | [int32](#int32) | optional | timeoutSeconds specifies the seconds of ClientIP type session sticky time. The value must be >0 && <=86400(for 1 day) if ServiceAffinity == "ClientIP". Default value is 10800(for 3 hours). +optional |






<a name="k8s-io-api-core-v1-ClusterTrustBundleProjection"></a>

### ClusterTrustBundleProjection
ClusterTrustBundleProjection describes how to select a set of
ClusterTrustBundle objects and project their contents into the pod
filesystem.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Select a single ClusterTrustBundle by object name. Mutually-exclusive with signerName and labelSelector. +optional |
| signerName | [string](#string) | optional | Select all ClusterTrustBundles that match this signer name. Mutually-exclusive with name. The contents of all selected ClusterTrustBundles will be unified and deduplicated. +optional |
| labelSelector | [k8s.io.apimachinery.pkg.apis.meta.v1.LabelSelector](#k8s-io-apimachinery-pkg-apis-meta-v1-LabelSelector) | optional | Select all ClusterTrustBundles that match this label selector. Only has effect if signerName is set. Mutually-exclusive with name. If unset, interpreted as "match nothing". If set but empty, interpreted as "match everything". +optional |
| optional | [bool](#bool) | optional | If true, don't block pod startup if the referenced ClusterTrustBundle(s) aren't available. If using name, then the named ClusterTrustBundle is allowed not to exist. If using signerName, then the combination of signerName and labelSelector is allowed to match zero ClusterTrustBundles. +optional |
| path | [string](#string) | optional | Relative path from the volume root to write the bundle. |






<a name="k8s-io-api-core-v1-ComponentCondition"></a>

### ComponentCondition
Information about the condition of a component.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) | optional | Type of condition for a component. Valid value: "Healthy" |
| status | [string](#string) | optional | Status of the condition for a component. Valid values for "Healthy": "True", "False", or "Unknown". |
| message | [string](#string) | optional | Message about the condition for a component. For example, information about a health check. +optional |
| error | [string](#string) | optional | Condition error code for a component. For example, a health check error code. +optional |






<a name="k8s-io-api-core-v1-ComponentStatus"></a>

### ComponentStatus
ComponentStatus (and ComponentStatusList) holds the cluster validation info.
Deprecated: This API is deprecated in v1.19+


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional | Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional |
| conditions | [ComponentCondition](#k8s-io-api-core-v1-ComponentCondition) | repeated | List of component conditions observed +optional +patchMergeKey=type +patchStrategy=merge +listType=map +listMapKey=type |






<a name="k8s-io-api-core-v1-ComponentStatusList"></a>

### ComponentStatusList
Status of all the conditions for the component as a list of ComponentStatus objects.
Deprecated: This API is deprecated in v1.19+


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional | Standard list metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds +optional |
| items | [ComponentStatus](#k8s-io-api-core-v1-ComponentStatus) | repeated | List of ComponentStatus objects. |






<a name="k8s-io-api-core-v1-ConfigMap"></a>

### ConfigMap
ConfigMap holds configuration data for pods to consume.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional | Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional |
| immutable | [bool](#bool) | optional | Immutable, if set to true, ensures that data stored in the ConfigMap cannot be updated (only object metadata can be modified). If not set to true, the field can be modified at any time. Defaulted to nil. +optional |
| data | [ConfigMap.DataEntry](#k8s-io-api-core-v1-ConfigMap-DataEntry) | repeated | Data contains the configuration data. Each key must consist of alphanumeric characters, '-', '_' or '.'. Values with non-UTF-8 byte sequences must use the BinaryData field. The keys stored in Data must not overlap with the keys in the BinaryData field, this is enforced during validation process. +optional |
| binaryData | [ConfigMap.BinaryDataEntry](#k8s-io-api-core-v1-ConfigMap-BinaryDataEntry) | repeated | BinaryData contains the binary data. Each key must consist of alphanumeric characters, '-', '_' or '.'. BinaryData can contain byte sequences that are not in the UTF-8 range. The keys stored in BinaryData must not overlap with the ones in the Data field, this is enforced during validation process. Using this field will require 1.10+ apiserver and kubelet. +optional |






<a name="k8s-io-api-core-v1-ConfigMap-BinaryDataEntry"></a>

### ConfigMap.BinaryDataEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [bytes](#bytes) | optional |  |






<a name="k8s-io-api-core-v1-ConfigMap-DataEntry"></a>

### ConfigMap.DataEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [string](#string) | optional |  |






<a name="k8s-io-api-core-v1-ConfigMapEnvSource"></a>

### ConfigMapEnvSource
ConfigMapEnvSource selects a ConfigMap to populate the environment
variables with.

The contents of the target ConfigMap's Data field will represent the
key-value pairs as environment variables.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| localObjectReference | [LocalObjectReference](#k8s-io-api-core-v1-LocalObjectReference) | optional | The ConfigMap to select from. |
| optional | [bool](#bool) | optional | Specify whether the ConfigMap must be defined +optional |






<a name="k8s-io-api-core-v1-ConfigMapKeySelector"></a>

### ConfigMapKeySelector
Selects a key from a ConfigMap.
+structType=atomic


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| localObjectReference | [LocalObjectReference](#k8s-io-api-core-v1-LocalObjectReference) | optional | The ConfigMap to select from. |
| key | [string](#string) | optional | The key to select. |
| optional | [bool](#bool) | optional | Specify whether the ConfigMap or its key must be defined +optional |






<a name="k8s-io-api-core-v1-ConfigMapList"></a>

### ConfigMapList
ConfigMapList is a resource containing a list of ConfigMap objects.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional | More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional |
| items | [ConfigMap](#k8s-io-api-core-v1-ConfigMap) | repeated | Items is the list of ConfigMaps. |






<a name="k8s-io-api-core-v1-ConfigMapNodeConfigSource"></a>

### ConfigMapNodeConfigSource
ConfigMapNodeConfigSource contains the information to reference a ConfigMap as a config source for the Node.
This API is deprecated since 1.22: https://git.k8s.io/enhancements/keps/sig-node/281-dynamic-kubelet-configuration


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| namespace | [string](#string) | optional | Namespace is the metadata.namespace of the referenced ConfigMap. This field is required in all cases. |
| name | [string](#string) | optional | Name is the metadata.name of the referenced ConfigMap. This field is required in all cases. |
| uid | [string](#string) | optional | UID is the metadata.UID of the referenced ConfigMap. This field is forbidden in Node.Spec, and required in Node.Status. +optional |
| resourceVersion | [string](#string) | optional | ResourceVersion is the metadata.ResourceVersion of the referenced ConfigMap. This field is forbidden in Node.Spec, and required in Node.Status. +optional |
| kubeletConfigKey | [string](#string) | optional | KubeletConfigKey declares which key of the referenced ConfigMap corresponds to the KubeletConfiguration structure This field is required in all cases. |






<a name="k8s-io-api-core-v1-ConfigMapProjection"></a>

### ConfigMapProjection
Adapts a ConfigMap into a projected volume.

The contents of the target ConfigMap's Data field will be presented in a
projected volume as files using the keys in the Data field as the file names,
unless the items element is populated with specific mappings of keys to paths.
Note that this is identical to a configmap volume source without the default
mode.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| localObjectReference | [LocalObjectReference](#k8s-io-api-core-v1-LocalObjectReference) | optional |  |
| items | [KeyToPath](#k8s-io-api-core-v1-KeyToPath) | repeated | items if unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the ConfigMap, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'. +optional +listType=atomic |
| optional | [bool](#bool) | optional | optional specify whether the ConfigMap or its keys must be defined +optional |






<a name="k8s-io-api-core-v1-ConfigMapVolumeSource"></a>

### ConfigMapVolumeSource
Adapts a ConfigMap into a volume.

The contents of the target ConfigMap's Data field will be presented in a
volume as files using the keys in the Data field as the file names, unless
the items element is populated with specific mappings of keys to paths.
ConfigMap volumes support ownership management and SELinux relabeling.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| localObjectReference | [LocalObjectReference](#k8s-io-api-core-v1-LocalObjectReference) | optional |  |
| items | [KeyToPath](#k8s-io-api-core-v1-KeyToPath) | repeated | items if unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the ConfigMap, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'. +optional +listType=atomic |
| defaultMode | [int32](#int32) | optional | defaultMode is optional: mode bits used to set permissions on created files by default. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. Defaults to 0644. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set. +optional |
| optional | [bool](#bool) | optional | optional specify whether the ConfigMap or its keys must be defined +optional |






<a name="k8s-io-api-core-v1-Container"></a>

### Container
A single application container that you want to run within a pod.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name of the container specified as a DNS_LABEL. Each container in a pod must have a unique name (DNS_LABEL). Cannot be updated. |
| image | [string](#string) | optional | Container image name. More info: https://kubernetes.io/docs/concepts/containers/images This field is optional to allow higher level config management to default or override container images in workload controllers like Deployments and StatefulSets. +optional |
| command | [string](#string) | repeated | Entrypoint array. Not executed within a shell. The container image's ENTRYPOINT is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell +optional +listType=atomic |
| args | [string](#string) | repeated | Arguments to the entrypoint. The container image's CMD is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell +optional +listType=atomic |
| workingDir | [string](#string) | optional | Container's working directory. If not specified, the container runtime's default will be used, which might be configured in the container image. Cannot be updated. +optional |
| ports | [ContainerPort](#k8s-io-api-core-v1-ContainerPort) | repeated | List of ports to expose from the container. Not specifying a port here DOES NOT prevent that port from being exposed. Any port which is listening on the default "0.0.0.0" address inside a container will be accessible from the network. Modifying this array with strategic merge patch may corrupt the data. For more information See https://github.com/kubernetes/kubernetes/issues/108255. Cannot be updated. +optional +patchMergeKey=containerPort +patchStrategy=merge +listType=map +listMapKey=containerPort +listMapKey=protocol |
| envFrom | [EnvFromSource](#k8s-io-api-core-v1-EnvFromSource) | repeated | List of sources to populate environment variables in the container. The keys defined within a source must be a C_IDENTIFIER. All invalid keys will be reported as an event when the container is starting. When a key exists in multiple sources, the value associated with the last source will take precedence. Values defined by an Env with a duplicate key will take precedence. Cannot be updated. +optional +listType=atomic |
| env | [EnvVar](#k8s-io-api-core-v1-EnvVar) | repeated | List of environment variables to set in the container. Cannot be updated. +optional +patchMergeKey=name +patchStrategy=merge +listType=map +listMapKey=name |
| resources | [ResourceRequirements](#k8s-io-api-core-v1-ResourceRequirements) | optional | Compute Resources required by this container. Cannot be updated. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/ +optional |
| resizePolicy | [ContainerResizePolicy](#k8s-io-api-core-v1-ContainerResizePolicy) | repeated | Resources resize policy for the container. +featureGate=InPlacePodVerticalScaling +optional +listType=atomic |
| restartPolicy | [string](#string) | optional | RestartPolicy defines the restart behavior of individual containers in a pod. This field may only be set for init containers, and the only allowed value is "Always". For non-init containers or when this field is not specified, the restart behavior is defined by the Pod's restart policy and the container type. Setting the RestartPolicy as "Always" for the init container will have the following effect: this init container will be continually restarted on exit until all regular containers have terminated. Once all regular containers have completed, all init containers with restartPolicy "Always" will be shut down. This lifecycle differs from normal init containers and is often referred to as a "sidecar" container. Although this init container still starts in the init container sequence, it does not wait for the container to complete before proceeding to the next init container. Instead, the next init container starts immediately after this init container is started, or after any startupProbe has successfully completed. +featureGate=SidecarContainers +optional |
| volumeMounts | [VolumeMount](#k8s-io-api-core-v1-VolumeMount) | repeated | Pod volumes to mount into the container's filesystem. Cannot be updated. +optional +patchMergeKey=mountPath +patchStrategy=merge +listType=map +listMapKey=mountPath |
| volumeDevices | [VolumeDevice](#k8s-io-api-core-v1-VolumeDevice) | repeated | volumeDevices is the list of block devices to be used by the container. +patchMergeKey=devicePath +patchStrategy=merge +listType=map +listMapKey=devicePath +optional |
| livenessProbe | [Probe](#k8s-io-api-core-v1-Probe) | optional | Periodic probe of container liveness. Container will be restarted if the probe fails. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes +optional |
| readinessProbe | [Probe](#k8s-io-api-core-v1-Probe) | optional | Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes +optional |
| startupProbe | [Probe](#k8s-io-api-core-v1-Probe) | optional | StartupProbe indicates that the Pod has successfully initialized. If specified, no other probes are executed until this completes successfully. If this probe fails, the Pod will be restarted, just as if the livenessProbe failed. This can be used to provide different probe parameters at the beginning of a Pod's lifecycle, when it might take a long time to load data or warm a cache, than during steady-state operation. This cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes +optional |
| lifecycle | [Lifecycle](#k8s-io-api-core-v1-Lifecycle) | optional | Actions that the management system should take in response to container lifecycle events. Cannot be updated. +optional |
| terminationMessagePath | [string](#string) | optional | Optional: Path at which the file to which the container's termination message will be written is mounted into the container's filesystem. Message written is intended to be brief final status, such as an assertion failure message. Will be truncated by the node if greater than 4096 bytes. The total message length across all containers will be limited to 12kb. Defaults to /dev/termination-log. Cannot be updated. +optional |
| terminationMessagePolicy | [string](#string) | optional | Indicate how the termination message should be populated. File will use the contents of terminationMessagePath to populate the container status message on both success and failure. FallbackToLogsOnError will use the last chunk of container log output if the termination message file is empty and the container exited with an error. The log output is limited to 2048 bytes or 80 lines, whichever is smaller. Defaults to File. Cannot be updated. +optional |
| imagePullPolicy | [string](#string) | optional | Image pull policy. One of Always, Never, IfNotPresent. Defaults to Always if :latest tag is specified, or IfNotPresent otherwise. Cannot be updated. More info: https://kubernetes.io/docs/concepts/containers/images#updating-images +optional |
| securityContext | [SecurityContext](#k8s-io-api-core-v1-SecurityContext) | optional | SecurityContext defines the security options the container should be run with. If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext. More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/ +optional |
| stdin | [bool](#bool) | optional | Whether this container should allocate a buffer for stdin in the container runtime. If this is not set, reads from stdin in the container will always result in EOF. Default is false. +optional |
| stdinOnce | [bool](#bool) | optional | Whether the container runtime should close the stdin channel after it has been opened by a single attach. When stdin is true the stdin stream will remain open across multiple attach sessions. If stdinOnce is set to true, stdin is opened on container start, is empty until the first client attaches to stdin, and then remains open and accepts data until the client disconnects, at which time stdin is closed and remains closed until the container is restarted. If this flag is false, a container processes that reads from stdin will never receive an EOF. Default is false +optional |
| tty | [bool](#bool) | optional | Whether this container should allocate a TTY for itself, also requires 'stdin' to be true. Default is false. +optional |






<a name="k8s-io-api-core-v1-ContainerImage"></a>

### ContainerImage
Describe a container image


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| names | [string](#string) | repeated | Names by which this image is known. e.g. ["kubernetes.example/hyperkube:v1.0.7", "cloud-vendor.registry.example/cloud-vendor/hyperkube:v1.0.7"] +optional +listType=atomic |
| sizeBytes | [int64](#int64) | optional | The size of the image in bytes. +optional |






<a name="k8s-io-api-core-v1-ContainerPort"></a>

### ContainerPort
ContainerPort represents a network port in a single container.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | If specified, this must be an IANA_SVC_NAME and unique within the pod. Each named port in a pod must have a unique name. Name for the port that can be referred to by services. +optional |
| hostPort | [int32](#int32) | optional | Number of port to expose on the host. If specified, this must be a valid port number, 0 < x < 65536. If HostNetwork is specified, this must match ContainerPort. Most containers do not need this. +optional |
| containerPort | [int32](#int32) | optional | Number of port to expose on the pod's IP address. This must be a valid port number, 0 < x < 65536. |
| protocol | [string](#string) | optional | Protocol for port. Must be UDP, TCP, or SCTP. Defaults to "TCP". +optional +default="TCP" |
| hostIP | [string](#string) | optional | What host IP to bind the external port to. +optional |






<a name="k8s-io-api-core-v1-ContainerResizePolicy"></a>

### ContainerResizePolicy
ContainerResizePolicy represents resource resize policy for the container.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resourceName | [string](#string) | optional | Name of the resource to which this resource resize policy applies. Supported values: cpu, memory. |
| restartPolicy | [string](#string) | optional | Restart policy to apply when specified resource is resized. If not specified, it defaults to NotRequired. |






<a name="k8s-io-api-core-v1-ContainerState"></a>

### ContainerState
ContainerState holds a possible state of container.
Only one of its members may be specified.
If none of them is specified, the default one is ContainerStateWaiting.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| waiting | [ContainerStateWaiting](#k8s-io-api-core-v1-ContainerStateWaiting) | optional | Details about a waiting container +optional |
| running | [ContainerStateRunning](#k8s-io-api-core-v1-ContainerStateRunning) | optional | Details about a running container +optional |
| terminated | [ContainerStateTerminated](#k8s-io-api-core-v1-ContainerStateTerminated) | optional | Details about a terminated container +optional |






<a name="k8s-io-api-core-v1-ContainerStateRunning"></a>

### ContainerStateRunning
ContainerStateRunning is a running state of a container.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| startedAt | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | Time at which the container was last (re-)started +optional |






<a name="k8s-io-api-core-v1-ContainerStateTerminated"></a>

### ContainerStateTerminated
ContainerStateTerminated is a terminated state of a container.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| exitCode | [int32](#int32) | optional | Exit status from the last termination of the container |
| signal | [int32](#int32) | optional | Signal from the last termination of the container +optional |
| reason | [string](#string) | optional | (brief) reason from the last termination of the container +optional |
| message | [string](#string) | optional | Message regarding the last termination of the container +optional |
| startedAt | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | Time at which previous execution of the container started +optional |
| finishedAt | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | Time at which the container last terminated +optional |
| containerID | [string](#string) | optional | Container's ID in the format '<type>://<container_id>' +optional |






<a name="k8s-io-api-core-v1-ContainerStateWaiting"></a>

### ContainerStateWaiting
ContainerStateWaiting is a waiting state of a container.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| reason | [string](#string) | optional | (brief) reason the container is not yet running. +optional |
| message | [string](#string) | optional | Message regarding why the container is not yet running. +optional |






<a name="k8s-io-api-core-v1-ContainerStatus"></a>

### ContainerStatus
ContainerStatus contains details for the current status of this container.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name is a DNS_LABEL representing the unique name of the container. Each container in a pod must have a unique name across all container types. Cannot be updated. |
| state | [ContainerState](#k8s-io-api-core-v1-ContainerState) | optional | State holds details about the container's current condition. +optional |
| lastState | [ContainerState](#k8s-io-api-core-v1-ContainerState) | optional | LastTerminationState holds the last termination state of the container to help debug container crashes and restarts. This field is not populated if the container is still running and RestartCount is 0. +optional |
| ready | [bool](#bool) | optional | Ready specifies whether the container is currently passing its readiness check. The value will change as readiness probes keep executing. If no readiness probes are specified, this field defaults to true once the container is fully started (see Started field).

The value is typically used to determine whether a container is ready to accept traffic. |
| restartCount | [int32](#int32) | optional | RestartCount holds the number of times the container has been restarted. Kubelet makes an effort to always increment the value, but there are cases when the state may be lost due to node restarts and then the value may be reset to 0. The value is never negative. |
| image | [string](#string) | optional | Image is the name of container image that the container is running. The container image may not match the image used in the PodSpec, as it may have been resolved by the runtime. More info: https://kubernetes.io/docs/concepts/containers/images. |
| imageID | [string](#string) | optional | ImageID is the image ID of the container's image. The image ID may not match the image ID of the image used in the PodSpec, as it may have been resolved by the runtime. |
| containerID | [string](#string) | optional | ContainerID is the ID of the container in the format '<type>://<container_id>'. Where type is a container runtime identifier, returned from Version call of CRI API (for example "containerd"). +optional |
| started | [bool](#bool) | optional | Started indicates whether the container has finished its postStart lifecycle hook and passed its startup probe. Initialized as false, becomes true after startupProbe is considered successful. Resets to false when the container is restarted, or if kubelet loses state temporarily. In both cases, startup probes will run again. Is always true when no startupProbe is defined and container is running and has passed the postStart lifecycle hook. The null value must be treated the same as false. +optional |
| allocatedResources | [ContainerStatus.AllocatedResourcesEntry](#k8s-io-api-core-v1-ContainerStatus-AllocatedResourcesEntry) | repeated | AllocatedResources represents the compute resources allocated for this container by the node. Kubelet sets this value to Container.Resources.Requests upon successful pod admission and after successfully admitting desired pod resize. +featureGate=InPlacePodVerticalScalingAllocatedStatus +optional |
| resources | [ResourceRequirements](#k8s-io-api-core-v1-ResourceRequirements) | optional | Resources represents the compute resource requests and limits that have been successfully enacted on the running container after it has been started or has been successfully resized. +featureGate=InPlacePodVerticalScaling +optional |
| volumeMounts | [VolumeMountStatus](#k8s-io-api-core-v1-VolumeMountStatus) | repeated | Status of volume mounts. +optional +patchMergeKey=mountPath +patchStrategy=merge +listType=map +listMapKey=mountPath +featureGate=RecursiveReadOnlyMounts |
| user | [ContainerUser](#k8s-io-api-core-v1-ContainerUser) | optional | User represents user identity information initially attached to the first process of the container +featureGate=SupplementalGroupsPolicy +optional |
| allocatedResourcesStatus | [ResourceStatus](#k8s-io-api-core-v1-ResourceStatus) | repeated | AllocatedResourcesStatus represents the status of various resources allocated for this Pod. +featureGate=ResourceHealthStatus +optional +patchMergeKey=name +patchStrategy=merge +listType=map +listMapKey=name |
| stopSignal | [string](#string) | optional | StopSignal reports the effective stop signal for this container +featureGate=ContainerStopSignals +optional |






<a name="k8s-io-api-core-v1-ContainerStatus-AllocatedResourcesEntry"></a>

### ContainerStatus.AllocatedResourcesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [k8s.io.apimachinery.pkg.api.resource.Quantity](#k8s-io-apimachinery-pkg-api-resource-Quantity) | optional |  |






<a name="k8s-io-api-core-v1-ContainerUser"></a>

### ContainerUser
ContainerUser represents user identity information


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| linux | [LinuxContainerUser](#k8s-io-api-core-v1-LinuxContainerUser) | optional | Linux holds user identity information initially attached to the first process of the containers in Linux. Note that the actual running identity can be changed if the process has enough privilege to do so. +optional |






<a name="k8s-io-api-core-v1-DaemonEndpoint"></a>

### DaemonEndpoint
DaemonEndpoint contains information about a single Daemon endpoint.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| Port | [int32](#int32) | optional | Port number of the given endpoint. |






<a name="k8s-io-api-core-v1-DownwardAPIProjection"></a>

### DownwardAPIProjection
Represents downward API info for projecting into a projected volume.
Note that this is identical to a downwardAPI volume source without the default
mode.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| items | [DownwardAPIVolumeFile](#k8s-io-api-core-v1-DownwardAPIVolumeFile) | repeated | Items is a list of DownwardAPIVolume file +optional +listType=atomic |






<a name="k8s-io-api-core-v1-DownwardAPIVolumeFile"></a>

### DownwardAPIVolumeFile
DownwardAPIVolumeFile represents information to create the file containing the pod field


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) | optional | Required: Path is the relative path name of the file to be created. Must not be absolute or contain the '..' path. Must be utf-8 encoded. The first item of the relative path must not start with '..' |
| fieldRef | [ObjectFieldSelector](#k8s-io-api-core-v1-ObjectFieldSelector) | optional | Required: Selects a field of the pod: only annotations, labels, name, namespace and uid are supported. +optional |
| resourceFieldRef | [ResourceFieldSelector](#k8s-io-api-core-v1-ResourceFieldSelector) | optional | Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported. +optional |
| mode | [int32](#int32) | optional | Optional: mode bits used to set permissions on this file, must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set. +optional |






<a name="k8s-io-api-core-v1-DownwardAPIVolumeSource"></a>

### DownwardAPIVolumeSource
DownwardAPIVolumeSource represents a volume containing downward API info.
Downward API volumes support ownership management and SELinux relabeling.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| items | [DownwardAPIVolumeFile](#k8s-io-api-core-v1-DownwardAPIVolumeFile) | repeated | Items is a list of downward API volume file +optional +listType=atomic |
| defaultMode | [int32](#int32) | optional | Optional: mode bits to use on created files by default. Must be a Optional: mode bits used to set permissions on created files by default. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. Defaults to 0644. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set. +optional |






<a name="k8s-io-api-core-v1-EmptyDirVolumeSource"></a>

### EmptyDirVolumeSource
Represents an empty directory for a pod.
Empty directory volumes support ownership management and SELinux relabeling.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| medium | [string](#string) | optional | medium represents what type of storage medium should back this directory. The default is "" which means to use the node's default medium. Must be an empty string (default) or Memory. More info: https://kubernetes.io/docs/concepts/storage/volumes#emptydir +optional |
| sizeLimit | [k8s.io.apimachinery.pkg.api.resource.Quantity](#k8s-io-apimachinery-pkg-api-resource-Quantity) | optional | sizeLimit is the total amount of local storage required for this EmptyDir volume. The size limit is also applicable for memory medium. The maximum usage on memory medium EmptyDir would be the minimum value between the SizeLimit specified here and the sum of memory limits of all containers in a pod. The default is nil which means that the limit is undefined. More info: https://kubernetes.io/docs/concepts/storage/volumes#emptydir +optional |






<a name="k8s-io-api-core-v1-EndpointAddress"></a>

### EndpointAddress
EndpointAddress is a tuple that describes single IP address.
Deprecated: This API is deprecated in v1.33+.
+structType=atomic


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ip | [string](#string) | optional | The IP of this endpoint. May not be loopback (127.0.0.0/8 or ::1), link-local (169.254.0.0/16 or fe80::/10), or link-local multicast (224.0.0.0/24 or ff02::/16). |
| hostname | [string](#string) | optional | The Hostname of this endpoint +optional |
| nodeName | [string](#string) | optional | Optional: Node hosting this endpoint. This can be used to determine endpoints local to a node. +optional |
| targetRef | [ObjectReference](#k8s-io-api-core-v1-ObjectReference) | optional | Reference to object providing the endpoint. +optional |






<a name="k8s-io-api-core-v1-EndpointPort"></a>

### EndpointPort
EndpointPort is a tuple that describes a single port.
Deprecated: This API is deprecated in v1.33+.
+structType=atomic


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | The name of this port. This must match the 'name' field in the corresponding ServicePort. Must be a DNS_LABEL. Optional only if one port is defined. +optional |
| port | [int32](#int32) | optional | The port number of the endpoint. |
| protocol | [string](#string) | optional | The IP protocol for this port. Must be UDP, TCP, or SCTP. Default is TCP. +optional |
| appProtocol | [string](#string) | optional | The application protocol for this port. This is used as a hint for implementations to offer richer behavior for protocols that they understand. This field follows standard Kubernetes label syntax. Valid values are either:

* Un-prefixed protocol names - reserved for IANA standard service names (as per RFC-6335 and https://www.iana.org/assignments/service-names).

* Kubernetes-defined prefixed names: * 'kubernetes.io/h2c' - HTTP/2 prior knowledge over cleartext as described in https://www.rfc-editor.org/rfc/rfc9113.html#name-starting-http-2-with-prior- * 'kubernetes.io/ws' - WebSocket over cleartext as described in https://www.rfc-editor.org/rfc/rfc6455 * 'kubernetes.io/wss' - WebSocket over TLS as described in https://www.rfc-editor.org/rfc/rfc6455

* Other protocols should use implementation-defined prefixed names such as mycompany.com/my-custom-protocol. +optional |






<a name="k8s-io-api-core-v1-EndpointSubset"></a>

### EndpointSubset
EndpointSubset is a group of addresses with a common set of ports. The
expanded set of endpoints is the Cartesian product of Addresses x Ports.
For example, given:

	{
	  Addresses: [{"ip": "10.10.1.1"}, {"ip": "10.10.2.2"}],
	  Ports:     [{"name": "a", "port": 8675}, {"name": "b", "port": 309}]
	}

The resulting set of endpoints can be viewed as:

	a: [ 10.10.1.1:8675, 10.10.2.2:8675 ],
	b: [ 10.10.1.1:309, 10.10.2.2:309 ]

Deprecated: This API is deprecated in v1.33+.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| addresses | [EndpointAddress](#k8s-io-api-core-v1-EndpointAddress) | repeated | IP addresses which offer the related ports that are marked as ready. These endpoints should be considered safe for load balancers and clients to utilize. +optional +listType=atomic |
| notReadyAddresses | [EndpointAddress](#k8s-io-api-core-v1-EndpointAddress) | repeated | IP addresses which offer the related ports but are not currently marked as ready because they have not yet finished starting, have recently failed a readiness check, or have recently failed a liveness check. +optional +listType=atomic |
| ports | [EndpointPort](#k8s-io-api-core-v1-EndpointPort) | repeated | Port numbers available on the related IP addresses. +optional +listType=atomic |






<a name="k8s-io-api-core-v1-Endpoints"></a>

### Endpoints
Endpoints is a collection of endpoints that implement the actual service. Example:

	 Name: "mysvc",
	 Subsets: [
	   {
	     Addresses: [{"ip": "10.10.1.1"}, {"ip": "10.10.2.2"}],
	     Ports: [{"name": "a", "port": 8675}, {"name": "b", "port": 309}]
	   },
	   {
	     Addresses: [{"ip": "10.10.3.3"}],
	     Ports: [{"name": "a", "port": 93}, {"name": "b", "port": 76}]
	   },
	]

Endpoints is a legacy API and does not contain information about all Service features.
Use discoveryv1.EndpointSlice for complete information about Service endpoints.

Deprecated: This API is deprecated in v1.33+. Use discoveryv1.EndpointSlice.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional | Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional |
| subsets | [EndpointSubset](#k8s-io-api-core-v1-EndpointSubset) | repeated | The set of all endpoints is the union of all subsets. Addresses are placed into subsets according to the IPs they share. A single address with multiple ports, some of which are ready and some of which are not (because they come from different containers) will result in the address being displayed in different subsets for the different ports. No address will appear in both Addresses and NotReadyAddresses in the same subset. Sets of addresses and ports that comprise a service. +optional +listType=atomic |






<a name="k8s-io-api-core-v1-EndpointsList"></a>

### EndpointsList
EndpointsList is a list of endpoints.
Deprecated: This API is deprecated in v1.33+.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional | Standard list metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds +optional |
| items | [Endpoints](#k8s-io-api-core-v1-Endpoints) | repeated | List of endpoints. |






<a name="k8s-io-api-core-v1-EnvFromSource"></a>

### EnvFromSource
EnvFromSource represents the source of a set of ConfigMaps or Secrets


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| prefix | [string](#string) | optional | Optional text to prepend to the name of each environment variable. Must be a C_IDENTIFIER. +optional |
| configMapRef | [ConfigMapEnvSource](#k8s-io-api-core-v1-ConfigMapEnvSource) | optional | The ConfigMap to select from +optional |
| secretRef | [SecretEnvSource](#k8s-io-api-core-v1-SecretEnvSource) | optional | The Secret to select from +optional |






<a name="k8s-io-api-core-v1-EnvVar"></a>

### EnvVar
EnvVar represents an environment variable present in a Container.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name of the environment variable. Must be a C_IDENTIFIER. |
| value | [string](#string) | optional | Variable references $(VAR_NAME) are expanded using the previously defined environment variables in the container and any service environment variables. If a variable cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless of whether the variable exists or not. Defaults to "". +optional |
| valueFrom | [EnvVarSource](#k8s-io-api-core-v1-EnvVarSource) | optional | Source for the environment variable's value. Cannot be used if value is not empty. +optional |






<a name="k8s-io-api-core-v1-EnvVarSource"></a>

### EnvVarSource
EnvVarSource represents a source for the value of an EnvVar.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| fieldRef | [ObjectFieldSelector](#k8s-io-api-core-v1-ObjectFieldSelector) | optional | Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.podIP, status.podIPs. +optional |
| resourceFieldRef | [ResourceFieldSelector](#k8s-io-api-core-v1-ResourceFieldSelector) | optional | Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.ephemeral-storage) are currently supported. +optional |
| configMapKeyRef | [ConfigMapKeySelector](#k8s-io-api-core-v1-ConfigMapKeySelector) | optional | Selects a key of a ConfigMap. +optional |
| secretKeyRef | [SecretKeySelector](#k8s-io-api-core-v1-SecretKeySelector) | optional | Selects a key of a secret in the pod's namespace +optional |






<a name="k8s-io-api-core-v1-EphemeralContainer"></a>

### EphemeralContainer
An EphemeralContainer is a temporary container that you may add to an existing Pod for
user-initiated activities such as debugging. Ephemeral containers have no resource or
scheduling guarantees, and they will not be restarted when they exit or when a Pod is
removed or restarted. The kubelet may evict a Pod if an ephemeral container causes the
Pod to exceed its resource allocation.

To add an ephemeral container, use the ephemeralcontainers subresource of an existing
Pod. Ephemeral containers may not be removed or restarted.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ephemeralContainerCommon | [EphemeralContainerCommon](#k8s-io-api-core-v1-EphemeralContainerCommon) | optional | Ephemeral containers have all of the fields of Container, plus additional fields specific to ephemeral containers. Fields in common with Container are in the following inlined struct so than an EphemeralContainer may easily be converted to a Container. |
| targetContainerName | [string](#string) | optional | If set, the name of the container from PodSpec that this ephemeral container targets. The ephemeral container will be run in the namespaces (IPC, PID, etc) of this container. If not set then the ephemeral container uses the namespaces configured in the Pod spec.

The container runtime must implement support for this feature. If the runtime does not support namespace targeting then the result of setting this field is undefined. +optional |






<a name="k8s-io-api-core-v1-EphemeralContainerCommon"></a>

### EphemeralContainerCommon
EphemeralContainerCommon is a copy of all fields in Container to be inlined in
EphemeralContainer. This separate type allows easy conversion from EphemeralContainer
to Container and allows separate documentation for the fields of EphemeralContainer.
When a new field is added to Container it must be added here as well.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name of the ephemeral container specified as a DNS_LABEL. This name must be unique among all containers, init containers and ephemeral containers. |
| image | [string](#string) | optional | Container image name. More info: https://kubernetes.io/docs/concepts/containers/images |
| command | [string](#string) | repeated | Entrypoint array. Not executed within a shell. The image's ENTRYPOINT is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell +optional +listType=atomic |
| args | [string](#string) | repeated | Arguments to the entrypoint. The image's CMD is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell +optional +listType=atomic |
| workingDir | [string](#string) | optional | Container's working directory. If not specified, the container runtime's default will be used, which might be configured in the container image. Cannot be updated. +optional |
| ports | [ContainerPort](#k8s-io-api-core-v1-ContainerPort) | repeated | Ports are not allowed for ephemeral containers. +optional +patchMergeKey=containerPort +patchStrategy=merge +listType=map +listMapKey=containerPort +listMapKey=protocol |
| envFrom | [EnvFromSource](#k8s-io-api-core-v1-EnvFromSource) | repeated | List of sources to populate environment variables in the container. The keys defined within a source must be a C_IDENTIFIER. All invalid keys will be reported as an event when the container is starting. When a key exists in multiple sources, the value associated with the last source will take precedence. Values defined by an Env with a duplicate key will take precedence. Cannot be updated. +optional +listType=atomic |
| env | [EnvVar](#k8s-io-api-core-v1-EnvVar) | repeated | List of environment variables to set in the container. Cannot be updated. +optional +patchMergeKey=name +patchStrategy=merge +listType=map +listMapKey=name |
| resources | [ResourceRequirements](#k8s-io-api-core-v1-ResourceRequirements) | optional | Resources are not allowed for ephemeral containers. Ephemeral containers use spare resources already allocated to the pod. +optional |
| resizePolicy | [ContainerResizePolicy](#k8s-io-api-core-v1-ContainerResizePolicy) | repeated | Resources resize policy for the container. +featureGate=InPlacePodVerticalScaling +optional +listType=atomic |
| restartPolicy | [string](#string) | optional | Restart policy for the container to manage the restart behavior of each container within a pod. This may only be set for init containers. You cannot set this field on ephemeral containers. +featureGate=SidecarContainers +optional |
| volumeMounts | [VolumeMount](#k8s-io-api-core-v1-VolumeMount) | repeated | Pod volumes to mount into the container's filesystem. Subpath mounts are not allowed for ephemeral containers. Cannot be updated. +optional +patchMergeKey=mountPath +patchStrategy=merge +listType=map +listMapKey=mountPath |
| volumeDevices | [VolumeDevice](#k8s-io-api-core-v1-VolumeDevice) | repeated | volumeDevices is the list of block devices to be used by the container. +patchMergeKey=devicePath +patchStrategy=merge +listType=map +listMapKey=devicePath +optional |
| livenessProbe | [Probe](#k8s-io-api-core-v1-Probe) | optional | Probes are not allowed for ephemeral containers. +optional |
| readinessProbe | [Probe](#k8s-io-api-core-v1-Probe) | optional | Probes are not allowed for ephemeral containers. +optional |
| startupProbe | [Probe](#k8s-io-api-core-v1-Probe) | optional | Probes are not allowed for ephemeral containers. +optional |
| lifecycle | [Lifecycle](#k8s-io-api-core-v1-Lifecycle) | optional | Lifecycle is not allowed for ephemeral containers. +optional |
| terminationMessagePath | [string](#string) | optional | Optional: Path at which the file to which the container's termination message will be written is mounted into the container's filesystem. Message written is intended to be brief final status, such as an assertion failure message. Will be truncated by the node if greater than 4096 bytes. The total message length across all containers will be limited to 12kb. Defaults to /dev/termination-log. Cannot be updated. +optional |
| terminationMessagePolicy | [string](#string) | optional | Indicate how the termination message should be populated. File will use the contents of terminationMessagePath to populate the container status message on both success and failure. FallbackToLogsOnError will use the last chunk of container log output if the termination message file is empty and the container exited with an error. The log output is limited to 2048 bytes or 80 lines, whichever is smaller. Defaults to File. Cannot be updated. +optional |
| imagePullPolicy | [string](#string) | optional | Image pull policy. One of Always, Never, IfNotPresent. Defaults to Always if :latest tag is specified, or IfNotPresent otherwise. Cannot be updated. More info: https://kubernetes.io/docs/concepts/containers/images#updating-images +optional |
| securityContext | [SecurityContext](#k8s-io-api-core-v1-SecurityContext) | optional | Optional: SecurityContext defines the security options the ephemeral container should be run with. If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext. +optional |
| stdin | [bool](#bool) | optional | Whether this container should allocate a buffer for stdin in the container runtime. If this is not set, reads from stdin in the container will always result in EOF. Default is false. +optional |
| stdinOnce | [bool](#bool) | optional | Whether the container runtime should close the stdin channel after it has been opened by a single attach. When stdin is true the stdin stream will remain open across multiple attach sessions. If stdinOnce is set to true, stdin is opened on container start, is empty until the first client attaches to stdin, and then remains open and accepts data until the client disconnects, at which time stdin is closed and remains closed until the container is restarted. If this flag is false, a container processes that reads from stdin will never receive an EOF. Default is false +optional |
| tty | [bool](#bool) | optional | Whether this container should allocate a TTY for itself, also requires 'stdin' to be true. Default is false. +optional |






<a name="k8s-io-api-core-v1-EphemeralVolumeSource"></a>

### EphemeralVolumeSource
Represents an ephemeral volume that is handled by a normal storage driver.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| volumeClaimTemplate | [PersistentVolumeClaimTemplate](#k8s-io-api-core-v1-PersistentVolumeClaimTemplate) | optional | Will be used to create a stand-alone PVC to provision the volume. The pod in which this EphemeralVolumeSource is embedded will be the owner of the PVC, i.e. the PVC will be deleted together with the pod. The name of the PVC will be `<pod name>-<volume name>` where `<volume name>` is the name from the `PodSpec.Volumes` array entry. Pod validation will reject the pod if the concatenated name is not valid for a PVC (for example, too long).

An existing PVC with that name that is not owned by the pod will *not* be used for the pod to avoid using an unrelated volume by mistake. Starting the pod is then blocked until the unrelated PVC is removed. If such a pre-created PVC is meant to be used by the pod, the PVC has to updated with an owner reference to the pod once the pod exists. Normally this should not be necessary, but it may be useful when manually reconstructing a broken cluster.

This field is read-only and no changes will be made by Kubernetes to the PVC after it has been created.

Required, must not be nil. |






<a name="k8s-io-api-core-v1-Event"></a>

### Event
Event is a report of an event somewhere in the cluster.  Events
have a limited retention time and triggers and messages may evolve
with time.  Event consumers should not rely on the timing of an event
with a given Reason reflecting a consistent underlying trigger, or the
continued existence of events with that Reason.  Events should be
treated as informative, best-effort, supplemental data.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional | Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata |
| involvedObject | [ObjectReference](#k8s-io-api-core-v1-ObjectReference) | optional | The object that this event is about. |
| reason | [string](#string) | optional | This should be a short, machine understandable string that gives the reason for the transition into the object's current status. TODO: provide exact specification for format. +optional |
| message | [string](#string) | optional | A human-readable description of the status of this operation. TODO: decide on maximum length. +optional |
| source | [EventSource](#k8s-io-api-core-v1-EventSource) | optional | The component reporting this event. Should be a short machine understandable string. +optional |
| firstTimestamp | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | The time at which the event was first recorded. (Time of server receipt is in TypeMeta.) +optional |
| lastTimestamp | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | The time at which the most recent occurrence of this event was recorded. +optional |
| count | [int32](#int32) | optional | The number of times this event has occurred. +optional |
| type | [string](#string) | optional | Type of this event (Normal, Warning), new types could be added in the future +optional |
| eventTime | [k8s.io.apimachinery.pkg.apis.meta.v1.MicroTime](#k8s-io-apimachinery-pkg-apis-meta-v1-MicroTime) | optional | Time when this Event was first observed. +optional |
| series | [EventSeries](#k8s-io-api-core-v1-EventSeries) | optional | Data about the Event series this event represents or nil if it's a singleton Event. +optional |
| action | [string](#string) | optional | What action was taken/failed regarding to the Regarding object. +optional |
| related | [ObjectReference](#k8s-io-api-core-v1-ObjectReference) | optional | Optional secondary object for more complex actions. +optional |
| reportingComponent | [string](#string) | optional | Name of the controller that emitted this Event, e.g. `kubernetes.io/kubelet`. +optional |
| reportingInstance | [string](#string) | optional | ID of the controller instance, e.g. `kubelet-xyzf`. +optional |






<a name="k8s-io-api-core-v1-EventList"></a>

### EventList
EventList is a list of events.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional | Standard list metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds +optional |
| items | [Event](#k8s-io-api-core-v1-Event) | repeated | List of events |






<a name="k8s-io-api-core-v1-EventSeries"></a>

### EventSeries
EventSeries contain information on series of events, i.e. thing that was/is happening
continuously for some time.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| count | [int32](#int32) | optional | Number of occurrences in this series up to the last heartbeat time |
| lastObservedTime | [k8s.io.apimachinery.pkg.apis.meta.v1.MicroTime](#k8s-io-apimachinery-pkg-apis-meta-v1-MicroTime) | optional | Time of the last occurrence observed |






<a name="k8s-io-api-core-v1-EventSource"></a>

### EventSource
EventSource contains information for an event.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| component | [string](#string) | optional | Component from which the event is generated. +optional |
| host | [string](#string) | optional | Node name on which the event is generated. +optional |






<a name="k8s-io-api-core-v1-ExecAction"></a>

### ExecAction
ExecAction describes a "run in container" action.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| command | [string](#string) | repeated | Command is the command line to execute inside the container, the working directory for the command is root ('/') in the container's filesystem. The command is simply exec'd, it is not run inside a shell, so traditional shell instructions ('|', etc) won't work. To use a shell, you need to explicitly call out to that shell. Exit status of 0 is treated as live/healthy and non-zero is unhealthy. +optional +listType=atomic |






<a name="k8s-io-api-core-v1-FCVolumeSource"></a>

### FCVolumeSource
Represents a Fibre Channel volume.
Fibre Channel volumes can only be mounted as read/write once.
Fibre Channel volumes support ownership management and SELinux relabeling.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| targetWWNs | [string](#string) | repeated | targetWWNs is Optional: FC target worldwide names (WWNs) +optional +listType=atomic |
| lun | [int32](#int32) | optional | lun is Optional: FC target lun number +optional |
| fsType | [string](#string) | optional | fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. TODO: how do we prevent errors in the filesystem from compromising the machine +optional |
| readOnly | [bool](#bool) | optional | readOnly is Optional: Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. +optional |
| wwids | [string](#string) | repeated | wwids Optional: FC volume world wide identifiers (wwids) Either wwids or combination of targetWWNs and lun must be set, but not both simultaneously. +optional +listType=atomic |






<a name="k8s-io-api-core-v1-FlexPersistentVolumeSource"></a>

### FlexPersistentVolumeSource
FlexPersistentVolumeSource represents a generic persistent volume resource that is
provisioned/attached using an exec based plugin.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| driver | [string](#string) | optional | driver is the name of the driver to use for this volume. |
| fsType | [string](#string) | optional | fsType is the Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". The default filesystem depends on FlexVolume script. +optional |
| secretRef | [SecretReference](#k8s-io-api-core-v1-SecretReference) | optional | secretRef is Optional: SecretRef is reference to the secret object containing sensitive information to pass to the plugin scripts. This may be empty if no secret object is specified. If the secret object contains more than one secret, all secrets are passed to the plugin scripts. +optional |
| readOnly | [bool](#bool) | optional | readOnly is Optional: defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. +optional |
| options | [FlexPersistentVolumeSource.OptionsEntry](#k8s-io-api-core-v1-FlexPersistentVolumeSource-OptionsEntry) | repeated | options is Optional: this field holds extra command options if any. +optional |






<a name="k8s-io-api-core-v1-FlexPersistentVolumeSource-OptionsEntry"></a>

### FlexPersistentVolumeSource.OptionsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [string](#string) | optional |  |






<a name="k8s-io-api-core-v1-FlexVolumeSource"></a>

### FlexVolumeSource
FlexVolume represents a generic volume resource that is
provisioned/attached using an exec based plugin.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| driver | [string](#string) | optional | driver is the name of the driver to use for this volume. |
| fsType | [string](#string) | optional | fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". The default filesystem depends on FlexVolume script. +optional |
| secretRef | [LocalObjectReference](#k8s-io-api-core-v1-LocalObjectReference) | optional | secretRef is Optional: secretRef is reference to the secret object containing sensitive information to pass to the plugin scripts. This may be empty if no secret object is specified. If the secret object contains more than one secret, all secrets are passed to the plugin scripts. +optional |
| readOnly | [bool](#bool) | optional | readOnly is Optional: defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. +optional |
| options | [FlexVolumeSource.OptionsEntry](#k8s-io-api-core-v1-FlexVolumeSource-OptionsEntry) | repeated | options is Optional: this field holds extra command options if any. +optional |






<a name="k8s-io-api-core-v1-FlexVolumeSource-OptionsEntry"></a>

### FlexVolumeSource.OptionsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [string](#string) | optional |  |






<a name="k8s-io-api-core-v1-FlockerVolumeSource"></a>

### FlockerVolumeSource
Represents a Flocker volume mounted by the Flocker agent.
One and only one of datasetName and datasetUUID should be set.
Flocker volumes do not support ownership management or SELinux relabeling.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| datasetName | [string](#string) | optional | datasetName is Name of the dataset stored as metadata -> name on the dataset for Flocker should be considered as deprecated +optional |
| datasetUUID | [string](#string) | optional | datasetUUID is the UUID of the dataset. This is unique identifier of a Flocker dataset +optional |






<a name="k8s-io-api-core-v1-GCEPersistentDiskVolumeSource"></a>

### GCEPersistentDiskVolumeSource
Represents a Persistent Disk resource in Google Compute Engine.

A GCE PD must exist before mounting to a container. The disk must
also be in the same GCE project and zone as the kubelet. A GCE PD
can only be mounted as read/write once or read-only many times. GCE
PDs support ownership management and SELinux relabeling.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| pdName | [string](#string) | optional | pdName is unique name of the PD resource in GCE. Used to identify the disk in GCE. More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk |
| fsType | [string](#string) | optional | fsType is filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk TODO: how do we prevent errors in the filesystem from compromising the machine +optional |
| partition | [int32](#int32) | optional | partition is the partition in the volume that you want to mount. If omitted, the default is to mount by volume name. Examples: For volume /dev/sda1, you specify the partition as "1". Similarly, the volume partition for /dev/sda is "0" (or you can leave the property empty). More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk +optional |
| readOnly | [bool](#bool) | optional | readOnly here will force the ReadOnly setting in VolumeMounts. Defaults to false. More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk +optional |






<a name="k8s-io-api-core-v1-GRPCAction"></a>

### GRPCAction
GRPCAction specifies an action involving a GRPC service.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| port | [int32](#int32) | optional | Port number of the gRPC service. Number must be in the range 1 to 65535. |
| service | [string](#string) | optional | Service is the name of the service to place in the gRPC HealthCheckRequest (see https://github.com/grpc/grpc/blob/master/doc/health-checking.md).

If this is not specified, the default behavior is defined by gRPC. +optional +default="" |






<a name="k8s-io-api-core-v1-GitRepoVolumeSource"></a>

### GitRepoVolumeSource
Represents a volume that is populated with the contents of a git repository.
Git repo volumes do not support ownership management.
Git repo volumes support SELinux relabeling.

DEPRECATED: GitRepo is deprecated. To provision a container with a git repo, mount an
EmptyDir into an InitContainer that clones the repo using git, then mount the EmptyDir
into the Pod's container.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repository | [string](#string) | optional | repository is the URL |
| revision | [string](#string) | optional | revision is the commit hash for the specified revision. +optional |
| directory | [string](#string) | optional | directory is the target directory name. Must not contain or start with '..'. If '.' is supplied, the volume directory will be the git repository. Otherwise, if specified, the volume will contain the git repository in the subdirectory with the given name. +optional |






<a name="k8s-io-api-core-v1-GlusterfsPersistentVolumeSource"></a>

### GlusterfsPersistentVolumeSource
Represents a Glusterfs mount that lasts the lifetime of a pod.
Glusterfs volumes do not support ownership management or SELinux relabeling.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| endpoints | [string](#string) | optional | endpoints is the endpoint name that details Glusterfs topology. More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod |
| path | [string](#string) | optional | path is the Glusterfs volume path. More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod |
| readOnly | [bool](#bool) | optional | readOnly here will force the Glusterfs volume to be mounted with read-only permissions. Defaults to false. More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod +optional |
| endpointsNamespace | [string](#string) | optional | endpointsNamespace is the namespace that contains Glusterfs endpoint. If this field is empty, the EndpointNamespace defaults to the same namespace as the bound PVC. More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod +optional |






<a name="k8s-io-api-core-v1-GlusterfsVolumeSource"></a>

### GlusterfsVolumeSource
Represents a Glusterfs mount that lasts the lifetime of a pod.
Glusterfs volumes do not support ownership management or SELinux relabeling.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| endpoints | [string](#string) | optional | endpoints is the endpoint name that details Glusterfs topology. More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod |
| path | [string](#string) | optional | path is the Glusterfs volume path. More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod |
| readOnly | [bool](#bool) | optional | readOnly here will force the Glusterfs volume to be mounted with read-only permissions. Defaults to false. More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod +optional |






<a name="k8s-io-api-core-v1-HTTPGetAction"></a>

### HTTPGetAction
HTTPGetAction describes an action based on HTTP Get requests.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) | optional | Path to access on the HTTP server. +optional |
| port | [k8s.io.apimachinery.pkg.util.intstr.IntOrString](#k8s-io-apimachinery-pkg-util-intstr-IntOrString) | optional | Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME. |
| host | [string](#string) | optional | Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead. +optional |
| scheme | [string](#string) | optional | Scheme to use for connecting to the host. Defaults to HTTP. +optional |
| httpHeaders | [HTTPHeader](#k8s-io-api-core-v1-HTTPHeader) | repeated | Custom headers to set in the request. HTTP allows repeated headers. +optional +listType=atomic |






<a name="k8s-io-api-core-v1-HTTPHeader"></a>

### HTTPHeader
HTTPHeader describes a custom header to be used in HTTP probes


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | The header field name. This will be canonicalized upon output, so case-variant names will be understood as the same header. |
| value | [string](#string) | optional | The header field value |






<a name="k8s-io-api-core-v1-HostAlias"></a>

### HostAlias
HostAlias holds the mapping between IP and hostnames that will be injected as an entry in the
pod's hosts file.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ip | [string](#string) | optional | IP address of the host file entry. +required |
| hostnames | [string](#string) | repeated | Hostnames for the above IP address. +listType=atomic |






<a name="k8s-io-api-core-v1-HostIP"></a>

### HostIP
HostIP represents a single IP address allocated to the host.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ip | [string](#string) | optional | IP is the IP address assigned to the host +required |






<a name="k8s-io-api-core-v1-HostPathVolumeSource"></a>

### HostPathVolumeSource
Represents a host path mapped into a pod.
Host path volumes do not support ownership management or SELinux relabeling.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) | optional | path of the directory on the host. If the path is a symlink, it will follow the link to the real path. More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath |
| type | [string](#string) | optional | type for HostPath Volume Defaults to "" More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath +optional |






<a name="k8s-io-api-core-v1-ISCSIPersistentVolumeSource"></a>

### ISCSIPersistentVolumeSource
ISCSIPersistentVolumeSource represents an ISCSI disk.
ISCSI volumes can only be mounted as read/write once.
ISCSI volumes support ownership management and SELinux relabeling.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| targetPortal | [string](#string) | optional | targetPortal is iSCSI Target Portal. The Portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260). |
| iqn | [string](#string) | optional | iqn is Target iSCSI Qualified Name. |
| lun | [int32](#int32) | optional | lun is iSCSI Target Lun number. |
| iscsiInterface | [string](#string) | optional | iscsiInterface is the interface Name that uses an iSCSI transport. Defaults to 'default' (tcp). +optional +default="default" |
| fsType | [string](#string) | optional | fsType is the filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. More info: https://kubernetes.io/docs/concepts/storage/volumes#iscsi TODO: how do we prevent errors in the filesystem from compromising the machine +optional |
| readOnly | [bool](#bool) | optional | readOnly here will force the ReadOnly setting in VolumeMounts. Defaults to false. +optional |
| portals | [string](#string) | repeated | portals is the iSCSI Target Portal List. The Portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260). +optional +listType=atomic |
| chapAuthDiscovery | [bool](#bool) | optional | chapAuthDiscovery defines whether support iSCSI Discovery CHAP authentication +optional |
| chapAuthSession | [bool](#bool) | optional | chapAuthSession defines whether support iSCSI Session CHAP authentication +optional |
| secretRef | [SecretReference](#k8s-io-api-core-v1-SecretReference) | optional | secretRef is the CHAP Secret for iSCSI target and initiator authentication +optional |
| initiatorName | [string](#string) | optional | initiatorName is the custom iSCSI Initiator Name. If initiatorName is specified with iscsiInterface simultaneously, new iSCSI interface <target portal>:<volume name> will be created for the connection. +optional |






<a name="k8s-io-api-core-v1-ISCSIVolumeSource"></a>

### ISCSIVolumeSource
Represents an ISCSI disk.
ISCSI volumes can only be mounted as read/write once.
ISCSI volumes support ownership management and SELinux relabeling.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| targetPortal | [string](#string) | optional | targetPortal is iSCSI Target Portal. The Portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260). |
| iqn | [string](#string) | optional | iqn is the target iSCSI Qualified Name. |
| lun | [int32](#int32) | optional | lun represents iSCSI Target Lun number. |
| iscsiInterface | [string](#string) | optional | iscsiInterface is the interface Name that uses an iSCSI transport. Defaults to 'default' (tcp). +optional +default="default" |
| fsType | [string](#string) | optional | fsType is the filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. More info: https://kubernetes.io/docs/concepts/storage/volumes#iscsi TODO: how do we prevent errors in the filesystem from compromising the machine +optional |
| readOnly | [bool](#bool) | optional | readOnly here will force the ReadOnly setting in VolumeMounts. Defaults to false. +optional |
| portals | [string](#string) | repeated | portals is the iSCSI Target Portal List. The portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260). +optional +listType=atomic |
| chapAuthDiscovery | [bool](#bool) | optional | chapAuthDiscovery defines whether support iSCSI Discovery CHAP authentication +optional |
| chapAuthSession | [bool](#bool) | optional | chapAuthSession defines whether support iSCSI Session CHAP authentication +optional |
| secretRef | [LocalObjectReference](#k8s-io-api-core-v1-LocalObjectReference) | optional | secretRef is the CHAP Secret for iSCSI target and initiator authentication +optional |
| initiatorName | [string](#string) | optional | initiatorName is the custom iSCSI Initiator Name. If initiatorName is specified with iscsiInterface simultaneously, new iSCSI interface <target portal>:<volume name> will be created for the connection. +optional |






<a name="k8s-io-api-core-v1-ImageVolumeSource"></a>

### ImageVolumeSource
ImageVolumeSource represents a image volume resource.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| reference | [string](#string) | optional | Required: Image or artifact reference to be used. Behaves in the same way as pod.spec.containers[*].image. Pull secrets will be assembled in the same way as for the container image by looking up node credentials, SA image pull secrets, and pod spec image pull secrets. More info: https://kubernetes.io/docs/concepts/containers/images This field is optional to allow higher level config management to default or override container images in workload controllers like Deployments and StatefulSets. +optional |
| pullPolicy | [string](#string) | optional | Policy for pulling OCI objects. Possible values are: Always: the kubelet always attempts to pull the reference. Container creation will fail If the pull fails. Never: the kubelet never pulls the reference and only uses a local image or artifact. Container creation will fail if the reference isn't present. IfNotPresent: the kubelet pulls if the reference isn't already present on disk. Container creation will fail if the reference isn't present and the pull fails. Defaults to Always if :latest tag is specified, or IfNotPresent otherwise. +optional |






<a name="k8s-io-api-core-v1-KeyToPath"></a>

### KeyToPath
Maps a string key to a path within a volume.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional | key is the key to project. |
| path | [string](#string) | optional | path is the relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'. |
| mode | [int32](#int32) | optional | mode is Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set. +optional |






<a name="k8s-io-api-core-v1-Lifecycle"></a>

### Lifecycle
Lifecycle describes actions that the management system should take in response to container lifecycle
events. For the PostStart and PreStop lifecycle handlers, management of the container blocks
until the action is complete, unless the container process fails, in which case the handler is aborted.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| postStart | [LifecycleHandler](#k8s-io-api-core-v1-LifecycleHandler) | optional | PostStart is called immediately after a container is created. If the handler fails, the container is terminated and restarted according to its restart policy. Other management of the container blocks until the hook completes. More info: https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#container-hooks +optional |
| preStop | [LifecycleHandler](#k8s-io-api-core-v1-LifecycleHandler) | optional | PreStop is called immediately before a container is terminated due to an API request or management event such as liveness/startup probe failure, preemption, resource contention, etc. The handler is not called if the container crashes or exits. The Pod's termination grace period countdown begins before the PreStop hook is executed. Regardless of the outcome of the handler, the container will eventually terminate within the Pod's termination grace period (unless delayed by finalizers). Other management of the container blocks until the hook completes or until the termination grace period is reached. More info: https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#container-hooks +optional |
| stopSignal | [string](#string) | optional | StopSignal defines which signal will be sent to a container when it is being stopped. If not specified, the default is defined by the container runtime in use. StopSignal can only be set for Pods with a non-empty .spec.os.name +optional |






<a name="k8s-io-api-core-v1-LifecycleHandler"></a>

### LifecycleHandler
LifecycleHandler defines a specific action that should be taken in a lifecycle
hook. One and only one of the fields, except TCPSocket must be specified.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| exec | [ExecAction](#k8s-io-api-core-v1-ExecAction) | optional | Exec specifies a command to execute in the container. +optional |
| httpGet | [HTTPGetAction](#k8s-io-api-core-v1-HTTPGetAction) | optional | HTTPGet specifies an HTTP GET request to perform. +optional |
| tcpSocket | [TCPSocketAction](#k8s-io-api-core-v1-TCPSocketAction) | optional | Deprecated. TCPSocket is NOT supported as a LifecycleHandler and kept for backward compatibility. There is no validation of this field and lifecycle hooks will fail at runtime when it is specified. +optional |
| sleep | [SleepAction](#k8s-io-api-core-v1-SleepAction) | optional | Sleep represents a duration that the container should sleep. +featureGate=PodLifecycleSleepAction +optional |






<a name="k8s-io-api-core-v1-LimitRange"></a>

### LimitRange
LimitRange sets resource usage limits for each kind of resource in a Namespace.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional | Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional |
| spec | [LimitRangeSpec](#k8s-io-api-core-v1-LimitRangeSpec) | optional | Spec defines the limits enforced. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status +optional |






<a name="k8s-io-api-core-v1-LimitRangeItem"></a>

### LimitRangeItem
LimitRangeItem defines a min/max usage limit for any resource that matches on kind.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) | optional | Type of resource that this limit applies to. |
| max | [LimitRangeItem.MaxEntry](#k8s-io-api-core-v1-LimitRangeItem-MaxEntry) | repeated | Max usage constraints on this kind by resource name. +optional |
| min | [LimitRangeItem.MinEntry](#k8s-io-api-core-v1-LimitRangeItem-MinEntry) | repeated | Min usage constraints on this kind by resource name. +optional |
| default | [LimitRangeItem.DefaultEntry](#k8s-io-api-core-v1-LimitRangeItem-DefaultEntry) | repeated | Default resource requirement limit value by resource name if resource limit is omitted. +optional |
| defaultRequest | [LimitRangeItem.DefaultRequestEntry](#k8s-io-api-core-v1-LimitRangeItem-DefaultRequestEntry) | repeated | DefaultRequest is the default resource requirement request value by resource name if resource request is omitted. +optional |
| maxLimitRequestRatio | [LimitRangeItem.MaxLimitRequestRatioEntry](#k8s-io-api-core-v1-LimitRangeItem-MaxLimitRequestRatioEntry) | repeated | MaxLimitRequestRatio if specified, the named resource must have a request and limit that are both non-zero where limit divided by request is less than or equal to the enumerated value; this represents the max burst for the named resource. +optional |






<a name="k8s-io-api-core-v1-LimitRangeItem-DefaultEntry"></a>

### LimitRangeItem.DefaultEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [k8s.io.apimachinery.pkg.api.resource.Quantity](#k8s-io-apimachinery-pkg-api-resource-Quantity) | optional |  |






<a name="k8s-io-api-core-v1-LimitRangeItem-DefaultRequestEntry"></a>

### LimitRangeItem.DefaultRequestEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [k8s.io.apimachinery.pkg.api.resource.Quantity](#k8s-io-apimachinery-pkg-api-resource-Quantity) | optional |  |






<a name="k8s-io-api-core-v1-LimitRangeItem-MaxEntry"></a>

### LimitRangeItem.MaxEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [k8s.io.apimachinery.pkg.api.resource.Quantity](#k8s-io-apimachinery-pkg-api-resource-Quantity) | optional |  |






<a name="k8s-io-api-core-v1-LimitRangeItem-MaxLimitRequestRatioEntry"></a>

### LimitRangeItem.MaxLimitRequestRatioEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [k8s.io.apimachinery.pkg.api.resource.Quantity](#k8s-io-apimachinery-pkg-api-resource-Quantity) | optional |  |






<a name="k8s-io-api-core-v1-LimitRangeItem-MinEntry"></a>

### LimitRangeItem.MinEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [k8s.io.apimachinery.pkg.api.resource.Quantity](#k8s-io-apimachinery-pkg-api-resource-Quantity) | optional |  |






<a name="k8s-io-api-core-v1-LimitRangeList"></a>

### LimitRangeList
LimitRangeList is a list of LimitRange items.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional | Standard list metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds +optional |
| items | [LimitRange](#k8s-io-api-core-v1-LimitRange) | repeated | Items is a list of LimitRange objects. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/ |






<a name="k8s-io-api-core-v1-LimitRangeSpec"></a>

### LimitRangeSpec
LimitRangeSpec defines a min/max usage limit for resources that match on kind.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| limits | [LimitRangeItem](#k8s-io-api-core-v1-LimitRangeItem) | repeated | Limits is the list of LimitRangeItem objects that are enforced. +listType=atomic |






<a name="k8s-io-api-core-v1-LinuxContainerUser"></a>

### LinuxContainerUser
LinuxContainerUser represents user identity information in Linux containers


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| uid | [int64](#int64) | optional | UID is the primary uid initially attached to the first process in the container |
| gid | [int64](#int64) | optional | GID is the primary gid initially attached to the first process in the container |
| supplementalGroups | [int64](#int64) | repeated | SupplementalGroups are the supplemental groups initially attached to the first process in the container +optional +listType=atomic |






<a name="k8s-io-api-core-v1-List"></a>

### List
List holds a list of objects, which may not be known by the server.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional | Standard list metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds +optional |
| items | [k8s.io.apimachinery.pkg.runtime.RawExtension](#k8s-io-apimachinery-pkg-runtime-RawExtension) | repeated | List of objects |






<a name="k8s-io-api-core-v1-LoadBalancerIngress"></a>

### LoadBalancerIngress
LoadBalancerIngress represents the status of a load-balancer ingress point:
traffic intended for the service should be sent to an ingress point.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ip | [string](#string) | optional | IP is set for load-balancer ingress points that are IP based (typically GCE or OpenStack load-balancers) +optional |
| hostname | [string](#string) | optional | Hostname is set for load-balancer ingress points that are DNS based (typically AWS load-balancers) +optional |
| ipMode | [string](#string) | optional | IPMode specifies how the load-balancer IP behaves, and may only be specified when the ip field is specified. Setting this to "VIP" indicates that traffic is delivered to the node with the destination set to the load-balancer's IP and port. Setting this to "Proxy" indicates that traffic is delivered to the node or pod with the destination set to the node's IP and node port or the pod's IP and port. Service implementations may use this information to adjust traffic routing. +optional |
| ports | [PortStatus](#k8s-io-api-core-v1-PortStatus) | repeated | Ports is a list of records of service ports If used, every port defined in the service should have an entry in it +listType=atomic +optional |






<a name="k8s-io-api-core-v1-LoadBalancerStatus"></a>

### LoadBalancerStatus
LoadBalancerStatus represents the status of a load-balancer.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ingress | [LoadBalancerIngress](#k8s-io-api-core-v1-LoadBalancerIngress) | repeated | Ingress is a list containing ingress points for the load-balancer. Traffic intended for the service should be sent to these ingress points. +optional +listType=atomic |






<a name="k8s-io-api-core-v1-LocalObjectReference"></a>

### LocalObjectReference
LocalObjectReference contains enough information to let you locate the
referenced object inside the same namespace.
---
New uses of this type are discouraged because of difficulty describing its usage when embedded in APIs.
 1. Invalid usage help.  It is impossible to add specific help for individual usage.  In most embedded usages, there are particular
    restrictions like, "must refer only to types A and B" or "UID not honored" or "name must be restricted".
    Those cannot be well described when embedded.
 2. Inconsistent validation.  Because the usages are different, the validation rules are different by usage, which makes it hard for users to predict what will happen.
 3. We cannot easily change it.  Because this type is embedded in many locations, updates to this type
    will affect numerous schemas.  Don't make new APIs embed an underspecified API type they do not control.

Instead of using this type, create a locally provided and used type that is well-focused on your reference.
For example, ServiceReferences for admission registration: https://github.com/kubernetes/api/blob/release-1.17/admissionregistration/v1/types.go#L533 .
+structType=atomic


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name of the referent. This field is effectively required, but due to backwards compatibility is allowed to be empty. Instances of this type with an empty value here are almost certainly wrong. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names +optional +default="" +kubebuilder:default="" TODO: Drop `kubebuilder:default` when controller-gen doesn't need it https://github.com/kubernetes-sigs/kubebuilder/issues/3896. |






<a name="k8s-io-api-core-v1-LocalVolumeSource"></a>

### LocalVolumeSource
Local represents directly-attached storage with node affinity


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) | optional | path of the full path to the volume on the node. It can be either a directory or block device (disk, partition, ...). |
| fsType | [string](#string) | optional | fsType is the filesystem type to mount. It applies only when the Path is a block device. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". The default value is to auto-select a filesystem if unspecified. +optional |






<a name="k8s-io-api-core-v1-ModifyVolumeStatus"></a>

### ModifyVolumeStatus
ModifyVolumeStatus represents the status object of ControllerModifyVolume operation


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| targetVolumeAttributesClassName | [string](#string) | optional | targetVolumeAttributesClassName is the name of the VolumeAttributesClass the PVC currently being reconciled |
| status | [string](#string) | optional | status is the status of the ControllerModifyVolume operation. It can be in any of following states: - Pending Pending indicates that the PersistentVolumeClaim cannot be modified due to unmet requirements, such as the specified VolumeAttributesClass not existing. - InProgress InProgress indicates that the volume is being modified. - Infeasible Infeasible indicates that the request has been rejected as invalid by the CSI driver. To 	 resolve the error, a valid VolumeAttributesClass needs to be specified. Note: New statuses can be added in the future. Consumers should check for unknown statuses and fail appropriately. |






<a name="k8s-io-api-core-v1-NFSVolumeSource"></a>

### NFSVolumeSource
Represents an NFS mount that lasts the lifetime of a pod.
NFS volumes do not support ownership management or SELinux relabeling.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| server | [string](#string) | optional | server is the hostname or IP address of the NFS server. More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs |
| path | [string](#string) | optional | path that is exported by the NFS server. More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs |
| readOnly | [bool](#bool) | optional | readOnly here will force the NFS export to be mounted with read-only permissions. Defaults to false. More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs +optional |






<a name="k8s-io-api-core-v1-Namespace"></a>

### Namespace
Namespace provides a scope for Names.
Use of multiple namespaces is optional.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional | Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional |
| spec | [NamespaceSpec](#k8s-io-api-core-v1-NamespaceSpec) | optional | Spec defines the behavior of the Namespace. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status +optional |
| status | [NamespaceStatus](#k8s-io-api-core-v1-NamespaceStatus) | optional | Status describes the current status of a Namespace. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status +optional |






<a name="k8s-io-api-core-v1-NamespaceCondition"></a>

### NamespaceCondition
NamespaceCondition contains details about state of namespace.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) | optional | Type of namespace controller condition. |
| status | [string](#string) | optional | Status of the condition, one of True, False, Unknown. |
| lastTransitionTime | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | Last time the condition transitioned from one status to another. +optional |
| reason | [string](#string) | optional | Unique, one-word, CamelCase reason for the condition's last transition. +optional |
| message | [string](#string) | optional | Human-readable message indicating details about last transition. +optional |






<a name="k8s-io-api-core-v1-NamespaceList"></a>

### NamespaceList
NamespaceList is a list of Namespaces.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional | Standard list metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds +optional |
| items | [Namespace](#k8s-io-api-core-v1-Namespace) | repeated | Items is the list of Namespace objects in the list. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/ |






<a name="k8s-io-api-core-v1-NamespaceSpec"></a>

### NamespaceSpec
NamespaceSpec describes the attributes on a Namespace.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| finalizers | [string](#string) | repeated | Finalizers is an opaque list of values that must be empty to permanently remove object from storage. More info: https://kubernetes.io/docs/tasks/administer-cluster/namespaces/ +optional +listType=atomic |






<a name="k8s-io-api-core-v1-NamespaceStatus"></a>

### NamespaceStatus
NamespaceStatus is information about the current status of a Namespace.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| phase | [string](#string) | optional | Phase is the current lifecycle phase of the namespace. More info: https://kubernetes.io/docs/tasks/administer-cluster/namespaces/ +optional |
| conditions | [NamespaceCondition](#k8s-io-api-core-v1-NamespaceCondition) | repeated | Represents the latest available observations of a namespace's current state. +optional +patchMergeKey=type +patchStrategy=merge +listType=map +listMapKey=type |






<a name="k8s-io-api-core-v1-Node"></a>

### Node
Node is a worker node in Kubernetes.
Each node will have a unique identifier in the cache (i.e. in etcd).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional | Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional |
| spec | [NodeSpec](#k8s-io-api-core-v1-NodeSpec) | optional | Spec defines the behavior of a node. https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status +optional |
| status | [NodeStatus](#k8s-io-api-core-v1-NodeStatus) | optional | Most recently observed status of the node. Populated by the system. Read-only. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status +optional |






<a name="k8s-io-api-core-v1-NodeAddress"></a>

### NodeAddress
NodeAddress contains information for the node's address.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) | optional | Node address type, one of Hostname, ExternalIP or InternalIP. |
| address | [string](#string) | optional | The node address. |






<a name="k8s-io-api-core-v1-NodeAffinity"></a>

### NodeAffinity
Node affinity is a group of node affinity scheduling rules.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| requiredDuringSchedulingIgnoredDuringExecution | [NodeSelector](#k8s-io-api-core-v1-NodeSelector) | optional | If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to an update), the system may or may not try to eventually evict the pod from its node. +optional |
| preferredDuringSchedulingIgnoredDuringExecution | [PreferredSchedulingTerm](#k8s-io-api-core-v1-PreferredSchedulingTerm) | repeated | The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node matches the corresponding matchExpressions; the node(s) with the highest sum are the most preferred. +optional +listType=atomic |






<a name="k8s-io-api-core-v1-NodeCondition"></a>

### NodeCondition
NodeCondition contains condition information for a node.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) | optional | Type of node condition. |
| status | [string](#string) | optional | Status of the condition, one of True, False, Unknown. |
| lastHeartbeatTime | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | Last time we got an update on a given condition. +optional |
| lastTransitionTime | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | Last time the condition transit from one status to another. +optional |
| reason | [string](#string) | optional | (brief) reason for the condition's last transition. +optional |
| message | [string](#string) | optional | Human readable message indicating details about last transition. +optional |






<a name="k8s-io-api-core-v1-NodeConfigSource"></a>

### NodeConfigSource
NodeConfigSource specifies a source of node configuration. Exactly one subfield (excluding metadata) must be non-nil.
This API is deprecated since 1.22


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| configMap | [ConfigMapNodeConfigSource](#k8s-io-api-core-v1-ConfigMapNodeConfigSource) | optional | ConfigMap is a reference to a Node's ConfigMap |






<a name="k8s-io-api-core-v1-NodeConfigStatus"></a>

### NodeConfigStatus
NodeConfigStatus describes the status of the config assigned by Node.Spec.ConfigSource.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| assigned | [NodeConfigSource](#k8s-io-api-core-v1-NodeConfigSource) | optional | Assigned reports the checkpointed config the node will try to use. When Node.Spec.ConfigSource is updated, the node checkpoints the associated config payload to local disk, along with a record indicating intended config. The node refers to this record to choose its config checkpoint, and reports this record in Assigned. Assigned only updates in the status after the record has been checkpointed to disk. When the Kubelet is restarted, it tries to make the Assigned config the Active config by loading and validating the checkpointed payload identified by Assigned. +optional |
| active | [NodeConfigSource](#k8s-io-api-core-v1-NodeConfigSource) | optional | Active reports the checkpointed config the node is actively using. Active will represent either the current version of the Assigned config, or the current LastKnownGood config, depending on whether attempting to use the Assigned config results in an error. +optional |
| lastKnownGood | [NodeConfigSource](#k8s-io-api-core-v1-NodeConfigSource) | optional | LastKnownGood reports the checkpointed config the node will fall back to when it encounters an error attempting to use the Assigned config. The Assigned config becomes the LastKnownGood config when the node determines that the Assigned config is stable and correct. This is currently implemented as a 10-minute soak period starting when the local record of Assigned config is updated. If the Assigned config is Active at the end of this period, it becomes the LastKnownGood. Note that if Spec.ConfigSource is reset to nil (use local defaults), the LastKnownGood is also immediately reset to nil, because the local default config is always assumed good. You should not make assumptions about the node's method of determining config stability and correctness, as this may change or become configurable in the future. +optional |
| error | [string](#string) | optional | Error describes any problems reconciling the Spec.ConfigSource to the Active config. Errors may occur, for example, attempting to checkpoint Spec.ConfigSource to the local Assigned record, attempting to checkpoint the payload associated with Spec.ConfigSource, attempting to load or validate the Assigned config, etc. Errors may occur at different points while syncing config. Earlier errors (e.g. download or checkpointing errors) will not result in a rollback to LastKnownGood, and may resolve across Kubelet retries. Later errors (e.g. loading or validating a checkpointed config) will result in a rollback to LastKnownGood. In the latter case, it is usually possible to resolve the error by fixing the config assigned in Spec.ConfigSource. You can find additional information for debugging by searching the error message in the Kubelet log. Error is a human-readable description of the error state; machines can check whether or not Error is empty, but should not rely on the stability of the Error text across Kubelet versions. +optional |






<a name="k8s-io-api-core-v1-NodeDaemonEndpoints"></a>

### NodeDaemonEndpoints
NodeDaemonEndpoints lists ports opened by daemons running on the Node.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| kubeletEndpoint | [DaemonEndpoint](#k8s-io-api-core-v1-DaemonEndpoint) | optional | Endpoint on which Kubelet is listening. +optional |






<a name="k8s-io-api-core-v1-NodeFeatures"></a>

### NodeFeatures
NodeFeatures describes the set of features implemented by the CRI implementation.
The features contained in the NodeFeatures should depend only on the cri implementation
independent of runtime handlers.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| supplementalGroupsPolicy | [bool](#bool) | optional | SupplementalGroupsPolicy is set to true if the runtime supports SupplementalGroupsPolicy and ContainerUser. +optional |






<a name="k8s-io-api-core-v1-NodeList"></a>

### NodeList
NodeList is the whole list of all Nodes which have been registered with master.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional | Standard list metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds +optional |
| items | [Node](#k8s-io-api-core-v1-Node) | repeated | List of nodes |






<a name="k8s-io-api-core-v1-NodeProxyOptions"></a>

### NodeProxyOptions
NodeProxyOptions is the query options to a Node's proxy call.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) | optional | Path is the URL path to use for the current proxy request to node. +optional |






<a name="k8s-io-api-core-v1-NodeRuntimeHandler"></a>

### NodeRuntimeHandler
NodeRuntimeHandler is a set of runtime handler information.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Runtime handler name. Empty for the default runtime handler. +optional |
| features | [NodeRuntimeHandlerFeatures](#k8s-io-api-core-v1-NodeRuntimeHandlerFeatures) | optional | Supported features. +optional |






<a name="k8s-io-api-core-v1-NodeRuntimeHandlerFeatures"></a>

### NodeRuntimeHandlerFeatures
NodeRuntimeHandlerFeatures is a set of features implemented by the runtime handler.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| recursiveReadOnlyMounts | [bool](#bool) | optional | RecursiveReadOnlyMounts is set to true if the runtime handler supports RecursiveReadOnlyMounts. +featureGate=RecursiveReadOnlyMounts +optional |
| userNamespaces | [bool](#bool) | optional | UserNamespaces is set to true if the runtime handler supports UserNamespaces, including for volumes. +featureGate=UserNamespacesSupport +optional |






<a name="k8s-io-api-core-v1-NodeSelector"></a>

### NodeSelector
A node selector represents the union of the results of one or more label queries
over a set of nodes; that is, it represents the OR of the selectors represented
by the node selector terms.
+structType=atomic


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| nodeSelectorTerms | [NodeSelectorTerm](#k8s-io-api-core-v1-NodeSelectorTerm) | repeated | Required. A list of node selector terms. The terms are ORed. +listType=atomic |






<a name="k8s-io-api-core-v1-NodeSelectorRequirement"></a>

### NodeSelectorRequirement
A node selector requirement is a selector that contains values, a key, and an operator
that relates the key and values.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional | The label key that the selector applies to. |
| operator | [string](#string) | optional | Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt. |
| values | [string](#string) | repeated | An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch. +optional +listType=atomic |






<a name="k8s-io-api-core-v1-NodeSelectorTerm"></a>

### NodeSelectorTerm
A null or empty node selector term matches no objects. The requirements of
them are ANDed.
The TopologySelectorTerm type implements a subset of the NodeSelectorTerm.
+structType=atomic


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| matchExpressions | [NodeSelectorRequirement](#k8s-io-api-core-v1-NodeSelectorRequirement) | repeated | A list of node selector requirements by node's labels. +optional +listType=atomic |
| matchFields | [NodeSelectorRequirement](#k8s-io-api-core-v1-NodeSelectorRequirement) | repeated | A list of node selector requirements by node's fields. +optional +listType=atomic |






<a name="k8s-io-api-core-v1-NodeSpec"></a>

### NodeSpec
NodeSpec describes the attributes that a node is created with.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| podCIDR | [string](#string) | optional | PodCIDR represents the pod IP range assigned to the node. +optional |
| podCIDRs | [string](#string) | repeated | podCIDRs represents the IP ranges assigned to the node for usage by Pods on that node. If this field is specified, the 0th entry must match the podCIDR field. It may contain at most 1 value for each of IPv4 and IPv6. +optional +patchStrategy=merge +listType=set |
| providerID | [string](#string) | optional | ID of the node assigned by the cloud provider in the format: <ProviderName>://<ProviderSpecificNodeID> +optional |
| unschedulable | [bool](#bool) | optional | Unschedulable controls node schedulability of new pods. By default, node is schedulable. More info: https://kubernetes.io/docs/concepts/nodes/node/#manual-node-administration +optional |
| taints | [Taint](#k8s-io-api-core-v1-Taint) | repeated | If specified, the node's taints. +optional +listType=atomic |
| configSource | [NodeConfigSource](#k8s-io-api-core-v1-NodeConfigSource) | optional | Deprecated: Previously used to specify the source of the node's configuration for the DynamicKubeletConfig feature. This feature is removed. +optional |
| externalID | [string](#string) | optional | Deprecated. Not all kubelets will set this field. Remove field after 1.13. see: https://issues.k8s.io/61966 +optional |






<a name="k8s-io-api-core-v1-NodeStatus"></a>

### NodeStatus
NodeStatus is information about the current status of a node.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| capacity | [NodeStatus.CapacityEntry](#k8s-io-api-core-v1-NodeStatus-CapacityEntry) | repeated | Capacity represents the total resources of a node. More info: https://kubernetes.io/docs/reference/node/node-status/#capacity +optional |
| allocatable | [NodeStatus.AllocatableEntry](#k8s-io-api-core-v1-NodeStatus-AllocatableEntry) | repeated | Allocatable represents the resources of a node that are available for scheduling. Defaults to Capacity. +optional |
| phase | [string](#string) | optional | NodePhase is the recently observed lifecycle phase of the node. More info: https://kubernetes.io/docs/concepts/nodes/node/#phase The field is never populated, and now is deprecated. +optional |
| conditions | [NodeCondition](#k8s-io-api-core-v1-NodeCondition) | repeated | Conditions is an array of current observed node conditions. More info: https://kubernetes.io/docs/reference/node/node-status/#condition +optional +patchMergeKey=type +patchStrategy=merge +listType=map +listMapKey=type |
| addresses | [NodeAddress](#k8s-io-api-core-v1-NodeAddress) | repeated | List of addresses reachable to the node. Queried from cloud provider, if available. More info: https://kubernetes.io/docs/reference/node/node-status/#addresses Note: This field is declared as mergeable, but the merge key is not sufficiently unique, which can cause data corruption when it is merged. Callers should instead use a full-replacement patch. See https://pr.k8s.io/79391 for an example. Consumers should assume that addresses can change during the lifetime of a Node. However, there are some exceptions where this may not be possible, such as Pods that inherit a Node's address in its own status or consumers of the downward API (status.hostIP). +optional +patchMergeKey=type +patchStrategy=merge +listType=map +listMapKey=type |
| daemonEndpoints | [NodeDaemonEndpoints](#k8s-io-api-core-v1-NodeDaemonEndpoints) | optional | Endpoints of daemons running on the Node. +optional |
| nodeInfo | [NodeSystemInfo](#k8s-io-api-core-v1-NodeSystemInfo) | optional | Set of ids/uuids to uniquely identify the node. More info: https://kubernetes.io/docs/reference/node/node-status/#info +optional |
| images | [ContainerImage](#k8s-io-api-core-v1-ContainerImage) | repeated | List of container images on this node +optional +listType=atomic |
| volumesInUse | [string](#string) | repeated | List of attachable volumes in use (mounted) by the node. +optional +listType=atomic |
| volumesAttached | [AttachedVolume](#k8s-io-api-core-v1-AttachedVolume) | repeated | List of volumes that are attached to the node. +optional +listType=atomic |
| config | [NodeConfigStatus](#k8s-io-api-core-v1-NodeConfigStatus) | optional | Status of the config assigned to the node via the dynamic Kubelet config feature. +optional |
| runtimeHandlers | [NodeRuntimeHandler](#k8s-io-api-core-v1-NodeRuntimeHandler) | repeated | The available runtime handlers. +featureGate=RecursiveReadOnlyMounts +featureGate=UserNamespacesSupport +optional +listType=atomic |
| features | [NodeFeatures](#k8s-io-api-core-v1-NodeFeatures) | optional | Features describes the set of features implemented by the CRI implementation. +featureGate=SupplementalGroupsPolicy +optional |






<a name="k8s-io-api-core-v1-NodeStatus-AllocatableEntry"></a>

### NodeStatus.AllocatableEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [k8s.io.apimachinery.pkg.api.resource.Quantity](#k8s-io-apimachinery-pkg-api-resource-Quantity) | optional |  |






<a name="k8s-io-api-core-v1-NodeStatus-CapacityEntry"></a>

### NodeStatus.CapacityEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [k8s.io.apimachinery.pkg.api.resource.Quantity](#k8s-io-apimachinery-pkg-api-resource-Quantity) | optional |  |






<a name="k8s-io-api-core-v1-NodeSwapStatus"></a>

### NodeSwapStatus
NodeSwapStatus represents swap memory information.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| capacity | [int64](#int64) | optional | Total amount of swap memory in bytes. +optional |






<a name="k8s-io-api-core-v1-NodeSystemInfo"></a>

### NodeSystemInfo
NodeSystemInfo is a set of ids/uuids to uniquely identify the node.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| machineID | [string](#string) | optional | MachineID reported by the node. For unique machine identification in the cluster this field is preferred. Learn more from man(5) machine-id: http://man7.org/linux/man-pages/man5/machine-id.5.html |
| systemUUID | [string](#string) | optional | SystemUUID reported by the node. For unique machine identification MachineID is preferred. This field is specific to Red Hat hosts https://access.redhat.com/documentation/en-us/red_hat_subscription_management/1/html/rhsm/uuid |
| bootID | [string](#string) | optional | Boot ID reported by the node. |
| kernelVersion | [string](#string) | optional | Kernel Version reported by the node from 'uname -r' (e.g. 3.16.0-0.bpo.4-amd64). |
| osImage | [string](#string) | optional | OS Image reported by the node from /etc/os-release (e.g. Debian GNU/Linux 7 (wheezy)). |
| containerRuntimeVersion | [string](#string) | optional | ContainerRuntime Version reported by the node through runtime remote API (e.g. containerd://1.4.2). |
| kubeletVersion | [string](#string) | optional | Kubelet Version reported by the node. |
| kubeProxyVersion | [string](#string) | optional | Deprecated: KubeProxy Version reported by the node. |
| operatingSystem | [string](#string) | optional | The Operating System reported by the node |
| architecture | [string](#string) | optional | The Architecture reported by the node |
| swap | [NodeSwapStatus](#k8s-io-api-core-v1-NodeSwapStatus) | optional | Swap Info reported by the node. |






<a name="k8s-io-api-core-v1-ObjectFieldSelector"></a>

### ObjectFieldSelector
ObjectFieldSelector selects an APIVersioned field of an object.
+structType=atomic


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| apiVersion | [string](#string) | optional | Version of the schema the FieldPath is written in terms of, defaults to "v1". +optional |
| fieldPath | [string](#string) | optional | Path of the field to select in the specified API version. |






<a name="k8s-io-api-core-v1-ObjectReference"></a>

### ObjectReference
ObjectReference contains enough information to let you inspect or modify the referred object.
---
New uses of this type are discouraged because of difficulty describing its usage when embedded in APIs.
 1. Ignored fields.  It includes many fields which are not generally honored.  For instance, ResourceVersion and FieldPath are both very rarely valid in actual usage.
 2. Invalid usage help.  It is impossible to add specific help for individual usage.  In most embedded usages, there are particular
    restrictions like, "must refer only to types A and B" or "UID not honored" or "name must be restricted".
    Those cannot be well described when embedded.
 3. Inconsistent validation.  Because the usages are different, the validation rules are different by usage, which makes it hard for users to predict what will happen.
 4. The fields are both imprecise and overly precise.  Kind is not a precise mapping to a URL. This can produce ambiguity
    during interpretation and require a REST mapping.  In most cases, the dependency is on the group,resource tuple
    and the version of the actual struct is irrelevant.
 5. We cannot easily change it.  Because this type is embedded in many locations, updates to this type
    will affect numerous schemas.  Don't make new APIs embed an underspecified API type they do not control.

Instead of using this type, create a locally provided and used type that is well-focused on your reference.
For example, ServiceReferences for admission registration: https://github.com/kubernetes/api/blob/release-1.17/admissionregistration/v1/types.go#L533 .
+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
+structType=atomic


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| kind | [string](#string) | optional | Kind of the referent. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds +optional |
| namespace | [string](#string) | optional | Namespace of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/ +optional |
| name | [string](#string) | optional | Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names +optional |
| uid | [string](#string) | optional | UID of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids +optional |
| apiVersion | [string](#string) | optional | API version of the referent. +optional |
| resourceVersion | [string](#string) | optional | Specific resourceVersion to which this reference is made, if any. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency +optional |
| fieldPath | [string](#string) | optional | If referring to a piece of an object instead of an entire object, this string should contain a valid JSON/Go field access statement, such as desiredState.manifest.containers[2]. For example, if the object reference is to a container within a pod, this would take on a value like: "spec.containers{name}" (where "name" refers to the name of the container that triggered the event) or if no container name is specified "spec.containers[2]" (container with index 2 in this pod). This syntax is chosen only to have some well-defined way of referencing a part of an object. TODO: this design is not final and this field is subject to change in the future. +optional |






<a name="k8s-io-api-core-v1-PersistentVolume"></a>

### PersistentVolume
PersistentVolume (PV) is a storage resource provisioned by an administrator.
It is analogous to a node.
More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional | Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional |
| spec | [PersistentVolumeSpec](#k8s-io-api-core-v1-PersistentVolumeSpec) | optional | spec defines a specification of a persistent volume owned by the cluster. Provisioned by an administrator. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistent-volumes +optional |
| status | [PersistentVolumeStatus](#k8s-io-api-core-v1-PersistentVolumeStatus) | optional | status represents the current information/status for the persistent volume. Populated by the system. Read-only. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistent-volumes +optional |






<a name="k8s-io-api-core-v1-PersistentVolumeClaim"></a>

### PersistentVolumeClaim
PersistentVolumeClaim is a user's request for and claim to a persistent volume


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional | Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional |
| spec | [PersistentVolumeClaimSpec](#k8s-io-api-core-v1-PersistentVolumeClaimSpec) | optional | spec defines the desired characteristics of a volume requested by a pod author. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims +optional |
| status | [PersistentVolumeClaimStatus](#k8s-io-api-core-v1-PersistentVolumeClaimStatus) | optional | status represents the current information/status of a persistent volume claim. Read-only. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims +optional |






<a name="k8s-io-api-core-v1-PersistentVolumeClaimCondition"></a>

### PersistentVolumeClaimCondition
PersistentVolumeClaimCondition contains details about state of pvc


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) | optional | Type is the type of the condition. More info: https://kubernetes.io/docs/reference/kubernetes-api/config-and-storage-resources/persistent-volume-claim-v1/#:~:text=set%20to%20%27ResizeStarted%27.-,PersistentVolumeClaimCondition,-contains%20details%20about |
| status | [string](#string) | optional | Status is the status of the condition. Can be True, False, Unknown. More info: https://kubernetes.io/docs/reference/kubernetes-api/config-and-storage-resources/persistent-volume-claim-v1/#:~:text=state%20of%20pvc-,conditions.status,-(string)%2C%20required |
| lastProbeTime | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | lastProbeTime is the time we probed the condition. +optional |
| lastTransitionTime | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | lastTransitionTime is the time the condition transitioned from one status to another. +optional |
| reason | [string](#string) | optional | reason is a unique, this should be a short, machine understandable string that gives the reason for condition's last transition. If it reports "Resizing" that means the underlying persistent volume is being resized. +optional |
| message | [string](#string) | optional | message is the human-readable message indicating details about last transition. +optional |






<a name="k8s-io-api-core-v1-PersistentVolumeClaimList"></a>

### PersistentVolumeClaimList
PersistentVolumeClaimList is a list of PersistentVolumeClaim items.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional | Standard list metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds +optional |
| items | [PersistentVolumeClaim](#k8s-io-api-core-v1-PersistentVolumeClaim) | repeated | items is a list of persistent volume claims. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims |






<a name="k8s-io-api-core-v1-PersistentVolumeClaimSpec"></a>

### PersistentVolumeClaimSpec
PersistentVolumeClaimSpec describes the common attributes of storage devices
and allows a Source for provider-specific attributes


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| accessModes | [string](#string) | repeated | accessModes contains the desired access modes the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1 +optional +listType=atomic |
| selector | [k8s.io.apimachinery.pkg.apis.meta.v1.LabelSelector](#k8s-io-apimachinery-pkg-apis-meta-v1-LabelSelector) | optional | selector is a label query over volumes to consider for binding. +optional |
| resources | [VolumeResourceRequirements](#k8s-io-api-core-v1-VolumeResourceRequirements) | optional | resources represents the minimum resources the volume should have. If RecoverVolumeExpansionFailure feature is enabled users are allowed to specify resource requirements that are lower than previous value but must still be higher than capacity recorded in the status field of the claim. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources +optional |
| volumeName | [string](#string) | optional | volumeName is the binding reference to the PersistentVolume backing this claim. +optional |
| storageClassName | [string](#string) | optional | storageClassName is the name of the StorageClass required by the claim. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1 +optional |
| volumeMode | [string](#string) | optional | volumeMode defines what type of volume is required by the claim. Value of Filesystem is implied when not included in claim spec. +optional |
| dataSource | [TypedLocalObjectReference](#k8s-io-api-core-v1-TypedLocalObjectReference) | optional | dataSource field can be used to specify either: * An existing VolumeSnapshot object (snapshot.storage.k8s.io/VolumeSnapshot) * An existing PVC (PersistentVolumeClaim) If the provisioner or an external controller can support the specified data source, it will create a new volume based on the contents of the specified data source. When the AnyVolumeDataSource feature gate is enabled, dataSource contents will be copied to dataSourceRef, and dataSourceRef contents will be copied to dataSource when dataSourceRef.namespace is not specified. If the namespace is specified, then dataSourceRef will not be copied to dataSource. +optional |
| dataSourceRef | [TypedObjectReference](#k8s-io-api-core-v1-TypedObjectReference) | optional | dataSourceRef specifies the object from which to populate the volume with data, if a non-empty volume is desired. This may be any object from a non-empty API group (non core object) or a PersistentVolumeClaim object. When this field is specified, volume binding will only succeed if the type of the specified object matches some installed volume populator or dynamic provisioner. This field will replace the functionality of the dataSource field and as such if both fields are non-empty, they must have the same value. For backwards compatibility, when namespace isn't specified in dataSourceRef, both fields (dataSource and dataSourceRef) will be set to the same value automatically if one of them is empty and the other is non-empty. When namespace is specified in dataSourceRef, dataSource isn't set to the same value and must be empty. There are three important differences between dataSource and dataSourceRef: * While dataSource only allows two specific types of objects, dataSourceRef allows any non-core object, as well as PersistentVolumeClaim objects. * While dataSource ignores disallowed values (dropping them), dataSourceRef preserves all values, and generates an error if a disallowed value is specified. * While dataSource only allows local objects, dataSourceRef allows objects in any namespaces. (Beta) Using this field requires the AnyVolumeDataSource feature gate to be enabled. (Alpha) Using the namespace field of dataSourceRef requires the CrossNamespaceVolumeDataSource feature gate to be enabled. +optional |
| volumeAttributesClassName | [string](#string) | optional | volumeAttributesClassName may be used to set the VolumeAttributesClass used by this claim. If specified, the CSI driver will create or update the volume with the attributes defined in the corresponding VolumeAttributesClass. This has a different purpose than storageClassName, it can be changed after the claim is created. An empty string value means that no VolumeAttributesClass will be applied to the claim but it's not allowed to reset this field to empty string once it is set. If unspecified and the PersistentVolumeClaim is unbound, the default VolumeAttributesClass will be set by the persistentvolume controller if it exists. If the resource referred to by volumeAttributesClass does not exist, this PersistentVolumeClaim will be set to a Pending state, as reflected by the modifyVolumeStatus field, until such as a resource exists. More info: https://kubernetes.io/docs/concepts/storage/volume-attributes-classes/ (Beta) Using this field requires the VolumeAttributesClass feature gate to be enabled (off by default). +featureGate=VolumeAttributesClass +optional |






<a name="k8s-io-api-core-v1-PersistentVolumeClaimStatus"></a>

### PersistentVolumeClaimStatus
PersistentVolumeClaimStatus is the current status of a persistent volume claim.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| phase | [string](#string) | optional | phase represents the current phase of PersistentVolumeClaim. +optional |
| accessModes | [string](#string) | repeated | accessModes contains the actual access modes the volume backing the PVC has. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1 +optional +listType=atomic |
| capacity | [PersistentVolumeClaimStatus.CapacityEntry](#k8s-io-api-core-v1-PersistentVolumeClaimStatus-CapacityEntry) | repeated | capacity represents the actual resources of the underlying volume. +optional |
| conditions | [PersistentVolumeClaimCondition](#k8s-io-api-core-v1-PersistentVolumeClaimCondition) | repeated | conditions is the current Condition of persistent volume claim. If underlying persistent volume is being resized then the Condition will be set to 'Resizing'. +optional +patchMergeKey=type +patchStrategy=merge +listType=map +listMapKey=type |
| allocatedResources | [PersistentVolumeClaimStatus.AllocatedResourcesEntry](#k8s-io-api-core-v1-PersistentVolumeClaimStatus-AllocatedResourcesEntry) | repeated | allocatedResources tracks the resources allocated to a PVC including its capacity. Key names follow standard Kubernetes label syntax. Valid values are either: 	* Un-prefixed keys: 		- storage - the capacity of the volume. 	* Custom resources must use implementation-defined prefixed names such as "example.com/my-custom-resource" Apart from above values - keys that are unprefixed or have kubernetes.io prefix are considered reserved and hence may not be used.

Capacity reported here may be larger than the actual capacity when a volume expansion operation is requested. For storage quota, the larger value from allocatedResources and PVC.spec.resources is used. If allocatedResources is not set, PVC.spec.resources alone is used for quota calculation. If a volume expansion capacity request is lowered, allocatedResources is only lowered if there are no expansion operations in progress and if the actual volume capacity is equal or lower than the requested capacity.

A controller that receives PVC update with previously unknown resourceName should ignore the update for the purpose it was designed. For example - a controller that only is responsible for resizing capacity of the volume, should ignore PVC updates that change other valid resources associated with PVC.

This is an alpha field and requires enabling RecoverVolumeExpansionFailure feature. +featureGate=RecoverVolumeExpansionFailure +optional |
| allocatedResourceStatuses | [PersistentVolumeClaimStatus.AllocatedResourceStatusesEntry](#k8s-io-api-core-v1-PersistentVolumeClaimStatus-AllocatedResourceStatusesEntry) | repeated | allocatedResourceStatuses stores status of resource being resized for the given PVC. Key names follow standard Kubernetes label syntax. Valid values are either: 	* Un-prefixed keys: 		- storage - the capacity of the volume. 	* Custom resources must use implementation-defined prefixed names such as "example.com/my-custom-resource" Apart from above values - keys that are unprefixed or have kubernetes.io prefix are considered reserved and hence may not be used.

ClaimResourceStatus can be in any of following states: 	- ControllerResizeInProgress: 		State set when resize controller starts resizing the volume in control-plane. 	- ControllerResizeFailed: 		State set when resize has failed in resize controller with a terminal error. 	- NodeResizePending: 		State set when resize controller has finished resizing the volume but further resizing of 		volume is needed on the node. 	- NodeResizeInProgress: 		State set when kubelet starts resizing the volume. 	- NodeResizeFailed: 		State set when resizing has failed in kubelet with a terminal error. Transient errors don't set 		NodeResizeFailed. For example: if expanding a PVC for more capacity - this field can be one of the following states: 	- pvc.status.allocatedResourceStatus['storage'] = "ControllerResizeInProgress" - pvc.status.allocatedResourceStatus['storage'] = "ControllerResizeFailed" - pvc.status.allocatedResourceStatus['storage'] = "NodeResizePending" - pvc.status.allocatedResourceStatus['storage'] = "NodeResizeInProgress" - pvc.status.allocatedResourceStatus['storage'] = "NodeResizeFailed" When this field is not set, it means that no resize operation is in progress for the given PVC.

A controller that receives PVC update with previously unknown resourceName or ClaimResourceStatus should ignore the update for the purpose it was designed. For example - a controller that only is responsible for resizing capacity of the volume, should ignore PVC updates that change other valid resources associated with PVC.

This is an alpha field and requires enabling RecoverVolumeExpansionFailure feature. +featureGate=RecoverVolumeExpansionFailure +mapType=granular +optional |
| currentVolumeAttributesClassName | [string](#string) | optional | currentVolumeAttributesClassName is the current name of the VolumeAttributesClass the PVC is using. When unset, there is no VolumeAttributeClass applied to this PersistentVolumeClaim This is a beta field and requires enabling VolumeAttributesClass feature (off by default). +featureGate=VolumeAttributesClass +optional |
| modifyVolumeStatus | [ModifyVolumeStatus](#k8s-io-api-core-v1-ModifyVolumeStatus) | optional | ModifyVolumeStatus represents the status object of ControllerModifyVolume operation. When this is unset, there is no ModifyVolume operation being attempted. This is a beta field and requires enabling VolumeAttributesClass feature (off by default). +featureGate=VolumeAttributesClass +optional |






<a name="k8s-io-api-core-v1-PersistentVolumeClaimStatus-AllocatedResourceStatusesEntry"></a>

### PersistentVolumeClaimStatus.AllocatedResourceStatusesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [string](#string) | optional |  |






<a name="k8s-io-api-core-v1-PersistentVolumeClaimStatus-AllocatedResourcesEntry"></a>

### PersistentVolumeClaimStatus.AllocatedResourcesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [k8s.io.apimachinery.pkg.api.resource.Quantity](#k8s-io-apimachinery-pkg-api-resource-Quantity) | optional |  |






<a name="k8s-io-api-core-v1-PersistentVolumeClaimStatus-CapacityEntry"></a>

### PersistentVolumeClaimStatus.CapacityEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [k8s.io.apimachinery.pkg.api.resource.Quantity](#k8s-io-apimachinery-pkg-api-resource-Quantity) | optional |  |






<a name="k8s-io-api-core-v1-PersistentVolumeClaimTemplate"></a>

### PersistentVolumeClaimTemplate
PersistentVolumeClaimTemplate is used to produce
PersistentVolumeClaim objects as part of an EphemeralVolumeSource.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional | May contain labels and annotations that will be copied into the PVC when creating it. No other fields are allowed and will be rejected during validation.

+optional |
| spec | [PersistentVolumeClaimSpec](#k8s-io-api-core-v1-PersistentVolumeClaimSpec) | optional | The specification for the PersistentVolumeClaim. The entire content is copied unchanged into the PVC that gets created from this template. The same fields as in a PersistentVolumeClaim are also valid here. |






<a name="k8s-io-api-core-v1-PersistentVolumeClaimVolumeSource"></a>

### PersistentVolumeClaimVolumeSource
PersistentVolumeClaimVolumeSource references the user's PVC in the same namespace.
This volume finds the bound PV and mounts that volume for the pod. A
PersistentVolumeClaimVolumeSource is, essentially, a wrapper around another
type of volume that is owned by someone else (the system).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| claimName | [string](#string) | optional | claimName is the name of a PersistentVolumeClaim in the same namespace as the pod using this volume. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims |
| readOnly | [bool](#bool) | optional | readOnly Will force the ReadOnly setting in VolumeMounts. Default false. +optional |






<a name="k8s-io-api-core-v1-PersistentVolumeList"></a>

### PersistentVolumeList
PersistentVolumeList is a list of PersistentVolume items.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional | Standard list metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds +optional |
| items | [PersistentVolume](#k8s-io-api-core-v1-PersistentVolume) | repeated | items is a list of persistent volumes. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes |






<a name="k8s-io-api-core-v1-PersistentVolumeSource"></a>

### PersistentVolumeSource
PersistentVolumeSource is similar to VolumeSource but meant for the
administrator who creates PVs. Exactly one of its members must be set.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| gcePersistentDisk | [GCEPersistentDiskVolumeSource](#k8s-io-api-core-v1-GCEPersistentDiskVolumeSource) | optional | gcePersistentDisk represents a GCE Disk resource that is attached to a kubelet's host machine and then exposed to the pod. Provisioned by an admin. Deprecated: GCEPersistentDisk is deprecated. All operations for the in-tree gcePersistentDisk type are redirected to the pd.csi.storage.gke.io CSI driver. More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk +optional |
| awsElasticBlockStore | [AWSElasticBlockStoreVolumeSource](#k8s-io-api-core-v1-AWSElasticBlockStoreVolumeSource) | optional | awsElasticBlockStore represents an AWS Disk resource that is attached to a kubelet's host machine and then exposed to the pod. Deprecated: AWSElasticBlockStore is deprecated. All operations for the in-tree awsElasticBlockStore type are redirected to the ebs.csi.aws.com CSI driver. More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore +optional |
| hostPath | [HostPathVolumeSource](#k8s-io-api-core-v1-HostPathVolumeSource) | optional | hostPath represents a directory on the host. Provisioned by a developer or tester. This is useful for single-node development and testing only! On-host storage is not supported in any way and WILL NOT WORK in a multi-node cluster. More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath +optional |
| glusterfs | [GlusterfsPersistentVolumeSource](#k8s-io-api-core-v1-GlusterfsPersistentVolumeSource) | optional | glusterfs represents a Glusterfs volume that is attached to a host and exposed to the pod. Provisioned by an admin. Deprecated: Glusterfs is deprecated and the in-tree glusterfs type is no longer supported. More info: https://examples.k8s.io/volumes/glusterfs/README.md +optional |
| nfs | [NFSVolumeSource](#k8s-io-api-core-v1-NFSVolumeSource) | optional | nfs represents an NFS mount on the host. Provisioned by an admin. More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs +optional |
| rbd | [RBDPersistentVolumeSource](#k8s-io-api-core-v1-RBDPersistentVolumeSource) | optional | rbd represents a Rados Block Device mount on the host that shares a pod's lifetime. Deprecated: RBD is deprecated and the in-tree rbd type is no longer supported. More info: https://examples.k8s.io/volumes/rbd/README.md +optional |
| iscsi | [ISCSIPersistentVolumeSource](#k8s-io-api-core-v1-ISCSIPersistentVolumeSource) | optional | iscsi represents an ISCSI Disk resource that is attached to a kubelet's host machine and then exposed to the pod. Provisioned by an admin. +optional |
| cinder | [CinderPersistentVolumeSource](#k8s-io-api-core-v1-CinderPersistentVolumeSource) | optional | cinder represents a cinder volume attached and mounted on kubelets host machine. Deprecated: Cinder is deprecated. All operations for the in-tree cinder type are redirected to the cinder.csi.openstack.org CSI driver. More info: https://examples.k8s.io/mysql-cinder-pd/README.md +optional |
| cephfs | [CephFSPersistentVolumeSource](#k8s-io-api-core-v1-CephFSPersistentVolumeSource) | optional | cephFS represents a Ceph FS mount on the host that shares a pod's lifetime. Deprecated: CephFS is deprecated and the in-tree cephfs type is no longer supported. +optional |
| fc | [FCVolumeSource](#k8s-io-api-core-v1-FCVolumeSource) | optional | fc represents a Fibre Channel resource that is attached to a kubelet's host machine and then exposed to the pod. +optional |
| flocker | [FlockerVolumeSource](#k8s-io-api-core-v1-FlockerVolumeSource) | optional | flocker represents a Flocker volume attached to a kubelet's host machine and exposed to the pod for its usage. This depends on the Flocker control service being running. Deprecated: Flocker is deprecated and the in-tree flocker type is no longer supported. +optional |
| flexVolume | [FlexPersistentVolumeSource](#k8s-io-api-core-v1-FlexPersistentVolumeSource) | optional | flexVolume represents a generic volume resource that is provisioned/attached using an exec based plugin. Deprecated: FlexVolume is deprecated. Consider using a CSIDriver instead. +optional |
| azureFile | [AzureFilePersistentVolumeSource](#k8s-io-api-core-v1-AzureFilePersistentVolumeSource) | optional | azureFile represents an Azure File Service mount on the host and bind mount to the pod. Deprecated: AzureFile is deprecated. All operations for the in-tree azureFile type are redirected to the file.csi.azure.com CSI driver. +optional |
| vsphereVolume | [VsphereVirtualDiskVolumeSource](#k8s-io-api-core-v1-VsphereVirtualDiskVolumeSource) | optional | vsphereVolume represents a vSphere volume attached and mounted on kubelets host machine. Deprecated: VsphereVolume is deprecated. All operations for the in-tree vsphereVolume type are redirected to the csi.vsphere.vmware.com CSI driver. +optional |
| quobyte | [QuobyteVolumeSource](#k8s-io-api-core-v1-QuobyteVolumeSource) | optional | quobyte represents a Quobyte mount on the host that shares a pod's lifetime. Deprecated: Quobyte is deprecated and the in-tree quobyte type is no longer supported. +optional |
| azureDisk | [AzureDiskVolumeSource](#k8s-io-api-core-v1-AzureDiskVolumeSource) | optional | azureDisk represents an Azure Data Disk mount on the host and bind mount to the pod. Deprecated: AzureDisk is deprecated. All operations for the in-tree azureDisk type are redirected to the disk.csi.azure.com CSI driver. +optional |
| photonPersistentDisk | [PhotonPersistentDiskVolumeSource](#k8s-io-api-core-v1-PhotonPersistentDiskVolumeSource) | optional | photonPersistentDisk represents a PhotonController persistent disk attached and mounted on kubelets host machine. Deprecated: PhotonPersistentDisk is deprecated and the in-tree photonPersistentDisk type is no longer supported. |
| portworxVolume | [PortworxVolumeSource](#k8s-io-api-core-v1-PortworxVolumeSource) | optional | portworxVolume represents a portworx volume attached and mounted on kubelets host machine. Deprecated: PortworxVolume is deprecated. All operations for the in-tree portworxVolume type are redirected to the pxd.portworx.com CSI driver when the CSIMigrationPortworx feature-gate is on. +optional |
| scaleIO | [ScaleIOPersistentVolumeSource](#k8s-io-api-core-v1-ScaleIOPersistentVolumeSource) | optional | scaleIO represents a ScaleIO persistent volume attached and mounted on Kubernetes nodes. Deprecated: ScaleIO is deprecated and the in-tree scaleIO type is no longer supported. +optional |
| local | [LocalVolumeSource](#k8s-io-api-core-v1-LocalVolumeSource) | optional | local represents directly-attached storage with node affinity +optional |
| storageos | [StorageOSPersistentVolumeSource](#k8s-io-api-core-v1-StorageOSPersistentVolumeSource) | optional | storageOS represents a StorageOS volume that is attached to the kubelet's host machine and mounted into the pod. Deprecated: StorageOS is deprecated and the in-tree storageos type is no longer supported. More info: https://examples.k8s.io/volumes/storageos/README.md +optional |
| csi | [CSIPersistentVolumeSource](#k8s-io-api-core-v1-CSIPersistentVolumeSource) | optional | csi represents storage that is handled by an external CSI driver. +optional |






<a name="k8s-io-api-core-v1-PersistentVolumeSpec"></a>

### PersistentVolumeSpec
PersistentVolumeSpec is the specification of a persistent volume.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| capacity | [PersistentVolumeSpec.CapacityEntry](#k8s-io-api-core-v1-PersistentVolumeSpec-CapacityEntry) | repeated | capacity is the description of the persistent volume's resources and capacity. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#capacity +optional |
| persistentVolumeSource | [PersistentVolumeSource](#k8s-io-api-core-v1-PersistentVolumeSource) | optional | persistentVolumeSource is the actual volume backing the persistent volume. |
| accessModes | [string](#string) | repeated | accessModes contains all ways the volume can be mounted. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes +optional +listType=atomic |
| claimRef | [ObjectReference](#k8s-io-api-core-v1-ObjectReference) | optional | claimRef is part of a bi-directional binding between PersistentVolume and PersistentVolumeClaim. Expected to be non-nil when bound. claim.VolumeName is the authoritative bind between PV and PVC. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#binding +optional +structType=granular |
| persistentVolumeReclaimPolicy | [string](#string) | optional | persistentVolumeReclaimPolicy defines what happens to a persistent volume when released from its claim. Valid options are Retain (default for manually created PersistentVolumes), Delete (default for dynamically provisioned PersistentVolumes), and Recycle (deprecated). Recycle must be supported by the volume plugin underlying this PersistentVolume. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#reclaiming +optional |
| storageClassName | [string](#string) | optional | storageClassName is the name of StorageClass to which this persistent volume belongs. Empty value means that this volume does not belong to any StorageClass. +optional |
| mountOptions | [string](#string) | repeated | mountOptions is the list of mount options, e.g. ["ro", "soft"]. Not validated - mount will simply fail if one is invalid. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes/#mount-options +optional +listType=atomic |
| volumeMode | [string](#string) | optional | volumeMode defines if a volume is intended to be used with a formatted filesystem or to remain in raw block state. Value of Filesystem is implied when not included in spec. +optional |
| nodeAffinity | [VolumeNodeAffinity](#k8s-io-api-core-v1-VolumeNodeAffinity) | optional | nodeAffinity defines constraints that limit what nodes this volume can be accessed from. This field influences the scheduling of pods that use this volume. +optional |
| volumeAttributesClassName | [string](#string) | optional | Name of VolumeAttributesClass to which this persistent volume belongs. Empty value is not allowed. When this field is not set, it indicates that this volume does not belong to any VolumeAttributesClass. This field is mutable and can be changed by the CSI driver after a volume has been updated successfully to a new class. For an unbound PersistentVolume, the volumeAttributesClassName will be matched with unbound PersistentVolumeClaims during the binding process. This is a beta field and requires enabling VolumeAttributesClass feature (off by default). +featureGate=VolumeAttributesClass +optional |






<a name="k8s-io-api-core-v1-PersistentVolumeSpec-CapacityEntry"></a>

### PersistentVolumeSpec.CapacityEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [k8s.io.apimachinery.pkg.api.resource.Quantity](#k8s-io-apimachinery-pkg-api-resource-Quantity) | optional |  |






<a name="k8s-io-api-core-v1-PersistentVolumeStatus"></a>

### PersistentVolumeStatus
PersistentVolumeStatus is the current status of a persistent volume.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| phase | [string](#string) | optional | phase indicates if a volume is available, bound to a claim, or released by a claim. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#phase +optional |
| message | [string](#string) | optional | message is a human-readable message indicating details about why the volume is in this state. +optional |
| reason | [string](#string) | optional | reason is a brief CamelCase string that describes any failure and is meant for machine parsing and tidy display in the CLI. +optional |
| lastPhaseTransitionTime | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | lastPhaseTransitionTime is the time the phase transitioned from one to another and automatically resets to current time everytime a volume phase transitions. +optional |






<a name="k8s-io-api-core-v1-PhotonPersistentDiskVolumeSource"></a>

### PhotonPersistentDiskVolumeSource
Represents a Photon Controller persistent disk resource.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| pdID | [string](#string) | optional | pdID is the ID that identifies Photon Controller persistent disk |
| fsType | [string](#string) | optional | fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. |






<a name="k8s-io-api-core-v1-Pod"></a>

### Pod
Pod is a collection of containers that can run on a host. This resource is created
by clients and scheduled onto hosts.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional | Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional |
| spec | [PodSpec](#k8s-io-api-core-v1-PodSpec) | optional | Specification of the desired behavior of the pod. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status +optional |
| status | [PodStatus](#k8s-io-api-core-v1-PodStatus) | optional | Most recently observed status of the pod. This data may not be up to date. Populated by the system. Read-only. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status +optional |






<a name="k8s-io-api-core-v1-PodAffinity"></a>

### PodAffinity
Pod affinity is a group of inter pod affinity scheduling rules.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| requiredDuringSchedulingIgnoredDuringExecution | [PodAffinityTerm](#k8s-io-api-core-v1-PodAffinityTerm) | repeated | If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied. +optional +listType=atomic |
| preferredDuringSchedulingIgnoredDuringExecution | [WeightedPodAffinityTerm](#k8s-io-api-core-v1-WeightedPodAffinityTerm) | repeated | The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred. +optional +listType=atomic |






<a name="k8s-io-api-core-v1-PodAffinityTerm"></a>

### PodAffinityTerm
Defines a set of pods (namely those matching the labelSelector
relative to the given namespace(s)) that this pod should be
co-located (affinity) or not co-located (anti-affinity) with,
where co-located is defined as running on a node whose value of
the label with key <topologyKey> matches that of any node on which
a pod of the set of pods is running


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| labelSelector | [k8s.io.apimachinery.pkg.apis.meta.v1.LabelSelector](#k8s-io-apimachinery-pkg-apis-meta-v1-LabelSelector) | optional | A label query over a set of resources, in this case pods. If it's null, this PodAffinityTerm matches with no Pods. +optional |
| namespaces | [string](#string) | repeated | namespaces specifies a static list of namespace names that the term applies to. The term is applied to the union of the namespaces listed in this field and the ones selected by namespaceSelector. null or empty namespaces list and null namespaceSelector means "this pod's namespace". +optional +listType=atomic |
| topologyKey | [string](#string) | optional | This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed. |
| namespaceSelector | [k8s.io.apimachinery.pkg.apis.meta.v1.LabelSelector](#k8s-io-apimachinery-pkg-apis-meta-v1-LabelSelector) | optional | A label query over the set of namespaces that the term applies to. The term is applied to the union of the namespaces selected by this field and the ones listed in the namespaces field. null selector and null or empty namespaces list means "this pod's namespace". An empty selector ({}) matches all namespaces. +optional |
| matchLabelKeys | [string](#string) | repeated | MatchLabelKeys is a set of pod label keys to select which pods will be taken into consideration. The keys are used to lookup values from the incoming pod labels, those key-value labels are merged with `labelSelector` as `key in (value)` to select the group of existing pods which pods will be taken into consideration for the incoming pod's pod (anti) affinity. Keys that don't exist in the incoming pod labels will be ignored. The default value is empty. The same key is forbidden to exist in both matchLabelKeys and labelSelector. Also, matchLabelKeys cannot be set when labelSelector isn't set.

+listType=atomic +optional |
| mismatchLabelKeys | [string](#string) | repeated | MismatchLabelKeys is a set of pod label keys to select which pods will be taken into consideration. The keys are used to lookup values from the incoming pod labels, those key-value labels are merged with `labelSelector` as `key notin (value)` to select the group of existing pods which pods will be taken into consideration for the incoming pod's pod (anti) affinity. Keys that don't exist in the incoming pod labels will be ignored. The default value is empty. The same key is forbidden to exist in both mismatchLabelKeys and labelSelector. Also, mismatchLabelKeys cannot be set when labelSelector isn't set.

+listType=atomic +optional |






<a name="k8s-io-api-core-v1-PodAntiAffinity"></a>

### PodAntiAffinity
Pod anti affinity is a group of inter pod anti affinity scheduling rules.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| requiredDuringSchedulingIgnoredDuringExecution | [PodAffinityTerm](#k8s-io-api-core-v1-PodAffinityTerm) | repeated | If the anti-affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the anti-affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied. +optional +listType=atomic |
| preferredDuringSchedulingIgnoredDuringExecution | [WeightedPodAffinityTerm](#k8s-io-api-core-v1-WeightedPodAffinityTerm) | repeated | The scheduler will prefer to schedule pods to nodes that satisfy the anti-affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling anti-affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred. +optional +listType=atomic |






<a name="k8s-io-api-core-v1-PodAttachOptions"></a>

### PodAttachOptions
PodAttachOptions is the query options to a Pod's remote attach call.
---
TODO: merge w/ PodExecOptions below for stdin, stdout, etc
and also when we cut V2, we should export a "StreamOptions" or somesuch that contains Stdin, Stdout, Stder and TTY


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| stdin | [bool](#bool) | optional | Stdin if true, redirects the standard input stream of the pod for this call. Defaults to false. +optional |
| stdout | [bool](#bool) | optional | Stdout if true indicates that stdout is to be redirected for the attach call. Defaults to true. +optional |
| stderr | [bool](#bool) | optional | Stderr if true indicates that stderr is to be redirected for the attach call. Defaults to true. +optional |
| tty | [bool](#bool) | optional | TTY if true indicates that a tty will be allocated for the attach call. This is passed through the container runtime so the tty is allocated on the worker node by the container runtime. Defaults to false. +optional |
| container | [string](#string) | optional | The container in which to execute the command. Defaults to only container if there is only one container in the pod. +optional |






<a name="k8s-io-api-core-v1-PodCondition"></a>

### PodCondition
PodCondition contains details for the current condition of this pod.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) | optional | Type is the type of the condition. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#pod-conditions |
| observedGeneration | [int64](#int64) | optional | If set, this represents the .metadata.generation that the pod condition was set based upon. This is an alpha field. Enable PodObservedGenerationTracking to be able to use this field. +featureGate=PodObservedGenerationTracking +optional |
| status | [string](#string) | optional | Status is the status of the condition. Can be True, False, Unknown. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#pod-conditions |
| lastProbeTime | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | Last time we probed the condition. +optional |
| lastTransitionTime | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | Last time the condition transitioned from one status to another. +optional |
| reason | [string](#string) | optional | Unique, one-word, CamelCase reason for the condition's last transition. +optional |
| message | [string](#string) | optional | Human-readable message indicating details about last transition. +optional |






<a name="k8s-io-api-core-v1-PodDNSConfig"></a>

### PodDNSConfig
PodDNSConfig defines the DNS parameters of a pod in addition to
those generated from DNSPolicy.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| nameservers | [string](#string) | repeated | A list of DNS name server IP addresses. This will be appended to the base nameservers generated from DNSPolicy. Duplicated nameservers will be removed. +optional +listType=atomic |
| searches | [string](#string) | repeated | A list of DNS search domains for host-name lookup. This will be appended to the base search paths generated from DNSPolicy. Duplicated search paths will be removed. +optional +listType=atomic |
| options | [PodDNSConfigOption](#k8s-io-api-core-v1-PodDNSConfigOption) | repeated | A list of DNS resolver options. This will be merged with the base options generated from DNSPolicy. Duplicated entries will be removed. Resolution options given in Options will override those that appear in the base DNSPolicy. +optional +listType=atomic |






<a name="k8s-io-api-core-v1-PodDNSConfigOption"></a>

### PodDNSConfigOption
PodDNSConfigOption defines DNS resolver options of a pod.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name is this DNS resolver option's name. Required. |
| value | [string](#string) | optional | Value is this DNS resolver option's value. +optional |






<a name="k8s-io-api-core-v1-PodExecOptions"></a>

### PodExecOptions
PodExecOptions is the query options to a Pod's remote exec call.
---
TODO: This is largely identical to PodAttachOptions above, make sure they stay in sync and see about merging
and also when we cut V2, we should export a "StreamOptions" or somesuch that contains Stdin, Stdout, Stder and TTY


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| stdin | [bool](#bool) | optional | Redirect the standard input stream of the pod for this call. Defaults to false. +optional |
| stdout | [bool](#bool) | optional | Redirect the standard output stream of the pod for this call. +optional |
| stderr | [bool](#bool) | optional | Redirect the standard error stream of the pod for this call. +optional |
| tty | [bool](#bool) | optional | TTY if true indicates that a tty will be allocated for the exec call. Defaults to false. +optional |
| container | [string](#string) | optional | Container in which to execute the command. Defaults to only container if there is only one container in the pod. +optional |
| command | [string](#string) | repeated | Command is the remote command to execute. argv array. Not executed within a shell. +listType=atomic |






<a name="k8s-io-api-core-v1-PodIP"></a>

### PodIP
PodIP represents a single IP address allocated to the pod.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ip | [string](#string) | optional | IP is the IP address assigned to the pod +required |






<a name="k8s-io-api-core-v1-PodList"></a>

### PodList
PodList is a list of Pods.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional | Standard list metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds +optional |
| items | [Pod](#k8s-io-api-core-v1-Pod) | repeated | List of pods. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md |






<a name="k8s-io-api-core-v1-PodLogOptions"></a>

### PodLogOptions
PodLogOptions is the query options for a Pod's logs REST call.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| container | [string](#string) | optional | The container for which to stream logs. Defaults to only container if there is one container in the pod. +optional |
| follow | [bool](#bool) | optional | Follow the log stream of the pod. Defaults to false. +optional |
| previous | [bool](#bool) | optional | Return previous terminated container logs. Defaults to false. +optional |
| sinceSeconds | [int64](#int64) | optional | A relative time in seconds before the current time from which to show logs. If this value precedes the time a pod was started, only logs since the pod start will be returned. If this value is in the future, no logs will be returned. Only one of sinceSeconds or sinceTime may be specified. +optional |
| sinceTime | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | An RFC3339 timestamp from which to show logs. If this value precedes the time a pod was started, only logs since the pod start will be returned. If this value is in the future, no logs will be returned. Only one of sinceSeconds or sinceTime may be specified. +optional |
| timestamps | [bool](#bool) | optional | If true, add an RFC3339 or RFC3339Nano timestamp at the beginning of every line of log output. Defaults to false. +optional |
| tailLines | [int64](#int64) | optional | If set, the number of lines from the end of the logs to show. If not specified, logs are shown from the creation of the container or sinceSeconds or sinceTime. Note that when "TailLines" is specified, "Stream" can only be set to nil or "All". +optional |
| limitBytes | [int64](#int64) | optional | If set, the number of bytes to read from the server before terminating the log output. This may not display a complete final line of logging, and may return slightly more or slightly less than the specified limit. +optional |
| insecureSkipTLSVerifyBackend | [bool](#bool) | optional | insecureSkipTLSVerifyBackend indicates that the apiserver should not confirm the validity of the serving certificate of the backend it is connecting to. This will make the HTTPS connection between the apiserver and the backend insecure. This means the apiserver cannot verify the log data it is receiving came from the real kubelet. If the kubelet is configured to verify the apiserver's TLS credentials, it does not mean the connection to the real kubelet is vulnerable to a man in the middle attack (e.g. an attacker could not intercept the actual log data coming from the real kubelet). +optional |
| stream | [string](#string) | optional | Specify which container log stream to return to the client. Acceptable values are "All", "Stdout" and "Stderr". If not specified, "All" is used, and both stdout and stderr are returned interleaved. Note that when "TailLines" is specified, "Stream" can only be set to nil or "All". +featureGate=PodLogsQuerySplitStreams +optional |






<a name="k8s-io-api-core-v1-PodOS"></a>

### PodOS
PodOS defines the OS parameters of a pod.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name is the name of the operating system. The currently supported values are linux and windows. Additional value may be defined in future and can be one of: https://github.com/opencontainers/runtime-spec/blob/master/config.md#platform-specific-configuration Clients should expect to handle additional values and treat unrecognized values in this field as os: null |






<a name="k8s-io-api-core-v1-PodPortForwardOptions"></a>

### PodPortForwardOptions
PodPortForwardOptions is the query options to a Pod's port forward call
when using WebSockets.
The `port` query parameter must specify the port or
ports (comma separated) to forward over.
Port forwarding over SPDY does not use these options. It requires the port
to be passed in the `port` header as part of request.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ports | [int32](#int32) | repeated | List of ports to forward Required when using WebSockets +optional +listType=atomic |






<a name="k8s-io-api-core-v1-PodProxyOptions"></a>

### PodProxyOptions
PodProxyOptions is the query options to a Pod's proxy call.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) | optional | Path is the URL path to use for the current proxy request to pod. +optional |






<a name="k8s-io-api-core-v1-PodReadinessGate"></a>

### PodReadinessGate
PodReadinessGate contains the reference to a pod condition


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| conditionType | [string](#string) | optional | ConditionType refers to a condition in the pod's condition list with matching type. |






<a name="k8s-io-api-core-v1-PodResourceClaim"></a>

### PodResourceClaim
PodResourceClaim references exactly one ResourceClaim, either directly
or by naming a ResourceClaimTemplate which is then turned into a ResourceClaim
for the pod.

It adds a name to it that uniquely identifies the ResourceClaim inside the Pod.
Containers that need access to the ResourceClaim reference it with this name.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name uniquely identifies this resource claim inside the pod. This must be a DNS_LABEL. |
| resourceClaimName | [string](#string) | optional | ResourceClaimName is the name of a ResourceClaim object in the same namespace as this pod.

Exactly one of ResourceClaimName and ResourceClaimTemplateName must be set. |
| resourceClaimTemplateName | [string](#string) | optional | ResourceClaimTemplateName is the name of a ResourceClaimTemplate object in the same namespace as this pod.

The template will be used to create a new ResourceClaim, which will be bound to this pod. When this pod is deleted, the ResourceClaim will also be deleted. The pod name and resource name, along with a generated component, will be used to form a unique name for the ResourceClaim, which will be recorded in pod.status.resourceClaimStatuses.

This field is immutable and no changes will be made to the corresponding ResourceClaim by the control plane after creating the ResourceClaim.

Exactly one of ResourceClaimName and ResourceClaimTemplateName must be set. |






<a name="k8s-io-api-core-v1-PodResourceClaimStatus"></a>

### PodResourceClaimStatus
PodResourceClaimStatus is stored in the PodStatus for each PodResourceClaim
which references a ResourceClaimTemplate. It stores the generated name for
the corresponding ResourceClaim.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name uniquely identifies this resource claim inside the pod. This must match the name of an entry in pod.spec.resourceClaims, which implies that the string must be a DNS_LABEL. |
| resourceClaimName | [string](#string) | optional | ResourceClaimName is the name of the ResourceClaim that was generated for the Pod in the namespace of the Pod. If this is unset, then generating a ResourceClaim was not necessary. The pod.spec.resourceClaims entry can be ignored in this case.

+optional |






<a name="k8s-io-api-core-v1-PodSchedulingGate"></a>

### PodSchedulingGate
PodSchedulingGate is associated to a Pod to guard its scheduling.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name of the scheduling gate. Each scheduling gate must have a unique name field. |






<a name="k8s-io-api-core-v1-PodSecurityContext"></a>

### PodSecurityContext
PodSecurityContext holds pod-level security attributes and common container settings.
Some fields are also present in container.securityContext.  Field values of
container.securityContext take precedence over field values of PodSecurityContext.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| seLinuxOptions | [SELinuxOptions](#k8s-io-api-core-v1-SELinuxOptions) | optional | The SELinux context to be applied to all containers. If unspecified, the container runtime will allocate a random SELinux context for each container. May also be set in SecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence for that container. Note that this field cannot be set when spec.os.name is windows. +optional |
| windowsOptions | [WindowsSecurityContextOptions](#k8s-io-api-core-v1-WindowsSecurityContextOptions) | optional | The Windows specific settings applied to all containers. If unspecified, the options within a container's SecurityContext will be used. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. Note that this field cannot be set when spec.os.name is linux. +optional |
| runAsUser | [int64](#int64) | optional | The UID to run the entrypoint of the container process. Defaults to user specified in image metadata if unspecified. May also be set in SecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence for that container. Note that this field cannot be set when spec.os.name is windows. +optional |
| runAsGroup | [int64](#int64) | optional | The GID to run the entrypoint of the container process. Uses runtime default if unset. May also be set in SecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence for that container. Note that this field cannot be set when spec.os.name is windows. +optional |
| runAsNonRoot | [bool](#bool) | optional | Indicates that the container must run as a non-root user. If true, the Kubelet will validate the image at runtime to ensure that it does not run as UID 0 (root) and fail to start the container if it does. If unset or false, no such validation will be performed. May also be set in SecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. +optional |
| supplementalGroups | [int64](#int64) | repeated | A list of groups applied to the first process run in each container, in addition to the container's primary GID and fsGroup (if specified). If the SupplementalGroupsPolicy feature is enabled, the supplementalGroupsPolicy field determines whether these are in addition to or instead of any group memberships defined in the container image. If unspecified, no additional groups are added, though group memberships defined in the container image may still be used, depending on the supplementalGroupsPolicy field. Note that this field cannot be set when spec.os.name is windows. +optional +listType=atomic |
| supplementalGroupsPolicy | [string](#string) | optional | Defines how supplemental groups of the first container processes are calculated. Valid values are "Merge" and "Strict". If not specified, "Merge" is used. (Alpha) Using the field requires the SupplementalGroupsPolicy feature gate to be enabled and the container runtime must implement support for this feature. Note that this field cannot be set when spec.os.name is windows. TODO: update the default value to "Merge" when spec.os.name is not windows in v1.34 +featureGate=SupplementalGroupsPolicy +optional |
| fsGroup | [int64](#int64) | optional | A special supplemental group that applies to all containers in a pod. Some volume types allow the Kubelet to change the ownership of that volume to be owned by the pod:

1. The owning GID will be the FSGroup 2. The setgid bit is set (new files created in the volume will be owned by FSGroup) 3. The permission bits are OR'd with rw-rw----

If unset, the Kubelet will not modify the ownership and permissions of any volume. Note that this field cannot be set when spec.os.name is windows. +optional |
| sysctls | [Sysctl](#k8s-io-api-core-v1-Sysctl) | repeated | Sysctls hold a list of namespaced sysctls used for the pod. Pods with unsupported sysctls (by the container runtime) might fail to launch. Note that this field cannot be set when spec.os.name is windows. +optional +listType=atomic |
| fsGroupChangePolicy | [string](#string) | optional | fsGroupChangePolicy defines behavior of changing ownership and permission of the volume before being exposed inside Pod. This field will only apply to volume types which support fsGroup based ownership(and permissions). It will have no effect on ephemeral volume types such as: secret, configmaps and emptydir. Valid values are "OnRootMismatch" and "Always". If not specified, "Always" is used. Note that this field cannot be set when spec.os.name is windows. +optional |
| seccompProfile | [SeccompProfile](#k8s-io-api-core-v1-SeccompProfile) | optional | The seccomp options to use by the containers in this pod. Note that this field cannot be set when spec.os.name is windows. +optional |
| appArmorProfile | [AppArmorProfile](#k8s-io-api-core-v1-AppArmorProfile) | optional | appArmorProfile is the AppArmor options to use by the containers in this pod. Note that this field cannot be set when spec.os.name is windows. +optional |
| seLinuxChangePolicy | [string](#string) | optional | seLinuxChangePolicy defines how the container's SELinux label is applied to all volumes used by the Pod. It has no effect on nodes that do not support SELinux or to volumes does not support SELinux. Valid values are "MountOption" and "Recursive".

"Recursive" means relabeling of all files on all Pod volumes by the container runtime. This may be slow for large volumes, but allows mixing privileged and unprivileged Pods sharing the same volume on the same node.

"MountOption" mounts all eligible Pod volumes with `-o context` mount option. This requires all Pods that share the same volume to use the same SELinux label. It is not possible to share the same volume among privileged and unprivileged Pods. Eligible volumes are in-tree FibreChannel and iSCSI volumes, and all CSI volumes whose CSI driver announces SELinux support by setting spec.seLinuxMount: true in their CSIDriver instance. Other volumes are always re-labelled recursively. "MountOption" value is allowed only when SELinuxMount feature gate is enabled.

If not specified and SELinuxMount feature gate is enabled, "MountOption" is used. If not specified and SELinuxMount feature gate is disabled, "MountOption" is used for ReadWriteOncePod volumes and "Recursive" for all other volumes.

This field affects only Pods that have SELinux label set, either in PodSecurityContext or in SecurityContext of all containers.

All Pods that use the same volume should use the same seLinuxChangePolicy, otherwise some pods can get stuck in ContainerCreating state. Note that this field cannot be set when spec.os.name is windows. +featureGate=SELinuxChangePolicy +optional |






<a name="k8s-io-api-core-v1-PodSignature"></a>

### PodSignature
Describes the class of pods that should avoid this node.
Exactly one field should be set.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| podController | [k8s.io.apimachinery.pkg.apis.meta.v1.OwnerReference](#k8s-io-apimachinery-pkg-apis-meta-v1-OwnerReference) | optional | Reference to controller whose pods should avoid this node. +optional |






<a name="k8s-io-api-core-v1-PodSpec"></a>

### PodSpec
PodSpec is a description of a pod.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| volumes | [Volume](#k8s-io-api-core-v1-Volume) | repeated | List of volumes that can be mounted by containers belonging to the pod. More info: https://kubernetes.io/docs/concepts/storage/volumes +optional +patchMergeKey=name +patchStrategy=merge,retainKeys +listType=map +listMapKey=name |
| initContainers | [Container](#k8s-io-api-core-v1-Container) | repeated | List of initialization containers belonging to the pod. Init containers are executed in order prior to containers being started. If any init container fails, the pod is considered to have failed and is handled according to its restartPolicy. The name for an init container or normal container must be unique among all containers. Init containers may not have Lifecycle actions, Readiness probes, Liveness probes, or Startup probes. The resourceRequirements of an init container are taken into account during scheduling by finding the highest request/limit for each resource type, and then using the max of that value or the sum of the normal containers. Limits are applied to init containers in a similar fashion. Init containers cannot currently be added or removed. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/init-containers/ +patchMergeKey=name +patchStrategy=merge +listType=map +listMapKey=name |
| containers | [Container](#k8s-io-api-core-v1-Container) | repeated | List of containers belonging to the pod. Containers cannot currently be added or removed. There must be at least one container in a Pod. Cannot be updated. +patchMergeKey=name +patchStrategy=merge +listType=map +listMapKey=name |
| ephemeralContainers | [EphemeralContainer](#k8s-io-api-core-v1-EphemeralContainer) | repeated | List of ephemeral containers run in this pod. Ephemeral containers may be run in an existing pod to perform user-initiated actions such as debugging. This list cannot be specified when creating a pod, and it cannot be modified by updating the pod spec. In order to add an ephemeral container to an existing pod, use the pod's ephemeralcontainers subresource. +optional +patchMergeKey=name +patchStrategy=merge +listType=map +listMapKey=name |
| restartPolicy | [string](#string) | optional | Restart policy for all containers within the pod. One of Always, OnFailure, Never. In some contexts, only a subset of those values may be permitted. Default to Always. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#restart-policy +optional |
| terminationGracePeriodSeconds | [int64](#int64) | optional | Optional duration in seconds the pod needs to terminate gracefully. May be decreased in delete request. Value must be non-negative integer. The value zero indicates stop immediately via the kill signal (no opportunity to shut down). If this value is nil, the default grace period will be used instead. The grace period is the duration in seconds after the processes running in the pod are sent a termination signal and the time when the processes are forcibly halted with a kill signal. Set this value longer than the expected cleanup time for your process. Defaults to 30 seconds. +optional |
| activeDeadlineSeconds | [int64](#int64) | optional | Optional duration in seconds the pod may be active on the node relative to StartTime before the system will actively try to mark it failed and kill associated containers. Value must be a positive integer. +optional |
| dnsPolicy | [string](#string) | optional | Set DNS policy for the pod. Defaults to "ClusterFirst". Valid values are 'ClusterFirstWithHostNet', 'ClusterFirst', 'Default' or 'None'. DNS parameters given in DNSConfig will be merged with the policy selected with DNSPolicy. To have DNS options set along with hostNetwork, you have to specify DNS policy explicitly to 'ClusterFirstWithHostNet'. +optional |
| nodeSelector | [PodSpec.NodeSelectorEntry](#k8s-io-api-core-v1-PodSpec-NodeSelectorEntry) | repeated | NodeSelector is a selector which must be true for the pod to fit on a node. Selector which must match a node's labels for the pod to be scheduled on that node. More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/ +optional +mapType=atomic |
| serviceAccountName | [string](#string) | optional | ServiceAccountName is the name of the ServiceAccount to use to run this pod. More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/ +optional |
| serviceAccount | [string](#string) | optional | DeprecatedServiceAccount is a deprecated alias for ServiceAccountName. Deprecated: Use serviceAccountName instead. +k8s:conversion-gen=false +optional |
| automountServiceAccountToken | [bool](#bool) | optional | AutomountServiceAccountToken indicates whether a service account token should be automatically mounted. +optional |
| nodeName | [string](#string) | optional | NodeName indicates in which node this pod is scheduled. If empty, this pod is a candidate for scheduling by the scheduler defined in schedulerName. Once this field is set, the kubelet for this node becomes responsible for the lifecycle of this pod. This field should not be used to express a desire for the pod to be scheduled on a specific node. https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#nodename +optional |
| hostNetwork | [bool](#bool) | optional | Host networking requested for this pod. Use the host's network namespace. If this option is set, the ports that will be used must be specified. Default to false. +k8s:conversion-gen=false +optional |
| hostPID | [bool](#bool) | optional | Use the host's pid namespace. Optional: Default to false. +k8s:conversion-gen=false +optional |
| hostIPC | [bool](#bool) | optional | Use the host's ipc namespace. Optional: Default to false. +k8s:conversion-gen=false +optional |
| shareProcessNamespace | [bool](#bool) | optional | Share a single process namespace between all of the containers in a pod. When this is set containers will be able to view and signal processes from other containers in the same pod, and the first process in each container will not be assigned PID 1. HostPID and ShareProcessNamespace cannot both be set. Optional: Default to false. +k8s:conversion-gen=false +optional |
| securityContext | [PodSecurityContext](#k8s-io-api-core-v1-PodSecurityContext) | optional | SecurityContext holds pod-level security attributes and common container settings. Optional: Defaults to empty. See type description for default values of each field. +optional |
| imagePullSecrets | [LocalObjectReference](#k8s-io-api-core-v1-LocalObjectReference) | repeated | ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec. If specified, these secrets will be passed to individual puller implementations for them to use. More info: https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod +optional +patchMergeKey=name +patchStrategy=merge +listType=map +listMapKey=name |
| hostname | [string](#string) | optional | Specifies the hostname of the Pod If not specified, the pod's hostname will be set to a system-defined value. +optional |
| subdomain | [string](#string) | optional | If specified, the fully qualified Pod hostname will be "<hostname>.<subdomain>.<pod namespace>.svc.<cluster domain>". If not specified, the pod will not have a domainname at all. +optional |
| affinity | [Affinity](#k8s-io-api-core-v1-Affinity) | optional | If specified, the pod's scheduling constraints +optional |
| schedulerName | [string](#string) | optional | If specified, the pod will be dispatched by specified scheduler. If not specified, the pod will be dispatched by default scheduler. +optional |
| tolerations | [Toleration](#k8s-io-api-core-v1-Toleration) | repeated | If specified, the pod's tolerations. +optional +listType=atomic |
| hostAliases | [HostAlias](#k8s-io-api-core-v1-HostAlias) | repeated | HostAliases is an optional list of hosts and IPs that will be injected into the pod's hosts file if specified. +optional +patchMergeKey=ip +patchStrategy=merge +listType=map +listMapKey=ip |
| priorityClassName | [string](#string) | optional | If specified, indicates the pod's priority. "system-node-critical" and "system-cluster-critical" are two special keywords which indicate the highest priorities with the former being the highest priority. Any other name must be defined by creating a PriorityClass object with that name. If not specified, the pod priority will be default or zero if there is no default. +optional |
| priority | [int32](#int32) | optional | The priority value. Various system components use this field to find the priority of the pod. When Priority Admission Controller is enabled, it prevents users from setting this field. The admission controller populates this field from PriorityClassName. The higher the value, the higher the priority. +optional |
| dnsConfig | [PodDNSConfig](#k8s-io-api-core-v1-PodDNSConfig) | optional | Specifies the DNS parameters of a pod. Parameters specified here will be merged to the generated DNS configuration based on DNSPolicy. +optional |
| readinessGates | [PodReadinessGate](#k8s-io-api-core-v1-PodReadinessGate) | repeated | If specified, all readiness gates will be evaluated for pod readiness. A pod is ready when all its containers are ready AND all conditions specified in the readiness gates have status equal to "True" More info: https://git.k8s.io/enhancements/keps/sig-network/580-pod-readiness-gates +optional +listType=atomic |
| runtimeClassName | [string](#string) | optional | RuntimeClassName refers to a RuntimeClass object in the node.k8s.io group, which should be used to run this pod. If no RuntimeClass resource matches the named class, the pod will not be run. If unset or empty, the "legacy" RuntimeClass will be used, which is an implicit class with an empty definition that uses the default runtime handler. More info: https://git.k8s.io/enhancements/keps/sig-node/585-runtime-class +optional |
| enableServiceLinks | [bool](#bool) | optional | EnableServiceLinks indicates whether information about services should be injected into pod's environment variables, matching the syntax of Docker links. Optional: Defaults to true. +optional |
| preemptionPolicy | [string](#string) | optional | PreemptionPolicy is the Policy for preempting pods with lower priority. One of Never, PreemptLowerPriority. Defaults to PreemptLowerPriority if unset. +optional |
| overhead | [PodSpec.OverheadEntry](#k8s-io-api-core-v1-PodSpec-OverheadEntry) | repeated | Overhead represents the resource overhead associated with running a pod for a given RuntimeClass. This field will be autopopulated at admission time by the RuntimeClass admission controller. If the RuntimeClass admission controller is enabled, overhead must not be set in Pod create requests. The RuntimeClass admission controller will reject Pod create requests which have the overhead already set. If RuntimeClass is configured and selected in the PodSpec, Overhead will be set to the value defined in the corresponding RuntimeClass, otherwise it will remain unset and treated as zero. More info: https://git.k8s.io/enhancements/keps/sig-node/688-pod-overhead/README.md +optional |
| topologySpreadConstraints | [TopologySpreadConstraint](#k8s-io-api-core-v1-TopologySpreadConstraint) | repeated | TopologySpreadConstraints describes how a group of pods ought to spread across topology domains. Scheduler will schedule pods in a way which abides by the constraints. All topologySpreadConstraints are ANDed. +optional +patchMergeKey=topologyKey +patchStrategy=merge +listType=map +listMapKey=topologyKey +listMapKey=whenUnsatisfiable |
| setHostnameAsFQDN | [bool](#bool) | optional | If true the pod's hostname will be configured as the pod's FQDN, rather than the leaf name (the default). In Linux containers, this means setting the FQDN in the hostname field of the kernel (the nodename field of struct utsname). In Windows containers, this means setting the registry value of hostname for the registry key HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Services\\Tcpip\\Parameters to FQDN. If a pod does not have FQDN, this has no effect. Default to false. +optional |
| os | [PodOS](#k8s-io-api-core-v1-PodOS) | optional | Specifies the OS of the containers in the pod. Some pod and container fields are restricted if this is set.

If the OS field is set to linux, the following fields must be unset: -securityContext.windowsOptions

If the OS field is set to windows, following fields must be unset: - spec.hostPID - spec.hostIPC - spec.hostUsers - spec.securityContext.appArmorProfile - spec.securityContext.seLinuxOptions - spec.securityContext.seccompProfile - spec.securityContext.fsGroup - spec.securityContext.fsGroupChangePolicy - spec.securityContext.sysctls - spec.shareProcessNamespace - spec.securityContext.runAsUser - spec.securityContext.runAsGroup - spec.securityContext.supplementalGroups - spec.securityContext.supplementalGroupsPolicy - spec.containers[*].securityContext.appArmorProfile - spec.containers[*].securityContext.seLinuxOptions - spec.containers[*].securityContext.seccompProfile - spec.containers[*].securityContext.capabilities - spec.containers[*].securityContext.readOnlyRootFilesystem - spec.containers[*].securityContext.privileged - spec.containers[*].securityContext.allowPrivilegeEscalation - spec.containers[*].securityContext.procMount - spec.containers[*].securityContext.runAsUser - spec.containers[*].securityContext.runAsGroup +optional |
| hostUsers | [bool](#bool) | optional | Use the host's user namespace. Optional: Default to true. If set to true or not present, the pod will be run in the host user namespace, useful for when the pod needs a feature only available to the host user namespace, such as loading a kernel module with CAP_SYS_MODULE. When set to false, a new userns is created for the pod. Setting false is useful for mitigating container breakout vulnerabilities even allowing users to run their containers as root without actually having root privileges on the host. This field is alpha-level and is only honored by servers that enable the UserNamespacesSupport feature. +k8s:conversion-gen=false +optional |
| schedulingGates | [PodSchedulingGate](#k8s-io-api-core-v1-PodSchedulingGate) | repeated | SchedulingGates is an opaque list of values that if specified will block scheduling the pod. If schedulingGates is not empty, the pod will stay in the SchedulingGated state and the scheduler will not attempt to schedule the pod.

SchedulingGates can only be set at pod creation time, and be removed only afterwards.

+patchMergeKey=name +patchStrategy=merge +listType=map +listMapKey=name +optional |
| resourceClaims | [PodResourceClaim](#k8s-io-api-core-v1-PodResourceClaim) | repeated | ResourceClaims defines which ResourceClaims must be allocated and reserved before the Pod is allowed to start. The resources will be made available to those containers which consume them by name.

This is an alpha field and requires enabling the DynamicResourceAllocation feature gate.

This field is immutable.

+patchMergeKey=name +patchStrategy=merge,retainKeys +listType=map +listMapKey=name +featureGate=DynamicResourceAllocation +optional |
| resources | [ResourceRequirements](#k8s-io-api-core-v1-ResourceRequirements) | optional | Resources is the total amount of CPU and Memory resources required by all containers in the pod. It supports specifying Requests and Limits for "cpu" and "memory" resource names only. ResourceClaims are not supported.

This field enables fine-grained control over resource allocation for the entire pod, allowing resource sharing among containers in a pod. TODO: For beta graduation, expand this comment with a detailed explanation.

This is an alpha field and requires enabling the PodLevelResources feature gate.

+featureGate=PodLevelResources +optional |






<a name="k8s-io-api-core-v1-PodSpec-NodeSelectorEntry"></a>

### PodSpec.NodeSelectorEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [string](#string) | optional |  |






<a name="k8s-io-api-core-v1-PodSpec-OverheadEntry"></a>

### PodSpec.OverheadEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [k8s.io.apimachinery.pkg.api.resource.Quantity](#k8s-io-apimachinery-pkg-api-resource-Quantity) | optional |  |






<a name="k8s-io-api-core-v1-PodStatus"></a>

### PodStatus
PodStatus represents information about the status of a pod. Status may trail the actual
state of a system, especially if the node that hosts the pod cannot contact the control
plane.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| observedGeneration | [int64](#int64) | optional | If set, this represents the .metadata.generation that the pod status was set based upon. This is an alpha field. Enable PodObservedGenerationTracking to be able to use this field. +featureGate=PodObservedGenerationTracking +optional |
| phase | [string](#string) | optional | The phase of a Pod is a simple, high-level summary of where the Pod is in its lifecycle. The conditions array, the reason and message fields, and the individual container status arrays contain more detail about the pod's status. There are five possible phase values:

Pending: The pod has been accepted by the Kubernetes system, but one or more of the container images has not been created. This includes time before being scheduled as well as time spent downloading images over the network, which could take a while. Running: The pod has been bound to a node, and all of the containers have been created. At least one container is still running, or is in the process of starting or restarting. Succeeded: All containers in the pod have terminated in success, and will not be restarted. Failed: All containers in the pod have terminated, and at least one container has terminated in failure. The container either exited with non-zero status or was terminated by the system. Unknown: For some reason the state of the pod could not be obtained, typically due to an error in communicating with the host of the pod.

More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#pod-phase +optional |
| conditions | [PodCondition](#k8s-io-api-core-v1-PodCondition) | repeated | Current service state of pod. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#pod-conditions +optional +patchMergeKey=type +patchStrategy=merge +listType=map +listMapKey=type |
| message | [string](#string) | optional | A human readable message indicating details about why the pod is in this condition. +optional |
| reason | [string](#string) | optional | A brief CamelCase message indicating details about why the pod is in this state. e.g. 'Evicted' +optional |
| nominatedNodeName | [string](#string) | optional | nominatedNodeName is set only when this pod preempts other pods on the node, but it cannot be scheduled right away as preemption victims receive their graceful termination periods. This field does not guarantee that the pod will be scheduled on this node. Scheduler may decide to place the pod elsewhere if other nodes become available sooner. Scheduler may also decide to give the resources on this node to a higher priority pod that is created after preemption. As a result, this field may be different than PodSpec.nodeName when the pod is scheduled. +optional |
| hostIP | [string](#string) | optional | hostIP holds the IP address of the host to which the pod is assigned. Empty if the pod has not started yet. A pod can be assigned to a node that has a problem in kubelet which in turns mean that HostIP will not be updated even if there is a node is assigned to pod +optional |
| hostIPs | [HostIP](#k8s-io-api-core-v1-HostIP) | repeated | hostIPs holds the IP addresses allocated to the host. If this field is specified, the first entry must match the hostIP field. This list is empty if the pod has not started yet. A pod can be assigned to a node that has a problem in kubelet which in turns means that HostIPs will not be updated even if there is a node is assigned to this pod. +optional +patchStrategy=merge +patchMergeKey=ip +listType=atomic |
| podIP | [string](#string) | optional | podIP address allocated to the pod. Routable at least within the cluster. Empty if not yet allocated. +optional |
| podIPs | [PodIP](#k8s-io-api-core-v1-PodIP) | repeated | podIPs holds the IP addresses allocated to the pod. If this field is specified, the 0th entry must match the podIP field. Pods may be allocated at most 1 value for each of IPv4 and IPv6. This list is empty if no IPs have been allocated yet. +optional +patchStrategy=merge +patchMergeKey=ip +listType=map +listMapKey=ip |
| startTime | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | RFC 3339 date and time at which the object was acknowledged by the Kubelet. This is before the Kubelet pulled the container image(s) for the pod. +optional |
| initContainerStatuses | [ContainerStatus](#k8s-io-api-core-v1-ContainerStatus) | repeated | Statuses of init containers in this pod. The most recent successful non-restartable init container will have ready = true, the most recently started container will have startTime set. Each init container in the pod should have at most one status in this list, and all statuses should be for containers in the pod. However this is not enforced. If a status for a non-existent container is present in the list, or the list has duplicate names, the behavior of various Kubernetes components is not defined and those statuses might be ignored. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-and-container-status +listType=atomic |
| containerStatuses | [ContainerStatus](#k8s-io-api-core-v1-ContainerStatus) | repeated | Statuses of containers in this pod. Each container in the pod should have at most one status in this list, and all statuses should be for containers in the pod. However this is not enforced. If a status for a non-existent container is present in the list, or the list has duplicate names, the behavior of various Kubernetes components is not defined and those statuses might be ignored. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#pod-and-container-status +optional +listType=atomic |
| qosClass | [string](#string) | optional | The Quality of Service (QOS) classification assigned to the pod based on resource requirements See PodQOSClass type for available QOS classes More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-qos/#quality-of-service-classes +optional |
| ephemeralContainerStatuses | [ContainerStatus](#k8s-io-api-core-v1-ContainerStatus) | repeated | Statuses for any ephemeral containers that have run in this pod. Each ephemeral container in the pod should have at most one status in this list, and all statuses should be for containers in the pod. However this is not enforced. If a status for a non-existent container is present in the list, or the list has duplicate names, the behavior of various Kubernetes components is not defined and those statuses might be ignored. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#pod-and-container-status +optional +listType=atomic |
| resize | [string](#string) | optional | Status of resources resize desired for pod's containers. It is empty if no resources resize is pending. Any changes to container resources will automatically set this to "Proposed" Deprecated: Resize status is moved to two pod conditions PodResizePending and PodResizeInProgress. PodResizePending will track states where the spec has been resized, but the Kubelet has not yet allocated the resources. PodResizeInProgress will track in-progress resizes, and should be present whenever allocated resources != acknowledged resources. +featureGate=InPlacePodVerticalScaling +optional |
| resourceClaimStatuses | [PodResourceClaimStatus](#k8s-io-api-core-v1-PodResourceClaimStatus) | repeated | Status of resource claims. +patchMergeKey=name +patchStrategy=merge,retainKeys +listType=map +listMapKey=name +featureGate=DynamicResourceAllocation +optional |






<a name="k8s-io-api-core-v1-PodStatusResult"></a>

### PodStatusResult
PodStatusResult is a wrapper for PodStatus returned by kubelet that can be encode/decoded


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional | Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional |
| status | [PodStatus](#k8s-io-api-core-v1-PodStatus) | optional | Most recently observed status of the pod. This data may not be up to date. Populated by the system. Read-only. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status +optional |






<a name="k8s-io-api-core-v1-PodTemplate"></a>

### PodTemplate
PodTemplate describes a template for creating copies of a predefined pod.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional | Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional |
| template | [PodTemplateSpec](#k8s-io-api-core-v1-PodTemplateSpec) | optional | Template defines the pods that will be created from this pod template. https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status +optional |






<a name="k8s-io-api-core-v1-PodTemplateList"></a>

### PodTemplateList
PodTemplateList is a list of PodTemplates.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional | Standard list metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds +optional |
| items | [PodTemplate](#k8s-io-api-core-v1-PodTemplate) | repeated | List of pod templates |






<a name="k8s-io-api-core-v1-PodTemplateSpec"></a>

### PodTemplateSpec
PodTemplateSpec describes the data a pod should have when created from a template


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional | Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional |
| spec | [PodSpec](#k8s-io-api-core-v1-PodSpec) | optional | Specification of the desired behavior of the pod. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status +optional |






<a name="k8s-io-api-core-v1-PortStatus"></a>

### PortStatus
PortStatus represents the error condition of a service port


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| port | [int32](#int32) | optional | Port is the port number of the service port of which status is recorded here |
| protocol | [string](#string) | optional | Protocol is the protocol of the service port of which status is recorded here The supported values are: "TCP", "UDP", "SCTP" |
| error | [string](#string) | optional | Error is to record the problem with the service port The format of the error shall comply with the following rules: - built-in error values shall be specified in this file and those shall use CamelCase names - cloud provider specific error values must have names that comply with the format foo.example.com/CamelCase. --- The regex it matches is (dns1123SubdomainFmt/)?(qualifiedNameFmt) +optional +kubebuilder:validation:Required +kubebuilder:validation:Pattern=`^([a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*/)?(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])$` +kubebuilder:validation:MaxLength=316 |






<a name="k8s-io-api-core-v1-PortworxVolumeSource"></a>

### PortworxVolumeSource
PortworxVolumeSource represents a Portworx volume resource.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| volumeID | [string](#string) | optional | volumeID uniquely identifies a Portworx volume |
| fsType | [string](#string) | optional | fSType represents the filesystem type to mount Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs". Implicitly inferred to be "ext4" if unspecified. |
| readOnly | [bool](#bool) | optional | readOnly defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. +optional |






<a name="k8s-io-api-core-v1-Preconditions"></a>

### Preconditions
Preconditions must be fulfilled before an operation (update, delete, etc.) is carried out.
+k8s:openapi-gen=false


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| uid | [string](#string) | optional | Specifies the target UID. +optional |






<a name="k8s-io-api-core-v1-PreferAvoidPodsEntry"></a>

### PreferAvoidPodsEntry
Describes a class of pods that should avoid this node.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| podSignature | [PodSignature](#k8s-io-api-core-v1-PodSignature) | optional | The class of pods. |
| evictionTime | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | Time at which this entry was added to the list. +optional |
| reason | [string](#string) | optional | (brief) reason why this entry was added to the list. +optional |
| message | [string](#string) | optional | Human readable message indicating why this entry was added to the list. +optional |






<a name="k8s-io-api-core-v1-PreferredSchedulingTerm"></a>

### PreferredSchedulingTerm
An empty preferred scheduling term matches all objects with implicit weight 0
(i.e. it's a no-op). A null preferred scheduling term matches no objects (i.e. is also a no-op).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| weight | [int32](#int32) | optional | Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100. |
| preference | [NodeSelectorTerm](#k8s-io-api-core-v1-NodeSelectorTerm) | optional | A node selector term, associated with the corresponding weight. |






<a name="k8s-io-api-core-v1-Probe"></a>

### Probe
Probe describes a health check to be performed against a container to determine whether it is
alive or ready to receive traffic.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| handler | [ProbeHandler](#k8s-io-api-core-v1-ProbeHandler) | optional | The action taken to determine the health of a container |
| initialDelaySeconds | [int32](#int32) | optional | Number of seconds after the container has started before liveness probes are initiated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes +optional |
| timeoutSeconds | [int32](#int32) | optional | Number of seconds after which the probe times out. Defaults to 1 second. Minimum value is 1. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes +optional |
| periodSeconds | [int32](#int32) | optional | How often (in seconds) to perform the probe. Default to 10 seconds. Minimum value is 1. +optional |
| successThreshold | [int32](#int32) | optional | Minimum consecutive successes for the probe to be considered successful after having failed. Defaults to 1. Must be 1 for liveness and startup. Minimum value is 1. +optional |
| failureThreshold | [int32](#int32) | optional | Minimum consecutive failures for the probe to be considered failed after having succeeded. Defaults to 3. Minimum value is 1. +optional |
| terminationGracePeriodSeconds | [int64](#int64) | optional | Optional duration in seconds the pod needs to terminate gracefully upon probe failure. The grace period is the duration in seconds after the processes running in the pod are sent a termination signal and the time when the processes are forcibly halted with a kill signal. Set this value longer than the expected cleanup time for your process. If this value is nil, the pod's terminationGracePeriodSeconds will be used. Otherwise, this value overrides the value provided by the pod spec. Value must be non-negative integer. The value zero indicates stop immediately via the kill signal (no opportunity to shut down). This is a beta field and requires enabling ProbeTerminationGracePeriod feature gate. Minimum value is 1. spec.terminationGracePeriodSeconds is used if unset. +optional |






<a name="k8s-io-api-core-v1-ProbeHandler"></a>

### ProbeHandler
ProbeHandler defines a specific action that should be taken in a probe.
One and only one of the fields must be specified.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| exec | [ExecAction](#k8s-io-api-core-v1-ExecAction) | optional | Exec specifies a command to execute in the container. +optional |
| httpGet | [HTTPGetAction](#k8s-io-api-core-v1-HTTPGetAction) | optional | HTTPGet specifies an HTTP GET request to perform. +optional |
| tcpSocket | [TCPSocketAction](#k8s-io-api-core-v1-TCPSocketAction) | optional | TCPSocket specifies a connection to a TCP port. +optional |
| grpc | [GRPCAction](#k8s-io-api-core-v1-GRPCAction) | optional | GRPC specifies a GRPC HealthCheckRequest. +optional |






<a name="k8s-io-api-core-v1-ProjectedVolumeSource"></a>

### ProjectedVolumeSource
Represents a projected volume source


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sources | [VolumeProjection](#k8s-io-api-core-v1-VolumeProjection) | repeated | sources is the list of volume projections. Each entry in this list handles one source. +optional +listType=atomic |
| defaultMode | [int32](#int32) | optional | defaultMode are the mode bits used to set permissions on created files by default. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set. +optional |






<a name="k8s-io-api-core-v1-QuobyteVolumeSource"></a>

### QuobyteVolumeSource
Represents a Quobyte mount that lasts the lifetime of a pod.
Quobyte volumes do not support ownership management or SELinux relabeling.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| registry | [string](#string) | optional | registry represents a single or multiple Quobyte Registry services specified as a string as host:port pair (multiple entries are separated with commas) which acts as the central registry for volumes |
| volume | [string](#string) | optional | volume is a string that references an already created Quobyte volume by name. |
| readOnly | [bool](#bool) | optional | readOnly here will force the Quobyte volume to be mounted with read-only permissions. Defaults to false. +optional |
| user | [string](#string) | optional | user to map volume access to Defaults to serivceaccount user +optional |
| group | [string](#string) | optional | group to map volume access to Default is no group +optional |
| tenant | [string](#string) | optional | tenant owning the given Quobyte volume in the Backend Used with dynamically provisioned Quobyte volumes, value is set by the plugin +optional |






<a name="k8s-io-api-core-v1-RBDPersistentVolumeSource"></a>

### RBDPersistentVolumeSource
Represents a Rados Block Device mount that lasts the lifetime of a pod.
RBD volumes support ownership management and SELinux relabeling.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| monitors | [string](#string) | repeated | monitors is a collection of Ceph monitors. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it +listType=atomic |
| image | [string](#string) | optional | image is the rados image name. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it |
| fsType | [string](#string) | optional | fsType is the filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. More info: https://kubernetes.io/docs/concepts/storage/volumes#rbd TODO: how do we prevent errors in the filesystem from compromising the machine +optional |
| pool | [string](#string) | optional | pool is the rados pool name. Default is rbd. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it +optional +default="rbd" |
| user | [string](#string) | optional | user is the rados user name. Default is admin. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it +optional +default="admin" |
| keyring | [string](#string) | optional | keyring is the path to key ring for RBDUser. Default is /etc/ceph/keyring. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it +optional +default="/etc/ceph/keyring" |
| secretRef | [SecretReference](#k8s-io-api-core-v1-SecretReference) | optional | secretRef is name of the authentication secret for RBDUser. If provided overrides keyring. Default is nil. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it +optional |
| readOnly | [bool](#bool) | optional | readOnly here will force the ReadOnly setting in VolumeMounts. Defaults to false. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it +optional |






<a name="k8s-io-api-core-v1-RBDVolumeSource"></a>

### RBDVolumeSource
Represents a Rados Block Device mount that lasts the lifetime of a pod.
RBD volumes support ownership management and SELinux relabeling.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| monitors | [string](#string) | repeated | monitors is a collection of Ceph monitors. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it +listType=atomic |
| image | [string](#string) | optional | image is the rados image name. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it |
| fsType | [string](#string) | optional | fsType is the filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. More info: https://kubernetes.io/docs/concepts/storage/volumes#rbd TODO: how do we prevent errors in the filesystem from compromising the machine +optional |
| pool | [string](#string) | optional | pool is the rados pool name. Default is rbd. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it +optional +default="rbd" |
| user | [string](#string) | optional | user is the rados user name. Default is admin. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it +optional +default="admin" |
| keyring | [string](#string) | optional | keyring is the path to key ring for RBDUser. Default is /etc/ceph/keyring. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it +optional +default="/etc/ceph/keyring" |
| secretRef | [LocalObjectReference](#k8s-io-api-core-v1-LocalObjectReference) | optional | secretRef is name of the authentication secret for RBDUser. If provided overrides keyring. Default is nil. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it +optional |
| readOnly | [bool](#bool) | optional | readOnly here will force the ReadOnly setting in VolumeMounts. Defaults to false. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it +optional |






<a name="k8s-io-api-core-v1-RangeAllocation"></a>

### RangeAllocation
RangeAllocation is not a public type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional | Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional |
| range | [string](#string) | optional | Range is string that identifies the range represented by 'data'. |
| data | [bytes](#bytes) | optional | Data is a bit array containing all allocated addresses in the previous segment. |






<a name="k8s-io-api-core-v1-ReplicationController"></a>

### ReplicationController
ReplicationController represents the configuration of a replication controller.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional | If the Labels of a ReplicationController are empty, they are defaulted to be the same as the Pod(s) that the replication controller manages. Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional |
| spec | [ReplicationControllerSpec](#k8s-io-api-core-v1-ReplicationControllerSpec) | optional | Spec defines the specification of the desired behavior of the replication controller. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status +optional |
| status | [ReplicationControllerStatus](#k8s-io-api-core-v1-ReplicationControllerStatus) | optional | Status is the most recently observed status of the replication controller. This data may be out of date by some window of time. Populated by the system. Read-only. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status +optional |






<a name="k8s-io-api-core-v1-ReplicationControllerCondition"></a>

### ReplicationControllerCondition
ReplicationControllerCondition describes the state of a replication controller at a certain point.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) | optional | Type of replication controller condition. |
| status | [string](#string) | optional | Status of the condition, one of True, False, Unknown. |
| lastTransitionTime | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | The last time the condition transitioned from one status to another. +optional |
| reason | [string](#string) | optional | The reason for the condition's last transition. +optional |
| message | [string](#string) | optional | A human readable message indicating details about the transition. +optional |






<a name="k8s-io-api-core-v1-ReplicationControllerList"></a>

### ReplicationControllerList
ReplicationControllerList is a collection of replication controllers.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional | Standard list metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds +optional |
| items | [ReplicationController](#k8s-io-api-core-v1-ReplicationController) | repeated | List of replication controllers. More info: https://kubernetes.io/docs/concepts/workloads/controllers/replicationcontroller |






<a name="k8s-io-api-core-v1-ReplicationControllerSpec"></a>

### ReplicationControllerSpec
ReplicationControllerSpec is the specification of a replication controller.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| replicas | [int32](#int32) | optional | Replicas is the number of desired replicas. This is a pointer to distinguish between explicit zero and unspecified. Defaults to 1. More info: https://kubernetes.io/docs/concepts/workloads/controllers/replicationcontroller#what-is-a-replicationcontroller +optional +k8s:optional +default=1 +k8s:minimum=0 |
| minReadySeconds | [int32](#int32) | optional | Minimum number of seconds for which a newly created pod should be ready without any of its container crashing, for it to be considered available. Defaults to 0 (pod will be considered available as soon as it is ready) +optional +k8s:optional +default=0 +k8s:minimum=0 |
| selector | [ReplicationControllerSpec.SelectorEntry](#k8s-io-api-core-v1-ReplicationControllerSpec-SelectorEntry) | repeated | Selector is a label query over pods that should match the Replicas count. If Selector is empty, it is defaulted to the labels present on the Pod template. Label keys and values that must match in order to be controlled by this replication controller, if empty defaulted to labels on Pod template. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors +optional +mapType=atomic |
| template | [PodTemplateSpec](#k8s-io-api-core-v1-PodTemplateSpec) | optional | Template is the object that describes the pod that will be created if insufficient replicas are detected. This takes precedence over a TemplateRef. The only allowed template.spec.restartPolicy value is "Always". More info: https://kubernetes.io/docs/concepts/workloads/controllers/replicationcontroller#pod-template +optional |






<a name="k8s-io-api-core-v1-ReplicationControllerSpec-SelectorEntry"></a>

### ReplicationControllerSpec.SelectorEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [string](#string) | optional |  |






<a name="k8s-io-api-core-v1-ReplicationControllerStatus"></a>

### ReplicationControllerStatus
ReplicationControllerStatus represents the current status of a replication
controller.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| replicas | [int32](#int32) | optional | Replicas is the most recently observed number of replicas. More info: https://kubernetes.io/docs/concepts/workloads/controllers/replicationcontroller#what-is-a-replicationcontroller |
| fullyLabeledReplicas | [int32](#int32) | optional | The number of pods that have labels matching the labels of the pod template of the replication controller. +optional |
| readyReplicas | [int32](#int32) | optional | The number of ready replicas for this replication controller. +optional |
| availableReplicas | [int32](#int32) | optional | The number of available replicas (ready for at least minReadySeconds) for this replication controller. +optional |
| observedGeneration | [int64](#int64) | optional | ObservedGeneration reflects the generation of the most recently observed replication controller. +optional |
| conditions | [ReplicationControllerCondition](#k8s-io-api-core-v1-ReplicationControllerCondition) | repeated | Represents the latest available observations of a replication controller's current state. +optional +patchMergeKey=type +patchStrategy=merge +listType=map +listMapKey=type |






<a name="k8s-io-api-core-v1-ResourceClaim"></a>

### ResourceClaim
ResourceClaim references one entry in PodSpec.ResourceClaims.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name must match the name of one entry in pod.spec.resourceClaims of the Pod where this field is used. It makes that resource available inside a container. |
| request | [string](#string) | optional | Request is the name chosen for a request in the referenced claim. If empty, everything from the claim is made available, otherwise only the result of this request.

+optional |






<a name="k8s-io-api-core-v1-ResourceFieldSelector"></a>

### ResourceFieldSelector
ResourceFieldSelector represents container resources (cpu, memory) and their output format
+structType=atomic


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| containerName | [string](#string) | optional | Container name: required for volumes, optional for env vars +optional |
| resource | [string](#string) | optional | Required: resource to select |
| divisor | [k8s.io.apimachinery.pkg.api.resource.Quantity](#k8s-io-apimachinery-pkg-api-resource-Quantity) | optional | Specifies the output format of the exposed resources, defaults to "1" +optional |






<a name="k8s-io-api-core-v1-ResourceHealth"></a>

### ResourceHealth
ResourceHealth represents the health of a resource. It has the latest device health information.
This is a part of KEP https://kep.k8s.io/4680.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resourceID | [string](#string) | optional | ResourceID is the unique identifier of the resource. See the ResourceID type for more information. |
| health | [string](#string) | optional | Health of the resource. can be one of: - Healthy: operates as normal - Unhealthy: reported unhealthy. We consider this a temporary health issue since we do not have a mechanism today to distinguish temporary and permanent issues. - Unknown: The status cannot be determined. For example, Device Plugin got unregistered and hasn't been re-registered since.

In future we may want to introduce the PermanentlyUnhealthy Status. |






<a name="k8s-io-api-core-v1-ResourceQuota"></a>

### ResourceQuota
ResourceQuota sets aggregate quota restrictions enforced per namespace


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional | Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional |
| spec | [ResourceQuotaSpec](#k8s-io-api-core-v1-ResourceQuotaSpec) | optional | Spec defines the desired quota. https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status +optional |
| status | [ResourceQuotaStatus](#k8s-io-api-core-v1-ResourceQuotaStatus) | optional | Status defines the actual enforced quota and its current usage. https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status +optional |






<a name="k8s-io-api-core-v1-ResourceQuotaList"></a>

### ResourceQuotaList
ResourceQuotaList is a list of ResourceQuota items.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional | Standard list metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds +optional |
| items | [ResourceQuota](#k8s-io-api-core-v1-ResourceQuota) | repeated | Items is a list of ResourceQuota objects. More info: https://kubernetes.io/docs/concepts/policy/resource-quotas/ |






<a name="k8s-io-api-core-v1-ResourceQuotaSpec"></a>

### ResourceQuotaSpec
ResourceQuotaSpec defines the desired hard limits to enforce for Quota.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| hard | [ResourceQuotaSpec.HardEntry](#k8s-io-api-core-v1-ResourceQuotaSpec-HardEntry) | repeated | hard is the set of desired hard limits for each named resource. More info: https://kubernetes.io/docs/concepts/policy/resource-quotas/ +optional |
| scopes | [string](#string) | repeated | A collection of filters that must match each object tracked by a quota. If not specified, the quota matches all objects. +optional +listType=atomic |
| scopeSelector | [ScopeSelector](#k8s-io-api-core-v1-ScopeSelector) | optional | scopeSelector is also a collection of filters like scopes that must match each object tracked by a quota but expressed using ScopeSelectorOperator in combination with possible values. For a resource to match, both scopes AND scopeSelector (if specified in spec), must be matched. +optional |






<a name="k8s-io-api-core-v1-ResourceQuotaSpec-HardEntry"></a>

### ResourceQuotaSpec.HardEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [k8s.io.apimachinery.pkg.api.resource.Quantity](#k8s-io-apimachinery-pkg-api-resource-Quantity) | optional |  |






<a name="k8s-io-api-core-v1-ResourceQuotaStatus"></a>

### ResourceQuotaStatus
ResourceQuotaStatus defines the enforced hard limits and observed use.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| hard | [ResourceQuotaStatus.HardEntry](#k8s-io-api-core-v1-ResourceQuotaStatus-HardEntry) | repeated | Hard is the set of enforced hard limits for each named resource. More info: https://kubernetes.io/docs/concepts/policy/resource-quotas/ +optional |
| used | [ResourceQuotaStatus.UsedEntry](#k8s-io-api-core-v1-ResourceQuotaStatus-UsedEntry) | repeated | Used is the current observed total usage of the resource in the namespace. +optional |






<a name="k8s-io-api-core-v1-ResourceQuotaStatus-HardEntry"></a>

### ResourceQuotaStatus.HardEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [k8s.io.apimachinery.pkg.api.resource.Quantity](#k8s-io-apimachinery-pkg-api-resource-Quantity) | optional |  |






<a name="k8s-io-api-core-v1-ResourceQuotaStatus-UsedEntry"></a>

### ResourceQuotaStatus.UsedEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [k8s.io.apimachinery.pkg.api.resource.Quantity](#k8s-io-apimachinery-pkg-api-resource-Quantity) | optional |  |






<a name="k8s-io-api-core-v1-ResourceRequirements"></a>

### ResourceRequirements
ResourceRequirements describes the compute resource requirements.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| limits | [ResourceRequirements.LimitsEntry](#k8s-io-api-core-v1-ResourceRequirements-LimitsEntry) | repeated | Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/ +optional |
| requests | [ResourceRequirements.RequestsEntry](#k8s-io-api-core-v1-ResourceRequirements-RequestsEntry) | repeated | Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. Requests cannot exceed Limits. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/ +optional |
| claims | [ResourceClaim](#k8s-io-api-core-v1-ResourceClaim) | repeated | Claims lists the names of resources, defined in spec.resourceClaims, that are used by this container.

This is an alpha field and requires enabling the DynamicResourceAllocation feature gate.

This field is immutable. It can only be set for containers.

+listType=map +listMapKey=name +featureGate=DynamicResourceAllocation +optional |






<a name="k8s-io-api-core-v1-ResourceRequirements-LimitsEntry"></a>

### ResourceRequirements.LimitsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [k8s.io.apimachinery.pkg.api.resource.Quantity](#k8s-io-apimachinery-pkg-api-resource-Quantity) | optional |  |






<a name="k8s-io-api-core-v1-ResourceRequirements-RequestsEntry"></a>

### ResourceRequirements.RequestsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [k8s.io.apimachinery.pkg.api.resource.Quantity](#k8s-io-apimachinery-pkg-api-resource-Quantity) | optional |  |






<a name="k8s-io-api-core-v1-ResourceStatus"></a>

### ResourceStatus
ResourceStatus represents the status of a single resource allocated to a Pod.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name of the resource. Must be unique within the pod and in case of non-DRA resource, match one of the resources from the pod spec. For DRA resources, the value must be "claim:<claim_name>/<request>". When this status is reported about a container, the "claim_name" and "request" must match one of the claims of this container. +required |
| resources | [ResourceHealth](#k8s-io-api-core-v1-ResourceHealth) | repeated | List of unique resources health. Each element in the list contains an unique resource ID and its health. At a minimum, for the lifetime of a Pod, resource ID must uniquely identify the resource allocated to the Pod on the Node. If other Pod on the same Node reports the status with the same resource ID, it must be the same resource they share. See ResourceID type definition for a specific format it has in various use cases. +listType=map +listMapKey=resourceID |






<a name="k8s-io-api-core-v1-SELinuxOptions"></a>

### SELinuxOptions
SELinuxOptions are the labels to be applied to the container


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [string](#string) | optional | User is a SELinux user label that applies to the container. +optional |
| role | [string](#string) | optional | Role is a SELinux role label that applies to the container. +optional |
| type | [string](#string) | optional | Type is a SELinux type label that applies to the container. +optional |
| level | [string](#string) | optional | Level is SELinux level label that applies to the container. +optional |






<a name="k8s-io-api-core-v1-ScaleIOPersistentVolumeSource"></a>

### ScaleIOPersistentVolumeSource
ScaleIOPersistentVolumeSource represents a persistent ScaleIO volume


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| gateway | [string](#string) | optional | gateway is the host address of the ScaleIO API Gateway. |
| system | [string](#string) | optional | system is the name of the storage system as configured in ScaleIO. |
| secretRef | [SecretReference](#k8s-io-api-core-v1-SecretReference) | optional | secretRef references to the secret for ScaleIO user and other sensitive information. If this is not provided, Login operation will fail. |
| sslEnabled | [bool](#bool) | optional | sslEnabled is the flag to enable/disable SSL communication with Gateway, default false +optional |
| protectionDomain | [string](#string) | optional | protectionDomain is the name of the ScaleIO Protection Domain for the configured storage. +optional |
| storagePool | [string](#string) | optional | storagePool is the ScaleIO Storage Pool associated with the protection domain. +optional |
| storageMode | [string](#string) | optional | storageMode indicates whether the storage for a volume should be ThickProvisioned or ThinProvisioned. Default is ThinProvisioned. +optional +default="ThinProvisioned" |
| volumeName | [string](#string) | optional | volumeName is the name of a volume already created in the ScaleIO system that is associated with this volume source. |
| fsType | [string](#string) | optional | fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Default is "xfs" +optional +default="xfs" |
| readOnly | [bool](#bool) | optional | readOnly defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. +optional |






<a name="k8s-io-api-core-v1-ScaleIOVolumeSource"></a>

### ScaleIOVolumeSource
ScaleIOVolumeSource represents a persistent ScaleIO volume


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| gateway | [string](#string) | optional | gateway is the host address of the ScaleIO API Gateway. |
| system | [string](#string) | optional | system is the name of the storage system as configured in ScaleIO. |
| secretRef | [LocalObjectReference](#k8s-io-api-core-v1-LocalObjectReference) | optional | secretRef references to the secret for ScaleIO user and other sensitive information. If this is not provided, Login operation will fail. |
| sslEnabled | [bool](#bool) | optional | sslEnabled Flag enable/disable SSL communication with Gateway, default false +optional |
| protectionDomain | [string](#string) | optional | protectionDomain is the name of the ScaleIO Protection Domain for the configured storage. +optional |
| storagePool | [string](#string) | optional | storagePool is the ScaleIO Storage Pool associated with the protection domain. +optional |
| storageMode | [string](#string) | optional | storageMode indicates whether the storage for a volume should be ThickProvisioned or ThinProvisioned. Default is ThinProvisioned. +optional +default="ThinProvisioned" |
| volumeName | [string](#string) | optional | volumeName is the name of a volume already created in the ScaleIO system that is associated with this volume source. |
| fsType | [string](#string) | optional | fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Default is "xfs". +optional +default="xfs" |
| readOnly | [bool](#bool) | optional | readOnly Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. +optional |






<a name="k8s-io-api-core-v1-ScopeSelector"></a>

### ScopeSelector
A scope selector represents the AND of the selectors represented
by the scoped-resource selector requirements.
+structType=atomic


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| matchExpressions | [ScopedResourceSelectorRequirement](#k8s-io-api-core-v1-ScopedResourceSelectorRequirement) | repeated | A list of scope selector requirements by scope of the resources. +optional +listType=atomic |






<a name="k8s-io-api-core-v1-ScopedResourceSelectorRequirement"></a>

### ScopedResourceSelectorRequirement
A scoped-resource selector requirement is a selector that contains values, a scope name, and an operator
that relates the scope name and values.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| scopeName | [string](#string) | optional | The name of the scope that the selector applies to. |
| operator | [string](#string) | optional | Represents a scope's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. |
| values | [string](#string) | repeated | An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch. +optional +listType=atomic |






<a name="k8s-io-api-core-v1-SeccompProfile"></a>

### SeccompProfile
SeccompProfile defines a pod/container's seccomp profile settings.
Only one profile source may be set.
+union


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) | optional | type indicates which kind of seccomp profile will be applied. Valid options are:

Localhost - a profile defined in a file on the node should be used. RuntimeDefault - the container runtime default profile should be used. Unconfined - no profile should be applied. +unionDiscriminator |
| localhostProfile | [string](#string) | optional | localhostProfile indicates a profile defined in a file on the node should be used. The profile must be preconfigured on the node to work. Must be a descending path, relative to the kubelet's configured seccomp profile location. Must be set if type is "Localhost". Must NOT be set for any other type. +optional |






<a name="k8s-io-api-core-v1-Secret"></a>

### Secret
Secret holds secret data of a certain type. The total bytes of the values in
the Data field must be less than MaxSecretSize bytes.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional | Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional |
| immutable | [bool](#bool) | optional | Immutable, if set to true, ensures that data stored in the Secret cannot be updated (only object metadata can be modified). If not set to true, the field can be modified at any time. Defaulted to nil. +optional |
| data | [Secret.DataEntry](#k8s-io-api-core-v1-Secret-DataEntry) | repeated | Data contains the secret data. Each key must consist of alphanumeric characters, '-', '_' or '.'. The serialized form of the secret data is a base64 encoded string, representing the arbitrary (possibly non-string) data value here. Described in https://tools.ietf.org/html/rfc4648#section-4 +optional |
| stringData | [Secret.StringDataEntry](#k8s-io-api-core-v1-Secret-StringDataEntry) | repeated | stringData allows specifying non-binary secret data in string form. It is provided as a write-only input field for convenience. All keys and values are merged into the data field on write, overwriting any existing values. The stringData field is never output when reading from the API. +k8s:conversion-gen=false +optional |
| type | [string](#string) | optional | Used to facilitate programmatic handling of secret data. More info: https://kubernetes.io/docs/concepts/configuration/secret/#secret-types +optional |






<a name="k8s-io-api-core-v1-Secret-DataEntry"></a>

### Secret.DataEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [bytes](#bytes) | optional |  |






<a name="k8s-io-api-core-v1-Secret-StringDataEntry"></a>

### Secret.StringDataEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [string](#string) | optional |  |






<a name="k8s-io-api-core-v1-SecretEnvSource"></a>

### SecretEnvSource
SecretEnvSource selects a Secret to populate the environment
variables with.

The contents of the target Secret's Data field will represent the
key-value pairs as environment variables.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| localObjectReference | [LocalObjectReference](#k8s-io-api-core-v1-LocalObjectReference) | optional | The Secret to select from. |
| optional | [bool](#bool) | optional | Specify whether the Secret must be defined +optional |






<a name="k8s-io-api-core-v1-SecretKeySelector"></a>

### SecretKeySelector
SecretKeySelector selects a key of a Secret.
+structType=atomic


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| localObjectReference | [LocalObjectReference](#k8s-io-api-core-v1-LocalObjectReference) | optional | The name of the secret in the pod's namespace to select from. |
| key | [string](#string) | optional | The key of the secret to select from. Must be a valid secret key. |
| optional | [bool](#bool) | optional | Specify whether the Secret or its key must be defined +optional |






<a name="k8s-io-api-core-v1-SecretList"></a>

### SecretList
SecretList is a list of Secret.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional | Standard list metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds +optional |
| items | [Secret](#k8s-io-api-core-v1-Secret) | repeated | Items is a list of secret objects. More info: https://kubernetes.io/docs/concepts/configuration/secret |






<a name="k8s-io-api-core-v1-SecretProjection"></a>

### SecretProjection
Adapts a secret into a projected volume.

The contents of the target Secret's Data field will be presented in a
projected volume as files using the keys in the Data field as the file names.
Note that this is identical to a secret volume source without the default
mode.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| localObjectReference | [LocalObjectReference](#k8s-io-api-core-v1-LocalObjectReference) | optional |  |
| items | [KeyToPath](#k8s-io-api-core-v1-KeyToPath) | repeated | items if unspecified, each key-value pair in the Data field of the referenced Secret will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the Secret, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'. +optional +listType=atomic |
| optional | [bool](#bool) | optional | optional field specify whether the Secret or its key must be defined +optional |






<a name="k8s-io-api-core-v1-SecretReference"></a>

### SecretReference
SecretReference represents a Secret Reference. It has enough information to retrieve secret
in any namespace
+structType=atomic


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | name is unique within a namespace to reference a secret resource. +optional |
| namespace | [string](#string) | optional | namespace defines the space within which the secret name must be unique. +optional |






<a name="k8s-io-api-core-v1-SecretVolumeSource"></a>

### SecretVolumeSource
Adapts a Secret into a volume.

The contents of the target Secret's Data field will be presented in a volume
as files using the keys in the Data field as the file names.
Secret volumes support ownership management and SELinux relabeling.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| secretName | [string](#string) | optional | secretName is the name of the secret in the pod's namespace to use. More info: https://kubernetes.io/docs/concepts/storage/volumes#secret +optional |
| items | [KeyToPath](#k8s-io-api-core-v1-KeyToPath) | repeated | items If unspecified, each key-value pair in the Data field of the referenced Secret will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the Secret, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'. +optional +listType=atomic |
| defaultMode | [int32](#int32) | optional | defaultMode is Optional: mode bits used to set permissions on created files by default. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. Defaults to 0644. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set. +optional |
| optional | [bool](#bool) | optional | optional field specify whether the Secret or its keys must be defined +optional |






<a name="k8s-io-api-core-v1-SecurityContext"></a>

### SecurityContext
SecurityContext holds security configuration that will be applied to a container.
Some fields are present in both SecurityContext and PodSecurityContext.  When both
are set, the values in SecurityContext take precedence.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| capabilities | [Capabilities](#k8s-io-api-core-v1-Capabilities) | optional | The capabilities to add/drop when running containers. Defaults to the default set of capabilities granted by the container runtime. Note that this field cannot be set when spec.os.name is windows. +optional |
| privileged | [bool](#bool) | optional | Run container in privileged mode. Processes in privileged containers are essentially equivalent to root on the host. Defaults to false. Note that this field cannot be set when spec.os.name is windows. +optional |
| seLinuxOptions | [SELinuxOptions](#k8s-io-api-core-v1-SELinuxOptions) | optional | The SELinux context to be applied to the container. If unspecified, the container runtime will allocate a random SELinux context for each container. May also be set in PodSecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. Note that this field cannot be set when spec.os.name is windows. +optional |
| windowsOptions | [WindowsSecurityContextOptions](#k8s-io-api-core-v1-WindowsSecurityContextOptions) | optional | The Windows specific settings applied to all containers. If unspecified, the options from the PodSecurityContext will be used. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. Note that this field cannot be set when spec.os.name is linux. +optional |
| runAsUser | [int64](#int64) | optional | The UID to run the entrypoint of the container process. Defaults to user specified in image metadata if unspecified. May also be set in PodSecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. Note that this field cannot be set when spec.os.name is windows. +optional |
| runAsGroup | [int64](#int64) | optional | The GID to run the entrypoint of the container process. Uses runtime default if unset. May also be set in PodSecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. Note that this field cannot be set when spec.os.name is windows. +optional |
| runAsNonRoot | [bool](#bool) | optional | Indicates that the container must run as a non-root user. If true, the Kubelet will validate the image at runtime to ensure that it does not run as UID 0 (root) and fail to start the container if it does. If unset or false, no such validation will be performed. May also be set in PodSecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. +optional |
| readOnlyRootFilesystem | [bool](#bool) | optional | Whether this container has a read-only root filesystem. Default is false. Note that this field cannot be set when spec.os.name is windows. +optional |
| allowPrivilegeEscalation | [bool](#bool) | optional | AllowPrivilegeEscalation controls whether a process can gain more privileges than its parent process. This bool directly controls if the no_new_privs flag will be set on the container process. AllowPrivilegeEscalation is true always when the container is: 1) run as Privileged 2) has CAP_SYS_ADMIN Note that this field cannot be set when spec.os.name is windows. +optional |
| procMount | [string](#string) | optional | procMount denotes the type of proc mount to use for the containers. The default value is Default which uses the container runtime defaults for readonly paths and masked paths. This requires the ProcMountType feature flag to be enabled. Note that this field cannot be set when spec.os.name is windows. +optional |
| seccompProfile | [SeccompProfile](#k8s-io-api-core-v1-SeccompProfile) | optional | The seccomp options to use by this container. If seccomp options are provided at both the pod & container level, the container options override the pod options. Note that this field cannot be set when spec.os.name is windows. +optional |
| appArmorProfile | [AppArmorProfile](#k8s-io-api-core-v1-AppArmorProfile) | optional | appArmorProfile is the AppArmor options to use by this container. If set, this profile overrides the pod's appArmorProfile. Note that this field cannot be set when spec.os.name is windows. +optional |






<a name="k8s-io-api-core-v1-SerializedReference"></a>

### SerializedReference
SerializedReference is a reference to serialized object.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| reference | [ObjectReference](#k8s-io-api-core-v1-ObjectReference) | optional | The reference to an object in the system. +optional |






<a name="k8s-io-api-core-v1-Service"></a>

### Service
Service is a named abstraction of software service (for example, mysql) consisting of local port
(for example 3306) that the proxy listens on, and the selector that determines which pods
will answer requests sent through the proxy.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional | Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional |
| spec | [ServiceSpec](#k8s-io-api-core-v1-ServiceSpec) | optional | Spec defines the behavior of a service. https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status +optional |
| status | [ServiceStatus](#k8s-io-api-core-v1-ServiceStatus) | optional | Most recently observed status of the service. Populated by the system. Read-only. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status +optional |






<a name="k8s-io-api-core-v1-ServiceAccount"></a>

### ServiceAccount
ServiceAccount binds together:
* a name, understood by users, and perhaps by peripheral systems, for an identity
* a principal that can be authenticated and authorized
* a set of secrets


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional | Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional |
| secrets | [ObjectReference](#k8s-io-api-core-v1-ObjectReference) | repeated | Secrets is a list of the secrets in the same namespace that pods running using this ServiceAccount are allowed to use. Pods are only limited to this list if this service account has a "kubernetes.io/enforce-mountable-secrets" annotation set to "true". The "kubernetes.io/enforce-mountable-secrets" annotation is deprecated since v1.32. Prefer separate namespaces to isolate access to mounted secrets. This field should not be used to find auto-generated service account token secrets for use outside of pods. Instead, tokens can be requested directly using the TokenRequest API, or service account token secrets can be manually created. More info: https://kubernetes.io/docs/concepts/configuration/secret +optional +patchMergeKey=name +patchStrategy=merge +listType=map +listMapKey=name |
| imagePullSecrets | [LocalObjectReference](#k8s-io-api-core-v1-LocalObjectReference) | repeated | ImagePullSecrets is a list of references to secrets in the same namespace to use for pulling any images in pods that reference this ServiceAccount. ImagePullSecrets are distinct from Secrets because Secrets can be mounted in the pod, but ImagePullSecrets are only accessed by the kubelet. More info: https://kubernetes.io/docs/concepts/containers/images/#specifying-imagepullsecrets-on-a-pod +optional +listType=atomic |
| automountServiceAccountToken | [bool](#bool) | optional | AutomountServiceAccountToken indicates whether pods running as this service account should have an API token automatically mounted. Can be overridden at the pod level. +optional |






<a name="k8s-io-api-core-v1-ServiceAccountList"></a>

### ServiceAccountList
ServiceAccountList is a list of ServiceAccount objects


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional | Standard list metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds +optional |
| items | [ServiceAccount](#k8s-io-api-core-v1-ServiceAccount) | repeated | List of ServiceAccounts. More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/ |






<a name="k8s-io-api-core-v1-ServiceAccountTokenProjection"></a>

### ServiceAccountTokenProjection
ServiceAccountTokenProjection represents a projected service account token
volume. This projection can be used to insert a service account token into
the pods runtime filesystem for use against APIs (Kubernetes API Server or
otherwise).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| audience | [string](#string) | optional | audience is the intended audience of the token. A recipient of a token must identify itself with an identifier specified in the audience of the token, and otherwise should reject the token. The audience defaults to the identifier of the apiserver. +optional |
| expirationSeconds | [int64](#int64) | optional | expirationSeconds is the requested duration of validity of the service account token. As the token approaches expiration, the kubelet volume plugin will proactively rotate the service account token. The kubelet will start trying to rotate the token if the token is older than 80 percent of its time to live or if the token is older than 24 hours.Defaults to 1 hour and must be at least 10 minutes. +optional |
| path | [string](#string) | optional | path is the path relative to the mount point of the file to project the token into. |






<a name="k8s-io-api-core-v1-ServiceList"></a>

### ServiceList
ServiceList holds a list of services.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional | Standard list metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds +optional |
| items | [Service](#k8s-io-api-core-v1-Service) | repeated | List of services |






<a name="k8s-io-api-core-v1-ServicePort"></a>

### ServicePort
ServicePort contains information on service's port.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | The name of this port within the service. This must be a DNS_LABEL. All ports within a ServiceSpec must have unique names. When considering the endpoints for a Service, this must match the 'name' field in the EndpointPort. Optional if only one ServicePort is defined on this service. +optional |
| protocol | [string](#string) | optional | The IP protocol for this port. Supports "TCP", "UDP", and "SCTP". Default is TCP. +default="TCP" +optional |
| appProtocol | [string](#string) | optional | The application protocol for this port. This is used as a hint for implementations to offer richer behavior for protocols that they understand. This field follows standard Kubernetes label syntax. Valid values are either:

* Un-prefixed protocol names - reserved for IANA standard service names (as per RFC-6335 and https://www.iana.org/assignments/service-names).

* Kubernetes-defined prefixed names: * 'kubernetes.io/h2c' - HTTP/2 prior knowledge over cleartext as described in https://www.rfc-editor.org/rfc/rfc9113.html#name-starting-http-2-with-prior- * 'kubernetes.io/ws' - WebSocket over cleartext as described in https://www.rfc-editor.org/rfc/rfc6455 * 'kubernetes.io/wss' - WebSocket over TLS as described in https://www.rfc-editor.org/rfc/rfc6455

* Other protocols should use implementation-defined prefixed names such as mycompany.com/my-custom-protocol. +optional |
| port | [int32](#int32) | optional | The port that will be exposed by this service. |
| targetPort | [k8s.io.apimachinery.pkg.util.intstr.IntOrString](#k8s-io-apimachinery-pkg-util-intstr-IntOrString) | optional | Number or name of the port to access on the pods targeted by the service. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME. If this is a string, it will be looked up as a named port in the target Pod's container ports. If this is not specified, the value of the 'port' field is used (an identity map). This field is ignored for services with clusterIP=None, and should be omitted or set equal to the 'port' field. More info: https://kubernetes.io/docs/concepts/services-networking/service/#defining-a-service +optional |
| nodePort | [int32](#int32) | optional | The port on each node on which this service is exposed when type is NodePort or LoadBalancer. Usually assigned by the system. If a value is specified, in-range, and not in use it will be used, otherwise the operation will fail. If not specified, a port will be allocated if this Service requires one. If this field is specified when creating a Service which does not need it, creation will fail. This field will be wiped when updating a Service to no longer need it (e.g. changing type from NodePort to ClusterIP). More info: https://kubernetes.io/docs/concepts/services-networking/service/#type-nodeport +optional |






<a name="k8s-io-api-core-v1-ServiceProxyOptions"></a>

### ServiceProxyOptions
ServiceProxyOptions is the query options to a Service's proxy call.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) | optional | Path is the part of URLs that include service endpoints, suffixes, and parameters to use for the current proxy request to service. For example, the whole request URL is http://localhost/api/v1/namespaces/kube-system/services/elasticsearch-logging/_search?q=user:kimchy. Path is _search?q=user:kimchy. +optional |






<a name="k8s-io-api-core-v1-ServiceSpec"></a>

### ServiceSpec
ServiceSpec describes the attributes that a user creates on a service.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ports | [ServicePort](#k8s-io-api-core-v1-ServicePort) | repeated | The list of ports that are exposed by this service. More info: https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies +patchMergeKey=port +patchStrategy=merge +listType=map +listMapKey=port +listMapKey=protocol |
| selector | [ServiceSpec.SelectorEntry](#k8s-io-api-core-v1-ServiceSpec-SelectorEntry) | repeated | Route service traffic to pods with label keys and values matching this selector. If empty or not present, the service is assumed to have an external process managing its endpoints, which Kubernetes will not modify. Only applies to types ClusterIP, NodePort, and LoadBalancer. Ignored if type is ExternalName. More info: https://kubernetes.io/docs/concepts/services-networking/service/ +optional +mapType=atomic |
| clusterIP | [string](#string) | optional | clusterIP is the IP address of the service and is usually assigned randomly. If an address is specified manually, is in-range (as per system configuration), and is not in use, it will be allocated to the service; otherwise creation of the service will fail. This field may not be changed through updates unless the type field is also being changed to ExternalName (which requires this field to be blank) or the type field is being changed from ExternalName (in which case this field may optionally be specified, as describe above). Valid values are "None", empty string (""), or a valid IP address. Setting this to "None" makes a "headless service" (no virtual IP), which is useful when direct endpoint connections are preferred and proxying is not required. Only applies to types ClusterIP, NodePort, and LoadBalancer. If this field is specified when creating a Service of type ExternalName, creation will fail. This field will be wiped when updating a Service to type ExternalName. More info: https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies +optional |
| clusterIPs | [string](#string) | repeated | ClusterIPs is a list of IP addresses assigned to this service, and are usually assigned randomly. If an address is specified manually, is in-range (as per system configuration), and is not in use, it will be allocated to the service; otherwise creation of the service will fail. This field may not be changed through updates unless the type field is also being changed to ExternalName (which requires this field to be empty) or the type field is being changed from ExternalName (in which case this field may optionally be specified, as describe above). Valid values are "None", empty string (""), or a valid IP address. Setting this to "None" makes a "headless service" (no virtual IP), which is useful when direct endpoint connections are preferred and proxying is not required. Only applies to types ClusterIP, NodePort, and LoadBalancer. If this field is specified when creating a Service of type ExternalName, creation will fail. This field will be wiped when updating a Service to type ExternalName. If this field is not specified, it will be initialized from the clusterIP field. If this field is specified, clients must ensure that clusterIPs[0] and clusterIP have the same value.

This field may hold a maximum of two entries (dual-stack IPs, in either order). These IPs must correspond to the values of the ipFamilies field. Both clusterIPs and ipFamilies are governed by the ipFamilyPolicy field. More info: https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies +listType=atomic +optional |
| type | [string](#string) | optional | type determines how the Service is exposed. Defaults to ClusterIP. Valid options are ExternalName, ClusterIP, NodePort, and LoadBalancer. "ClusterIP" allocates a cluster-internal IP address for load-balancing to endpoints. Endpoints are determined by the selector or if that is not specified, by manual construction of an Endpoints object or EndpointSlice objects. If clusterIP is "None", no virtual IP is allocated and the endpoints are published as a set of endpoints rather than a virtual IP. "NodePort" builds on ClusterIP and allocates a port on every node which routes to the same endpoints as the clusterIP. "LoadBalancer" builds on NodePort and creates an external load-balancer (if supported in the current cloud) which routes to the same endpoints as the clusterIP. "ExternalName" aliases this service to the specified externalName. Several other fields do not apply to ExternalName services. More info: https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types +optional |
| externalIPs | [string](#string) | repeated | externalIPs is a list of IP addresses for which nodes in the cluster will also accept traffic for this service. These IPs are not managed by Kubernetes. The user is responsible for ensuring that traffic arrives at a node with this IP. A common example is external load-balancers that are not part of the Kubernetes system. +optional +listType=atomic |
| sessionAffinity | [string](#string) | optional | Supports "ClientIP" and "None". Used to maintain session affinity. Enable client IP based session affinity. Must be ClientIP or None. Defaults to None. More info: https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies +optional |
| loadBalancerIP | [string](#string) | optional | Only applies to Service Type: LoadBalancer. This feature depends on whether the underlying cloud-provider supports specifying the loadBalancerIP when a load balancer is created. This field will be ignored if the cloud-provider does not support the feature. Deprecated: This field was under-specified and its meaning varies across implementations. Using it is non-portable and it may not support dual-stack. Users are encouraged to use implementation-specific annotations when available. +optional |
| loadBalancerSourceRanges | [string](#string) | repeated | If specified and supported by the platform, this will restrict traffic through the cloud-provider load-balancer will be restricted to the specified client IPs. This field will be ignored if the cloud-provider does not support the feature." More info: https://kubernetes.io/docs/tasks/access-application-cluster/create-external-load-balancer/ +optional +listType=atomic |
| externalName | [string](#string) | optional | externalName is the external reference that discovery mechanisms will return as an alias for this service (e.g. a DNS CNAME record). No proxying will be involved. Must be a lowercase RFC-1123 hostname (https://tools.ietf.org/html/rfc1123) and requires `type` to be "ExternalName". +optional |
| externalTrafficPolicy | [string](#string) | optional | externalTrafficPolicy describes how nodes distribute service traffic they receive on one of the Service's "externally-facing" addresses (NodePorts, ExternalIPs, and LoadBalancer IPs). If set to "Local", the proxy will configure the service in a way that assumes that external load balancers will take care of balancing the service traffic between nodes, and so each node will deliver traffic only to the node-local endpoints of the service, without masquerading the client source IP. (Traffic mistakenly sent to a node with no endpoints will be dropped.) The default value, "Cluster", uses the standard behavior of routing to all endpoints evenly (possibly modified by topology and other features). Note that traffic sent to an External IP or LoadBalancer IP from within the cluster will always get "Cluster" semantics, but clients sending to a NodePort from within the cluster may need to take traffic policy into account when picking a node. +optional |
| healthCheckNodePort | [int32](#int32) | optional | healthCheckNodePort specifies the healthcheck nodePort for the service. This only applies when type is set to LoadBalancer and externalTrafficPolicy is set to Local. If a value is specified, is in-range, and is not in use, it will be used. If not specified, a value will be automatically allocated. External systems (e.g. load-balancers) can use this port to determine if a given node holds endpoints for this service or not. If this field is specified when creating a Service which does not need it, creation will fail. This field will be wiped when updating a Service to no longer need it (e.g. changing type). This field cannot be updated once set. +optional |
| publishNotReadyAddresses | [bool](#bool) | optional | publishNotReadyAddresses indicates that any agent which deals with endpoints for this Service should disregard any indications of ready/not-ready. The primary use case for setting this field is for a StatefulSet's Headless Service to propagate SRV DNS records for its Pods for the purpose of peer discovery. The Kubernetes controllers that generate Endpoints and EndpointSlice resources for Services interpret this to mean that all endpoints are considered "ready" even if the Pods themselves are not. Agents which consume only Kubernetes generated endpoints through the Endpoints or EndpointSlice resources can safely assume this behavior. +optional |
| sessionAffinityConfig | [SessionAffinityConfig](#k8s-io-api-core-v1-SessionAffinityConfig) | optional | sessionAffinityConfig contains the configurations of session affinity. +optional |
| ipFamilies | [string](#string) | repeated | IPFamilies is a list of IP families (e.g. IPv4, IPv6) assigned to this service. This field is usually assigned automatically based on cluster configuration and the ipFamilyPolicy field. If this field is specified manually, the requested family is available in the cluster, and ipFamilyPolicy allows it, it will be used; otherwise creation of the service will fail. This field is conditionally mutable: it allows for adding or removing a secondary IP family, but it does not allow changing the primary IP family of the Service. Valid values are "IPv4" and "IPv6". This field only applies to Services of types ClusterIP, NodePort, and LoadBalancer, and does apply to "headless" services. This field will be wiped when updating a Service to type ExternalName.

This field may hold a maximum of two entries (dual-stack families, in either order). These families must correspond to the values of the clusterIPs field, if specified. Both clusterIPs and ipFamilies are governed by the ipFamilyPolicy field. +listType=atomic +optional |
| ipFamilyPolicy | [string](#string) | optional | IPFamilyPolicy represents the dual-stack-ness requested or required by this Service. If there is no value provided, then this field will be set to SingleStack. Services can be "SingleStack" (a single IP family), "PreferDualStack" (two IP families on dual-stack configured clusters or a single IP family on single-stack clusters), or "RequireDualStack" (two IP families on dual-stack configured clusters, otherwise fail). The ipFamilies and clusterIPs fields depend on the value of this field. This field will be wiped when updating a service to type ExternalName. +optional |
| allocateLoadBalancerNodePorts | [bool](#bool) | optional | allocateLoadBalancerNodePorts defines if NodePorts will be automatically allocated for services with type LoadBalancer. Default is "true". It may be set to "false" if the cluster load-balancer does not rely on NodePorts. If the caller requests specific NodePorts (by specifying a value), those requests will be respected, regardless of this field. This field may only be set for services with type LoadBalancer and will be cleared if the type is changed to any other type. +optional |
| loadBalancerClass | [string](#string) | optional | loadBalancerClass is the class of the load balancer implementation this Service belongs to. If specified, the value of this field must be a label-style identifier, with an optional prefix, e.g. "internal-vip" or "example.com/internal-vip". Unprefixed names are reserved for end-users. This field can only be set when the Service type is 'LoadBalancer'. If not set, the default load balancer implementation is used, today this is typically done through the cloud provider integration, but should apply for any default implementation. If set, it is assumed that a load balancer implementation is watching for Services with a matching class. Any default load balancer implementation (e.g. cloud providers) should ignore Services that set this field. This field can only be set when creating or updating a Service to type 'LoadBalancer'. Once set, it can not be changed. This field will be wiped when a service is updated to a non 'LoadBalancer' type. +optional |
| internalTrafficPolicy | [string](#string) | optional | InternalTrafficPolicy describes how nodes distribute service traffic they receive on the ClusterIP. If set to "Local", the proxy will assume that pods only want to talk to endpoints of the service on the same node as the pod, dropping the traffic if there are no local endpoints. The default value, "Cluster", uses the standard behavior of routing to all endpoints evenly (possibly modified by topology and other features). +optional |
| trafficDistribution | [string](#string) | optional | TrafficDistribution offers a way to express preferences for how traffic is distributed to Service endpoints. Implementations can use this field as a hint, but are not required to guarantee strict adherence. If the field is not set, the implementation will apply its default routing strategy. If set to "PreferClose", implementations should prioritize endpoints that are in the same zone. +featureGate=ServiceTrafficDistribution +optional |






<a name="k8s-io-api-core-v1-ServiceSpec-SelectorEntry"></a>

### ServiceSpec.SelectorEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [string](#string) | optional |  |






<a name="k8s-io-api-core-v1-ServiceStatus"></a>

### ServiceStatus
ServiceStatus represents the current status of a service.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| loadBalancer | [LoadBalancerStatus](#k8s-io-api-core-v1-LoadBalancerStatus) | optional | LoadBalancer contains the current status of the load-balancer, if one is present. +optional |
| conditions | [k8s.io.apimachinery.pkg.apis.meta.v1.Condition](#k8s-io-apimachinery-pkg-apis-meta-v1-Condition) | repeated | Current service state +optional +patchMergeKey=type +patchStrategy=merge +listType=map +listMapKey=type |






<a name="k8s-io-api-core-v1-SessionAffinityConfig"></a>

### SessionAffinityConfig
SessionAffinityConfig represents the configurations of session affinity.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| clientIP | [ClientIPConfig](#k8s-io-api-core-v1-ClientIPConfig) | optional | clientIP contains the configurations of Client IP based session affinity. +optional |






<a name="k8s-io-api-core-v1-SleepAction"></a>

### SleepAction
SleepAction describes a "sleep" action.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| seconds | [int64](#int64) | optional | Seconds is the number of seconds to sleep. |






<a name="k8s-io-api-core-v1-StorageOSPersistentVolumeSource"></a>

### StorageOSPersistentVolumeSource
Represents a StorageOS persistent volume resource.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| volumeName | [string](#string) | optional | volumeName is the human-readable name of the StorageOS volume. Volume names are only unique within a namespace. |
| volumeNamespace | [string](#string) | optional | volumeNamespace specifies the scope of the volume within StorageOS. If no namespace is specified then the Pod's namespace will be used. This allows the Kubernetes name scoping to be mirrored within StorageOS for tighter integration. Set VolumeName to any name to override the default behaviour. Set to "default" if you are not using namespaces within StorageOS. Namespaces that do not pre-exist within StorageOS will be created. +optional |
| fsType | [string](#string) | optional | fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. +optional |
| readOnly | [bool](#bool) | optional | readOnly defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. +optional |
| secretRef | [ObjectReference](#k8s-io-api-core-v1-ObjectReference) | optional | secretRef specifies the secret to use for obtaining the StorageOS API credentials. If not specified, default values will be attempted. +optional |






<a name="k8s-io-api-core-v1-StorageOSVolumeSource"></a>

### StorageOSVolumeSource
Represents a StorageOS persistent volume resource.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| volumeName | [string](#string) | optional | volumeName is the human-readable name of the StorageOS volume. Volume names are only unique within a namespace. |
| volumeNamespace | [string](#string) | optional | volumeNamespace specifies the scope of the volume within StorageOS. If no namespace is specified then the Pod's namespace will be used. This allows the Kubernetes name scoping to be mirrored within StorageOS for tighter integration. Set VolumeName to any name to override the default behaviour. Set to "default" if you are not using namespaces within StorageOS. Namespaces that do not pre-exist within StorageOS will be created. +optional |
| fsType | [string](#string) | optional | fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. +optional |
| readOnly | [bool](#bool) | optional | readOnly defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. +optional |
| secretRef | [LocalObjectReference](#k8s-io-api-core-v1-LocalObjectReference) | optional | secretRef specifies the secret to use for obtaining the StorageOS API credentials. If not specified, default values will be attempted. +optional |






<a name="k8s-io-api-core-v1-Sysctl"></a>

### Sysctl
Sysctl defines a kernel parameter to be set


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name of a property to set |
| value | [string](#string) | optional | Value of a property to set |






<a name="k8s-io-api-core-v1-TCPSocketAction"></a>

### TCPSocketAction
TCPSocketAction describes an action based on opening a socket


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| port | [k8s.io.apimachinery.pkg.util.intstr.IntOrString](#k8s-io-apimachinery-pkg-util-intstr-IntOrString) | optional | Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME. |
| host | [string](#string) | optional | Optional: Host name to connect to, defaults to the pod IP. +optional |






<a name="k8s-io-api-core-v1-Taint"></a>

### Taint
The node this Taint is attached to has the "effect" on
any pod that does not tolerate the Taint.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional | Required. The taint key to be applied to a node. |
| value | [string](#string) | optional | The taint value corresponding to the taint key. +optional |
| effect | [string](#string) | optional | Required. The effect of the taint on pods that do not tolerate the taint. Valid effects are NoSchedule, PreferNoSchedule and NoExecute. |
| timeAdded | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | TimeAdded represents the time at which the taint was added. It is only written for NoExecute taints. +optional |






<a name="k8s-io-api-core-v1-Toleration"></a>

### Toleration
The pod this Toleration is attached to tolerates any taint that matches
the triple <key,value,effect> using the matching operator <operator>.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional | Key is the taint key that the toleration applies to. Empty means match all taint keys. If the key is empty, operator must be Exists; this combination means to match all values and all keys. +optional |
| operator | [string](#string) | optional | Operator represents a key's relationship to the value. Valid operators are Exists and Equal. Defaults to Equal. Exists is equivalent to wildcard for value, so that a pod can tolerate all taints of a particular category. +optional |
| value | [string](#string) | optional | Value is the taint value the toleration matches to. If the operator is Exists, the value should be empty, otherwise just a regular string. +optional |
| effect | [string](#string) | optional | Effect indicates the taint effect to match. Empty means match all taint effects. When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute. +optional |
| tolerationSeconds | [int64](#int64) | optional | TolerationSeconds represents the period of time the toleration (which must be of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default, it is not set, which means tolerate the taint forever (do not evict). Zero and negative values will be treated as 0 (evict immediately) by the system. +optional |






<a name="k8s-io-api-core-v1-TopologySelectorLabelRequirement"></a>

### TopologySelectorLabelRequirement
A topology selector requirement is a selector that matches given label.
This is an alpha feature and may change in the future.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional | The label key that the selector applies to. |
| values | [string](#string) | repeated | An array of string values. One value must match the label to be selected. Each entry in Values is ORed. +listType=atomic |






<a name="k8s-io-api-core-v1-TopologySelectorTerm"></a>

### TopologySelectorTerm
A topology selector term represents the result of label queries.
A null or empty topology selector term matches no objects.
The requirements of them are ANDed.
It provides a subset of functionality as NodeSelectorTerm.
This is an alpha feature and may change in the future.
+structType=atomic


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| matchLabelExpressions | [TopologySelectorLabelRequirement](#k8s-io-api-core-v1-TopologySelectorLabelRequirement) | repeated | A list of topology selector requirements by labels. +optional +listType=atomic |






<a name="k8s-io-api-core-v1-TopologySpreadConstraint"></a>

### TopologySpreadConstraint
TopologySpreadConstraint specifies how to spread matching pods among the given topology.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| maxSkew | [int32](#int32) | optional | MaxSkew describes the degree to which pods may be unevenly distributed. When `whenUnsatisfiable=DoNotSchedule`, it is the maximum permitted difference between the number of matching pods in the target topology and the global minimum. The global minimum is the minimum number of matching pods in an eligible domain or zero if the number of eligible domains is less than MinDomains. For example, in a 3-zone cluster, MaxSkew is set to 1, and pods with the same labelSelector spread as 2/2/1: In this case, the global minimum is 1. +-------+-------+-------+ | zone1 | zone2 | zone3 | +-------+-------+-------+ | P P | P P | P | +-------+-------+-------+ - if MaxSkew is 1, incoming pod can only be scheduled to zone3 to become 2/2/2; scheduling it onto zone1(zone2) would make the ActualSkew(3-1) on zone1(zone2) violate MaxSkew(1). - if MaxSkew is 2, incoming pod can be scheduled onto any zone. When `whenUnsatisfiable=ScheduleAnyway`, it is used to give higher precedence to topologies that satisfy it. It's a required field. Default value is 1 and 0 is not allowed. |
| topologyKey | [string](#string) | optional | TopologyKey is the key of node labels. Nodes that have a label with this key and identical values are considered to be in the same topology. We consider each <key, value> as a "bucket", and try to put balanced number of pods into each bucket. We define a domain as a particular instance of a topology. Also, we define an eligible domain as a domain whose nodes meet the requirements of nodeAffinityPolicy and nodeTaintsPolicy. e.g. If TopologyKey is "kubernetes.io/hostname", each Node is a domain of that topology. And, if TopologyKey is "topology.kubernetes.io/zone", each zone is a domain of that topology. It's a required field. |
| whenUnsatisfiable | [string](#string) | optional | WhenUnsatisfiable indicates how to deal with a pod if it doesn't satisfy the spread constraint. - DoNotSchedule (default) tells the scheduler not to schedule it. - ScheduleAnyway tells the scheduler to schedule the pod in any location, but giving higher precedence to topologies that would help reduce the skew. A constraint is considered "Unsatisfiable" for an incoming pod if and only if every possible node assignment for that pod would violate "MaxSkew" on some topology. For example, in a 3-zone cluster, MaxSkew is set to 1, and pods with the same labelSelector spread as 3/1/1: +-------+-------+-------+ | zone1 | zone2 | zone3 | +-------+-------+-------+ | P P P | P | P | +-------+-------+-------+ If WhenUnsatisfiable is set to DoNotSchedule, incoming pod can only be scheduled to zone2(zone3) to become 3/2/1(3/1/2) as ActualSkew(2-1) on zone2(zone3) satisfies MaxSkew(1). In other words, the cluster can still be imbalanced, but scheduler won't make it *more* imbalanced. It's a required field. |
| labelSelector | [k8s.io.apimachinery.pkg.apis.meta.v1.LabelSelector](#k8s-io-apimachinery-pkg-apis-meta-v1-LabelSelector) | optional | LabelSelector is used to find matching pods. Pods that match this label selector are counted to determine the number of pods in their corresponding topology domain. +optional |
| minDomains | [int32](#int32) | optional | MinDomains indicates a minimum number of eligible domains. When the number of eligible domains with matching topology keys is less than minDomains, Pod Topology Spread treats "global minimum" as 0, and then the calculation of Skew is performed. And when the number of eligible domains with matching topology keys equals or greater than minDomains, this value has no effect on scheduling. As a result, when the number of eligible domains is less than minDomains, scheduler won't schedule more than maxSkew Pods to those domains. If value is nil, the constraint behaves as if MinDomains is equal to 1. Valid values are integers greater than 0. When value is not nil, WhenUnsatisfiable must be DoNotSchedule.

For example, in a 3-zone cluster, MaxSkew is set to 2, MinDomains is set to 5 and pods with the same labelSelector spread as 2/2/2: +-------+-------+-------+ | zone1 | zone2 | zone3 | +-------+-------+-------+ | P P | P P | P P | +-------+-------+-------+ The number of domains is less than 5(MinDomains), so "global minimum" is treated as 0. In this situation, new pod with the same labelSelector cannot be scheduled, because computed skew will be 3(3 - 0) if new Pod is scheduled to any of the three zones, it will violate MaxSkew. +optional |
| nodeAffinityPolicy | [string](#string) | optional | NodeAffinityPolicy indicates how we will treat Pod's nodeAffinity/nodeSelector when calculating pod topology spread skew. Options are: - Honor: only nodes matching nodeAffinity/nodeSelector are included in the calculations. - Ignore: nodeAffinity/nodeSelector are ignored. All nodes are included in the calculations.

If this value is nil, the behavior is equivalent to the Honor policy. +optional |
| nodeTaintsPolicy | [string](#string) | optional | NodeTaintsPolicy indicates how we will treat node taints when calculating pod topology spread skew. Options are: - Honor: nodes without taints, along with tainted nodes for which the incoming pod has a toleration, are included. - Ignore: node taints are ignored. All nodes are included.

If this value is nil, the behavior is equivalent to the Ignore policy. +optional |
| matchLabelKeys | [string](#string) | repeated | MatchLabelKeys is a set of pod label keys to select the pods over which spreading will be calculated. The keys are used to lookup values from the incoming pod labels, those key-value labels are ANDed with labelSelector to select the group of existing pods over which spreading will be calculated for the incoming pod. The same key is forbidden to exist in both MatchLabelKeys and LabelSelector. MatchLabelKeys cannot be set when LabelSelector isn't set. Keys that don't exist in the incoming pod labels will be ignored. A null or empty list means only match against labelSelector.

This is a beta field and requires the MatchLabelKeysInPodTopologySpread feature gate to be enabled (enabled by default). +listType=atomic +optional |






<a name="k8s-io-api-core-v1-TypedLocalObjectReference"></a>

### TypedLocalObjectReference
TypedLocalObjectReference contains enough information to let you locate the
typed referenced object inside the same namespace.
---
New uses of this type are discouraged because of difficulty describing its usage when embedded in APIs.
 1. Invalid usage help.  It is impossible to add specific help for individual usage.  In most embedded usages, there are particular
    restrictions like, "must refer only to types A and B" or "UID not honored" or "name must be restricted".
    Those cannot be well described when embedded.
 2. Inconsistent validation.  Because the usages are different, the validation rules are different by usage, which makes it hard for users to predict what will happen.
 3. The fields are both imprecise and overly precise.  Kind is not a precise mapping to a URL. This can produce ambiguity
    during interpretation and require a REST mapping.  In most cases, the dependency is on the group,resource tuple
    and the version of the actual struct is irrelevant.
 4. We cannot easily change it.  Because this type is embedded in many locations, updates to this type
    will affect numerous schemas.  Don't make new APIs embed an underspecified API type they do not control.

Instead of using this type, create a locally provided and used type that is well-focused on your reference.
For example, ServiceReferences for admission registration: https://github.com/kubernetes/api/blob/release-1.17/admissionregistration/v1/types.go#L533 .
+structType=atomic


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| apiGroup | [string](#string) | optional | APIGroup is the group for the resource being referenced. If APIGroup is not specified, the specified Kind must be in the core API group. For any other third-party types, APIGroup is required. +optional |
| kind | [string](#string) | optional | Kind is the type of resource being referenced |
| name | [string](#string) | optional | Name is the name of resource being referenced |






<a name="k8s-io-api-core-v1-TypedObjectReference"></a>

### TypedObjectReference
TypedObjectReference contains enough information to let you locate the typed referenced object


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| apiGroup | [string](#string) | optional | APIGroup is the group for the resource being referenced. If APIGroup is not specified, the specified Kind must be in the core API group. For any other third-party types, APIGroup is required. +optional |
| kind | [string](#string) | optional | Kind is the type of resource being referenced |
| name | [string](#string) | optional | Name is the name of resource being referenced |
| namespace | [string](#string) | optional | Namespace is the namespace of resource being referenced Note that when a namespace is specified, a gateway.networking.k8s.io/ReferenceGrant object is required in the referent namespace to allow that namespace's owner to accept the reference. See the ReferenceGrant documentation for details. (Alpha) This field requires the CrossNamespaceVolumeDataSource feature gate to be enabled. +featureGate=CrossNamespaceVolumeDataSource +optional |






<a name="k8s-io-api-core-v1-Volume"></a>

### Volume
Volume represents a named volume in a pod that may be accessed by any container in the pod.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | name of the volume. Must be a DNS_LABEL and unique within the pod. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names |
| volumeSource | [VolumeSource](#k8s-io-api-core-v1-VolumeSource) | optional | volumeSource represents the location and type of the mounted volume. If not specified, the Volume is implied to be an EmptyDir. This implied behavior is deprecated and will be removed in a future version. |






<a name="k8s-io-api-core-v1-VolumeDevice"></a>

### VolumeDevice
volumeDevice describes a mapping of a raw block device within a container.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | name must match the name of a persistentVolumeClaim in the pod |
| devicePath | [string](#string) | optional | devicePath is the path inside of the container that the device will be mapped to. |






<a name="k8s-io-api-core-v1-VolumeMount"></a>

### VolumeMount
VolumeMount describes a mounting of a Volume within a container.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | This must match the Name of a Volume. |
| readOnly | [bool](#bool) | optional | Mounted read-only if true, read-write otherwise (false or unspecified). Defaults to false. +optional |
| recursiveReadOnly | [string](#string) | optional | RecursiveReadOnly specifies whether read-only mounts should be handled recursively.

If ReadOnly is false, this field has no meaning and must be unspecified.

If ReadOnly is true, and this field is set to Disabled, the mount is not made recursively read-only. If this field is set to IfPossible, the mount is made recursively read-only, if it is supported by the container runtime. If this field is set to Enabled, the mount is made recursively read-only if it is supported by the container runtime, otherwise the pod will not be started and an error will be generated to indicate the reason.

If this field is set to IfPossible or Enabled, MountPropagation must be set to None (or be unspecified, which defaults to None).

If this field is not specified, it is treated as an equivalent of Disabled.

+featureGate=RecursiveReadOnlyMounts +optional |
| mountPath | [string](#string) | optional | Path within the container at which the volume should be mounted. Must not contain ':'. |
| subPath | [string](#string) | optional | Path within the volume from which the container's volume should be mounted. Defaults to "" (volume's root). +optional |
| mountPropagation | [string](#string) | optional | mountPropagation determines how mounts are propagated from the host to container and the other way around. When not set, MountPropagationNone is used. This field is beta in 1.10. When RecursiveReadOnly is set to IfPossible or to Enabled, MountPropagation must be None or unspecified (which defaults to None). +optional |
| subPathExpr | [string](#string) | optional | Expanded path within the volume from which the container's volume should be mounted. Behaves similarly to SubPath but environment variable references $(VAR_NAME) are expanded using the container's environment. Defaults to "" (volume's root). SubPathExpr and SubPath are mutually exclusive. +optional |






<a name="k8s-io-api-core-v1-VolumeMountStatus"></a>

### VolumeMountStatus
VolumeMountStatus shows status of volume mounts.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name corresponds to the name of the original VolumeMount. |
| mountPath | [string](#string) | optional | MountPath corresponds to the original VolumeMount. |
| readOnly | [bool](#bool) | optional | ReadOnly corresponds to the original VolumeMount. +optional |
| recursiveReadOnly | [string](#string) | optional | RecursiveReadOnly must be set to Disabled, Enabled, or unspecified (for non-readonly mounts). An IfPossible value in the original VolumeMount must be translated to Disabled or Enabled, depending on the mount result. +featureGate=RecursiveReadOnlyMounts +optional |






<a name="k8s-io-api-core-v1-VolumeNodeAffinity"></a>

### VolumeNodeAffinity
VolumeNodeAffinity defines constraints that limit what nodes this volume can be accessed from.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| required | [NodeSelector](#k8s-io-api-core-v1-NodeSelector) | optional | required specifies hard node constraints that must be met. |






<a name="k8s-io-api-core-v1-VolumeProjection"></a>

### VolumeProjection
Projection that may be projected along with other supported volume types.
Exactly one of these fields must be set.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| secret | [SecretProjection](#k8s-io-api-core-v1-SecretProjection) | optional | secret information about the secret data to project +optional |
| downwardAPI | [DownwardAPIProjection](#k8s-io-api-core-v1-DownwardAPIProjection) | optional | downwardAPI information about the downwardAPI data to project +optional |
| configMap | [ConfigMapProjection](#k8s-io-api-core-v1-ConfigMapProjection) | optional | configMap information about the configMap data to project +optional |
| serviceAccountToken | [ServiceAccountTokenProjection](#k8s-io-api-core-v1-ServiceAccountTokenProjection) | optional | serviceAccountToken is information about the serviceAccountToken data to project +optional |
| clusterTrustBundle | [ClusterTrustBundleProjection](#k8s-io-api-core-v1-ClusterTrustBundleProjection) | optional | ClusterTrustBundle allows a pod to access the `.spec.trustBundle` field of ClusterTrustBundle objects in an auto-updating file.

Alpha, gated by the ClusterTrustBundleProjection feature gate.

ClusterTrustBundle objects can either be selected by name, or by the combination of signer name and a label selector.

Kubelet performs aggressive normalization of the PEM contents written into the pod filesystem. Esoteric PEM features such as inter-block comments and block headers are stripped. Certificates are deduplicated. The ordering of certificates within the file is arbitrary, and Kubelet may change the order over time.

+featureGate=ClusterTrustBundleProjection +optional |






<a name="k8s-io-api-core-v1-VolumeResourceRequirements"></a>

### VolumeResourceRequirements
VolumeResourceRequirements describes the storage resource requirements for a volume.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| limits | [VolumeResourceRequirements.LimitsEntry](#k8s-io-api-core-v1-VolumeResourceRequirements-LimitsEntry) | repeated | Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/ +optional |
| requests | [VolumeResourceRequirements.RequestsEntry](#k8s-io-api-core-v1-VolumeResourceRequirements-RequestsEntry) | repeated | Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. Requests cannot exceed Limits. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/ +optional |






<a name="k8s-io-api-core-v1-VolumeResourceRequirements-LimitsEntry"></a>

### VolumeResourceRequirements.LimitsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [k8s.io.apimachinery.pkg.api.resource.Quantity](#k8s-io-apimachinery-pkg-api-resource-Quantity) | optional |  |






<a name="k8s-io-api-core-v1-VolumeResourceRequirements-RequestsEntry"></a>

### VolumeResourceRequirements.RequestsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [k8s.io.apimachinery.pkg.api.resource.Quantity](#k8s-io-apimachinery-pkg-api-resource-Quantity) | optional |  |






<a name="k8s-io-api-core-v1-VolumeSource"></a>

### VolumeSource
Represents the source of a volume to mount.
Only one of its members may be specified.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| hostPath | [HostPathVolumeSource](#k8s-io-api-core-v1-HostPathVolumeSource) | optional | hostPath represents a pre-existing file or directory on the host machine that is directly exposed to the container. This is generally used for system agents or other privileged things that are allowed to see the host machine. Most containers will NOT need this. More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath --- TODO(jonesdl) We need to restrict who can use host directory mounts and who can/can not mount host directories as read/write. +optional |
| emptyDir | [EmptyDirVolumeSource](#k8s-io-api-core-v1-EmptyDirVolumeSource) | optional | emptyDir represents a temporary directory that shares a pod's lifetime. More info: https://kubernetes.io/docs/concepts/storage/volumes#emptydir +optional |
| gcePersistentDisk | [GCEPersistentDiskVolumeSource](#k8s-io-api-core-v1-GCEPersistentDiskVolumeSource) | optional | gcePersistentDisk represents a GCE Disk resource that is attached to a kubelet's host machine and then exposed to the pod. Deprecated: GCEPersistentDisk is deprecated. All operations for the in-tree gcePersistentDisk type are redirected to the pd.csi.storage.gke.io CSI driver. More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk +optional |
| awsElasticBlockStore | [AWSElasticBlockStoreVolumeSource](#k8s-io-api-core-v1-AWSElasticBlockStoreVolumeSource) | optional | awsElasticBlockStore represents an AWS Disk resource that is attached to a kubelet's host machine and then exposed to the pod. Deprecated: AWSElasticBlockStore is deprecated. All operations for the in-tree awsElasticBlockStore type are redirected to the ebs.csi.aws.com CSI driver. More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore +optional |
| gitRepo | [GitRepoVolumeSource](#k8s-io-api-core-v1-GitRepoVolumeSource) | optional | gitRepo represents a git repository at a particular revision. Deprecated: GitRepo is deprecated. To provision a container with a git repo, mount an EmptyDir into an InitContainer that clones the repo using git, then mount the EmptyDir into the Pod's container. +optional |
| secret | [SecretVolumeSource](#k8s-io-api-core-v1-SecretVolumeSource) | optional | secret represents a secret that should populate this volume. More info: https://kubernetes.io/docs/concepts/storage/volumes#secret +optional |
| nfs | [NFSVolumeSource](#k8s-io-api-core-v1-NFSVolumeSource) | optional | nfs represents an NFS mount on the host that shares a pod's lifetime More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs +optional |
| iscsi | [ISCSIVolumeSource](#k8s-io-api-core-v1-ISCSIVolumeSource) | optional | iscsi represents an ISCSI Disk resource that is attached to a kubelet's host machine and then exposed to the pod. More info: https://examples.k8s.io/volumes/iscsi/README.md +optional |
| glusterfs | [GlusterfsVolumeSource](#k8s-io-api-core-v1-GlusterfsVolumeSource) | optional | glusterfs represents a Glusterfs mount on the host that shares a pod's lifetime. Deprecated: Glusterfs is deprecated and the in-tree glusterfs type is no longer supported. More info: https://examples.k8s.io/volumes/glusterfs/README.md +optional |
| persistentVolumeClaim | [PersistentVolumeClaimVolumeSource](#k8s-io-api-core-v1-PersistentVolumeClaimVolumeSource) | optional | persistentVolumeClaimVolumeSource represents a reference to a PersistentVolumeClaim in the same namespace. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims +optional |
| rbd | [RBDVolumeSource](#k8s-io-api-core-v1-RBDVolumeSource) | optional | rbd represents a Rados Block Device mount on the host that shares a pod's lifetime. Deprecated: RBD is deprecated and the in-tree rbd type is no longer supported. More info: https://examples.k8s.io/volumes/rbd/README.md +optional |
| flexVolume | [FlexVolumeSource](#k8s-io-api-core-v1-FlexVolumeSource) | optional | flexVolume represents a generic volume resource that is provisioned/attached using an exec based plugin. Deprecated: FlexVolume is deprecated. Consider using a CSIDriver instead. +optional |
| cinder | [CinderVolumeSource](#k8s-io-api-core-v1-CinderVolumeSource) | optional | cinder represents a cinder volume attached and mounted on kubelets host machine. Deprecated: Cinder is deprecated. All operations for the in-tree cinder type are redirected to the cinder.csi.openstack.org CSI driver. More info: https://examples.k8s.io/mysql-cinder-pd/README.md +optional |
| cephfs | [CephFSVolumeSource](#k8s-io-api-core-v1-CephFSVolumeSource) | optional | cephFS represents a Ceph FS mount on the host that shares a pod's lifetime. Deprecated: CephFS is deprecated and the in-tree cephfs type is no longer supported. +optional |
| flocker | [FlockerVolumeSource](#k8s-io-api-core-v1-FlockerVolumeSource) | optional | flocker represents a Flocker volume attached to a kubelet's host machine. This depends on the Flocker control service being running. Deprecated: Flocker is deprecated and the in-tree flocker type is no longer supported. +optional |
| downwardAPI | [DownwardAPIVolumeSource](#k8s-io-api-core-v1-DownwardAPIVolumeSource) | optional | downwardAPI represents downward API about the pod that should populate this volume +optional |
| fc | [FCVolumeSource](#k8s-io-api-core-v1-FCVolumeSource) | optional | fc represents a Fibre Channel resource that is attached to a kubelet's host machine and then exposed to the pod. +optional |
| azureFile | [AzureFileVolumeSource](#k8s-io-api-core-v1-AzureFileVolumeSource) | optional | azureFile represents an Azure File Service mount on the host and bind mount to the pod. Deprecated: AzureFile is deprecated. All operations for the in-tree azureFile type are redirected to the file.csi.azure.com CSI driver. +optional |
| configMap | [ConfigMapVolumeSource](#k8s-io-api-core-v1-ConfigMapVolumeSource) | optional | configMap represents a configMap that should populate this volume +optional |
| vsphereVolume | [VsphereVirtualDiskVolumeSource](#k8s-io-api-core-v1-VsphereVirtualDiskVolumeSource) | optional | vsphereVolume represents a vSphere volume attached and mounted on kubelets host machine. Deprecated: VsphereVolume is deprecated. All operations for the in-tree vsphereVolume type are redirected to the csi.vsphere.vmware.com CSI driver. +optional |
| quobyte | [QuobyteVolumeSource](#k8s-io-api-core-v1-QuobyteVolumeSource) | optional | quobyte represents a Quobyte mount on the host that shares a pod's lifetime. Deprecated: Quobyte is deprecated and the in-tree quobyte type is no longer supported. +optional |
| azureDisk | [AzureDiskVolumeSource](#k8s-io-api-core-v1-AzureDiskVolumeSource) | optional | azureDisk represents an Azure Data Disk mount on the host and bind mount to the pod. Deprecated: AzureDisk is deprecated. All operations for the in-tree azureDisk type are redirected to the disk.csi.azure.com CSI driver. +optional |
| photonPersistentDisk | [PhotonPersistentDiskVolumeSource](#k8s-io-api-core-v1-PhotonPersistentDiskVolumeSource) | optional | photonPersistentDisk represents a PhotonController persistent disk attached and mounted on kubelets host machine. Deprecated: PhotonPersistentDisk is deprecated and the in-tree photonPersistentDisk type is no longer supported. |
| projected | [ProjectedVolumeSource](#k8s-io-api-core-v1-ProjectedVolumeSource) | optional | projected items for all in one resources secrets, configmaps, and downward API |
| portworxVolume | [PortworxVolumeSource](#k8s-io-api-core-v1-PortworxVolumeSource) | optional | portworxVolume represents a portworx volume attached and mounted on kubelets host machine. Deprecated: PortworxVolume is deprecated. All operations for the in-tree portworxVolume type are redirected to the pxd.portworx.com CSI driver when the CSIMigrationPortworx feature-gate is on. +optional |
| scaleIO | [ScaleIOVolumeSource](#k8s-io-api-core-v1-ScaleIOVolumeSource) | optional | scaleIO represents a ScaleIO persistent volume attached and mounted on Kubernetes nodes. Deprecated: ScaleIO is deprecated and the in-tree scaleIO type is no longer supported. +optional |
| storageos | [StorageOSVolumeSource](#k8s-io-api-core-v1-StorageOSVolumeSource) | optional | storageOS represents a StorageOS volume attached and mounted on Kubernetes nodes. Deprecated: StorageOS is deprecated and the in-tree storageos type is no longer supported. +optional |
| csi | [CSIVolumeSource](#k8s-io-api-core-v1-CSIVolumeSource) | optional | csi (Container Storage Interface) represents ephemeral storage that is handled by certain external CSI drivers. +optional |
| ephemeral | [EphemeralVolumeSource](#k8s-io-api-core-v1-EphemeralVolumeSource) | optional | ephemeral represents a volume that is handled by a cluster storage driver. The volume's lifecycle is tied to the pod that defines it - it will be created before the pod starts, and deleted when the pod is removed.

Use this if: a) the volume is only needed while the pod runs, b) features of normal volumes like restoring from snapshot or capacity tracking are needed, c) the storage driver is specified through a storage class, and d) the storage driver supports dynamic volume provisioning through a PersistentVolumeClaim (see EphemeralVolumeSource for more information on the connection between this volume type and PersistentVolumeClaim).

Use PersistentVolumeClaim or one of the vendor-specific APIs for volumes that persist for longer than the lifecycle of an individual pod.

Use CSI for light-weight local ephemeral volumes if the CSI driver is meant to be used that way - see the documentation of the driver for more information.

A pod can use both types of ephemeral volumes and persistent volumes at the same time.

+optional |
| image | [ImageVolumeSource](#k8s-io-api-core-v1-ImageVolumeSource) | optional | image represents an OCI object (a container image or artifact) pulled and mounted on the kubelet's host machine. The volume is resolved at pod startup depending on which PullPolicy value is provided:

- Always: the kubelet always attempts to pull the reference. Container creation will fail If the pull fails. - Never: the kubelet never pulls the reference and only uses a local image or artifact. Container creation will fail if the reference isn't present. - IfNotPresent: the kubelet pulls if the reference isn't already present on disk. Container creation will fail if the reference isn't present and the pull fails.

The volume gets re-resolved if the pod gets deleted and recreated, which means that new remote content will become available on pod recreation. A failure to resolve or pull the image during pod startup will block containers from starting and may add significant latency. Failures will be retried using normal volume backoff and will be reported on the pod reason and message. The types of objects that may be mounted by this volume are defined by the container runtime implementation on a host machine and at minimum must include all valid types supported by the container image field. The OCI object gets mounted in a single directory (spec.containers[*].volumeMounts.mountPath) by merging the manifest layers in the same way as for container images. The volume will be mounted read-only (ro) and non-executable files (noexec). Sub path mounts for containers are not supported (spec.containers[*].volumeMounts.subpath) before 1.33. The field spec.securityContext.fsGroupChangePolicy has no effect on this volume type. +featureGate=ImageVolume +optional |






<a name="k8s-io-api-core-v1-VsphereVirtualDiskVolumeSource"></a>

### VsphereVirtualDiskVolumeSource
Represents a vSphere volume resource.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| volumePath | [string](#string) | optional | volumePath is the path that identifies vSphere volume vmdk |
| fsType | [string](#string) | optional | fsType is filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. +optional |
| storagePolicyName | [string](#string) | optional | storagePolicyName is the storage Policy Based Management (SPBM) profile name. +optional |
| storagePolicyID | [string](#string) | optional | storagePolicyID is the storage Policy Based Management (SPBM) profile ID associated with the StoragePolicyName. +optional |






<a name="k8s-io-api-core-v1-WeightedPodAffinityTerm"></a>

### WeightedPodAffinityTerm
The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| weight | [int32](#int32) | optional | weight associated with matching the corresponding podAffinityTerm, in the range 1-100. |
| podAffinityTerm | [PodAffinityTerm](#k8s-io-api-core-v1-PodAffinityTerm) | optional | Required. A pod affinity term, associated with the corresponding weight. |






<a name="k8s-io-api-core-v1-WindowsSecurityContextOptions"></a>

### WindowsSecurityContextOptions
WindowsSecurityContextOptions contain Windows-specific options and credentials.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| gmsaCredentialSpecName | [string](#string) | optional | GMSACredentialSpecName is the name of the GMSA credential spec to use. +optional |
| gmsaCredentialSpec | [string](#string) | optional | GMSACredentialSpec is where the GMSA admission webhook (https://github.com/kubernetes-sigs/windows-gmsa) inlines the contents of the GMSA credential spec named by the GMSACredentialSpecName field. +optional |
| runAsUserName | [string](#string) | optional | The UserName in Windows to run the entrypoint of the container process. Defaults to the user specified in image metadata if unspecified. May also be set in PodSecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. +optional |
| hostProcess | [bool](#bool) | optional | HostProcess determines if a container should be run as a 'Host Process' container. All of a Pod's containers must have the same effective HostProcess value (it is not allowed to have a mix of HostProcess containers and non-HostProcess containers). In addition, if HostProcess is true then HostNetwork must also be set to true. +optional |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="k8s-io_api_batch_v1_generated-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## k8s.io/api/batch/v1/generated.proto



<a name="k8s-io-api-batch-v1-CronJob"></a>

### CronJob
CronJob represents the configuration of a single cron job.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional | Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional |
| spec | [CronJobSpec](#k8s-io-api-batch-v1-CronJobSpec) | optional | Specification of the desired behavior of a cron job, including the schedule. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status +optional |
| status | [CronJobStatus](#k8s-io-api-batch-v1-CronJobStatus) | optional | Current status of a cron job. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status +optional |






<a name="k8s-io-api-batch-v1-CronJobList"></a>

### CronJobList
CronJobList is a collection of cron jobs.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional | Standard list metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional |
| items | [CronJob](#k8s-io-api-batch-v1-CronJob) | repeated | items is the list of CronJobs. |






<a name="k8s-io-api-batch-v1-CronJobSpec"></a>

### CronJobSpec
CronJobSpec describes how the job execution will look like and when it will actually run.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schedule | [string](#string) | optional | The schedule in Cron format, see https://en.wikipedia.org/wiki/Cron. |
| timeZone | [string](#string) | optional | The time zone name for the given schedule, see https://en.wikipedia.org/wiki/List_of_tz_database_time_zones. If not specified, this will default to the time zone of the kube-controller-manager process. The set of valid time zone names and the time zone offset is loaded from the system-wide time zone database by the API server during CronJob validation and the controller manager during execution. If no system-wide time zone database can be found a bundled version of the database is used instead. If the time zone name becomes invalid during the lifetime of a CronJob or due to a change in host configuration, the controller will stop creating new new Jobs and will create a system event with the reason UnknownTimeZone. More information can be found in https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/#time-zones +optional |
| startingDeadlineSeconds | [int64](#int64) | optional | Optional deadline in seconds for starting the job if it misses scheduled time for any reason. Missed jobs executions will be counted as failed ones. +optional |
| concurrencyPolicy | [string](#string) | optional | Specifies how to treat concurrent executions of a Job. Valid values are:

- "Allow" (default): allows CronJobs to run concurrently; - "Forbid": forbids concurrent runs, skipping next run if previous run hasn't finished yet; - "Replace": cancels currently running job and replaces it with a new one +optional |
| suspend | [bool](#bool) | optional | This flag tells the controller to suspend subsequent executions, it does not apply to already started executions. Defaults to false. +optional |
| jobTemplate | [JobTemplateSpec](#k8s-io-api-batch-v1-JobTemplateSpec) | optional | Specifies the job that will be created when executing a CronJob. |
| successfulJobsHistoryLimit | [int32](#int32) | optional | The number of successful finished jobs to retain. Value must be non-negative integer. Defaults to 3. +optional |
| failedJobsHistoryLimit | [int32](#int32) | optional | The number of failed finished jobs to retain. Value must be non-negative integer. Defaults to 1. +optional |






<a name="k8s-io-api-batch-v1-CronJobStatus"></a>

### CronJobStatus
CronJobStatus represents the current state of a cron job.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| active | [k8s.io.api.core.v1.ObjectReference](#k8s-io-api-core-v1-ObjectReference) | repeated | A list of pointers to currently running jobs. +optional +listType=atomic |
| lastScheduleTime | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | Information when was the last time the job was successfully scheduled. +optional |
| lastSuccessfulTime | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | Information when was the last time the job successfully completed. +optional |






<a name="k8s-io-api-batch-v1-Job"></a>

### Job
Job represents the configuration of a single job.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional | Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional |
| spec | [JobSpec](#k8s-io-api-batch-v1-JobSpec) | optional | Specification of the desired behavior of a job. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status +optional |
| status | [JobStatus](#k8s-io-api-batch-v1-JobStatus) | optional | Current status of a job. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status +optional |






<a name="k8s-io-api-batch-v1-JobCondition"></a>

### JobCondition
JobCondition describes current state of a job.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) | optional | Type of job condition, Complete or Failed. |
| status | [string](#string) | optional | Status of the condition, one of True, False, Unknown. |
| lastProbeTime | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | Last time the condition was checked. +optional |
| lastTransitionTime | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | Last time the condition transit from one status to another. +optional |
| reason | [string](#string) | optional | (brief) reason for the condition's last transition. +optional |
| message | [string](#string) | optional | Human readable message indicating details about last transition. +optional |






<a name="k8s-io-api-batch-v1-JobList"></a>

### JobList
JobList is a collection of jobs.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional | Standard list metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional |
| items | [Job](#k8s-io-api-batch-v1-Job) | repeated | items is the list of Jobs. |






<a name="k8s-io-api-batch-v1-JobSpec"></a>

### JobSpec
JobSpec describes how the job execution will look like.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parallelism | [int32](#int32) | optional | Specifies the maximum desired number of pods the job should run at any given time. The actual number of pods running in steady state will be less than this number when ((.spec.completions - .status.successful) < .spec.parallelism), i.e. when the work left to do is less than max parallelism. More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/ +optional |
| completions | [int32](#int32) | optional | Specifies the desired number of successfully finished pods the job should be run with. Setting to null means that the success of any pod signals the success of all pods, and allows parallelism to have any positive value. Setting to 1 means that parallelism is limited to 1 and the success of that pod signals the success of the job. More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/ +optional |
| activeDeadlineSeconds | [int64](#int64) | optional | Specifies the duration in seconds relative to the startTime that the job may be continuously active before the system tries to terminate it; value must be positive integer. If a Job is suspended (at creation or through an update), this timer will effectively be stopped and reset when the Job is resumed again. +optional |
| podFailurePolicy | [PodFailurePolicy](#k8s-io-api-batch-v1-PodFailurePolicy) | optional | Specifies the policy of handling failed pods. In particular, it allows to specify the set of actions and conditions which need to be satisfied to take the associated action. If empty, the default behaviour applies - the counter of failed pods, represented by the jobs's .status.failed field, is incremented and it is checked against the backoffLimit. This field cannot be used in combination with restartPolicy=OnFailure.

+optional |
| successPolicy | [SuccessPolicy](#k8s-io-api-batch-v1-SuccessPolicy) | optional | successPolicy specifies the policy when the Job can be declared as succeeded. If empty, the default behavior applies - the Job is declared as succeeded only when the number of succeeded pods equals to the completions. When the field is specified, it must be immutable and works only for the Indexed Jobs. Once the Job meets the SuccessPolicy, the lingering pods are terminated.

+optional |
| backoffLimit | [int32](#int32) | optional | Specifies the number of retries before marking this job failed. Defaults to 6 +optional |
| backoffLimitPerIndex | [int32](#int32) | optional | Specifies the limit for the number of retries within an index before marking this index as failed. When enabled the number of failures per index is kept in the pod's batch.kubernetes.io/job-index-failure-count annotation. It can only be set when Job's completionMode=Indexed, and the Pod's restart policy is Never. The field is immutable. +optional |
| maxFailedIndexes | [int32](#int32) | optional | Specifies the maximal number of failed indexes before marking the Job as failed, when backoffLimitPerIndex is set. Once the number of failed indexes exceeds this number the entire Job is marked as Failed and its execution is terminated. When left as null the job continues execution of all of its indexes and is marked with the `Complete` Job condition. It can only be specified when backoffLimitPerIndex is set. It can be null or up to completions. It is required and must be less than or equal to 10^4 when is completions greater than 10^5. +optional |
| selector | [k8s.io.apimachinery.pkg.apis.meta.v1.LabelSelector](#k8s-io-apimachinery-pkg-apis-meta-v1-LabelSelector) | optional | A label query over pods that should match the pod count. Normally, the system sets this field for you. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors +optional |
| manualSelector | [bool](#bool) | optional | manualSelector controls generation of pod labels and pod selectors. Leave `manualSelector` unset unless you are certain what you are doing. When false or unset, the system pick labels unique to this job and appends those labels to the pod template. When true, the user is responsible for picking unique labels and specifying the selector. Failure to pick a unique label may cause this and other jobs to not function correctly. However, You may see `manualSelector=true` in jobs that were created with the old `extensions/v1beta1` API. More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/#specifying-your-own-pod-selector +optional |
| template | [k8s.io.api.core.v1.PodTemplateSpec](#k8s-io-api-core-v1-PodTemplateSpec) | optional | Describes the pod that will be created when executing a job. The only allowed template.spec.restartPolicy values are "Never" or "OnFailure". More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/ |
| ttlSecondsAfterFinished | [int32](#int32) | optional | ttlSecondsAfterFinished limits the lifetime of a Job that has finished execution (either Complete or Failed). If this field is set, ttlSecondsAfterFinished after the Job finishes, it is eligible to be automatically deleted. When the Job is being deleted, its lifecycle guarantees (e.g. finalizers) will be honored. If this field is unset, the Job won't be automatically deleted. If this field is set to zero, the Job becomes eligible to be deleted immediately after it finishes. +optional |
| completionMode | [string](#string) | optional | completionMode specifies how Pod completions are tracked. It can be `NonIndexed` (default) or `Indexed`.

`NonIndexed` means that the Job is considered complete when there have been .spec.completions successfully completed Pods. Each Pod completion is homologous to each other.

`Indexed` means that the Pods of a Job get an associated completion index from 0 to (.spec.completions - 1), available in the annotation batch.kubernetes.io/job-completion-index. The Job is considered complete when there is one successfully completed Pod for each index. When value is `Indexed`, .spec.completions must be specified and `.spec.parallelism` must be less than or equal to 10^5. In addition, The Pod name takes the form `$(job-name)-$(index)-$(random-string)`, the Pod hostname takes the form `$(job-name)-$(index)`.

More completion modes can be added in the future. If the Job controller observes a mode that it doesn't recognize, which is possible during upgrades due to version skew, the controller skips updates for the Job. +optional |
| suspend | [bool](#bool) | optional | suspend specifies whether the Job controller should create Pods or not. If a Job is created with suspend set to true, no Pods are created by the Job controller. If a Job is suspended after creation (i.e. the flag goes from false to true), the Job controller will delete all active Pods associated with this Job. Users must design their workload to gracefully handle this. Suspending a Job will reset the StartTime field of the Job, effectively resetting the ActiveDeadlineSeconds timer too. Defaults to false.

+optional |
| podReplacementPolicy | [string](#string) | optional | podReplacementPolicy specifies when to create replacement Pods. Possible values are: - TerminatingOrFailed means that we recreate pods when they are terminating (has a metadata.deletionTimestamp) or failed. - Failed means to wait until a previously created Pod is fully terminated (has phase Failed or Succeeded) before creating a replacement Pod.

When using podFailurePolicy, Failed is the the only allowed value. TerminatingOrFailed and Failed are allowed values when podFailurePolicy is not in use. This is an beta field. To use this, enable the JobPodReplacementPolicy feature toggle. This is on by default. +optional |
| managedBy | [string](#string) | optional | ManagedBy field indicates the controller that manages a Job. The k8s Job controller reconciles jobs which don't have this field at all or the field value is the reserved string `kubernetes.io/job-controller`, but skips reconciling Jobs with a custom value for this field. The value must be a valid domain-prefixed path (e.g. acme.io/foo) - all characters before the first "/" must be a valid subdomain as defined by RFC 1123. All characters trailing the first "/" must be valid HTTP Path characters as defined by RFC 3986. The value cannot exceed 63 characters. This field is immutable.

This field is beta-level. The job controller accepts setting the field when the feature gate JobManagedBy is enabled (enabled by default). +optional |






<a name="k8s-io-api-batch-v1-JobStatus"></a>

### JobStatus
JobStatus represents the current state of a Job.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| conditions | [JobCondition](#k8s-io-api-batch-v1-JobCondition) | repeated | The latest available observations of an object's current state. When a Job fails, one of the conditions will have type "Failed" and status true. When a Job is suspended, one of the conditions will have type "Suspended" and status true; when the Job is resumed, the status of this condition will become false. When a Job is completed, one of the conditions will have type "Complete" and status true.

A job is considered finished when it is in a terminal condition, either "Complete" or "Failed". A Job cannot have both the "Complete" and "Failed" conditions. Additionally, it cannot be in the "Complete" and "FailureTarget" conditions. The "Complete", "Failed" and "FailureTarget" conditions cannot be disabled.

More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/ +optional +patchMergeKey=type +patchStrategy=merge +listType=atomic |
| startTime | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | Represents time when the job controller started processing a job. When a Job is created in the suspended state, this field is not set until the first time it is resumed. This field is reset every time a Job is resumed from suspension. It is represented in RFC3339 form and is in UTC.

Once set, the field can only be removed when the job is suspended. The field cannot be modified while the job is unsuspended or finished.

+optional |
| completionTime | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | Represents time when the job was completed. It is not guaranteed to be set in happens-before order across separate operations. It is represented in RFC3339 form and is in UTC. The completion time is set when the job finishes successfully, and only then. The value cannot be updated or removed. The value indicates the same or later point in time as the startTime field. +optional |
| active | [int32](#int32) | optional | The number of pending and running pods which are not terminating (without a deletionTimestamp). The value is zero for finished jobs. +optional |
| succeeded | [int32](#int32) | optional | The number of pods which reached phase Succeeded. The value increases monotonically for a given spec. However, it may decrease in reaction to scale down of elastic indexed jobs. +optional |
| failed | [int32](#int32) | optional | The number of pods which reached phase Failed. The value increases monotonically. +optional |
| terminating | [int32](#int32) | optional | The number of pods which are terminating (in phase Pending or Running and have a deletionTimestamp).

This field is beta-level. The job controller populates the field when the feature gate JobPodReplacementPolicy is enabled (enabled by default). +optional |
| completedIndexes | [string](#string) | optional | completedIndexes holds the completed indexes when .spec.completionMode = "Indexed" in a text format. The indexes are represented as decimal integers separated by commas. The numbers are listed in increasing order. Three or more consecutive numbers are compressed and represented by the first and last element of the series, separated by a hyphen. For example, if the completed indexes are 1, 3, 4, 5 and 7, they are represented as "1,3-5,7". +optional |
| failedIndexes | [string](#string) | optional | FailedIndexes holds the failed indexes when spec.backoffLimitPerIndex is set. The indexes are represented in the text format analogous as for the `completedIndexes` field, ie. they are kept as decimal integers separated by commas. The numbers are listed in increasing order. Three or more consecutive numbers are compressed and represented by the first and last element of the series, separated by a hyphen. For example, if the failed indexes are 1, 3, 4, 5 and 7, they are represented as "1,3-5,7". The set of failed indexes cannot overlap with the set of completed indexes.

+optional |
| uncountedTerminatedPods | [UncountedTerminatedPods](#k8s-io-api-batch-v1-UncountedTerminatedPods) | optional | uncountedTerminatedPods holds the UIDs of Pods that have terminated but the job controller hasn't yet accounted for in the status counters.

The job controller creates pods with a finalizer. When a pod terminates (succeeded or failed), the controller does three steps to account for it in the job status:

1. Add the pod UID to the arrays in this field. 2. Remove the pod finalizer. 3. Remove the pod UID from the arrays while increasing the corresponding counter.

Old jobs might not be tracked using this field, in which case the field remains null. The structure is empty for finished jobs. +optional |
| ready | [int32](#int32) | optional | The number of active pods which have a Ready condition and are not terminating (without a deletionTimestamp). |






<a name="k8s-io-api-batch-v1-JobTemplateSpec"></a>

### JobTemplateSpec
JobTemplateSpec describes the data a Job should have when created from a template


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional | Standard object's metadata of the jobs created from this template. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional |
| spec | [JobSpec](#k8s-io-api-batch-v1-JobSpec) | optional | Specification of the desired behavior of the job. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status +optional |






<a name="k8s-io-api-batch-v1-PodFailurePolicy"></a>

### PodFailurePolicy
PodFailurePolicy describes how failed pods influence the backoffLimit.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rules | [PodFailurePolicyRule](#k8s-io-api-batch-v1-PodFailurePolicyRule) | repeated | A list of pod failure policy rules. The rules are evaluated in order. Once a rule matches a Pod failure, the remaining of the rules are ignored. When no rule matches the Pod failure, the default handling applies - the counter of pod failures is incremented and it is checked against the backoffLimit. At most 20 elements are allowed. +listType=atomic |






<a name="k8s-io-api-batch-v1-PodFailurePolicyOnExitCodesRequirement"></a>

### PodFailurePolicyOnExitCodesRequirement
PodFailurePolicyOnExitCodesRequirement describes the requirement for handling
a failed pod based on its container exit codes. In particular, it lookups the
.state.terminated.exitCode for each app container and init container status,
represented by the .status.containerStatuses and .status.initContainerStatuses
fields in the Pod status, respectively. Containers completed with success
(exit code 0) are excluded from the requirement check.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| containerName | [string](#string) | optional | Restricts the check for exit codes to the container with the specified name. When null, the rule applies to all containers. When specified, it should match one the container or initContainer names in the pod template. +optional |
| operator | [string](#string) | optional | Represents the relationship between the container exit code(s) and the specified values. Containers completed with success (exit code 0) are excluded from the requirement check. Possible values are:

- In: the requirement is satisfied if at least one container exit code (might be multiple if there are multiple containers not restricted by the 'containerName' field) is in the set of specified values. - NotIn: the requirement is satisfied if at least one container exit code (might be multiple if there are multiple containers not restricted by the 'containerName' field) is not in the set of specified values. Additional values are considered to be added in the future. Clients should react to an unknown operator by assuming the requirement is not satisfied. |
| values | [int32](#int32) | repeated | Specifies the set of values. Each returned container exit code (might be multiple in case of multiple containers) is checked against this set of values with respect to the operator. The list of values must be ordered and must not contain duplicates. Value '0' cannot be used for the In operator. At least one element is required. At most 255 elements are allowed. +listType=set |






<a name="k8s-io-api-batch-v1-PodFailurePolicyOnPodConditionsPattern"></a>

### PodFailurePolicyOnPodConditionsPattern
PodFailurePolicyOnPodConditionsPattern describes a pattern for matching
an actual pod condition type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) | optional | Specifies the required Pod condition type. To match a pod condition it is required that specified type equals the pod condition type. |
| status | [string](#string) | optional | Specifies the required Pod condition status. To match a pod condition it is required that the specified status equals the pod condition status. Defaults to True. |






<a name="k8s-io-api-batch-v1-PodFailurePolicyRule"></a>

### PodFailurePolicyRule
PodFailurePolicyRule describes how a pod failure is handled when the requirements are met.
One of onExitCodes and onPodConditions, but not both, can be used in each rule.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| action | [string](#string) | optional | Specifies the action taken on a pod failure when the requirements are satisfied. Possible values are:

- FailJob: indicates that the pod's job is marked as Failed and all running pods are terminated. - FailIndex: indicates that the pod's index is marked as Failed and will not be restarted. - Ignore: indicates that the counter towards the .backoffLimit is not incremented and a replacement pod is created. - Count: indicates that the pod is handled in the default way - the counter towards the .backoffLimit is incremented. Additional values are considered to be added in the future. Clients should react to an unknown action by skipping the rule. |
| onExitCodes | [PodFailurePolicyOnExitCodesRequirement](#k8s-io-api-batch-v1-PodFailurePolicyOnExitCodesRequirement) | optional | Represents the requirement on the container exit codes. +optional |
| onPodConditions | [PodFailurePolicyOnPodConditionsPattern](#k8s-io-api-batch-v1-PodFailurePolicyOnPodConditionsPattern) | repeated | Represents the requirement on the pod conditions. The requirement is represented as a list of pod condition patterns. The requirement is satisfied if at least one pattern matches an actual pod condition. At most 20 elements are allowed. +listType=atomic +optional |






<a name="k8s-io-api-batch-v1-SuccessPolicy"></a>

### SuccessPolicy
SuccessPolicy describes when a Job can be declared as succeeded based on the success of some indexes.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rules | [SuccessPolicyRule](#k8s-io-api-batch-v1-SuccessPolicyRule) | repeated | rules represents the list of alternative rules for the declaring the Jobs as successful before `.status.succeeded >= .spec.completions`. Once any of the rules are met, the "SucceededCriteriaMet" condition is added, and the lingering pods are removed. The terminal state for such a Job has the "Complete" condition. Additionally, these rules are evaluated in order; Once the Job meets one of the rules, other rules are ignored. At most 20 elements are allowed. +listType=atomic |






<a name="k8s-io-api-batch-v1-SuccessPolicyRule"></a>

### SuccessPolicyRule
SuccessPolicyRule describes rule for declaring a Job as succeeded.
Each rule must have at least one of the "succeededIndexes" or "succeededCount" specified.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| succeededIndexes | [string](#string) | optional | succeededIndexes specifies the set of indexes which need to be contained in the actual set of the succeeded indexes for the Job. The list of indexes must be within 0 to ".spec.completions-1" and must not contain duplicates. At least one element is required. The indexes are represented as intervals separated by commas. The intervals can be a decimal integer or a pair of decimal integers separated by a hyphen. The number are listed in represented by the first and last element of the series, separated by a hyphen. For example, if the completed indexes are 1, 3, 4, 5 and 7, they are represented as "1,3-5,7". When this field is null, this field doesn't default to any value and is never evaluated at any time.

+optional |
| succeededCount | [int32](#int32) | optional | succeededCount specifies the minimal required size of the actual set of the succeeded indexes for the Job. When succeededCount is used along with succeededIndexes, the check is constrained only to the set of indexes specified by succeededIndexes. For example, given that succeededIndexes is "1-4", succeededCount is "3", and completed indexes are "1", "3", and "5", the Job isn't declared as succeeded because only "1" and "3" indexes are considered in that rules. When this field is null, this doesn't default to any value and is never evaluated at any time. When specified it needs to be a positive integer.

+optional |






<a name="k8s-io-api-batch-v1-UncountedTerminatedPods"></a>

### UncountedTerminatedPods
UncountedTerminatedPods holds UIDs of Pods that have terminated but haven't
been accounted in Job status counters.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| succeeded | [string](#string) | repeated | succeeded holds UIDs of succeeded Pods. +listType=set +optional |
| failed | [string](#string) | repeated | failed holds UIDs of failed Pods. +listType=set +optional |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="k8s-io_api_rbac_v1_generated-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## k8s.io/api/rbac/v1/generated.proto



<a name="k8s-io-api-rbac-v1-AggregationRule"></a>

### AggregationRule
AggregationRule describes how to locate ClusterRoles to aggregate into the ClusterRole


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| clusterRoleSelectors | [k8s.io.apimachinery.pkg.apis.meta.v1.LabelSelector](#k8s-io-apimachinery-pkg-apis-meta-v1-LabelSelector) | repeated | ClusterRoleSelectors holds a list of selectors which will be used to find ClusterRoles and create the rules. If any of the selectors match, then the ClusterRole's permissions will be added +optional +listType=atomic |






<a name="k8s-io-api-rbac-v1-ClusterRole"></a>

### ClusterRole
ClusterRole is a cluster level, logical grouping of PolicyRules that can be referenced as a unit by a RoleBinding or ClusterRoleBinding.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional | Standard object's metadata. +optional |
| rules | [PolicyRule](#k8s-io-api-rbac-v1-PolicyRule) | repeated | Rules holds all the PolicyRules for this ClusterRole +optional +listType=atomic |
| aggregationRule | [AggregationRule](#k8s-io-api-rbac-v1-AggregationRule) | optional | AggregationRule is an optional field that describes how to build the Rules for this ClusterRole. If AggregationRule is set, then the Rules are controller managed and direct changes to Rules will be stomped by the controller. +optional |






<a name="k8s-io-api-rbac-v1-ClusterRoleBinding"></a>

### ClusterRoleBinding
ClusterRoleBinding references a ClusterRole, but not contain it.  It can reference a ClusterRole in the global namespace,
and adds who information via Subject.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional | Standard object's metadata. +optional |
| subjects | [Subject](#k8s-io-api-rbac-v1-Subject) | repeated | Subjects holds references to the objects the role applies to. +optional +listType=atomic |
| roleRef | [RoleRef](#k8s-io-api-rbac-v1-RoleRef) | optional | RoleRef can only reference a ClusterRole in the global namespace. If the RoleRef cannot be resolved, the Authorizer must return an error. This field is immutable. |






<a name="k8s-io-api-rbac-v1-ClusterRoleBindingList"></a>

### ClusterRoleBindingList
ClusterRoleBindingList is a collection of ClusterRoleBindings


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional | Standard object's metadata. +optional |
| items | [ClusterRoleBinding](#k8s-io-api-rbac-v1-ClusterRoleBinding) | repeated | Items is a list of ClusterRoleBindings |






<a name="k8s-io-api-rbac-v1-ClusterRoleList"></a>

### ClusterRoleList
ClusterRoleList is a collection of ClusterRoles


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional | Standard object's metadata. +optional |
| items | [ClusterRole](#k8s-io-api-rbac-v1-ClusterRole) | repeated | Items is a list of ClusterRoles |






<a name="k8s-io-api-rbac-v1-PolicyRule"></a>

### PolicyRule
PolicyRule holds information that describes a policy rule, but does not contain information
about who the rule applies to or which namespace the rule applies to.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| verbs | [string](#string) | repeated | Verbs is a list of Verbs that apply to ALL the ResourceKinds contained in this rule. '*' represents all verbs. +listType=atomic |
| apiGroups | [string](#string) | repeated | APIGroups is the name of the APIGroup that contains the resources. If multiple API groups are specified, any action requested against one of the enumerated resources in any API group will be allowed. "" represents the core API group and "*" represents all API groups. +optional +listType=atomic |
| resources | [string](#string) | repeated | Resources is a list of resources this rule applies to. '*' represents all resources. +optional +listType=atomic |
| resourceNames | [string](#string) | repeated | ResourceNames is an optional white list of names that the rule applies to. An empty set means that everything is allowed. +optional +listType=atomic |
| nonResourceURLs | [string](#string) | repeated | NonResourceURLs is a set of partial urls that a user should have access to. *s are allowed, but only as the full, final step in the path Since non-resource URLs are not namespaced, this field is only applicable for ClusterRoles referenced from a ClusterRoleBinding. Rules can either apply to API resources (such as "pods" or "secrets") or non-resource URL paths (such as "/api"), but not both. +optional +listType=atomic |






<a name="k8s-io-api-rbac-v1-Role"></a>

### Role
Role is a namespaced, logical grouping of PolicyRules that can be referenced as a unit by a RoleBinding.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional | Standard object's metadata. +optional |
| rules | [PolicyRule](#k8s-io-api-rbac-v1-PolicyRule) | repeated | Rules holds all the PolicyRules for this Role +optional +listType=atomic |






<a name="k8s-io-api-rbac-v1-RoleBinding"></a>

### RoleBinding
RoleBinding references a role, but does not contain it.  It can reference a Role in the same namespace or a ClusterRole in the global namespace.
It adds who information via Subjects and namespace information by which namespace it exists in.  RoleBindings in a given
namespace only have effect in that namespace.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional | Standard object's metadata. +optional |
| subjects | [Subject](#k8s-io-api-rbac-v1-Subject) | repeated | Subjects holds references to the objects the role applies to. +optional +listType=atomic |
| roleRef | [RoleRef](#k8s-io-api-rbac-v1-RoleRef) | optional | RoleRef can reference a Role in the current namespace or a ClusterRole in the global namespace. If the RoleRef cannot be resolved, the Authorizer must return an error. This field is immutable. |






<a name="k8s-io-api-rbac-v1-RoleBindingList"></a>

### RoleBindingList
RoleBindingList is a collection of RoleBindings


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional | Standard object's metadata. +optional |
| items | [RoleBinding](#k8s-io-api-rbac-v1-RoleBinding) | repeated | Items is a list of RoleBindings |






<a name="k8s-io-api-rbac-v1-RoleList"></a>

### RoleList
RoleList is a collection of Roles


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional | Standard object's metadata. +optional |
| items | [Role](#k8s-io-api-rbac-v1-Role) | repeated | Items is a list of Roles |






<a name="k8s-io-api-rbac-v1-RoleRef"></a>

### RoleRef
RoleRef contains information that points to the role being used
+structType=atomic


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| apiGroup | [string](#string) | optional | APIGroup is the group for the resource being referenced |
| kind | [string](#string) | optional | Kind is the type of resource being referenced |
| name | [string](#string) | optional | Name is the name of resource being referenced |






<a name="k8s-io-api-rbac-v1-Subject"></a>

### Subject
Subject contains a reference to the object or user identities a role binding applies to.  This can either hold a direct API object reference,
or a value for non-objects such as user and group names.
+structType=atomic


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| kind | [string](#string) | optional | Kind of object being referenced. Values defined by this API group are "User", "Group", and "ServiceAccount". If the Authorizer does not recognized the kind value, the Authorizer should report an error. |
| apiGroup | [string](#string) | optional | APIGroup holds the API group of the referenced subject. Defaults to "" for ServiceAccount subjects. Defaults to "rbac.authorization.k8s.io" for User and Group subjects. +optional |
| name | [string](#string) | optional | Name of the object being referenced. |
| namespace | [string](#string) | optional | Namespace of the referenced object. If the object kind is non-namespace, such as "User" or "Group", and this value is not empty the Authorizer should report an error. +optional |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="k8s-io_apiextensions-apiserver_pkg_apis_apiextensions_v1_generated-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1/generated.proto



<a name="k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-ConversionRequest"></a>

### ConversionRequest
ConversionRequest describes the conversion request parameters.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| uid | [string](#string) | optional | uid is an identifier for the individual request/response. It allows distinguishing instances of requests which are otherwise identical (parallel requests, etc). The UID is meant to track the round trip (request/response) between the Kubernetes API server and the webhook, not the user request. It is suitable for correlating log entries between the webhook and apiserver, for either auditing or debugging. |
| desiredAPIVersion | [string](#string) | optional | desiredAPIVersion is the version to convert given objects to. e.g. "myapi.example.com/v1" |
| objects | [k8s.io.apimachinery.pkg.runtime.RawExtension](#k8s-io-apimachinery-pkg-runtime-RawExtension) | repeated | objects is the list of custom resource objects to be converted. +listType=atomic |






<a name="k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-ConversionResponse"></a>

### ConversionResponse
ConversionResponse describes a conversion response.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| uid | [string](#string) | optional | uid is an identifier for the individual request/response. This should be copied over from the corresponding `request.uid`. |
| convertedObjects | [k8s.io.apimachinery.pkg.runtime.RawExtension](#k8s-io-apimachinery-pkg-runtime-RawExtension) | repeated | convertedObjects is the list of converted version of `request.objects` if the `result` is successful, otherwise empty. The webhook is expected to set `apiVersion` of these objects to the `request.desiredAPIVersion`. The list must also have the same size as the input list with the same objects in the same order (equal kind, metadata.uid, metadata.name and metadata.namespace). The webhook is allowed to mutate labels and annotations. Any other change to the metadata is silently ignored. +listType=atomic |
| result | [k8s.io.apimachinery.pkg.apis.meta.v1.Status](#k8s-io-apimachinery-pkg-apis-meta-v1-Status) | optional | result contains the result of conversion with extra details if the conversion failed. `result.status` determines if the conversion failed or succeeded. The `result.status` field is required and represents the success or failure of the conversion. A successful conversion must set `result.status` to `Success`. A failed conversion must set `result.status` to `Failure` and provide more details in `result.message` and return http status 200. The `result.message` will be used to construct an error message for the end user. |






<a name="k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-ConversionReview"></a>

### ConversionReview
ConversionReview describes a conversion request/response.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| request | [ConversionRequest](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-ConversionRequest) | optional | request describes the attributes for the conversion request. +optional |
| response | [ConversionResponse](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-ConversionResponse) | optional | response describes the attributes for the conversion response. +optional |






<a name="k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceColumnDefinition"></a>

### CustomResourceColumnDefinition
CustomResourceColumnDefinition specifies a column for server side printing.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | name is a human readable name for the column. |
| type | [string](#string) | optional | type is an OpenAPI type definition for this column. See https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#data-types for details. |
| format | [string](#string) | optional | format is an optional OpenAPI type definition for this column. The 'name' format is applied to the primary identifier column to assist in clients identifying column is the resource name. See https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#data-types for details. +optional |
| description | [string](#string) | optional | description is a human readable description of this column. +optional |
| priority | [int32](#int32) | optional | priority is an integer defining the relative importance of this column compared to others. Lower numbers are considered higher priority. Columns that may be omitted in limited space scenarios should be given a priority greater than 0. +optional |
| jsonPath | [string](#string) | optional | jsonPath is a simple JSON path (i.e. with array notation) which is evaluated against each custom resource to produce the value for this column. |






<a name="k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceConversion"></a>

### CustomResourceConversion
CustomResourceConversion describes how to convert different versions of a CR.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| strategy | [string](#string) | optional | strategy specifies how custom resources are converted between versions. Allowed values are: - `"None"`: The converter only change the apiVersion and would not touch any other field in the custom resource. - `"Webhook"`: API Server will call to an external webhook to do the conversion. Additional information is needed for this option. This requires spec.preserveUnknownFields to be false, and spec.conversion.webhook to be set. |
| webhook | [WebhookConversion](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-WebhookConversion) | optional | webhook describes how to call the conversion webhook. Required when `strategy` is set to `"Webhook"`. +optional |






<a name="k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceDefinition"></a>

### CustomResourceDefinition
CustomResourceDefinition represents a resource that should be exposed on the API server.  Its name MUST be in the format
<.spec.name>.<.spec.group>.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional | Standard object's metadata More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional |
| spec | [CustomResourceDefinitionSpec](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceDefinitionSpec) | optional | spec describes how the user wants the resources to appear |
| status | [CustomResourceDefinitionStatus](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceDefinitionStatus) | optional | status indicates the actual state of the CustomResourceDefinition +optional |






<a name="k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceDefinitionCondition"></a>

### CustomResourceDefinitionCondition
CustomResourceDefinitionCondition contains details for the current condition of this pod.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) | optional | type is the type of the condition. Types include Established, NamesAccepted and Terminating. |
| status | [string](#string) | optional | status is the status of the condition. Can be True, False, Unknown. |
| lastTransitionTime | [k8s.io.apimachinery.pkg.apis.meta.v1.Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | lastTransitionTime last time the condition transitioned from one status to another. +optional |
| reason | [string](#string) | optional | reason is a unique, one-word, CamelCase reason for the condition's last transition. +optional |
| message | [string](#string) | optional | message is a human-readable message indicating details about last transition. +optional |






<a name="k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceDefinitionList"></a>

### CustomResourceDefinitionList
CustomResourceDefinitionList is a list of CustomResourceDefinition objects.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional | Standard object's metadata More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional |
| items | [CustomResourceDefinition](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceDefinition) | repeated | items list individual CustomResourceDefinition objects |






<a name="k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceDefinitionNames"></a>

### CustomResourceDefinitionNames
CustomResourceDefinitionNames indicates the names to serve this CustomResourceDefinition


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| plural | [string](#string) | optional | plural is the plural name of the resource to serve. The custom resources are served under `/apis/<group>/<version>/.../<plural>`. Must match the name of the CustomResourceDefinition (in the form `<names.plural>.<group>`). Must be all lowercase. |
| singular | [string](#string) | optional | singular is the singular name of the resource. It must be all lowercase. Defaults to lowercased `kind`. +optional |
| shortNames | [string](#string) | repeated | shortNames are short names for the resource, exposed in API discovery documents, and used by clients to support invocations like `kubectl get <shortname>`. It must be all lowercase. +optional +listType=atomic |
| kind | [string](#string) | optional | kind is the serialized kind of the resource. It is normally CamelCase and singular. Custom resource instances will use this value as the `kind` attribute in API calls. |
| listKind | [string](#string) | optional | listKind is the serialized kind of the list for this resource. Defaults to "`kind`List". +optional |
| categories | [string](#string) | repeated | categories is a list of grouped resources this custom resource belongs to (e.g. 'all'). This is published in API discovery documents, and used by clients to support invocations like `kubectl get all`. +optional +listType=atomic |






<a name="k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceDefinitionSpec"></a>

### CustomResourceDefinitionSpec
CustomResourceDefinitionSpec describes how a user wants their resource to appear


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| group | [string](#string) | optional | group is the API group of the defined custom resource. The custom resources are served under `/apis/<group>/...`. Must match the name of the CustomResourceDefinition (in the form `<names.plural>.<group>`). |
| names | [CustomResourceDefinitionNames](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceDefinitionNames) | optional | names specify the resource and kind names for the custom resource. |
| scope | [string](#string) | optional | scope indicates whether the defined custom resource is cluster- or namespace-scoped. Allowed values are `Cluster` and `Namespaced`. |
| versions | [CustomResourceDefinitionVersion](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceDefinitionVersion) | repeated | versions is the list of all API versions of the defined custom resource. Version names are used to compute the order in which served versions are listed in API discovery. If the version string is "kube-like", it will sort above non "kube-like" version strings, which are ordered lexicographically. "Kube-like" versions start with a "v", then are followed by a number (the major version), then optionally the string "alpha" or "beta" and another number (the minor version). These are sorted first by GA > beta > alpha (where GA is a version with no suffix such as beta or alpha), and then by comparing major version, then minor version. An example sorted list of versions: v10, v2, v1, v11beta2, v10beta3, v3beta1, v12alpha1, v11alpha2, foo1, foo10. +listType=atomic |
| conversion | [CustomResourceConversion](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceConversion) | optional | conversion defines conversion settings for the CRD. +optional |
| preserveUnknownFields | [bool](#bool) | optional | preserveUnknownFields indicates that object fields which are not specified in the OpenAPI schema should be preserved when persisting to storage. apiVersion, kind, metadata and known fields inside metadata are always preserved. This field is deprecated in favor of setting `x-preserve-unknown-fields` to true in `spec.versions[*].schema.openAPIV3Schema`. See https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#field-pruning for details. +optional |






<a name="k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceDefinitionStatus"></a>

### CustomResourceDefinitionStatus
CustomResourceDefinitionStatus indicates the state of the CustomResourceDefinition


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| conditions | [CustomResourceDefinitionCondition](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceDefinitionCondition) | repeated | conditions indicate state for particular aspects of a CustomResourceDefinition +optional +listType=map +listMapKey=type |
| acceptedNames | [CustomResourceDefinitionNames](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceDefinitionNames) | optional | acceptedNames are the names that are actually being used to serve discovery. They may be different than the names in spec. +optional |
| storedVersions | [string](#string) | repeated | storedVersions lists all versions of CustomResources that were ever persisted. Tracking these versions allows a migration path for stored versions in etcd. The field is mutable so a migration controller can finish a migration to another version (ensuring no old objects are left in storage), and then remove the rest of the versions from this list. Versions may not be removed from `spec.versions` while they exist in this list. +optional +listType=atomic |






<a name="k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceDefinitionVersion"></a>

### CustomResourceDefinitionVersion
CustomResourceDefinitionVersion describes a version for CRD.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | name is the version name, e.g. “v1”, “v2beta1”, etc. The custom resources are served under this version at `/apis/<group>/<version>/...` if `served` is true. |
| served | [bool](#bool) | optional | served is a flag enabling/disabling this version from being served via REST APIs |
| storage | [bool](#bool) | optional | storage indicates this version should be used when persisting custom resources to storage. There must be exactly one version with storage=true. |
| deprecated | [bool](#bool) | optional | deprecated indicates this version of the custom resource API is deprecated. When set to true, API requests to this version receive a warning header in the server response. Defaults to false. +optional |
| deprecationWarning | [string](#string) | optional | deprecationWarning overrides the default warning returned to API clients. May only be set when `deprecated` is true. The default warning indicates this version is deprecated and recommends use of the newest served version of equal or greater stability, if one exists. +optional |
| schema | [CustomResourceValidation](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceValidation) | optional | schema describes the schema used for validation, pruning, and defaulting of this version of the custom resource. +optional |
| subresources | [CustomResourceSubresources](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceSubresources) | optional | subresources specify what subresources this version of the defined custom resource have. +optional |
| additionalPrinterColumns | [CustomResourceColumnDefinition](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceColumnDefinition) | repeated | additionalPrinterColumns specifies additional columns returned in Table output. See https://kubernetes.io/docs/reference/using-api/api-concepts/#receiving-resources-as-tables for details. If no columns are specified, a single column displaying the age of the custom resource is used. +optional +listType=atomic |
| selectableFields | [SelectableField](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-SelectableField) | repeated | selectableFields specifies paths to fields that may be used as field selectors. A maximum of 8 selectable fields are allowed. See https://kubernetes.io/docs/concepts/overview/working-with-objects/field-selectors

+featureGate=CustomResourceFieldSelectors +optional +listType=atomic |






<a name="k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceSubresourceScale"></a>

### CustomResourceSubresourceScale
CustomResourceSubresourceScale defines how to serve the scale subresource for CustomResources.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| specReplicasPath | [string](#string) | optional | specReplicasPath defines the JSON path inside of a custom resource that corresponds to Scale `spec.replicas`. Only JSON paths without the array notation are allowed. Must be a JSON Path under `.spec`. If there is no value under the given path in the custom resource, the `/scale` subresource will return an error on GET. |
| statusReplicasPath | [string](#string) | optional | statusReplicasPath defines the JSON path inside of a custom resource that corresponds to Scale `status.replicas`. Only JSON paths without the array notation are allowed. Must be a JSON Path under `.status`. If there is no value under the given path in the custom resource, the `status.replicas` value in the `/scale` subresource will default to 0. |
| labelSelectorPath | [string](#string) | optional | labelSelectorPath defines the JSON path inside of a custom resource that corresponds to Scale `status.selector`. Only JSON paths without the array notation are allowed. Must be a JSON Path under `.status` or `.spec`. Must be set to work with HorizontalPodAutoscaler. The field pointed by this JSON path must be a string field (not a complex selector struct) which contains a serialized label selector in string form. More info: https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions#scale-subresource If there is no value under the given path in the custom resource, the `status.selector` value in the `/scale` subresource will default to the empty string. +optional |






<a name="k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceSubresourceStatus"></a>

### CustomResourceSubresourceStatus
CustomResourceSubresourceStatus defines how to serve the status subresource for CustomResources.
Status is represented by the `.status` JSON path inside of a CustomResource. When set,
* exposes a /status subresource for the custom resource
* PUT requests to the /status subresource take a custom resource object, and ignore changes to anything except the status stanza
* PUT/POST/PATCH requests to the custom resource ignore changes to the status stanza






<a name="k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceSubresources"></a>

### CustomResourceSubresources
CustomResourceSubresources defines the status and scale subresources for CustomResources.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [CustomResourceSubresourceStatus](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceSubresourceStatus) | optional | status indicates the custom resource should serve a `/status` subresource. When enabled: 1. requests to the custom resource primary endpoint ignore changes to the `status` stanza of the object. 2. requests to the custom resource `/status` subresource ignore changes to anything other than the `status` stanza of the object. +optional |
| scale | [CustomResourceSubresourceScale](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceSubresourceScale) | optional | scale indicates the custom resource should serve a `/scale` subresource that returns an `autoscaling/v1` Scale object. +optional |






<a name="k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-CustomResourceValidation"></a>

### CustomResourceValidation
CustomResourceValidation is a list of validation methods for CustomResources.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| openAPIV3Schema | [JSONSchemaProps](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaProps) | optional | openAPIV3Schema is the OpenAPI v3 schema to use for validation and pruning. +optional |






<a name="k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-ExternalDocumentation"></a>

### ExternalDocumentation
ExternalDocumentation allows referencing an external resource for extended documentation.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| description | [string](#string) | optional |  |
| url | [string](#string) | optional |  |






<a name="k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSON"></a>

### JSON
JSON represents any valid JSON value.
These types are supported: bool, int64, float64, string, []interface{}, map[string]interface{} and nil.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| raw | [bytes](#bytes) | optional |  |






<a name="k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaProps"></a>

### JSONSchemaProps
JSONSchemaProps is a JSON-Schema following Specification Draft 4 (http://json-schema.org/).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) | optional |  |
| schema | [string](#string) | optional |  |
| ref | [string](#string) | optional |  |
| description | [string](#string) | optional |  |
| type | [string](#string) | optional |  |
| format | [string](#string) | optional | format is an OpenAPI v3 format string. Unknown formats are ignored. The following formats are validated:

- bsonobjectid: a bson object ID, i.e. a 24 characters hex string - uri: an URI as parsed by Golang net/url.ParseRequestURI - email: an email address as parsed by Golang net/mail.ParseAddress - hostname: a valid representation for an Internet host name, as defined by RFC 1034, section 3.1 [RFC1034]. - ipv4: an IPv4 IP as parsed by Golang net.ParseIP - ipv6: an IPv6 IP as parsed by Golang net.ParseIP - cidr: a CIDR as parsed by Golang net.ParseCIDR - mac: a MAC address as parsed by Golang net.ParseMAC - uuid: an UUID that allows uppercase defined by the regex (?i)^[0-9a-f]{8}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{12}$ - uuid3: an UUID3 that allows uppercase defined by the regex (?i)^[0-9a-f]{8}-?[0-9a-f]{4}-?3[0-9a-f]{3}-?[0-9a-f]{4}-?[0-9a-f]{12}$ - uuid4: an UUID4 that allows uppercase defined by the regex (?i)^[0-9a-f]{8}-?[0-9a-f]{4}-?4[0-9a-f]{3}-?[89ab][0-9a-f]{3}-?[0-9a-f]{12}$ - uuid5: an UUID5 that allows uppercase defined by the regex (?i)^[0-9a-f]{8}-?[0-9a-f]{4}-?5[0-9a-f]{3}-?[89ab][0-9a-f]{3}-?[0-9a-f]{12}$ - isbn: an ISBN10 or ISBN13 number string like "0321751043" or "978-0321751041" - isbn10: an ISBN10 number string like "0321751043" - isbn13: an ISBN13 number string like "978-0321751041" - creditcard: a credit card number defined by the regex ^(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|6(?:011|5[0-9][0-9])[0-9]{12}|3[47][0-9]{13}|3(?:0[0-5]|[68][0-9])[0-9]{11}|(?:2131|1800|35\\d{3})\\d{11})$ with any non digit characters mixed in - ssn: a U.S. social security number following the regex ^\\d{3}[- ]?\\d{2}[- ]?\\d{4}$ - hexcolor: an hexadecimal color code like "#FFFFFF: following the regex ^#?([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$ - rgbcolor: an RGB color code like rgb like "rgb(255,255,2559" - byte: base64 encoded binary data - password: any kind of string - date: a date string like "2006-01-02" as defined by full-date in RFC3339 - duration: a duration string like "22 ns" as parsed by Golang time.ParseDuration or compatible with Scala duration format - datetime: a date time string like "2014-12-15T19:30:20.000Z" as defined by date-time in RFC3339. |
| title | [string](#string) | optional |  |
| default | [JSON](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSON) | optional | default is a default value for undefined object fields. Defaulting is a beta feature under the CustomResourceDefaulting feature gate. Defaulting requires spec.preserveUnknownFields to be false. |
| maximum | [double](#double) | optional |  |
| exclusiveMaximum | [bool](#bool) | optional |  |
| minimum | [double](#double) | optional |  |
| exclusiveMinimum | [bool](#bool) | optional |  |
| maxLength | [int64](#int64) | optional |  |
| minLength | [int64](#int64) | optional |  |
| pattern | [string](#string) | optional |  |
| maxItems | [int64](#int64) | optional |  |
| minItems | [int64](#int64) | optional |  |
| uniqueItems | [bool](#bool) | optional |  |
| multipleOf | [double](#double) | optional |  |
| enum | [JSON](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSON) | repeated | +listType=atomic |
| maxProperties | [int64](#int64) | optional |  |
| minProperties | [int64](#int64) | optional |  |
| required | [string](#string) | repeated | +listType=atomic |
| items | [JSONSchemaPropsOrArray](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaPropsOrArray) | optional |  |
| allOf | [JSONSchemaProps](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaProps) | repeated | +listType=atomic |
| oneOf | [JSONSchemaProps](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaProps) | repeated | +listType=atomic |
| anyOf | [JSONSchemaProps](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaProps) | repeated | +listType=atomic |
| not | [JSONSchemaProps](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaProps) | optional |  |
| properties | [JSONSchemaProps.PropertiesEntry](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaProps-PropertiesEntry) | repeated |  |
| additionalProperties | [JSONSchemaPropsOrBool](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaPropsOrBool) | optional |  |
| patternProperties | [JSONSchemaProps.PatternPropertiesEntry](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaProps-PatternPropertiesEntry) | repeated |  |
| dependencies | [JSONSchemaProps.DependenciesEntry](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaProps-DependenciesEntry) | repeated |  |
| additionalItems | [JSONSchemaPropsOrBool](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaPropsOrBool) | optional |  |
| definitions | [JSONSchemaProps.DefinitionsEntry](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaProps-DefinitionsEntry) | repeated |  |
| externalDocs | [ExternalDocumentation](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-ExternalDocumentation) | optional |  |
| example | [JSON](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSON) | optional |  |
| nullable | [bool](#bool) | optional |  |
| xKubernetesPreserveUnknownFields | [bool](#bool) | optional | x-kubernetes-preserve-unknown-fields stops the API server decoding step from pruning fields which are not specified in the validation schema. This affects fields recursively, but switches back to normal pruning behaviour if nested properties or additionalProperties are specified in the schema. This can either be true or undefined. False is forbidden. |
| xKubernetesEmbeddedResource | [bool](#bool) | optional | x-kubernetes-embedded-resource defines that the value is an embedded Kubernetes runtime.Object, with TypeMeta and ObjectMeta. The type must be object. It is allowed to further restrict the embedded object. kind, apiVersion and metadata are validated automatically. x-kubernetes-preserve-unknown-fields is allowed to be true, but does not have to be if the object is fully specified (up to kind, apiVersion, metadata). |
| xKubernetesIntOrString | [bool](#bool) | optional | x-kubernetes-int-or-string specifies that this value is either an integer or a string. If this is true, an empty type is allowed and type as child of anyOf is permitted if following one of the following patterns:

1) anyOf: - type: integer - type: string 2) allOf: - anyOf: - type: integer - type: string - ... zero or more |
| xKubernetesListMapKeys | [string](#string) | repeated | x-kubernetes-list-map-keys annotates an array with the x-kubernetes-list-type `map` by specifying the keys used as the index of the map.

This tag MUST only be used on lists that have the "x-kubernetes-list-type" extension set to "map". Also, the values specified for this attribute must be a scalar typed field of the child structure (no nesting is supported).

The properties specified must either be required or have a default value, to ensure those properties are present for all list items.

+optional +listType=atomic |
| xKubernetesListType | [string](#string) | optional | x-kubernetes-list-type annotates an array to further describe its topology. This extension must only be used on lists and may have 3 possible values:

1) `atomic`: the list is treated as a single entity, like a scalar. Atomic lists will be entirely replaced when updated. This extension may be used on any type of list (struct, scalar, ...). 2) `set`: Sets are lists that must not have multiple items with the same value. Each value must be a scalar, an object with x-kubernetes-map-type `atomic` or an array with x-kubernetes-list-type `atomic`. 3) `map`: These lists are like maps in that their elements have a non-index key used to identify them. Order is preserved upon merge. The map tag must only be used on a list with elements of type object. Defaults to atomic for arrays. +optional |
| xKubernetesMapType | [string](#string) | optional | x-kubernetes-map-type annotates an object to further describe its topology. This extension must only be used when type is object and may have 2 possible values:

1) `granular`: These maps are actual maps (key-value pairs) and each fields are independent from each other (they can each be manipulated by separate actors). This is the default behaviour for all maps. 2) `atomic`: the list is treated as a single entity, like a scalar. Atomic maps will be entirely replaced when updated. +optional |
| xKubernetesValidations | [ValidationRule](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-ValidationRule) | repeated | x-kubernetes-validations describes a list of validation rules written in the CEL expression language. +patchMergeKey=rule +patchStrategy=merge +listType=map +listMapKey=rule |






<a name="k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaProps-DefinitionsEntry"></a>

### JSONSchemaProps.DefinitionsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [JSONSchemaProps](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaProps) | optional |  |






<a name="k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaProps-DependenciesEntry"></a>

### JSONSchemaProps.DependenciesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [JSONSchemaPropsOrStringArray](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaPropsOrStringArray) | optional |  |






<a name="k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaProps-PatternPropertiesEntry"></a>

### JSONSchemaProps.PatternPropertiesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [JSONSchemaProps](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaProps) | optional |  |






<a name="k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaProps-PropertiesEntry"></a>

### JSONSchemaProps.PropertiesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [JSONSchemaProps](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaProps) | optional |  |






<a name="k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaPropsOrArray"></a>

### JSONSchemaPropsOrArray
JSONSchemaPropsOrArray represents a value that can either be a JSONSchemaProps
or an array of JSONSchemaProps. Mainly here for serialization purposes.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema | [JSONSchemaProps](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaProps) | optional |  |
| jSONSchemas | [JSONSchemaProps](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaProps) | repeated | +listType=atomic |






<a name="k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaPropsOrBool"></a>

### JSONSchemaPropsOrBool
JSONSchemaPropsOrBool represents JSONSchemaProps or a boolean value.
Defaults to true for the boolean property.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| allows | [bool](#bool) | optional |  |
| schema | [JSONSchemaProps](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaProps) | optional |  |






<a name="k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaPropsOrStringArray"></a>

### JSONSchemaPropsOrStringArray
JSONSchemaPropsOrStringArray represents a JSONSchemaProps or a string array.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema | [JSONSchemaProps](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-JSONSchemaProps) | optional |  |
| property | [string](#string) | repeated | +listType=atomic |






<a name="k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-SelectableField"></a>

### SelectableField
SelectableField specifies the JSON path of a field that may be used with field selectors.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| jsonPath | [string](#string) | optional | jsonPath is a simple JSON path which is evaluated against each custom resource to produce a field selector value. Only JSON paths without the array notation are allowed. Must point to a field of type string, boolean or integer. Types with enum values and strings with formats are allowed. If jsonPath refers to absent field in a resource, the jsonPath evaluates to an empty string. Must not point to metdata fields. Required. |






<a name="k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-ServiceReference"></a>

### ServiceReference
ServiceReference holds a reference to Service.legacy.k8s.io


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| namespace | [string](#string) | optional | namespace is the namespace of the service. Required |
| name | [string](#string) | optional | name is the name of the service. Required |
| path | [string](#string) | optional | path is an optional URL path at which the webhook will be contacted. +optional |
| port | [int32](#int32) | optional | port is an optional service port at which the webhook will be contacted. `port` should be a valid port number (1-65535, inclusive). Defaults to 443 for backward compatibility. +optional |






<a name="k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-ValidationRule"></a>

### ValidationRule
ValidationRule describes a validation rule written in the CEL expression language.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rule | [string](#string) | optional | Rule represents the expression which will be evaluated by CEL. ref: https://github.com/google/cel-spec The Rule is scoped to the location of the x-kubernetes-validations extension in the schema. The `self` variable in the CEL expression is bound to the scoped value. Example: - Rule scoped to the root of a resource with a status subresource: {"rule": "self.status.actual <= self.spec.maxDesired"}

If the Rule is scoped to an object with properties, the accessible properties of the object are field selectable via `self.field` and field presence can be checked via `has(self.field)`. Null valued fields are treated as absent fields in CEL expressions. If the Rule is scoped to an object with additionalProperties (i.e. a map) the value of the map are accessible via `self[mapKey]`, map containment can be checked via `mapKey in self` and all entries of the map are accessible via CEL macros and functions such as `self.all(...)`. If the Rule is scoped to an array, the elements of the array are accessible via `self[i]` and also by macros and functions. If the Rule is scoped to a scalar, `self` is bound to the scalar value. Examples: - Rule scoped to a map of objects: {"rule": "self.components['Widget'].priority < 10"} - Rule scoped to a list of integers: {"rule": "self.values.all(value, value >= 0 && value < 100)"} - Rule scoped to a string value: {"rule": "self.startsWith('kube')"}

The `apiVersion`, `kind`, `metadata.name` and `metadata.generateName` are always accessible from the root of the object and from any x-kubernetes-embedded-resource annotated objects. No other metadata properties are accessible.

Unknown data preserved in custom resources via x-kubernetes-preserve-unknown-fields is not accessible in CEL expressions. This includes: - Unknown field values that are preserved by object schemas with x-kubernetes-preserve-unknown-fields. - Object properties where the property schema is of an "unknown type". An "unknown type" is recursively defined as: - A schema with no type and x-kubernetes-preserve-unknown-fields set to true - An array where the items schema is of an "unknown type" - An object where the additionalProperties schema is of an "unknown type"

Only property names of the form `[a-zA-Z_.-/][a-zA-Z0-9_.-/]*` are accessible. Accessible property names are escaped according to the following rules when accessed in the expression: - '__' escapes to '__underscores__' - '.' escapes to '__dot__' - '-' escapes to '__dash__' - '/' escapes to '__slash__' - Property names that exactly match a CEL RESERVED keyword escape to '__{keyword}__'. The keywords are: 	 "true", "false", "null", "in", "as", "break", "const", "continue", "else", "for", "function", "if", 	 "import", "let", "loop", "package", "namespace", "return". Examples: - Rule accessing a property named "namespace": {"rule": "self.__namespace__ > 0"} - Rule accessing a property named "x-prop": {"rule": "self.x__dash__prop > 0"} - Rule accessing a property named "redact__d": {"rule": "self.redact__underscores__d > 0"}

Equality on arrays with x-kubernetes-list-type of 'set' or 'map' ignores element order, i.e. [1, 2] == [2, 1]. Concatenation on arrays with x-kubernetes-list-type use the semantics of the list type: - 'set': `X + Y` performs a union where the array positions of all elements in `X` are preserved and non-intersecting elements in `Y` are appended, retaining their partial order. - 'map': `X + Y` performs a merge where the array positions of all keys in `X` are preserved but the values are overwritten by values in `Y` when the key sets of `X` and `Y` intersect. Elements in `Y` with non-intersecting keys are appended, retaining their partial order.

If `rule` makes use of the `oldSelf` variable it is implicitly a `transition rule`.

By default, the `oldSelf` variable is the same type as `self`. When `optionalOldSelf` is true, the `oldSelf` variable is a CEL optional variable whose value() is the same type as `self`. See the documentation for the `optionalOldSelf` field for details.

Transition rules by default are applied only on UPDATE requests and are skipped if an old value could not be found. You can opt a transition rule into unconditional evaluation by setting `optionalOldSelf` to true. |
| message | [string](#string) | optional | Message represents the message displayed when validation fails. The message is required if the Rule contains line breaks. The message must not contain line breaks. If unset, the message is "failed rule: {Rule}". e.g. "must be a URL with the host matching spec.host" |
| messageExpression | [string](#string) | optional | MessageExpression declares a CEL expression that evaluates to the validation failure message that is returned when this rule fails. Since messageExpression is used as a failure message, it must evaluate to a string. If both message and messageExpression are present on a rule, then messageExpression will be used if validation fails. If messageExpression results in a runtime error, the runtime error is logged, and the validation failure message is produced as if the messageExpression field were unset. If messageExpression evaluates to an empty string, a string with only spaces, or a string that contains line breaks, then the validation failure message will also be produced as if the messageExpression field were unset, and the fact that messageExpression produced an empty string/string with only spaces/string with line breaks will be logged. messageExpression has access to all the same variables as the rule; the only difference is the return type. Example: "x must be less than max ("+string(self.max)+")" +optional |
| reason | [string](#string) | optional | reason provides a machine-readable validation failure reason that is returned to the caller when a request fails this validation rule. The HTTP status code returned to the caller will match the reason of the reason of the first failed validation rule. The currently supported reasons are: "FieldValueInvalid", "FieldValueForbidden", "FieldValueRequired", "FieldValueDuplicate". If not set, default to use "FieldValueInvalid". All future added reasons must be accepted by clients when reading this value and unknown reasons should be treated as FieldValueInvalid. +optional |
| fieldPath | [string](#string) | optional | fieldPath represents the field path returned when the validation fails. It must be a relative JSON path (i.e. with array notation) scoped to the location of this x-kubernetes-validations extension in the schema and refer to an existing field. e.g. when validation checks if a specific attribute `foo` under a map `testMap`, the fieldPath could be set to `.testMap.foo` If the validation checks two lists must have unique attributes, the fieldPath could be set to either of the list: e.g. `.testList` It does not support list numeric index. It supports child operation to refer to an existing field currently. Refer to [JSONPath support in Kubernetes](https://kubernetes.io/docs/reference/kubectl/jsonpath/) for more info. Numeric index of array is not supported. For field name which contains special characters, use `['specialName']` to refer the field name. e.g. for attribute `foo.34$` appears in a list `testList`, the fieldPath could be set to `.testList['foo.34$']` +optional |
| optionalOldSelf | [bool](#bool) | optional | optionalOldSelf is used to opt a transition rule into evaluation even when the object is first created, or if the old object is missing the value.

When enabled `oldSelf` will be a CEL optional whose value will be `None` if there is no old value, or when the object is initially created.

You may check for presence of oldSelf using `oldSelf.hasValue()` and unwrap it after checking using `oldSelf.value()`. Check the CEL documentation for Optional types for more information: https://pkg.go.dev/github.com/google/cel-go/cel#OptionalTypes

May not be set unless `oldSelf` is used in `rule`.

+featureGate=CRDValidationRatcheting +optional |






<a name="k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-WebhookClientConfig"></a>

### WebhookClientConfig
WebhookClientConfig contains the information to make a TLS connection with the webhook.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| url | [string](#string) | optional | url gives the location of the webhook, in standard URL form (`scheme://host:port/path`). Exactly one of `url` or `service` must be specified.

The `host` should not refer to a service running in the cluster; use the `service` field instead. The host might be resolved via external DNS in some apiservers (e.g., `kube-apiserver` cannot resolve in-cluster DNS as that would be a layering violation). `host` may also be an IP address.

Please note that using `localhost` or `127.0.0.1` as a `host` is risky unless you take great care to run this webhook on all hosts which run an apiserver which might need to make calls to this webhook. Such installs are likely to be non-portable, i.e., not easy to turn up in a new cluster.

The scheme must be "https"; the URL must begin with "https://".

A path is optional, and if present may be any string permissible in a URL. You may use the path to pass an arbitrary string to the webhook, for example, a cluster identifier.

Attempting to use a user or basic auth e.g. "user:password@" is not allowed. Fragments ("#...") and query parameters ("?...") are not allowed, either.

+optional |
| service | [ServiceReference](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-ServiceReference) | optional | service is a reference to the service for this webhook. Either service or url must be specified.

If the webhook is running within the cluster, then you should use `service`.

+optional |
| caBundle | [bytes](#bytes) | optional | caBundle is a PEM encoded CA bundle which will be used to validate the webhook's server certificate. If unspecified, system trust roots on the apiserver are used. +optional |






<a name="k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-WebhookConversion"></a>

### WebhookConversion
WebhookConversion describes how to call a conversion webhook


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| clientConfig | [WebhookClientConfig](#k8s-io-apiextensions_apiserver-pkg-apis-apiextensions-v1-WebhookClientConfig) | optional | clientConfig is the instructions for how to call the webhook if strategy is `Webhook`. +optional |
| conversionReviewVersions | [string](#string) | repeated | conversionReviewVersions is an ordered list of preferred `ConversionReview` versions the Webhook expects. The API server will use the first version in the list which it supports. If none of the versions specified in this list are supported by API server, conversion will fail for the custom resource. If a persisted Webhook configuration specifies allowed versions and does not include any versions known to the API Server, calls to the webhook will fail. +listType=atomic |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="k8s-io_apimachinery_pkg_util_intstr_generated-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## k8s.io/apimachinery/pkg/util/intstr/generated.proto



<a name="k8s-io-apimachinery-pkg-util-intstr-IntOrString"></a>

### IntOrString
IntOrString is a type that can hold an int32 or a string.  When used in
JSON or YAML marshalling and unmarshalling, it produces or consumes the
inner type.  This allows you to have, for example, a JSON field that can
accept a name or number.
TODO: Rename to Int32OrString

+protobuf=true
+protobuf.options.(gogoproto.goproto_stringer)=false
+k8s:openapi-gen=true


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [int64](#int64) | optional |  |
| intVal | [int32](#int32) | optional |  |
| strVal | [string](#string) | optional |  |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="k8s-io_apimachinery_pkg_api_resource_generated-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## k8s.io/apimachinery/pkg/api/resource/generated.proto



<a name="k8s-io-apimachinery-pkg-api-resource-Quantity"></a>

### Quantity
Quantity is a fixed-point representation of a number.
It provides convenient marshaling/unmarshaling in JSON and YAML,
in addition to String() and AsInt64() accessors.

The serialization format is:

```
<quantity>        ::= <signedNumber><suffix>

	(Note that <suffix> may be empty, from the "" case in <decimalSI>.)

<digit>           ::= 0 | 1 | ... | 9
<digits>          ::= <digit> | <digit><digits>
<number>          ::= <digits> | <digits>.<digits> | <digits>. | .<digits>
<sign>            ::= "+" | "-"
<signedNumber>    ::= <number> | <sign><number>
<suffix>          ::= <binarySI> | <decimalExponent> | <decimalSI>
<binarySI>        ::= Ki | Mi | Gi | Ti | Pi | Ei

	(International System of units; See: http://physics.nist.gov/cuu/Units/binary.html)

<decimalSI>       ::= m | "" | k | M | G | T | P | E

	(Note that 1024 = 1Ki but 1000 = 1k; I didn't choose the capitalization.)

<decimalExponent> ::= "e" <signedNumber> | "E" <signedNumber>
```

No matter which of the three exponent forms is used, no quantity may represent
a number greater than 2^63-1 in magnitude, nor may it have more than 3 decimal
places. Numbers larger or more precise will be capped or rounded up.
(E.g.: 0.1m will rounded up to 1m.)
This may be extended in the future if we require larger or smaller quantities.

When a Quantity is parsed from a string, it will remember the type of suffix
it had, and will use the same type again when it is serialized.

Before serializing, Quantity will be put in "canonical form".
This means that Exponent/suffix will be adjusted up or down (with a
corresponding increase or decrease in Mantissa) such that:

- No precision is lost
- No fractional digits will be emitted
- The exponent (or suffix) is as large as possible.

The sign will be omitted unless the number is negative.

Examples:

- 1.5 will be serialized as "1500m"
- 1.5Gi will be serialized as "1536Mi"

Note that the quantity will NEVER be internally represented by a
floating point number. That is the whole point of this exercise.

Non-canonical values will still parse as long as they are well formed,
but will be re-emitted in their canonical form. (So always use canonical
form, or don't diff.)

This format is intended to make it difficult to use these numbers without
writing some sort of special handling code in the hopes that that will
cause implementors to also use a fixed point implementation.

+protobuf=true
+protobuf.embed=string
+protobuf.options.marshal=false
+protobuf.options.(gogoproto.goproto_stringer)=false
+k8s:deepcopy-gen=true
+k8s:openapi-gen=true


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| string | [string](#string) | optional |  |






<a name="k8s-io-apimachinery-pkg-api-resource-QuantityValue"></a>

### QuantityValue
QuantityValue makes it possible to use a Quantity as value for a command
line parameter.

+protobuf=true
+protobuf.embed=string
+protobuf.options.marshal=false
+protobuf.options.(gogoproto.goproto_stringer)=false
+k8s:deepcopy-gen=true


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| string | [string](#string) | optional |  |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="k8s-io_apimachinery_pkg_runtime_schema_generated-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## k8s.io/apimachinery/pkg/runtime/schema/generated.proto


 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="k8s-io_apimachinery_pkg_runtime_generated-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## k8s.io/apimachinery/pkg/runtime/generated.proto



<a name="k8s-io-apimachinery-pkg-runtime-RawExtension"></a>

### RawExtension
RawExtension is used to hold extensions in external versions.

To use this, make a field which has RawExtension as its type in your external, versioned
struct, and Object in your internal struct. You also need to register your
various plugin types.

// Internal package:

	type MyAPIObject struct {
		runtime.TypeMeta `json:",inline"`
		MyPlugin runtime.Object `json:"myPlugin"`
	}

	type PluginA struct {
		AOption string `json:"aOption"`
	}

// External package:

	type MyAPIObject struct {
		runtime.TypeMeta `json:",inline"`
		MyPlugin runtime.RawExtension `json:"myPlugin"`
	}

	type PluginA struct {
		AOption string `json:"aOption"`
	}

// On the wire, the JSON will look something like this:

	{
		"kind":"MyAPIObject",
		"apiVersion":"v1",
		"myPlugin": {
			"kind":"PluginA",
			"aOption":"foo",
		},
	}

So what happens? Decode first uses json or yaml to unmarshal the serialized data into
your external MyAPIObject. That causes the raw JSON to be stored, but not unpacked.
The next step is to copy (using pkg/conversion) into the internal struct. The runtime
package's DefaultScheme has conversion functions installed which will unpack the
JSON stored in RawExtension, turning it into the correct object type, and storing it
in the Object. (TODO: In the case where the object is of an unknown type, a
runtime.Unknown object will be created and stored.)

+k8s:deepcopy-gen=true
+protobuf=true
+k8s:openapi-gen=true


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| raw | [bytes](#bytes) | optional | Raw is the underlying serialization of this object.

TODO: Determine how to detect ContentType and ContentEncoding of 'Raw' data. |






<a name="k8s-io-apimachinery-pkg-runtime-TypeMeta"></a>

### TypeMeta
TypeMeta is shared by all top level objects. The proper way to use it is to inline it in your type,
like this:

	type MyAwesomeAPIObject struct {
	     runtime.TypeMeta    `json:",inline"`
	     ... // other fields
	}

func (obj *MyAwesomeAPIObject) SetGroupVersionKind(gvk *metav1.GroupVersionKind) { metav1.UpdateTypeMeta(obj,gvk) }; GroupVersionKind() *GroupVersionKind

TypeMeta is provided here for convenience. You may use it directly from this package or define
your own with the same fields.

+k8s:deepcopy-gen=false
+protobuf=true
+k8s:openapi-gen=true


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| apiVersion | [string](#string) | optional | +optional |
| kind | [string](#string) | optional | +optional |






<a name="k8s-io-apimachinery-pkg-runtime-Unknown"></a>

### Unknown
Unknown allows api objects with unknown types to be passed-through. This can be used
to deal with the API objects from a plug-in. Unknown objects still have functioning
TypeMeta features-- kind, version, etc.
TODO: Make this object have easy access to field based accessors and settors for
metadata and field mutatation.

+k8s:deepcopy-gen=true
+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
+protobuf=true
+k8s:openapi-gen=true


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| typeMeta | [TypeMeta](#k8s-io-apimachinery-pkg-runtime-TypeMeta) | optional |  |
| raw | [bytes](#bytes) | optional | Raw will hold the complete serialized object which couldn't be matched with a registered type. Most likely, nothing should be done with this except for passing it through the system. |
| contentEncoding | [string](#string) | optional | ContentEncoding is encoding used to encode 'Raw' data. Unspecified means no encoding. |
| contentType | [string](#string) | optional | ContentType is serialization method used to serialize 'Raw'. Unspecified means ContentTypeJSON. |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="k8s-io_apimachinery_pkg_apis_meta_v1_generated-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## k8s.io/apimachinery/pkg/apis/meta/v1/generated.proto



<a name="k8s-io-apimachinery-pkg-apis-meta-v1-APIGroup"></a>

### APIGroup
APIGroup contains the name, the supported versions, and the preferred version
of a group.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | name is the name of the group. |
| versions | [GroupVersionForDiscovery](#k8s-io-apimachinery-pkg-apis-meta-v1-GroupVersionForDiscovery) | repeated | versions are the versions supported in this group. +listType=atomic |
| preferredVersion | [GroupVersionForDiscovery](#k8s-io-apimachinery-pkg-apis-meta-v1-GroupVersionForDiscovery) | optional | preferredVersion is the version preferred by the API server, which probably is the storage version. +optional |
| serverAddressByClientCIDRs | [ServerAddressByClientCIDR](#k8s-io-apimachinery-pkg-apis-meta-v1-ServerAddressByClientCIDR) | repeated | a map of client CIDR to server address that is serving this group. This is to help clients reach servers in the most network-efficient way possible. Clients can use the appropriate server address as per the CIDR that they match. In case of multiple matches, clients should use the longest matching CIDR. The server returns only those CIDRs that it thinks that the client can match. For example: the master will return an internal IP CIDR only, if the client reaches the server using an internal IP. Server looks at X-Forwarded-For header or X-Real-Ip header or request.RemoteAddr (in that order) to get the client IP. +optional +listType=atomic |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-APIGroupList"></a>

### APIGroupList
APIGroupList is a list of APIGroup, to allow clients to discover the API at
/apis.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| groups | [APIGroup](#k8s-io-apimachinery-pkg-apis-meta-v1-APIGroup) | repeated | groups is a list of APIGroup. +listType=atomic |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-APIResource"></a>

### APIResource
APIResource specifies the name of a resource and whether it is namespaced.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | name is the plural name of the resource. |
| singularName | [string](#string) | optional | singularName is the singular name of the resource. This allows clients to handle plural and singular opaquely. The singularName is more correct for reporting status on a single item and both singular and plural are allowed from the kubectl CLI interface. |
| namespaced | [bool](#bool) | optional | namespaced indicates if a resource is namespaced or not. |
| group | [string](#string) | optional | group is the preferred group of the resource. Empty implies the group of the containing resource list. For subresources, this may have a different value, for example: Scale". |
| version | [string](#string) | optional | version is the preferred version of the resource. Empty implies the version of the containing resource list For subresources, this may have a different value, for example: v1 (while inside a v1beta1 version of the core resource's group)". |
| kind | [string](#string) | optional | kind is the kind for the resource (e.g. 'Foo' is the kind for a resource 'foo') |
| verbs | [Verbs](#k8s-io-apimachinery-pkg-apis-meta-v1-Verbs) | optional | verbs is a list of supported kube verbs (this includes get, list, watch, create, update, patch, delete, deletecollection, and proxy) |
| shortNames | [string](#string) | repeated | shortNames is a list of suggested short names of the resource. +listType=atomic |
| categories | [string](#string) | repeated | categories is a list of the grouped resources this resource belongs to (e.g. 'all') +listType=atomic |
| storageVersionHash | [string](#string) | optional | The hash value of the storage version, the version this resource is converted to when written to the data store. Value must be treated as opaque by clients. Only equality comparison on the value is valid. This is an alpha feature and may change or be removed in the future. The field is populated by the apiserver only if the StorageVersionHash feature gate is enabled. This field will remain optional even if it graduates. +optional |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-APIResourceList"></a>

### APIResourceList
APIResourceList is a list of APIResource, it is used to expose the name of the
resources supported in a specific group and version, and if the resource
is namespaced.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| groupVersion | [string](#string) | optional | groupVersion is the group and version this APIResourceList is for. |
| resources | [APIResource](#k8s-io-apimachinery-pkg-apis-meta-v1-APIResource) | repeated | resources contains the name of the resources and if they are namespaced. +listType=atomic |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-APIVersions"></a>

### APIVersions
APIVersions lists the versions that are available, to allow clients to
discover the API at /api, which is the root path of the legacy v1 API.

+protobuf.options.(gogoproto.goproto_stringer)=false
+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| versions | [string](#string) | repeated | versions are the api versions that are available. +listType=atomic |
| serverAddressByClientCIDRs | [ServerAddressByClientCIDR](#k8s-io-apimachinery-pkg-apis-meta-v1-ServerAddressByClientCIDR) | repeated | a map of client CIDR to server address that is serving this group. This is to help clients reach servers in the most network-efficient way possible. Clients can use the appropriate server address as per the CIDR that they match. In case of multiple matches, clients should use the longest matching CIDR. The server returns only those CIDRs that it thinks that the client can match. For example: the master will return an internal IP CIDR only, if the client reaches the server using an internal IP. Server looks at X-Forwarded-For header or X-Real-Ip header or request.RemoteAddr (in that order) to get the client IP. +listType=atomic |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-ApplyOptions"></a>

### ApplyOptions
ApplyOptions may be provided when applying an API object.
FieldManager is required for apply requests.
ApplyOptions is equivalent to PatchOptions. It is provided as a convenience with documentation
that speaks specifically to how the options fields relate to apply.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| dryRun | [string](#string) | repeated | When present, indicates that modifications should not be persisted. An invalid or unrecognized dryRun directive will result in an error response and no further processing of the request. Valid values are: - All: all dry run stages will be processed +optional +listType=atomic |
| force | [bool](#bool) | optional | Force is going to "force" Apply requests. It means user will re-acquire conflicting fields owned by other people. |
| fieldManager | [string](#string) | optional | fieldManager is a name associated with the actor or entity that is making these changes. The value must be less than or 128 characters long, and only contain printable characters, as defined by https://golang.org/pkg/unicode/#IsPrint. This field is required. |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-Condition"></a>

### Condition
Condition contains details for one aspect of the current state of this API Resource.
---
This struct is intended for direct use as an array at the field path .status.conditions.  For example,

	type FooStatus struct{
	    // Represents the observations of a foo's current state.
	    // Known .status.conditions.type are: "Available", "Progressing", and "Degraded"
	    // +patchMergeKey=type
	    // +patchStrategy=merge
	    // +listType=map
	    // +listMapKey=type
	    Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	    // other fields
	}


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) | optional | type of condition in CamelCase or in foo.example.com/CamelCase. --- Many .condition.type values are consistent across resources like Available, but because arbitrary conditions can be useful (see .node.status.conditions), the ability to deconflict is important. The regex it matches is (dns1123SubdomainFmt/)?(qualifiedNameFmt) +required +kubebuilder:validation:Required +kubebuilder:validation:Pattern=`^([a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*/)?(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])$` +kubebuilder:validation:MaxLength=316 |
| status | [string](#string) | optional | status of the condition, one of True, False, Unknown. +required +kubebuilder:validation:Required +kubebuilder:validation:Enum=True;False;Unknown |
| observedGeneration | [int64](#int64) | optional | observedGeneration represents the .metadata.generation that the condition was set based upon. For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date with respect to the current state of the instance. +optional +kubebuilder:validation:Minimum=0 |
| lastTransitionTime | [Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | lastTransitionTime is the last time the condition transitioned from one status to another. This should be when the underlying condition changed. If that is not known, then using the time when the API field changed is acceptable. +required +kubebuilder:validation:Required +kubebuilder:validation:Type=string +kubebuilder:validation:Format=date-time |
| reason | [string](#string) | optional | reason contains a programmatic identifier indicating the reason for the condition's last transition. Producers of specific condition types may define expected values and meanings for this field, and whether the values are considered a guaranteed API. The value should be a CamelCase string. This field may not be empty. +required +kubebuilder:validation:Required +kubebuilder:validation:MaxLength=1024 +kubebuilder:validation:MinLength=1 +kubebuilder:validation:Pattern=`^[A-Za-z]([A-Za-z0-9_,:]*[A-Za-z0-9_])?$` |
| message | [string](#string) | optional | message is a human readable message indicating details about the transition. This may be an empty string. +required +kubebuilder:validation:Required +kubebuilder:validation:MaxLength=32768 |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-CreateOptions"></a>

### CreateOptions
CreateOptions may be provided when creating an API object.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| dryRun | [string](#string) | repeated | When present, indicates that modifications should not be persisted. An invalid or unrecognized dryRun directive will result in an error response and no further processing of the request. Valid values are: - All: all dry run stages will be processed +optional +listType=atomic |
| fieldManager | [string](#string) | optional | fieldManager is a name associated with the actor or entity that is making these changes. The value must be less than or 128 characters long, and only contain printable characters, as defined by https://golang.org/pkg/unicode/#IsPrint. +optional |
| fieldValidation | [string](#string) | optional | fieldValidation instructs the server on how to handle objects in the request (POST/PUT/PATCH) containing unknown or duplicate fields. Valid values are: - Ignore: This will ignore any unknown fields that are silently dropped from the object, and will ignore all but the last duplicate field that the decoder encounters. This is the default behavior prior to v1.23. - Warn: This will send a warning via the standard warning response header for each unknown field that is dropped from the object, and for each duplicate field that is encountered. The request will still succeed if there are no other errors, and will only persist the last of any duplicate fields. This is the default in v1.23+ - Strict: This will fail the request with a BadRequest error if any unknown fields would be dropped from the object, or if any duplicate fields are present. The error returned from the server will contain all unknown and duplicate fields encountered. +optional |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-DeleteOptions"></a>

### DeleteOptions
DeleteOptions may be provided when deleting an API object.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| gracePeriodSeconds | [int64](#int64) | optional | The duration in seconds before the object should be deleted. Value must be non-negative integer. The value zero indicates delete immediately. If this value is nil, the default grace period for the specified type will be used. Defaults to a per object value if not specified. zero means delete immediately. +optional |
| preconditions | [Preconditions](#k8s-io-apimachinery-pkg-apis-meta-v1-Preconditions) | optional | Must be fulfilled before a deletion is carried out. If not possible, a 409 Conflict status will be returned. +k8s:conversion-gen=false +optional |
| orphanDependents | [bool](#bool) | optional | Deprecated: please use the PropagationPolicy, this field will be deprecated in 1.7. Should the dependent objects be orphaned. If true/false, the "orphan" finalizer will be added to/removed from the object's finalizers list. Either this field or PropagationPolicy may be set, but not both. +optional |
| propagationPolicy | [string](#string) | optional | Whether and how garbage collection will be performed. Either this field or OrphanDependents may be set, but not both. The default policy is decided by the existing finalizer set in the metadata.finalizers and the resource-specific default policy. Acceptable values are: 'Orphan' - orphan the dependents; 'Background' - allow the garbage collector to delete the dependents in the background; 'Foreground' - a cascading policy that deletes all dependents in the foreground. +optional |
| dryRun | [string](#string) | repeated | When present, indicates that modifications should not be persisted. An invalid or unrecognized dryRun directive will result in an error response and no further processing of the request. Valid values are: - All: all dry run stages will be processed +optional +listType=atomic |
| ignoreStoreReadErrorWithClusterBreakingPotential | [bool](#bool) | optional | if set to true, it will trigger an unsafe deletion of the resource in case the normal deletion flow fails with a corrupt object error. A resource is considered corrupt if it can not be retrieved from the underlying storage successfully because of a) its data can not be transformed e.g. decryption failure, or b) it fails to decode into an object. NOTE: unsafe deletion ignores finalizer constraints, skips precondition checks, and removes the object from the storage. WARNING: This may potentially break the cluster if the workload associated with the resource being unsafe-deleted relies on normal deletion flow. Use only if you REALLY know what you are doing. The default value is false, and the user must opt in to enable it +optional |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-Duration"></a>

### Duration
Duration is a wrapper around time.Duration which supports correct
marshaling to YAML and JSON. In particular, it marshals into strings, which
can be used as map keys in json.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| duration | [int64](#int64) | optional |  |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-FieldSelectorRequirement"></a>

### FieldSelectorRequirement
FieldSelectorRequirement is a selector that contains values, a key, and an operator that
relates the key and values.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional | key is the field selector key that the requirement applies to. |
| operator | [string](#string) | optional | operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. The list of operators may grow in the future. |
| values | [string](#string) | repeated | values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. +optional +listType=atomic |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-FieldsV1"></a>

### FieldsV1
FieldsV1 stores a set of fields in a data structure like a Trie, in JSON format.

Each key is either a '.' representing the field itself, and will always map to an empty set,
or a string representing a sub-field or item. The string will follow one of these four formats:
'f:<name>', where <name> is the name of a field in a struct, or key in a map
'v:<value>', where <value> is the exact json formatted value of a list item
'i:<index>', where <index> is position of a item in a list
'k:<keys>', where <keys> is a map of  a list item's key fields to their unique values
If a key maps to an empty Fields value, the field that key represents is part of the set.

The exact format is defined in sigs.k8s.io/structured-merge-diff
+protobuf.options.(gogoproto.goproto_stringer)=false


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| Raw | [bytes](#bytes) | optional | Raw is the underlying serialization of this object. |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-GetOptions"></a>

### GetOptions
GetOptions is the standard query options to the standard REST get call.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resourceVersion | [string](#string) | optional | resourceVersion sets a constraint on what resource versions a request may be served from. See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.

Defaults to unset +optional |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-GroupKind"></a>

### GroupKind
GroupKind specifies a Group and a Kind, but does not force a version.  This is useful for identifying
concepts during lookup stages without having partially valid types

+protobuf.options.(gogoproto.goproto_stringer)=false


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| group | [string](#string) | optional |  |
| kind | [string](#string) | optional |  |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-GroupResource"></a>

### GroupResource
GroupResource specifies a Group and a Resource, but does not force a version.  This is useful for identifying
concepts during lookup stages without having partially valid types

+protobuf.options.(gogoproto.goproto_stringer)=false


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| group | [string](#string) | optional |  |
| resource | [string](#string) | optional |  |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-GroupVersion"></a>

### GroupVersion
GroupVersion contains the "group" and the "version", which uniquely identifies the API.

+protobuf.options.(gogoproto.goproto_stringer)=false


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| group | [string](#string) | optional |  |
| version | [string](#string) | optional |  |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-GroupVersionForDiscovery"></a>

### GroupVersionForDiscovery
GroupVersion contains the "group/version" and "version" string of a version.
It is made a struct to keep extensibility.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| groupVersion | [string](#string) | optional | groupVersion specifies the API group and version in the form "group/version" |
| version | [string](#string) | optional | version specifies the version in the form of "version". This is to save the clients the trouble of splitting the GroupVersion. |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-GroupVersionKind"></a>

### GroupVersionKind
GroupVersionKind unambiguously identifies a kind.  It doesn't anonymously include GroupVersion
to avoid automatic coercion.  It doesn't use a GroupVersion to avoid custom marshalling

+protobuf.options.(gogoproto.goproto_stringer)=false


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| group | [string](#string) | optional |  |
| version | [string](#string) | optional |  |
| kind | [string](#string) | optional |  |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-GroupVersionResource"></a>

### GroupVersionResource
GroupVersionResource unambiguously identifies a resource.  It doesn't anonymously include GroupVersion
to avoid automatic coercion.  It doesn't use a GroupVersion to avoid custom marshalling

+protobuf.options.(gogoproto.goproto_stringer)=false


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| group | [string](#string) | optional |  |
| version | [string](#string) | optional |  |
| resource | [string](#string) | optional |  |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-LabelSelector"></a>

### LabelSelector
A label selector is a label query over a set of resources. The result of matchLabels and
matchExpressions are ANDed. An empty label selector matches all objects. A null
label selector matches no objects.
+structType=atomic


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| matchLabels | [LabelSelector.MatchLabelsEntry](#k8s-io-apimachinery-pkg-apis-meta-v1-LabelSelector-MatchLabelsEntry) | repeated | matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed. +optional |
| matchExpressions | [LabelSelectorRequirement](#k8s-io-apimachinery-pkg-apis-meta-v1-LabelSelectorRequirement) | repeated | matchExpressions is a list of label selector requirements. The requirements are ANDed. +optional +listType=atomic |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-LabelSelector-MatchLabelsEntry"></a>

### LabelSelector.MatchLabelsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [string](#string) | optional |  |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-LabelSelectorRequirement"></a>

### LabelSelectorRequirement
A label selector requirement is a selector that contains values, a key, and an operator that
relates the key and values.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional | key is the label key that the selector applies to. |
| operator | [string](#string) | optional | operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist. |
| values | [string](#string) | repeated | values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch. +optional +listType=atomic |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-List"></a>

### List
List holds a list of objects, which may not be known by the server.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional | Standard list metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds +optional |
| items | [k8s.io.apimachinery.pkg.runtime.RawExtension](#k8s-io-apimachinery-pkg-runtime-RawExtension) | repeated | List of objects |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta"></a>

### ListMeta
ListMeta describes metadata that synthetic resources must have, including lists and
various status objects. A resource may have only one of {ObjectMeta, ListMeta}.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| selfLink | [string](#string) | optional | Deprecated: selfLink is a legacy read-only field that is no longer populated by the system. +optional |
| resourceVersion | [string](#string) | optional | String that identifies the server's internal version of this object that can be used by clients to determine when objects have changed. Value must be treated as opaque by clients and passed unmodified back to the server. Populated by the system. Read-only. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency +optional |
| continue | [string](#string) | optional | continue may be set if the user set a limit on the number of items returned, and indicates that the server has more data available. The value is opaque and may be used to issue another request to the endpoint that served this list to retrieve the next set of available objects. Continuing a consistent list may not be possible if the server configuration has changed or more than a few minutes have passed. The resourceVersion field returned when using this continue value will be identical to the value in the first response, unless you have received this token from an error message. |
| remainingItemCount | [int64](#int64) | optional | remainingItemCount is the number of subsequent items in the list which are not included in this list response. If the list request contained label or field selectors, then the number of remaining items is unknown and the field will be left unset and omitted during serialization. If the list is complete (either because it is not chunking or because this is the last chunk), then there are no more remaining items and this field will be left unset and omitted during serialization. Servers older than v1.15 do not set this field. The intended use of the remainingItemCount is *estimating* the size of a collection. Clients should not rely on the remainingItemCount to be set or to be exact. +optional |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-ListOptions"></a>

### ListOptions
ListOptions is the query options to a standard REST list call.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| labelSelector | [string](#string) | optional | A selector to restrict the list of returned objects by their labels. Defaults to everything. +optional |
| fieldSelector | [string](#string) | optional | A selector to restrict the list of returned objects by their fields. Defaults to everything. +optional |
| watch | [bool](#bool) | optional | Watch for changes to the described resources and return them as a stream of add, update, and remove notifications. Specify resourceVersion. +optional |
| allowWatchBookmarks | [bool](#bool) | optional | allowWatchBookmarks requests watch events with type "BOOKMARK". Servers that do not implement bookmarks may ignore this flag and bookmarks are sent at the server's discretion. Clients should not assume bookmarks are returned at any specific interval, nor may they assume the server will send any BOOKMARK event during a session. If this is not a watch, this field is ignored. +optional |
| resourceVersion | [string](#string) | optional | resourceVersion sets a constraint on what resource versions a request may be served from. See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.

Defaults to unset +optional |
| resourceVersionMatch | [string](#string) | optional | resourceVersionMatch determines how resourceVersion is applied to list calls. It is highly recommended that resourceVersionMatch be set for list calls where resourceVersion is set See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.

Defaults to unset +optional |
| timeoutSeconds | [int64](#int64) | optional | Timeout for the list/watch call. This limits the duration of the call, regardless of any activity or inactivity. +optional |
| limit | [int64](#int64) | optional | limit is a maximum number of responses to return for a list call. If more items exist, the server will set the `continue` field on the list metadata to a value that can be used with the same initial query to retrieve the next set of results. Setting a limit may return fewer than the requested amount of items (up to zero items) in the event all requested objects are filtered out and clients should only use the presence of the continue field to determine whether more results are available. Servers may choose not to support the limit argument and will return all of the available results. If limit is specified and the continue field is empty, clients may assume that no more results are available. This field is not supported if watch is true.

The server guarantees that the objects returned when using continue will be identical to issuing a single list call without a limit - that is, no objects created, modified, or deleted after the first request is issued will be included in any subsequent continued requests. This is sometimes referred to as a consistent snapshot, and ensures that a client that is using limit to receive smaller chunks of a very large result can ensure they see all possible objects. If objects are updated during a chunked list the version of the object that was present at the time the first list result was calculated is returned. |
| continue | [string](#string) | optional | The continue option should be set when retrieving more results from the server. Since this value is server defined, clients may only use the continue value from a previous query result with identical query parameters (except for the value of continue) and the server may reject a continue value it does not recognize. If the specified continue value is no longer valid whether due to expiration (generally five to fifteen minutes) or a configuration change on the server, the server will respond with a 410 ResourceExpired error together with a continue token. If the client needs a consistent list, it must restart their list without the continue field. Otherwise, the client may send another list request with the token received with the 410 error, the server will respond with a list starting from the next key, but from the latest snapshot, which is inconsistent from the previous list results - objects that are created, modified, or deleted after the first list request will be included in the response, as long as their keys are after the "next key".

This field is not supported when watch is true. Clients may start a watch from the last resourceVersion value returned by the server and not miss any modifications. |
| sendInitialEvents | [bool](#bool) | optional | `sendInitialEvents=true` may be set together with `watch=true`. In that case, the watch stream will begin with synthetic events to produce the current state of objects in the collection. Once all such events have been sent, a synthetic "Bookmark" event will be sent. The bookmark will report the ResourceVersion (RV) corresponding to the set of objects, and be marked with `"k8s.io/initial-events-end": "true"` annotation. Afterwards, the watch stream will proceed as usual, sending watch events corresponding to changes (subsequent to the RV) to objects watched.

When `sendInitialEvents` option is set, we require `resourceVersionMatch` option to also be set. The semantic of the watch request is as following: - `resourceVersionMatch` = NotOlderThan is interpreted as "data at least as new as the provided `resourceVersion`" and the bookmark event is send when the state is synced to a `resourceVersion` at least as fresh as the one provided by the ListOptions. If `resourceVersion` is unset, this is interpreted as "consistent read" and the bookmark event is send when the state is synced at least to the moment when request started being processed. - `resourceVersionMatch` set to any other value or unset Invalid error is returned.

Defaults to true if `resourceVersion=""` or `resourceVersion="0"` (for backward compatibility reasons) and to false otherwise. +optional |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-ManagedFieldsEntry"></a>

### ManagedFieldsEntry
ManagedFieldsEntry is a workflow-id, a FieldSet and the group version of the resource
that the fieldset applies to.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| manager | [string](#string) | optional | Manager is an identifier of the workflow managing these fields. |
| operation | [string](#string) | optional | Operation is the type of operation which lead to this ManagedFieldsEntry being created. The only valid values for this field are 'Apply' and 'Update'. |
| apiVersion | [string](#string) | optional | APIVersion defines the version of this resource that this field set applies to. The format is "group/version" just like the top-level APIVersion field. It is necessary to track the version of a field set because it cannot be automatically converted. |
| time | [Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | Time is the timestamp of when the ManagedFields entry was added. The timestamp will also be updated if a field is added, the manager changes any of the owned fields value or removes a field. The timestamp does not update when a field is removed from the entry because another manager took it over. +optional |
| fieldsType | [string](#string) | optional | FieldsType is the discriminator for the different fields format and version. There is currently only one possible value: "FieldsV1" |
| fieldsV1 | [FieldsV1](#k8s-io-apimachinery-pkg-apis-meta-v1-FieldsV1) | optional | FieldsV1 holds the first JSON version format as described in the "FieldsV1" type. +optional |
| subresource | [string](#string) | optional | Subresource is the name of the subresource used to update that object, or empty string if the object was updated through the main resource. The value of this field is used to distinguish between managers, even if they share the same name. For example, a status update will be distinct from a regular update using the same manager name. Note that the APIVersion field is not related to the Subresource field and it always corresponds to the version of the main resource. |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-MicroTime"></a>

### MicroTime
MicroTime is version of Time with microsecond level precision.

+protobuf.options.marshal=false
+protobuf.as=Timestamp
+protobuf.options.(gogoproto.goproto_stringer)=false


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| seconds | [int64](#int64) | optional | Represents seconds of UTC time since Unix epoch 1970-01-01T00:00:00Z. Must be from 0001-01-01T00:00:00Z to 9999-12-31T23:59:59Z inclusive. |
| nanos | [int32](#int32) | optional | Non-negative fractions of a second at nanosecond resolution. Negative second values with fractions must still have non-negative nanos values that count forward in time. Must be from 0 to 999,999,999 inclusive. This field may be limited in precision depending on context. |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta"></a>

### ObjectMeta
ObjectMeta is metadata that all persisted resources must have, which includes all objects
users must create.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | Name must be unique within a namespace. Is required when creating resources, although some resources may allow a client to request the generation of an appropriate name automatically. Name is primarily intended for creation idempotence and configuration definition. Cannot be updated. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#names +optional |
| generateName | [string](#string) | optional | GenerateName is an optional prefix, used by the server, to generate a unique name ONLY IF the Name field has not been provided. If this field is used, the name returned to the client will be different than the name passed. This value will also be combined with a unique suffix. The provided value has the same validation rules as the Name field, and may be truncated by the length of the suffix required to make the value unique on the server.

If this field is specified and the generated name exists, the server will return a 409.

Applied only if Name is not specified. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#idempotency +optional |
| namespace | [string](#string) | optional | Namespace defines the space within which each name must be unique. An empty namespace is equivalent to the "default" namespace, but "default" is the canonical representation. Not all objects are required to be scoped to a namespace - the value of this field for those objects will be empty.

Must be a DNS_LABEL. Cannot be updated. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces +optional |
| selfLink | [string](#string) | optional | Deprecated: selfLink is a legacy read-only field that is no longer populated by the system. +optional |
| uid | [string](#string) | optional | UID is the unique in time and space value for this object. It is typically generated by the server on successful creation of a resource and is not allowed to change on PUT operations.

Populated by the system. Read-only. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#uids +optional |
| resourceVersion | [string](#string) | optional | An opaque value that represents the internal version of this object that can be used by clients to determine when objects have changed. May be used for optimistic concurrency, change detection, and the watch operation on a resource or set of resources. Clients must treat these values as opaque and passed unmodified back to the server. They may only be valid for a particular resource or set of resources.

Populated by the system. Read-only. Value must be treated as opaque by clients and . More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency +optional |
| generation | [int64](#int64) | optional | A sequence number representing a specific generation of the desired state. Populated by the system. Read-only. +optional |
| creationTimestamp | [Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | CreationTimestamp is a timestamp representing the server time when this object was created. It is not guaranteed to be set in happens-before order across separate operations. Clients may not set this value. It is represented in RFC3339 form and is in UTC.

Populated by the system. Read-only. Null for lists. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional |
| deletionTimestamp | [Time](#k8s-io-apimachinery-pkg-apis-meta-v1-Time) | optional | DeletionTimestamp is RFC 3339 date and time at which this resource will be deleted. This field is set by the server when a graceful deletion is requested by the user, and is not directly settable by a client. The resource is expected to be deleted (no longer visible from resource lists, and not reachable by name) after the time in this field, once the finalizers list is empty. As long as the finalizers list contains items, deletion is blocked. Once the deletionTimestamp is set, this value may not be unset or be set further into the future, although it may be shortened or the resource may be deleted prior to this time. For example, a user may request that a pod is deleted in 30 seconds. The Kubelet will react by sending a graceful termination signal to the containers in the pod. After that 30 seconds, the Kubelet will send a hard termination signal (SIGKILL) to the container and after cleanup, remove the pod from the API. In the presence of network partitions, this object may still exist after this timestamp, until an administrator or automated process can determine the resource is fully terminated. If not set, graceful deletion of the object has not been requested.

Populated by the system when a graceful deletion is requested. Read-only. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional |
| deletionGracePeriodSeconds | [int64](#int64) | optional | Number of seconds allowed for this object to gracefully terminate before it will be removed from the system. Only set when deletionTimestamp is also set. May only be shortened. Read-only. +optional |
| labels | [ObjectMeta.LabelsEntry](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta-LabelsEntry) | repeated | Map of string keys and values that can be used to organize and categorize (scope and select) objects. May match selectors of replication controllers and services. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels +optional |
| annotations | [ObjectMeta.AnnotationsEntry](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta-AnnotationsEntry) | repeated | Annotations is an unstructured key value map stored with a resource that may be set by external tools to store and retrieve arbitrary metadata. They are not queryable and should be preserved when modifying objects. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations +optional |
| ownerReferences | [OwnerReference](#k8s-io-apimachinery-pkg-apis-meta-v1-OwnerReference) | repeated | List of objects depended by this object. If ALL objects in the list have been deleted, this object will be garbage collected. If this object is managed by a controller, then an entry in this list will point to this controller, with the controller field set to true. There cannot be more than one managing controller. +optional +patchMergeKey=uid +patchStrategy=merge +listType=map +listMapKey=uid |
| finalizers | [string](#string) | repeated | Must be empty before the object is deleted from the registry. Each entry is an identifier for the responsible component that will remove the entry from the list. If the deletionTimestamp of the object is non-nil, entries in this list can only be removed. Finalizers may be processed and removed in any order. Order is NOT enforced because it introduces significant risk of stuck finalizers. finalizers is a shared field, any actor with permission can reorder it. If the finalizer list is processed in order, then this can lead to a situation in which the component responsible for the first finalizer in the list is waiting for a signal (field value, external system, or other) produced by a component responsible for a finalizer later in the list, resulting in a deadlock. Without enforced ordering finalizers are free to order amongst themselves and are not vulnerable to ordering changes in the list. +optional +patchStrategy=merge +listType=set |
| managedFields | [ManagedFieldsEntry](#k8s-io-apimachinery-pkg-apis-meta-v1-ManagedFieldsEntry) | repeated | ManagedFields maps workflow-id and version to the set of fields that are managed by that workflow. This is mostly for internal housekeeping, and users typically shouldn't need to set or understand this field. A workflow can be the user's name, a controller's name, or the name of a specific apply path like "ci-cd". The set of fields is always in the version that the workflow used when modifying the object.

+optional +listType=atomic |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta-AnnotationsEntry"></a>

### ObjectMeta.AnnotationsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [string](#string) | optional |  |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta-LabelsEntry"></a>

### ObjectMeta.LabelsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) | optional |  |
| value | [string](#string) | optional |  |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-OwnerReference"></a>

### OwnerReference
OwnerReference contains enough information to let you identify an owning
object. An owning object must be in the same namespace as the dependent, or
be cluster-scoped, so there is no namespace field.
+structType=atomic


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| apiVersion | [string](#string) | optional | API version of the referent. |
| kind | [string](#string) | optional | Kind of the referent. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |
| name | [string](#string) | optional | Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#names |
| uid | [string](#string) | optional | UID of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#uids |
| controller | [bool](#bool) | optional | If true, this reference points to the managing controller. +optional |
| blockOwnerDeletion | [bool](#bool) | optional | If true, AND if the owner has the "foregroundDeletion" finalizer, then the owner cannot be deleted from the key-value store until this reference is removed. See https://kubernetes.io/docs/concepts/architecture/garbage-collection/#foreground-deletion for how the garbage collector interacts with this field and enforces the foreground deletion. Defaults to false. To set this field, a user needs "delete" permission of the owner, otherwise 422 (Unprocessable Entity) will be returned. +optional |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-PartialObjectMetadata"></a>

### PartialObjectMetadata
PartialObjectMetadata is a generic representation of any object with ObjectMeta. It allows clients
to get access to a particular ObjectMeta schema without knowing the details of the version.
+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [ObjectMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ObjectMeta) | optional | Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-PartialObjectMetadataList"></a>

### PartialObjectMetadataList
PartialObjectMetadataList contains a list of objects containing only their metadata
+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional | Standard list metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds +optional |
| items | [PartialObjectMetadata](#k8s-io-apimachinery-pkg-apis-meta-v1-PartialObjectMetadata) | repeated | items contains each of the included items. |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-Patch"></a>

### Patch
Patch is provided to give a concrete name and type to the Kubernetes PATCH request body.






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-PatchOptions"></a>

### PatchOptions
PatchOptions may be provided when patching an API object.
PatchOptions is meant to be a superset of UpdateOptions.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| dryRun | [string](#string) | repeated | When present, indicates that modifications should not be persisted. An invalid or unrecognized dryRun directive will result in an error response and no further processing of the request. Valid values are: - All: all dry run stages will be processed +optional +listType=atomic |
| force | [bool](#bool) | optional | Force is going to "force" Apply requests. It means user will re-acquire conflicting fields owned by other people. Force flag must be unset for non-apply patch requests. +optional |
| fieldManager | [string](#string) | optional | fieldManager is a name associated with the actor or entity that is making these changes. The value must be less than or 128 characters long, and only contain printable characters, as defined by https://golang.org/pkg/unicode/#IsPrint. This field is required for apply requests (application/apply-patch) but optional for non-apply patch types (JsonPatch, MergePatch, StrategicMergePatch). +optional |
| fieldValidation | [string](#string) | optional | fieldValidation instructs the server on how to handle objects in the request (POST/PUT/PATCH) containing unknown or duplicate fields. Valid values are: - Ignore: This will ignore any unknown fields that are silently dropped from the object, and will ignore all but the last duplicate field that the decoder encounters. This is the default behavior prior to v1.23. - Warn: This will send a warning via the standard warning response header for each unknown field that is dropped from the object, and for each duplicate field that is encountered. The request will still succeed if there are no other errors, and will only persist the last of any duplicate fields. This is the default in v1.23+ - Strict: This will fail the request with a BadRequest error if any unknown fields would be dropped from the object, or if any duplicate fields are present. The error returned from the server will contain all unknown and duplicate fields encountered. +optional |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-Preconditions"></a>

### Preconditions
Preconditions must be fulfilled before an operation (update, delete, etc.) is carried out.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| uid | [string](#string) | optional | Specifies the target UID. +optional |
| resourceVersion | [string](#string) | optional | Specifies the target ResourceVersion +optional |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-RootPaths"></a>

### RootPaths
RootPaths lists the paths available at root.
For example: "/healthz", "/apis".


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| paths | [string](#string) | repeated | paths are the paths available at root. +listType=atomic |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-ServerAddressByClientCIDR"></a>

### ServerAddressByClientCIDR
ServerAddressByClientCIDR helps the client to determine the server address that they should use, depending on the clientCIDR that they match.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| clientCIDR | [string](#string) | optional | The CIDR with which clients can match their IP to figure out the server address that they should use. |
| serverAddress | [string](#string) | optional | Address of this server, suitable for a client that matches the above CIDR. This can be a hostname, hostname:port, IP or IP:port. |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-Status"></a>

### Status
Status is a return value for calls that don't return other objects.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [ListMeta](#k8s-io-apimachinery-pkg-apis-meta-v1-ListMeta) | optional | Standard list metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds +optional |
| status | [string](#string) | optional | Status of the operation. One of: "Success" or "Failure". More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status +optional |
| message | [string](#string) | optional | A human-readable description of the status of this operation. +optional |
| reason | [string](#string) | optional | A machine-readable description of why this operation is in the "Failure" status. If this value is empty there is no information available. A Reason clarifies an HTTP status code but does not override it. +optional |
| details | [StatusDetails](#k8s-io-apimachinery-pkg-apis-meta-v1-StatusDetails) | optional | Extended data associated with the reason. Each reason may define its own extended details. This field is optional and the data returned is not guaranteed to conform to any schema except that defined by the reason type. +optional +listType=atomic |
| code | [int32](#int32) | optional | Suggested HTTP return code for this status, 0 if not set. +optional |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-StatusCause"></a>

### StatusCause
StatusCause provides more information about an api.Status failure, including
cases when multiple errors are encountered.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| reason | [string](#string) | optional | A machine-readable description of the cause of the error. If this value is empty there is no information available. +optional |
| message | [string](#string) | optional | A human-readable description of the cause of the error. This field may be presented as-is to a reader. +optional |
| field | [string](#string) | optional | The field of the resource that has caused this error, as named by its JSON serialization. May include dot and postfix notation for nested attributes. Arrays are zero-indexed. Fields may appear more than once in an array of causes due to fields having multiple errors. Optional.

Examples: "name" - the field "name" on the current resource "items[0].name" - the field "name" on the first array entry in "items" +optional |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-StatusDetails"></a>

### StatusDetails
StatusDetails is a set of additional properties that MAY be set by the
server to provide additional information about a response. The Reason
field of a Status object defines what attributes will be set. Clients
must ignore fields that do not match the defined type of each attribute,
and should assume that any attribute may be empty, invalid, or under
defined.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) | optional | The name attribute of the resource associated with the status StatusReason (when there is a single name which can be described). +optional |
| group | [string](#string) | optional | The group attribute of the resource associated with the status StatusReason. +optional |
| kind | [string](#string) | optional | The kind attribute of the resource associated with the status StatusReason. On some operations may differ from the requested resource Kind. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds +optional |
| uid | [string](#string) | optional | UID of the resource. (when there is a single resource which can be described). More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#uids +optional |
| causes | [StatusCause](#k8s-io-apimachinery-pkg-apis-meta-v1-StatusCause) | repeated | The Causes array includes more details associated with the StatusReason failure. Not all StatusReasons may provide detailed causes. +optional +listType=atomic |
| retryAfterSeconds | [int32](#int32) | optional | If specified, the time in seconds before the operation should be retried. Some errors may indicate the client must take an alternate action - for those errors this field may indicate how long to wait before taking the alternate action. +optional |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-TableOptions"></a>

### TableOptions
TableOptions are used when a Table is requested by the caller.
+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| includeObject | [string](#string) | optional | includeObject decides whether to include each object along with its columnar information. Specifying "None" will return no object, specifying "Object" will return the full object contents, and specifying "Metadata" (the default) will return the object's metadata in the PartialObjectMetadata kind in version v1beta1 of the meta.k8s.io API group. |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-Time"></a>

### Time
Time is a wrapper around time.Time which supports correct
marshaling to YAML and JSON.  Wrappers are provided for many
of the factory methods that the time package offers.

+protobuf.options.marshal=false
+protobuf.as=Timestamp
+protobuf.options.(gogoproto.goproto_stringer)=false


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| seconds | [int64](#int64) | optional | Represents seconds of UTC time since Unix epoch 1970-01-01T00:00:00Z. Must be from 0001-01-01T00:00:00Z to 9999-12-31T23:59:59Z inclusive. |
| nanos | [int32](#int32) | optional | Non-negative fractions of a second at nanosecond resolution. Negative second values with fractions must still have non-negative nanos values that count forward in time. Must be from 0 to 999,999,999 inclusive. This field may be limited in precision depending on context. |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-Timestamp"></a>

### Timestamp
Timestamp is a struct that is equivalent to Time, but intended for
protobuf marshalling/unmarshalling. It is generated into a serialization
that matches Time. Do not use in Go structs.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| seconds | [int64](#int64) | optional | Represents seconds of UTC time since Unix epoch 1970-01-01T00:00:00Z. Must be from 0001-01-01T00:00:00Z to 9999-12-31T23:59:59Z inclusive. |
| nanos | [int32](#int32) | optional | Non-negative fractions of a second at nanosecond resolution. Negative second values with fractions must still have non-negative nanos values that count forward in time. Must be from 0 to 999,999,999 inclusive. This field may be limited in precision depending on context. |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-TypeMeta"></a>

### TypeMeta
TypeMeta describes an individual object in an API response or request
with strings representing the type of the object and its API schema version.
Structures that are versioned or persisted should inline TypeMeta.

+k8s:deepcopy-gen=false


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| kind | [string](#string) | optional | Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds +optional |
| apiVersion | [string](#string) | optional | APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources +optional |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-UpdateOptions"></a>

### UpdateOptions
UpdateOptions may be provided when updating an API object.
All fields in UpdateOptions should also be present in PatchOptions.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| dryRun | [string](#string) | repeated | When present, indicates that modifications should not be persisted. An invalid or unrecognized dryRun directive will result in an error response and no further processing of the request. Valid values are: - All: all dry run stages will be processed +optional +listType=atomic |
| fieldManager | [string](#string) | optional | fieldManager is a name associated with the actor or entity that is making these changes. The value must be less than or 128 characters long, and only contain printable characters, as defined by https://golang.org/pkg/unicode/#IsPrint. +optional |
| fieldValidation | [string](#string) | optional | fieldValidation instructs the server on how to handle objects in the request (POST/PUT/PATCH) containing unknown or duplicate fields. Valid values are: - Ignore: This will ignore any unknown fields that are silently dropped from the object, and will ignore all but the last duplicate field that the decoder encounters. This is the default behavior prior to v1.23. - Warn: This will send a warning via the standard warning response header for each unknown field that is dropped from the object, and for each duplicate field that is encountered. The request will still succeed if there are no other errors, and will only persist the last of any duplicate fields. This is the default in v1.23+ - Strict: This will fail the request with a BadRequest error if any unknown fields would be dropped from the object, or if any duplicate fields are present. The error returned from the server will contain all unknown and duplicate fields encountered. +optional |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-Verbs"></a>

### Verbs
Verbs masks the value so protobuf can generate

+protobuf.nullable=true
+protobuf.options.(gogoproto.goproto_stringer)=false

items, if empty, will result in an empty slice


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| items | [string](#string) | repeated |  |






<a name="k8s-io-apimachinery-pkg-apis-meta-v1-WatchEvent"></a>

### WatchEvent
Event represents a single event to a watched resource.

+protobuf=true
+k8s:deepcopy-gen=true
+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) | optional |  |
| object | [k8s.io.apimachinery.pkg.runtime.RawExtension](#k8s-io-apimachinery-pkg-runtime-RawExtension) | optional | Object is: * If Type is Added or Modified: the new state of the object. * If Type is Deleted: the state of the object immediately before deletion. * If Type is Error: *Status is recommended; other types may make sense depending on context. |





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

