# FreightCollection

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | Pointer to **string** | ID is a unique and deterministically calculated identifier for the FreightCollection. It is updated on each use of the UpdateOrPush method. | [optional] 
**Items** | Pointer to [**map[string]FreightReference**](FreightReference.md) | Freight is a map of FreightReference objects, indexed by their Warehouse origin. | [optional] 
**VerificationHistory** | Pointer to [**[]VerificationInfo**](VerificationInfo.md) | VerificationHistory is a stack of recent VerificationInfo. By default, the last ten VerificationInfo are stored. | [optional] 

## Methods

### NewFreightCollection

`func NewFreightCollection() *FreightCollection`

NewFreightCollection instantiates a new FreightCollection object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewFreightCollectionWithDefaults

`func NewFreightCollectionWithDefaults() *FreightCollection`

NewFreightCollectionWithDefaults instantiates a new FreightCollection object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *FreightCollection) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *FreightCollection) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *FreightCollection) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *FreightCollection) HasId() bool`

HasId returns a boolean if a field has been set.

### GetItems

`func (o *FreightCollection) GetItems() map[string]FreightReference`

GetItems returns the Items field if non-nil, zero value otherwise.

### GetItemsOk

`func (o *FreightCollection) GetItemsOk() (*map[string]FreightReference, bool)`

GetItemsOk returns a tuple with the Items field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetItems

`func (o *FreightCollection) SetItems(v map[string]FreightReference)`

SetItems sets Items field to given value.

### HasItems

`func (o *FreightCollection) HasItems() bool`

HasItems returns a boolean if a field has been set.

### GetVerificationHistory

`func (o *FreightCollection) GetVerificationHistory() []VerificationInfo`

GetVerificationHistory returns the VerificationHistory field if non-nil, zero value otherwise.

### GetVerificationHistoryOk

`func (o *FreightCollection) GetVerificationHistoryOk() (*[]VerificationInfo, bool)`

GetVerificationHistoryOk returns a tuple with the VerificationHistory field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVerificationHistory

`func (o *FreightCollection) SetVerificationHistory(v []VerificationInfo)`

SetVerificationHistory sets VerificationHistory field to given value.

### HasVerificationHistory

`func (o *FreightCollection) HasVerificationHistory() bool`

HasVerificationHistory returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


