# V1TypedLocalObjectReference

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ApiGroup** | Pointer to **string** | APIGroup is the group for the resource being referenced. If APIGroup is not specified, the specified Kind must be in the core API group. For any other third-party types, APIGroup is required. +optional | [optional] 
**Kind** | Pointer to **string** | Kind is the type of resource being referenced | [optional] 
**Name** | Pointer to **string** | Name is the name of resource being referenced | [optional] 

## Methods

### NewV1TypedLocalObjectReference

`func NewV1TypedLocalObjectReference() *V1TypedLocalObjectReference`

NewV1TypedLocalObjectReference instantiates a new V1TypedLocalObjectReference object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1TypedLocalObjectReferenceWithDefaults

`func NewV1TypedLocalObjectReferenceWithDefaults() *V1TypedLocalObjectReference`

NewV1TypedLocalObjectReferenceWithDefaults instantiates a new V1TypedLocalObjectReference object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetApiGroup

`func (o *V1TypedLocalObjectReference) GetApiGroup() string`

GetApiGroup returns the ApiGroup field if non-nil, zero value otherwise.

### GetApiGroupOk

`func (o *V1TypedLocalObjectReference) GetApiGroupOk() (*string, bool)`

GetApiGroupOk returns a tuple with the ApiGroup field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetApiGroup

`func (o *V1TypedLocalObjectReference) SetApiGroup(v string)`

SetApiGroup sets ApiGroup field to given value.

### HasApiGroup

`func (o *V1TypedLocalObjectReference) HasApiGroup() bool`

HasApiGroup returns a boolean if a field has been set.

### GetKind

`func (o *V1TypedLocalObjectReference) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *V1TypedLocalObjectReference) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *V1TypedLocalObjectReference) SetKind(v string)`

SetKind sets Kind field to given value.

### HasKind

`func (o *V1TypedLocalObjectReference) HasKind() bool`

HasKind returns a boolean if a field has been set.

### GetName

`func (o *V1TypedLocalObjectReference) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *V1TypedLocalObjectReference) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *V1TypedLocalObjectReference) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *V1TypedLocalObjectReference) HasName() bool`

HasName returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


