# RolloutsJobMetric

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Metadata** | Pointer to [**V1ObjectMeta**](V1ObjectMeta.md) |  | [optional] 
**Spec** | Pointer to [**V1JobSpec**](V1JobSpec.md) |  | [optional] 

## Methods

### NewRolloutsJobMetric

`func NewRolloutsJobMetric() *RolloutsJobMetric`

NewRolloutsJobMetric instantiates a new RolloutsJobMetric object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRolloutsJobMetricWithDefaults

`func NewRolloutsJobMetricWithDefaults() *RolloutsJobMetric`

NewRolloutsJobMetricWithDefaults instantiates a new RolloutsJobMetric object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetMetadata

`func (o *RolloutsJobMetric) GetMetadata() V1ObjectMeta`

GetMetadata returns the Metadata field if non-nil, zero value otherwise.

### GetMetadataOk

`func (o *RolloutsJobMetric) GetMetadataOk() (*V1ObjectMeta, bool)`

GetMetadataOk returns a tuple with the Metadata field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMetadata

`func (o *RolloutsJobMetric) SetMetadata(v V1ObjectMeta)`

SetMetadata sets Metadata field to given value.

### HasMetadata

`func (o *RolloutsJobMetric) HasMetadata() bool`

HasMetadata returns a boolean if a field has been set.

### GetSpec

`func (o *RolloutsJobMetric) GetSpec() V1JobSpec`

GetSpec returns the Spec field if non-nil, zero value otherwise.

### GetSpecOk

`func (o *RolloutsJobMetric) GetSpecOk() (*V1JobSpec, bool)`

GetSpecOk returns a tuple with the Spec field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSpec

`func (o *RolloutsJobMetric) SetSpec(v V1JobSpec)`

SetSpec sets Spec field to given value.

### HasSpec

`func (o *RolloutsJobMetric) HasSpec() bool`

HasSpec returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


