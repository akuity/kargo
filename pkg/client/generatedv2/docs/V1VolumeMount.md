# V1VolumeMount

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**MountPath** | Pointer to **string** | Path within the container at which the volume should be mounted.  Must not contain &#39;:&#39;. | [optional] 
**MountPropagation** | Pointer to **string** | mountPropagation determines how mounts are propagated from the host to container and the other way around. When not set, MountPropagationNone is used. This field is beta in 1.10. When RecursiveReadOnly is set to IfPossible or to Enabled, MountPropagation must be None or unspecified (which defaults to None). +optional | [optional] 
**Name** | Pointer to **string** | This must match the Name of a Volume. | [optional] 
**ReadOnly** | Pointer to **bool** | Mounted read-only if true, read-write otherwise (false or unspecified). Defaults to false. +optional | [optional] 
**RecursiveReadOnly** | Pointer to **string** | RecursiveReadOnly specifies whether read-only mounts should be handled recursively.  If ReadOnly is false, this field has no meaning and must be unspecified.  If ReadOnly is true, and this field is set to Disabled, the mount is not made recursively read-only.  If this field is set to IfPossible, the mount is made recursively read-only, if it is supported by the container runtime.  If this field is set to Enabled, the mount is made recursively read-only if it is supported by the container runtime, otherwise the pod will not be started and an error will be generated to indicate the reason.  If this field is set to IfPossible or Enabled, MountPropagation must be set to None (or be unspecified, which defaults to None).  If this field is not specified, it is treated as an equivalent of Disabled.  +featureGate&#x3D;RecursiveReadOnlyMounts +optional | [optional] 
**SubPath** | Pointer to **string** | Path within the volume from which the container&#39;s volume should be mounted. Defaults to \&quot;\&quot; (volume&#39;s root). +optional | [optional] 
**SubPathExpr** | Pointer to **string** | Expanded path within the volume from which the container&#39;s volume should be mounted. Behaves similarly to SubPath but environment variable references $(VAR_NAME) are expanded using the container&#39;s environment. Defaults to \&quot;\&quot; (volume&#39;s root). SubPathExpr and SubPath are mutually exclusive. +optional | [optional] 

## Methods

### NewV1VolumeMount

`func NewV1VolumeMount() *V1VolumeMount`

NewV1VolumeMount instantiates a new V1VolumeMount object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1VolumeMountWithDefaults

`func NewV1VolumeMountWithDefaults() *V1VolumeMount`

NewV1VolumeMountWithDefaults instantiates a new V1VolumeMount object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetMountPath

`func (o *V1VolumeMount) GetMountPath() string`

GetMountPath returns the MountPath field if non-nil, zero value otherwise.

### GetMountPathOk

`func (o *V1VolumeMount) GetMountPathOk() (*string, bool)`

GetMountPathOk returns a tuple with the MountPath field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMountPath

`func (o *V1VolumeMount) SetMountPath(v string)`

SetMountPath sets MountPath field to given value.

### HasMountPath

`func (o *V1VolumeMount) HasMountPath() bool`

HasMountPath returns a boolean if a field has been set.

### GetMountPropagation

`func (o *V1VolumeMount) GetMountPropagation() string`

GetMountPropagation returns the MountPropagation field if non-nil, zero value otherwise.

### GetMountPropagationOk

`func (o *V1VolumeMount) GetMountPropagationOk() (*string, bool)`

GetMountPropagationOk returns a tuple with the MountPropagation field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMountPropagation

`func (o *V1VolumeMount) SetMountPropagation(v string)`

SetMountPropagation sets MountPropagation field to given value.

### HasMountPropagation

`func (o *V1VolumeMount) HasMountPropagation() bool`

HasMountPropagation returns a boolean if a field has been set.

### GetName

`func (o *V1VolumeMount) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *V1VolumeMount) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *V1VolumeMount) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *V1VolumeMount) HasName() bool`

HasName returns a boolean if a field has been set.

### GetReadOnly

`func (o *V1VolumeMount) GetReadOnly() bool`

GetReadOnly returns the ReadOnly field if non-nil, zero value otherwise.

### GetReadOnlyOk

`func (o *V1VolumeMount) GetReadOnlyOk() (*bool, bool)`

GetReadOnlyOk returns a tuple with the ReadOnly field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReadOnly

`func (o *V1VolumeMount) SetReadOnly(v bool)`

SetReadOnly sets ReadOnly field to given value.

### HasReadOnly

`func (o *V1VolumeMount) HasReadOnly() bool`

HasReadOnly returns a boolean if a field has been set.

### GetRecursiveReadOnly

`func (o *V1VolumeMount) GetRecursiveReadOnly() string`

GetRecursiveReadOnly returns the RecursiveReadOnly field if non-nil, zero value otherwise.

### GetRecursiveReadOnlyOk

`func (o *V1VolumeMount) GetRecursiveReadOnlyOk() (*string, bool)`

GetRecursiveReadOnlyOk returns a tuple with the RecursiveReadOnly field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRecursiveReadOnly

`func (o *V1VolumeMount) SetRecursiveReadOnly(v string)`

SetRecursiveReadOnly sets RecursiveReadOnly field to given value.

### HasRecursiveReadOnly

`func (o *V1VolumeMount) HasRecursiveReadOnly() bool`

HasRecursiveReadOnly returns a boolean if a field has been set.

### GetSubPath

`func (o *V1VolumeMount) GetSubPath() string`

GetSubPath returns the SubPath field if non-nil, zero value otherwise.

### GetSubPathOk

`func (o *V1VolumeMount) GetSubPathOk() (*string, bool)`

GetSubPathOk returns a tuple with the SubPath field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSubPath

`func (o *V1VolumeMount) SetSubPath(v string)`

SetSubPath sets SubPath field to given value.

### HasSubPath

`func (o *V1VolumeMount) HasSubPath() bool`

HasSubPath returns a boolean if a field has been set.

### GetSubPathExpr

`func (o *V1VolumeMount) GetSubPathExpr() string`

GetSubPathExpr returns the SubPathExpr field if non-nil, zero value otherwise.

### GetSubPathExprOk

`func (o *V1VolumeMount) GetSubPathExprOk() (*string, bool)`

GetSubPathExprOk returns a tuple with the SubPathExpr field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSubPathExpr

`func (o *V1VolumeMount) SetSubPathExpr(v string)`

SetSubPathExpr sets SubPathExpr field to given value.

### HasSubPathExpr

`func (o *V1VolumeMount) HasSubPathExpr() bool`

HasSubPathExpr returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


