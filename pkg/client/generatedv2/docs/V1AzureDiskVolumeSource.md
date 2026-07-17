# V1AzureDiskVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**CachingMode** | Pointer to **string** | cachingMode is the Host Caching mode: None, Read Only, Read Write. +optional +default&#x3D;ref(AzureDataDiskCachingReadWrite) | [optional] 
**DiskName** | Pointer to **string** | diskName is the Name of the data disk in the blob storage | [optional] 
**DiskURI** | Pointer to **string** | diskURI is the URI of data disk in the blob storage | [optional] 
**FsType** | Pointer to **string** | fsType is Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. Implicitly inferred to be \&quot;ext4\&quot; if unspecified. +optional +default&#x3D;\&quot;ext4\&quot; | [optional] 
**Kind** | Pointer to **string** | kind expected values are Shared: multiple blob disks per storage account  Dedicated: single blob disk per storage account  Managed: azure managed data disk (only in managed availability set). defaults to shared +default&#x3D;ref(AzureSharedBlobDisk) | [optional] 
**ReadOnly** | Pointer to **bool** | readOnly Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. +optional +default&#x3D;false | [optional] 

## Methods

### NewV1AzureDiskVolumeSource

`func NewV1AzureDiskVolumeSource() *V1AzureDiskVolumeSource`

NewV1AzureDiskVolumeSource instantiates a new V1AzureDiskVolumeSource object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1AzureDiskVolumeSourceWithDefaults

`func NewV1AzureDiskVolumeSourceWithDefaults() *V1AzureDiskVolumeSource`

NewV1AzureDiskVolumeSourceWithDefaults instantiates a new V1AzureDiskVolumeSource object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetCachingMode

`func (o *V1AzureDiskVolumeSource) GetCachingMode() string`

GetCachingMode returns the CachingMode field if non-nil, zero value otherwise.

### GetCachingModeOk

`func (o *V1AzureDiskVolumeSource) GetCachingModeOk() (*string, bool)`

GetCachingModeOk returns a tuple with the CachingMode field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCachingMode

`func (o *V1AzureDiskVolumeSource) SetCachingMode(v string)`

SetCachingMode sets CachingMode field to given value.

### HasCachingMode

`func (o *V1AzureDiskVolumeSource) HasCachingMode() bool`

HasCachingMode returns a boolean if a field has been set.

### GetDiskName

`func (o *V1AzureDiskVolumeSource) GetDiskName() string`

GetDiskName returns the DiskName field if non-nil, zero value otherwise.

### GetDiskNameOk

`func (o *V1AzureDiskVolumeSource) GetDiskNameOk() (*string, bool)`

GetDiskNameOk returns a tuple with the DiskName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDiskName

`func (o *V1AzureDiskVolumeSource) SetDiskName(v string)`

SetDiskName sets DiskName field to given value.

### HasDiskName

`func (o *V1AzureDiskVolumeSource) HasDiskName() bool`

HasDiskName returns a boolean if a field has been set.

### GetDiskURI

`func (o *V1AzureDiskVolumeSource) GetDiskURI() string`

GetDiskURI returns the DiskURI field if non-nil, zero value otherwise.

### GetDiskURIOk

`func (o *V1AzureDiskVolumeSource) GetDiskURIOk() (*string, bool)`

GetDiskURIOk returns a tuple with the DiskURI field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDiskURI

`func (o *V1AzureDiskVolumeSource) SetDiskURI(v string)`

SetDiskURI sets DiskURI field to given value.

### HasDiskURI

`func (o *V1AzureDiskVolumeSource) HasDiskURI() bool`

HasDiskURI returns a boolean if a field has been set.

### GetFsType

`func (o *V1AzureDiskVolumeSource) GetFsType() string`

GetFsType returns the FsType field if non-nil, zero value otherwise.

### GetFsTypeOk

`func (o *V1AzureDiskVolumeSource) GetFsTypeOk() (*string, bool)`

GetFsTypeOk returns a tuple with the FsType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFsType

`func (o *V1AzureDiskVolumeSource) SetFsType(v string)`

SetFsType sets FsType field to given value.

### HasFsType

`func (o *V1AzureDiskVolumeSource) HasFsType() bool`

HasFsType returns a boolean if a field has been set.

### GetKind

`func (o *V1AzureDiskVolumeSource) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *V1AzureDiskVolumeSource) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *V1AzureDiskVolumeSource) SetKind(v string)`

SetKind sets Kind field to given value.

### HasKind

`func (o *V1AzureDiskVolumeSource) HasKind() bool`

HasKind returns a boolean if a field has been set.

### GetReadOnly

`func (o *V1AzureDiskVolumeSource) GetReadOnly() bool`

GetReadOnly returns the ReadOnly field if non-nil, zero value otherwise.

### GetReadOnlyOk

`func (o *V1AzureDiskVolumeSource) GetReadOnlyOk() (*bool, bool)`

GetReadOnlyOk returns a tuple with the ReadOnly field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReadOnly

`func (o *V1AzureDiskVolumeSource) SetReadOnly(v bool)`

SetReadOnly sets ReadOnly field to given value.

### HasReadOnly

`func (o *V1AzureDiskVolumeSource) HasReadOnly() bool`

HasReadOnly returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


