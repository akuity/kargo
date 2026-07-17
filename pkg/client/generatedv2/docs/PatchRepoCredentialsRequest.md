# PatchRepoCredentialsRequest

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

### NewPatchRepoCredentialsRequest

`func NewPatchRepoCredentialsRequest() *PatchRepoCredentialsRequest`

NewPatchRepoCredentialsRequest instantiates a new PatchRepoCredentialsRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPatchRepoCredentialsRequestWithDefaults

`func NewPatchRepoCredentialsRequestWithDefaults() *PatchRepoCredentialsRequest`

NewPatchRepoCredentialsRequestWithDefaults instantiates a new PatchRepoCredentialsRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDescription

`func (o *PatchRepoCredentialsRequest) GetDescription() string`

GetDescription returns the Description field if non-nil, zero value otherwise.

### GetDescriptionOk

`func (o *PatchRepoCredentialsRequest) GetDescriptionOk() (*string, bool)`

GetDescriptionOk returns a tuple with the Description field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDescription

`func (o *PatchRepoCredentialsRequest) SetDescription(v string)`

SetDescription sets Description field to given value.

### HasDescription

`func (o *PatchRepoCredentialsRequest) HasDescription() bool`

HasDescription returns a boolean if a field has been set.

### GetPassword

`func (o *PatchRepoCredentialsRequest) GetPassword() string`

GetPassword returns the Password field if non-nil, zero value otherwise.

### GetPasswordOk

`func (o *PatchRepoCredentialsRequest) GetPasswordOk() (*string, bool)`

GetPasswordOk returns a tuple with the Password field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPassword

`func (o *PatchRepoCredentialsRequest) SetPassword(v string)`

SetPassword sets Password field to given value.

### HasPassword

`func (o *PatchRepoCredentialsRequest) HasPassword() bool`

HasPassword returns a boolean if a field has been set.

### GetRepoUrl

`func (o *PatchRepoCredentialsRequest) GetRepoUrl() string`

GetRepoUrl returns the RepoUrl field if non-nil, zero value otherwise.

### GetRepoUrlOk

`func (o *PatchRepoCredentialsRequest) GetRepoUrlOk() (*string, bool)`

GetRepoUrlOk returns a tuple with the RepoUrl field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRepoUrl

`func (o *PatchRepoCredentialsRequest) SetRepoUrl(v string)`

SetRepoUrl sets RepoUrl field to given value.

### HasRepoUrl

`func (o *PatchRepoCredentialsRequest) HasRepoUrl() bool`

HasRepoUrl returns a boolean if a field has been set.

### GetRepoUrlIsRegex

`func (o *PatchRepoCredentialsRequest) GetRepoUrlIsRegex() bool`

GetRepoUrlIsRegex returns the RepoUrlIsRegex field if non-nil, zero value otherwise.

### GetRepoUrlIsRegexOk

`func (o *PatchRepoCredentialsRequest) GetRepoUrlIsRegexOk() (*bool, bool)`

GetRepoUrlIsRegexOk returns a tuple with the RepoUrlIsRegex field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRepoUrlIsRegex

`func (o *PatchRepoCredentialsRequest) SetRepoUrlIsRegex(v bool)`

SetRepoUrlIsRegex sets RepoUrlIsRegex field to given value.

### HasRepoUrlIsRegex

`func (o *PatchRepoCredentialsRequest) HasRepoUrlIsRegex() bool`

HasRepoUrlIsRegex returns a boolean if a field has been set.

### GetType

`func (o *PatchRepoCredentialsRequest) GetType() string`

GetType returns the Type field if non-nil, zero value otherwise.

### GetTypeOk

`func (o *PatchRepoCredentialsRequest) GetTypeOk() (*string, bool)`

GetTypeOk returns a tuple with the Type field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetType

`func (o *PatchRepoCredentialsRequest) SetType(v string)`

SetType sets Type field to given value.

### HasType

`func (o *PatchRepoCredentialsRequest) HasType() bool`

HasType returns a boolean if a field has been set.

### GetUsername

`func (o *PatchRepoCredentialsRequest) GetUsername() string`

GetUsername returns the Username field if non-nil, zero value otherwise.

### GetUsernameOk

`func (o *PatchRepoCredentialsRequest) GetUsernameOk() (*string, bool)`

GetUsernameOk returns a tuple with the Username field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUsername

`func (o *PatchRepoCredentialsRequest) SetUsername(v string)`

SetUsername sets Username field to given value.

### HasUsername

`func (o *PatchRepoCredentialsRequest) HasUsername() bool`

HasUsername returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


