# V1ISCSIVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ChapAuthDiscovery** | Pointer to **bool** | chapAuthDiscovery defines whether support iSCSI Discovery CHAP authentication +optional | [optional] 
**ChapAuthSession** | Pointer to **bool** | chapAuthSession defines whether support iSCSI Session CHAP authentication +optional | [optional] 
**FsType** | Pointer to **string** | fsType is the filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. Implicitly inferred to be \&quot;ext4\&quot; if unspecified. More info: https://kubernetes.io/docs/concepts/storage/volumes#iscsi TODO: how do we prevent errors in the filesystem from compromising the machine +optional | [optional] 
**InitiatorName** | Pointer to **string** | initiatorName is the custom iSCSI Initiator Name. If initiatorName is specified with iscsiInterface simultaneously, new iSCSI interface &lt;target portal&gt;:&lt;volume name&gt; will be created for the connection. +optional | [optional] 
**Iqn** | Pointer to **string** | iqn is the target iSCSI Qualified Name. | [optional] 
**IscsiInterface** | Pointer to **string** | iscsiInterface is the interface Name that uses an iSCSI transport. Defaults to &#39;default&#39; (tcp). +optional +default&#x3D;\&quot;default\&quot; | [optional] 
**Lun** | Pointer to **int32** | lun represents iSCSI Target Lun number. | [optional] 
**Portals** | Pointer to **[]string** | portals is the iSCSI Target Portal List. The portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260). +optional +listType&#x3D;atomic | [optional] 
**ReadOnly** | Pointer to **bool** | readOnly here will force the ReadOnly setting in VolumeMounts. Defaults to false. +optional | [optional] 
**SecretRef** | Pointer to [**V1LocalObjectReference**](V1LocalObjectReference.md) | secretRef is the CHAP Secret for iSCSI target and initiator authentication +optional | [optional] 
**TargetPortal** | Pointer to **string** | targetPortal is iSCSI Target Portal. The Portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260). | [optional] 

## Methods

### NewV1ISCSIVolumeSource

`func NewV1ISCSIVolumeSource() *V1ISCSIVolumeSource`

NewV1ISCSIVolumeSource instantiates a new V1ISCSIVolumeSource object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1ISCSIVolumeSourceWithDefaults

`func NewV1ISCSIVolumeSourceWithDefaults() *V1ISCSIVolumeSource`

NewV1ISCSIVolumeSourceWithDefaults instantiates a new V1ISCSIVolumeSource object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetChapAuthDiscovery

`func (o *V1ISCSIVolumeSource) GetChapAuthDiscovery() bool`

GetChapAuthDiscovery returns the ChapAuthDiscovery field if non-nil, zero value otherwise.

### GetChapAuthDiscoveryOk

`func (o *V1ISCSIVolumeSource) GetChapAuthDiscoveryOk() (*bool, bool)`

GetChapAuthDiscoveryOk returns a tuple with the ChapAuthDiscovery field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetChapAuthDiscovery

`func (o *V1ISCSIVolumeSource) SetChapAuthDiscovery(v bool)`

SetChapAuthDiscovery sets ChapAuthDiscovery field to given value.

### HasChapAuthDiscovery

`func (o *V1ISCSIVolumeSource) HasChapAuthDiscovery() bool`

HasChapAuthDiscovery returns a boolean if a field has been set.

### GetChapAuthSession

`func (o *V1ISCSIVolumeSource) GetChapAuthSession() bool`

GetChapAuthSession returns the ChapAuthSession field if non-nil, zero value otherwise.

### GetChapAuthSessionOk

`func (o *V1ISCSIVolumeSource) GetChapAuthSessionOk() (*bool, bool)`

GetChapAuthSessionOk returns a tuple with the ChapAuthSession field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetChapAuthSession

`func (o *V1ISCSIVolumeSource) SetChapAuthSession(v bool)`

SetChapAuthSession sets ChapAuthSession field to given value.

### HasChapAuthSession

`func (o *V1ISCSIVolumeSource) HasChapAuthSession() bool`

HasChapAuthSession returns a boolean if a field has been set.

### GetFsType

`func (o *V1ISCSIVolumeSource) GetFsType() string`

GetFsType returns the FsType field if non-nil, zero value otherwise.

### GetFsTypeOk

`func (o *V1ISCSIVolumeSource) GetFsTypeOk() (*string, bool)`

GetFsTypeOk returns a tuple with the FsType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFsType

`func (o *V1ISCSIVolumeSource) SetFsType(v string)`

SetFsType sets FsType field to given value.

### HasFsType

`func (o *V1ISCSIVolumeSource) HasFsType() bool`

HasFsType returns a boolean if a field has been set.

### GetInitiatorName

`func (o *V1ISCSIVolumeSource) GetInitiatorName() string`

GetInitiatorName returns the InitiatorName field if non-nil, zero value otherwise.

### GetInitiatorNameOk

`func (o *V1ISCSIVolumeSource) GetInitiatorNameOk() (*string, bool)`

GetInitiatorNameOk returns a tuple with the InitiatorName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInitiatorName

`func (o *V1ISCSIVolumeSource) SetInitiatorName(v string)`

SetInitiatorName sets InitiatorName field to given value.

### HasInitiatorName

`func (o *V1ISCSIVolumeSource) HasInitiatorName() bool`

HasInitiatorName returns a boolean if a field has been set.

### GetIqn

`func (o *V1ISCSIVolumeSource) GetIqn() string`

GetIqn returns the Iqn field if non-nil, zero value otherwise.

### GetIqnOk

`func (o *V1ISCSIVolumeSource) GetIqnOk() (*string, bool)`

GetIqnOk returns a tuple with the Iqn field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIqn

`func (o *V1ISCSIVolumeSource) SetIqn(v string)`

SetIqn sets Iqn field to given value.

### HasIqn

`func (o *V1ISCSIVolumeSource) HasIqn() bool`

HasIqn returns a boolean if a field has been set.

### GetIscsiInterface

`func (o *V1ISCSIVolumeSource) GetIscsiInterface() string`

GetIscsiInterface returns the IscsiInterface field if non-nil, zero value otherwise.

### GetIscsiInterfaceOk

`func (o *V1ISCSIVolumeSource) GetIscsiInterfaceOk() (*string, bool)`

GetIscsiInterfaceOk returns a tuple with the IscsiInterface field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIscsiInterface

`func (o *V1ISCSIVolumeSource) SetIscsiInterface(v string)`

SetIscsiInterface sets IscsiInterface field to given value.

### HasIscsiInterface

`func (o *V1ISCSIVolumeSource) HasIscsiInterface() bool`

HasIscsiInterface returns a boolean if a field has been set.

### GetLun

`func (o *V1ISCSIVolumeSource) GetLun() int32`

GetLun returns the Lun field if non-nil, zero value otherwise.

### GetLunOk

`func (o *V1ISCSIVolumeSource) GetLunOk() (*int32, bool)`

GetLunOk returns a tuple with the Lun field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLun

`func (o *V1ISCSIVolumeSource) SetLun(v int32)`

SetLun sets Lun field to given value.

### HasLun

`func (o *V1ISCSIVolumeSource) HasLun() bool`

HasLun returns a boolean if a field has been set.

### GetPortals

`func (o *V1ISCSIVolumeSource) GetPortals() []string`

GetPortals returns the Portals field if non-nil, zero value otherwise.

### GetPortalsOk

`func (o *V1ISCSIVolumeSource) GetPortalsOk() (*[]string, bool)`

GetPortalsOk returns a tuple with the Portals field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPortals

`func (o *V1ISCSIVolumeSource) SetPortals(v []string)`

SetPortals sets Portals field to given value.

### HasPortals

`func (o *V1ISCSIVolumeSource) HasPortals() bool`

HasPortals returns a boolean if a field has been set.

### GetReadOnly

`func (o *V1ISCSIVolumeSource) GetReadOnly() bool`

GetReadOnly returns the ReadOnly field if non-nil, zero value otherwise.

### GetReadOnlyOk

`func (o *V1ISCSIVolumeSource) GetReadOnlyOk() (*bool, bool)`

GetReadOnlyOk returns a tuple with the ReadOnly field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReadOnly

`func (o *V1ISCSIVolumeSource) SetReadOnly(v bool)`

SetReadOnly sets ReadOnly field to given value.

### HasReadOnly

`func (o *V1ISCSIVolumeSource) HasReadOnly() bool`

HasReadOnly returns a boolean if a field has been set.

### GetSecretRef

`func (o *V1ISCSIVolumeSource) GetSecretRef() V1LocalObjectReference`

GetSecretRef returns the SecretRef field if non-nil, zero value otherwise.

### GetSecretRefOk

`func (o *V1ISCSIVolumeSource) GetSecretRefOk() (*V1LocalObjectReference, bool)`

GetSecretRefOk returns a tuple with the SecretRef field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecretRef

`func (o *V1ISCSIVolumeSource) SetSecretRef(v V1LocalObjectReference)`

SetSecretRef sets SecretRef field to given value.

### HasSecretRef

`func (o *V1ISCSIVolumeSource) HasSecretRef() bool`

HasSecretRef returns a boolean if a field has been set.

### GetTargetPortal

`func (o *V1ISCSIVolumeSource) GetTargetPortal() string`

GetTargetPortal returns the TargetPortal field if non-nil, zero value otherwise.

### GetTargetPortalOk

`func (o *V1ISCSIVolumeSource) GetTargetPortalOk() (*string, bool)`

GetTargetPortalOk returns a tuple with the TargetPortal field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTargetPortal

`func (o *V1ISCSIVolumeSource) SetTargetPortal(v string)`

SetTargetPortal sets TargetPortal field to given value.

### HasTargetPortal

`func (o *V1ISCSIVolumeSource) HasTargetPortal() bool`

HasTargetPortal returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


