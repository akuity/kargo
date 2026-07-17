# PromoteToStageRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Freight** | Pointer to **string** |  | [optional] 
**FreightAlias** | Pointer to **string** |  | [optional] 
**Origin** | Pointer to **string** | Origin is the canonical Freight origin key (e.g. \&quot;Warehouse/foo\&quot;). When set, the promotion webhook resolves it to the current auto-promotion candidate. Exactly one of Freight, FreightAlias, or Origin must be set. | [optional] 

## Methods

### NewPromoteToStageRequest

`func NewPromoteToStageRequest() *PromoteToStageRequest`

NewPromoteToStageRequest instantiates a new PromoteToStageRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPromoteToStageRequestWithDefaults

`func NewPromoteToStageRequestWithDefaults() *PromoteToStageRequest`

NewPromoteToStageRequestWithDefaults instantiates a new PromoteToStageRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetFreight

`func (o *PromoteToStageRequest) GetFreight() string`

GetFreight returns the Freight field if non-nil, zero value otherwise.

### GetFreightOk

`func (o *PromoteToStageRequest) GetFreightOk() (*string, bool)`

GetFreightOk returns a tuple with the Freight field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFreight

`func (o *PromoteToStageRequest) SetFreight(v string)`

SetFreight sets Freight field to given value.

### HasFreight

`func (o *PromoteToStageRequest) HasFreight() bool`

HasFreight returns a boolean if a field has been set.

### GetFreightAlias

`func (o *PromoteToStageRequest) GetFreightAlias() string`

GetFreightAlias returns the FreightAlias field if non-nil, zero value otherwise.

### GetFreightAliasOk

`func (o *PromoteToStageRequest) GetFreightAliasOk() (*string, bool)`

GetFreightAliasOk returns a tuple with the FreightAlias field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFreightAlias

`func (o *PromoteToStageRequest) SetFreightAlias(v string)`

SetFreightAlias sets FreightAlias field to given value.

### HasFreightAlias

`func (o *PromoteToStageRequest) HasFreightAlias() bool`

HasFreightAlias returns a boolean if a field has been set.

### GetOrigin

`func (o *PromoteToStageRequest) GetOrigin() string`

GetOrigin returns the Origin field if non-nil, zero value otherwise.

### GetOriginOk

`func (o *PromoteToStageRequest) GetOriginOk() (*string, bool)`

GetOriginOk returns a tuple with the Origin field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOrigin

`func (o *PromoteToStageRequest) SetOrigin(v string)`

SetOrigin sets Origin field to given value.

### HasOrigin

`func (o *PromoteToStageRequest) HasOrigin() bool`

HasOrigin returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


