# PromotionStatus

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**CurrentStep** | Pointer to **int32** | CurrentStep is the index of the current promotion step being executed. This permits steps that have already run successfully to be skipped on subsequent reconciliations attempts. | [optional] 
**FinishedAt** | Pointer to **string** | FinishedAt is the time when the promotion was completed. | [optional] 
**Freight** | Pointer to [**FreightReference**](FreightReference.md) | Freight is the detail of the piece of freight that was referenced by this promotion. | [optional] 
**FreightCollection** | Pointer to [**FreightCollection**](FreightCollection.md) | FreightCollection contains the details of the piece of Freight referenced by this Promotion as well as any additional Freight that is carried over from the target Stage&#39;s current state. | [optional] 
**HealthChecks** | Pointer to [**[]HealthCheckStep**](HealthCheckStep.md) | HealthChecks contains the health check directives to be executed after the Promotion has completed. | [optional] 
**LastHandledRefresh** | Pointer to **string** | LastHandledRefresh holds the value of the most recent AnnotationKeyRefresh annotation that was handled by the controller. This field can be used to determine whether the request to refresh the resource has been handled. +optional | [optional] 
**Message** | Pointer to **string** | Message is a display message about the promotion, including any errors preventing the Promotion controller from executing this Promotion. i.e. If the Phase field has a value of Failed, this field can be expected to explain why. | [optional] 
**Phase** | Pointer to **string** | Phase describes where the Promotion currently is in its lifecycle. | [optional] 
**StartedAt** | Pointer to **string** | StartedAt is the time when the promotion started. | [optional] 
**State** | Pointer to **interface{}** | State stores the state of the promotion process between reconciliation attempts. | [optional] 
**StepExecutionMetadata** | Pointer to [**[]StepExecutionMetadata**](StepExecutionMetadata.md) | StepExecutionMetadata tracks metadata pertaining to the execution of individual promotion steps. | [optional] 

## Methods

### NewPromotionStatus

`func NewPromotionStatus() *PromotionStatus`

NewPromotionStatus instantiates a new PromotionStatus object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPromotionStatusWithDefaults

`func NewPromotionStatusWithDefaults() *PromotionStatus`

NewPromotionStatusWithDefaults instantiates a new PromotionStatus object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetCurrentStep

`func (o *PromotionStatus) GetCurrentStep() int32`

GetCurrentStep returns the CurrentStep field if non-nil, zero value otherwise.

### GetCurrentStepOk

`func (o *PromotionStatus) GetCurrentStepOk() (*int32, bool)`

GetCurrentStepOk returns a tuple with the CurrentStep field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCurrentStep

`func (o *PromotionStatus) SetCurrentStep(v int32)`

SetCurrentStep sets CurrentStep field to given value.

### HasCurrentStep

`func (o *PromotionStatus) HasCurrentStep() bool`

HasCurrentStep returns a boolean if a field has been set.

### GetFinishedAt

`func (o *PromotionStatus) GetFinishedAt() string`

GetFinishedAt returns the FinishedAt field if non-nil, zero value otherwise.

### GetFinishedAtOk

`func (o *PromotionStatus) GetFinishedAtOk() (*string, bool)`

GetFinishedAtOk returns a tuple with the FinishedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFinishedAt

`func (o *PromotionStatus) SetFinishedAt(v string)`

SetFinishedAt sets FinishedAt field to given value.

### HasFinishedAt

`func (o *PromotionStatus) HasFinishedAt() bool`

HasFinishedAt returns a boolean if a field has been set.

### GetFreight

`func (o *PromotionStatus) GetFreight() FreightReference`

GetFreight returns the Freight field if non-nil, zero value otherwise.

### GetFreightOk

`func (o *PromotionStatus) GetFreightOk() (*FreightReference, bool)`

GetFreightOk returns a tuple with the Freight field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFreight

`func (o *PromotionStatus) SetFreight(v FreightReference)`

SetFreight sets Freight field to given value.

### HasFreight

`func (o *PromotionStatus) HasFreight() bool`

HasFreight returns a boolean if a field has been set.

### GetFreightCollection

`func (o *PromotionStatus) GetFreightCollection() FreightCollection`

GetFreightCollection returns the FreightCollection field if non-nil, zero value otherwise.

### GetFreightCollectionOk

`func (o *PromotionStatus) GetFreightCollectionOk() (*FreightCollection, bool)`

GetFreightCollectionOk returns a tuple with the FreightCollection field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFreightCollection

`func (o *PromotionStatus) SetFreightCollection(v FreightCollection)`

SetFreightCollection sets FreightCollection field to given value.

### HasFreightCollection

`func (o *PromotionStatus) HasFreightCollection() bool`

HasFreightCollection returns a boolean if a field has been set.

### GetHealthChecks

`func (o *PromotionStatus) GetHealthChecks() []HealthCheckStep`

GetHealthChecks returns the HealthChecks field if non-nil, zero value otherwise.

### GetHealthChecksOk

`func (o *PromotionStatus) GetHealthChecksOk() (*[]HealthCheckStep, bool)`

GetHealthChecksOk returns a tuple with the HealthChecks field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHealthChecks

`func (o *PromotionStatus) SetHealthChecks(v []HealthCheckStep)`

SetHealthChecks sets HealthChecks field to given value.

### HasHealthChecks

`func (o *PromotionStatus) HasHealthChecks() bool`

HasHealthChecks returns a boolean if a field has been set.

### GetLastHandledRefresh

`func (o *PromotionStatus) GetLastHandledRefresh() string`

GetLastHandledRefresh returns the LastHandledRefresh field if non-nil, zero value otherwise.

### GetLastHandledRefreshOk

`func (o *PromotionStatus) GetLastHandledRefreshOk() (*string, bool)`

GetLastHandledRefreshOk returns a tuple with the LastHandledRefresh field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastHandledRefresh

`func (o *PromotionStatus) SetLastHandledRefresh(v string)`

SetLastHandledRefresh sets LastHandledRefresh field to given value.

### HasLastHandledRefresh

`func (o *PromotionStatus) HasLastHandledRefresh() bool`

HasLastHandledRefresh returns a boolean if a field has been set.

### GetMessage

`func (o *PromotionStatus) GetMessage() string`

GetMessage returns the Message field if non-nil, zero value otherwise.

### GetMessageOk

`func (o *PromotionStatus) GetMessageOk() (*string, bool)`

GetMessageOk returns a tuple with the Message field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMessage

`func (o *PromotionStatus) SetMessage(v string)`

SetMessage sets Message field to given value.

### HasMessage

`func (o *PromotionStatus) HasMessage() bool`

HasMessage returns a boolean if a field has been set.

### GetPhase

`func (o *PromotionStatus) GetPhase() string`

GetPhase returns the Phase field if non-nil, zero value otherwise.

### GetPhaseOk

`func (o *PromotionStatus) GetPhaseOk() (*string, bool)`

GetPhaseOk returns a tuple with the Phase field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPhase

`func (o *PromotionStatus) SetPhase(v string)`

SetPhase sets Phase field to given value.

### HasPhase

`func (o *PromotionStatus) HasPhase() bool`

HasPhase returns a boolean if a field has been set.

### GetStartedAt

`func (o *PromotionStatus) GetStartedAt() string`

GetStartedAt returns the StartedAt field if non-nil, zero value otherwise.

### GetStartedAtOk

`func (o *PromotionStatus) GetStartedAtOk() (*string, bool)`

GetStartedAtOk returns a tuple with the StartedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStartedAt

`func (o *PromotionStatus) SetStartedAt(v string)`

SetStartedAt sets StartedAt field to given value.

### HasStartedAt

`func (o *PromotionStatus) HasStartedAt() bool`

HasStartedAt returns a boolean if a field has been set.

### GetState

`func (o *PromotionStatus) GetState() interface{}`

GetState returns the State field if non-nil, zero value otherwise.

### GetStateOk

`func (o *PromotionStatus) GetStateOk() (*interface{}, bool)`

GetStateOk returns a tuple with the State field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetState

`func (o *PromotionStatus) SetState(v interface{})`

SetState sets State field to given value.

### HasState

`func (o *PromotionStatus) HasState() bool`

HasState returns a boolean if a field has been set.

### GetStepExecutionMetadata

`func (o *PromotionStatus) GetStepExecutionMetadata() []StepExecutionMetadata`

GetStepExecutionMetadata returns the StepExecutionMetadata field if non-nil, zero value otherwise.

### GetStepExecutionMetadataOk

`func (o *PromotionStatus) GetStepExecutionMetadataOk() (*[]StepExecutionMetadata, bool)`

GetStepExecutionMetadataOk returns a tuple with the StepExecutionMetadata field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStepExecutionMetadata

`func (o *PromotionStatus) SetStepExecutionMetadata(v []StepExecutionMetadata)`

SetStepExecutionMetadata sets StepExecutionMetadata field to given value.

### HasStepExecutionMetadata

`func (o *PromotionStatus) HasStepExecutionMetadata() bool`

HasStepExecutionMetadata returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


