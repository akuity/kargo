# StageSpec

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**PromotionTemplate** | Pointer to [**PromotionTemplate**](PromotionTemplate.md) | PromotionTemplate describes how to incorporate Freight into the Stage using a Promotion. | [optional] 
**RequestedFreight** | Pointer to [**[]FreightRequest**](FreightRequest.md) | RequestedFreight expresses the Stage&#39;s need for certain pieces of Freight, each having originated from a particular Warehouse. This list must be non-empty. In the common case, a Stage will request Freight having originated from just one specific Warehouse. In advanced cases, requesting Freight from multiple Warehouses provides a method of advancing new artifacts of different types through parallel pipelines at different speeds. This can be useful, for instance, if a Stage is home to multiple microservices that are independently versioned.  +kubebuilder:validation:MinItems&#x3D;1 | [optional] 
**Shard** | Pointer to **string** | Shard is the name of the shard that this Stage belongs to. This is an optional field. If not specified, the Stage will belong to the default shard. A defaulting webhook will sync the value of the kargo.akuity.io/shard label with the value of this field. When this field is empty, the webhook will ensure that label is absent. | [optional] 
**TargetSelectors** | Pointer to [**[]V1LabelSelector**](V1LabelSelector.md) | TargetSelectors select the Targets that this Stage governs and promotes Freight to, matching Targets by their labels within the Stage&#39;s own Project. A Target is selected when it matches any selector in this list. A Stage may govern any number of Targets this way.  When this field is nil (the default), the Stage operates in classic mode: it governs a single implicit \&quot;stage-self\&quot; Target that the controller creates and maintains on the Stage&#39;s behalf. This preserves the behavior of Stages authored before Targets existed. An empty selector in a non-empty list selects all Targets in the Project.  +optional | [optional] 
**Vars** | Pointer to [**[]ExpressionVariable**](ExpressionVariable.md) | Vars is a list of variables that can be referenced anywhere in the StageSpec that supports expressions. For example, the PromotionTemplate and arguments of the Verification. | [optional] 
**Verification** | Pointer to [**Verification**](Verification.md) | Verification describes how to verify a Stage&#39;s current Freight is fit for promotion downstream. | [optional] 

## Methods

### NewStageSpec

`func NewStageSpec() *StageSpec`

NewStageSpec instantiates a new StageSpec object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewStageSpecWithDefaults

`func NewStageSpecWithDefaults() *StageSpec`

NewStageSpecWithDefaults instantiates a new StageSpec object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetPromotionTemplate

`func (o *StageSpec) GetPromotionTemplate() PromotionTemplate`

GetPromotionTemplate returns the PromotionTemplate field if non-nil, zero value otherwise.

### GetPromotionTemplateOk

`func (o *StageSpec) GetPromotionTemplateOk() (*PromotionTemplate, bool)`

GetPromotionTemplateOk returns a tuple with the PromotionTemplate field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPromotionTemplate

`func (o *StageSpec) SetPromotionTemplate(v PromotionTemplate)`

SetPromotionTemplate sets PromotionTemplate field to given value.

### HasPromotionTemplate

`func (o *StageSpec) HasPromotionTemplate() bool`

HasPromotionTemplate returns a boolean if a field has been set.

### GetRequestedFreight

`func (o *StageSpec) GetRequestedFreight() []FreightRequest`

GetRequestedFreight returns the RequestedFreight field if non-nil, zero value otherwise.

### GetRequestedFreightOk

`func (o *StageSpec) GetRequestedFreightOk() (*[]FreightRequest, bool)`

GetRequestedFreightOk returns a tuple with the RequestedFreight field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRequestedFreight

`func (o *StageSpec) SetRequestedFreight(v []FreightRequest)`

SetRequestedFreight sets RequestedFreight field to given value.

### HasRequestedFreight

`func (o *StageSpec) HasRequestedFreight() bool`

HasRequestedFreight returns a boolean if a field has been set.

### GetShard

`func (o *StageSpec) GetShard() string`

GetShard returns the Shard field if non-nil, zero value otherwise.

### GetShardOk

`func (o *StageSpec) GetShardOk() (*string, bool)`

GetShardOk returns a tuple with the Shard field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetShard

`func (o *StageSpec) SetShard(v string)`

SetShard sets Shard field to given value.

### HasShard

`func (o *StageSpec) HasShard() bool`

HasShard returns a boolean if a field has been set.

### GetTargetSelectors

`func (o *StageSpec) GetTargetSelectors() []V1LabelSelector`

GetTargetSelectors returns the TargetSelectors field if non-nil, zero value otherwise.

### GetTargetSelectorsOk

`func (o *StageSpec) GetTargetSelectorsOk() (*[]V1LabelSelector, bool)`

GetTargetSelectorsOk returns a tuple with the TargetSelectors field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTargetSelectors

`func (o *StageSpec) SetTargetSelectors(v []V1LabelSelector)`

SetTargetSelectors sets TargetSelectors field to given value.

### HasTargetSelectors

`func (o *StageSpec) HasTargetSelectors() bool`

HasTargetSelectors returns a boolean if a field has been set.

### GetVars

`func (o *StageSpec) GetVars() []ExpressionVariable`

GetVars returns the Vars field if non-nil, zero value otherwise.

### GetVarsOk

`func (o *StageSpec) GetVarsOk() (*[]ExpressionVariable, bool)`

GetVarsOk returns a tuple with the Vars field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVars

`func (o *StageSpec) SetVars(v []ExpressionVariable)`

SetVars sets Vars field to given value.

### HasVars

`func (o *StageSpec) HasVars() bool`

HasVars returns a boolean if a field has been set.

### GetVerification

`func (o *StageSpec) GetVerification() Verification`

GetVerification returns the Verification field if non-nil, zero value otherwise.

### GetVerificationOk

`func (o *StageSpec) GetVerificationOk() (*Verification, bool)`

GetVerificationOk returns a tuple with the Verification field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVerification

`func (o *StageSpec) SetVerification(v Verification)`

SetVerification sets Verification field to given value.

### HasVerification

`func (o *StageSpec) HasVerification() bool`

HasVerification returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


