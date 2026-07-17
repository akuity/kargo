# Freight

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Alias** | Pointer to **string** | Alias is a human-friendly alias for a piece of Freight. This is an optional field. A defaulting webhook will sync this field with the value of the kargo.akuity.io/alias label. When the alias label is not present or differs from the value of this field, the defaulting webhook will set the label to the value of this field. If the alias label is present and this field is empty, the defaulting webhook will set the value of this field to the value of the alias label. If this field is empty and the alias label is not present, the defaulting webhook will choose an available alias and assign it to both the field and label. | [optional] 
**ApiVersion** | Pointer to **string** | APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources +optional | [optional] 
**Artifacts** | Pointer to [**[]ArtifactReference**](ArtifactReference.md) | Artifacts describes specific versions of artifacts other than Git repository commits, container images, and Helm charts. | [optional] 
**Charts** | Pointer to [**[]Chart**](Chart.md) | Charts describes specific versions of specific Helm charts. | [optional] 
**Commits** | Pointer to [**[]GitCommit**](GitCommit.md) | Commits describes specific Git repository commits. | [optional] 
**DiscoveredAt** | Pointer to **string** | DiscoveredAt is the time at which this Freight was discovered/created. A defaulting webhook initializes this to the creation time of the Freight.  +optional | [optional] 
**Images** | Pointer to [**[]Image**](Image.md) | Images describes specific versions of specific container images. | [optional] 
**Kind** | Pointer to **string** | Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds +optional | [optional] 
**Metadata** | Pointer to [**V1ObjectMeta**](V1ObjectMeta.md) |  | [optional] 
**Origin** | [**FreightOrigin**](FreightOrigin.md) | Origin describes a kind of Freight in terms of its origin.  +kubebuilder:validation:Required | 
**Status** | Pointer to [**FreightStatus**](FreightStatus.md) | Status describes the current status of this Freight. | [optional] 

## Methods

### NewFreight

`func NewFreight(origin FreightOrigin, ) *Freight`

NewFreight instantiates a new Freight object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewFreightWithDefaults

`func NewFreightWithDefaults() *Freight`

NewFreightWithDefaults instantiates a new Freight object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAlias

`func (o *Freight) GetAlias() string`

GetAlias returns the Alias field if non-nil, zero value otherwise.

### GetAliasOk

`func (o *Freight) GetAliasOk() (*string, bool)`

GetAliasOk returns a tuple with the Alias field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAlias

`func (o *Freight) SetAlias(v string)`

SetAlias sets Alias field to given value.

### HasAlias

`func (o *Freight) HasAlias() bool`

HasAlias returns a boolean if a field has been set.

### GetApiVersion

`func (o *Freight) GetApiVersion() string`

GetApiVersion returns the ApiVersion field if non-nil, zero value otherwise.

### GetApiVersionOk

`func (o *Freight) GetApiVersionOk() (*string, bool)`

GetApiVersionOk returns a tuple with the ApiVersion field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetApiVersion

`func (o *Freight) SetApiVersion(v string)`

SetApiVersion sets ApiVersion field to given value.

### HasApiVersion

`func (o *Freight) HasApiVersion() bool`

HasApiVersion returns a boolean if a field has been set.

### GetArtifacts

`func (o *Freight) GetArtifacts() []ArtifactReference`

GetArtifacts returns the Artifacts field if non-nil, zero value otherwise.

### GetArtifactsOk

`func (o *Freight) GetArtifactsOk() (*[]ArtifactReference, bool)`

GetArtifactsOk returns a tuple with the Artifacts field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetArtifacts

`func (o *Freight) SetArtifacts(v []ArtifactReference)`

SetArtifacts sets Artifacts field to given value.

### HasArtifacts

`func (o *Freight) HasArtifacts() bool`

HasArtifacts returns a boolean if a field has been set.

### GetCharts

`func (o *Freight) GetCharts() []Chart`

GetCharts returns the Charts field if non-nil, zero value otherwise.

### GetChartsOk

`func (o *Freight) GetChartsOk() (*[]Chart, bool)`

GetChartsOk returns a tuple with the Charts field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCharts

`func (o *Freight) SetCharts(v []Chart)`

SetCharts sets Charts field to given value.

### HasCharts

`func (o *Freight) HasCharts() bool`

HasCharts returns a boolean if a field has been set.

### GetCommits

`func (o *Freight) GetCommits() []GitCommit`

GetCommits returns the Commits field if non-nil, zero value otherwise.

### GetCommitsOk

`func (o *Freight) GetCommitsOk() (*[]GitCommit, bool)`

GetCommitsOk returns a tuple with the Commits field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCommits

`func (o *Freight) SetCommits(v []GitCommit)`

SetCommits sets Commits field to given value.

### HasCommits

`func (o *Freight) HasCommits() bool`

HasCommits returns a boolean if a field has been set.

### GetDiscoveredAt

`func (o *Freight) GetDiscoveredAt() string`

GetDiscoveredAt returns the DiscoveredAt field if non-nil, zero value otherwise.

### GetDiscoveredAtOk

`func (o *Freight) GetDiscoveredAtOk() (*string, bool)`

GetDiscoveredAtOk returns a tuple with the DiscoveredAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDiscoveredAt

`func (o *Freight) SetDiscoveredAt(v string)`

SetDiscoveredAt sets DiscoveredAt field to given value.

### HasDiscoveredAt

`func (o *Freight) HasDiscoveredAt() bool`

HasDiscoveredAt returns a boolean if a field has been set.

### GetImages

`func (o *Freight) GetImages() []Image`

GetImages returns the Images field if non-nil, zero value otherwise.

### GetImagesOk

`func (o *Freight) GetImagesOk() (*[]Image, bool)`

GetImagesOk returns a tuple with the Images field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetImages

`func (o *Freight) SetImages(v []Image)`

SetImages sets Images field to given value.

### HasImages

`func (o *Freight) HasImages() bool`

HasImages returns a boolean if a field has been set.

### GetKind

`func (o *Freight) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *Freight) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *Freight) SetKind(v string)`

SetKind sets Kind field to given value.

### HasKind

`func (o *Freight) HasKind() bool`

HasKind returns a boolean if a field has been set.

### GetMetadata

`func (o *Freight) GetMetadata() V1ObjectMeta`

GetMetadata returns the Metadata field if non-nil, zero value otherwise.

### GetMetadataOk

`func (o *Freight) GetMetadataOk() (*V1ObjectMeta, bool)`

GetMetadataOk returns a tuple with the Metadata field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMetadata

`func (o *Freight) SetMetadata(v V1ObjectMeta)`

SetMetadata sets Metadata field to given value.

### HasMetadata

`func (o *Freight) HasMetadata() bool`

HasMetadata returns a boolean if a field has been set.

### GetOrigin

`func (o *Freight) GetOrigin() FreightOrigin`

GetOrigin returns the Origin field if non-nil, zero value otherwise.

### GetOriginOk

`func (o *Freight) GetOriginOk() (*FreightOrigin, bool)`

GetOriginOk returns a tuple with the Origin field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOrigin

`func (o *Freight) SetOrigin(v FreightOrigin)`

SetOrigin sets Origin field to given value.


### GetStatus

`func (o *Freight) GetStatus() FreightStatus`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *Freight) GetStatusOk() (*FreightStatus, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *Freight) SetStatus(v FreightStatus)`

SetStatus sets Status field to given value.

### HasStatus

`func (o *Freight) HasStatus() bool`

HasStatus returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


