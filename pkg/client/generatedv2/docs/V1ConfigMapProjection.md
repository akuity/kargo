# V1ConfigMapProjection

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Items** | Pointer to [**[]V1KeyToPath**](V1KeyToPath.md) | items if unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the ConfigMap, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the &#39;..&#39; path or start with &#39;..&#39;. +optional +listType&#x3D;atomic | [optional] 
**Name** | Pointer to **string** | Name of the referent. This field is effectively required, but due to backwards compatibility is allowed to be empty. Instances of this type with an empty value here are almost certainly wrong. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names +optional +default&#x3D;\&quot;\&quot; +kubebuilder:default&#x3D;\&quot;\&quot; TODO: Drop &#x60;kubebuilder:default&#x60; when controller-gen doesn&#39;t need it https://github.com/kubernetes-sigs/kubebuilder/issues/3896. | [optional] 
**Optional** | Pointer to **bool** | optional specify whether the ConfigMap or its keys must be defined +optional | [optional] 

## Methods

### NewV1ConfigMapProjection

`func NewV1ConfigMapProjection() *V1ConfigMapProjection`

NewV1ConfigMapProjection instantiates a new V1ConfigMapProjection object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1ConfigMapProjectionWithDefaults

`func NewV1ConfigMapProjectionWithDefaults() *V1ConfigMapProjection`

NewV1ConfigMapProjectionWithDefaults instantiates a new V1ConfigMapProjection object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetItems

`func (o *V1ConfigMapProjection) GetItems() []V1KeyToPath`

GetItems returns the Items field if non-nil, zero value otherwise.

### GetItemsOk

`func (o *V1ConfigMapProjection) GetItemsOk() (*[]V1KeyToPath, bool)`

GetItemsOk returns a tuple with the Items field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetItems

`func (o *V1ConfigMapProjection) SetItems(v []V1KeyToPath)`

SetItems sets Items field to given value.

### HasItems

`func (o *V1ConfigMapProjection) HasItems() bool`

HasItems returns a boolean if a field has been set.

### GetName

`func (o *V1ConfigMapProjection) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *V1ConfigMapProjection) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *V1ConfigMapProjection) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *V1ConfigMapProjection) HasName() bool`

HasName returns a boolean if a field has been set.

### GetOptional

`func (o *V1ConfigMapProjection) GetOptional() bool`

GetOptional returns the Optional field if non-nil, zero value otherwise.

### GetOptionalOk

`func (o *V1ConfigMapProjection) GetOptionalOk() (*bool, bool)`

GetOptionalOk returns a tuple with the Optional field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOptional

`func (o *V1ConfigMapProjection) SetOptional(v bool)`

SetOptional sets Optional field to given value.

### HasOptional

`func (o *V1ConfigMapProjection) HasOptional() bool`

HasOptional returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


