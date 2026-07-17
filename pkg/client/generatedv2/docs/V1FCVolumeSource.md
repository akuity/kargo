# V1FCVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**FsType** | Pointer to **string** | fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. Implicitly inferred to be \&quot;ext4\&quot; if unspecified. TODO: how do we prevent errors in the filesystem from compromising the machine +optional | [optional] 
**Lun** | Pointer to **int32** | lun is Optional: FC target lun number +optional | [optional] 
**ReadOnly** | Pointer to **bool** | readOnly is Optional: Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. +optional | [optional] 
**TargetWWNs** | Pointer to **[]string** | targetWWNs is Optional: FC target worldwide names (WWNs) +optional +listType&#x3D;atomic | [optional] 
**Wwids** | Pointer to **[]string** | wwids Optional: FC volume world wide identifiers (wwids) Either wwids or combination of targetWWNs and lun must be set, but not both simultaneously. +optional +listType&#x3D;atomic | [optional] 

## Methods

### NewV1FCVolumeSource

`func NewV1FCVolumeSource() *V1FCVolumeSource`

NewV1FCVolumeSource instantiates a new V1FCVolumeSource object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1FCVolumeSourceWithDefaults

`func NewV1FCVolumeSourceWithDefaults() *V1FCVolumeSource`

NewV1FCVolumeSourceWithDefaults instantiates a new V1FCVolumeSource object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetFsType

`func (o *V1FCVolumeSource) GetFsType() string`

GetFsType returns the FsType field if non-nil, zero value otherwise.

### GetFsTypeOk

`func (o *V1FCVolumeSource) GetFsTypeOk() (*string, bool)`

GetFsTypeOk returns a tuple with the FsType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFsType

`func (o *V1FCVolumeSource) SetFsType(v string)`

SetFsType sets FsType field to given value.

### HasFsType

`func (o *V1FCVolumeSource) HasFsType() bool`

HasFsType returns a boolean if a field has been set.

### GetLun

`func (o *V1FCVolumeSource) GetLun() int32`

GetLun returns the Lun field if non-nil, zero value otherwise.

### GetLunOk

`func (o *V1FCVolumeSource) GetLunOk() (*int32, bool)`

GetLunOk returns a tuple with the Lun field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLun

`func (o *V1FCVolumeSource) SetLun(v int32)`

SetLun sets Lun field to given value.

### HasLun

`func (o *V1FCVolumeSource) HasLun() bool`

HasLun returns a boolean if a field has been set.

### GetReadOnly

`func (o *V1FCVolumeSource) GetReadOnly() bool`

GetReadOnly returns the ReadOnly field if non-nil, zero value otherwise.

### GetReadOnlyOk

`func (o *V1FCVolumeSource) GetReadOnlyOk() (*bool, bool)`

GetReadOnlyOk returns a tuple with the ReadOnly field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReadOnly

`func (o *V1FCVolumeSource) SetReadOnly(v bool)`

SetReadOnly sets ReadOnly field to given value.

### HasReadOnly

`func (o *V1FCVolumeSource) HasReadOnly() bool`

HasReadOnly returns a boolean if a field has been set.

### GetTargetWWNs

`func (o *V1FCVolumeSource) GetTargetWWNs() []string`

GetTargetWWNs returns the TargetWWNs field if non-nil, zero value otherwise.

### GetTargetWWNsOk

`func (o *V1FCVolumeSource) GetTargetWWNsOk() (*[]string, bool)`

GetTargetWWNsOk returns a tuple with the TargetWWNs field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTargetWWNs

`func (o *V1FCVolumeSource) SetTargetWWNs(v []string)`

SetTargetWWNs sets TargetWWNs field to given value.

### HasTargetWWNs

`func (o *V1FCVolumeSource) HasTargetWWNs() bool`

HasTargetWWNs returns a boolean if a field has been set.

### GetWwids

`func (o *V1FCVolumeSource) GetWwids() []string`

GetWwids returns the Wwids field if non-nil, zero value otherwise.

### GetWwidsOk

`func (o *V1FCVolumeSource) GetWwidsOk() (*[]string, bool)`

GetWwidsOk returns a tuple with the Wwids field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWwids

`func (o *V1FCVolumeSource) SetWwids(v []string)`

SetWwids sets Wwids field to given value.

### HasWwids

`func (o *V1FCVolumeSource) HasWwids() bool`

HasWwids returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


