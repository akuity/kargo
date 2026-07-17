# StageStatus

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**AutoPromotionEnabled** | Pointer to **bool** | AutoPromotionEnabled indicates whether automatic promotion is enabled for the Stage based on the ProjectConfig. | [optional] 
**AutoPromotionHolds** | Pointer to [**map[string]AutoPromotionHold**](AutoPromotionHold.md) | AutoPromotionHolds records active auto-promotion holds for this Stage. A hold is established when a Promotion selects Freight other than the auto-promotion candidate for that origin, pausing auto-promotion for that origin until explicitly released. Auto-promotions themselves never establish holds. Keys are string representations of FreightOrigins (e.g. \&quot;Warehouse/my-warehouse\&quot;); values describe the Promotion that established the hold. | [optional] 
**Conditions** | Pointer to [**[]V1Condition**](V1Condition.md) | Conditions contains the last observations of the Stage&#39;s current state. +patchMergeKey&#x3D;type +patchStrategy&#x3D;merge +listType&#x3D;map +listMapKey&#x3D;type | [optional] 
**CurrentPromotion** | Pointer to [**PromotionReference**](PromotionReference.md) | CurrentPromotion is a reference to the currently Running promotion. | [optional] 
**FreightHistory** | Pointer to [**[]FreightCollection**](FreightCollection.md) | FreightHistory is a list of recent Freight selections that were deployed to the Stage. By default, the last ten Freight selections are stored. The first item in the list is the most recent Freight selection and currently deployed to the Stage, subsequent items are older selections. | [optional] 
**FreightSummary** | Pointer to **string** | FreightSummary is human-readable text maintained by the controller that summarizes what Freight is currently deployed to the Stage. For Stages that request a single piece of Freight AND the request has been fulfilled, this field will simply contain the name of the Freight. For Stages that request a single piece of Freight AND the request has NOT been fulfilled, or for Stages that request multiple pieces of Freight, this field will contain a summary of fulfilled/requested Freight. The existence of this field is a workaround for kubectl limitations so that this complex but valuable information can be displayed in a column in response to &#x60;kubectl get stages&#x60;. | [optional] 
**Health** | Pointer to [**Health**](Health.md) | Health is the Stage&#39;s last observed health. | [optional] 
**LastHandledRefresh** | Pointer to **string** | LastHandledRefresh holds the value of the most recent AnnotationKeyRefresh annotation that was handled by the controller. This field can be used to determine whether the request to refresh the resource has been handled. +optional | [optional] 
**LastPromotion** | Pointer to [**PromotionReference**](PromotionReference.md) | LastPromotion is a reference to the last completed promotion. | [optional] 
**Metadata** | Pointer to **map[string]interface{}** | Metadata is a map of arbitrary metadata associated with the Stage. This is useful for storing additional information about the Stage that can be shared across promotions, verifications, or other processes. | [optional] 
**ObservedGeneration** | Pointer to **int32** | ObservedGeneration represents the .metadata.generation that this Stage status was reconciled against. | [optional] 

## Methods

### NewStageStatus

`func NewStageStatus() *StageStatus`

NewStageStatus instantiates a new StageStatus object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewStageStatusWithDefaults

`func NewStageStatusWithDefaults() *StageStatus`

NewStageStatusWithDefaults instantiates a new StageStatus object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAutoPromotionEnabled

`func (o *StageStatus) GetAutoPromotionEnabled() bool`

GetAutoPromotionEnabled returns the AutoPromotionEnabled field if non-nil, zero value otherwise.

### GetAutoPromotionEnabledOk

`func (o *StageStatus) GetAutoPromotionEnabledOk() (*bool, bool)`

GetAutoPromotionEnabledOk returns a tuple with the AutoPromotionEnabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAutoPromotionEnabled

`func (o *StageStatus) SetAutoPromotionEnabled(v bool)`

SetAutoPromotionEnabled sets AutoPromotionEnabled field to given value.

### HasAutoPromotionEnabled

`func (o *StageStatus) HasAutoPromotionEnabled() bool`

HasAutoPromotionEnabled returns a boolean if a field has been set.

### GetAutoPromotionHolds

`func (o *StageStatus) GetAutoPromotionHolds() map[string]AutoPromotionHold`

GetAutoPromotionHolds returns the AutoPromotionHolds field if non-nil, zero value otherwise.

### GetAutoPromotionHoldsOk

`func (o *StageStatus) GetAutoPromotionHoldsOk() (*map[string]AutoPromotionHold, bool)`

GetAutoPromotionHoldsOk returns a tuple with the AutoPromotionHolds field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAutoPromotionHolds

`func (o *StageStatus) SetAutoPromotionHolds(v map[string]AutoPromotionHold)`

SetAutoPromotionHolds sets AutoPromotionHolds field to given value.

### HasAutoPromotionHolds

`func (o *StageStatus) HasAutoPromotionHolds() bool`

HasAutoPromotionHolds returns a boolean if a field has been set.

### GetConditions

`func (o *StageStatus) GetConditions() []V1Condition`

GetConditions returns the Conditions field if non-nil, zero value otherwise.

### GetConditionsOk

`func (o *StageStatus) GetConditionsOk() (*[]V1Condition, bool)`

GetConditionsOk returns a tuple with the Conditions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConditions

`func (o *StageStatus) SetConditions(v []V1Condition)`

SetConditions sets Conditions field to given value.

### HasConditions

`func (o *StageStatus) HasConditions() bool`

HasConditions returns a boolean if a field has been set.

### GetCurrentPromotion

`func (o *StageStatus) GetCurrentPromotion() PromotionReference`

GetCurrentPromotion returns the CurrentPromotion field if non-nil, zero value otherwise.

### GetCurrentPromotionOk

`func (o *StageStatus) GetCurrentPromotionOk() (*PromotionReference, bool)`

GetCurrentPromotionOk returns a tuple with the CurrentPromotion field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCurrentPromotion

`func (o *StageStatus) SetCurrentPromotion(v PromotionReference)`

SetCurrentPromotion sets CurrentPromotion field to given value.

### HasCurrentPromotion

`func (o *StageStatus) HasCurrentPromotion() bool`

HasCurrentPromotion returns a boolean if a field has been set.

### GetFreightHistory

`func (o *StageStatus) GetFreightHistory() []FreightCollection`

GetFreightHistory returns the FreightHistory field if non-nil, zero value otherwise.

### GetFreightHistoryOk

`func (o *StageStatus) GetFreightHistoryOk() (*[]FreightCollection, bool)`

GetFreightHistoryOk returns a tuple with the FreightHistory field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFreightHistory

`func (o *StageStatus) SetFreightHistory(v []FreightCollection)`

SetFreightHistory sets FreightHistory field to given value.

### HasFreightHistory

`func (o *StageStatus) HasFreightHistory() bool`

HasFreightHistory returns a boolean if a field has been set.

### GetFreightSummary

`func (o *StageStatus) GetFreightSummary() string`

GetFreightSummary returns the FreightSummary field if non-nil, zero value otherwise.

### GetFreightSummaryOk

`func (o *StageStatus) GetFreightSummaryOk() (*string, bool)`

GetFreightSummaryOk returns a tuple with the FreightSummary field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFreightSummary

`func (o *StageStatus) SetFreightSummary(v string)`

SetFreightSummary sets FreightSummary field to given value.

### HasFreightSummary

`func (o *StageStatus) HasFreightSummary() bool`

HasFreightSummary returns a boolean if a field has been set.

### GetHealth

`func (o *StageStatus) GetHealth() Health`

GetHealth returns the Health field if non-nil, zero value otherwise.

### GetHealthOk

`func (o *StageStatus) GetHealthOk() (*Health, bool)`

GetHealthOk returns a tuple with the Health field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHealth

`func (o *StageStatus) SetHealth(v Health)`

SetHealth sets Health field to given value.

### HasHealth

`func (o *StageStatus) HasHealth() bool`

HasHealth returns a boolean if a field has been set.

### GetLastHandledRefresh

`func (o *StageStatus) GetLastHandledRefresh() string`

GetLastHandledRefresh returns the LastHandledRefresh field if non-nil, zero value otherwise.

### GetLastHandledRefreshOk

`func (o *StageStatus) GetLastHandledRefreshOk() (*string, bool)`

GetLastHandledRefreshOk returns a tuple with the LastHandledRefresh field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastHandledRefresh

`func (o *StageStatus) SetLastHandledRefresh(v string)`

SetLastHandledRefresh sets LastHandledRefresh field to given value.

### HasLastHandledRefresh

`func (o *StageStatus) HasLastHandledRefresh() bool`

HasLastHandledRefresh returns a boolean if a field has been set.

### GetLastPromotion

`func (o *StageStatus) GetLastPromotion() PromotionReference`

GetLastPromotion returns the LastPromotion field if non-nil, zero value otherwise.

### GetLastPromotionOk

`func (o *StageStatus) GetLastPromotionOk() (*PromotionReference, bool)`

GetLastPromotionOk returns a tuple with the LastPromotion field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastPromotion

`func (o *StageStatus) SetLastPromotion(v PromotionReference)`

SetLastPromotion sets LastPromotion field to given value.

### HasLastPromotion

`func (o *StageStatus) HasLastPromotion() bool`

HasLastPromotion returns a boolean if a field has been set.

### GetMetadata

`func (o *StageStatus) GetMetadata() map[string]interface{}`

GetMetadata returns the Metadata field if non-nil, zero value otherwise.

### GetMetadataOk

`func (o *StageStatus) GetMetadataOk() (*map[string]interface{}, bool)`

GetMetadataOk returns a tuple with the Metadata field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMetadata

`func (o *StageStatus) SetMetadata(v map[string]interface{})`

SetMetadata sets Metadata field to given value.

### HasMetadata

`func (o *StageStatus) HasMetadata() bool`

HasMetadata returns a boolean if a field has been set.

### GetObservedGeneration

`func (o *StageStatus) GetObservedGeneration() int32`

GetObservedGeneration returns the ObservedGeneration field if non-nil, zero value otherwise.

### GetObservedGenerationOk

`func (o *StageStatus) GetObservedGenerationOk() (*int32, bool)`

GetObservedGenerationOk returns a tuple with the ObservedGeneration field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetObservedGeneration

`func (o *StageStatus) SetObservedGeneration(v int32)`

SetObservedGeneration sets ObservedGeneration field to given value.

### HasObservedGeneration

`func (o *StageStatus) HasObservedGeneration() bool`

HasObservedGeneration returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


