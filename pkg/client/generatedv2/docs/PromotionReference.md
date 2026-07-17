# PromotionReference

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**FinishedAt** | Pointer to **string** | FinishedAt is the time at which the Promotion was completed. | [optional] 
**Freight** | Pointer to [**FreightReference**](FreightReference.md) | Freight is the freight being promoted. | [optional] 
**Name** | Pointer to **string** | Name is the name of the Promotion. | [optional] 
**Status** | Pointer to [**PromotionStatus**](PromotionStatus.md) | Status is the (optional) status of the Promotion. | [optional] 

## Methods

### NewPromotionReference

`func NewPromotionReference() *PromotionReference`

NewPromotionReference instantiates a new PromotionReference object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPromotionReferenceWithDefaults

`func NewPromotionReferenceWithDefaults() *PromotionReference`

NewPromotionReferenceWithDefaults instantiates a new PromotionReference object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetFinishedAt

`func (o *PromotionReference) GetFinishedAt() string`

GetFinishedAt returns the FinishedAt field if non-nil, zero value otherwise.

### GetFinishedAtOk

`func (o *PromotionReference) GetFinishedAtOk() (*string, bool)`

GetFinishedAtOk returns a tuple with the FinishedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFinishedAt

`func (o *PromotionReference) SetFinishedAt(v string)`

SetFinishedAt sets FinishedAt field to given value.

### HasFinishedAt

`func (o *PromotionReference) HasFinishedAt() bool`

HasFinishedAt returns a boolean if a field has been set.

### GetFreight

`func (o *PromotionReference) GetFreight() FreightReference`

GetFreight returns the Freight field if non-nil, zero value otherwise.

### GetFreightOk

`func (o *PromotionReference) GetFreightOk() (*FreightReference, bool)`

GetFreightOk returns a tuple with the Freight field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFreight

`func (o *PromotionReference) SetFreight(v FreightReference)`

SetFreight sets Freight field to given value.

### HasFreight

`func (o *PromotionReference) HasFreight() bool`

HasFreight returns a boolean if a field has been set.

### GetName

`func (o *PromotionReference) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *PromotionReference) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *PromotionReference) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *PromotionReference) HasName() bool`

HasName returns a boolean if a field has been set.

### GetStatus

`func (o *PromotionReference) GetStatus() PromotionStatus`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *PromotionReference) GetStatusOk() (*PromotionStatus, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *PromotionReference) SetStatus(v PromotionStatus)`

SetStatus sets Status field to given value.

### HasStatus

`func (o *PromotionReference) HasStatus() bool`

HasStatus returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


