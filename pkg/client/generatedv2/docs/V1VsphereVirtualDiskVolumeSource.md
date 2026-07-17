# V1VsphereVirtualDiskVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**FsType** | Pointer to **string** | fsType is filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. Implicitly inferred to be \&quot;ext4\&quot; if unspecified. +optional | [optional] 
**StoragePolicyID** | Pointer to **string** | storagePolicyID is the storage Policy Based Management (SPBM) profile ID associated with the StoragePolicyName. +optional | [optional] 
**StoragePolicyName** | Pointer to **string** | storagePolicyName is the storage Policy Based Management (SPBM) profile name. +optional | [optional] 
**VolumePath** | Pointer to **string** | volumePath is the path that identifies vSphere volume vmdk | [optional] 

## Methods

### NewV1VsphereVirtualDiskVolumeSource

`func NewV1VsphereVirtualDiskVolumeSource() *V1VsphereVirtualDiskVolumeSource`

NewV1VsphereVirtualDiskVolumeSource instantiates a new V1VsphereVirtualDiskVolumeSource object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1VsphereVirtualDiskVolumeSourceWithDefaults

`func NewV1VsphereVirtualDiskVolumeSourceWithDefaults() *V1VsphereVirtualDiskVolumeSource`

NewV1VsphereVirtualDiskVolumeSourceWithDefaults instantiates a new V1VsphereVirtualDiskVolumeSource object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetFsType

`func (o *V1VsphereVirtualDiskVolumeSource) GetFsType() string`

GetFsType returns the FsType field if non-nil, zero value otherwise.

### GetFsTypeOk

`func (o *V1VsphereVirtualDiskVolumeSource) GetFsTypeOk() (*string, bool)`

GetFsTypeOk returns a tuple with the FsType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFsType

`func (o *V1VsphereVirtualDiskVolumeSource) SetFsType(v string)`

SetFsType sets FsType field to given value.

### HasFsType

`func (o *V1VsphereVirtualDiskVolumeSource) HasFsType() bool`

HasFsType returns a boolean if a field has been set.

### GetStoragePolicyID

`func (o *V1VsphereVirtualDiskVolumeSource) GetStoragePolicyID() string`

GetStoragePolicyID returns the StoragePolicyID field if non-nil, zero value otherwise.

### GetStoragePolicyIDOk

`func (o *V1VsphereVirtualDiskVolumeSource) GetStoragePolicyIDOk() (*string, bool)`

GetStoragePolicyIDOk returns a tuple with the StoragePolicyID field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStoragePolicyID

`func (o *V1VsphereVirtualDiskVolumeSource) SetStoragePolicyID(v string)`

SetStoragePolicyID sets StoragePolicyID field to given value.

### HasStoragePolicyID

`func (o *V1VsphereVirtualDiskVolumeSource) HasStoragePolicyID() bool`

HasStoragePolicyID returns a boolean if a field has been set.

### GetStoragePolicyName

`func (o *V1VsphereVirtualDiskVolumeSource) GetStoragePolicyName() string`

GetStoragePolicyName returns the StoragePolicyName field if non-nil, zero value otherwise.

### GetStoragePolicyNameOk

`func (o *V1VsphereVirtualDiskVolumeSource) GetStoragePolicyNameOk() (*string, bool)`

GetStoragePolicyNameOk returns a tuple with the StoragePolicyName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStoragePolicyName

`func (o *V1VsphereVirtualDiskVolumeSource) SetStoragePolicyName(v string)`

SetStoragePolicyName sets StoragePolicyName field to given value.

### HasStoragePolicyName

`func (o *V1VsphereVirtualDiskVolumeSource) HasStoragePolicyName() bool`

HasStoragePolicyName returns a boolean if a field has been set.

### GetVolumePath

`func (o *V1VsphereVirtualDiskVolumeSource) GetVolumePath() string`

GetVolumePath returns the VolumePath field if non-nil, zero value otherwise.

### GetVolumePathOk

`func (o *V1VsphereVirtualDiskVolumeSource) GetVolumePathOk() (*string, bool)`

GetVolumePathOk returns a tuple with the VolumePath field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVolumePath

`func (o *V1VsphereVirtualDiskVolumeSource) SetVolumePath(v string)`

SetVolumePath sets VolumePath field to given value.

### HasVolumePath

`func (o *V1VsphereVirtualDiskVolumeSource) HasVolumePath() bool`

HasVolumePath returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


