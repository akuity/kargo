# V1PhotonPersistentDiskVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**FsType** | Pointer to **string** | fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. Implicitly inferred to be \&quot;ext4\&quot; if unspecified. | [optional] 
**PdID** | Pointer to **string** | pdID is the ID that identifies Photon Controller persistent disk | [optional] 

## Methods

### NewV1PhotonPersistentDiskVolumeSource

`func NewV1PhotonPersistentDiskVolumeSource() *V1PhotonPersistentDiskVolumeSource`

NewV1PhotonPersistentDiskVolumeSource instantiates a new V1PhotonPersistentDiskVolumeSource object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1PhotonPersistentDiskVolumeSourceWithDefaults

`func NewV1PhotonPersistentDiskVolumeSourceWithDefaults() *V1PhotonPersistentDiskVolumeSource`

NewV1PhotonPersistentDiskVolumeSourceWithDefaults instantiates a new V1PhotonPersistentDiskVolumeSource object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetFsType

`func (o *V1PhotonPersistentDiskVolumeSource) GetFsType() string`

GetFsType returns the FsType field if non-nil, zero value otherwise.

### GetFsTypeOk

`func (o *V1PhotonPersistentDiskVolumeSource) GetFsTypeOk() (*string, bool)`

GetFsTypeOk returns a tuple with the FsType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFsType

`func (o *V1PhotonPersistentDiskVolumeSource) SetFsType(v string)`

SetFsType sets FsType field to given value.

### HasFsType

`func (o *V1PhotonPersistentDiskVolumeSource) HasFsType() bool`

HasFsType returns a boolean if a field has been set.

### GetPdID

`func (o *V1PhotonPersistentDiskVolumeSource) GetPdID() string`

GetPdID returns the PdID field if non-nil, zero value otherwise.

### GetPdIDOk

`func (o *V1PhotonPersistentDiskVolumeSource) GetPdIDOk() (*string, bool)`

GetPdIDOk returns a tuple with the PdID field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPdID

`func (o *V1PhotonPersistentDiskVolumeSource) SetPdID(v string)`

SetPdID sets PdID field to given value.

### HasPdID

`func (o *V1PhotonPersistentDiskVolumeSource) HasPdID() bool`

HasPdID returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


