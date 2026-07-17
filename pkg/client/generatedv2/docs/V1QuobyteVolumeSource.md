# V1QuobyteVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Group** | Pointer to **string** | group to map volume access to Default is no group +optional | [optional] 
**ReadOnly** | Pointer to **bool** | readOnly here will force the Quobyte volume to be mounted with read-only permissions. Defaults to false. +optional | [optional] 
**Registry** | Pointer to **string** | registry represents a single or multiple Quobyte Registry services specified as a string as host:port pair (multiple entries are separated with commas) which acts as the central registry for volumes | [optional] 
**Tenant** | Pointer to **string** | tenant owning the given Quobyte volume in the Backend Used with dynamically provisioned Quobyte volumes, value is set by the plugin +optional | [optional] 
**User** | Pointer to **string** | user to map volume access to Defaults to serivceaccount user +optional | [optional] 
**Volume** | Pointer to **string** | volume is a string that references an already created Quobyte volume by name. | [optional] 

## Methods

### NewV1QuobyteVolumeSource

`func NewV1QuobyteVolumeSource() *V1QuobyteVolumeSource`

NewV1QuobyteVolumeSource instantiates a new V1QuobyteVolumeSource object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1QuobyteVolumeSourceWithDefaults

`func NewV1QuobyteVolumeSourceWithDefaults() *V1QuobyteVolumeSource`

NewV1QuobyteVolumeSourceWithDefaults instantiates a new V1QuobyteVolumeSource object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetGroup

`func (o *V1QuobyteVolumeSource) GetGroup() string`

GetGroup returns the Group field if non-nil, zero value otherwise.

### GetGroupOk

`func (o *V1QuobyteVolumeSource) GetGroupOk() (*string, bool)`

GetGroupOk returns a tuple with the Group field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetGroup

`func (o *V1QuobyteVolumeSource) SetGroup(v string)`

SetGroup sets Group field to given value.

### HasGroup

`func (o *V1QuobyteVolumeSource) HasGroup() bool`

HasGroup returns a boolean if a field has been set.

### GetReadOnly

`func (o *V1QuobyteVolumeSource) GetReadOnly() bool`

GetReadOnly returns the ReadOnly field if non-nil, zero value otherwise.

### GetReadOnlyOk

`func (o *V1QuobyteVolumeSource) GetReadOnlyOk() (*bool, bool)`

GetReadOnlyOk returns a tuple with the ReadOnly field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReadOnly

`func (o *V1QuobyteVolumeSource) SetReadOnly(v bool)`

SetReadOnly sets ReadOnly field to given value.

### HasReadOnly

`func (o *V1QuobyteVolumeSource) HasReadOnly() bool`

HasReadOnly returns a boolean if a field has been set.

### GetRegistry

`func (o *V1QuobyteVolumeSource) GetRegistry() string`

GetRegistry returns the Registry field if non-nil, zero value otherwise.

### GetRegistryOk

`func (o *V1QuobyteVolumeSource) GetRegistryOk() (*string, bool)`

GetRegistryOk returns a tuple with the Registry field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRegistry

`func (o *V1QuobyteVolumeSource) SetRegistry(v string)`

SetRegistry sets Registry field to given value.

### HasRegistry

`func (o *V1QuobyteVolumeSource) HasRegistry() bool`

HasRegistry returns a boolean if a field has been set.

### GetTenant

`func (o *V1QuobyteVolumeSource) GetTenant() string`

GetTenant returns the Tenant field if non-nil, zero value otherwise.

### GetTenantOk

`func (o *V1QuobyteVolumeSource) GetTenantOk() (*string, bool)`

GetTenantOk returns a tuple with the Tenant field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTenant

`func (o *V1QuobyteVolumeSource) SetTenant(v string)`

SetTenant sets Tenant field to given value.

### HasTenant

`func (o *V1QuobyteVolumeSource) HasTenant() bool`

HasTenant returns a boolean if a field has been set.

### GetUser

`func (o *V1QuobyteVolumeSource) GetUser() string`

GetUser returns the User field if non-nil, zero value otherwise.

### GetUserOk

`func (o *V1QuobyteVolumeSource) GetUserOk() (*string, bool)`

GetUserOk returns a tuple with the User field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUser

`func (o *V1QuobyteVolumeSource) SetUser(v string)`

SetUser sets User field to given value.

### HasUser

`func (o *V1QuobyteVolumeSource) HasUser() bool`

HasUser returns a boolean if a field has been set.

### GetVolume

`func (o *V1QuobyteVolumeSource) GetVolume() string`

GetVolume returns the Volume field if non-nil, zero value otherwise.

### GetVolumeOk

`func (o *V1QuobyteVolumeSource) GetVolumeOk() (*string, bool)`

GetVolumeOk returns a tuple with the Volume field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVolume

`func (o *V1QuobyteVolumeSource) SetVolume(v string)`

SetVolume sets Volume field to given value.

### HasVolume

`func (o *V1QuobyteVolumeSource) HasVolume() bool`

HasVolume returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


