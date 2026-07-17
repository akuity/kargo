# V1RBDVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**FsType** | Pointer to **string** | fsType is the filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. Implicitly inferred to be \&quot;ext4\&quot; if unspecified. More info: https://kubernetes.io/docs/concepts/storage/volumes#rbd TODO: how do we prevent errors in the filesystem from compromising the machine +optional | [optional] 
**Image** | Pointer to **string** | image is the rados image name. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it | [optional] 
**Keyring** | Pointer to **string** | keyring is the path to key ring for RBDUser. Default is /etc/ceph/keyring. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it +optional +default&#x3D;\&quot;/etc/ceph/keyring\&quot; | [optional] 
**Monitors** | Pointer to **[]string** | monitors is a collection of Ceph monitors. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it +listType&#x3D;atomic | [optional] 
**Pool** | Pointer to **string** | pool is the rados pool name. Default is rbd. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it +optional +default&#x3D;\&quot;rbd\&quot; | [optional] 
**ReadOnly** | Pointer to **bool** | readOnly here will force the ReadOnly setting in VolumeMounts. Defaults to false. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it +optional | [optional] 
**SecretRef** | Pointer to [**V1LocalObjectReference**](V1LocalObjectReference.md) | secretRef is name of the authentication secret for RBDUser. If provided overrides keyring. Default is nil. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it +optional | [optional] 
**User** | Pointer to **string** | user is the rados user name. Default is admin. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it +optional +default&#x3D;\&quot;admin\&quot; | [optional] 

## Methods

### NewV1RBDVolumeSource

`func NewV1RBDVolumeSource() *V1RBDVolumeSource`

NewV1RBDVolumeSource instantiates a new V1RBDVolumeSource object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1RBDVolumeSourceWithDefaults

`func NewV1RBDVolumeSourceWithDefaults() *V1RBDVolumeSource`

NewV1RBDVolumeSourceWithDefaults instantiates a new V1RBDVolumeSource object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetFsType

`func (o *V1RBDVolumeSource) GetFsType() string`

GetFsType returns the FsType field if non-nil, zero value otherwise.

### GetFsTypeOk

`func (o *V1RBDVolumeSource) GetFsTypeOk() (*string, bool)`

GetFsTypeOk returns a tuple with the FsType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFsType

`func (o *V1RBDVolumeSource) SetFsType(v string)`

SetFsType sets FsType field to given value.

### HasFsType

`func (o *V1RBDVolumeSource) HasFsType() bool`

HasFsType returns a boolean if a field has been set.

### GetImage

`func (o *V1RBDVolumeSource) GetImage() string`

GetImage returns the Image field if non-nil, zero value otherwise.

### GetImageOk

`func (o *V1RBDVolumeSource) GetImageOk() (*string, bool)`

GetImageOk returns a tuple with the Image field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetImage

`func (o *V1RBDVolumeSource) SetImage(v string)`

SetImage sets Image field to given value.

### HasImage

`func (o *V1RBDVolumeSource) HasImage() bool`

HasImage returns a boolean if a field has been set.

### GetKeyring

`func (o *V1RBDVolumeSource) GetKeyring() string`

GetKeyring returns the Keyring field if non-nil, zero value otherwise.

### GetKeyringOk

`func (o *V1RBDVolumeSource) GetKeyringOk() (*string, bool)`

GetKeyringOk returns a tuple with the Keyring field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKeyring

`func (o *V1RBDVolumeSource) SetKeyring(v string)`

SetKeyring sets Keyring field to given value.

### HasKeyring

`func (o *V1RBDVolumeSource) HasKeyring() bool`

HasKeyring returns a boolean if a field has been set.

### GetMonitors

`func (o *V1RBDVolumeSource) GetMonitors() []string`

GetMonitors returns the Monitors field if non-nil, zero value otherwise.

### GetMonitorsOk

`func (o *V1RBDVolumeSource) GetMonitorsOk() (*[]string, bool)`

GetMonitorsOk returns a tuple with the Monitors field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMonitors

`func (o *V1RBDVolumeSource) SetMonitors(v []string)`

SetMonitors sets Monitors field to given value.

### HasMonitors

`func (o *V1RBDVolumeSource) HasMonitors() bool`

HasMonitors returns a boolean if a field has been set.

### GetPool

`func (o *V1RBDVolumeSource) GetPool() string`

GetPool returns the Pool field if non-nil, zero value otherwise.

### GetPoolOk

`func (o *V1RBDVolumeSource) GetPoolOk() (*string, bool)`

GetPoolOk returns a tuple with the Pool field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPool

`func (o *V1RBDVolumeSource) SetPool(v string)`

SetPool sets Pool field to given value.

### HasPool

`func (o *V1RBDVolumeSource) HasPool() bool`

HasPool returns a boolean if a field has been set.

### GetReadOnly

`func (o *V1RBDVolumeSource) GetReadOnly() bool`

GetReadOnly returns the ReadOnly field if non-nil, zero value otherwise.

### GetReadOnlyOk

`func (o *V1RBDVolumeSource) GetReadOnlyOk() (*bool, bool)`

GetReadOnlyOk returns a tuple with the ReadOnly field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReadOnly

`func (o *V1RBDVolumeSource) SetReadOnly(v bool)`

SetReadOnly sets ReadOnly field to given value.

### HasReadOnly

`func (o *V1RBDVolumeSource) HasReadOnly() bool`

HasReadOnly returns a boolean if a field has been set.

### GetSecretRef

`func (o *V1RBDVolumeSource) GetSecretRef() V1LocalObjectReference`

GetSecretRef returns the SecretRef field if non-nil, zero value otherwise.

### GetSecretRefOk

`func (o *V1RBDVolumeSource) GetSecretRefOk() (*V1LocalObjectReference, bool)`

GetSecretRefOk returns a tuple with the SecretRef field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecretRef

`func (o *V1RBDVolumeSource) SetSecretRef(v V1LocalObjectReference)`

SetSecretRef sets SecretRef field to given value.

### HasSecretRef

`func (o *V1RBDVolumeSource) HasSecretRef() bool`

HasSecretRef returns a boolean if a field has been set.

### GetUser

`func (o *V1RBDVolumeSource) GetUser() string`

GetUser returns the User field if non-nil, zero value otherwise.

### GetUserOk

`func (o *V1RBDVolumeSource) GetUserOk() (*string, bool)`

GetUserOk returns a tuple with the User field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUser

`func (o *V1RBDVolumeSource) SetUser(v string)`

SetUser sets User field to given value.

### HasUser

`func (o *V1RBDVolumeSource) HasUser() bool`

HasUser returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


