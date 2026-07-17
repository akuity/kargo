# DiscoveryResult

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ArtifactReferences** | Pointer to [**[]ArtifactReference**](ArtifactReference.md) | ArtifactReferences is a list of references to specific versions of an artifact.  +optional | [optional] 
**Name** | Pointer to **string** | SubscriptionName is the name of the Subscription that discovered these results.  +kubebuilder:validation:MinLength&#x3D;1 | [optional] 

## Methods

### NewDiscoveryResult

`func NewDiscoveryResult() *DiscoveryResult`

NewDiscoveryResult instantiates a new DiscoveryResult object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewDiscoveryResultWithDefaults

`func NewDiscoveryResultWithDefaults() *DiscoveryResult`

NewDiscoveryResultWithDefaults instantiates a new DiscoveryResult object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetArtifactReferences

`func (o *DiscoveryResult) GetArtifactReferences() []ArtifactReference`

GetArtifactReferences returns the ArtifactReferences field if non-nil, zero value otherwise.

### GetArtifactReferencesOk

`func (o *DiscoveryResult) GetArtifactReferencesOk() (*[]ArtifactReference, bool)`

GetArtifactReferencesOk returns a tuple with the ArtifactReferences field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetArtifactReferences

`func (o *DiscoveryResult) SetArtifactReferences(v []ArtifactReference)`

SetArtifactReferences sets ArtifactReferences field to given value.

### HasArtifactReferences

`func (o *DiscoveryResult) HasArtifactReferences() bool`

HasArtifactReferences returns a boolean if a field has been set.

### GetName

`func (o *DiscoveryResult) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *DiscoveryResult) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *DiscoveryResult) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *DiscoveryResult) HasName() bool`

HasName returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


