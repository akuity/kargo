# V1GCEPersistentDiskVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**FsType** | Pointer to **string** | fsType is filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. Implicitly inferred to be \&quot;ext4\&quot; if unspecified. More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk TODO: how do we prevent errors in the filesystem from compromising the machine +optional | [optional] 
**Partition** | Pointer to **int32** | partition is the partition in the volume that you want to mount. If omitted, the default is to mount by volume name. Examples: For volume /dev/sda1, you specify the partition as \&quot;1\&quot;. Similarly, the volume partition for /dev/sda is \&quot;0\&quot; (or you can leave the property empty). More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk +optional | [optional] 
**PdName** | Pointer to **string** | pdName is unique name of the PD resource in GCE. Used to identify the disk in GCE. More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk | [optional] 
**ReadOnly** | Pointer to **bool** | readOnly here will force the ReadOnly setting in VolumeMounts. Defaults to false. More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk +optional | [optional] 

## Methods

### NewV1GCEPersistentDiskVolumeSource

`func NewV1GCEPersistentDiskVolumeSource() *V1GCEPersistentDiskVolumeSource`

NewV1GCEPersistentDiskVolumeSource instantiates a new V1GCEPersistentDiskVolumeSource object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1GCEPersistentDiskVolumeSourceWithDefaults

`func NewV1GCEPersistentDiskVolumeSourceWithDefaults() *V1GCEPersistentDiskVolumeSource`

NewV1GCEPersistentDiskVolumeSourceWithDefaults instantiates a new V1GCEPersistentDiskVolumeSource object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetFsType

`func (o *V1GCEPersistentDiskVolumeSource) GetFsType() string`

GetFsType returns the FsType field if non-nil, zero value otherwise.

### GetFsTypeOk

`func (o *V1GCEPersistentDiskVolumeSource) GetFsTypeOk() (*string, bool)`

GetFsTypeOk returns a tuple with the FsType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFsType

`func (o *V1GCEPersistentDiskVolumeSource) SetFsType(v string)`

SetFsType sets FsType field to given value.

### HasFsType

`func (o *V1GCEPersistentDiskVolumeSource) HasFsType() bool`

HasFsType returns a boolean if a field has been set.

### GetPartition

`func (o *V1GCEPersistentDiskVolumeSource) GetPartition() int32`

GetPartition returns the Partition field if non-nil, zero value otherwise.

### GetPartitionOk

`func (o *V1GCEPersistentDiskVolumeSource) GetPartitionOk() (*int32, bool)`

GetPartitionOk returns a tuple with the Partition field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPartition

`func (o *V1GCEPersistentDiskVolumeSource) SetPartition(v int32)`

SetPartition sets Partition field to given value.

### HasPartition

`func (o *V1GCEPersistentDiskVolumeSource) HasPartition() bool`

HasPartition returns a boolean if a field has been set.

### GetPdName

`func (o *V1GCEPersistentDiskVolumeSource) GetPdName() string`

GetPdName returns the PdName field if non-nil, zero value otherwise.

### GetPdNameOk

`func (o *V1GCEPersistentDiskVolumeSource) GetPdNameOk() (*string, bool)`

GetPdNameOk returns a tuple with the PdName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPdName

`func (o *V1GCEPersistentDiskVolumeSource) SetPdName(v string)`

SetPdName sets PdName field to given value.

### HasPdName

`func (o *V1GCEPersistentDiskVolumeSource) HasPdName() bool`

HasPdName returns a boolean if a field has been set.

### GetReadOnly

`func (o *V1GCEPersistentDiskVolumeSource) GetReadOnly() bool`

GetReadOnly returns the ReadOnly field if non-nil, zero value otherwise.

### GetReadOnlyOk

`func (o *V1GCEPersistentDiskVolumeSource) GetReadOnlyOk() (*bool, bool)`

GetReadOnlyOk returns a tuple with the ReadOnly field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReadOnly

`func (o *V1GCEPersistentDiskVolumeSource) SetReadOnly(v bool)`

SetReadOnly sets ReadOnly field to given value.

### HasReadOnly

`func (o *V1GCEPersistentDiskVolumeSource) HasReadOnly() bool`

HasReadOnly returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


