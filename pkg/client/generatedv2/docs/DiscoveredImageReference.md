# DiscoveredImageReference

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Annotations** | Pointer to **map[string]string** | Annotations is a map of key-value pairs that provide additional information about the image. | [optional] 
**CreatedAt** | Pointer to **string** | CreatedAt is the time the image was created. This field is optional, and not populated for every ImageSelectionStrategy. | [optional] 
**Digest** | Pointer to **string** | Digest is the digest of the image.  +kubebuilder:validation:MinLength&#x3D;1 +kubebuilder:validation:Pattern&#x3D;&#x60;^[a-z0-9]+:[a-f0-9]+$&#x60; +akuity:test-kubebuilder-pattern&#x3D;Digest | [optional] 
**Tag** | Pointer to **string** | Tag is the tag of the image.  +kubebuilder:validation:MinLength&#x3D;1 +kubebuilder:validation:MaxLength&#x3D;128 +kubebuilder:validation:Pattern&#x3D;&#x60;^[\\w.\\-\\_]+$&#x60; +akuity:test-kubebuilder-pattern&#x3D;Tag | [optional] 

## Methods

### NewDiscoveredImageReference

`func NewDiscoveredImageReference() *DiscoveredImageReference`

NewDiscoveredImageReference instantiates a new DiscoveredImageReference object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewDiscoveredImageReferenceWithDefaults

`func NewDiscoveredImageReferenceWithDefaults() *DiscoveredImageReference`

NewDiscoveredImageReferenceWithDefaults instantiates a new DiscoveredImageReference object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAnnotations

`func (o *DiscoveredImageReference) GetAnnotations() map[string]string`

GetAnnotations returns the Annotations field if non-nil, zero value otherwise.

### GetAnnotationsOk

`func (o *DiscoveredImageReference) GetAnnotationsOk() (*map[string]string, bool)`

GetAnnotationsOk returns a tuple with the Annotations field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAnnotations

`func (o *DiscoveredImageReference) SetAnnotations(v map[string]string)`

SetAnnotations sets Annotations field to given value.

### HasAnnotations

`func (o *DiscoveredImageReference) HasAnnotations() bool`

HasAnnotations returns a boolean if a field has been set.

### GetCreatedAt

`func (o *DiscoveredImageReference) GetCreatedAt() string`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *DiscoveredImageReference) GetCreatedAtOk() (*string, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *DiscoveredImageReference) SetCreatedAt(v string)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *DiscoveredImageReference) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.

### GetDigest

`func (o *DiscoveredImageReference) GetDigest() string`

GetDigest returns the Digest field if non-nil, zero value otherwise.

### GetDigestOk

`func (o *DiscoveredImageReference) GetDigestOk() (*string, bool)`

GetDigestOk returns a tuple with the Digest field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDigest

`func (o *DiscoveredImageReference) SetDigest(v string)`

SetDigest sets Digest field to given value.

### HasDigest

`func (o *DiscoveredImageReference) HasDigest() bool`

HasDigest returns a boolean if a field has been set.

### GetTag

`func (o *DiscoveredImageReference) GetTag() string`

GetTag returns the Tag field if non-nil, zero value otherwise.

### GetTagOk

`func (o *DiscoveredImageReference) GetTagOk() (*string, bool)`

GetTagOk returns a tuple with the Tag field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTag

`func (o *DiscoveredImageReference) SetTag(v string)`

SetTag sets Tag field to given value.

### HasTag

`func (o *DiscoveredImageReference) HasTag() bool`

HasTag returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


