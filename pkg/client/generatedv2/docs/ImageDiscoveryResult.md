# ImageDiscoveryResult

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Platform** | Pointer to **string** | Platform is the target platform constraint of the ImageSubscription for which references were discovered. This field is optional, and only populated if the ImageSubscription specifies a Platform. | [optional] 
**References** | Pointer to [**[]DiscoveredImageReference**](DiscoveredImageReference.md) | References is a list of image references discovered by the Warehouse for the ImageSubscription. An empty list indicates that the discovery operation was successful, but no images matching the ImageSubscription criteria were found.  +optional | [optional] 
**RepoURL** | Pointer to **string** | RepoURL is the repository URL of the image, as specified in the ImageSubscription.  +kubebuilder:validation:MinLength&#x3D;1 | [optional] 

## Methods

### NewImageDiscoveryResult

`func NewImageDiscoveryResult() *ImageDiscoveryResult`

NewImageDiscoveryResult instantiates a new ImageDiscoveryResult object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewImageDiscoveryResultWithDefaults

`func NewImageDiscoveryResultWithDefaults() *ImageDiscoveryResult`

NewImageDiscoveryResultWithDefaults instantiates a new ImageDiscoveryResult object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetPlatform

`func (o *ImageDiscoveryResult) GetPlatform() string`

GetPlatform returns the Platform field if non-nil, zero value otherwise.

### GetPlatformOk

`func (o *ImageDiscoveryResult) GetPlatformOk() (*string, bool)`

GetPlatformOk returns a tuple with the Platform field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPlatform

`func (o *ImageDiscoveryResult) SetPlatform(v string)`

SetPlatform sets Platform field to given value.

### HasPlatform

`func (o *ImageDiscoveryResult) HasPlatform() bool`

HasPlatform returns a boolean if a field has been set.

### GetReferences

`func (o *ImageDiscoveryResult) GetReferences() []DiscoveredImageReference`

GetReferences returns the References field if non-nil, zero value otherwise.

### GetReferencesOk

`func (o *ImageDiscoveryResult) GetReferencesOk() (*[]DiscoveredImageReference, bool)`

GetReferencesOk returns a tuple with the References field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReferences

`func (o *ImageDiscoveryResult) SetReferences(v []DiscoveredImageReference)`

SetReferences sets References field to given value.

### HasReferences

`func (o *ImageDiscoveryResult) HasReferences() bool`

HasReferences returns a boolean if a field has been set.

### GetRepoURL

`func (o *ImageDiscoveryResult) GetRepoURL() string`

GetRepoURL returns the RepoURL field if non-nil, zero value otherwise.

### GetRepoURLOk

`func (o *ImageDiscoveryResult) GetRepoURLOk() (*string, bool)`

GetRepoURLOk returns a tuple with the RepoURL field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRepoURL

`func (o *ImageDiscoveryResult) SetRepoURL(v string)`

SetRepoURL sets RepoURL field to given value.

### HasRepoURL

`func (o *ImageDiscoveryResult) HasRepoURL() bool`

HasRepoURL returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


