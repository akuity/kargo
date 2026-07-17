# V1SuccessPolicy

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Rules** | Pointer to [**[]V1SuccessPolicyRule**](V1SuccessPolicyRule.md) | rules represents the list of alternative rules for the declaring the Jobs as successful before &#x60;.status.succeeded &gt;&#x3D; .spec.completions&#x60;. Once any of the rules are met, the \&quot;SuccessCriteriaMet\&quot; condition is added, and the lingering pods are removed. The terminal state for such a Job has the \&quot;Complete\&quot; condition. Additionally, these rules are evaluated in order; Once the Job meets one of the rules, other rules are ignored. At most 20 elements are allowed. +listType&#x3D;atomic | [optional] 

## Methods

### NewV1SuccessPolicy

`func NewV1SuccessPolicy() *V1SuccessPolicy`

NewV1SuccessPolicy instantiates a new V1SuccessPolicy object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1SuccessPolicyWithDefaults

`func NewV1SuccessPolicyWithDefaults() *V1SuccessPolicy`

NewV1SuccessPolicyWithDefaults instantiates a new V1SuccessPolicy object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetRules

`func (o *V1SuccessPolicy) GetRules() []V1SuccessPolicyRule`

GetRules returns the Rules field if non-nil, zero value otherwise.

### GetRulesOk

`func (o *V1SuccessPolicy) GetRulesOk() (*[]V1SuccessPolicyRule, bool)`

GetRulesOk returns a tuple with the Rules field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRules

`func (o *V1SuccessPolicy) SetRules(v []V1SuccessPolicyRule)`

SetRules sets Rules field to given value.

### HasRules

`func (o *V1SuccessPolicy) HasRules() bool`

HasRules returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


