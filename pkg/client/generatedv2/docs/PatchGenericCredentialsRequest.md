# PatchGenericCredentialsRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Data** | Pointer to **map[string]string** |  | [optional] 
**Description** | Pointer to **string** |  | [optional] 
**RemoveKeys** | Pointer to **[]string** |  | [optional] 

## Methods

### NewPatchGenericCredentialsRequest

`func NewPatchGenericCredentialsRequest() *PatchGenericCredentialsRequest`

NewPatchGenericCredentialsRequest instantiates a new PatchGenericCredentialsRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPatchGenericCredentialsRequestWithDefaults

`func NewPatchGenericCredentialsRequestWithDefaults() *PatchGenericCredentialsRequest`

NewPatchGenericCredentialsRequestWithDefaults instantiates a new PatchGenericCredentialsRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetData

`func (o *PatchGenericCredentialsRequest) GetData() map[string]string`

GetData returns the Data field if non-nil, zero value otherwise.

### GetDataOk

`func (o *PatchGenericCredentialsRequest) GetDataOk() (*map[string]string, bool)`

GetDataOk returns a tuple with the Data field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetData

`func (o *PatchGenericCredentialsRequest) SetData(v map[string]string)`

SetData sets Data field to given value.

### HasData

`func (o *PatchGenericCredentialsRequest) HasData() bool`

HasData returns a boolean if a field has been set.

### GetDescription

`func (o *PatchGenericCredentialsRequest) GetDescription() string`

GetDescription returns the Description field if non-nil, zero value otherwise.

### GetDescriptionOk

`func (o *PatchGenericCredentialsRequest) GetDescriptionOk() (*string, bool)`

GetDescriptionOk returns a tuple with the Description field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDescription

`func (o *PatchGenericCredentialsRequest) SetDescription(v string)`

SetDescription sets Description field to given value.

### HasDescription

`func (o *PatchGenericCredentialsRequest) HasDescription() bool`

HasDescription returns a boolean if a field has been set.

### GetRemoveKeys

`func (o *PatchGenericCredentialsRequest) GetRemoveKeys() []string`

GetRemoveKeys returns the RemoveKeys field if non-nil, zero value otherwise.

### GetRemoveKeysOk

`func (o *PatchGenericCredentialsRequest) GetRemoveKeysOk() (*[]string, bool)`

GetRemoveKeysOk returns a tuple with the RemoveKeys field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRemoveKeys

`func (o *PatchGenericCredentialsRequest) SetRemoveKeys(v []string)`

SetRemoveKeys sets RemoveKeys field to given value.

### HasRemoveKeys

`func (o *PatchGenericCredentialsRequest) HasRemoveKeys() bool`

HasRemoveKeys returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


