# V1AppArmorProfile

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**LocalhostProfile** | Pointer to **string** | localhostProfile indicates a profile loaded on the node that should be used. The profile must be preconfigured on the node to work. Must match the loaded name of the profile. Must be set if and only if type is \&quot;Localhost\&quot;. +optional | [optional] 
**Type** | Pointer to **string** | type indicates which kind of AppArmor profile will be applied. Valid options are:   Localhost - a profile pre-loaded on the node.   RuntimeDefault - the container runtime&#39;s default profile.   Unconfined - no AppArmor enforcement. +unionDiscriminator | [optional] 

## Methods

### NewV1AppArmorProfile

`func NewV1AppArmorProfile() *V1AppArmorProfile`

NewV1AppArmorProfile instantiates a new V1AppArmorProfile object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1AppArmorProfileWithDefaults

`func NewV1AppArmorProfileWithDefaults() *V1AppArmorProfile`

NewV1AppArmorProfileWithDefaults instantiates a new V1AppArmorProfile object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetLocalhostProfile

`func (o *V1AppArmorProfile) GetLocalhostProfile() string`

GetLocalhostProfile returns the LocalhostProfile field if non-nil, zero value otherwise.

### GetLocalhostProfileOk

`func (o *V1AppArmorProfile) GetLocalhostProfileOk() (*string, bool)`

GetLocalhostProfileOk returns a tuple with the LocalhostProfile field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLocalhostProfile

`func (o *V1AppArmorProfile) SetLocalhostProfile(v string)`

SetLocalhostProfile sets LocalhostProfile field to given value.

### HasLocalhostProfile

`func (o *V1AppArmorProfile) HasLocalhostProfile() bool`

HasLocalhostProfile returns a boolean if a field has been set.

### GetType

`func (o *V1AppArmorProfile) GetType() string`

GetType returns the Type field if non-nil, zero value otherwise.

### GetTypeOk

`func (o *V1AppArmorProfile) GetTypeOk() (*string, bool)`

GetTypeOk returns a tuple with the Type field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetType

`func (o *V1AppArmorProfile) SetType(v string)`

SetType sets Type field to given value.

### HasType

`func (o *V1AppArmorProfile) HasType() bool`

HasType returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


