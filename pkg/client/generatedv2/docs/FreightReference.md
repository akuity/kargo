# FreightReference

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Artifacts** | Pointer to [**[]ArtifactReference**](ArtifactReference.md) | Artifacts describes specific versions of artifacts other than Git repository commits, container images, and Helm charts. | [optional] 
**Charts** | Pointer to [**[]Chart**](Chart.md) | Charts describes specific versions of specific Helm charts. | [optional] 
**Commits** | Pointer to [**[]GitCommit**](GitCommit.md) | Commits describes specific Git repository commits. | [optional] 
**Images** | Pointer to [**[]Image**](Image.md) | Images describes specific versions of specific container images. | [optional] 
**Name** | Pointer to **string** | Name is a system-assigned identifier derived deterministically from the contents of the Freight. I.e., two pieces of Freight can be compared for equality by comparing their Names. | [optional] 
**Origin** | Pointer to [**FreightOrigin**](FreightOrigin.md) | Origin describes a kind of Freight in terms of its origin. | [optional] 

## Methods

### NewFreightReference

`func NewFreightReference() *FreightReference`

NewFreightReference instantiates a new FreightReference object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewFreightReferenceWithDefaults

`func NewFreightReferenceWithDefaults() *FreightReference`

NewFreightReferenceWithDefaults instantiates a new FreightReference object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetArtifacts

`func (o *FreightReference) GetArtifacts() []ArtifactReference`

GetArtifacts returns the Artifacts field if non-nil, zero value otherwise.

### GetArtifactsOk

`func (o *FreightReference) GetArtifactsOk() (*[]ArtifactReference, bool)`

GetArtifactsOk returns a tuple with the Artifacts field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetArtifacts

`func (o *FreightReference) SetArtifacts(v []ArtifactReference)`

SetArtifacts sets Artifacts field to given value.

### HasArtifacts

`func (o *FreightReference) HasArtifacts() bool`

HasArtifacts returns a boolean if a field has been set.

### GetCharts

`func (o *FreightReference) GetCharts() []Chart`

GetCharts returns the Charts field if non-nil, zero value otherwise.

### GetChartsOk

`func (o *FreightReference) GetChartsOk() (*[]Chart, bool)`

GetChartsOk returns a tuple with the Charts field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCharts

`func (o *FreightReference) SetCharts(v []Chart)`

SetCharts sets Charts field to given value.

### HasCharts

`func (o *FreightReference) HasCharts() bool`

HasCharts returns a boolean if a field has been set.

### GetCommits

`func (o *FreightReference) GetCommits() []GitCommit`

GetCommits returns the Commits field if non-nil, zero value otherwise.

### GetCommitsOk

`func (o *FreightReference) GetCommitsOk() (*[]GitCommit, bool)`

GetCommitsOk returns a tuple with the Commits field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCommits

`func (o *FreightReference) SetCommits(v []GitCommit)`

SetCommits sets Commits field to given value.

### HasCommits

`func (o *FreightReference) HasCommits() bool`

HasCommits returns a boolean if a field has been set.

### GetImages

`func (o *FreightReference) GetImages() []Image`

GetImages returns the Images field if non-nil, zero value otherwise.

### GetImagesOk

`func (o *FreightReference) GetImagesOk() (*[]Image, bool)`

GetImagesOk returns a tuple with the Images field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetImages

`func (o *FreightReference) SetImages(v []Image)`

SetImages sets Images field to given value.

### HasImages

`func (o *FreightReference) HasImages() bool`

HasImages returns a boolean if a field has been set.

### GetName

`func (o *FreightReference) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *FreightReference) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *FreightReference) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *FreightReference) HasName() bool`

HasName returns a boolean if a field has been set.

### GetOrigin

`func (o *FreightReference) GetOrigin() FreightOrigin`

GetOrigin returns the Origin field if non-nil, zero value otherwise.

### GetOriginOk

`func (o *FreightReference) GetOriginOk() (*FreightOrigin, bool)`

GetOriginOk returns a tuple with the Origin field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOrigin

`func (o *FreightReference) SetOrigin(v FreightOrigin)`

SetOrigin sets Origin field to given value.

### HasOrigin

`func (o *FreightReference) HasOrigin() bool`

HasOrigin returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


