# V1SecretKeySelector

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Key** | Pointer to **string** | The key of the secret to select from.  Must be a valid secret key. | [optional] 
**Name** | Pointer to **string** | Name of the referent. This field is effectively required, but due to backwards compatibility is allowed to be empty. Instances of this type with an empty value here are almost certainly wrong. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names +optional +default&#x3D;\&quot;\&quot; +kubebuilder:default&#x3D;\&quot;\&quot; TODO: Drop &#x60;kubebuilder:default&#x60; when controller-gen doesn&#39;t need it https://github.com/kubernetes-sigs/kubebuilder/issues/3896. | [optional] 
**Optional** | Pointer to **bool** | Specify whether the Secret or its key must be defined +optional | [optional] 

## Methods

### NewV1SecretKeySelector

`func NewV1SecretKeySelector() *V1SecretKeySelector`

NewV1SecretKeySelector instantiates a new V1SecretKeySelector object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1SecretKeySelectorWithDefaults

`func NewV1SecretKeySelectorWithDefaults() *V1SecretKeySelector`

NewV1SecretKeySelectorWithDefaults instantiates a new V1SecretKeySelector object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetKey

`func (o *V1SecretKeySelector) GetKey() string`

GetKey returns the Key field if non-nil, zero value otherwise.

### GetKeyOk

`func (o *V1SecretKeySelector) GetKeyOk() (*string, bool)`

GetKeyOk returns a tuple with the Key field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKey

`func (o *V1SecretKeySelector) SetKey(v string)`

SetKey sets Key field to given value.

### HasKey

`func (o *V1SecretKeySelector) HasKey() bool`

HasKey returns a boolean if a field has been set.

### GetName

`func (o *V1SecretKeySelector) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *V1SecretKeySelector) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *V1SecretKeySelector) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *V1SecretKeySelector) HasName() bool`

HasName returns a boolean if a field has been set.

### GetOptional

`func (o *V1SecretKeySelector) GetOptional() bool`

GetOptional returns the Optional field if non-nil, zero value otherwise.

### GetOptionalOk

`func (o *V1SecretKeySelector) GetOptionalOk() (*bool, bool)`

GetOptionalOk returns a tuple with the Optional field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOptional

`func (o *V1SecretKeySelector) SetOptional(v bool)`

SetOptional sets Optional field to given value.

### HasOptional

`func (o *V1SecretKeySelector) HasOptional() bool`

HasOptional returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


