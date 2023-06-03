// @generated by protoc-gen-es v1.2.0 with parameter "target=ts"
// @generated from file service/v1alpha1/service.proto (package akuity.io.kargo.service.v1alpha1, syntax proto3)
/* eslint-disable */
// @ts-nocheck

import type { BinaryReadOptions, FieldList, JsonReadOptions, JsonValue, PartialMessage, PlainMessage } from "@bufbuild/protobuf";
import { Message, proto3 } from "@bufbuild/protobuf";
import { Environment, Promotion } from "../../v1alpha1/generated_pb.js";

/**
 * @generated from message akuity.io.kargo.service.v1alpha1.ListEnvironmentsRequest
 */
export class ListEnvironmentsRequest extends Message<ListEnvironmentsRequest> {
  /**
   * @generated from field: string project = 1;
   */
  project = "";

  constructor(data?: PartialMessage<ListEnvironmentsRequest>) {
    super();
    proto3.util.initPartial(data, this);
  }

  static readonly runtime: typeof proto3 = proto3;
  static readonly typeName = "akuity.io.kargo.service.v1alpha1.ListEnvironmentsRequest";
  static readonly fields: FieldList = proto3.util.newFieldList(() => [
    { no: 1, name: "project", kind: "scalar", T: 9 /* ScalarType.STRING */ },
  ]);

  static fromBinary(bytes: Uint8Array, options?: Partial<BinaryReadOptions>): ListEnvironmentsRequest {
    return new ListEnvironmentsRequest().fromBinary(bytes, options);
  }

  static fromJson(jsonValue: JsonValue, options?: Partial<JsonReadOptions>): ListEnvironmentsRequest {
    return new ListEnvironmentsRequest().fromJson(jsonValue, options);
  }

  static fromJsonString(jsonString: string, options?: Partial<JsonReadOptions>): ListEnvironmentsRequest {
    return new ListEnvironmentsRequest().fromJsonString(jsonString, options);
  }

  static equals(a: ListEnvironmentsRequest | PlainMessage<ListEnvironmentsRequest> | undefined, b: ListEnvironmentsRequest | PlainMessage<ListEnvironmentsRequest> | undefined): boolean {
    return proto3.util.equals(ListEnvironmentsRequest, a, b);
  }
}

/**
 * @generated from message akuity.io.kargo.service.v1alpha1.ListEnvironmentsResponse
 */
export class ListEnvironmentsResponse extends Message<ListEnvironmentsResponse> {
  /**
   * @generated from field: repeated github.com.akuity.kargo.pkg.api.v1alpha1.Environment environments = 1;
   */
  environments: Environment[] = [];

  constructor(data?: PartialMessage<ListEnvironmentsResponse>) {
    super();
    proto3.util.initPartial(data, this);
  }

  static readonly runtime: typeof proto3 = proto3;
  static readonly typeName = "akuity.io.kargo.service.v1alpha1.ListEnvironmentsResponse";
  static readonly fields: FieldList = proto3.util.newFieldList(() => [
    { no: 1, name: "environments", kind: "message", T: Environment, repeated: true },
  ]);

  static fromBinary(bytes: Uint8Array, options?: Partial<BinaryReadOptions>): ListEnvironmentsResponse {
    return new ListEnvironmentsResponse().fromBinary(bytes, options);
  }

  static fromJson(jsonValue: JsonValue, options?: Partial<JsonReadOptions>): ListEnvironmentsResponse {
    return new ListEnvironmentsResponse().fromJson(jsonValue, options);
  }

  static fromJsonString(jsonString: string, options?: Partial<JsonReadOptions>): ListEnvironmentsResponse {
    return new ListEnvironmentsResponse().fromJsonString(jsonString, options);
  }

  static equals(a: ListEnvironmentsResponse | PlainMessage<ListEnvironmentsResponse> | undefined, b: ListEnvironmentsResponse | PlainMessage<ListEnvironmentsResponse> | undefined): boolean {
    return proto3.util.equals(ListEnvironmentsResponse, a, b);
  }
}

/**
 * @generated from message akuity.io.kargo.service.v1alpha1.GetEnvironmentRequest
 */
export class GetEnvironmentRequest extends Message<GetEnvironmentRequest> {
  /**
   * @generated from field: string project = 1;
   */
  project = "";

  /**
   * @generated from field: string name = 2;
   */
  name = "";

  constructor(data?: PartialMessage<GetEnvironmentRequest>) {
    super();
    proto3.util.initPartial(data, this);
  }

  static readonly runtime: typeof proto3 = proto3;
  static readonly typeName = "akuity.io.kargo.service.v1alpha1.GetEnvironmentRequest";
  static readonly fields: FieldList = proto3.util.newFieldList(() => [
    { no: 1, name: "project", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 2, name: "name", kind: "scalar", T: 9 /* ScalarType.STRING */ },
  ]);

  static fromBinary(bytes: Uint8Array, options?: Partial<BinaryReadOptions>): GetEnvironmentRequest {
    return new GetEnvironmentRequest().fromBinary(bytes, options);
  }

  static fromJson(jsonValue: JsonValue, options?: Partial<JsonReadOptions>): GetEnvironmentRequest {
    return new GetEnvironmentRequest().fromJson(jsonValue, options);
  }

  static fromJsonString(jsonString: string, options?: Partial<JsonReadOptions>): GetEnvironmentRequest {
    return new GetEnvironmentRequest().fromJsonString(jsonString, options);
  }

  static equals(a: GetEnvironmentRequest | PlainMessage<GetEnvironmentRequest> | undefined, b: GetEnvironmentRequest | PlainMessage<GetEnvironmentRequest> | undefined): boolean {
    return proto3.util.equals(GetEnvironmentRequest, a, b);
  }
}

/**
 * @generated from message akuity.io.kargo.service.v1alpha1.GetEnvironmentResponse
 */
export class GetEnvironmentResponse extends Message<GetEnvironmentResponse> {
  /**
   * @generated from field: github.com.akuity.kargo.pkg.api.v1alpha1.Environment environment = 1;
   */
  environment?: Environment;

  constructor(data?: PartialMessage<GetEnvironmentResponse>) {
    super();
    proto3.util.initPartial(data, this);
  }

  static readonly runtime: typeof proto3 = proto3;
  static readonly typeName = "akuity.io.kargo.service.v1alpha1.GetEnvironmentResponse";
  static readonly fields: FieldList = proto3.util.newFieldList(() => [
    { no: 1, name: "environment", kind: "message", T: Environment },
  ]);

  static fromBinary(bytes: Uint8Array, options?: Partial<BinaryReadOptions>): GetEnvironmentResponse {
    return new GetEnvironmentResponse().fromBinary(bytes, options);
  }

  static fromJson(jsonValue: JsonValue, options?: Partial<JsonReadOptions>): GetEnvironmentResponse {
    return new GetEnvironmentResponse().fromJson(jsonValue, options);
  }

  static fromJsonString(jsonString: string, options?: Partial<JsonReadOptions>): GetEnvironmentResponse {
    return new GetEnvironmentResponse().fromJsonString(jsonString, options);
  }

  static equals(a: GetEnvironmentResponse | PlainMessage<GetEnvironmentResponse> | undefined, b: GetEnvironmentResponse | PlainMessage<GetEnvironmentResponse> | undefined): boolean {
    return proto3.util.equals(GetEnvironmentResponse, a, b);
  }
}

/**
 * @generated from message akuity.io.kargo.service.v1alpha1.PromoteEnvironmentRequest
 */
export class PromoteEnvironmentRequest extends Message<PromoteEnvironmentRequest> {
  /**
   * @generated from field: string project = 1;
   */
  project = "";

  /**
   * @generated from field: string name = 2;
   */
  name = "";

  /**
   * @generated from field: string state = 3;
   */
  state = "";

  constructor(data?: PartialMessage<PromoteEnvironmentRequest>) {
    super();
    proto3.util.initPartial(data, this);
  }

  static readonly runtime: typeof proto3 = proto3;
  static readonly typeName = "akuity.io.kargo.service.v1alpha1.PromoteEnvironmentRequest";
  static readonly fields: FieldList = proto3.util.newFieldList(() => [
    { no: 1, name: "project", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 2, name: "name", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 3, name: "state", kind: "scalar", T: 9 /* ScalarType.STRING */ },
  ]);

  static fromBinary(bytes: Uint8Array, options?: Partial<BinaryReadOptions>): PromoteEnvironmentRequest {
    return new PromoteEnvironmentRequest().fromBinary(bytes, options);
  }

  static fromJson(jsonValue: JsonValue, options?: Partial<JsonReadOptions>): PromoteEnvironmentRequest {
    return new PromoteEnvironmentRequest().fromJson(jsonValue, options);
  }

  static fromJsonString(jsonString: string, options?: Partial<JsonReadOptions>): PromoteEnvironmentRequest {
    return new PromoteEnvironmentRequest().fromJsonString(jsonString, options);
  }

  static equals(a: PromoteEnvironmentRequest | PlainMessage<PromoteEnvironmentRequest> | undefined, b: PromoteEnvironmentRequest | PlainMessage<PromoteEnvironmentRequest> | undefined): boolean {
    return proto3.util.equals(PromoteEnvironmentRequest, a, b);
  }
}

/**
 * @generated from message akuity.io.kargo.service.v1alpha1.PromoteEnvironmentResponse
 */
export class PromoteEnvironmentResponse extends Message<PromoteEnvironmentResponse> {
  /**
   * @generated from field: github.com.akuity.kargo.pkg.api.v1alpha1.Promotion promotion = 1;
   */
  promotion?: Promotion;

  constructor(data?: PartialMessage<PromoteEnvironmentResponse>) {
    super();
    proto3.util.initPartial(data, this);
  }

  static readonly runtime: typeof proto3 = proto3;
  static readonly typeName = "akuity.io.kargo.service.v1alpha1.PromoteEnvironmentResponse";
  static readonly fields: FieldList = proto3.util.newFieldList(() => [
    { no: 1, name: "promotion", kind: "message", T: Promotion },
  ]);

  static fromBinary(bytes: Uint8Array, options?: Partial<BinaryReadOptions>): PromoteEnvironmentResponse {
    return new PromoteEnvironmentResponse().fromBinary(bytes, options);
  }

  static fromJson(jsonValue: JsonValue, options?: Partial<JsonReadOptions>): PromoteEnvironmentResponse {
    return new PromoteEnvironmentResponse().fromJson(jsonValue, options);
  }

  static fromJsonString(jsonString: string, options?: Partial<JsonReadOptions>): PromoteEnvironmentResponse {
    return new PromoteEnvironmentResponse().fromJsonString(jsonString, options);
  }

  static equals(a: PromoteEnvironmentResponse | PlainMessage<PromoteEnvironmentResponse> | undefined, b: PromoteEnvironmentResponse | PlainMessage<PromoteEnvironmentResponse> | undefined): boolean {
    return proto3.util.equals(PromoteEnvironmentResponse, a, b);
  }
}

