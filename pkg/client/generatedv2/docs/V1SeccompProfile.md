# V1SeccompProfile

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**LocalhostProfile** | Pointer to **string** | localhostProfile indicates a profile defined in a file on the node should be used. The profile must be preconfigured on the node to work. Must be a descending path, relative to the kubelet&#39;s configured seccomp profile location. Must be set if type is \&quot;Localhost\&quot;. Must NOT be set for any other type. +optional | [optional] 
**Type** | Pointer to **string** | type indicates which kind of seccomp profile will be applied. Valid options are:  Localhost - a profile defined in a file on the node should be used. RuntimeDefault - the container runtime default profile should be used. Unconfined - no profile should be applied. +unionDiscriminator | [optional] 

## Methods

### NewV1SeccompProfile

`func NewV1SeccompProfile() *V1SeccompProfile`

NewV1SeccompProfile instantiates a new V1SeccompProfile object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1SeccompProfileWithDefaults

`func NewV1SeccompProfileWithDefaults() *V1SeccompProfile`

NewV1SeccompProfileWithDefaults instantiates a new V1SeccompProfile object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetLocalhostProfile

`func (o *V1SeccompProfile) GetLocalhostProfile() string`

GetLocalhostProfile returns the LocalhostProfile field if non-nil, zero value otherwise.

### GetLocalhostProfileOk

`func (o *V1SeccompProfile) GetLocalhostProfileOk() (*string, bool)`

GetLocalhostProfileOk returns a tuple with the LocalhostProfile field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLocalhostProfile

`func (o *V1SeccompProfile) SetLocalhostProfile(v string)`

SetLocalhostProfile sets LocalhostProfile field to given value.

### HasLocalhostProfile

`func (o *V1SeccompProfile) HasLocalhostProfile() bool`

HasLocalhostProfile returns a boolean if a field has been set.

### GetType

`func (o *V1SeccompProfile) GetType() string`

GetType returns the Type field if non-nil, zero value otherwise.

### GetTypeOk

`func (o *V1SeccompProfile) GetTypeOk() (*string, bool)`

GetTypeOk returns a tuple with the Type field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetType

`func (o *V1SeccompProfile) SetType(v string)`

SetType sets Type field to given value.

### HasType

`func (o *V1SeccompProfile) HasType() bool`

HasType returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


