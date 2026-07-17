# V1ScaleIOVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**FsType** | Pointer to **string** | fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. Default is \&quot;xfs\&quot;. +optional +default&#x3D;\&quot;xfs\&quot; | [optional] 
**Gateway** | Pointer to **string** | gateway is the host address of the ScaleIO API Gateway. | [optional] 
**ProtectionDomain** | Pointer to **string** | protectionDomain is the name of the ScaleIO Protection Domain for the configured storage. +optional | [optional] 
**ReadOnly** | Pointer to **bool** | readOnly Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. +optional | [optional] 
**SecretRef** | Pointer to [**V1LocalObjectReference**](V1LocalObjectReference.md) | secretRef references to the secret for ScaleIO user and other sensitive information. If this is not provided, Login operation will fail. | [optional] 
**SslEnabled** | Pointer to **bool** | sslEnabled Flag enable/disable SSL communication with Gateway, default false +optional | [optional] 
**StorageMode** | Pointer to **string** | storageMode indicates whether the storage for a volume should be ThickProvisioned or ThinProvisioned. Default is ThinProvisioned. +optional +default&#x3D;\&quot;ThinProvisioned\&quot; | [optional] 
**StoragePool** | Pointer to **string** | storagePool is the ScaleIO Storage Pool associated with the protection domain. +optional | [optional] 
**System** | Pointer to **string** | system is the name of the storage system as configured in ScaleIO. | [optional] 
**VolumeName** | Pointer to **string** | volumeName is the name of a volume already created in the ScaleIO system that is associated with this volume source. | [optional] 

## Methods

### NewV1ScaleIOVolumeSource

`func NewV1ScaleIOVolumeSource() *V1ScaleIOVolumeSource`

NewV1ScaleIOVolumeSource instantiates a new V1ScaleIOVolumeSource object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1ScaleIOVolumeSourceWithDefaults

`func NewV1ScaleIOVolumeSourceWithDefaults() *V1ScaleIOVolumeSource`

NewV1ScaleIOVolumeSourceWithDefaults instantiates a new V1ScaleIOVolumeSource object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetFsType

`func (o *V1ScaleIOVolumeSource) GetFsType() string`

GetFsType returns the FsType field if non-nil, zero value otherwise.

### GetFsTypeOk

`func (o *V1ScaleIOVolumeSource) GetFsTypeOk() (*string, bool)`

GetFsTypeOk returns a tuple with the FsType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFsType

`func (o *V1ScaleIOVolumeSource) SetFsType(v string)`

SetFsType sets FsType field to given value.

### HasFsType

`func (o *V1ScaleIOVolumeSource) HasFsType() bool`

HasFsType returns a boolean if a field has been set.

### GetGateway

`func (o *V1ScaleIOVolumeSource) GetGateway() string`

GetGateway returns the Gateway field if non-nil, zero value otherwise.

### GetGatewayOk

`func (o *V1ScaleIOVolumeSource) GetGatewayOk() (*string, bool)`

GetGatewayOk returns a tuple with the Gateway field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetGateway

`func (o *V1ScaleIOVolumeSource) SetGateway(v string)`

SetGateway sets Gateway field to given value.

### HasGateway

`func (o *V1ScaleIOVolumeSource) HasGateway() bool`

HasGateway returns a boolean if a field has been set.

### GetProtectionDomain

`func (o *V1ScaleIOVolumeSource) GetProtectionDomain() string`

GetProtectionDomain returns the ProtectionDomain field if non-nil, zero value otherwise.

### GetProtectionDomainOk

`func (o *V1ScaleIOVolumeSource) GetProtectionDomainOk() (*string, bool)`

GetProtectionDomainOk returns a tuple with the ProtectionDomain field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProtectionDomain

`func (o *V1ScaleIOVolumeSource) SetProtectionDomain(v string)`

SetProtectionDomain sets ProtectionDomain field to given value.

### HasProtectionDomain

`func (o *V1ScaleIOVolumeSource) HasProtectionDomain() bool`

HasProtectionDomain returns a boolean if a field has been set.

### GetReadOnly

`func (o *V1ScaleIOVolumeSource) GetReadOnly() bool`

GetReadOnly returns the ReadOnly field if non-nil, zero value otherwise.

### GetReadOnlyOk

`func (o *V1ScaleIOVolumeSource) GetReadOnlyOk() (*bool, bool)`

GetReadOnlyOk returns a tuple with the ReadOnly field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReadOnly

`func (o *V1ScaleIOVolumeSource) SetReadOnly(v bool)`

SetReadOnly sets ReadOnly field to given value.

### HasReadOnly

`func (o *V1ScaleIOVolumeSource) HasReadOnly() bool`

HasReadOnly returns a boolean if a field has been set.

### GetSecretRef

`func (o *V1ScaleIOVolumeSource) GetSecretRef() V1LocalObjectReference`

GetSecretRef returns the SecretRef field if non-nil, zero value otherwise.

### GetSecretRefOk

`func (o *V1ScaleIOVolumeSource) GetSecretRefOk() (*V1LocalObjectReference, bool)`

GetSecretRefOk returns a tuple with the SecretRef field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecretRef

`func (o *V1ScaleIOVolumeSource) SetSecretRef(v V1LocalObjectReference)`

SetSecretRef sets SecretRef field to given value.

### HasSecretRef

`func (o *V1ScaleIOVolumeSource) HasSecretRef() bool`

HasSecretRef returns a boolean if a field has been set.

### GetSslEnabled

`func (o *V1ScaleIOVolumeSource) GetSslEnabled() bool`

GetSslEnabled returns the SslEnabled field if non-nil, zero value otherwise.

### GetSslEnabledOk

`func (o *V1ScaleIOVolumeSource) GetSslEnabledOk() (*bool, bool)`

GetSslEnabledOk returns a tuple with the SslEnabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSslEnabled

`func (o *V1ScaleIOVolumeSource) SetSslEnabled(v bool)`

SetSslEnabled sets SslEnabled field to given value.

### HasSslEnabled

`func (o *V1ScaleIOVolumeSource) HasSslEnabled() bool`

HasSslEnabled returns a boolean if a field has been set.

### GetStorageMode

`func (o *V1ScaleIOVolumeSource) GetStorageMode() string`

GetStorageMode returns the StorageMode field if non-nil, zero value otherwise.

### GetStorageModeOk

`func (o *V1ScaleIOVolumeSource) GetStorageModeOk() (*string, bool)`

GetStorageModeOk returns a tuple with the StorageMode field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStorageMode

`func (o *V1ScaleIOVolumeSource) SetStorageMode(v string)`

SetStorageMode sets StorageMode field to given value.

### HasStorageMode

`func (o *V1ScaleIOVolumeSource) HasStorageMode() bool`

HasStorageMode returns a boolean if a field has been set.

### GetStoragePool

`func (o *V1ScaleIOVolumeSource) GetStoragePool() string`

GetStoragePool returns the StoragePool field if non-nil, zero value otherwise.

### GetStoragePoolOk

`func (o *V1ScaleIOVolumeSource) GetStoragePoolOk() (*string, bool)`

GetStoragePoolOk returns a tuple with the StoragePool field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStoragePool

`func (o *V1ScaleIOVolumeSource) SetStoragePool(v string)`

SetStoragePool sets StoragePool field to given value.

### HasStoragePool

`func (o *V1ScaleIOVolumeSource) HasStoragePool() bool`

HasStoragePool returns a boolean if a field has been set.

### GetSystem

`func (o *V1ScaleIOVolumeSource) GetSystem() string`

GetSystem returns the System field if non-nil, zero value otherwise.

### GetSystemOk

`func (o *V1ScaleIOVolumeSource) GetSystemOk() (*string, bool)`

GetSystemOk returns a tuple with the System field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSystem

`func (o *V1ScaleIOVolumeSource) SetSystem(v string)`

SetSystem sets System field to given value.

### HasSystem

`func (o *V1ScaleIOVolumeSource) HasSystem() bool`

HasSystem returns a boolean if a field has been set.

### GetVolumeName

`func (o *V1ScaleIOVolumeSource) GetVolumeName() string`

GetVolumeName returns the VolumeName field if non-nil, zero value otherwise.

### GetVolumeNameOk

`func (o *V1ScaleIOVolumeSource) GetVolumeNameOk() (*string, bool)`

GetVolumeNameOk returns a tuple with the VolumeName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVolumeName

`func (o *V1ScaleIOVolumeSource) SetVolumeName(v string)`

SetVolumeName sets VolumeName field to given value.

### HasVolumeName

`func (o *V1ScaleIOVolumeSource) HasVolumeName() bool`

HasVolumeName returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


