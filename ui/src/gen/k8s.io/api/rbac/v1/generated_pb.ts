//
//Copyright The Kubernetes Authors.
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.

// This file was autogenerated by go-to-protobuf. Do not edit it manually!

// @generated by protoc-gen-es v2.2.2 with parameter "target=ts"
// @generated from file k8s.io/api/rbac/v1/generated.proto (package k8s.io.api.rbac.v1, syntax proto2)
/* eslint-disable */

import type { GenFile, GenMessage } from "@bufbuild/protobuf/codegenv1";
import { fileDesc, messageDesc } from "@bufbuild/protobuf/codegenv1";
import type { LabelSelector, ListMeta, ObjectMeta } from "../../../apimachinery/pkg/apis/meta/v1/generated_pb";
import { file_k8s_io_apimachinery_pkg_apis_meta_v1_generated } from "../../../apimachinery/pkg/apis/meta/v1/generated_pb";
import { file_k8s_io_apimachinery_pkg_runtime_generated } from "../../../apimachinery/pkg/runtime/generated_pb";
import { file_k8s_io_apimachinery_pkg_runtime_schema_generated } from "../../../apimachinery/pkg/runtime/schema/generated_pb";
import type { Message } from "@bufbuild/protobuf";

/**
 * Describes the file k8s.io/api/rbac/v1/generated.proto.
 */
export const file_k8s_io_api_rbac_v1_generated: GenFile = /*@__PURE__*/
  fileDesc("CiJrOHMuaW8vYXBpL3JiYWMvdjEvZ2VuZXJhdGVkLnByb3RvEhJrOHMuaW8uYXBpLnJiYWMudjEiZAoPQWdncmVnYXRpb25SdWxlElEKFGNsdXN0ZXJSb2xlU2VsZWN0b3JzGAEgAygLMjMuazhzLmlvLmFwaW1hY2hpbmVyeS5wa2cuYXBpcy5tZXRhLnYxLkxhYmVsU2VsZWN0b3IivgEKC0NsdXN0ZXJSb2xlEkIKCG1ldGFkYXRhGAEgASgLMjAuazhzLmlvLmFwaW1hY2hpbmVyeS5wa2cuYXBpcy5tZXRhLnYxLk9iamVjdE1ldGESLQoFcnVsZXMYAiADKAsyHi5rOHMuaW8uYXBpLnJiYWMudjEuUG9saWN5UnVsZRI8Cg9hZ2dyZWdhdGlvblJ1bGUYAyABKAsyIy5rOHMuaW8uYXBpLnJiYWMudjEuQWdncmVnYXRpb25SdWxlIrUBChJDbHVzdGVyUm9sZUJpbmRpbmcSQgoIbWV0YWRhdGEYASABKAsyMC5rOHMuaW8uYXBpbWFjaGluZXJ5LnBrZy5hcGlzLm1ldGEudjEuT2JqZWN0TWV0YRItCghzdWJqZWN0cxgCIAMoCzIbLms4cy5pby5hcGkucmJhYy52MS5TdWJqZWN0EiwKB3JvbGVSZWYYAyABKAsyGy5rOHMuaW8uYXBpLnJiYWMudjEuUm9sZVJlZiKRAQoWQ2x1c3RlclJvbGVCaW5kaW5nTGlzdBJACghtZXRhZGF0YRgBIAEoCzIuLms4cy5pby5hcGltYWNoaW5lcnkucGtnLmFwaXMubWV0YS52MS5MaXN0TWV0YRI1CgVpdGVtcxgCIAMoCzImLms4cy5pby5hcGkucmJhYy52MS5DbHVzdGVyUm9sZUJpbmRpbmcigwEKD0NsdXN0ZXJSb2xlTGlzdBJACghtZXRhZGF0YRgBIAEoCzIuLms4cy5pby5hcGltYWNoaW5lcnkucGtnLmFwaXMubWV0YS52MS5MaXN0TWV0YRIuCgVpdGVtcxgCIAMoCzIfLms4cy5pby5hcGkucmJhYy52MS5DbHVzdGVyUm9sZSJxCgpQb2xpY3lSdWxlEg0KBXZlcmJzGAEgAygJEhEKCWFwaUdyb3VwcxgCIAMoCRIRCglyZXNvdXJjZXMYAyADKAkSFQoNcmVzb3VyY2VOYW1lcxgEIAMoCRIXCg9ub25SZXNvdXJjZVVSTHMYBSADKAkieQoEUm9sZRJCCghtZXRhZGF0YRgBIAEoCzIwLms4cy5pby5hcGltYWNoaW5lcnkucGtnLmFwaXMubWV0YS52MS5PYmplY3RNZXRhEi0KBXJ1bGVzGAIgAygLMh4uazhzLmlvLmFwaS5yYmFjLnYxLlBvbGljeVJ1bGUirgEKC1JvbGVCaW5kaW5nEkIKCG1ldGFkYXRhGAEgASgLMjAuazhzLmlvLmFwaW1hY2hpbmVyeS5wa2cuYXBpcy5tZXRhLnYxLk9iamVjdE1ldGESLQoIc3ViamVjdHMYAiADKAsyGy5rOHMuaW8uYXBpLnJiYWMudjEuU3ViamVjdBIsCgdyb2xlUmVmGAMgASgLMhsuazhzLmlvLmFwaS5yYmFjLnYxLlJvbGVSZWYigwEKD1JvbGVCaW5kaW5nTGlzdBJACghtZXRhZGF0YRgBIAEoCzIuLms4cy5pby5hcGltYWNoaW5lcnkucGtnLmFwaXMubWV0YS52MS5MaXN0TWV0YRIuCgVpdGVtcxgCIAMoCzIfLms4cy5pby5hcGkucmJhYy52MS5Sb2xlQmluZGluZyJ1CghSb2xlTGlzdBJACghtZXRhZGF0YRgBIAEoCzIuLms4cy5pby5hcGltYWNoaW5lcnkucGtnLmFwaXMubWV0YS52MS5MaXN0TWV0YRInCgVpdGVtcxgCIAMoCzIYLms4cy5pby5hcGkucmJhYy52MS5Sb2xlIjcKB1JvbGVSZWYSEAoIYXBpR3JvdXAYASABKAkSDAoEa2luZBgCIAEoCRIMCgRuYW1lGAMgASgJIkoKB1N1YmplY3QSDAoEa2luZBgBIAEoCRIQCghhcGlHcm91cBgCIAEoCRIMCgRuYW1lGAMgASgJEhEKCW5hbWVzcGFjZRgEIAEoCUKpAQoWY29tLms4cy5pby5hcGkucmJhYy52MUIOR2VuZXJhdGVkUHJvdG9QAVoSazhzLmlvL2FwaS9yYmFjL3YxogIES0lBUqoCEks4cy5Jby5BcGkuUmJhYy5WMcoCEks4c1xJb1xBcGlcUmJhY1xWMeICHks4c1xJb1xBcGlcUmJhY1xWMVxHUEJNZXRhZGF0YeoCFks4czo6SW86OkFwaTo6UmJhYzo6VjE", [file_k8s_io_apimachinery_pkg_apis_meta_v1_generated, file_k8s_io_apimachinery_pkg_runtime_generated, file_k8s_io_apimachinery_pkg_runtime_schema_generated]);

/**
 * AggregationRule describes how to locate ClusterRoles to aggregate into the ClusterRole
 *
 * @generated from message k8s.io.api.rbac.v1.AggregationRule
 */
export type AggregationRule = Message<"k8s.io.api.rbac.v1.AggregationRule"> & {
  /**
   * ClusterRoleSelectors holds a list of selectors which will be used to find ClusterRoles and create the rules.
   * If any of the selectors match, then the ClusterRole's permissions will be added
   * +optional
   * +listType=atomic
   *
   * @generated from field: repeated k8s.io.apimachinery.pkg.apis.meta.v1.LabelSelector clusterRoleSelectors = 1;
   */
  clusterRoleSelectors: LabelSelector[];
};

/**
 * Describes the message k8s.io.api.rbac.v1.AggregationRule.
 * Use `create(AggregationRuleSchema)` to create a new message.
 */
export const AggregationRuleSchema: GenMessage<AggregationRule> = /*@__PURE__*/
  messageDesc(file_k8s_io_api_rbac_v1_generated, 0);

/**
 * ClusterRole is a cluster level, logical grouping of PolicyRules that can be referenced as a unit by a RoleBinding or ClusterRoleBinding.
 *
 * @generated from message k8s.io.api.rbac.v1.ClusterRole
 */
export type ClusterRole = Message<"k8s.io.api.rbac.v1.ClusterRole"> & {
  /**
   * Standard object's metadata.
   * +optional
   *
   * @generated from field: optional k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta metadata = 1;
   */
  metadata?: ObjectMeta;

  /**
   * Rules holds all the PolicyRules for this ClusterRole
   * +optional
   * +listType=atomic
   *
   * @generated from field: repeated k8s.io.api.rbac.v1.PolicyRule rules = 2;
   */
  rules: PolicyRule[];

  /**
   * AggregationRule is an optional field that describes how to build the Rules for this ClusterRole.
   * If AggregationRule is set, then the Rules are controller managed and direct changes to Rules will be
   * stomped by the controller.
   * +optional
   *
   * @generated from field: optional k8s.io.api.rbac.v1.AggregationRule aggregationRule = 3;
   */
  aggregationRule?: AggregationRule;
};

/**
 * Describes the message k8s.io.api.rbac.v1.ClusterRole.
 * Use `create(ClusterRoleSchema)` to create a new message.
 */
export const ClusterRoleSchema: GenMessage<ClusterRole> = /*@__PURE__*/
  messageDesc(file_k8s_io_api_rbac_v1_generated, 1);

/**
 * ClusterRoleBinding references a ClusterRole, but not contain it.  It can reference a ClusterRole in the global namespace,
 * and adds who information via Subject.
 *
 * @generated from message k8s.io.api.rbac.v1.ClusterRoleBinding
 */
export type ClusterRoleBinding = Message<"k8s.io.api.rbac.v1.ClusterRoleBinding"> & {
  /**
   * Standard object's metadata.
   * +optional
   *
   * @generated from field: optional k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta metadata = 1;
   */
  metadata?: ObjectMeta;

  /**
   * Subjects holds references to the objects the role applies to.
   * +optional
   * +listType=atomic
   *
   * @generated from field: repeated k8s.io.api.rbac.v1.Subject subjects = 2;
   */
  subjects: Subject[];

  /**
   * RoleRef can only reference a ClusterRole in the global namespace.
   * If the RoleRef cannot be resolved, the Authorizer must return an error.
   * This field is immutable.
   *
   * @generated from field: optional k8s.io.api.rbac.v1.RoleRef roleRef = 3;
   */
  roleRef?: RoleRef;
};

/**
 * Describes the message k8s.io.api.rbac.v1.ClusterRoleBinding.
 * Use `create(ClusterRoleBindingSchema)` to create a new message.
 */
export const ClusterRoleBindingSchema: GenMessage<ClusterRoleBinding> = /*@__PURE__*/
  messageDesc(file_k8s_io_api_rbac_v1_generated, 2);

/**
 * ClusterRoleBindingList is a collection of ClusterRoleBindings
 *
 * @generated from message k8s.io.api.rbac.v1.ClusterRoleBindingList
 */
export type ClusterRoleBindingList = Message<"k8s.io.api.rbac.v1.ClusterRoleBindingList"> & {
  /**
   * Standard object's metadata.
   * +optional
   *
   * @generated from field: optional k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta metadata = 1;
   */
  metadata?: ListMeta;

  /**
   * Items is a list of ClusterRoleBindings
   *
   * @generated from field: repeated k8s.io.api.rbac.v1.ClusterRoleBinding items = 2;
   */
  items: ClusterRoleBinding[];
};

/**
 * Describes the message k8s.io.api.rbac.v1.ClusterRoleBindingList.
 * Use `create(ClusterRoleBindingListSchema)` to create a new message.
 */
export const ClusterRoleBindingListSchema: GenMessage<ClusterRoleBindingList> = /*@__PURE__*/
  messageDesc(file_k8s_io_api_rbac_v1_generated, 3);

/**
 * ClusterRoleList is a collection of ClusterRoles
 *
 * @generated from message k8s.io.api.rbac.v1.ClusterRoleList
 */
export type ClusterRoleList = Message<"k8s.io.api.rbac.v1.ClusterRoleList"> & {
  /**
   * Standard object's metadata.
   * +optional
   *
   * @generated from field: optional k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta metadata = 1;
   */
  metadata?: ListMeta;

  /**
   * Items is a list of ClusterRoles
   *
   * @generated from field: repeated k8s.io.api.rbac.v1.ClusterRole items = 2;
   */
  items: ClusterRole[];
};

/**
 * Describes the message k8s.io.api.rbac.v1.ClusterRoleList.
 * Use `create(ClusterRoleListSchema)` to create a new message.
 */
export const ClusterRoleListSchema: GenMessage<ClusterRoleList> = /*@__PURE__*/
  messageDesc(file_k8s_io_api_rbac_v1_generated, 4);

/**
 * PolicyRule holds information that describes a policy rule, but does not contain information
 * about who the rule applies to or which namespace the rule applies to.
 *
 * @generated from message k8s.io.api.rbac.v1.PolicyRule
 */
export type PolicyRule = Message<"k8s.io.api.rbac.v1.PolicyRule"> & {
  /**
   * Verbs is a list of Verbs that apply to ALL the ResourceKinds contained in this rule. '*' represents all verbs.
   * +listType=atomic
   *
   * @generated from field: repeated string verbs = 1;
   */
  verbs: string[];

  /**
   * APIGroups is the name of the APIGroup that contains the resources.  If multiple API groups are specified, any action requested against one of
   * the enumerated resources in any API group will be allowed. "" represents the core API group and "*" represents all API groups.
   * +optional
   * +listType=atomic
   *
   * @generated from field: repeated string apiGroups = 2;
   */
  apiGroups: string[];

  /**
   * Resources is a list of resources this rule applies to. '*' represents all resources.
   * +optional
   * +listType=atomic
   *
   * @generated from field: repeated string resources = 3;
   */
  resources: string[];

  /**
   * ResourceNames is an optional white list of names that the rule applies to.  An empty set means that everything is allowed.
   * +optional
   * +listType=atomic
   *
   * @generated from field: repeated string resourceNames = 4;
   */
  resourceNames: string[];

  /**
   * NonResourceURLs is a set of partial urls that a user should have access to.  *s are allowed, but only as the full, final step in the path
   * Since non-resource URLs are not namespaced, this field is only applicable for ClusterRoles referenced from a ClusterRoleBinding.
   * Rules can either apply to API resources (such as "pods" or "secrets") or non-resource URL paths (such as "/api"),  but not both.
   * +optional
   * +listType=atomic
   *
   * @generated from field: repeated string nonResourceURLs = 5;
   */
  nonResourceURLs: string[];
};

/**
 * Describes the message k8s.io.api.rbac.v1.PolicyRule.
 * Use `create(PolicyRuleSchema)` to create a new message.
 */
export const PolicyRuleSchema: GenMessage<PolicyRule> = /*@__PURE__*/
  messageDesc(file_k8s_io_api_rbac_v1_generated, 5);

/**
 * Role is a namespaced, logical grouping of PolicyRules that can be referenced as a unit by a RoleBinding.
 *
 * @generated from message k8s.io.api.rbac.v1.Role
 */
export type Role = Message<"k8s.io.api.rbac.v1.Role"> & {
  /**
   * Standard object's metadata.
   * +optional
   *
   * @generated from field: optional k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta metadata = 1;
   */
  metadata?: ObjectMeta;

  /**
   * Rules holds all the PolicyRules for this Role
   * +optional
   * +listType=atomic
   *
   * @generated from field: repeated k8s.io.api.rbac.v1.PolicyRule rules = 2;
   */
  rules: PolicyRule[];
};

/**
 * Describes the message k8s.io.api.rbac.v1.Role.
 * Use `create(RoleSchema)` to create a new message.
 */
export const RoleSchema: GenMessage<Role> = /*@__PURE__*/
  messageDesc(file_k8s_io_api_rbac_v1_generated, 6);

/**
 * RoleBinding references a role, but does not contain it.  It can reference a Role in the same namespace or a ClusterRole in the global namespace.
 * It adds who information via Subjects and namespace information by which namespace it exists in.  RoleBindings in a given
 * namespace only have effect in that namespace.
 *
 * @generated from message k8s.io.api.rbac.v1.RoleBinding
 */
export type RoleBinding = Message<"k8s.io.api.rbac.v1.RoleBinding"> & {
  /**
   * Standard object's metadata.
   * +optional
   *
   * @generated from field: optional k8s.io.apimachinery.pkg.apis.meta.v1.ObjectMeta metadata = 1;
   */
  metadata?: ObjectMeta;

  /**
   * Subjects holds references to the objects the role applies to.
   * +optional
   * +listType=atomic
   *
   * @generated from field: repeated k8s.io.api.rbac.v1.Subject subjects = 2;
   */
  subjects: Subject[];

  /**
   * RoleRef can reference a Role in the current namespace or a ClusterRole in the global namespace.
   * If the RoleRef cannot be resolved, the Authorizer must return an error.
   * This field is immutable.
   *
   * @generated from field: optional k8s.io.api.rbac.v1.RoleRef roleRef = 3;
   */
  roleRef?: RoleRef;
};

/**
 * Describes the message k8s.io.api.rbac.v1.RoleBinding.
 * Use `create(RoleBindingSchema)` to create a new message.
 */
export const RoleBindingSchema: GenMessage<RoleBinding> = /*@__PURE__*/
  messageDesc(file_k8s_io_api_rbac_v1_generated, 7);

/**
 * RoleBindingList is a collection of RoleBindings
 *
 * @generated from message k8s.io.api.rbac.v1.RoleBindingList
 */
export type RoleBindingList = Message<"k8s.io.api.rbac.v1.RoleBindingList"> & {
  /**
   * Standard object's metadata.
   * +optional
   *
   * @generated from field: optional k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta metadata = 1;
   */
  metadata?: ListMeta;

  /**
   * Items is a list of RoleBindings
   *
   * @generated from field: repeated k8s.io.api.rbac.v1.RoleBinding items = 2;
   */
  items: RoleBinding[];
};

/**
 * Describes the message k8s.io.api.rbac.v1.RoleBindingList.
 * Use `create(RoleBindingListSchema)` to create a new message.
 */
export const RoleBindingListSchema: GenMessage<RoleBindingList> = /*@__PURE__*/
  messageDesc(file_k8s_io_api_rbac_v1_generated, 8);

/**
 * RoleList is a collection of Roles
 *
 * @generated from message k8s.io.api.rbac.v1.RoleList
 */
export type RoleList = Message<"k8s.io.api.rbac.v1.RoleList"> & {
  /**
   * Standard object's metadata.
   * +optional
   *
   * @generated from field: optional k8s.io.apimachinery.pkg.apis.meta.v1.ListMeta metadata = 1;
   */
  metadata?: ListMeta;

  /**
   * Items is a list of Roles
   *
   * @generated from field: repeated k8s.io.api.rbac.v1.Role items = 2;
   */
  items: Role[];
};

/**
 * Describes the message k8s.io.api.rbac.v1.RoleList.
 * Use `create(RoleListSchema)` to create a new message.
 */
export const RoleListSchema: GenMessage<RoleList> = /*@__PURE__*/
  messageDesc(file_k8s_io_api_rbac_v1_generated, 9);

/**
 * RoleRef contains information that points to the role being used
 * +structType=atomic
 *
 * @generated from message k8s.io.api.rbac.v1.RoleRef
 */
export type RoleRef = Message<"k8s.io.api.rbac.v1.RoleRef"> & {
  /**
   * APIGroup is the group for the resource being referenced
   *
   * @generated from field: optional string apiGroup = 1;
   */
  apiGroup: string;

  /**
   * Kind is the type of resource being referenced
   *
   * @generated from field: optional string kind = 2;
   */
  kind: string;

  /**
   * Name is the name of resource being referenced
   *
   * @generated from field: optional string name = 3;
   */
  name: string;
};

/**
 * Describes the message k8s.io.api.rbac.v1.RoleRef.
 * Use `create(RoleRefSchema)` to create a new message.
 */
export const RoleRefSchema: GenMessage<RoleRef> = /*@__PURE__*/
  messageDesc(file_k8s_io_api_rbac_v1_generated, 10);

/**
 * Subject contains a reference to the object or user identities a role binding applies to.  This can either hold a direct API object reference,
 * or a value for non-objects such as user and group names.
 * +structType=atomic
 *
 * @generated from message k8s.io.api.rbac.v1.Subject
 */
export type Subject = Message<"k8s.io.api.rbac.v1.Subject"> & {
  /**
   * Kind of object being referenced. Values defined by this API group are "User", "Group", and "ServiceAccount".
   * If the Authorizer does not recognized the kind value, the Authorizer should report an error.
   *
   * @generated from field: optional string kind = 1;
   */
  kind: string;

  /**
   * APIGroup holds the API group of the referenced subject.
   * Defaults to "" for ServiceAccount subjects.
   * Defaults to "rbac.authorization.k8s.io" for User and Group subjects.
   * +optional
   *
   * @generated from field: optional string apiGroup = 2;
   */
  apiGroup: string;

  /**
   * Name of the object being referenced.
   *
   * @generated from field: optional string name = 3;
   */
  name: string;

  /**
   * Namespace of the referenced object.  If the object kind is non-namespace, such as "User" or "Group", and this value is not empty
   * the Authorizer should report an error.
   * +optional
   *
   * @generated from field: optional string namespace = 4;
   */
  namespace: string;
};

/**
 * Describes the message k8s.io.api.rbac.v1.Subject.
 * Use `create(SubjectSchema)` to create a new message.
 */
export const SubjectSchema: GenMessage<Subject> = /*@__PURE__*/
  messageDesc(file_k8s_io_api_rbac_v1_generated, 11);

