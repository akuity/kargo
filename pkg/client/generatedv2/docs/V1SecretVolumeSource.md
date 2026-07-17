# V1SecretVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**DefaultMode** | Pointer to **int32** | defaultMode is Optional: mode bits used to set permissions on created files by default. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. Defaults to 0644. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set. +optional | [optional] 
**Items** | Pointer to [**[]V1KeyToPath**](V1KeyToPath.md) | items If unspecified, each key-value pair in the Data field of the referenced Secret will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the Secret, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the &#39;..&#39; path or start with &#39;..&#39;. +optional +listType&#x3D;atomic | [optional] 
**Optional** | Pointer to **bool** | optional field specify whether the Secret or its keys must be defined +optional | [optional] 
**SecretName** | Pointer to **string** | secretName is the name of the secret in the pod&#39;s namespace to use. More info: https://kubernetes.io/docs/concepts/storage/volumes#secret +optional | [optional] 

## Methods

### NewV1SecretVolumeSource

`func NewV1SecretVolumeSource() *V1SecretVolumeSource`

NewV1SecretVolumeSource instantiates a new V1SecretVolumeSource object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1SecretVolumeSourceWithDefaults

`func NewV1SecretVolumeSourceWithDefaults() *V1SecretVolumeSource`

NewV1SecretVolumeSourceWithDefaults instantiates a new V1SecretVolumeSource object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDefaultMode

`func (o *V1SecretVolumeSource) GetDefaultMode() int32`

GetDefaultMode returns the DefaultMode field if non-nil, zero value otherwise.

### GetDefaultModeOk

`func (o *V1SecretVolumeSource) GetDefaultModeOk() (*int32, bool)`

GetDefaultModeOk returns a tuple with the DefaultMode field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDefaultMode

`func (o *V1SecretVolumeSource) SetDefaultMode(v int32)`

SetDefaultMode sets DefaultMode field to given value.

### HasDefaultMode

`func (o *V1SecretVolumeSource) HasDefaultMode() bool`

HasDefaultMode returns a boolean if a field has been set.

### GetItems

`func (o *V1SecretVolumeSource) GetItems() []V1KeyToPath`

GetItems returns the Items field if non-nil, zero value otherwise.

### GetItemsOk

`func (o *V1SecretVolumeSource) GetItemsOk() (*[]V1KeyToPath, bool)`

GetItemsOk returns a tuple with the Items field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetItems

`func (o *V1SecretVolumeSource) SetItems(v []V1KeyToPath)`

SetItems sets Items field to given value.

### HasItems

`func (o *V1SecretVolumeSource) HasItems() bool`

HasItems returns a boolean if a field has been set.

### GetOptional

`func (o *V1SecretVolumeSource) GetOptional() bool`

GetOptional returns the Optional field if non-nil, zero value otherwise.

### GetOptionalOk

`func (o *V1SecretVolumeSource) GetOptionalOk() (*bool, bool)`

GetOptionalOk returns a tuple with the Optional field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOptional

`func (o *V1SecretVolumeSource) SetOptional(v bool)`

SetOptional sets Optional field to given value.

### HasOptional

`func (o *V1SecretVolumeSource) HasOptional() bool`

HasOptional returns a boolean if a field has been set.

### GetSecretName

`func (o *V1SecretVolumeSource) GetSecretName() string`

GetSecretName returns the SecretName field if non-nil, zero value otherwise.

### GetSecretNameOk

`func (o *V1SecretVolumeSource) GetSecretNameOk() (*string, bool)`

GetSecretNameOk returns a tuple with the SecretName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecretName

`func (o *V1SecretVolumeSource) SetSecretName(v string)`

SetSecretName sets SecretName field to given value.

### HasSecretName

`func (o *V1SecretVolumeSource) HasSecretName() bool`

HasSecretName returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


