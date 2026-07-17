# V1HostAlias

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Hostnames** | Pointer to **[]string** | Hostnames for the above IP address. +listType&#x3D;atomic | [optional] 
**Ip** | Pointer to **string** | IP address of the host file entry. +required | [optional] 

## Methods

### NewV1HostAlias

`func NewV1HostAlias() *V1HostAlias`

NewV1HostAlias instantiates a new V1HostAlias object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1HostAliasWithDefaults

`func NewV1HostAliasWithDefaults() *V1HostAlias`

NewV1HostAliasWithDefaults instantiates a new V1HostAlias object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetHostnames

`func (o *V1HostAlias) GetHostnames() []string`

GetHostnames returns the Hostnames field if non-nil, zero value otherwise.

### GetHostnamesOk

`func (o *V1HostAlias) GetHostnamesOk() (*[]string, bool)`

GetHostnamesOk returns a tuple with the Hostnames field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHostnames

`func (o *V1HostAlias) SetHostnames(v []string)`

SetHostnames sets Hostnames field to given value.

### HasHostnames

`func (o *V1HostAlias) HasHostnames() bool`

HasHostnames returns a boolean if a field has been set.

### GetIp

`func (o *V1HostAlias) GetIp() string`

GetIp returns the Ip field if non-nil, zero value otherwise.

### GetIpOk

`func (o *V1HostAlias) GetIpOk() (*string, bool)`

GetIpOk returns a tuple with the Ip field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIp

`func (o *V1HostAlias) SetIp(v string)`

SetIp sets Ip field to given value.

### HasIp

`func (o *V1HostAlias) HasIp() bool`

HasIp returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


