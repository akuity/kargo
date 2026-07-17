# Chart

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | Pointer to **string** | Name specifies the name of the chart. | [optional] 
**RepoURL** | Pointer to **string** | RepoURL specifies the URL of a Helm chart repository. Classic chart repositories (using HTTP/S) can contain differently named charts. When this field points to such a repository, the Name field will specify the name of the chart within the repository. In the case of a repository within an OCI registry, the URL implicitly points to a specific chart and the Name field will be empty. | [optional] 
**Version** | Pointer to **string** | Version specifies a particular version of the chart. | [optional] 

## Methods

### NewChart

`func NewChart() *Chart`

NewChart instantiates a new Chart object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewChartWithDefaults

`func NewChartWithDefaults() *Chart`

NewChartWithDefaults instantiates a new Chart object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetName

`func (o *Chart) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *Chart) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *Chart) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *Chart) HasName() bool`

HasName returns a boolean if a field has been set.

### GetRepoURL

`func (o *Chart) GetRepoURL() string`

GetRepoURL returns the RepoURL field if non-nil, zero value otherwise.

### GetRepoURLOk

`func (o *Chart) GetRepoURLOk() (*string, bool)`

GetRepoURLOk returns a tuple with the RepoURL field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRepoURL

`func (o *Chart) SetRepoURL(v string)`

SetRepoURL sets RepoURL field to given value.

### HasRepoURL

`func (o *Chart) HasRepoURL() bool`

HasRepoURL returns a boolean if a field has been set.

### GetVersion

`func (o *Chart) GetVersion() string`

GetVersion returns the Version field if non-nil, zero value otherwise.

### GetVersionOk

`func (o *Chart) GetVersionOk() (*string, bool)`

GetVersionOk returns a tuple with the Version field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVersion

`func (o *Chart) SetVersion(v string)`

SetVersion sets Version field to given value.

### HasVersion

`func (o *Chart) HasVersion() bool`

HasVersion returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


