# V1ConfigMapEnvSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | Pointer to **string** | Name of the referent. This field is effectively required, but due to backwards compatibility is allowed to be empty. Instances of this type with an empty value here are almost certainly wrong. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names +optional +default&#x3D;\&quot;\&quot; +kubebuilder:default&#x3D;\&quot;\&quot; TODO: Drop &#x60;kubebuilder:default&#x60; when controller-gen doesn&#39;t need it https://github.com/kubernetes-sigs/kubebuilder/issues/3896. | [optional] 
**Optional** | Pointer to **bool** | Specify whether the ConfigMap must be defined +optional | [optional] 

## Methods

### NewV1ConfigMapEnvSource

`func NewV1ConfigMapEnvSource() *V1ConfigMapEnvSource`

NewV1ConfigMapEnvSource instantiates a new V1ConfigMapEnvSource object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1ConfigMapEnvSourceWithDefaults

`func NewV1ConfigMapEnvSourceWithDefaults() *V1ConfigMapEnvSource`

NewV1ConfigMapEnvSourceWithDefaults instantiates a new V1ConfigMapEnvSource object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetName

`func (o *V1ConfigMapEnvSource) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *V1ConfigMapEnvSource) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *V1ConfigMapEnvSource) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *V1ConfigMapEnvSource) HasName() bool`

HasName returns a boolean if a field has been set.

### GetOptional

`func (o *V1ConfigMapEnvSource) GetOptional() bool`

GetOptional returns the Optional field if non-nil, zero value otherwise.

### GetOptionalOk

`func (o *V1ConfigMapEnvSource) GetOptionalOk() (*bool, bool)`

GetOptionalOk returns a tuple with the Optional field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOptional

`func (o *V1ConfigMapEnvSource) SetOptional(v bool)`

SetOptional sets Optional field to given value.

### HasOptional

`func (o *V1ConfigMapEnvSource) HasOptional() bool`

HasOptional returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


