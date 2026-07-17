# V1FileKeySelector

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Key** | Pointer to **string** | The key within the env file. An invalid key will prevent the pod from starting. The keys defined within a source may consist of any printable ASCII characters except &#39;&#x3D;&#39;. During Alpha stage of the EnvFiles feature gate, the key size is limited to 128 characters. +required | [optional] 
**Optional** | Pointer to **bool** | Specify whether the file or its key must be defined. If the file or key does not exist, then the env var is not published. If optional is set to true and the specified key does not exist, the environment variable will not be set in the Pod&#39;s containers.  If optional is set to false and the specified key does not exist, an error will be returned during Pod creation. +optional +default&#x3D;false | [optional] 
**Path** | Pointer to **string** | The path within the volume from which to select the file. Must be relative and may not contain the &#39;..&#39; path or start with &#39;..&#39;. +required | [optional] 
**VolumeName** | Pointer to **string** | The name of the volume mount containing the env file. +required | [optional] 

## Methods

### NewV1FileKeySelector

`func NewV1FileKeySelector() *V1FileKeySelector`

NewV1FileKeySelector instantiates a new V1FileKeySelector object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1FileKeySelectorWithDefaults

`func NewV1FileKeySelectorWithDefaults() *V1FileKeySelector`

NewV1FileKeySelectorWithDefaults instantiates a new V1FileKeySelector object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetKey

`func (o *V1FileKeySelector) GetKey() string`

GetKey returns the Key field if non-nil, zero value otherwise.

### GetKeyOk

`func (o *V1FileKeySelector) GetKeyOk() (*string, bool)`

GetKeyOk returns a tuple with the Key field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKey

`func (o *V1FileKeySelector) SetKey(v string)`

SetKey sets Key field to given value.

### HasKey

`func (o *V1FileKeySelector) HasKey() bool`

HasKey returns a boolean if a field has been set.

### GetOptional

`func (o *V1FileKeySelector) GetOptional() bool`

GetOptional returns the Optional field if non-nil, zero value otherwise.

### GetOptionalOk

`func (o *V1FileKeySelector) GetOptionalOk() (*bool, bool)`

GetOptionalOk returns a tuple with the Optional field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOptional

`func (o *V1FileKeySelector) SetOptional(v bool)`

SetOptional sets Optional field to given value.

### HasOptional

`func (o *V1FileKeySelector) HasOptional() bool`

HasOptional returns a boolean if a field has been set.

### GetPath

`func (o *V1FileKeySelector) GetPath() string`

GetPath returns the Path field if non-nil, zero value otherwise.

### GetPathOk

`func (o *V1FileKeySelector) GetPathOk() (*string, bool)`

GetPathOk returns a tuple with the Path field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPath

`func (o *V1FileKeySelector) SetPath(v string)`

SetPath sets Path field to given value.

### HasPath

`func (o *V1FileKeySelector) HasPath() bool`

HasPath returns a boolean if a field has been set.

### GetVolumeName

`func (o *V1FileKeySelector) GetVolumeName() string`

GetVolumeName returns the VolumeName field if non-nil, zero value otherwise.

### GetVolumeNameOk

`func (o *V1FileKeySelector) GetVolumeNameOk() (*string, bool)`

GetVolumeNameOk returns a tuple with the VolumeName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVolumeName

`func (o *V1FileKeySelector) SetVolumeName(v string)`

SetVolumeName sets VolumeName field to given value.

### HasVolumeName

`func (o *V1FileKeySelector) HasVolumeName() bool`

HasVolumeName returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


