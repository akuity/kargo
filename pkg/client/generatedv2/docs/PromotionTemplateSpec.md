# PromotionTemplateSpec

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Steps** | Pointer to [**[]PromotionStep**](PromotionStep.md) | Steps specifies the directives to be executed as part of a Promotion. The order in which the directives are executed is the order in which they are listed in this field.  +kubebuilder:validation:MinItems&#x3D;1 +kubebuilder:validation:items:XValidation:message&#x3D;\&quot;PromotionTemplate step must have exactly one of uses or task set\&quot;,rule&#x3D;\&quot;(has(self.uses) ? !has(self.task) : has(self.task))\&quot; +kubebuilder:validation:items:XValidation:message&#x3D;\&quot;PromotionTemplate step referencing a task cannot set continueOnError\&quot;,rule&#x3D;\&quot;!has(self.task) || !has(self.continueOnError)\&quot; +kubebuilder:validation:items:XValidation:message&#x3D;\&quot;PromotionTemplate step referencing a task cannot set retry\&quot;,rule&#x3D;\&quot;!has(self.task) || !has(self.retry)\&quot; | [optional] 
**Vars** | Pointer to [**[]ExpressionVariable**](ExpressionVariable.md) | Vars is a list of variables that can be referenced by expressions in promotion steps. | [optional] 

## Methods

### NewPromotionTemplateSpec

`func NewPromotionTemplateSpec() *PromotionTemplateSpec`

NewPromotionTemplateSpec instantiates a new PromotionTemplateSpec object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPromotionTemplateSpecWithDefaults

`func NewPromotionTemplateSpecWithDefaults() *PromotionTemplateSpec`

NewPromotionTemplateSpecWithDefaults instantiates a new PromotionTemplateSpec object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSteps

`func (o *PromotionTemplateSpec) GetSteps() []PromotionStep`

GetSteps returns the Steps field if non-nil, zero value otherwise.

### GetStepsOk

`func (o *PromotionTemplateSpec) GetStepsOk() (*[]PromotionStep, bool)`

GetStepsOk returns a tuple with the Steps field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSteps

`func (o *PromotionTemplateSpec) SetSteps(v []PromotionStep)`

SetSteps sets Steps field to given value.

### HasSteps

`func (o *PromotionTemplateSpec) HasSteps() bool`

HasSteps returns a boolean if a field has been set.

### GetVars

`func (o *PromotionTemplateSpec) GetVars() []ExpressionVariable`

GetVars returns the Vars field if non-nil, zero value otherwise.

### GetVarsOk

`func (o *PromotionTemplateSpec) GetVarsOk() (*[]ExpressionVariable, bool)`

GetVarsOk returns a tuple with the Vars field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVars

`func (o *PromotionTemplateSpec) SetVars(v []ExpressionVariable)`

SetVars sets Vars field to given value.

### HasVars

`func (o *PromotionTemplateSpec) HasVars() bool`

HasVars returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


