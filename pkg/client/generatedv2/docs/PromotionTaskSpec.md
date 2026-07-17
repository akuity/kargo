# PromotionTaskSpec

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Steps** | [**[]PromotionStep**](PromotionStep.md) | Steps specifies the directives to be executed as part of this PromotionTask. The steps as defined here are inflated into a Promotion when it is built from a PromotionTemplate.  +kubebuilder:validation:Required +kubebuilder:validation:MinItems&#x3D;1 +kubebuilder:validation:items:XValidation:message&#x3D;\&quot;PromotionTask step must have uses set and must not reference another task\&quot;,rule&#x3D;\&quot;has(self.uses) &amp;&amp; !has(self.task)\&quot; | 
**Vars** | Pointer to [**[]ExpressionVariable**](ExpressionVariable.md) | Vars specifies the variables available to the PromotionTask. The values of these variables are the default values that can be overridden by the step referencing the task. | [optional] 

## Methods

### NewPromotionTaskSpec

`func NewPromotionTaskSpec(steps []PromotionStep, ) *PromotionTaskSpec`

NewPromotionTaskSpec instantiates a new PromotionTaskSpec object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPromotionTaskSpecWithDefaults

`func NewPromotionTaskSpecWithDefaults() *PromotionTaskSpec`

NewPromotionTaskSpecWithDefaults instantiates a new PromotionTaskSpec object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSteps

`func (o *PromotionTaskSpec) GetSteps() []PromotionStep`

GetSteps returns the Steps field if non-nil, zero value otherwise.

### GetStepsOk

`func (o *PromotionTaskSpec) GetStepsOk() (*[]PromotionStep, bool)`

GetStepsOk returns a tuple with the Steps field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSteps

`func (o *PromotionTaskSpec) SetSteps(v []PromotionStep)`

SetSteps sets Steps field to given value.


### GetVars

`func (o *PromotionTaskSpec) GetVars() []ExpressionVariable`

GetVars returns the Vars field if non-nil, zero value otherwise.

### GetVarsOk

`func (o *PromotionTaskSpec) GetVarsOk() (*[]ExpressionVariable, bool)`

GetVarsOk returns a tuple with the Vars field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVars

`func (o *PromotionTaskSpec) SetVars(v []ExpressionVariable)`

SetVars sets Vars field to given value.

### HasVars

`func (o *PromotionTaskSpec) HasVars() bool`

HasVars returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


