# V1FlexVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Driver** | Pointer to **string** | driver is the name of the driver to use for this volume. | [optional] 
**FsType** | Pointer to **string** | fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. The default filesystem depends on FlexVolume script. +optional | [optional] 
**Options** | Pointer to **map[string]string** | options is Optional: this field holds extra command options if any. +optional | [optional] 
**ReadOnly** | Pointer to **bool** | readOnly is Optional: defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. +optional | [optional] 
**SecretRef** | Pointer to [**V1LocalObjectReference**](V1LocalObjectReference.md) | secretRef is Optional: secretRef is reference to the secret object containing sensitive information to pass to the plugin scripts. This may be empty if no secret object is specified. If the secret object contains more than one secret, all secrets are passed to the plugin scripts. +optional | [optional] 

## Methods

### NewV1FlexVolumeSource

`func NewV1FlexVolumeSource() *V1FlexVolumeSource`

NewV1FlexVolumeSource instantiates a new V1FlexVolumeSource object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1FlexVolumeSourceWithDefaults

`func NewV1FlexVolumeSourceWithDefaults() *V1FlexVolumeSource`

NewV1FlexVolumeSourceWithDefaults instantiates a new V1FlexVolumeSource object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDriver

`func (o *V1FlexVolumeSource) GetDriver() string`

GetDriver returns the Driver field if non-nil, zero value otherwise.

### GetDriverOk

`func (o *V1FlexVolumeSource) GetDriverOk() (*string, bool)`

GetDriverOk returns a tuple with the Driver field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDriver

`func (o *V1FlexVolumeSource) SetDriver(v string)`

SetDriver sets Driver field to given value.

### HasDriver

`func (o *V1FlexVolumeSource) HasDriver() bool`

HasDriver returns a boolean if a field has been set.

### GetFsType

`func (o *V1FlexVolumeSource) GetFsType() string`

GetFsType returns the FsType field if non-nil, zero value otherwise.

### GetFsTypeOk

`func (o *V1FlexVolumeSource) GetFsTypeOk() (*string, bool)`

GetFsTypeOk returns a tuple with the FsType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFsType

`func (o *V1FlexVolumeSource) SetFsType(v string)`

SetFsType sets FsType field to given value.

### HasFsType

`func (o *V1FlexVolumeSource) HasFsType() bool`

HasFsType returns a boolean if a field has been set.

### GetOptions

`func (o *V1FlexVolumeSource) GetOptions() map[string]string`

GetOptions returns the Options field if non-nil, zero value otherwise.

### GetOptionsOk

`func (o *V1FlexVolumeSource) GetOptionsOk() (*map[string]string, bool)`

GetOptionsOk returns a tuple with the Options field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOptions

`func (o *V1FlexVolumeSource) SetOptions(v map[string]string)`

SetOptions sets Options field to given value.

### HasOptions

`func (o *V1FlexVolumeSource) HasOptions() bool`

HasOptions returns a boolean if a field has been set.

### GetReadOnly

`func (o *V1FlexVolumeSource) GetReadOnly() bool`

GetReadOnly returns the ReadOnly field if non-nil, zero value otherwise.

### GetReadOnlyOk

`func (o *V1FlexVolumeSource) GetReadOnlyOk() (*bool, bool)`

GetReadOnlyOk returns a tuple with the ReadOnly field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReadOnly

`func (o *V1FlexVolumeSource) SetReadOnly(v bool)`

SetReadOnly sets ReadOnly field to given value.

### HasReadOnly

`func (o *V1FlexVolumeSource) HasReadOnly() bool`

HasReadOnly returns a boolean if a field has been set.

### GetSecretRef

`func (o *V1FlexVolumeSource) GetSecretRef() V1LocalObjectReference`

GetSecretRef returns the SecretRef field if non-nil, zero value otherwise.

### GetSecretRefOk

`func (o *V1FlexVolumeSource) GetSecretRefOk() (*V1LocalObjectReference, bool)`

GetSecretRefOk returns a tuple with the SecretRef field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecretRef

`func (o *V1FlexVolumeSource) SetSecretRef(v V1LocalObjectReference)`

SetSecretRef sets SecretRef field to given value.

### HasSecretRef

`func (o *V1FlexVolumeSource) HasSecretRef() bool`

HasSecretRef returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


