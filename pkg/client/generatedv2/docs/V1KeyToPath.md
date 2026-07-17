# V1KeyToPath

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Key** | Pointer to **string** | key is the key to project. | [optional] 
**Mode** | Pointer to **int32** | mode is Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set. +optional | [optional] 
**Path** | Pointer to **string** | path is the relative path of the file to map the key to. May not be an absolute path. May not contain the path element &#39;..&#39;. May not start with the string &#39;..&#39;. | [optional] 

## Methods

### NewV1KeyToPath

`func NewV1KeyToPath() *V1KeyToPath`

NewV1KeyToPath instantiates a new V1KeyToPath object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1KeyToPathWithDefaults

`func NewV1KeyToPathWithDefaults() *V1KeyToPath`

NewV1KeyToPathWithDefaults instantiates a new V1KeyToPath object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetKey

`func (o *V1KeyToPath) GetKey() string`

GetKey returns the Key field if non-nil, zero value otherwise.

### GetKeyOk

`func (o *V1KeyToPath) GetKeyOk() (*string, bool)`

GetKeyOk returns a tuple with the Key field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKey

`func (o *V1KeyToPath) SetKey(v string)`

SetKey sets Key field to given value.

### HasKey

`func (o *V1KeyToPath) HasKey() bool`

HasKey returns a boolean if a field has been set.

### GetMode

`func (o *V1KeyToPath) GetMode() int32`

GetMode returns the Mode field if non-nil, zero value otherwise.

### GetModeOk

`func (o *V1KeyToPath) GetModeOk() (*int32, bool)`

GetModeOk returns a tuple with the Mode field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMode

`func (o *V1KeyToPath) SetMode(v int32)`

SetMode sets Mode field to given value.

### HasMode

`func (o *V1KeyToPath) HasMode() bool`

HasMode returns a boolean if a field has been set.

### GetPath

`func (o *V1KeyToPath) GetPath() string`

GetPath returns the Path field if non-nil, zero value otherwise.

### GetPathOk

`func (o *V1KeyToPath) GetPathOk() (*string, bool)`

GetPathOk returns a tuple with the Path field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPath

`func (o *V1KeyToPath) SetPath(v string)`

SetPath sets Path field to given value.

### HasPath

`func (o *V1KeyToPath) HasPath() bool`

HasPath returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


