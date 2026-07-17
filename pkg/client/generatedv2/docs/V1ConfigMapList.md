# V1ConfigMapList

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ApiVersion** | Pointer to **string** | APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources +optional | [optional] 
**Items** | Pointer to [**[]V1ConfigMap**](V1ConfigMap.md) | Items is the list of ConfigMaps. | [optional] 
**Kind** | Pointer to **string** | Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds +optional | [optional] 
**Metadata** | Pointer to [**V1ListMeta**](V1ListMeta.md) | More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata +optional | [optional] 

## Methods

### NewV1ConfigMapList

`func NewV1ConfigMapList() *V1ConfigMapList`

NewV1ConfigMapList instantiates a new V1ConfigMapList object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1ConfigMapListWithDefaults

`func NewV1ConfigMapListWithDefaults() *V1ConfigMapList`

NewV1ConfigMapListWithDefaults instantiates a new V1ConfigMapList object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetApiVersion

`func (o *V1ConfigMapList) GetApiVersion() string`

GetApiVersion returns the ApiVersion field if non-nil, zero value otherwise.

### GetApiVersionOk

`func (o *V1ConfigMapList) GetApiVersionOk() (*string, bool)`

GetApiVersionOk returns a tuple with the ApiVersion field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetApiVersion

`func (o *V1ConfigMapList) SetApiVersion(v string)`

SetApiVersion sets ApiVersion field to given value.

### HasApiVersion

`func (o *V1ConfigMapList) HasApiVersion() bool`

HasApiVersion returns a boolean if a field has been set.

### GetItems

`func (o *V1ConfigMapList) GetItems() []V1ConfigMap`

GetItems returns the Items field if non-nil, zero value otherwise.

### GetItemsOk

`func (o *V1ConfigMapList) GetItemsOk() (*[]V1ConfigMap, bool)`

GetItemsOk returns a tuple with the Items field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetItems

`func (o *V1ConfigMapList) SetItems(v []V1ConfigMap)`

SetItems sets Items field to given value.

### HasItems

`func (o *V1ConfigMapList) HasItems() bool`

HasItems returns a boolean if a field has been set.

### GetKind

`func (o *V1ConfigMapList) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *V1ConfigMapList) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *V1ConfigMapList) SetKind(v string)`

SetKind sets Kind field to given value.

### HasKind

`func (o *V1ConfigMapList) HasKind() bool`

HasKind returns a boolean if a field has been set.

### GetMetadata

`func (o *V1ConfigMapList) GetMetadata() V1ListMeta`

GetMetadata returns the Metadata field if non-nil, zero value otherwise.

### GetMetadataOk

`func (o *V1ConfigMapList) GetMetadataOk() (*V1ListMeta, bool)`

GetMetadataOk returns a tuple with the Metadata field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMetadata

`func (o *V1ConfigMapList) SetMetadata(v V1ListMeta)`

SetMetadata sets Metadata field to given value.

### HasMetadata

`func (o *V1ConfigMapList) HasMetadata() bool`

HasMetadata returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


