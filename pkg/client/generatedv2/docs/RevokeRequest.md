# RevokeRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ResourceDetails** | Pointer to [**ResourceDetails**](ResourceDetails.md) |  | [optional] 
**Role** | Pointer to **string** |  | [optional] 
**UserClaims** | Pointer to [**UserClaims**](UserClaims.md) |  | [optional] 

## Methods

### NewRevokeRequest

`func NewRevokeRequest() *RevokeRequest`

NewRevokeRequest instantiates a new RevokeRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRevokeRequestWithDefaults

`func NewRevokeRequestWithDefaults() *RevokeRequest`

NewRevokeRequestWithDefaults instantiates a new RevokeRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetResourceDetails

`func (o *RevokeRequest) GetResourceDetails() ResourceDetails`

GetResourceDetails returns the ResourceDetails field if non-nil, zero value otherwise.

### GetResourceDetailsOk

`func (o *RevokeRequest) GetResourceDetailsOk() (*ResourceDetails, bool)`

GetResourceDetailsOk returns a tuple with the ResourceDetails field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResourceDetails

`func (o *RevokeRequest) SetResourceDetails(v ResourceDetails)`

SetResourceDetails sets ResourceDetails field to given value.

### HasResourceDetails

`func (o *RevokeRequest) HasResourceDetails() bool`

HasResourceDetails returns a boolean if a field has been set.

### GetRole

`func (o *RevokeRequest) GetRole() string`

GetRole returns the Role field if non-nil, zero value otherwise.

### GetRoleOk

`func (o *RevokeRequest) GetRoleOk() (*string, bool)`

GetRoleOk returns a tuple with the Role field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRole

`func (o *RevokeRequest) SetRole(v string)`

SetRole sets Role field to given value.

### HasRole

`func (o *RevokeRequest) HasRole() bool`

HasRole returns a boolean if a field has been set.

### GetUserClaims

`func (o *RevokeRequest) GetUserClaims() UserClaims`

GetUserClaims returns the UserClaims field if non-nil, zero value otherwise.

### GetUserClaimsOk

`func (o *RevokeRequest) GetUserClaimsOk() (*UserClaims, bool)`

GetUserClaimsOk returns a tuple with the UserClaims field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUserClaims

`func (o *RevokeRequest) SetUserClaims(v UserClaims)`

SetUserClaims sets UserClaims field to given value.

### HasUserClaims

`func (o *RevokeRequest) HasUserClaims() bool`

HasUserClaims returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


