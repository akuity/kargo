# RolloutsClusterAnalysisTemplateList

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ApiVersion** | Pointer to **string** | APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources +optional | [optional] 
**Items** | Pointer to [**[]RolloutsClusterAnalysisTemplate**](RolloutsClusterAnalysisTemplate.md) |  | [optional] 
**Kind** | Pointer to **string** | Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds +optional | [optional] 
**Metadata** | Pointer to [**V1ListMeta**](V1ListMeta.md) |  | [optional] 

## Methods

### NewRolloutsClusterAnalysisTemplateList

`func NewRolloutsClusterAnalysisTemplateList() *RolloutsClusterAnalysisTemplateList`

NewRolloutsClusterAnalysisTemplateList instantiates a new RolloutsClusterAnalysisTemplateList object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRolloutsClusterAnalysisTemplateListWithDefaults

`func NewRolloutsClusterAnalysisTemplateListWithDefaults() *RolloutsClusterAnalysisTemplateList`

NewRolloutsClusterAnalysisTemplateListWithDefaults instantiates a new RolloutsClusterAnalysisTemplateList object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetApiVersion

`func (o *RolloutsClusterAnalysisTemplateList) GetApiVersion() string`

GetApiVersion returns the ApiVersion field if non-nil, zero value otherwise.

### GetApiVersionOk

`func (o *RolloutsClusterAnalysisTemplateList) GetApiVersionOk() (*string, bool)`

GetApiVersionOk returns a tuple with the ApiVersion field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetApiVersion

`func (o *RolloutsClusterAnalysisTemplateList) SetApiVersion(v string)`

SetApiVersion sets ApiVersion field to given value.

### HasApiVersion

`func (o *RolloutsClusterAnalysisTemplateList) HasApiVersion() bool`

HasApiVersion returns a boolean if a field has been set.

### GetItems

`func (o *RolloutsClusterAnalysisTemplateList) GetItems() []RolloutsClusterAnalysisTemplate`

GetItems returns the Items field if non-nil, zero value otherwise.

### GetItemsOk

`func (o *RolloutsClusterAnalysisTemplateList) GetItemsOk() (*[]RolloutsClusterAnalysisTemplate, bool)`

GetItemsOk returns a tuple with the Items field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetItems

`func (o *RolloutsClusterAnalysisTemplateList) SetItems(v []RolloutsClusterAnalysisTemplate)`

SetItems sets Items field to given value.

### HasItems

`func (o *RolloutsClusterAnalysisTemplateList) HasItems() bool`

HasItems returns a boolean if a field has been set.

### GetKind

`func (o *RolloutsClusterAnalysisTemplateList) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *RolloutsClusterAnalysisTemplateList) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *RolloutsClusterAnalysisTemplateList) SetKind(v string)`

SetKind sets Kind field to given value.

### HasKind

`func (o *RolloutsClusterAnalysisTemplateList) HasKind() bool`

HasKind returns a boolean if a field has been set.

### GetMetadata

`func (o *RolloutsClusterAnalysisTemplateList) GetMetadata() V1ListMeta`

GetMetadata returns the Metadata field if non-nil, zero value otherwise.

### GetMetadataOk

`func (o *RolloutsClusterAnalysisTemplateList) GetMetadataOk() (*V1ListMeta, bool)`

GetMetadataOk returns a tuple with the Metadata field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMetadata

`func (o *RolloutsClusterAnalysisTemplateList) SetMetadata(v V1ListMeta)`

SetMetadata sets Metadata field to given value.

### HasMetadata

`func (o *RolloutsClusterAnalysisTemplateList) HasMetadata() bool`

HasMetadata returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


