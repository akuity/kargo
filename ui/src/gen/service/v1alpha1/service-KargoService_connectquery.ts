// @generated by protoc-gen-connect-query v1.3.1 with parameter "target=ts"
// @generated from file service/v1alpha1/service.proto (package akuity.io.kargo.service.v1alpha1, syntax proto3)
/* eslint-disable */
// @ts-nocheck

import { MethodKind } from "@bufbuild/protobuf";
import { AbortVerificationRequest, AbortVerificationResponse, AdminLoginRequest, AdminLoginResponse, ApproveFreightRequest, ApproveFreightResponse, CreateCredentialsRequest, CreateCredentialsResponse, CreateOrUpdateResourceRequest, CreateOrUpdateResourceResponse, CreateResourceRequest, CreateResourceResponse, CreateRoleRequest, CreateRoleResponse, DeleteAnalysisTemplateRequest, DeleteAnalysisTemplateResponse, DeleteCredentialsRequest, DeleteCredentialsResponse, DeleteFreightRequest, DeleteFreightResponse, DeleteProjectRequest, DeleteProjectResponse, DeleteResourceRequest, DeleteResourceResponse, DeleteRoleRequest, DeleteRoleResponse, DeleteStageRequest, DeleteStageResponse, DeleteWarehouseRequest, DeleteWarehouseResponse, GetAnalysisRunRequest, GetAnalysisRunResponse, GetAnalysisTemplateRequest, GetAnalysisTemplateResponse, GetConfigRequest, GetConfigResponse, GetCredentialsRequest, GetCredentialsResponse, GetFreightRequest, GetFreightResponse, GetProjectRequest, GetProjectResponse, GetPromotionRequest, GetPromotionResponse, GetPublicConfigRequest, GetPublicConfigResponse, GetRoleRequest, GetRoleResponse, GetStageRequest, GetStageResponse, GetVersionInfoRequest, GetVersionInfoResponse, GetWarehouseRequest, GetWarehouseResponse, GrantRequest, GrantResponse, ListAnalysisTemplatesRequest, ListAnalysisTemplatesResponse, ListCredentialsRequest, ListCredentialsResponse, ListDetailedProjectsRequest, ListDetailedProjectsResponse, ListProjectEventsRequest, ListProjectEventsResponse, ListProjectsRequest, ListProjectsResponse, ListPromotionsRequest, ListPromotionsResponse, ListRolesRequest, ListRolesResponse, ListStagesRequest, ListStagesResponse, ListWarehousesRequest, ListWarehousesResponse, PromoteToStageRequest, PromoteToStageResponse, PromoteToStageSubscribersRequest, PromoteToStageSubscribersResponse, QueryFreightRequest, QueryFreightResponse, RefreshStageRequest, RefreshStageResponse, RefreshWarehouseRequest, RefreshWarehouseResponse, ReverifyRequest, ReverifyResponse, RevokeRequest, RevokeResponse, UpdateCredentialsRequest, UpdateCredentialsResponse, UpdateFreightAliasRequest, UpdateFreightAliasResponse, UpdateResourceRequest, UpdateResourceResponse, UpdateRoleRequest, UpdateRoleResponse } from "./service_pb.js";

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.GetVersionInfo
 */
export const getVersionInfo = {
  localName: "getVersionInfo",
  name: "GetVersionInfo",
  kind: MethodKind.Unary,
  I: GetVersionInfoRequest,
  O: GetVersionInfoResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.GetConfig
 */
export const getConfig = {
  localName: "getConfig",
  name: "GetConfig",
  kind: MethodKind.Unary,
  I: GetConfigRequest,
  O: GetConfigResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.GetPublicConfig
 */
export const getPublicConfig = {
  localName: "getPublicConfig",
  name: "GetPublicConfig",
  kind: MethodKind.Unary,
  I: GetPublicConfigRequest,
  O: GetPublicConfigResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.AdminLogin
 */
export const adminLogin = {
  localName: "adminLogin",
  name: "AdminLogin",
  kind: MethodKind.Unary,
  I: AdminLoginRequest,
  O: AdminLoginResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * TODO(devholic): Add ApplyResource API
 * rpc ApplyResource(ApplyResourceRequest) returns (ApplyResourceRequest);
 *
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.CreateResource
 */
export const createResource = {
  localName: "createResource",
  name: "CreateResource",
  kind: MethodKind.Unary,
  I: CreateResourceRequest,
  O: CreateResourceResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.CreateOrUpdateResource
 */
export const createOrUpdateResource = {
  localName: "createOrUpdateResource",
  name: "CreateOrUpdateResource",
  kind: MethodKind.Unary,
  I: CreateOrUpdateResourceRequest,
  O: CreateOrUpdateResourceResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.UpdateResource
 */
export const updateResource = {
  localName: "updateResource",
  name: "UpdateResource",
  kind: MethodKind.Unary,
  I: UpdateResourceRequest,
  O: UpdateResourceResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.DeleteResource
 */
export const deleteResource = {
  localName: "deleteResource",
  name: "DeleteResource",
  kind: MethodKind.Unary,
  I: DeleteResourceRequest,
  O: DeleteResourceResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.ListStages
 */
export const listStages = {
  localName: "listStages",
  name: "ListStages",
  kind: MethodKind.Unary,
  I: ListStagesRequest,
  O: ListStagesResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.GetStage
 */
export const getStage = {
  localName: "getStage",
  name: "GetStage",
  kind: MethodKind.Unary,
  I: GetStageRequest,
  O: GetStageResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.DeleteStage
 */
export const deleteStage = {
  localName: "deleteStage",
  name: "DeleteStage",
  kind: MethodKind.Unary,
  I: DeleteStageRequest,
  O: DeleteStageResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.RefreshStage
 */
export const refreshStage = {
  localName: "refreshStage",
  name: "RefreshStage",
  kind: MethodKind.Unary,
  I: RefreshStageRequest,
  O: RefreshStageResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.ListPromotions
 */
export const listPromotions = {
  localName: "listPromotions",
  name: "ListPromotions",
  kind: MethodKind.Unary,
  I: ListPromotionsRequest,
  O: ListPromotionsResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.GetPromotion
 */
export const getPromotion = {
  localName: "getPromotion",
  name: "GetPromotion",
  kind: MethodKind.Unary,
  I: GetPromotionRequest,
  O: GetPromotionResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.DeleteProject
 */
export const deleteProject = {
  localName: "deleteProject",
  name: "DeleteProject",
  kind: MethodKind.Unary,
  I: DeleteProjectRequest,
  O: DeleteProjectResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.GetProject
 */
export const getProject = {
  localName: "getProject",
  name: "GetProject",
  kind: MethodKind.Unary,
  I: GetProjectRequest,
  O: GetProjectResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.ListProjects
 */
export const listProjects = {
  localName: "listProjects",
  name: "ListProjects",
  kind: MethodKind.Unary,
  I: ListProjectsRequest,
  O: ListProjectsResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.ListDetailedProjects
 */
export const listDetailedProjects = {
  localName: "listDetailedProjects",
  name: "ListDetailedProjects",
  kind: MethodKind.Unary,
  I: ListDetailedProjectsRequest,
  O: ListDetailedProjectsResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.ApproveFreight
 */
export const approveFreight = {
  localName: "approveFreight",
  name: "ApproveFreight",
  kind: MethodKind.Unary,
  I: ApproveFreightRequest,
  O: ApproveFreightResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.DeleteFreight
 */
export const deleteFreight = {
  localName: "deleteFreight",
  name: "DeleteFreight",
  kind: MethodKind.Unary,
  I: DeleteFreightRequest,
  O: DeleteFreightResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.GetFreight
 */
export const getFreight = {
  localName: "getFreight",
  name: "GetFreight",
  kind: MethodKind.Unary,
  I: GetFreightRequest,
  O: GetFreightResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.PromoteToStage
 */
export const promoteToStage = {
  localName: "promoteToStage",
  name: "PromoteToStage",
  kind: MethodKind.Unary,
  I: PromoteToStageRequest,
  O: PromoteToStageResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.PromoteToStageSubscribers
 */
export const promoteToStageSubscribers = {
  localName: "promoteToStageSubscribers",
  name: "PromoteToStageSubscribers",
  kind: MethodKind.Unary,
  I: PromoteToStageSubscribersRequest,
  O: PromoteToStageSubscribersResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.QueryFreight
 */
export const queryFreight = {
  localName: "queryFreight",
  name: "QueryFreight",
  kind: MethodKind.Unary,
  I: QueryFreightRequest,
  O: QueryFreightResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.UpdateFreightAlias
 */
export const updateFreightAlias = {
  localName: "updateFreightAlias",
  name: "UpdateFreightAlias",
  kind: MethodKind.Unary,
  I: UpdateFreightAliasRequest,
  O: UpdateFreightAliasResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.Reverify
 */
export const reverify = {
  localName: "reverify",
  name: "Reverify",
  kind: MethodKind.Unary,
  I: ReverifyRequest,
  O: ReverifyResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.AbortVerification
 */
export const abortVerification = {
  localName: "abortVerification",
  name: "AbortVerification",
  kind: MethodKind.Unary,
  I: AbortVerificationRequest,
  O: AbortVerificationResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.ListWarehouses
 */
export const listWarehouses = {
  localName: "listWarehouses",
  name: "ListWarehouses",
  kind: MethodKind.Unary,
  I: ListWarehousesRequest,
  O: ListWarehousesResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.GetWarehouse
 */
export const getWarehouse = {
  localName: "getWarehouse",
  name: "GetWarehouse",
  kind: MethodKind.Unary,
  I: GetWarehouseRequest,
  O: GetWarehouseResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.DeleteWarehouse
 */
export const deleteWarehouse = {
  localName: "deleteWarehouse",
  name: "DeleteWarehouse",
  kind: MethodKind.Unary,
  I: DeleteWarehouseRequest,
  O: DeleteWarehouseResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.RefreshWarehouse
 */
export const refreshWarehouse = {
  localName: "refreshWarehouse",
  name: "RefreshWarehouse",
  kind: MethodKind.Unary,
  I: RefreshWarehouseRequest,
  O: RefreshWarehouseResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.CreateCredentials
 */
export const createCredentials = {
  localName: "createCredentials",
  name: "CreateCredentials",
  kind: MethodKind.Unary,
  I: CreateCredentialsRequest,
  O: CreateCredentialsResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.DeleteCredentials
 */
export const deleteCredentials = {
  localName: "deleteCredentials",
  name: "DeleteCredentials",
  kind: MethodKind.Unary,
  I: DeleteCredentialsRequest,
  O: DeleteCredentialsResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.GetCredentials
 */
export const getCredentials = {
  localName: "getCredentials",
  name: "GetCredentials",
  kind: MethodKind.Unary,
  I: GetCredentialsRequest,
  O: GetCredentialsResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.ListCredentials
 */
export const listCredentials = {
  localName: "listCredentials",
  name: "ListCredentials",
  kind: MethodKind.Unary,
  I: ListCredentialsRequest,
  O: ListCredentialsResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.UpdateCredentials
 */
export const updateCredentials = {
  localName: "updateCredentials",
  name: "UpdateCredentials",
  kind: MethodKind.Unary,
  I: UpdateCredentialsRequest,
  O: UpdateCredentialsResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.ListAnalysisTemplates
 */
export const listAnalysisTemplates = {
  localName: "listAnalysisTemplates",
  name: "ListAnalysisTemplates",
  kind: MethodKind.Unary,
  I: ListAnalysisTemplatesRequest,
  O: ListAnalysisTemplatesResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.GetAnalysisTemplate
 */
export const getAnalysisTemplate = {
  localName: "getAnalysisTemplate",
  name: "GetAnalysisTemplate",
  kind: MethodKind.Unary,
  I: GetAnalysisTemplateRequest,
  O: GetAnalysisTemplateResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.DeleteAnalysisTemplate
 */
export const deleteAnalysisTemplate = {
  localName: "deleteAnalysisTemplate",
  name: "DeleteAnalysisTemplate",
  kind: MethodKind.Unary,
  I: DeleteAnalysisTemplateRequest,
  O: DeleteAnalysisTemplateResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.GetAnalysisRun
 */
export const getAnalysisRun = {
  localName: "getAnalysisRun",
  name: "GetAnalysisRun",
  kind: MethodKind.Unary,
  I: GetAnalysisRunRequest,
  O: GetAnalysisRunResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.ListProjectEvents
 */
export const listProjectEvents = {
  localName: "listProjectEvents",
  name: "ListProjectEvents",
  kind: MethodKind.Unary,
  I: ListProjectEventsRequest,
  O: ListProjectEventsResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.CreateRole
 */
export const createRole = {
  localName: "createRole",
  name: "CreateRole",
  kind: MethodKind.Unary,
  I: CreateRoleRequest,
  O: CreateRoleResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.DeleteRole
 */
export const deleteRole = {
  localName: "deleteRole",
  name: "DeleteRole",
  kind: MethodKind.Unary,
  I: DeleteRoleRequest,
  O: DeleteRoleResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.GetRole
 */
export const getRole = {
  localName: "getRole",
  name: "GetRole",
  kind: MethodKind.Unary,
  I: GetRoleRequest,
  O: GetRoleResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.Grant
 */
export const grant = {
  localName: "grant",
  name: "Grant",
  kind: MethodKind.Unary,
  I: GrantRequest,
  O: GrantResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.ListRoles
 */
export const listRoles = {
  localName: "listRoles",
  name: "ListRoles",
  kind: MethodKind.Unary,
  I: ListRolesRequest,
  O: ListRolesResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.Revoke
 */
export const revoke = {
  localName: "revoke",
  name: "Revoke",
  kind: MethodKind.Unary,
  I: RevokeRequest,
  O: RevokeResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.UpdateRole
 */
export const updateRole = {
  localName: "updateRole",
  name: "UpdateRole",
  kind: MethodKind.Unary,
  I: UpdateRoleRequest,
  O: UpdateRoleResponse,
  service: {
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService"
  }
} as const;
