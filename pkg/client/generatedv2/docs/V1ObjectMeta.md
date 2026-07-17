# V1ObjectMeta

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Annotations** | Pointer to **map[string]string** | Annotations is an unstructured key value map stored with a resource that may be set by external tools to store and retrieve arbitrary metadata. They are not queryable and should be preserved when modifying objects. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations +optional | [optional] 
**CreationTimestamp** | Pointer to **string** | CreationTimestamp is a timestamp representing the server time when this object was created. It is not guaranteed to be set in happens-before order across separate operations. Clients may not set this value. It is represented in RFC3339 form and is in UTC.  Populated by the system. Read-only. Null for lists. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional | [optional] 
**DeletionGracePeriodSeconds** | Pointer to **int32** | Number of seconds allowed for this object to gracefully terminate before it will be removed from the system. Only set when deletionTimestamp is also set. May only be shortened. Read-only. +optional | [optional] 
**DeletionTimestamp** | Pointer to **string** | DeletionTimestamp is RFC 3339 date and time at which this resource will be deleted. This field is set by the server when a graceful deletion is requested by the user, and is not directly settable by a client. The resource is expected to be deleted (no longer visible from resource lists, and not reachable by name) after the time in this field, once the finalizers list is empty. As long as the finalizers list contains items, deletion is blocked. Once the deletionTimestamp is set, this value may not be unset or be set further into the future, although it may be shortened or the resource may be deleted prior to this time. For example, a user may request that a pod is deleted in 30 seconds. The Kubelet will react by sending a graceful termination signal to the containers in the pod. After that 30 seconds, the Kubelet will send a hard termination signal (SIGKILL) to the container and after cleanup, remove the pod from the API. In the presence of network partitions, this object may still exist after this timestamp, until an administrator or automated process can determine the resource is fully terminated. If not set, graceful deletion of the object has not been requested.  Populated by the system when a graceful deletion is requested. Read-only. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional | [optional] 
**Finalizers** | Pointer to **[]string** | Must be empty before the object is deleted from the registry. Each entry is an identifier for the responsible component that will remove the entry from the list. If the deletionTimestamp of the object is non-nil, entries in this list can only be removed. Finalizers may be processed and removed in any order.  Order is NOT enforced because it introduces significant risk of stuck finalizers. finalizers is a shared field, any actor with permission can reorder it. If the finalizer list is processed in order, then this can lead to a situation in which the component responsible for the first finalizer in the list is waiting for a signal (field value, external system, or other) produced by a component responsible for a finalizer later in the list, resulting in a deadlock. Without enforced ordering finalizers are free to order amongst themselves and are not vulnerable to ordering changes in the list. +optional +patchStrategy&#x3D;merge +listType&#x3D;set | [optional] 
**GenerateName** | Pointer to **string** | GenerateName is an optional prefix, used by the server, to generate a unique name ONLY IF the Name field has not been provided. If this field is used, the name returned to the client will be different than the name passed. This value will also be combined with a unique suffix. The provided value has the same validation rules as the Name field, and may be truncated by the length of the suffix required to make the value unique on the server.  If this field is specified and the generated name exists, the server will return a 409.  Applied only if Name is not specified. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#idempotency +optional | [optional] 
**Generation** | Pointer to **int32** | A sequence number representing a specific generation of the desired state. Populated by the system. Read-only. +optional | [optional] 
**Labels** | Pointer to **map[string]string** | Map of string keys and values that can be used to organize and categorize (scope and select) objects. May match selectors of replication controllers and services. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels +optional | [optional] 
**ManagedFields** | Pointer to [**[]V1ManagedFieldsEntry**](V1ManagedFieldsEntry.md) | ManagedFields maps workflow-id and version to the set of fields that are managed by that workflow. This is mostly for internal housekeeping, and users typically shouldn&#39;t need to set or understand this field. A workflow can be the user&#39;s name, a controller&#39;s name, or the name of a specific apply path like \&quot;ci-cd\&quot;. The set of fields is always in the version that the workflow used when modifying the object.  +optional +listType&#x3D;atomic | [optional] 
**Name** | Pointer to **string** | Name must be unique within a namespace. Is required when creating resources, although some resources may allow a client to request the generation of an appropriate name automatically. Name is primarily intended for creation idempotence and configuration definition. Cannot be updated. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#names +optional | [optional] 
**Namespace** | Pointer to **string** | Namespace defines the space within which each name must be unique. An empty namespace is equivalent to the \&quot;default\&quot; namespace, but \&quot;default\&quot; is the canonical representation. Not all objects are required to be scoped to a namespace - the value of this field for those objects will be empty.  Must be a DNS_LABEL. Cannot be updated. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces +optional | [optional] 
**OwnerReferences** | Pointer to [**[]V1OwnerReference**](V1OwnerReference.md) | List of objects depended by this object. If ALL objects in the list have been deleted, this object will be garbage collected. If this object is managed by a controller, then an entry in this list will point to this controller, with the controller field set to true. There cannot be more than one managing controller. +optional +patchMergeKey&#x3D;uid +patchStrategy&#x3D;merge +listType&#x3D;map +listMapKey&#x3D;uid | [optional] 
**ResourceVersion** | Pointer to **string** | An opaque value that represents the internal version of this object that can be used by clients to determine when objects have changed. May be used for optimistic concurrency, change detection, and the watch operation on a resource or set of resources. Clients must treat these values as opaque and passed unmodified back to the server. They may only be valid for a particular resource or set of resources.  Populated by the system. Read-only. Value must be treated as opaque by clients and . More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency +optional | [optional] 
**SelfLink** | Pointer to **string** | Deprecated: selfLink is a legacy read-only field that is no longer populated by the system. +optional | [optional] 
**Uid** | Pointer to **string** | UID is the unique in time and space value for this object. It is typically generated by the server on successful creation of a resource and is not allowed to change on PUT operations.  Populated by the system. Read-only. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#uids +optional | [optional] 

## Methods

### NewV1ObjectMeta

`func NewV1ObjectMeta() *V1ObjectMeta`

NewV1ObjectMeta instantiates a new V1ObjectMeta object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1ObjectMetaWithDefaults

`func NewV1ObjectMetaWithDefaults() *V1ObjectMeta`

NewV1ObjectMetaWithDefaults instantiates a new V1ObjectMeta object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAnnotations

`func (o *V1ObjectMeta) GetAnnotations() map[string]string`

GetAnnotations returns the Annotations field if non-nil, zero value otherwise.

### GetAnnotationsOk

`func (o *V1ObjectMeta) GetAnnotationsOk() (*map[string]string, bool)`

GetAnnotationsOk returns a tuple with the Annotations field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAnnotations

`func (o *V1ObjectMeta) SetAnnotations(v map[string]string)`

SetAnnotations sets Annotations field to given value.

### HasAnnotations

`func (o *V1ObjectMeta) HasAnnotations() bool`

HasAnnotations returns a boolean if a field has been set.

### GetCreationTimestamp

`func (o *V1ObjectMeta) GetCreationTimestamp() string`

GetCreationTimestamp returns the CreationTimestamp field if non-nil, zero value otherwise.

### GetCreationTimestampOk

`func (o *V1ObjectMeta) GetCreationTimestampOk() (*string, bool)`

GetCreationTimestampOk returns a tuple with the CreationTimestamp field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreationTimestamp

`func (o *V1ObjectMeta) SetCreationTimestamp(v string)`

SetCreationTimestamp sets CreationTimestamp field to given value.

### HasCreationTimestamp

`func (o *V1ObjectMeta) HasCreationTimestamp() bool`

HasCreationTimestamp returns a boolean if a field has been set.

### GetDeletionGracePeriodSeconds

`func (o *V1ObjectMeta) GetDeletionGracePeriodSeconds() int32`

GetDeletionGracePeriodSeconds returns the DeletionGracePeriodSeconds field if non-nil, zero value otherwise.

### GetDeletionGracePeriodSecondsOk

`func (o *V1ObjectMeta) GetDeletionGracePeriodSecondsOk() (*int32, bool)`

GetDeletionGracePeriodSecondsOk returns a tuple with the DeletionGracePeriodSeconds field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDeletionGracePeriodSeconds

`func (o *V1ObjectMeta) SetDeletionGracePeriodSeconds(v int32)`

SetDeletionGracePeriodSeconds sets DeletionGracePeriodSeconds field to given value.

### HasDeletionGracePeriodSeconds

`func (o *V1ObjectMeta) HasDeletionGracePeriodSeconds() bool`

HasDeletionGracePeriodSeconds returns a boolean if a field has been set.

### GetDeletionTimestamp

`func (o *V1ObjectMeta) GetDeletionTimestamp() string`

GetDeletionTimestamp returns the DeletionTimestamp field if non-nil, zero value otherwise.

### GetDeletionTimestampOk

`func (o *V1ObjectMeta) GetDeletionTimestampOk() (*string, bool)`

GetDeletionTimestampOk returns a tuple with the DeletionTimestamp field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDeletionTimestamp

`func (o *V1ObjectMeta) SetDeletionTimestamp(v string)`

SetDeletionTimestamp sets DeletionTimestamp field to given value.

### HasDeletionTimestamp

`func (o *V1ObjectMeta) HasDeletionTimestamp() bool`

HasDeletionTimestamp returns a boolean if a field has been set.

### GetFinalizers

`func (o *V1ObjectMeta) GetFinalizers() []string`

GetFinalizers returns the Finalizers field if non-nil, zero value otherwise.

### GetFinalizersOk

`func (o *V1ObjectMeta) GetFinalizersOk() (*[]string, bool)`

GetFinalizersOk returns a tuple with the Finalizers field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFinalizers

`func (o *V1ObjectMeta) SetFinalizers(v []string)`

SetFinalizers sets Finalizers field to given value.

### HasFinalizers

`func (o *V1ObjectMeta) HasFinalizers() bool`

HasFinalizers returns a boolean if a field has been set.

### GetGenerateName

`func (o *V1ObjectMeta) GetGenerateName() string`

GetGenerateName returns the GenerateName field if non-nil, zero value otherwise.

### GetGenerateNameOk

`func (o *V1ObjectMeta) GetGenerateNameOk() (*string, bool)`

GetGenerateNameOk returns a tuple with the GenerateName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetGenerateName

`func (o *V1ObjectMeta) SetGenerateName(v string)`

SetGenerateName sets GenerateName field to given value.

### HasGenerateName

`func (o *V1ObjectMeta) HasGenerateName() bool`

HasGenerateName returns a boolean if a field has been set.

### GetGeneration

`func (o *V1ObjectMeta) GetGeneration() int32`

GetGeneration returns the Generation field if non-nil, zero value otherwise.

### GetGenerationOk

`func (o *V1ObjectMeta) GetGenerationOk() (*int32, bool)`

GetGenerationOk returns a tuple with the Generation field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetGeneration

`func (o *V1ObjectMeta) SetGeneration(v int32)`

SetGeneration sets Generation field to given value.

### HasGeneration

`func (o *V1ObjectMeta) HasGeneration() bool`

HasGeneration returns a boolean if a field has been set.

### GetLabels

`func (o *V1ObjectMeta) GetLabels() map[string]string`

GetLabels returns the Labels field if non-nil, zero value otherwise.

### GetLabelsOk

`func (o *V1ObjectMeta) GetLabelsOk() (*map[string]string, bool)`

GetLabelsOk returns a tuple with the Labels field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLabels

`func (o *V1ObjectMeta) SetLabels(v map[string]string)`

SetLabels sets Labels field to given value.

### HasLabels

`func (o *V1ObjectMeta) HasLabels() bool`

HasLabels returns a boolean if a field has been set.

### GetManagedFields

`func (o *V1ObjectMeta) GetManagedFields() []V1ManagedFieldsEntry`

GetManagedFields returns the ManagedFields field if non-nil, zero value otherwise.

### GetManagedFieldsOk

`func (o *V1ObjectMeta) GetManagedFieldsOk() (*[]V1ManagedFieldsEntry, bool)`

GetManagedFieldsOk returns a tuple with the ManagedFields field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetManagedFields

`func (o *V1ObjectMeta) SetManagedFields(v []V1ManagedFieldsEntry)`

SetManagedFields sets ManagedFields field to given value.

### HasManagedFields

`func (o *V1ObjectMeta) HasManagedFields() bool`

HasManagedFields returns a boolean if a field has been set.

### GetName

`func (o *V1ObjectMeta) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *V1ObjectMeta) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *V1ObjectMeta) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *V1ObjectMeta) HasName() bool`

HasName returns a boolean if a field has been set.

### GetNamespace

`func (o *V1ObjectMeta) GetNamespace() string`

GetNamespace returns the Namespace field if non-nil, zero value otherwise.

### GetNamespaceOk

`func (o *V1ObjectMeta) GetNamespaceOk() (*string, bool)`

GetNamespaceOk returns a tuple with the Namespace field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNamespace

`func (o *V1ObjectMeta) SetNamespace(v string)`

SetNamespace sets Namespace field to given value.

### HasNamespace

`func (o *V1ObjectMeta) HasNamespace() bool`

HasNamespace returns a boolean if a field has been set.

### GetOwnerReferences

`func (o *V1ObjectMeta) GetOwnerReferences() []V1OwnerReference`

GetOwnerReferences returns the OwnerReferences field if non-nil, zero value otherwise.

### GetOwnerReferencesOk

`func (o *V1ObjectMeta) GetOwnerReferencesOk() (*[]V1OwnerReference, bool)`

GetOwnerReferencesOk returns a tuple with the OwnerReferences field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOwnerReferences

`func (o *V1ObjectMeta) SetOwnerReferences(v []V1OwnerReference)`

SetOwnerReferences sets OwnerReferences field to given value.

### HasOwnerReferences

`func (o *V1ObjectMeta) HasOwnerReferences() bool`

HasOwnerReferences returns a boolean if a field has been set.

### GetResourceVersion

`func (o *V1ObjectMeta) GetResourceVersion() string`

GetResourceVersion returns the ResourceVersion field if non-nil, zero value otherwise.

### GetResourceVersionOk

`func (o *V1ObjectMeta) GetResourceVersionOk() (*string, bool)`

GetResourceVersionOk returns a tuple with the ResourceVersion field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResourceVersion

`func (o *V1ObjectMeta) SetResourceVersion(v string)`

SetResourceVersion sets ResourceVersion field to given value.

### HasResourceVersion

`func (o *V1ObjectMeta) HasResourceVersion() bool`

HasResourceVersion returns a boolean if a field has been set.

### GetSelfLink

`func (o *V1ObjectMeta) GetSelfLink() string`

GetSelfLink returns the SelfLink field if non-nil, zero value otherwise.

### GetSelfLinkOk

`func (o *V1ObjectMeta) GetSelfLinkOk() (*string, bool)`

GetSelfLinkOk returns a tuple with the SelfLink field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSelfLink

`func (o *V1ObjectMeta) SetSelfLink(v string)`

SetSelfLink sets SelfLink field to given value.

### HasSelfLink

`func (o *V1ObjectMeta) HasSelfLink() bool`

HasSelfLink returns a boolean if a field has been set.

### GetUid

`func (o *V1ObjectMeta) GetUid() string`

GetUid returns the Uid field if non-nil, zero value otherwise.

### GetUidOk

`func (o *V1ObjectMeta) GetUidOk() (*string, bool)`

GetUidOk returns a tuple with the Uid field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUid

`func (o *V1ObjectMeta) SetUid(v string)`

SetUid sets Uid field to given value.

### HasUid

`func (o *V1ObjectMeta) HasUid() bool`

HasUid returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


