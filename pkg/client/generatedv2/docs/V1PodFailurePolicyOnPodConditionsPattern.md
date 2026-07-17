# V1PodFailurePolicyOnPodConditionsPattern

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Status** | Pointer to **string** | Specifies the required Pod condition status. To match a pod condition it is required that the specified status equals the pod condition status. Defaults to True. | [optional] 
**Type** | Pointer to **string** | Specifies the required Pod condition type. To match a pod condition it is required that specified type equals the pod condition type. | [optional] 

## Methods

### NewV1PodFailurePolicyOnPodConditionsPattern

`func NewV1PodFailurePolicyOnPodConditionsPattern() *V1PodFailurePolicyOnPodConditionsPattern`

NewV1PodFailurePolicyOnPodConditionsPattern instantiates a new V1PodFailurePolicyOnPodConditionsPattern object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1PodFailurePolicyOnPodConditionsPatternWithDefaults

`func NewV1PodFailurePolicyOnPodConditionsPatternWithDefaults() *V1PodFailurePolicyOnPodConditionsPattern`

NewV1PodFailurePolicyOnPodConditionsPatternWithDefaults instantiates a new V1PodFailurePolicyOnPodConditionsPattern object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetStatus

`func (o *V1PodFailurePolicyOnPodConditionsPattern) GetStatus() string`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *V1PodFailurePolicyOnPodConditionsPattern) GetStatusOk() (*string, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *V1PodFailurePolicyOnPodConditionsPattern) SetStatus(v string)`

SetStatus sets Status field to given value.

### HasStatus

`func (o *V1PodFailurePolicyOnPodConditionsPattern) HasStatus() bool`

HasStatus returns a boolean if a field has been set.

### GetType

`func (o *V1PodFailurePolicyOnPodConditionsPattern) GetType() string`

GetType returns the Type field if non-nil, zero value otherwise.

### GetTypeOk

`func (o *V1PodFailurePolicyOnPodConditionsPattern) GetTypeOk() (*string, bool)`

GetTypeOk returns a tuple with the Type field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetType

`func (o *V1PodFailurePolicyOnPodConditionsPattern) SetType(v string)`

SetType sets Type field to given value.

### HasType

`func (o *V1PodFailurePolicyOnPodConditionsPattern) HasType() bool`

HasType returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


