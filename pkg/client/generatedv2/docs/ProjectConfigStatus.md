# ProjectConfigStatus

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Conditions** | Pointer to [**[]V1Condition**](V1Condition.md) | Conditions contains the last observations of the Project Config&#39;s current state.  +patchMergeKey&#x3D;type +patchStrategy&#x3D;merge +listType&#x3D;map +listMapKey&#x3D;type | [optional] 
**LastHandledRefresh** | Pointer to **string** | LastHandledRefresh holds the value of the most recent AnnotationKeyRefresh annotation that was handled by the controller. This field can be used to determine whether the request to refresh the resource has been handled. +optional | [optional] 
**ObservedGeneration** | Pointer to **int32** | ObservedGeneration represents the .metadata.generation that this ProjectConfig was reconciled against. | [optional] 
**WebhookReceivers** | Pointer to [**[]WebhookReceiverDetails**](WebhookReceiverDetails.md) | WebhookReceivers describes the status of Project-specific webhook receivers. | [optional] 

## Methods

### NewProjectConfigStatus

`func NewProjectConfigStatus() *ProjectConfigStatus`

NewProjectConfigStatus instantiates a new ProjectConfigStatus object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewProjectConfigStatusWithDefaults

`func NewProjectConfigStatusWithDefaults() *ProjectConfigStatus`

NewProjectConfigStatusWithDefaults instantiates a new ProjectConfigStatus object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetConditions

`func (o *ProjectConfigStatus) GetConditions() []V1Condition`

GetConditions returns the Conditions field if non-nil, zero value otherwise.

### GetConditionsOk

`func (o *ProjectConfigStatus) GetConditionsOk() (*[]V1Condition, bool)`

GetConditionsOk returns a tuple with the Conditions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConditions

`func (o *ProjectConfigStatus) SetConditions(v []V1Condition)`

SetConditions sets Conditions field to given value.

### HasConditions

`func (o *ProjectConfigStatus) HasConditions() bool`

HasConditions returns a boolean if a field has been set.

### GetLastHandledRefresh

`func (o *ProjectConfigStatus) GetLastHandledRefresh() string`

GetLastHandledRefresh returns the LastHandledRefresh field if non-nil, zero value otherwise.

### GetLastHandledRefreshOk

`func (o *ProjectConfigStatus) GetLastHandledRefreshOk() (*string, bool)`

GetLastHandledRefreshOk returns a tuple with the LastHandledRefresh field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastHandledRefresh

`func (o *ProjectConfigStatus) SetLastHandledRefresh(v string)`

SetLastHandledRefresh sets LastHandledRefresh field to given value.

### HasLastHandledRefresh

`func (o *ProjectConfigStatus) HasLastHandledRefresh() bool`

HasLastHandledRefresh returns a boolean if a field has been set.

### GetObservedGeneration

`func (o *ProjectConfigStatus) GetObservedGeneration() int32`

GetObservedGeneration returns the ObservedGeneration field if non-nil, zero value otherwise.

### GetObservedGenerationOk

`func (o *ProjectConfigStatus) GetObservedGenerationOk() (*int32, bool)`

GetObservedGenerationOk returns a tuple with the ObservedGeneration field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetObservedGeneration

`func (o *ProjectConfigStatus) SetObservedGeneration(v int32)`

SetObservedGeneration sets ObservedGeneration field to given value.

### HasObservedGeneration

`func (o *ProjectConfigStatus) HasObservedGeneration() bool`

HasObservedGeneration returns a boolean if a field has been set.

### GetWebhookReceivers

`func (o *ProjectConfigStatus) GetWebhookReceivers() []WebhookReceiverDetails`

GetWebhookReceivers returns the WebhookReceivers field if non-nil, zero value otherwise.

### GetWebhookReceiversOk

`func (o *ProjectConfigStatus) GetWebhookReceiversOk() (*[]WebhookReceiverDetails, bool)`

GetWebhookReceiversOk returns a tuple with the WebhookReceivers field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWebhookReceivers

`func (o *ProjectConfigStatus) SetWebhookReceivers(v []WebhookReceiverDetails)`

SetWebhookReceivers sets WebhookReceivers field to given value.

### HasWebhookReceivers

`func (o *ProjectConfigStatus) HasWebhookReceivers() bool`

HasWebhookReceivers returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


