# DeleteResourceResult

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**DeletedResourceManifest** | Pointer to **map[string]interface{}** |  | [optional] 
**Error** | Pointer to **string** |  | [optional] 

## Methods

### NewDeleteResourceResult

`func NewDeleteResourceResult() *DeleteResourceResult`

NewDeleteResourceResult instantiates a new DeleteResourceResult object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewDeleteResourceResultWithDefaults

`func NewDeleteResourceResultWithDefaults() *DeleteResourceResult`

NewDeleteResourceResultWithDefaults instantiates a new DeleteResourceResult object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDeletedResourceManifest

`func (o *DeleteResourceResult) GetDeletedResourceManifest() map[string]interface{}`

GetDeletedResourceManifest returns the DeletedResourceManifest field if non-nil, zero value otherwise.

### GetDeletedResourceManifestOk

`func (o *DeleteResourceResult) GetDeletedResourceManifestOk() (*map[string]interface{}, bool)`

GetDeletedResourceManifestOk returns a tuple with the DeletedResourceManifest field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDeletedResourceManifest

`func (o *DeleteResourceResult) SetDeletedResourceManifest(v map[string]interface{})`

SetDeletedResourceManifest sets DeletedResourceManifest field to given value.

### HasDeletedResourceManifest

`func (o *DeleteResourceResult) HasDeletedResourceManifest() bool`

HasDeletedResourceManifest returns a boolean if a field has been set.

### GetError

`func (o *DeleteResourceResult) GetError() string`

GetError returns the Error field if non-nil, zero value otherwise.

### GetErrorOk

`func (o *DeleteResourceResult) GetErrorOk() (*string, bool)`

GetErrorOk returns a tuple with the Error field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetError

`func (o *DeleteResourceResult) SetError(v string)`

SetError sets Error field to given value.

### HasError

`func (o *DeleteResourceResult) HasError() bool`

HasError returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


