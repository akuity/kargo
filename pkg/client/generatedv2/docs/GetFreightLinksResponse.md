# GetFreightLinksResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Errors** | Pointer to **[]string** |  | [optional] 
**Links** | Pointer to [**[]ResolvedLink**](ResolvedLink.md) |  | [optional] 

## Methods

### NewGetFreightLinksResponse

`func NewGetFreightLinksResponse() *GetFreightLinksResponse`

NewGetFreightLinksResponse instantiates a new GetFreightLinksResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewGetFreightLinksResponseWithDefaults

`func NewGetFreightLinksResponseWithDefaults() *GetFreightLinksResponse`

NewGetFreightLinksResponseWithDefaults instantiates a new GetFreightLinksResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetErrors

`func (o *GetFreightLinksResponse) GetErrors() []string`

GetErrors returns the Errors field if non-nil, zero value otherwise.

### GetErrorsOk

`func (o *GetFreightLinksResponse) GetErrorsOk() (*[]string, bool)`

GetErrorsOk returns a tuple with the Errors field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetErrors

`func (o *GetFreightLinksResponse) SetErrors(v []string)`

SetErrors sets Errors field to given value.

### HasErrors

`func (o *GetFreightLinksResponse) HasErrors() bool`

HasErrors returns a boolean if a field has been set.

### GetLinks

`func (o *GetFreightLinksResponse) GetLinks() []ResolvedLink`

GetLinks returns the Links field if non-nil, zero value otherwise.

### GetLinksOk

`func (o *GetFreightLinksResponse) GetLinksOk() (*[]ResolvedLink, bool)`

GetLinksOk returns a tuple with the Links field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLinks

`func (o *GetFreightLinksResponse) SetLinks(v []ResolvedLink)`

SetLinks sets Links field to given value.

### HasLinks

`func (o *GetFreightLinksResponse) HasLinks() bool`

HasLinks returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


