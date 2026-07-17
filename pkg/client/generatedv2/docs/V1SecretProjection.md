# V1SecretProjection

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Items** | Pointer to [**[]V1KeyToPath**](V1KeyToPath.md) | items if unspecified, each key-value pair in the Data field of the referenced Secret will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the Secret, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the &#39;..&#39; path or start with &#39;..&#39;. +optional +listType&#x3D;atomic | [optional] 
**Name** | Pointer to **string** | Name of the referent. This field is effectively required, but due to backwards compatibility is allowed to be empty. Instances of this type with an empty value here are almost certainly wrong. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names +optional +default&#x3D;\&quot;\&quot; +kubebuilder:default&#x3D;\&quot;\&quot; TODO: Drop &#x60;kubebuilder:default&#x60; when controller-gen doesn&#39;t need it https://github.com/kubernetes-sigs/kubebuilder/issues/3896. | [optional] 
**Optional** | Pointer to **bool** | optional field specify whether the Secret or its key must be defined +optional | [optional] 

## Methods

### NewV1SecretProjection

`func NewV1SecretProjection() *V1SecretProjection`

NewV1SecretProjection instantiates a new V1SecretProjection object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1SecretProjectionWithDefaults

`func NewV1SecretProjectionWithDefaults() *V1SecretProjection`

NewV1SecretProjectionWithDefaults instantiates a new V1SecretProjection object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetItems

`func (o *V1SecretProjection) GetItems() []V1KeyToPath`

GetItems returns the Items field if non-nil, zero value otherwise.

### GetItemsOk

`func (o *V1SecretProjection) GetItemsOk() (*[]V1KeyToPath, bool)`

GetItemsOk returns a tuple with the Items field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetItems

`func (o *V1SecretProjection) SetItems(v []V1KeyToPath)`

SetItems sets Items field to given value.

### HasItems

`func (o *V1SecretProjection) HasItems() bool`

HasItems returns a boolean if a field has been set.

### GetName

`func (o *V1SecretProjection) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *V1SecretProjection) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *V1SecretProjection) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *V1SecretProjection) HasName() bool`

HasName returns a boolean if a field has been set.

### GetOptional

`func (o *V1SecretProjection) GetOptional() bool`

GetOptional returns the Optional field if non-nil, zero value otherwise.

### GetOptionalOk

`func (o *V1SecretProjection) GetOptionalOk() (*bool, bool)`

GetOptionalOk returns a tuple with the Optional field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOptional

`func (o *V1SecretProjection) SetOptional(v bool)`

SetOptional sets Optional field to given value.

### HasOptional

`func (o *V1SecretProjection) HasOptional() bool`

HasOptional returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


