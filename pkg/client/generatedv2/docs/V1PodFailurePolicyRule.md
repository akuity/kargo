# V1PodFailurePolicyRule

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Action** | Pointer to [**V1PodFailurePolicyAction**](V1PodFailurePolicyAction.md) | Specifies the action taken on a pod failure when the requirements are satisfied. Possible values are:  - FailJob: indicates that the pod&#39;s job is marked as Failed and all   running pods are terminated. - FailIndex: indicates that the pod&#39;s index is marked as Failed and will   not be restarted. - Ignore: indicates that the counter towards the .backoffLimit is not   incremented and a replacement pod is created. - Count: indicates that the pod is handled in the default way - the   counter towards the .backoffLimit is incremented. Additional values are considered to be added in the future. Clients should react to an unknown action by skipping the rule. | [optional] 
**OnExitCodes** | Pointer to [**V1PodFailurePolicyOnExitCodesRequirement**](V1PodFailurePolicyOnExitCodesRequirement.md) | Represents the requirement on the container exit codes. +optional | [optional] 
**OnPodConditions** | Pointer to [**[]V1PodFailurePolicyOnPodConditionsPattern**](V1PodFailurePolicyOnPodConditionsPattern.md) | Represents the requirement on the pod conditions. The requirement is represented as a list of pod condition patterns. The requirement is satisfied if at least one pattern matches an actual pod condition. At most 20 elements are allowed. +listType&#x3D;atomic +optional | [optional] 

## Methods

### NewV1PodFailurePolicyRule

`func NewV1PodFailurePolicyRule() *V1PodFailurePolicyRule`

NewV1PodFailurePolicyRule instantiates a new V1PodFailurePolicyRule object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1PodFailurePolicyRuleWithDefaults

`func NewV1PodFailurePolicyRuleWithDefaults() *V1PodFailurePolicyRule`

NewV1PodFailurePolicyRuleWithDefaults instantiates a new V1PodFailurePolicyRule object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAction

`func (o *V1PodFailurePolicyRule) GetAction() V1PodFailurePolicyAction`

GetAction returns the Action field if non-nil, zero value otherwise.

### GetActionOk

`func (o *V1PodFailurePolicyRule) GetActionOk() (*V1PodFailurePolicyAction, bool)`

GetActionOk returns a tuple with the Action field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAction

`func (o *V1PodFailurePolicyRule) SetAction(v V1PodFailurePolicyAction)`

SetAction sets Action field to given value.

### HasAction

`func (o *V1PodFailurePolicyRule) HasAction() bool`

HasAction returns a boolean if a field has been set.

### GetOnExitCodes

`func (o *V1PodFailurePolicyRule) GetOnExitCodes() V1PodFailurePolicyOnExitCodesRequirement`

GetOnExitCodes returns the OnExitCodes field if non-nil, zero value otherwise.

### GetOnExitCodesOk

`func (o *V1PodFailurePolicyRule) GetOnExitCodesOk() (*V1PodFailurePolicyOnExitCodesRequirement, bool)`

GetOnExitCodesOk returns a tuple with the OnExitCodes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOnExitCodes

`func (o *V1PodFailurePolicyRule) SetOnExitCodes(v V1PodFailurePolicyOnExitCodesRequirement)`

SetOnExitCodes sets OnExitCodes field to given value.

### HasOnExitCodes

`func (o *V1PodFailurePolicyRule) HasOnExitCodes() bool`

HasOnExitCodes returns a boolean if a field has been set.

### GetOnPodConditions

`func (o *V1PodFailurePolicyRule) GetOnPodConditions() []V1PodFailurePolicyOnPodConditionsPattern`

GetOnPodConditions returns the OnPodConditions field if non-nil, zero value otherwise.

### GetOnPodConditionsOk

`func (o *V1PodFailurePolicyRule) GetOnPodConditionsOk() (*[]V1PodFailurePolicyOnPodConditionsPattern, bool)`

GetOnPodConditionsOk returns a tuple with the OnPodConditions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOnPodConditions

`func (o *V1PodFailurePolicyRule) SetOnPodConditions(v []V1PodFailurePolicyOnPodConditionsPattern)`

SetOnPodConditions sets OnPodConditions field to given value.

### HasOnPodConditions

`func (o *V1PodFailurePolicyRule) HasOnPodConditions() bool`

HasOnPodConditions returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


