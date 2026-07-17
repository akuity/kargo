# V1CSIVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Driver** | Pointer to **string** | driver is the name of the CSI driver that handles this volume. Consult with your admin for the correct name as registered in the cluster. | [optional] 
**FsType** | Pointer to **string** | fsType to mount. Ex. \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. If not provided, the empty value is passed to the associated CSI driver which will determine the default filesystem to apply. +optional | [optional] 
**NodePublishSecretRef** | Pointer to [**V1LocalObjectReference**](V1LocalObjectReference.md) | nodePublishSecretRef is a reference to the secret object containing sensitive information to pass to the CSI driver to complete the CSI NodePublishVolume and NodeUnpublishVolume calls. This field is optional, and  may be empty if no secret is required. If the secret object contains more than one secret, all secret references are passed. +optional | [optional] 
**ReadOnly** | Pointer to **bool** | readOnly specifies a read-only configuration for the volume. Defaults to false (read/write). +optional | [optional] 
**VolumeAttributes** | Pointer to **map[string]string** | volumeAttributes stores driver-specific properties that are passed to the CSI driver. Consult your driver&#39;s documentation for supported values. +optional | [optional] 

## Methods

### NewV1CSIVolumeSource

`func NewV1CSIVolumeSource() *V1CSIVolumeSource`

NewV1CSIVolumeSource instantiates a new V1CSIVolumeSource object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1CSIVolumeSourceWithDefaults

`func NewV1CSIVolumeSourceWithDefaults() *V1CSIVolumeSource`

NewV1CSIVolumeSourceWithDefaults instantiates a new V1CSIVolumeSource object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDriver

`func (o *V1CSIVolumeSource) GetDriver() string`

GetDriver returns the Driver field if non-nil, zero value otherwise.

### GetDriverOk

`func (o *V1CSIVolumeSource) GetDriverOk() (*string, bool)`

GetDriverOk returns a tuple with the Driver field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDriver

`func (o *V1CSIVolumeSource) SetDriver(v string)`

SetDriver sets Driver field to given value.

### HasDriver

`func (o *V1CSIVolumeSource) HasDriver() bool`

HasDriver returns a boolean if a field has been set.

### GetFsType

`func (o *V1CSIVolumeSource) GetFsType() string`

GetFsType returns the FsType field if non-nil, zero value otherwise.

### GetFsTypeOk

`func (o *V1CSIVolumeSource) GetFsTypeOk() (*string, bool)`

GetFsTypeOk returns a tuple with the FsType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFsType

`func (o *V1CSIVolumeSource) SetFsType(v string)`

SetFsType sets FsType field to given value.

### HasFsType

`func (o *V1CSIVolumeSource) HasFsType() bool`

HasFsType returns a boolean if a field has been set.

### GetNodePublishSecretRef

`func (o *V1CSIVolumeSource) GetNodePublishSecretRef() V1LocalObjectReference`

GetNodePublishSecretRef returns the NodePublishSecretRef field if non-nil, zero value otherwise.

### GetNodePublishSecretRefOk

`func (o *V1CSIVolumeSource) GetNodePublishSecretRefOk() (*V1LocalObjectReference, bool)`

GetNodePublishSecretRefOk returns a tuple with the NodePublishSecretRef field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNodePublishSecretRef

`func (o *V1CSIVolumeSource) SetNodePublishSecretRef(v V1LocalObjectReference)`

SetNodePublishSecretRef sets NodePublishSecretRef field to given value.

### HasNodePublishSecretRef

`func (o *V1CSIVolumeSource) HasNodePublishSecretRef() bool`

HasNodePublishSecretRef returns a boolean if a field has been set.

### GetReadOnly

`func (o *V1CSIVolumeSource) GetReadOnly() bool`

GetReadOnly returns the ReadOnly field if non-nil, zero value otherwise.

### GetReadOnlyOk

`func (o *V1CSIVolumeSource) GetReadOnlyOk() (*bool, bool)`

GetReadOnlyOk returns a tuple with the ReadOnly field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReadOnly

`func (o *V1CSIVolumeSource) SetReadOnly(v bool)`

SetReadOnly sets ReadOnly field to given value.

### HasReadOnly

`func (o *V1CSIVolumeSource) HasReadOnly() bool`

HasReadOnly returns a boolean if a field has been set.

### GetVolumeAttributes

`func (o *V1CSIVolumeSource) GetVolumeAttributes() map[string]string`

GetVolumeAttributes returns the VolumeAttributes field if non-nil, zero value otherwise.

### GetVolumeAttributesOk

`func (o *V1CSIVolumeSource) GetVolumeAttributesOk() (*map[string]string, bool)`

GetVolumeAttributesOk returns a tuple with the VolumeAttributes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVolumeAttributes

`func (o *V1CSIVolumeSource) SetVolumeAttributes(v map[string]string)`

SetVolumeAttributes sets VolumeAttributes field to given value.

### HasVolumeAttributes

`func (o *V1CSIVolumeSource) HasVolumeAttributes() bool`

HasVolumeAttributes returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


