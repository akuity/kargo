# V1OwnerReference

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ApiVersion** | Pointer to **string** | API version of the referent. | [optional] 
**BlockOwnerDeletion** | Pointer to **bool** | If true, AND if the owner has the \&quot;foregroundDeletion\&quot; finalizer, then the owner cannot be deleted from the key-value store until this reference is removed. See https://kubernetes.io/docs/concepts/architecture/garbage-collection/#foreground-deletion for how the garbage collector interacts with this field and enforces the foreground deletion. Defaults to false. To set this field, a user needs \&quot;delete\&quot; permission of the owner, otherwise 422 (Unprocessable Entity) will be returned. +optional | [optional] 
**Controller** | Pointer to **bool** | If true, this reference points to the managing controller. +optional | [optional] 
**Kind** | Pointer to **string** | Kind of the referent. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds | [optional] 
**Name** | Pointer to **string** | Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#names | [optional] 
**Uid** | Pointer to **string** | UID of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#uids | [optional] 

## Methods

### NewV1OwnerReference

`func NewV1OwnerReference() *V1OwnerReference`

NewV1OwnerReference instantiates a new V1OwnerReference object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1OwnerReferenceWithDefaults

`func NewV1OwnerReferenceWithDefaults() *V1OwnerReference`

NewV1OwnerReferenceWithDefaults instantiates a new V1OwnerReference object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetApiVersion

`func (o *V1OwnerReference) GetApiVersion() string`

GetApiVersion returns the ApiVersion field if non-nil, zero value otherwise.

### GetApiVersionOk

`func (o *V1OwnerReference) GetApiVersionOk() (*string, bool)`

GetApiVersionOk returns a tuple with the ApiVersion field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetApiVersion

`func (o *V1OwnerReference) SetApiVersion(v string)`

SetApiVersion sets ApiVersion field to given value.

### HasApiVersion

`func (o *V1OwnerReference) HasApiVersion() bool`

HasApiVersion returns a boolean if a field has been set.

### GetBlockOwnerDeletion

`func (o *V1OwnerReference) GetBlockOwnerDeletion() bool`

GetBlockOwnerDeletion returns the BlockOwnerDeletion field if non-nil, zero value otherwise.

### GetBlockOwnerDeletionOk

`func (o *V1OwnerReference) GetBlockOwnerDeletionOk() (*bool, bool)`

GetBlockOwnerDeletionOk returns a tuple with the BlockOwnerDeletion field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBlockOwnerDeletion

`func (o *V1OwnerReference) SetBlockOwnerDeletion(v bool)`

SetBlockOwnerDeletion sets BlockOwnerDeletion field to given value.

### HasBlockOwnerDeletion

`func (o *V1OwnerReference) HasBlockOwnerDeletion() bool`

HasBlockOwnerDeletion returns a boolean if a field has been set.

### GetController

`func (o *V1OwnerReference) GetController() bool`

GetController returns the Controller field if non-nil, zero value otherwise.

### GetControllerOk

`func (o *V1OwnerReference) GetControllerOk() (*bool, bool)`

GetControllerOk returns a tuple with the Controller field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetController

`func (o *V1OwnerReference) SetController(v bool)`

SetController sets Controller field to given value.

### HasController

`func (o *V1OwnerReference) HasController() bool`

HasController returns a boolean if a field has been set.

### GetKind

`func (o *V1OwnerReference) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *V1OwnerReference) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *V1OwnerReference) SetKind(v string)`

SetKind sets Kind field to given value.

### HasKind

`func (o *V1OwnerReference) HasKind() bool`

HasKind returns a boolean if a field has been set.

### GetName

`func (o *V1OwnerReference) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *V1OwnerReference) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *V1OwnerReference) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *V1OwnerReference) HasName() bool`

HasName returns a boolean if a field has been set.

### GetUid

`func (o *V1OwnerReference) GetUid() string`

GetUid returns the Uid field if non-nil, zero value otherwise.

### GetUidOk

`func (o *V1OwnerReference) GetUidOk() (*string, bool)`

GetUidOk returns a tuple with the Uid field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUid

`func (o *V1OwnerReference) SetUid(v string)`

SetUid sets Uid field to given value.

### HasUid

`func (o *V1OwnerReference) HasUid() bool`

HasUid returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


