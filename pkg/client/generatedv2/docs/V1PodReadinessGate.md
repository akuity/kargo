# V1PodReadinessGate

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ConditionType** | Pointer to **string** | ConditionType refers to a condition in the pod&#39;s condition list with matching type. | [optional] 

## Methods

### NewV1PodReadinessGate

`func NewV1PodReadinessGate() *V1PodReadinessGate`

NewV1PodReadinessGate instantiates a new V1PodReadinessGate object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1PodReadinessGateWithDefaults

`func NewV1PodReadinessGateWithDefaults() *V1PodReadinessGate`

NewV1PodReadinessGateWithDefaults instantiates a new V1PodReadinessGate object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetConditionType

`func (o *V1PodReadinessGate) GetConditionType() string`

GetConditionType returns the ConditionType field if non-nil, zero value otherwise.

### GetConditionTypeOk

`func (o *V1PodReadinessGate) GetConditionTypeOk() (*string, bool)`

GetConditionTypeOk returns a tuple with the ConditionType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConditionType

`func (o *V1PodReadinessGate) SetConditionType(v string)`

SetConditionType sets ConditionType field to given value.

### HasConditionType

`func (o *V1PodReadinessGate) HasConditionType() bool`

HasConditionType returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


