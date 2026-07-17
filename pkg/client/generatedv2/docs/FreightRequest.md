# FreightRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Origin** | [**FreightOrigin**](FreightOrigin.md) | Origin specifies from where the requested Freight must have originated. This is a required field.  +kubebuilder:validation:Required | 
**Sources** | Pointer to [**FreightSources**](FreightSources.md) | Sources describes where the requested Freight may be obtained from. This is a required field. | [optional] 

## Methods

### NewFreightRequest

`func NewFreightRequest(origin FreightOrigin, ) *FreightRequest`

NewFreightRequest instantiates a new FreightRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewFreightRequestWithDefaults

`func NewFreightRequestWithDefaults() *FreightRequest`

NewFreightRequestWithDefaults instantiates a new FreightRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetOrigin

`func (o *FreightRequest) GetOrigin() FreightOrigin`

GetOrigin returns the Origin field if non-nil, zero value otherwise.

### GetOriginOk

`func (o *FreightRequest) GetOriginOk() (*FreightOrigin, bool)`

GetOriginOk returns a tuple with the Origin field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOrigin

`func (o *FreightRequest) SetOrigin(v FreightOrigin)`

SetOrigin sets Origin field to given value.


### GetSources

`func (o *FreightRequest) GetSources() FreightSources`

GetSources returns the Sources field if non-nil, zero value otherwise.

### GetSourcesOk

`func (o *FreightRequest) GetSourcesOk() (*FreightSources, bool)`

GetSourcesOk returns a tuple with the Sources field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSources

`func (o *FreightRequest) SetSources(v FreightSources)`

SetSources sets Sources field to given value.

### HasSources

`func (o *FreightRequest) HasSources() bool`

HasSources returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


