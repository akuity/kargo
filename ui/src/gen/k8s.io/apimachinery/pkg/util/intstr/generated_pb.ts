// This file was autogenerated by go-to-protobuf. Do not edit it manually!

// @generated by protoc-gen-es v1.8.0 with parameter "target=ts"
// @generated from file k8s.io/apimachinery/pkg/util/intstr/generated.proto (package k8s.io.apimachinery.pkg.util.intstr, syntax proto2)
/* eslint-disable */
// @ts-nocheck

import type { BinaryReadOptions, FieldList, JsonReadOptions, JsonValue, PartialMessage, PlainMessage } from "@bufbuild/protobuf";
import { Message, proto2 } from "@bufbuild/protobuf";

/**
 * IntOrString is a type that can hold an int32 or a string.  When used in
 * JSON or YAML marshalling and unmarshalling, it produces or consumes the
 * inner type.  This allows you to have, for example, a JSON field that can
 * accept a name or number.
 * TODO: Rename to Int32OrString
 *
 * +protobuf=true
 * +protobuf.options.(gogoproto.goproto_stringer)=false
 * +k8s:openapi-gen=true
 *
 * @generated from message k8s.io.apimachinery.pkg.util.intstr.IntOrString
 */
export class IntOrString extends Message<IntOrString> {
  /**
   * @generated from field: optional int64 type = 1;
   */
  type?: bigint;

  /**
   * @generated from field: optional int32 intVal = 2;
   */
  intVal?: number;

  /**
   * @generated from field: optional string strVal = 3;
   */
  strVal?: string;

  constructor(data?: PartialMessage<IntOrString>) {
    super();
    proto2.util.initPartial(data, this);
  }

  static readonly runtime: typeof proto2 = proto2;
  static readonly typeName = "k8s.io.apimachinery.pkg.util.intstr.IntOrString";
  static readonly fields: FieldList = proto2.util.newFieldList(() => [
    { no: 1, name: "type", kind: "scalar", T: 3 /* ScalarType.INT64 */, opt: true },
    { no: 2, name: "intVal", kind: "scalar", T: 5 /* ScalarType.INT32 */, opt: true },
    { no: 3, name: "strVal", kind: "scalar", T: 9 /* ScalarType.STRING */, opt: true },
  ]);

  static fromBinary(bytes: Uint8Array, options?: Partial<BinaryReadOptions>): IntOrString {
    return new IntOrString().fromBinary(bytes, options);
  }

  static fromJson(jsonValue: JsonValue, options?: Partial<JsonReadOptions>): IntOrString {
    return new IntOrString().fromJson(jsonValue, options);
  }

  static fromJsonString(jsonString: string, options?: Partial<JsonReadOptions>): IntOrString {
    return new IntOrString().fromJsonString(jsonString, options);
  }

  static equals(a: IntOrString | PlainMessage<IntOrString> | undefined, b: IntOrString | PlainMessage<IntOrString> | undefined): boolean {
    return proto2.util.equals(IntOrString, a, b);
  }
}

