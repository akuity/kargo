# PromotionStep

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**As** | Pointer to **string** | As is the alias this step can be referred to as. | [optional] 
**Config** | Pointer to **interface{}** | Config is opaque configuration for the PromotionStep that is understood only by each PromotionStep&#39;s implementation. It is legal to utilize expressions in defining values at any level of this block. See https://docs.kargo.io/user-guide/reference-docs/expressions for details. | [optional] 
**ContinueOnError** | Pointer to **bool** | ContinueOnError is a boolean value that, if set to true, will cause the Promotion to continue executing the next step even if this step fails. It also will not permit this failure to impact the overall status of the Promotion. | [optional] 
**If** | Pointer to **string** | If is an optional expression that, if present, must evaluate to a boolean value. If the expression evaluates to false, the step will be skipped. If the expression does not evaluate to a boolean value, the step will be considered to have failed. | [optional] 
**Retry** | Pointer to [**PromotionStepRetry**](PromotionStepRetry.md) | Retry is the retry policy for this step. | [optional] 
**Task** | Pointer to [**PromotionTaskReference**](PromotionTaskReference.md) | Task is a reference to a PromotionTask that should be inflated into a Promotion when it is built from a PromotionTemplate. | [optional] 
**Uses** | Pointer to **string** | Uses identifies a runner that can execute this step.  +kubebuilder:validation:Optional +kubebuilder:validation:MinLength&#x3D;1 | [optional] 
**Vars** | Pointer to [**[]ExpressionVariable**](ExpressionVariable.md) | Vars is a list of variables that can be referenced by expressions in the step&#39;s Config. The values override the values specified in the PromotionSpec. | [optional] 

## Methods

### NewPromotionStep

`func NewPromotionStep() *PromotionStep`

NewPromotionStep instantiates a new PromotionStep object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPromotionStepWithDefaults

`func NewPromotionStepWithDefaults() *PromotionStep`

NewPromotionStepWithDefaults instantiates a new PromotionStep object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAs

`func (o *PromotionStep) GetAs() string`

GetAs returns the As field if non-nil, zero value otherwise.

### GetAsOk

`func (o *PromotionStep) GetAsOk() (*string, bool)`

GetAsOk returns a tuple with the As field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAs

`func (o *PromotionStep) SetAs(v string)`

SetAs sets As field to given value.

### HasAs

`func (o *PromotionStep) HasAs() bool`

HasAs returns a boolean if a field has been set.

### GetConfig

`func (o *PromotionStep) GetConfig() interface{}`

GetConfig returns the Config field if non-nil, zero value otherwise.

### GetConfigOk

`func (o *PromotionStep) GetConfigOk() (*interface{}, bool)`

GetConfigOk returns a tuple with the Config field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConfig

`func (o *PromotionStep) SetConfig(v interface{})`

SetConfig sets Config field to given value.

### HasConfig

`func (o *PromotionStep) HasConfig() bool`

HasConfig returns a boolean if a field has been set.

### GetContinueOnError

`func (o *PromotionStep) GetContinueOnError() bool`

GetContinueOnError returns the ContinueOnError field if non-nil, zero value otherwise.

### GetContinueOnErrorOk

`func (o *PromotionStep) GetContinueOnErrorOk() (*bool, bool)`

GetContinueOnErrorOk returns a tuple with the ContinueOnError field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetContinueOnError

`func (o *PromotionStep) SetContinueOnError(v bool)`

SetContinueOnError sets ContinueOnError field to given value.

### HasContinueOnError

`func (o *PromotionStep) HasContinueOnError() bool`

HasContinueOnError returns a boolean if a field has been set.

### GetIf

`func (o *PromotionStep) GetIf() string`

GetIf returns the If field if non-nil, zero value otherwise.

### GetIfOk

`func (o *PromotionStep) GetIfOk() (*string, bool)`

GetIfOk returns a tuple with the If field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIf

`func (o *PromotionStep) SetIf(v string)`

SetIf sets If field to given value.

### HasIf

`func (o *PromotionStep) HasIf() bool`

HasIf returns a boolean if a field has been set.

### GetRetry

`func (o *PromotionStep) GetRetry() PromotionStepRetry`

GetRetry returns the Retry field if non-nil, zero value otherwise.

### GetRetryOk

`func (o *PromotionStep) GetRetryOk() (*PromotionStepRetry, bool)`

GetRetryOk returns a tuple with the Retry field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRetry

`func (o *PromotionStep) SetRetry(v PromotionStepRetry)`

SetRetry sets Retry field to given value.

### HasRetry

`func (o *PromotionStep) HasRetry() bool`

HasRetry returns a boolean if a field has been set.

### GetTask

`func (o *PromotionStep) GetTask() PromotionTaskReference`

GetTask returns the Task field if non-nil, zero value otherwise.

### GetTaskOk

`func (o *PromotionStep) GetTaskOk() (*PromotionTaskReference, bool)`

GetTaskOk returns a tuple with the Task field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTask

`func (o *PromotionStep) SetTask(v PromotionTaskReference)`

SetTask sets Task field to given value.

### HasTask

`func (o *PromotionStep) HasTask() bool`

HasTask returns a boolean if a field has been set.

### GetUses

`func (o *PromotionStep) GetUses() string`

GetUses returns the Uses field if non-nil, zero value otherwise.

### GetUsesOk

`func (o *PromotionStep) GetUsesOk() (*string, bool)`

GetUsesOk returns a tuple with the Uses field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUses

`func (o *PromotionStep) SetUses(v string)`

SetUses sets Uses field to given value.

### HasUses

`func (o *PromotionStep) HasUses() bool`

HasUses returns a boolean if a field has been set.

### GetVars

`func (o *PromotionStep) GetVars() []ExpressionVariable`

GetVars returns the Vars field if non-nil, zero value otherwise.

### GetVarsOk

`func (o *PromotionStep) GetVarsOk() (*[]ExpressionVariable, bool)`

GetVarsOk returns a tuple with the Vars field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVars

`func (o *PromotionStep) SetVars(v []ExpressionVariable)`

SetVars sets Vars field to given value.

### HasVars

`func (o *PromotionStep) HasVars() bool`

HasVars returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


