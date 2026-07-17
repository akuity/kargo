# OIDCConfig

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**CliClientId** | Pointer to **string** |  | [optional] 
**ClientId** | Pointer to **string** |  | [optional] 
**IssuerUrl** | Pointer to **string** |  | [optional] 
**Scopes** | Pointer to **[]string** |  | [optional] 

## Methods

### NewOIDCConfig

`func NewOIDCConfig() *OIDCConfig`

NewOIDCConfig instantiates a new OIDCConfig object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewOIDCConfigWithDefaults

`func NewOIDCConfigWithDefaults() *OIDCConfig`

NewOIDCConfigWithDefaults instantiates a new OIDCConfig object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetCliClientId

`func (o *OIDCConfig) GetCliClientId() string`

GetCliClientId returns the CliClientId field if non-nil, zero value otherwise.

### GetCliClientIdOk

`func (o *OIDCConfig) GetCliClientIdOk() (*string, bool)`

GetCliClientIdOk returns a tuple with the CliClientId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCliClientId

`func (o *OIDCConfig) SetCliClientId(v string)`

SetCliClientId sets CliClientId field to given value.

### HasCliClientId

`func (o *OIDCConfig) HasCliClientId() bool`

HasCliClientId returns a boolean if a field has been set.

### GetClientId

`func (o *OIDCConfig) GetClientId() string`

GetClientId returns the ClientId field if non-nil, zero value otherwise.

### GetClientIdOk

`func (o *OIDCConfig) GetClientIdOk() (*string, bool)`

GetClientIdOk returns a tuple with the ClientId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetClientId

`func (o *OIDCConfig) SetClientId(v string)`

SetClientId sets ClientId field to given value.

### HasClientId

`func (o *OIDCConfig) HasClientId() bool`

HasClientId returns a boolean if a field has been set.

### GetIssuerUrl

`func (o *OIDCConfig) GetIssuerUrl() string`

GetIssuerUrl returns the IssuerUrl field if non-nil, zero value otherwise.

### GetIssuerUrlOk

`func (o *OIDCConfig) GetIssuerUrlOk() (*string, bool)`

GetIssuerUrlOk returns a tuple with the IssuerUrl field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIssuerUrl

`func (o *OIDCConfig) SetIssuerUrl(v string)`

SetIssuerUrl sets IssuerUrl field to given value.

### HasIssuerUrl

`func (o *OIDCConfig) HasIssuerUrl() bool`

HasIssuerUrl returns a boolean if a field has been set.

### GetScopes

`func (o *OIDCConfig) GetScopes() []string`

GetScopes returns the Scopes field if non-nil, zero value otherwise.

### GetScopesOk

`func (o *OIDCConfig) GetScopesOk() (*[]string, bool)`

GetScopesOk returns a tuple with the Scopes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetScopes

`func (o *OIDCConfig) SetScopes(v []string)`

SetScopes sets Scopes field to given value.

### HasScopes

`func (o *OIDCConfig) HasScopes() bool`

HasScopes returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


