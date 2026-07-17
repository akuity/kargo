# AutoPromotionHold

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Actor** | Pointer to **string** | Actor identifies the user who triggered the hold. | [optional] 
**CreatedAt** | Pointer to **string** | CreatedAt is the creation timestamp of the Promotion that established this hold. | [optional] 
**FreightName** | Pointer to **string** | FreightName is the name of the Freight selected when the hold was created. | [optional] 
**Origin** | Pointer to [**FreightOrigin**](FreightOrigin.md) | Origin describes the FreightOrigin pinned by this hold. It matches the enclosing map key. | [optional] 
**PromotionName** | Pointer to **string** | PromotionName is the name of the Promotion that established this hold. Stored here as a paper trail that survives Promotion garbage collection. | [optional] 

## Methods

### NewAutoPromotionHold

`func NewAutoPromotionHold() *AutoPromotionHold`

NewAutoPromotionHold instantiates a new AutoPromotionHold object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAutoPromotionHoldWithDefaults

`func NewAutoPromotionHoldWithDefaults() *AutoPromotionHold`

NewAutoPromotionHoldWithDefaults instantiates a new AutoPromotionHold object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetActor

`func (o *AutoPromotionHold) GetActor() string`

GetActor returns the Actor field if non-nil, zero value otherwise.

### GetActorOk

`func (o *AutoPromotionHold) GetActorOk() (*string, bool)`

GetActorOk returns a tuple with the Actor field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetActor

`func (o *AutoPromotionHold) SetActor(v string)`

SetActor sets Actor field to given value.

### HasActor

`func (o *AutoPromotionHold) HasActor() bool`

HasActor returns a boolean if a field has been set.

### GetCreatedAt

`func (o *AutoPromotionHold) GetCreatedAt() string`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *AutoPromotionHold) GetCreatedAtOk() (*string, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *AutoPromotionHold) SetCreatedAt(v string)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *AutoPromotionHold) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.

### GetFreightName

`func (o *AutoPromotionHold) GetFreightName() string`

GetFreightName returns the FreightName field if non-nil, zero value otherwise.

### GetFreightNameOk

`func (o *AutoPromotionHold) GetFreightNameOk() (*string, bool)`

GetFreightNameOk returns a tuple with the FreightName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFreightName

`func (o *AutoPromotionHold) SetFreightName(v string)`

SetFreightName sets FreightName field to given value.

### HasFreightName

`func (o *AutoPromotionHold) HasFreightName() bool`

HasFreightName returns a boolean if a field has been set.

### GetOrigin

`func (o *AutoPromotionHold) GetOrigin() FreightOrigin`

GetOrigin returns the Origin field if non-nil, zero value otherwise.

### GetOriginOk

`func (o *AutoPromotionHold) GetOriginOk() (*FreightOrigin, bool)`

GetOriginOk returns a tuple with the Origin field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOrigin

`func (o *AutoPromotionHold) SetOrigin(v FreightOrigin)`

SetOrigin sets Origin field to given value.

### HasOrigin

`func (o *AutoPromotionHold) HasOrigin() bool`

HasOrigin returns a boolean if a field has been set.

### GetPromotionName

`func (o *AutoPromotionHold) GetPromotionName() string`

GetPromotionName returns the PromotionName field if non-nil, zero value otherwise.

### GetPromotionNameOk

`func (o *AutoPromotionHold) GetPromotionNameOk() (*string, bool)`

GetPromotionNameOk returns a tuple with the PromotionName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPromotionName

`func (o *AutoPromotionHold) SetPromotionName(v string)`

SetPromotionName sets PromotionName field to given value.

### HasPromotionName

`func (o *AutoPromotionHold) HasPromotionName() bool`

HasPromotionName returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


