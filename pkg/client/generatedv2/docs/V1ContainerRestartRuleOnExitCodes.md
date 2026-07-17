# V1ContainerRestartRuleOnExitCodes

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Operator** | Pointer to **string** | Represents the relationship between the container exit code(s) and the specified values. Possible values are: - In: the requirement is satisfied if the container exit code is in the   set of specified values. - NotIn: the requirement is satisfied if the container exit code is   not in the set of specified values. +required | [optional] 
**Values** | Pointer to **[]int32** | Specifies the set of values to check for container exit codes. At most 255 elements are allowed. +optional +listType&#x3D;set | [optional] 

## Methods

### NewV1ContainerRestartRuleOnExitCodes

`func NewV1ContainerRestartRuleOnExitCodes() *V1ContainerRestartRuleOnExitCodes`

NewV1ContainerRestartRuleOnExitCodes instantiates a new V1ContainerRestartRuleOnExitCodes object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1ContainerRestartRuleOnExitCodesWithDefaults

`func NewV1ContainerRestartRuleOnExitCodesWithDefaults() *V1ContainerRestartRuleOnExitCodes`

NewV1ContainerRestartRuleOnExitCodesWithDefaults instantiates a new V1ContainerRestartRuleOnExitCodes object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetOperator

`func (o *V1ContainerRestartRuleOnExitCodes) GetOperator() string`

GetOperator returns the Operator field if non-nil, zero value otherwise.

### GetOperatorOk

`func (o *V1ContainerRestartRuleOnExitCodes) GetOperatorOk() (*string, bool)`

GetOperatorOk returns a tuple with the Operator field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOperator

`func (o *V1ContainerRestartRuleOnExitCodes) SetOperator(v string)`

SetOperator sets Operator field to given value.

### HasOperator

`func (o *V1ContainerRestartRuleOnExitCodes) HasOperator() bool`

HasOperator returns a boolean if a field has been set.

### GetValues

`func (o *V1ContainerRestartRuleOnExitCodes) GetValues() []int32`

GetValues returns the Values field if non-nil, zero value otherwise.

### GetValuesOk

`func (o *V1ContainerRestartRuleOnExitCodes) GetValuesOk() (*[]int32, bool)`

GetValuesOk returns a tuple with the Values field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetValues

`func (o *V1ContainerRestartRuleOnExitCodes) SetValues(v []int32)`

SetValues sets Values field to given value.

### HasValues

`func (o *V1ContainerRestartRuleOnExitCodes) HasValues() bool`

HasValues returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


