# UpdateRepoCredentialsRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Description** | Pointer to **string** |  | [optional] 
**Password** | Pointer to **string** | #nosec G117 -- Request data is unmarshaled into this struct, but the struct is never marshaled and transmitted to anywhere. | [optional] 
**RepoUrl** | Pointer to **string** |  | [optional] 
**RepoUrlIsRegex** | Pointer to **bool** |  | [optional] 
**Type** | Pointer to **string** |  | [optional] 
**Username** | Pointer to **string** |  | [optional] 

## Methods

### NewUpdateRepoCredentialsRequest

`func NewUpdateRepoCredentialsRequest() *UpdateRepoCredentialsRequest`

NewUpdateRepoCredentialsRequest instantiates a new UpdateRepoCredentialsRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewUpdateRepoCredentialsRequestWithDefaults

`func NewUpdateRepoCredentialsRequestWithDefaults() *UpdateRepoCredentialsRequest`

NewUpdateRepoCredentialsRequestWithDefaults instantiates a new UpdateRepoCredentialsRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDescription

`func (o *UpdateRepoCredentialsRequest) GetDescription() string`

GetDescription returns the Description field if non-nil, zero value otherwise.

### GetDescriptionOk

`func (o *UpdateRepoCredentialsRequest) GetDescriptionOk() (*string, bool)`

GetDescriptionOk returns a tuple with the Description field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDescription

`func (o *UpdateRepoCredentialsRequest) SetDescription(v string)`

SetDescription sets Description field to given value.

### HasDescription

`func (o *UpdateRepoCredentialsRequest) HasDescription() bool`

HasDescription returns a boolean if a field has been set.

### GetPassword

`func (o *UpdateRepoCredentialsRequest) GetPassword() string`

GetPassword returns the Password field if non-nil, zero value otherwise.

### GetPasswordOk

`func (o *UpdateRepoCredentialsRequest) GetPasswordOk() (*string, bool)`

GetPasswordOk returns a tuple with the Password field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPassword

`func (o *UpdateRepoCredentialsRequest) SetPassword(v string)`

SetPassword sets Password field to given value.

### HasPassword

`func (o *UpdateRepoCredentialsRequest) HasPassword() bool`

HasPassword returns a boolean if a field has been set.

### GetRepoUrl

`func (o *UpdateRepoCredentialsRequest) GetRepoUrl() string`

GetRepoUrl returns the RepoUrl field if non-nil, zero value otherwise.

### GetRepoUrlOk

`func (o *UpdateRepoCredentialsRequest) GetRepoUrlOk() (*string, bool)`

GetRepoUrlOk returns a tuple with the RepoUrl field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRepoUrl

`func (o *UpdateRepoCredentialsRequest) SetRepoUrl(v string)`

SetRepoUrl sets RepoUrl field to given value.

### HasRepoUrl

`func (o *UpdateRepoCredentialsRequest) HasRepoUrl() bool`

HasRepoUrl returns a boolean if a field has been set.

### GetRepoUrlIsRegex

`func (o *UpdateRepoCredentialsRequest) GetRepoUrlIsRegex() bool`

GetRepoUrlIsRegex returns the RepoUrlIsRegex field if non-nil, zero value otherwise.

### GetRepoUrlIsRegexOk

`func (o *UpdateRepoCredentialsRequest) GetRepoUrlIsRegexOk() (*bool, bool)`

GetRepoUrlIsRegexOk returns a tuple with the RepoUrlIsRegex field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRepoUrlIsRegex

`func (o *UpdateRepoCredentialsRequest) SetRepoUrlIsRegex(v bool)`

SetRepoUrlIsRegex sets RepoUrlIsRegex field to given value.

### HasRepoUrlIsRegex

`func (o *UpdateRepoCredentialsRequest) HasRepoUrlIsRegex() bool`

HasRepoUrlIsRegex returns a boolean if a field has been set.

### GetType

`func (o *UpdateRepoCredentialsRequest) GetType() string`

GetType returns the Type field if non-nil, zero value otherwise.

### GetTypeOk

`func (o *UpdateRepoCredentialsRequest) GetTypeOk() (*string, bool)`

GetTypeOk returns a tuple with the Type field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetType

`func (o *UpdateRepoCredentialsRequest) SetType(v string)`

SetType sets Type field to given value.

### HasType

`func (o *UpdateRepoCredentialsRequest) HasType() bool`

HasType returns a boolean if a field has been set.

### GetUsername

`func (o *UpdateRepoCredentialsRequest) GetUsername() string`

GetUsername returns the Username field if non-nil, zero value otherwise.

### GetUsernameOk

`func (o *UpdateRepoCredentialsRequest) GetUsernameOk() (*string, bool)`

GetUsernameOk returns a tuple with the Username field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUsername

`func (o *UpdateRepoCredentialsRequest) SetUsername(v string)`

SetUsername sets Username field to given value.

### HasUsername

`func (o *UpdateRepoCredentialsRequest) HasUsername() bool`

HasUsername returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


