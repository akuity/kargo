# V1SELinuxOptions

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Level** | Pointer to **string** | Level is SELinux level label that applies to the container. +optional | [optional] 
**Role** | Pointer to **string** | Role is a SELinux role label that applies to the container. +optional | [optional] 
**Type** | Pointer to **string** | Type is a SELinux type label that applies to the container. +optional | [optional] 
**User** | Pointer to **string** | User is a SELinux user label that applies to the container. +optional | [optional] 

## Methods

### NewV1SELinuxOptions

`func NewV1SELinuxOptions() *V1SELinuxOptions`

NewV1SELinuxOptions instantiates a new V1SELinuxOptions object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1SELinuxOptionsWithDefaults

`func NewV1SELinuxOptionsWithDefaults() *V1SELinuxOptions`

NewV1SELinuxOptionsWithDefaults instantiates a new V1SELinuxOptions object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetLevel

`func (o *V1SELinuxOptions) GetLevel() string`

GetLevel returns the Level field if non-nil, zero value otherwise.

### GetLevelOk

`func (o *V1SELinuxOptions) GetLevelOk() (*string, bool)`

GetLevelOk returns a tuple with the Level field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLevel

`func (o *V1SELinuxOptions) SetLevel(v string)`

SetLevel sets Level field to given value.

### HasLevel

`func (o *V1SELinuxOptions) HasLevel() bool`

HasLevel returns a boolean if a field has been set.

### GetRole

`func (o *V1SELinuxOptions) GetRole() string`

GetRole returns the Role field if non-nil, zero value otherwise.

### GetRoleOk

`func (o *V1SELinuxOptions) GetRoleOk() (*string, bool)`

GetRoleOk returns a tuple with the Role field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRole

`func (o *V1SELinuxOptions) SetRole(v string)`

SetRole sets Role field to given value.

### HasRole

`func (o *V1SELinuxOptions) HasRole() bool`

HasRole returns a boolean if a field has been set.

### GetType

`func (o *V1SELinuxOptions) GetType() string`

GetType returns the Type field if non-nil, zero value otherwise.

### GetTypeOk

`func (o *V1SELinuxOptions) GetTypeOk() (*string, bool)`

GetTypeOk returns a tuple with the Type field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetType

`func (o *V1SELinuxOptions) SetType(v string)`

SetType sets Type field to given value.

### HasType

`func (o *V1SELinuxOptions) HasType() bool`

HasType returns a boolean if a field has been set.

### GetUser

`func (o *V1SELinuxOptions) GetUser() string`

GetUser returns the User field if non-nil, zero value otherwise.

### GetUserOk

`func (o *V1SELinuxOptions) GetUserOk() (*string, bool)`

GetUserOk returns a tuple with the User field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUser

`func (o *V1SELinuxOptions) SetUser(v string)`

SetUser sets User field to given value.

### HasUser

`func (o *V1SELinuxOptions) HasUser() bool`

HasUser returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


