# V1ContainerRestartRule

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Action** | Pointer to **string** | Specifies the action taken on a container exit if the requirements are satisfied. The only possible value is \&quot;Restart\&quot; to restart the container. +required | [optional] 
**ExitCodes** | Pointer to [**V1ContainerRestartRuleOnExitCodes**](V1ContainerRestartRuleOnExitCodes.md) | Represents the exit codes to check on container exits. +optional +oneOf&#x3D;when | [optional] 

## Methods

### NewV1ContainerRestartRule

`func NewV1ContainerRestartRule() *V1ContainerRestartRule`

NewV1ContainerRestartRule instantiates a new V1ContainerRestartRule object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1ContainerRestartRuleWithDefaults

`func NewV1ContainerRestartRuleWithDefaults() *V1ContainerRestartRule`

NewV1ContainerRestartRuleWithDefaults instantiates a new V1ContainerRestartRule object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAction

`func (o *V1ContainerRestartRule) GetAction() string`

GetAction returns the Action field if non-nil, zero value otherwise.

### GetActionOk

`func (o *V1ContainerRestartRule) GetActionOk() (*string, bool)`

GetActionOk returns a tuple with the Action field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAction

`func (o *V1ContainerRestartRule) SetAction(v string)`

SetAction sets Action field to given value.

### HasAction

`func (o *V1ContainerRestartRule) HasAction() bool`

HasAction returns a boolean if a field has been set.

### GetExitCodes

`func (o *V1ContainerRestartRule) GetExitCodes() V1ContainerRestartRuleOnExitCodes`

GetExitCodes returns the ExitCodes field if non-nil, zero value otherwise.

### GetExitCodesOk

`func (o *V1ContainerRestartRule) GetExitCodesOk() (*V1ContainerRestartRuleOnExitCodes, bool)`

GetExitCodesOk returns a tuple with the ExitCodes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExitCodes

`func (o *V1ContainerRestartRule) SetExitCodes(v V1ContainerRestartRuleOnExitCodes)`

SetExitCodes sets ExitCodes field to given value.

### HasExitCodes

`func (o *V1ContainerRestartRule) HasExitCodes() bool`

HasExitCodes returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


