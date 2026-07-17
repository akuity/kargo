# V1HostPathVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Path** | Pointer to **string** | path of the directory on the host. If the path is a symlink, it will follow the link to the real path. More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath | [optional] 
**Type** | Pointer to **string** | type for HostPath Volume Defaults to \&quot;\&quot; More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath +optional | [optional] 

## Methods

### NewV1HostPathVolumeSource

`func NewV1HostPathVolumeSource() *V1HostPathVolumeSource`

NewV1HostPathVolumeSource instantiates a new V1HostPathVolumeSource object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1HostPathVolumeSourceWithDefaults

`func NewV1HostPathVolumeSourceWithDefaults() *V1HostPathVolumeSource`

NewV1HostPathVolumeSourceWithDefaults instantiates a new V1HostPathVolumeSource object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetPath

`func (o *V1HostPathVolumeSource) GetPath() string`

GetPath returns the Path field if non-nil, zero value otherwise.

### GetPathOk

`func (o *V1HostPathVolumeSource) GetPathOk() (*string, bool)`

GetPathOk returns a tuple with the Path field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPath

`func (o *V1HostPathVolumeSource) SetPath(v string)`

SetPath sets Path field to given value.

### HasPath

`func (o *V1HostPathVolumeSource) HasPath() bool`

HasPath returns a boolean if a field has been set.

### GetType

`func (o *V1HostPathVolumeSource) GetType() string`

GetType returns the Type field if non-nil, zero value otherwise.

### GetTypeOk

`func (o *V1HostPathVolumeSource) GetTypeOk() (*string, bool)`

GetTypeOk returns a tuple with the Type field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetType

`func (o *V1HostPathVolumeSource) SetType(v string)`

SetType sets Type field to given value.

### HasType

`func (o *V1HostPathVolumeSource) HasType() bool`

HasType returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


