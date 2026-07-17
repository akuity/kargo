# V1PersistentVolumeClaimSpec

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**AccessModes** | Pointer to **[]string** | accessModes contains the desired access modes the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1 +optional +listType&#x3D;atomic | [optional] 
**DataSource** | Pointer to [**V1TypedLocalObjectReference**](V1TypedLocalObjectReference.md) | dataSource field can be used to specify either: * An existing VolumeSnapshot object (snapshot.storage.k8s.io/VolumeSnapshot) * An existing PVC (PersistentVolumeClaim) If the provisioner or an external controller can support the specified data source, it will create a new volume based on the contents of the specified data source. When the AnyVolumeDataSource feature gate is enabled, dataSource contents will be copied to dataSourceRef, and dataSourceRef contents will be copied to dataSource when dataSourceRef.namespace is not specified. If the namespace is specified, then dataSourceRef will not be copied to dataSource. +optional | [optional] 
**DataSourceRef** | Pointer to [**V1TypedObjectReference**](V1TypedObjectReference.md) | dataSourceRef specifies the object from which to populate the volume with data, if a non-empty volume is desired. This may be any object from a non-empty API group (non core object) or a PersistentVolumeClaim object. When this field is specified, volume binding will only succeed if the type of the specified object matches some installed volume populator or dynamic provisioner. This field will replace the functionality of the dataSource field and as such if both fields are non-empty, they must have the same value. For backwards compatibility, when namespace isn&#39;t specified in dataSourceRef, both fields (dataSource and dataSourceRef) will be set to the same value automatically if one of them is empty and the other is non-empty. When namespace is specified in dataSourceRef, dataSource isn&#39;t set to the same value and must be empty. There are three important differences between dataSource and dataSourceRef: * While dataSource only allows two specific types of objects, dataSourceRef   allows any non-core object, as well as PersistentVolumeClaim objects. * While dataSource ignores disallowed values (dropping them), dataSourceRef   preserves all values, and generates an error if a disallowed value is   specified. * While dataSource only allows local objects, dataSourceRef allows objects   in any namespaces. (Beta) Using this field requires the AnyVolumeDataSource feature gate to be enabled. (Alpha) Using the namespace field of dataSourceRef requires the CrossNamespaceVolumeDataSource feature gate to be enabled. +optional | [optional] 
**Resources** | Pointer to [**V1VolumeResourceRequirements**](V1VolumeResourceRequirements.md) | resources represents the minimum resources the volume should have. If RecoverVolumeExpansionFailure feature is enabled users are allowed to specify resource requirements that are lower than previous value but must still be higher than capacity recorded in the status field of the claim. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources +optional | [optional] 
**Selector** | Pointer to [**V1LabelSelector**](V1LabelSelector.md) | selector is a label query over volumes to consider for binding. +optional | [optional] 
**StorageClassName** | Pointer to **string** | storageClassName is the name of the StorageClass required by the claim. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1 +optional | [optional] 
**VolumeAttributesClassName** | Pointer to **string** | volumeAttributesClassName may be used to set the VolumeAttributesClass used by this claim. If specified, the CSI driver will create or update the volume with the attributes defined in the corresponding VolumeAttributesClass. This has a different purpose than storageClassName, it can be changed after the claim is created. An empty string or nil value indicates that no VolumeAttributesClass will be applied to the claim. If the claim enters an Infeasible error state, this field can be reset to its previous value (including nil) to cancel the modification. If the resource referred to by volumeAttributesClass does not exist, this PersistentVolumeClaim will be set to a Pending state, as reflected by the modifyVolumeStatus field, until such as a resource exists. More info: https://kubernetes.io/docs/concepts/storage/volume-attributes-classes/ +featureGate&#x3D;VolumeAttributesClass +optional | [optional] 
**VolumeMode** | Pointer to **string** | volumeMode defines what type of volume is required by the claim. Value of Filesystem is implied when not included in claim spec. +optional | [optional] 
**VolumeName** | Pointer to **string** | volumeName is the binding reference to the PersistentVolume backing this claim. +optional | [optional] 

## Methods

### NewV1PersistentVolumeClaimSpec

`func NewV1PersistentVolumeClaimSpec() *V1PersistentVolumeClaimSpec`

NewV1PersistentVolumeClaimSpec instantiates a new V1PersistentVolumeClaimSpec object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1PersistentVolumeClaimSpecWithDefaults

`func NewV1PersistentVolumeClaimSpecWithDefaults() *V1PersistentVolumeClaimSpec`

NewV1PersistentVolumeClaimSpecWithDefaults instantiates a new V1PersistentVolumeClaimSpec object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAccessModes

`func (o *V1PersistentVolumeClaimSpec) GetAccessModes() []string`

GetAccessModes returns the AccessModes field if non-nil, zero value otherwise.

### GetAccessModesOk

`func (o *V1PersistentVolumeClaimSpec) GetAccessModesOk() (*[]string, bool)`

GetAccessModesOk returns a tuple with the AccessModes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAccessModes

`func (o *V1PersistentVolumeClaimSpec) SetAccessModes(v []string)`

SetAccessModes sets AccessModes field to given value.

### HasAccessModes

`func (o *V1PersistentVolumeClaimSpec) HasAccessModes() bool`

HasAccessModes returns a boolean if a field has been set.

### GetDataSource

`func (o *V1PersistentVolumeClaimSpec) GetDataSource() V1TypedLocalObjectReference`

GetDataSource returns the DataSource field if non-nil, zero value otherwise.

### GetDataSourceOk

`func (o *V1PersistentVolumeClaimSpec) GetDataSourceOk() (*V1TypedLocalObjectReference, bool)`

GetDataSourceOk returns a tuple with the DataSource field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDataSource

`func (o *V1PersistentVolumeClaimSpec) SetDataSource(v V1TypedLocalObjectReference)`

SetDataSource sets DataSource field to given value.

### HasDataSource

`func (o *V1PersistentVolumeClaimSpec) HasDataSource() bool`

HasDataSource returns a boolean if a field has been set.

### GetDataSourceRef

`func (o *V1PersistentVolumeClaimSpec) GetDataSourceRef() V1TypedObjectReference`

GetDataSourceRef returns the DataSourceRef field if non-nil, zero value otherwise.

### GetDataSourceRefOk

`func (o *V1PersistentVolumeClaimSpec) GetDataSourceRefOk() (*V1TypedObjectReference, bool)`

GetDataSourceRefOk returns a tuple with the DataSourceRef field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDataSourceRef

`func (o *V1PersistentVolumeClaimSpec) SetDataSourceRef(v V1TypedObjectReference)`

SetDataSourceRef sets DataSourceRef field to given value.

### HasDataSourceRef

`func (o *V1PersistentVolumeClaimSpec) HasDataSourceRef() bool`

HasDataSourceRef returns a boolean if a field has been set.

### GetResources

`func (o *V1PersistentVolumeClaimSpec) GetResources() V1VolumeResourceRequirements`

GetResources returns the Resources field if non-nil, zero value otherwise.

### GetResourcesOk

`func (o *V1PersistentVolumeClaimSpec) GetResourcesOk() (*V1VolumeResourceRequirements, bool)`

GetResourcesOk returns a tuple with the Resources field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResources

`func (o *V1PersistentVolumeClaimSpec) SetResources(v V1VolumeResourceRequirements)`

SetResources sets Resources field to given value.

### HasResources

`func (o *V1PersistentVolumeClaimSpec) HasResources() bool`

HasResources returns a boolean if a field has been set.

### GetSelector

`func (o *V1PersistentVolumeClaimSpec) GetSelector() V1LabelSelector`

GetSelector returns the Selector field if non-nil, zero value otherwise.

### GetSelectorOk

`func (o *V1PersistentVolumeClaimSpec) GetSelectorOk() (*V1LabelSelector, bool)`

GetSelectorOk returns a tuple with the Selector field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSelector

`func (o *V1PersistentVolumeClaimSpec) SetSelector(v V1LabelSelector)`

SetSelector sets Selector field to given value.

### HasSelector

`func (o *V1PersistentVolumeClaimSpec) HasSelector() bool`

HasSelector returns a boolean if a field has been set.

### GetStorageClassName

`func (o *V1PersistentVolumeClaimSpec) GetStorageClassName() string`

GetStorageClassName returns the StorageClassName field if non-nil, zero value otherwise.

### GetStorageClassNameOk

`func (o *V1PersistentVolumeClaimSpec) GetStorageClassNameOk() (*string, bool)`

GetStorageClassNameOk returns a tuple with the StorageClassName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStorageClassName

`func (o *V1PersistentVolumeClaimSpec) SetStorageClassName(v string)`

SetStorageClassName sets StorageClassName field to given value.

### HasStorageClassName

`func (o *V1PersistentVolumeClaimSpec) HasStorageClassName() bool`

HasStorageClassName returns a boolean if a field has been set.

### GetVolumeAttributesClassName

`func (o *V1PersistentVolumeClaimSpec) GetVolumeAttributesClassName() string`

GetVolumeAttributesClassName returns the VolumeAttributesClassName field if non-nil, zero value otherwise.

### GetVolumeAttributesClassNameOk

`func (o *V1PersistentVolumeClaimSpec) GetVolumeAttributesClassNameOk() (*string, bool)`

GetVolumeAttributesClassNameOk returns a tuple with the VolumeAttributesClassName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVolumeAttributesClassName

`func (o *V1PersistentVolumeClaimSpec) SetVolumeAttributesClassName(v string)`

SetVolumeAttributesClassName sets VolumeAttributesClassName field to given value.

### HasVolumeAttributesClassName

`func (o *V1PersistentVolumeClaimSpec) HasVolumeAttributesClassName() bool`

HasVolumeAttributesClassName returns a boolean if a field has been set.

### GetVolumeMode

`func (o *V1PersistentVolumeClaimSpec) GetVolumeMode() string`

GetVolumeMode returns the VolumeMode field if non-nil, zero value otherwise.

### GetVolumeModeOk

`func (o *V1PersistentVolumeClaimSpec) GetVolumeModeOk() (*string, bool)`

GetVolumeModeOk returns a tuple with the VolumeMode field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVolumeMode

`func (o *V1PersistentVolumeClaimSpec) SetVolumeMode(v string)`

SetVolumeMode sets VolumeMode field to given value.

### HasVolumeMode

`func (o *V1PersistentVolumeClaimSpec) HasVolumeMode() bool`

HasVolumeMode returns a boolean if a field has been set.

### GetVolumeName

`func (o *V1PersistentVolumeClaimSpec) GetVolumeName() string`

GetVolumeName returns the VolumeName field if non-nil, zero value otherwise.

### GetVolumeNameOk

`func (o *V1PersistentVolumeClaimSpec) GetVolumeNameOk() (*string, bool)`

GetVolumeNameOk returns a tuple with the VolumeName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVolumeName

`func (o *V1PersistentVolumeClaimSpec) SetVolumeName(v string)`

SetVolumeName sets VolumeName field to given value.

### HasVolumeName

`func (o *V1PersistentVolumeClaimSpec) HasVolumeName() bool`

HasVolumeName returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


