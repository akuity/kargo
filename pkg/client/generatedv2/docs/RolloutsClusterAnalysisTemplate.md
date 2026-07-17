# RolloutsClusterAnalysisTemplate

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ApiVersion** | Pointer to **string** | APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources +optional | [optional] 
**Kind** | Pointer to **string** | Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds +optional | [optional] 
**Metadata** | Pointer to [**V1ObjectMeta**](V1ObjectMeta.md) |  | [optional] 
**Spec** | Pointer to [**RolloutsAnalysisTemplateSpec**](RolloutsAnalysisTemplateSpec.md) |  | [optional] 

## Methods

### NewRolloutsClusterAnalysisTemplate

`func NewRolloutsClusterAnalysisTemplate() *RolloutsClusterAnalysisTemplate`

NewRolloutsClusterAnalysisTemplate instantiates a new RolloutsClusterAnalysisTemplate object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRolloutsClusterAnalysisTemplateWithDefaults

`func NewRolloutsClusterAnalysisTemplateWithDefaults() *RolloutsClusterAnalysisTemplate`

NewRolloutsClusterAnalysisTemplateWithDefaults instantiates a new RolloutsClusterAnalysisTemplate object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetApiVersion

`func (o *RolloutsClusterAnalysisTemplate) GetApiVersion() string`

GetApiVersion returns the ApiVersion field if non-nil, zero value otherwise.

### GetApiVersionOk

`func (o *RolloutsClusterAnalysisTemplate) GetApiVersionOk() (*string, bool)`

GetApiVersionOk returns a tuple with the ApiVersion field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetApiVersion

`func (o *RolloutsClusterAnalysisTemplate) SetApiVersion(v string)`

SetApiVersion sets ApiVersion field to given value.

### HasApiVersion

`func (o *RolloutsClusterAnalysisTemplate) HasApiVersion() bool`

HasApiVersion returns a boolean if a field has been set.

### GetKind

`func (o *RolloutsClusterAnalysisTemplate) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *RolloutsClusterAnalysisTemplate) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *RolloutsClusterAnalysisTemplate) SetKind(v string)`

SetKind sets Kind field to given value.

### HasKind

`func (o *RolloutsClusterAnalysisTemplate) HasKind() bool`

HasKind returns a boolean if a field has been set.

### GetMetadata

`func (o *RolloutsClusterAnalysisTemplate) GetMetadata() V1ObjectMeta`

GetMetadata returns the Metadata field if non-nil, zero value otherwise.

### GetMetadataOk

`func (o *RolloutsClusterAnalysisTemplate) GetMetadataOk() (*V1ObjectMeta, bool)`

GetMetadataOk returns a tuple with the Metadata field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMetadata

`func (o *RolloutsClusterAnalysisTemplate) SetMetadata(v V1ObjectMeta)`

SetMetadata sets Metadata field to given value.

### HasMetadata

`func (o *RolloutsClusterAnalysisTemplate) HasMetadata() bool`

HasMetadata returns a boolean if a field has been set.

### GetSpec

`func (o *RolloutsClusterAnalysisTemplate) GetSpec() RolloutsAnalysisTemplateSpec`

GetSpec returns the Spec field if non-nil, zero value otherwise.

### GetSpecOk

`func (o *RolloutsClusterAnalysisTemplate) GetSpecOk() (*RolloutsAnalysisTemplateSpec, bool)`

GetSpecOk returns a tuple with the Spec field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSpec

`func (o *RolloutsClusterAnalysisTemplate) SetSpec(v RolloutsAnalysisTemplateSpec)`

SetSpec sets Spec field to given value.

### HasSpec

`func (o *RolloutsClusterAnalysisTemplate) HasSpec() bool`

HasSpec returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


