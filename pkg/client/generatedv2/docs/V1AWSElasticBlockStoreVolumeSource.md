# V1AWSElasticBlockStoreVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**FsType** | Pointer to **string** | fsType is the filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. Implicitly inferred to be \&quot;ext4\&quot; if unspecified. More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore TODO: how do we prevent errors in the filesystem from compromising the machine +optional | [optional] 
**Partition** | Pointer to **int32** | partition is the partition in the volume that you want to mount. If omitted, the default is to mount by volume name. Examples: For volume /dev/sda1, you specify the partition as \&quot;1\&quot;. Similarly, the volume partition for /dev/sda is \&quot;0\&quot; (or you can leave the property empty). +optional | [optional] 
**ReadOnly** | Pointer to **bool** | readOnly value true will force the readOnly setting in VolumeMounts. More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore +optional | [optional] 
**VolumeID** | Pointer to **string** | volumeID is unique ID of the persistent disk resource in AWS (Amazon EBS volume). More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore | [optional] 

## Methods

### NewV1AWSElasticBlockStoreVolumeSource

`func NewV1AWSElasticBlockStoreVolumeSource() *V1AWSElasticBlockStoreVolumeSource`

NewV1AWSElasticBlockStoreVolumeSource instantiates a new V1AWSElasticBlockStoreVolumeSource object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1AWSElasticBlockStoreVolumeSourceWithDefaults

`func NewV1AWSElasticBlockStoreVolumeSourceWithDefaults() *V1AWSElasticBlockStoreVolumeSource`

NewV1AWSElasticBlockStoreVolumeSourceWithDefaults instantiates a new V1AWSElasticBlockStoreVolumeSource object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetFsType

`func (o *V1AWSElasticBlockStoreVolumeSource) GetFsType() string`

GetFsType returns the FsType field if non-nil, zero value otherwise.

### GetFsTypeOk

`func (o *V1AWSElasticBlockStoreVolumeSource) GetFsTypeOk() (*string, bool)`

GetFsTypeOk returns a tuple with the FsType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFsType

`func (o *V1AWSElasticBlockStoreVolumeSource) SetFsType(v string)`

SetFsType sets FsType field to given value.

### HasFsType

`func (o *V1AWSElasticBlockStoreVolumeSource) HasFsType() bool`

HasFsType returns a boolean if a field has been set.

### GetPartition

`func (o *V1AWSElasticBlockStoreVolumeSource) GetPartition() int32`

GetPartition returns the Partition field if non-nil, zero value otherwise.

### GetPartitionOk

`func (o *V1AWSElasticBlockStoreVolumeSource) GetPartitionOk() (*int32, bool)`

GetPartitionOk returns a tuple with the Partition field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPartition

`func (o *V1AWSElasticBlockStoreVolumeSource) SetPartition(v int32)`

SetPartition sets Partition field to given value.

### HasPartition

`func (o *V1AWSElasticBlockStoreVolumeSource) HasPartition() bool`

HasPartition returns a boolean if a field has been set.

### GetReadOnly

`func (o *V1AWSElasticBlockStoreVolumeSource) GetReadOnly() bool`

GetReadOnly returns the ReadOnly field if non-nil, zero value otherwise.

### GetReadOnlyOk

`func (o *V1AWSElasticBlockStoreVolumeSource) GetReadOnlyOk() (*bool, bool)`

GetReadOnlyOk returns a tuple with the ReadOnly field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReadOnly

`func (o *V1AWSElasticBlockStoreVolumeSource) SetReadOnly(v bool)`

SetReadOnly sets ReadOnly field to given value.

### HasReadOnly

`func (o *V1AWSElasticBlockStoreVolumeSource) HasReadOnly() bool`

HasReadOnly returns a boolean if a field has been set.

### GetVolumeID

`func (o *V1AWSElasticBlockStoreVolumeSource) GetVolumeID() string`

GetVolumeID returns the VolumeID field if non-nil, zero value otherwise.

### GetVolumeIDOk

`func (o *V1AWSElasticBlockStoreVolumeSource) GetVolumeIDOk() (*string, bool)`

GetVolumeIDOk returns a tuple with the VolumeID field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVolumeID

`func (o *V1AWSElasticBlockStoreVolumeSource) SetVolumeID(v string)`

SetVolumeID sets VolumeID field to given value.

### HasVolumeID

`func (o *V1AWSElasticBlockStoreVolumeSource) HasVolumeID() bool`

HasVolumeID returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


