# V1ProjectedVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**DefaultMode** | Pointer to **int32** | defaultMode are the mode bits used to set permissions on created files by default. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set. +optional | [optional] 
**Sources** | Pointer to [**[]V1VolumeProjection**](V1VolumeProjection.md) | sources is the list of volume projections. Each entry in this list handles one source. +optional +listType&#x3D;atomic | [optional] 

## Methods

### NewV1ProjectedVolumeSource

`func NewV1ProjectedVolumeSource() *V1ProjectedVolumeSource`

NewV1ProjectedVolumeSource instantiates a new V1ProjectedVolumeSource object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1ProjectedVolumeSourceWithDefaults

`func NewV1ProjectedVolumeSourceWithDefaults() *V1ProjectedVolumeSource`

NewV1ProjectedVolumeSourceWithDefaults instantiates a new V1ProjectedVolumeSource object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDefaultMode

`func (o *V1ProjectedVolumeSource) GetDefaultMode() int32`

GetDefaultMode returns the DefaultMode field if non-nil, zero value otherwise.

### GetDefaultModeOk

`func (o *V1ProjectedVolumeSource) GetDefaultModeOk() (*int32, bool)`

GetDefaultModeOk returns a tuple with the DefaultMode field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDefaultMode

`func (o *V1ProjectedVolumeSource) SetDefaultMode(v int32)`

SetDefaultMode sets DefaultMode field to given value.

### HasDefaultMode

`func (o *V1ProjectedVolumeSource) HasDefaultMode() bool`

HasDefaultMode returns a boolean if a field has been set.

### GetSources

`func (o *V1ProjectedVolumeSource) GetSources() []V1VolumeProjection`

GetSources returns the Sources field if non-nil, zero value otherwise.

### GetSourcesOk

`func (o *V1ProjectedVolumeSource) GetSourcesOk() (*[]V1VolumeProjection, bool)`

GetSourcesOk returns a tuple with the Sources field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSources

`func (o *V1ProjectedVolumeSource) SetSources(v []V1VolumeProjection)`

SetSources sets Sources field to given value.

### HasSources

`func (o *V1ProjectedVolumeSource) HasSources() bool`

HasSources returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


