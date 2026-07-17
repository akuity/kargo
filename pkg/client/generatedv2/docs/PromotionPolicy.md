# PromotionPolicy

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**AutoPromotionEnabled** | Pointer to **bool** | AutoPromotionEnabled indicates whether new Freight can automatically be promoted into the Stage referenced by the Stage field. Note: There are may be other conditions also required for an auto-promotion to occur. This field defaults to false, but is commonly set to true for Stages that subscribe to Warehouses instead of other, upstream Stages. This allows users to define Stages that are automatically updated as soon as new artifacts are detected. | [optional] 
**AutoRollback** | Pointer to [**AutoRollbackConfig**](AutoRollbackConfig.md) | AutoRollback describes the conditions under which this Stage should automatically roll back to the last known-good (verified) Freight. When nil, auto-rollback is disabled.  Kargo Enterprise only: This field is ignored in Kargo OSS. | [optional] 
**Stage** | Pointer to **string** | Stage is the name of the Stage to which this policy applies.  Deprecated: Use StageSelector instead.  +kubebuilder:validation:Pattern&#x3D;^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$ | [optional] 
**StageSelector** | Pointer to [**PromotionPolicySelector**](PromotionPolicySelector.md) | StageSelector is a selector that matches the Stage resource to which this policy applies. | [optional] 

## Methods

### NewPromotionPolicy

`func NewPromotionPolicy() *PromotionPolicy`

NewPromotionPolicy instantiates a new PromotionPolicy object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPromotionPolicyWithDefaults

`func NewPromotionPolicyWithDefaults() *PromotionPolicy`

NewPromotionPolicyWithDefaults instantiates a new PromotionPolicy object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAutoPromotionEnabled

`func (o *PromotionPolicy) GetAutoPromotionEnabled() bool`

GetAutoPromotionEnabled returns the AutoPromotionEnabled field if non-nil, zero value otherwise.

### GetAutoPromotionEnabledOk

`func (o *PromotionPolicy) GetAutoPromotionEnabledOk() (*bool, bool)`

GetAutoPromotionEnabledOk returns a tuple with the AutoPromotionEnabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAutoPromotionEnabled

`func (o *PromotionPolicy) SetAutoPromotionEnabled(v bool)`

SetAutoPromotionEnabled sets AutoPromotionEnabled field to given value.

### HasAutoPromotionEnabled

`func (o *PromotionPolicy) HasAutoPromotionEnabled() bool`

HasAutoPromotionEnabled returns a boolean if a field has been set.

### GetAutoRollback

`func (o *PromotionPolicy) GetAutoRollback() AutoRollbackConfig`

GetAutoRollback returns the AutoRollback field if non-nil, zero value otherwise.

### GetAutoRollbackOk

`func (o *PromotionPolicy) GetAutoRollbackOk() (*AutoRollbackConfig, bool)`

GetAutoRollbackOk returns a tuple with the AutoRollback field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAutoRollback

`func (o *PromotionPolicy) SetAutoRollback(v AutoRollbackConfig)`

SetAutoRollback sets AutoRollback field to given value.

### HasAutoRollback

`func (o *PromotionPolicy) HasAutoRollback() bool`

HasAutoRollback returns a boolean if a field has been set.

### GetStage

`func (o *PromotionPolicy) GetStage() string`

GetStage returns the Stage field if non-nil, zero value otherwise.

### GetStageOk

`func (o *PromotionPolicy) GetStageOk() (*string, bool)`

GetStageOk returns a tuple with the Stage field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStage

`func (o *PromotionPolicy) SetStage(v string)`

SetStage sets Stage field to given value.

### HasStage

`func (o *PromotionPolicy) HasStage() bool`

HasStage returns a boolean if a field has been set.

### GetStageSelector

`func (o *PromotionPolicy) GetStageSelector() PromotionPolicySelector`

GetStageSelector returns the StageSelector field if non-nil, zero value otherwise.

### GetStageSelectorOk

`func (o *PromotionPolicy) GetStageSelectorOk() (*PromotionPolicySelector, bool)`

GetStageSelectorOk returns a tuple with the StageSelector field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStageSelector

`func (o *PromotionPolicy) SetStageSelector(v PromotionPolicySelector)`

SetStageSelector sets StageSelector field to given value.

### HasStageSelector

`func (o *PromotionPolicy) HasStageSelector() bool`

HasStageSelector returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


