# ArtifactReference

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ArtifactType** | Pointer to **string** | ArtifactType specifies the type of artifact this is. Often, but not always, it will be the media type (MIME type) of the artifact referenced by this ArtifactReference.  +kubebuilder:validation:MinLength&#x3D;1 | [optional] 
**Metadata** | Pointer to **interface{}** | Metadata is a JSON object containing a mostly opaque collection of artifact attributes. (It must be an object. It may not be a list or a scalar value.) \&quot;Mostly\&quot; because Kargo may understand how to interpret some documented, well-known, top-level keys. Those aside, this metadata is only understood by a corresponding Subscriber implementation that created it.  +optional | [optional] 
**SubscriptionName** | Pointer to **string** | SubscriptionName is the name of the Subscription that discovered this artifact.  +kubebuilder:validation:MinLength&#x3D;1 | [optional] 
**Version** | Pointer to **string** | Version identifies a specific revision of this artifact.  +kubebuilder:validation:MinLength&#x3D;1 | [optional] 

## Methods

### NewArtifactReference

`func NewArtifactReference() *ArtifactReference`

NewArtifactReference instantiates a new ArtifactReference object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewArtifactReferenceWithDefaults

`func NewArtifactReferenceWithDefaults() *ArtifactReference`

NewArtifactReferenceWithDefaults instantiates a new ArtifactReference object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetArtifactType

`func (o *ArtifactReference) GetArtifactType() string`

GetArtifactType returns the ArtifactType field if non-nil, zero value otherwise.

### GetArtifactTypeOk

`func (o *ArtifactReference) GetArtifactTypeOk() (*string, bool)`

GetArtifactTypeOk returns a tuple with the ArtifactType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetArtifactType

`func (o *ArtifactReference) SetArtifactType(v string)`

SetArtifactType sets ArtifactType field to given value.

### HasArtifactType

`func (o *ArtifactReference) HasArtifactType() bool`

HasArtifactType returns a boolean if a field has been set.

### GetMetadata

`func (o *ArtifactReference) GetMetadata() interface{}`

GetMetadata returns the Metadata field if non-nil, zero value otherwise.

### GetMetadataOk

`func (o *ArtifactReference) GetMetadataOk() (*interface{}, bool)`

GetMetadataOk returns a tuple with the Metadata field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMetadata

`func (o *ArtifactReference) SetMetadata(v interface{})`

SetMetadata sets Metadata field to given value.

### HasMetadata

`func (o *ArtifactReference) HasMetadata() bool`

HasMetadata returns a boolean if a field has been set.

### GetSubscriptionName

`func (o *ArtifactReference) GetSubscriptionName() string`

GetSubscriptionName returns the SubscriptionName field if non-nil, zero value otherwise.

### GetSubscriptionNameOk

`func (o *ArtifactReference) GetSubscriptionNameOk() (*string, bool)`

GetSubscriptionNameOk returns a tuple with the SubscriptionName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSubscriptionName

`func (o *ArtifactReference) SetSubscriptionName(v string)`

SetSubscriptionName sets SubscriptionName field to given value.

### HasSubscriptionName

`func (o *ArtifactReference) HasSubscriptionName() bool`

HasSubscriptionName returns a boolean if a field has been set.

### GetVersion

`func (o *ArtifactReference) GetVersion() string`

GetVersion returns the Version field if non-nil, zero value otherwise.

### GetVersionOk

`func (o *ArtifactReference) GetVersionOk() (*string, bool)`

GetVersionOk returns a tuple with the Version field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVersion

`func (o *ArtifactReference) SetVersion(v string)`

SetVersion sets Version field to given value.

### HasVersion

`func (o *ArtifactReference) HasVersion() bool`

HasVersion returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


