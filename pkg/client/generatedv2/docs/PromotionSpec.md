# PromotionSpec

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Freight** | Pointer to **string** | Freight specifies the piece of Freight to be promoted into the Stage. Exactly one of Freight or Origin must be set.  +kubebuilder:validation:Optional +kubebuilder:validation:MinLength&#x3D;1 +kubebuilder:validation:MaxLength&#x3D;253 +kubebuilder:validation:Pattern&#x3D;&#x60;^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$&#x60; +akuity:test-kubebuilder-pattern&#x3D;KubernetesName | [optional] 
**Origin** | Pointer to [**FreightOrigin**](FreightOrigin.md) | Origin, when set, identifies the FreightOrigin whose auto-promotion candidate should be promoted. The mutating webhook resolves this to the candidate Freight for that origin and fills Freight before the Promotion is persisted. Exactly one of Freight or Origin must be set.  +kubebuilder:validation:Optional | [optional] 
**Stage** | **string** | Stage specifies the name of the Stage to which this Promotion applies. The Stage referenced by this field MUST be in the same namespace as the Promotion.  +kubebuilder:validation:Required +kubebuilder:validation:MinLength&#x3D;1 +kubebuilder:validation:MaxLength&#x3D;253 +kubebuilder:validation:Pattern&#x3D;&#x60;^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$&#x60; +akuity:test-kubebuilder-pattern&#x3D;KubernetesName | 
**Steps** | [**[]PromotionStep**](PromotionStep.md) | Steps specifies the directives to be executed as part of this Promotion. The order in which the directives are executed is the order in which they are listed in this field.  +kubebuilder:validation:Required +kubebuilder:validation:MinItems&#x3D;1 +kubebuilder:validation:items:XValidation:message&#x3D;\&quot;Promotion step must have uses set and must not reference a task\&quot;,rule&#x3D;\&quot;has(self.uses) &amp;&amp; !has(self.task)\&quot; | 
**Target** | Pointer to **string** | Target optionally names the Target, within the Promotion&#39;s own Project (namespace), that this Promotion promotes Freight to. Targets allow a single Stage to govern -- and promote Freight to -- multiple destinations. When set, the named Target must be one that the referenced Stage governs, i.e. one selected by the Stage&#39;s targetSelectors.  When empty (the default), the Promotion promotes to the Stage itself. This preserves the behavior of Promotions created before Targets existed: classic Stages -- those without targetSelectors -- govern no Targets, so their Promotions leave this field empty.  +kubebuilder:validation:Optional +kubebuilder:validation:MaxLength&#x3D;253 +kubebuilder:validation:Pattern&#x3D;&#x60;^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$&#x60; +akuity:test-kubebuilder-pattern&#x3D;KubernetesName | [optional] 
**Vars** | Pointer to [**[]ExpressionVariable**](ExpressionVariable.md) | Vars is a list of variables that can be referenced by expressions in promotion steps. | [optional] 

## Methods

### NewPromotionSpec

`func NewPromotionSpec(stage string, steps []PromotionStep, ) *PromotionSpec`

NewPromotionSpec instantiates a new PromotionSpec object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPromotionSpecWithDefaults

`func NewPromotionSpecWithDefaults() *PromotionSpec`

NewPromotionSpecWithDefaults instantiates a new PromotionSpec object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetFreight

`func (o *PromotionSpec) GetFreight() string`

GetFreight returns the Freight field if non-nil, zero value otherwise.

### GetFreightOk

`func (o *PromotionSpec) GetFreightOk() (*string, bool)`

GetFreightOk returns a tuple with the Freight field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFreight

`func (o *PromotionSpec) SetFreight(v string)`

SetFreight sets Freight field to given value.

### HasFreight

`func (o *PromotionSpec) HasFreight() bool`

HasFreight returns a boolean if a field has been set.

### GetOrigin

`func (o *PromotionSpec) GetOrigin() FreightOrigin`

GetOrigin returns the Origin field if non-nil, zero value otherwise.

### GetOriginOk

`func (o *PromotionSpec) GetOriginOk() (*FreightOrigin, bool)`

GetOriginOk returns a tuple with the Origin field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOrigin

`func (o *PromotionSpec) SetOrigin(v FreightOrigin)`

SetOrigin sets Origin field to given value.

### HasOrigin

`func (o *PromotionSpec) HasOrigin() bool`

HasOrigin returns a boolean if a field has been set.

### GetStage

`func (o *PromotionSpec) GetStage() string`

GetStage returns the Stage field if non-nil, zero value otherwise.

### GetStageOk

`func (o *PromotionSpec) GetStageOk() (*string, bool)`

GetStageOk returns a tuple with the Stage field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStage

`func (o *PromotionSpec) SetStage(v string)`

SetStage sets Stage field to given value.


### GetSteps

`func (o *PromotionSpec) GetSteps() []PromotionStep`

GetSteps returns the Steps field if non-nil, zero value otherwise.

### GetStepsOk

`func (o *PromotionSpec) GetStepsOk() (*[]PromotionStep, bool)`

GetStepsOk returns a tuple with the Steps field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSteps

`func (o *PromotionSpec) SetSteps(v []PromotionStep)`

SetSteps sets Steps field to given value.


### GetTarget

`func (o *PromotionSpec) GetTarget() string`

GetTarget returns the Target field if non-nil, zero value otherwise.

### GetTargetOk

`func (o *PromotionSpec) GetTargetOk() (*string, bool)`

GetTargetOk returns a tuple with the Target field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTarget

`func (o *PromotionSpec) SetTarget(v string)`

SetTarget sets Target field to given value.

### HasTarget

`func (o *PromotionSpec) HasTarget() bool`

HasTarget returns a boolean if a field has been set.

### GetVars

`func (o *PromotionSpec) GetVars() []ExpressionVariable`

GetVars returns the Vars field if non-nil, zero value otherwise.

### GetVarsOk

`func (o *PromotionSpec) GetVarsOk() (*[]ExpressionVariable, bool)`

GetVarsOk returns a tuple with the Vars field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVars

`func (o *PromotionSpec) SetVars(v []ExpressionVariable)`

SetVars sets Vars field to given value.

### HasVars

`func (o *PromotionSpec) HasVars() bool`

HasVars returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


