# RolloutsAnalysisRun

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ApiVersion** | Pointer to **string** | APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources +optional | [optional] 
**Kind** | Pointer to **string** | Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds +optional | [optional] 
**Metadata** | Pointer to [**V1ObjectMeta**](V1ObjectMeta.md) |  | [optional] 
**Spec** | Pointer to [**RolloutsAnalysisRunSpec**](RolloutsAnalysisRunSpec.md) |  | [optional] 
**Status** | Pointer to [**RolloutsAnalysisRunStatus**](RolloutsAnalysisRunStatus.md) |  | [optional] 

## Methods

### NewRolloutsAnalysisRun

`func NewRolloutsAnalysisRun() *RolloutsAnalysisRun`

NewRolloutsAnalysisRun instantiates a new RolloutsAnalysisRun object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRolloutsAnalysisRunWithDefaults

`func NewRolloutsAnalysisRunWithDefaults() *RolloutsAnalysisRun`

NewRolloutsAnalysisRunWithDefaults instantiates a new RolloutsAnalysisRun object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetApiVersion

`func (o *RolloutsAnalysisRun) GetApiVersion() string`

GetApiVersion returns the ApiVersion field if non-nil, zero value otherwise.

### GetApiVersionOk

`func (o *RolloutsAnalysisRun) GetApiVersionOk() (*string, bool)`

GetApiVersionOk returns a tuple with the ApiVersion field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetApiVersion

`func (o *RolloutsAnalysisRun) SetApiVersion(v string)`

SetApiVersion sets ApiVersion field to given value.

### HasApiVersion

`func (o *RolloutsAnalysisRun) HasApiVersion() bool`

HasApiVersion returns a boolean if a field has been set.

### GetKind

`func (o *RolloutsAnalysisRun) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *RolloutsAnalysisRun) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *RolloutsAnalysisRun) SetKind(v string)`

SetKind sets Kind field to given value.

### HasKind

`func (o *RolloutsAnalysisRun) HasKind() bool`

HasKind returns a boolean if a field has been set.

### GetMetadata

`func (o *RolloutsAnalysisRun) GetMetadata() V1ObjectMeta`

GetMetadata returns the Metadata field if non-nil, zero value otherwise.

### GetMetadataOk

`func (o *RolloutsAnalysisRun) GetMetadataOk() (*V1ObjectMeta, bool)`

GetMetadataOk returns a tuple with the Metadata field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMetadata

`func (o *RolloutsAnalysisRun) SetMetadata(v V1ObjectMeta)`

SetMetadata sets Metadata field to given value.

### HasMetadata

`func (o *RolloutsAnalysisRun) HasMetadata() bool`

HasMetadata returns a boolean if a field has been set.

### GetSpec

`func (o *RolloutsAnalysisRun) GetSpec() RolloutsAnalysisRunSpec`

GetSpec returns the Spec field if non-nil, zero value otherwise.

### GetSpecOk

`func (o *RolloutsAnalysisRun) GetSpecOk() (*RolloutsAnalysisRunSpec, bool)`

GetSpecOk returns a tuple with the Spec field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSpec

`func (o *RolloutsAnalysisRun) SetSpec(v RolloutsAnalysisRunSpec)`

SetSpec sets Spec field to given value.

### HasSpec

`func (o *RolloutsAnalysisRun) HasSpec() bool`

HasSpec returns a boolean if a field has been set.

### GetStatus

`func (o *RolloutsAnalysisRun) GetStatus() RolloutsAnalysisRunStatus`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *RolloutsAnalysisRun) GetStatusOk() (*RolloutsAnalysisRunStatus, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *RolloutsAnalysisRun) SetStatus(v RolloutsAnalysisRunStatus)`

SetStatus sets Status field to given value.

### HasStatus

`func (o *RolloutsAnalysisRun) HasStatus() bool`

HasStatus returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


