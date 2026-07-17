# DiscoveredArtifacts

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Charts** | Pointer to [**[]ChartDiscoveryResult**](ChartDiscoveryResult.md) | Charts holds the charts discovered by the Warehouse for the chart subscriptions.  +optional | [optional] 
**DiscoveredAt** | Pointer to **string** | DiscoveredAt is the time at which the Warehouse discovered the artifacts.  +optional | [optional] 
**Git** | Pointer to [**[]GitDiscoveryResult**](GitDiscoveryResult.md) | Git holds the commits discovered by the Warehouse for the Git subscriptions.  +optional | [optional] 
**Images** | Pointer to [**[]ImageDiscoveryResult**](ImageDiscoveryResult.md) | Images holds the image references discovered by the Warehouse for the image subscriptions.  +optional | [optional] 
**Results** | Pointer to [**[]DiscoveryResult**](DiscoveryResult.md) | Results holds the artifact references discovered by the Warehouse.  +optional | [optional] 

## Methods

### NewDiscoveredArtifacts

`func NewDiscoveredArtifacts() *DiscoveredArtifacts`

NewDiscoveredArtifacts instantiates a new DiscoveredArtifacts object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewDiscoveredArtifactsWithDefaults

`func NewDiscoveredArtifactsWithDefaults() *DiscoveredArtifacts`

NewDiscoveredArtifactsWithDefaults instantiates a new DiscoveredArtifacts object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetCharts

`func (o *DiscoveredArtifacts) GetCharts() []ChartDiscoveryResult`

GetCharts returns the Charts field if non-nil, zero value otherwise.

### GetChartsOk

`func (o *DiscoveredArtifacts) GetChartsOk() (*[]ChartDiscoveryResult, bool)`

GetChartsOk returns a tuple with the Charts field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCharts

`func (o *DiscoveredArtifacts) SetCharts(v []ChartDiscoveryResult)`

SetCharts sets Charts field to given value.

### HasCharts

`func (o *DiscoveredArtifacts) HasCharts() bool`

HasCharts returns a boolean if a field has been set.

### GetDiscoveredAt

`func (o *DiscoveredArtifacts) GetDiscoveredAt() string`

GetDiscoveredAt returns the DiscoveredAt field if non-nil, zero value otherwise.

### GetDiscoveredAtOk

`func (o *DiscoveredArtifacts) GetDiscoveredAtOk() (*string, bool)`

GetDiscoveredAtOk returns a tuple with the DiscoveredAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDiscoveredAt

`func (o *DiscoveredArtifacts) SetDiscoveredAt(v string)`

SetDiscoveredAt sets DiscoveredAt field to given value.

### HasDiscoveredAt

`func (o *DiscoveredArtifacts) HasDiscoveredAt() bool`

HasDiscoveredAt returns a boolean if a field has been set.

### GetGit

`func (o *DiscoveredArtifacts) GetGit() []GitDiscoveryResult`

GetGit returns the Git field if non-nil, zero value otherwise.

### GetGitOk

`func (o *DiscoveredArtifacts) GetGitOk() (*[]GitDiscoveryResult, bool)`

GetGitOk returns a tuple with the Git field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetGit

`func (o *DiscoveredArtifacts) SetGit(v []GitDiscoveryResult)`

SetGit sets Git field to given value.

### HasGit

`func (o *DiscoveredArtifacts) HasGit() bool`

HasGit returns a boolean if a field has been set.

### GetImages

`func (o *DiscoveredArtifacts) GetImages() []ImageDiscoveryResult`

GetImages returns the Images field if non-nil, zero value otherwise.

### GetImagesOk

`func (o *DiscoveredArtifacts) GetImagesOk() (*[]ImageDiscoveryResult, bool)`

GetImagesOk returns a tuple with the Images field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetImages

`func (o *DiscoveredArtifacts) SetImages(v []ImageDiscoveryResult)`

SetImages sets Images field to given value.

### HasImages

`func (o *DiscoveredArtifacts) HasImages() bool`

HasImages returns a boolean if a field has been set.

### GetResults

`func (o *DiscoveredArtifacts) GetResults() []DiscoveryResult`

GetResults returns the Results field if non-nil, zero value otherwise.

### GetResultsOk

`func (o *DiscoveredArtifacts) GetResultsOk() (*[]DiscoveryResult, bool)`

GetResultsOk returns a tuple with the Results field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResults

`func (o *DiscoveredArtifacts) SetResults(v []DiscoveryResult)`

SetResults sets Results field to given value.

### HasResults

`func (o *DiscoveredArtifacts) HasResults() bool`

HasResults returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


