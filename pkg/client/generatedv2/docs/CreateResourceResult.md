# CreateResourceResult

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**CreatedResourceManifest** | Pointer to **map[string]interface{}** |  | [optional] 
**Error** | Pointer to **string** |  | [optional] 

## Methods

### NewCreateResourceResult

`func NewCreateResourceResult() *CreateResourceResult`

NewCreateResourceResult instantiates a new CreateResourceResult object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewCreateResourceResultWithDefaults

`func NewCreateResourceResultWithDefaults() *CreateResourceResult`

NewCreateResourceResultWithDefaults instantiates a new CreateResourceResult object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetCreatedResourceManifest

`func (o *CreateResourceResult) GetCreatedResourceManifest() map[string]interface{}`

GetCreatedResourceManifest returns the CreatedResourceManifest field if non-nil, zero value otherwise.

### GetCreatedResourceManifestOk

`func (o *CreateResourceResult) GetCreatedResourceManifestOk() (*map[string]interface{}, bool)`

GetCreatedResourceManifestOk returns a tuple with the CreatedResourceManifest field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedResourceManifest

`func (o *CreateResourceResult) SetCreatedResourceManifest(v map[string]interface{})`

SetCreatedResourceManifest sets CreatedResourceManifest field to given value.

### HasCreatedResourceManifest

`func (o *CreateResourceResult) HasCreatedResourceManifest() bool`

HasCreatedResourceManifest returns a boolean if a field has been set.

### GetError

`func (o *CreateResourceResult) GetError() string`

GetError returns the Error field if non-nil, zero value otherwise.

### GetErrorOk

`func (o *CreateResourceResult) GetErrorOk() (*string, bool)`

GetErrorOk returns a tuple with the Error field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetError

`func (o *CreateResourceResult) SetError(v string)`

SetError sets Error field to given value.

### HasError

`func (o *CreateResourceResult) HasError() bool`

HasError returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


