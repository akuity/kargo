# CreateRepoCredentialsRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Description** | Pointer to **string** |  | [optional] 
**Name** | Pointer to **string** |  | [optional] 
**Password** | Pointer to **string** | #nosec G117 -- Request data is unmarshaled into this struct, but the struct is never marshaled and transmitted to anywhere. | [optional] 
**RepoUrl** | Pointer to **string** |  | [optional] 
**RepoUrlIsRegex** | Pointer to **bool** |  | [optional] 
**Type** | Pointer to **string** |  | [optional] 
**Username** | Pointer to **string** |  | [optional] 

## Methods

### NewCreateRepoCredentialsRequest

`func NewCreateRepoCredentialsRequest() *CreateRepoCredentialsRequest`

NewCreateRepoCredentialsRequest instantiates a new CreateRepoCredentialsRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewCreateRepoCredentialsRequestWithDefaults

`func NewCreateRepoCredentialsRequestWithDefaults() *CreateRepoCredentialsRequest`

NewCreateRepoCredentialsRequestWithDefaults instantiates a new CreateRepoCredentialsRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDescription

`func (o *CreateRepoCredentialsRequest) GetDescription() string`

GetDescription returns the Description field if non-nil, zero value otherwise.

### GetDescriptionOk

`func (o *CreateRepoCredentialsRequest) GetDescriptionOk() (*string, bool)`

GetDescriptionOk returns a tuple with the Description field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDescription

`func (o *CreateRepoCredentialsRequest) SetDescription(v string)`

SetDescription sets Description field to given value.

### HasDescription

`func (o *CreateRepoCredentialsRequest) HasDescription() bool`

HasDescription returns a boolean if a field has been set.

### GetName

`func (o *CreateRepoCredentialsRequest) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *CreateRepoCredentialsRequest) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *CreateRepoCredentialsRequest) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *CreateRepoCredentialsRequest) HasName() bool`

HasName returns a boolean if a field has been set.

### GetPassword

`func (o *CreateRepoCredentialsRequest) GetPassword() string`

GetPassword returns the Password field if non-nil, zero value otherwise.

### GetPasswordOk

`func (o *CreateRepoCredentialsRequest) GetPasswordOk() (*string, bool)`

GetPasswordOk returns a tuple with the Password field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPassword

`func (o *CreateRepoCredentialsRequest) SetPassword(v string)`

SetPassword sets Password field to given value.

### HasPassword

`func (o *CreateRepoCredentialsRequest) HasPassword() bool`

HasPassword returns a boolean if a field has been set.

### GetRepoUrl

`func (o *CreateRepoCredentialsRequest) GetRepoUrl() string`

GetRepoUrl returns the RepoUrl field if non-nil, zero value otherwise.

### GetRepoUrlOk

`func (o *CreateRepoCredentialsRequest) GetRepoUrlOk() (*string, bool)`

GetRepoUrlOk returns a tuple with the RepoUrl field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRepoUrl

`func (o *CreateRepoCredentialsRequest) SetRepoUrl(v string)`

SetRepoUrl sets RepoUrl field to given value.

### HasRepoUrl

`func (o *CreateRepoCredentialsRequest) HasRepoUrl() bool`

HasRepoUrl returns a boolean if a field has been set.

### GetRepoUrlIsRegex

`func (o *CreateRepoCredentialsRequest) GetRepoUrlIsRegex() bool`

GetRepoUrlIsRegex returns the RepoUrlIsRegex field if non-nil, zero value otherwise.

### GetRepoUrlIsRegexOk

`func (o *CreateRepoCredentialsRequest) GetRepoUrlIsRegexOk() (*bool, bool)`

GetRepoUrlIsRegexOk returns a tuple with the RepoUrlIsRegex field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRepoUrlIsRegex

`func (o *CreateRepoCredentialsRequest) SetRepoUrlIsRegex(v bool)`

SetRepoUrlIsRegex sets RepoUrlIsRegex field to given value.

### HasRepoUrlIsRegex

`func (o *CreateRepoCredentialsRequest) HasRepoUrlIsRegex() bool`

HasRepoUrlIsRegex returns a boolean if a field has been set.

### GetType

`func (o *CreateRepoCredentialsRequest) GetType() string`

GetType returns the Type field if non-nil, zero value otherwise.

### GetTypeOk

`func (o *CreateRepoCredentialsRequest) GetTypeOk() (*string, bool)`

GetTypeOk returns a tuple with the Type field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetType

`func (o *CreateRepoCredentialsRequest) SetType(v string)`

SetType sets Type field to given value.

### HasType

`func (o *CreateRepoCredentialsRequest) HasType() bool`

HasType returns a boolean if a field has been set.

### GetUsername

`func (o *CreateRepoCredentialsRequest) GetUsername() string`

GetUsername returns the Username field if non-nil, zero value otherwise.

### GetUsernameOk

`func (o *CreateRepoCredentialsRequest) GetUsernameOk() (*string, bool)`

GetUsernameOk returns a tuple with the Username field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUsername

`func (o *CreateRepoCredentialsRequest) SetUsername(v string)`

SetUsername sets Username field to given value.

### HasUsername

`func (o *CreateRepoCredentialsRequest) HasUsername() bool`

HasUsername returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


