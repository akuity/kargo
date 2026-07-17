# GetStageLinksResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Errors** | Pointer to **[]string** |  | [optional] 
**Links** | Pointer to [**[]ResolvedLink**](ResolvedLink.md) |  | [optional] 

## Methods

### NewGetStageLinksResponse

`func NewGetStageLinksResponse() *GetStageLinksResponse`

NewGetStageLinksResponse instantiates a new GetStageLinksResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewGetStageLinksResponseWithDefaults

`func NewGetStageLinksResponseWithDefaults() *GetStageLinksResponse`

NewGetStageLinksResponseWithDefaults instantiates a new GetStageLinksResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetErrors

`func (o *GetStageLinksResponse) GetErrors() []string`

GetErrors returns the Errors field if non-nil, zero value otherwise.

### GetErrorsOk

`func (o *GetStageLinksResponse) GetErrorsOk() (*[]string, bool)`

GetErrorsOk returns a tuple with the Errors field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetErrors

`func (o *GetStageLinksResponse) SetErrors(v []string)`

SetErrors sets Errors field to given value.

### HasErrors

`func (o *GetStageLinksResponse) HasErrors() bool`

HasErrors returns a boolean if a field has been set.

### GetLinks

`func (o *GetStageLinksResponse) GetLinks() []ResolvedLink`

GetLinks returns the Links field if non-nil, zero value otherwise.

### GetLinksOk

`func (o *GetStageLinksResponse) GetLinksOk() (*[]ResolvedLink, bool)`

GetLinksOk returns a tuple with the Links field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLinks

`func (o *GetStageLinksResponse) SetLinks(v []ResolvedLink)`

SetLinks sets Links field to given value.

### HasLinks

`func (o *GetStageLinksResponse) HasLinks() bool`

HasLinks returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


