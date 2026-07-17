# V1DownwardAPIVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**DefaultMode** | Pointer to **int32** | Optional: mode bits to use on created files by default. Must be a Optional: mode bits used to set permissions on created files by default. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. Defaults to 0644. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set. +optional | [optional] 
**Items** | Pointer to [**[]V1DownwardAPIVolumeFile**](V1DownwardAPIVolumeFile.md) | Items is a list of downward API volume file +optional +listType&#x3D;atomic | [optional] 

## Methods

### NewV1DownwardAPIVolumeSource

`func NewV1DownwardAPIVolumeSource() *V1DownwardAPIVolumeSource`

NewV1DownwardAPIVolumeSource instantiates a new V1DownwardAPIVolumeSource object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1DownwardAPIVolumeSourceWithDefaults

`func NewV1DownwardAPIVolumeSourceWithDefaults() *V1DownwardAPIVolumeSource`

NewV1DownwardAPIVolumeSourceWithDefaults instantiates a new V1DownwardAPIVolumeSource object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDefaultMode

`func (o *V1DownwardAPIVolumeSource) GetDefaultMode() int32`

GetDefaultMode returns the DefaultMode field if non-nil, zero value otherwise.

### GetDefaultModeOk

`func (o *V1DownwardAPIVolumeSource) GetDefaultModeOk() (*int32, bool)`

GetDefaultModeOk returns a tuple with the DefaultMode field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDefaultMode

`func (o *V1DownwardAPIVolumeSource) SetDefaultMode(v int32)`

SetDefaultMode sets DefaultMode field to given value.

### HasDefaultMode

`func (o *V1DownwardAPIVolumeSource) HasDefaultMode() bool`

HasDefaultMode returns a boolean if a field has been set.

### GetItems

`func (o *V1DownwardAPIVolumeSource) GetItems() []V1DownwardAPIVolumeFile`

GetItems returns the Items field if non-nil, zero value otherwise.

### GetItemsOk

`func (o *V1DownwardAPIVolumeSource) GetItemsOk() (*[]V1DownwardAPIVolumeFile, bool)`

GetItemsOk returns a tuple with the Items field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetItems

`func (o *V1DownwardAPIVolumeSource) SetItems(v []V1DownwardAPIVolumeFile)`

SetItems sets Items field to given value.

### HasItems

`func (o *V1DownwardAPIVolumeSource) HasItems() bool`

HasItems returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


