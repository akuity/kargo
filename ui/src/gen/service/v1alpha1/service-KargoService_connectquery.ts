// @generated by protoc-gen-connect-query v0.4.1 with parameter "target=ts"
// @generated from file service/v1alpha1/service.proto (package akuity.io.kargo.service.v1alpha1, syntax proto3)
/* eslint-disable */
// @ts-nocheck

import { createQueryService } from "@bufbuild/connect-query";
import { MethodKind } from "@bufbuild/protobuf";
import { AdminLoginRequest, AdminLoginResponse, ApproveFreightRequest, ApproveFreightResponse, CreateOrUpdateResourceRequest, CreateOrUpdateResourceResponse, CreateResourceRequest, CreateResourceResponse, DeleteFreightRequest, DeleteFreightResponse, DeleteProjectRequest, DeleteProjectResponse, DeleteResourceRequest, DeleteResourceResponse, DeleteStageRequest, DeleteStageResponse, DeleteWarehouseRequest, DeleteWarehouseResponse, GetConfigRequest, GetConfigResponse, GetFreightRequest, GetFreightResponse, GetProjectRequest, GetProjectResponse, GetPromotionRequest, GetPromotionResponse, GetPublicConfigRequest, GetPublicConfigResponse, GetStageRequest, GetStageResponse, GetVersionInfoRequest, GetVersionInfoResponse, GetWarehouseRequest, GetWarehouseResponse, ListProjectsRequest, ListProjectsResponse, ListPromotionsRequest, ListPromotionsResponse, ListStagesRequest, ListStagesResponse, ListWarehousesRequest, ListWarehousesResponse, PromoteStageRequest, PromoteStageResponse, PromoteSubscribersRequest, PromoteSubscribersResponse, QueryFreightRequest, QueryFreightResponse, RefreshStageRequest, RefreshStageResponse, RefreshWarehouseRequest, RefreshWarehouseResponse, UpdateFreightAliasRequest, UpdateFreightAliasResponse, UpdateResourceRequest, UpdateResourceResponse } from "./service_pb.js";

export const typeName = "akuity.io.kargo.service.v1alpha1.KargoService";

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.GetVersionInfo
 */
export const getVersionInfo = createQueryService({
  service: {
    methods: {
      getVersionInfo: {
        name: "GetVersionInfo",
        kind: MethodKind.Unary,
        I: GetVersionInfoRequest,
        O: GetVersionInfoResponse,
      },
    },
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService",
  },
}).getVersionInfo;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.GetConfig
 */
export const getConfig = createQueryService({
  service: {
    methods: {
      getConfig: {
        name: "GetConfig",
        kind: MethodKind.Unary,
        I: GetConfigRequest,
        O: GetConfigResponse,
      },
    },
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService",
  },
}).getConfig;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.GetPublicConfig
 */
export const getPublicConfig = createQueryService({
  service: {
    methods: {
      getPublicConfig: {
        name: "GetPublicConfig",
        kind: MethodKind.Unary,
        I: GetPublicConfigRequest,
        O: GetPublicConfigResponse,
      },
    },
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService",
  },
}).getPublicConfig;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.AdminLogin
 */
export const adminLogin = createQueryService({
  service: {
    methods: {
      adminLogin: {
        name: "AdminLogin",
        kind: MethodKind.Unary,
        I: AdminLoginRequest,
        O: AdminLoginResponse,
      },
    },
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService",
  },
}).adminLogin;

/**
 * TODO(devholic): Add ApplyResource API
 * rpc ApplyResource(ApplyResourceRequest) returns (ApplyResourceRequest);
 *
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.CreateResource
 */
export const createResource = createQueryService({
  service: {
    methods: {
      createResource: {
        name: "CreateResource",
        kind: MethodKind.Unary,
        I: CreateResourceRequest,
        O: CreateResourceResponse,
      },
    },
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService",
  },
}).createResource;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.CreateOrUpdateResource
 */
export const createOrUpdateResource = createQueryService({
  service: {
    methods: {
      createOrUpdateResource: {
        name: "CreateOrUpdateResource",
        kind: MethodKind.Unary,
        I: CreateOrUpdateResourceRequest,
        O: CreateOrUpdateResourceResponse,
      },
    },
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService",
  },
}).createOrUpdateResource;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.UpdateResource
 */
export const updateResource = createQueryService({
  service: {
    methods: {
      updateResource: {
        name: "UpdateResource",
        kind: MethodKind.Unary,
        I: UpdateResourceRequest,
        O: UpdateResourceResponse,
      },
    },
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService",
  },
}).updateResource;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.DeleteResource
 */
export const deleteResource = createQueryService({
  service: {
    methods: {
      deleteResource: {
        name: "DeleteResource",
        kind: MethodKind.Unary,
        I: DeleteResourceRequest,
        O: DeleteResourceResponse,
      },
    },
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService",
  },
}).deleteResource;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.ListStages
 */
export const listStages = createQueryService({
  service: {
    methods: {
      listStages: {
        name: "ListStages",
        kind: MethodKind.Unary,
        I: ListStagesRequest,
        O: ListStagesResponse,
      },
    },
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService",
  },
}).listStages;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.GetStage
 */
export const getStage = createQueryService({
  service: {
    methods: {
      getStage: {
        name: "GetStage",
        kind: MethodKind.Unary,
        I: GetStageRequest,
        O: GetStageResponse,
      },
    },
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService",
  },
}).getStage;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.DeleteStage
 */
export const deleteStage = createQueryService({
  service: {
    methods: {
      deleteStage: {
        name: "DeleteStage",
        kind: MethodKind.Unary,
        I: DeleteStageRequest,
        O: DeleteStageResponse,
      },
    },
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService",
  },
}).deleteStage;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.PromoteStage
 */
export const promoteStage = createQueryService({
  service: {
    methods: {
      promoteStage: {
        name: "PromoteStage",
        kind: MethodKind.Unary,
        I: PromoteStageRequest,
        O: PromoteStageResponse,
      },
    },
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService",
  },
}).promoteStage;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.PromoteSubscribers
 */
export const promoteSubscribers = createQueryService({
  service: {
    methods: {
      promoteSubscribers: {
        name: "PromoteSubscribers",
        kind: MethodKind.Unary,
        I: PromoteSubscribersRequest,
        O: PromoteSubscribersResponse,
      },
    },
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService",
  },
}).promoteSubscribers;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.RefreshStage
 */
export const refreshStage = createQueryService({
  service: {
    methods: {
      refreshStage: {
        name: "RefreshStage",
        kind: MethodKind.Unary,
        I: RefreshStageRequest,
        O: RefreshStageResponse,
      },
    },
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService",
  },
}).refreshStage;

/**
 * Promotion APIs 
 *
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.ListPromotions
 */
export const listPromotions = createQueryService({
  service: {
    methods: {
      listPromotions: {
        name: "ListPromotions",
        kind: MethodKind.Unary,
        I: ListPromotionsRequest,
        O: ListPromotionsResponse,
      },
    },
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService",
  },
}).listPromotions;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.GetPromotion
 */
export const getPromotion = createQueryService({
  service: {
    methods: {
      getPromotion: {
        name: "GetPromotion",
        kind: MethodKind.Unary,
        I: GetPromotionRequest,
        O: GetPromotionResponse,
      },
    },
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService",
  },
}).getPromotion;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.DeleteProject
 */
export const deleteProject = createQueryService({
  service: {
    methods: {
      deleteProject: {
        name: "DeleteProject",
        kind: MethodKind.Unary,
        I: DeleteProjectRequest,
        O: DeleteProjectResponse,
      },
    },
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService",
  },
}).deleteProject;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.GetProject
 */
export const getProject = createQueryService({
  service: {
    methods: {
      getProject: {
        name: "GetProject",
        kind: MethodKind.Unary,
        I: GetProjectRequest,
        O: GetProjectResponse,
      },
    },
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService",
  },
}).getProject;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.ListProjects
 */
export const listProjects = createQueryService({
  service: {
    methods: {
      listProjects: {
        name: "ListProjects",
        kind: MethodKind.Unary,
        I: ListProjectsRequest,
        O: ListProjectsResponse,
      },
    },
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService",
  },
}).listProjects;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.ApproveFreight
 */
export const approveFreight = createQueryService({
  service: {
    methods: {
      approveFreight: {
        name: "ApproveFreight",
        kind: MethodKind.Unary,
        I: ApproveFreightRequest,
        O: ApproveFreightResponse,
      },
    },
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService",
  },
}).approveFreight;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.DeleteFreight
 */
export const deleteFreight = createQueryService({
  service: {
    methods: {
      deleteFreight: {
        name: "DeleteFreight",
        kind: MethodKind.Unary,
        I: DeleteFreightRequest,
        O: DeleteFreightResponse,
      },
    },
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService",
  },
}).deleteFreight;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.GetFreight
 */
export const getFreight = createQueryService({
  service: {
    methods: {
      getFreight: {
        name: "GetFreight",
        kind: MethodKind.Unary,
        I: GetFreightRequest,
        O: GetFreightResponse,
      },
    },
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService",
  },
}).getFreight;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.QueryFreight
 */
export const queryFreight = createQueryService({
  service: {
    methods: {
      queryFreight: {
        name: "QueryFreight",
        kind: MethodKind.Unary,
        I: QueryFreightRequest,
        O: QueryFreightResponse,
      },
    },
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService",
  },
}).queryFreight;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.UpdateFreightAlias
 */
export const updateFreightAlias = createQueryService({
  service: {
    methods: {
      updateFreightAlias: {
        name: "UpdateFreightAlias",
        kind: MethodKind.Unary,
        I: UpdateFreightAliasRequest,
        O: UpdateFreightAliasResponse,
      },
    },
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService",
  },
}).updateFreightAlias;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.ListWarehouses
 */
export const listWarehouses = createQueryService({
  service: {
    methods: {
      listWarehouses: {
        name: "ListWarehouses",
        kind: MethodKind.Unary,
        I: ListWarehousesRequest,
        O: ListWarehousesResponse,
      },
    },
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService",
  },
}).listWarehouses;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.GetWarehouse
 */
export const getWarehouse = createQueryService({
  service: {
    methods: {
      getWarehouse: {
        name: "GetWarehouse",
        kind: MethodKind.Unary,
        I: GetWarehouseRequest,
        O: GetWarehouseResponse,
      },
    },
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService",
  },
}).getWarehouse;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.DeleteWarehouse
 */
export const deleteWarehouse = createQueryService({
  service: {
    methods: {
      deleteWarehouse: {
        name: "DeleteWarehouse",
        kind: MethodKind.Unary,
        I: DeleteWarehouseRequest,
        O: DeleteWarehouseResponse,
      },
    },
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService",
  },
}).deleteWarehouse;

/**
 * @generated from rpc akuity.io.kargo.service.v1alpha1.KargoService.RefreshWarehouse
 */
export const refreshWarehouse = createQueryService({
  service: {
    methods: {
      refreshWarehouse: {
        name: "RefreshWarehouse",
        kind: MethodKind.Unary,
        I: RefreshWarehouseRequest,
        O: RefreshWarehouseResponse,
      },
    },
    typeName: "akuity.io.kargo.service.v1alpha1.KargoService",
  },
}).refreshWarehouse;
