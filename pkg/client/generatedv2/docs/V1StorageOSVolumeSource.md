# V1StorageOSVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**FsType** | Pointer to **string** | fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. Implicitly inferred to be \&quot;ext4\&quot; if unspecified. +optional | [optional] 
**ReadOnly** | Pointer to **bool** | readOnly defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. +optional | [optional] 
**SecretRef** | Pointer to [**V1LocalObjectReference**](V1LocalObjectReference.md) | secretRef specifies the secret to use for obtaining the StorageOS API credentials.  If not specified, default values will be attempted. +optional | [optional] 
**VolumeName** | Pointer to **string** | volumeName is the human-readable name of the StorageOS volume.  Volume names are only unique within a namespace. | [optional] 
**VolumeNamespace** | Pointer to **string** | volumeNamespace specifies the scope of the volume within StorageOS.  If no namespace is specified then the Pod&#39;s namespace will be used.  This allows the Kubernetes name scoping to be mirrored within StorageOS for tighter integration. Set VolumeName to any name to override the default behaviour. Set to \&quot;default\&quot; if you are not using namespaces within StorageOS. Namespaces that do not pre-exist within StorageOS will be created. +optional | [optional] 

## Methods

### NewV1StorageOSVolumeSource

`func NewV1StorageOSVolumeSource() *V1StorageOSVolumeSource`

NewV1StorageOSVolumeSource instantiates a new V1StorageOSVolumeSource object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1StorageOSVolumeSourceWithDefaults

`func NewV1StorageOSVolumeSourceWithDefaults() *V1StorageOSVolumeSource`

NewV1StorageOSVolumeSourceWithDefaults instantiates a new V1StorageOSVolumeSource object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetFsType

`func (o *V1StorageOSVolumeSource) GetFsType() string`

GetFsType returns the FsType field if non-nil, zero value otherwise.

### GetFsTypeOk

`func (o *V1StorageOSVolumeSource) GetFsTypeOk() (*string, bool)`

GetFsTypeOk returns a tuple with the FsType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFsType

`func (o *V1StorageOSVolumeSource) SetFsType(v string)`

SetFsType sets FsType field to given value.

### HasFsType

`func (o *V1StorageOSVolumeSource) HasFsType() bool`

HasFsType returns a boolean if a field has been set.

### GetReadOnly

`func (o *V1StorageOSVolumeSource) GetReadOnly() bool`

GetReadOnly returns the ReadOnly field if non-nil, zero value otherwise.

### GetReadOnlyOk

`func (o *V1StorageOSVolumeSource) GetReadOnlyOk() (*bool, bool)`

GetReadOnlyOk returns a tuple with the ReadOnly field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReadOnly

`func (o *V1StorageOSVolumeSource) SetReadOnly(v bool)`

SetReadOnly sets ReadOnly field to given value.

### HasReadOnly

`func (o *V1StorageOSVolumeSource) HasReadOnly() bool`

HasReadOnly returns a boolean if a field has been set.

### GetSecretRef

`func (o *V1StorageOSVolumeSource) GetSecretRef() V1LocalObjectReference`

GetSecretRef returns the SecretRef field if non-nil, zero value otherwise.

### GetSecretRefOk

`func (o *V1StorageOSVolumeSource) GetSecretRefOk() (*V1LocalObjectReference, bool)`

GetSecretRefOk returns a tuple with the SecretRef field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecretRef

`func (o *V1StorageOSVolumeSource) SetSecretRef(v V1LocalObjectReference)`

SetSecretRef sets SecretRef field to given value.

### HasSecretRef

`func (o *V1StorageOSVolumeSource) HasSecretRef() bool`

HasSecretRef returns a boolean if a field has been set.

### GetVolumeName

`func (o *V1StorageOSVolumeSource) GetVolumeName() string`

GetVolumeName returns the VolumeName field if non-nil, zero value otherwise.

### GetVolumeNameOk

`func (o *V1StorageOSVolumeSource) GetVolumeNameOk() (*string, bool)`

GetVolumeNameOk returns a tuple with the VolumeName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVolumeName

`func (o *V1StorageOSVolumeSource) SetVolumeName(v string)`

SetVolumeName sets VolumeName field to given value.

### HasVolumeName

`func (o *V1StorageOSVolumeSource) HasVolumeName() bool`

HasVolumeName returns a boolean if a field has been set.

### GetVolumeNamespace

`func (o *V1StorageOSVolumeSource) GetVolumeNamespace() string`

GetVolumeNamespace returns the VolumeNamespace field if non-nil, zero value otherwise.

### GetVolumeNamespaceOk

`func (o *V1StorageOSVolumeSource) GetVolumeNamespaceOk() (*string, bool)`

GetVolumeNamespaceOk returns a tuple with the VolumeNamespace field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVolumeNamespace

`func (o *V1StorageOSVolumeSource) SetVolumeNamespace(v string)`

SetVolumeNamespace sets VolumeNamespace field to given value.

### HasVolumeNamespace

`func (o *V1StorageOSVolumeSource) HasVolumeNamespace() bool`

HasVolumeNamespace returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


