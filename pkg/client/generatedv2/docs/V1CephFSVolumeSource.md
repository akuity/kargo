# V1CephFSVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Monitors** | Pointer to **[]string** | monitors is Required: Monitors is a collection of Ceph monitors More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it +listType&#x3D;atomic | [optional] 
**Path** | Pointer to **string** | path is Optional: Used as the mounted root, rather than the full Ceph tree, default is / +optional | [optional] 
**ReadOnly** | Pointer to **bool** | readOnly is Optional: Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it +optional | [optional] 
**SecretFile** | Pointer to **string** | secretFile is Optional: SecretFile is the path to key ring for User, default is /etc/ceph/user.secret More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it +optional | [optional] 
**SecretRef** | Pointer to [**V1LocalObjectReference**](V1LocalObjectReference.md) | secretRef is Optional: SecretRef is reference to the authentication secret for User, default is empty. More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it +optional | [optional] 
**User** | Pointer to **string** | user is optional: User is the rados user name, default is admin More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it +optional | [optional] 

## Methods

### NewV1CephFSVolumeSource

`func NewV1CephFSVolumeSource() *V1CephFSVolumeSource`

NewV1CephFSVolumeSource instantiates a new V1CephFSVolumeSource object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1CephFSVolumeSourceWithDefaults

`func NewV1CephFSVolumeSourceWithDefaults() *V1CephFSVolumeSource`

NewV1CephFSVolumeSourceWithDefaults instantiates a new V1CephFSVolumeSource object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetMonitors

`func (o *V1CephFSVolumeSource) GetMonitors() []string`

GetMonitors returns the Monitors field if non-nil, zero value otherwise.

### GetMonitorsOk

`func (o *V1CephFSVolumeSource) GetMonitorsOk() (*[]string, bool)`

GetMonitorsOk returns a tuple with the Monitors field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMonitors

`func (o *V1CephFSVolumeSource) SetMonitors(v []string)`

SetMonitors sets Monitors field to given value.

### HasMonitors

`func (o *V1CephFSVolumeSource) HasMonitors() bool`

HasMonitors returns a boolean if a field has been set.

### GetPath

`func (o *V1CephFSVolumeSource) GetPath() string`

GetPath returns the Path field if non-nil, zero value otherwise.

### GetPathOk

`func (o *V1CephFSVolumeSource) GetPathOk() (*string, bool)`

GetPathOk returns a tuple with the Path field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPath

`func (o *V1CephFSVolumeSource) SetPath(v string)`

SetPath sets Path field to given value.

### HasPath

`func (o *V1CephFSVolumeSource) HasPath() bool`

HasPath returns a boolean if a field has been set.

### GetReadOnly

`func (o *V1CephFSVolumeSource) GetReadOnly() bool`

GetReadOnly returns the ReadOnly field if non-nil, zero value otherwise.

### GetReadOnlyOk

`func (o *V1CephFSVolumeSource) GetReadOnlyOk() (*bool, bool)`

GetReadOnlyOk returns a tuple with the ReadOnly field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReadOnly

`func (o *V1CephFSVolumeSource) SetReadOnly(v bool)`

SetReadOnly sets ReadOnly field to given value.

### HasReadOnly

`func (o *V1CephFSVolumeSource) HasReadOnly() bool`

HasReadOnly returns a boolean if a field has been set.

### GetSecretFile

`func (o *V1CephFSVolumeSource) GetSecretFile() string`

GetSecretFile returns the SecretFile field if non-nil, zero value otherwise.

### GetSecretFileOk

`func (o *V1CephFSVolumeSource) GetSecretFileOk() (*string, bool)`

GetSecretFileOk returns a tuple with the SecretFile field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecretFile

`func (o *V1CephFSVolumeSource) SetSecretFile(v string)`

SetSecretFile sets SecretFile field to given value.

### HasSecretFile

`func (o *V1CephFSVolumeSource) HasSecretFile() bool`

HasSecretFile returns a boolean if a field has been set.

### GetSecretRef

`func (o *V1CephFSVolumeSource) GetSecretRef() V1LocalObjectReference`

GetSecretRef returns the SecretRef field if non-nil, zero value otherwise.

### GetSecretRefOk

`func (o *V1CephFSVolumeSource) GetSecretRefOk() (*V1LocalObjectReference, bool)`

GetSecretRefOk returns a tuple with the SecretRef field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecretRef

`func (o *V1CephFSVolumeSource) SetSecretRef(v V1LocalObjectReference)`

SetSecretRef sets SecretRef field to given value.

### HasSecretRef

`func (o *V1CephFSVolumeSource) HasSecretRef() bool`

HasSecretRef returns a boolean if a field has been set.

### GetUser

`func (o *V1CephFSVolumeSource) GetUser() string`

GetUser returns the User field if non-nil, zero value otherwise.

### GetUserOk

`func (o *V1CephFSVolumeSource) GetUserOk() (*string, bool)`

GetUserOk returns a tuple with the User field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUser

`func (o *V1CephFSVolumeSource) SetUser(v string)`

SetUser sets User field to given value.

### HasUser

`func (o *V1CephFSVolumeSource) HasUser() bool`

HasUser returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


