# RolloutsAnalysisTemplate

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ApiVersion** | Pointer to **string** | APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources +optional | [optional] 
**Kind** | Pointer to **string** | Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds +optional | [optional] 
**Metadata** | Pointer to [**V1ObjectMeta**](V1ObjectMeta.md) |  | [optional] 
**Spec** | Pointer to [**RolloutsAnalysisTemplateSpec**](RolloutsAnalysisTemplateSpec.md) |  | [optional] 

## Methods

### NewRolloutsAnalysisTemplate

`func NewRolloutsAnalysisTemplate() *RolloutsAnalysisTemplate`

NewRolloutsAnalysisTemplate instantiates a new RolloutsAnalysisTemplate object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRolloutsAnalysisTemplateWithDefaults

`func NewRolloutsAnalysisTemplateWithDefaults() *RolloutsAnalysisTemplate`

NewRolloutsAnalysisTemplateWithDefaults instantiates a new RolloutsAnalysisTemplate object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetApiVersion

`func (o *RolloutsAnalysisTemplate) GetApiVersion() string`

GetApiVersion returns the ApiVersion field if non-nil, zero value otherwise.

### GetApiVersionOk

`func (o *RolloutsAnalysisTemplate) GetApiVersionOk() (*string, bool)`

GetApiVersionOk returns a tuple with the ApiVersion field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetApiVersion

`func (o *RolloutsAnalysisTemplate) SetApiVersion(v string)`

SetApiVersion sets ApiVersion field to given value.

### HasApiVersion

`func (o *RolloutsAnalysisTemplate) HasApiVersion() bool`

HasApiVersion returns a boolean if a field has been set.

### GetKind

`func (o *RolloutsAnalysisTemplate) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *RolloutsAnalysisTemplate) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *RolloutsAnalysisTemplate) SetKind(v string)`

SetKind sets Kind field to given value.

### HasKind

`func (o *RolloutsAnalysisTemplate) HasKind() bool`

HasKind returns a boolean if a field has been set.

### GetMetadata

`func (o *RolloutsAnalysisTemplate) GetMetadata() V1ObjectMeta`

GetMetadata returns the Metadata field if non-nil, zero value otherwise.

### GetMetadataOk

`func (o *RolloutsAnalysisTemplate) GetMetadataOk() (*V1ObjectMeta, bool)`

GetMetadataOk returns a tuple with the Metadata field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMetadata

`func (o *RolloutsAnalysisTemplate) SetMetadata(v V1ObjectMeta)`

SetMetadata sets Metadata field to given value.

### HasMetadata

`func (o *RolloutsAnalysisTemplate) HasMetadata() bool`

HasMetadata returns a boolean if a field has been set.

### GetSpec

`func (o *RolloutsAnalysisTemplate) GetSpec() RolloutsAnalysisTemplateSpec`

GetSpec returns the Spec field if non-nil, zero value otherwise.

### GetSpecOk

`func (o *RolloutsAnalysisTemplate) GetSpecOk() (*RolloutsAnalysisTemplateSpec, bool)`

GetSpecOk returns a tuple with the Spec field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSpec

`func (o *RolloutsAnalysisTemplate) SetSpec(v RolloutsAnalysisTemplateSpec)`

SetSpec sets Spec field to given value.

### HasSpec

`func (o *RolloutsAnalysisTemplate) HasSpec() bool`

HasSpec returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


