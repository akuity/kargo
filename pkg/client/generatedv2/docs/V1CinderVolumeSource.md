# V1CinderVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**FsType** | Pointer to **string** | fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Examples: \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. Implicitly inferred to be \&quot;ext4\&quot; if unspecified. More info: https://examples.k8s.io/mysql-cinder-pd/README.md +optional | [optional] 
**ReadOnly** | Pointer to **bool** | readOnly defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. More info: https://examples.k8s.io/mysql-cinder-pd/README.md +optional | [optional] 
**SecretRef** | Pointer to [**V1LocalObjectReference**](V1LocalObjectReference.md) | secretRef is optional: points to a secret object containing parameters used to connect to OpenStack. +optional | [optional] 
**VolumeID** | Pointer to **string** | volumeID used to identify the volume in cinder. More info: https://examples.k8s.io/mysql-cinder-pd/README.md | [optional] 

## Methods

### NewV1CinderVolumeSource

`func NewV1CinderVolumeSource() *V1CinderVolumeSource`

NewV1CinderVolumeSource instantiates a new V1CinderVolumeSource object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1CinderVolumeSourceWithDefaults

`func NewV1CinderVolumeSourceWithDefaults() *V1CinderVolumeSource`

NewV1CinderVolumeSourceWithDefaults instantiates a new V1CinderVolumeSource object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetFsType

`func (o *V1CinderVolumeSource) GetFsType() string`

GetFsType returns the FsType field if non-nil, zero value otherwise.

### GetFsTypeOk

`func (o *V1CinderVolumeSource) GetFsTypeOk() (*string, bool)`

GetFsTypeOk returns a tuple with the FsType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFsType

`func (o *V1CinderVolumeSource) SetFsType(v string)`

SetFsType sets FsType field to given value.

### HasFsType

`func (o *V1CinderVolumeSource) HasFsType() bool`

HasFsType returns a boolean if a field has been set.

### GetReadOnly

`func (o *V1CinderVolumeSource) GetReadOnly() bool`

GetReadOnly returns the ReadOnly field if non-nil, zero value otherwise.

### GetReadOnlyOk

`func (o *V1CinderVolumeSource) GetReadOnlyOk() (*bool, bool)`

GetReadOnlyOk returns a tuple with the ReadOnly field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReadOnly

`func (o *V1CinderVolumeSource) SetReadOnly(v bool)`

SetReadOnly sets ReadOnly field to given value.

### HasReadOnly

`func (o *V1CinderVolumeSource) HasReadOnly() bool`

HasReadOnly returns a boolean if a field has been set.

### GetSecretRef

`func (o *V1CinderVolumeSource) GetSecretRef() V1LocalObjectReference`

GetSecretRef returns the SecretRef field if non-nil, zero value otherwise.

### GetSecretRefOk

`func (o *V1CinderVolumeSource) GetSecretRefOk() (*V1LocalObjectReference, bool)`

GetSecretRefOk returns a tuple with the SecretRef field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecretRef

`func (o *V1CinderVolumeSource) SetSecretRef(v V1LocalObjectReference)`

SetSecretRef sets SecretRef field to given value.

### HasSecretRef

`func (o *V1CinderVolumeSource) HasSecretRef() bool`

HasSecretRef returns a boolean if a field has been set.

### GetVolumeID

`func (o *V1CinderVolumeSource) GetVolumeID() string`

GetVolumeID returns the VolumeID field if non-nil, zero value otherwise.

### GetVolumeIDOk

`func (o *V1CinderVolumeSource) GetVolumeIDOk() (*string, bool)`

GetVolumeIDOk returns a tuple with the VolumeID field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVolumeID

`func (o *V1CinderVolumeSource) SetVolumeID(v string)`

SetVolumeID sets VolumeID field to given value.

### HasVolumeID

`func (o *V1CinderVolumeSource) HasVolumeID() bool`

HasVolumeID returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


