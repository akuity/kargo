# CreateGenericCredentialsRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Data** | Pointer to **map[string]string** |  | [optional] 
**Description** | Pointer to **string** |  | [optional] 
**Name** | Pointer to **string** |  | [optional] 
**Replicate** | Pointer to **bool** |  | [optional] 
**Type** | Pointer to **string** | Type is the Kubernetes Secret type (e.g. \&quot;Opaque\&quot; or \&quot;kubernetes.io/dockerconfigjson\&quot;). It is immutable, so it may only be set at creation time. When empty, Kubernetes defaults it to \&quot;Opaque\&quot;. | [optional] 

## Methods

### NewCreateGenericCredentialsRequest

`func NewCreateGenericCredentialsRequest() *CreateGenericCredentialsRequest`

NewCreateGenericCredentialsRequest instantiates a new CreateGenericCredentialsRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewCreateGenericCredentialsRequestWithDefaults

`func NewCreateGenericCredentialsRequestWithDefaults() *CreateGenericCredentialsRequest`

NewCreateGenericCredentialsRequestWithDefaults instantiates a new CreateGenericCredentialsRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetData

`func (o *CreateGenericCredentialsRequest) GetData() map[string]string`

GetData returns the Data field if non-nil, zero value otherwise.

### GetDataOk

`func (o *CreateGenericCredentialsRequest) GetDataOk() (*map[string]string, bool)`

GetDataOk returns a tuple with the Data field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetData

`func (o *CreateGenericCredentialsRequest) SetData(v map[string]string)`

SetData sets Data field to given value.

### HasData

`func (o *CreateGenericCredentialsRequest) HasData() bool`

HasData returns a boolean if a field has been set.

### GetDescription

`func (o *CreateGenericCredentialsRequest) GetDescription() string`

GetDescription returns the Description field if non-nil, zero value otherwise.

### GetDescriptionOk

`func (o *CreateGenericCredentialsRequest) GetDescriptionOk() (*string, bool)`

GetDescriptionOk returns a tuple with the Description field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDescription

`func (o *CreateGenericCredentialsRequest) SetDescription(v string)`

SetDescription sets Description field to given value.

### HasDescription

`func (o *CreateGenericCredentialsRequest) HasDescription() bool`

HasDescription returns a boolean if a field has been set.

### GetName

`func (o *CreateGenericCredentialsRequest) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *CreateGenericCredentialsRequest) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *CreateGenericCredentialsRequest) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *CreateGenericCredentialsRequest) HasName() bool`

HasName returns a boolean if a field has been set.

### GetReplicate

`func (o *CreateGenericCredentialsRequest) GetReplicate() bool`

GetReplicate returns the Replicate field if non-nil, zero value otherwise.

### GetReplicateOk

`func (o *CreateGenericCredentialsRequest) GetReplicateOk() (*bool, bool)`

GetReplicateOk returns a tuple with the Replicate field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReplicate

`func (o *CreateGenericCredentialsRequest) SetReplicate(v bool)`

SetReplicate sets Replicate field to given value.

### HasReplicate

`func (o *CreateGenericCredentialsRequest) HasReplicate() bool`

HasReplicate returns a boolean if a field has been set.

### GetType

`func (o *CreateGenericCredentialsRequest) GetType() string`

GetType returns the Type field if non-nil, zero value otherwise.

### GetTypeOk

`func (o *CreateGenericCredentialsRequest) GetTypeOk() (*string, bool)`

GetTypeOk returns a tuple with the Type field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetType

`func (o *CreateGenericCredentialsRequest) SetType(v string)`

SetType sets Type field to given value.

### HasType

`func (o *CreateGenericCredentialsRequest) HasType() bool`

HasType returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


