# V1PodFailurePolicy

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Rules** | Pointer to [**[]V1PodFailurePolicyRule**](V1PodFailurePolicyRule.md) | A list of pod failure policy rules. The rules are evaluated in order. Once a rule matches a Pod failure, the remaining of the rules are ignored. When no rule matches the Pod failure, the default handling applies - the counter of pod failures is incremented and it is checked against the backoffLimit. At most 20 elements are allowed. +listType&#x3D;atomic | [optional] 

## Methods

### NewV1PodFailurePolicy

`func NewV1PodFailurePolicy() *V1PodFailurePolicy`

NewV1PodFailurePolicy instantiates a new V1PodFailurePolicy object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1PodFailurePolicyWithDefaults

`func NewV1PodFailurePolicyWithDefaults() *V1PodFailurePolicy`

NewV1PodFailurePolicyWithDefaults instantiates a new V1PodFailurePolicy object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetRules

`func (o *V1PodFailurePolicy) GetRules() []V1PodFailurePolicyRule`

GetRules returns the Rules field if non-nil, zero value otherwise.

### GetRulesOk

`func (o *V1PodFailurePolicy) GetRulesOk() (*[]V1PodFailurePolicyRule, bool)`

GetRulesOk returns a tuple with the Rules field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRules

`func (o *V1PodFailurePolicy) SetRules(v []V1PodFailurePolicyRule)`

SetRules sets Rules field to given value.

### HasRules

`func (o *V1PodFailurePolicy) HasRules() bool`

HasRules returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


