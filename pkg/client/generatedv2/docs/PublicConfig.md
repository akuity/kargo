# PublicConfig

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**AdminAccountEnabled** | Pointer to **bool** |  | [optional] 
**OidcConfig** | Pointer to [**OIDCConfig**](OIDCConfig.md) |  | [optional] 
**SkipAuth** | Pointer to **bool** |  | [optional] 

## Methods

### NewPublicConfig

`func NewPublicConfig() *PublicConfig`

NewPublicConfig instantiates a new PublicConfig object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPublicConfigWithDefaults

`func NewPublicConfigWithDefaults() *PublicConfig`

NewPublicConfigWithDefaults instantiates a new PublicConfig object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAdminAccountEnabled

`func (o *PublicConfig) GetAdminAccountEnabled() bool`

GetAdminAccountEnabled returns the AdminAccountEnabled field if non-nil, zero value otherwise.

### GetAdminAccountEnabledOk

`func (o *PublicConfig) GetAdminAccountEnabledOk() (*bool, bool)`

GetAdminAccountEnabledOk returns a tuple with the AdminAccountEnabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAdminAccountEnabled

`func (o *PublicConfig) SetAdminAccountEnabled(v bool)`

SetAdminAccountEnabled sets AdminAccountEnabled field to given value.

### HasAdminAccountEnabled

`func (o *PublicConfig) HasAdminAccountEnabled() bool`

HasAdminAccountEnabled returns a boolean if a field has been set.

### GetOidcConfig

`func (o *PublicConfig) GetOidcConfig() OIDCConfig`

GetOidcConfig returns the OidcConfig field if non-nil, zero value otherwise.

### GetOidcConfigOk

`func (o *PublicConfig) GetOidcConfigOk() (*OIDCConfig, bool)`

GetOidcConfigOk returns a tuple with the OidcConfig field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOidcConfig

`func (o *PublicConfig) SetOidcConfig(v OIDCConfig)`

SetOidcConfig sets OidcConfig field to given value.

### HasOidcConfig

`func (o *PublicConfig) HasOidcConfig() bool`

HasOidcConfig returns a boolean if a field has been set.

### GetSkipAuth

`func (o *PublicConfig) GetSkipAuth() bool`

GetSkipAuth returns the SkipAuth field if non-nil, zero value otherwise.

### GetSkipAuthOk

`func (o *PublicConfig) GetSkipAuthOk() (*bool, bool)`

GetSkipAuthOk returns a tuple with the SkipAuth field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSkipAuth

`func (o *PublicConfig) SetSkipAuth(v bool)`

SetSkipAuth sets SkipAuth field to given value.

### HasSkipAuth

`func (o *PublicConfig) HasSkipAuth() bool`

HasSkipAuth returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


