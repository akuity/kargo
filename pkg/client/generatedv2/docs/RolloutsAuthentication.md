# RolloutsAuthentication

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Oauth2** | Pointer to [**RolloutsOAuth2Config**](RolloutsOAuth2Config.md) |  | [optional] 
**Sigv4** | Pointer to [**RolloutsSigv4Config**](RolloutsSigv4Config.md) |  | [optional] 

## Methods

### NewRolloutsAuthentication

`func NewRolloutsAuthentication() *RolloutsAuthentication`

NewRolloutsAuthentication instantiates a new RolloutsAuthentication object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRolloutsAuthenticationWithDefaults

`func NewRolloutsAuthenticationWithDefaults() *RolloutsAuthentication`

NewRolloutsAuthenticationWithDefaults instantiates a new RolloutsAuthentication object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetOauth2

`func (o *RolloutsAuthentication) GetOauth2() RolloutsOAuth2Config`

GetOauth2 returns the Oauth2 field if non-nil, zero value otherwise.

### GetOauth2Ok

`func (o *RolloutsAuthentication) GetOauth2Ok() (*RolloutsOAuth2Config, bool)`

GetOauth2Ok returns a tuple with the Oauth2 field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOauth2

`func (o *RolloutsAuthentication) SetOauth2(v RolloutsOAuth2Config)`

SetOauth2 sets Oauth2 field to given value.

### HasOauth2

`func (o *RolloutsAuthentication) HasOauth2() bool`

HasOauth2 returns a boolean if a field has been set.

### GetSigv4

`func (o *RolloutsAuthentication) GetSigv4() RolloutsSigv4Config`

GetSigv4 returns the Sigv4 field if non-nil, zero value otherwise.

### GetSigv4Ok

`func (o *RolloutsAuthentication) GetSigv4Ok() (*RolloutsSigv4Config, bool)`

GetSigv4Ok returns a tuple with the Sigv4 field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSigv4

`func (o *RolloutsAuthentication) SetSigv4(v RolloutsSigv4Config)`

SetSigv4 sets Sigv4 field to given value.

### HasSigv4

`func (o *RolloutsAuthentication) HasSigv4() bool`

HasSigv4 returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


