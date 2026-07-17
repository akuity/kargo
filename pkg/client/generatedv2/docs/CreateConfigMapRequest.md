# CreateConfigMapRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Data** | Pointer to **map[string]string** |  | [optional] 
**Description** | Pointer to **string** |  | [optional] 
**Name** | Pointer to **string** |  | [optional] 
**Replicate** | Pointer to **bool** |  | [optional] 

## Methods

### NewCreateConfigMapRequest

`func NewCreateConfigMapRequest() *CreateConfigMapRequest`

NewCreateConfigMapRequest instantiates a new CreateConfigMapRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewCreateConfigMapRequestWithDefaults

`func NewCreateConfigMapRequestWithDefaults() *CreateConfigMapRequest`

NewCreateConfigMapRequestWithDefaults instantiates a new CreateConfigMapRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetData

`func (o *CreateConfigMapRequest) GetData() map[string]string`

GetData returns the Data field if non-nil, zero value otherwise.

### GetDataOk

`func (o *CreateConfigMapRequest) GetDataOk() (*map[string]string, bool)`

GetDataOk returns a tuple with the Data field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetData

`func (o *CreateConfigMapRequest) SetData(v map[string]string)`

SetData sets Data field to given value.

### HasData

`func (o *CreateConfigMapRequest) HasData() bool`

HasData returns a boolean if a field has been set.

### GetDescription

`func (o *CreateConfigMapRequest) GetDescription() string`

GetDescription returns the Description field if non-nil, zero value otherwise.

### GetDescriptionOk

`func (o *CreateConfigMapRequest) GetDescriptionOk() (*string, bool)`

GetDescriptionOk returns a tuple with the Description field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDescription

`func (o *CreateConfigMapRequest) SetDescription(v string)`

SetDescription sets Description field to given value.

### HasDescription

`func (o *CreateConfigMapRequest) HasDescription() bool`

HasDescription returns a boolean if a field has been set.

### GetName

`func (o *CreateConfigMapRequest) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *CreateConfigMapRequest) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *CreateConfigMapRequest) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *CreateConfigMapRequest) HasName() bool`

HasName returns a boolean if a field has been set.

### GetReplicate

`func (o *CreateConfigMapRequest) GetReplicate() bool`

GetReplicate returns the Replicate field if non-nil, zero value otherwise.

### GetReplicateOk

`func (o *CreateConfigMapRequest) GetReplicateOk() (*bool, bool)`

GetReplicateOk returns a tuple with the Replicate field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReplicate

`func (o *CreateConfigMapRequest) SetReplicate(v bool)`

SetReplicate sets Replicate field to given value.

### HasReplicate

`func (o *CreateConfigMapRequest) HasReplicate() bool`

HasReplicate returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


