# V1VolumeDevice

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**DevicePath** | Pointer to **string** | devicePath is the path inside of the container that the device will be mapped to. | [optional] 
**Name** | Pointer to **string** | name must match the name of a persistentVolumeClaim in the pod | [optional] 

## Methods

### NewV1VolumeDevice

`func NewV1VolumeDevice() *V1VolumeDevice`

NewV1VolumeDevice instantiates a new V1VolumeDevice object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1VolumeDeviceWithDefaults

`func NewV1VolumeDeviceWithDefaults() *V1VolumeDevice`

NewV1VolumeDeviceWithDefaults instantiates a new V1VolumeDevice object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDevicePath

`func (o *V1VolumeDevice) GetDevicePath() string`

GetDevicePath returns the DevicePath field if non-nil, zero value otherwise.

### GetDevicePathOk

`func (o *V1VolumeDevice) GetDevicePathOk() (*string, bool)`

GetDevicePathOk returns a tuple with the DevicePath field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDevicePath

`func (o *V1VolumeDevice) SetDevicePath(v string)`

SetDevicePath sets DevicePath field to given value.

### HasDevicePath

`func (o *V1VolumeDevice) HasDevicePath() bool`

HasDevicePath returns a boolean if a field has been set.

### GetName

`func (o *V1VolumeDevice) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *V1VolumeDevice) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *V1VolumeDevice) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *V1VolumeDevice) HasName() bool`

HasName returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


