# ChartDiscoveryResult

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | Pointer to **string** | Name is the name of the Helm chart, as specified in the ChartSubscription. | [optional] 
**RepoURL** | Pointer to **string** | RepoURL is the repository URL of the Helm chart, as specified in the ChartSubscription.  +kubebuilder:validation:MinLength&#x3D;1 | [optional] 
**SemverConstraint** | Pointer to **string** | SemverConstraint is the constraint for which versions were discovered. This field is optional, and only populated if the ChartSubscription specifies a SemverConstraint. | [optional] 
**Versions** | Pointer to **[]string** | Versions is a list of versions discovered by the Warehouse for the ChartSubscription. An empty list indicates that the discovery operation was successful, but no versions matching the ChartSubscription criteria were found.  +optional | [optional] 

## Methods

### NewChartDiscoveryResult

`func NewChartDiscoveryResult() *ChartDiscoveryResult`

NewChartDiscoveryResult instantiates a new ChartDiscoveryResult object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewChartDiscoveryResultWithDefaults

`func NewChartDiscoveryResultWithDefaults() *ChartDiscoveryResult`

NewChartDiscoveryResultWithDefaults instantiates a new ChartDiscoveryResult object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetName

`func (o *ChartDiscoveryResult) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *ChartDiscoveryResult) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *ChartDiscoveryResult) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *ChartDiscoveryResult) HasName() bool`

HasName returns a boolean if a field has been set.

### GetRepoURL

`func (o *ChartDiscoveryResult) GetRepoURL() string`

GetRepoURL returns the RepoURL field if non-nil, zero value otherwise.

### GetRepoURLOk

`func (o *ChartDiscoveryResult) GetRepoURLOk() (*string, bool)`

GetRepoURLOk returns a tuple with the RepoURL field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRepoURL

`func (o *ChartDiscoveryResult) SetRepoURL(v string)`

SetRepoURL sets RepoURL field to given value.

### HasRepoURL

`func (o *ChartDiscoveryResult) HasRepoURL() bool`

HasRepoURL returns a boolean if a field has been set.

### GetSemverConstraint

`func (o *ChartDiscoveryResult) GetSemverConstraint() string`

GetSemverConstraint returns the SemverConstraint field if non-nil, zero value otherwise.

### GetSemverConstraintOk

`func (o *ChartDiscoveryResult) GetSemverConstraintOk() (*string, bool)`

GetSemverConstraintOk returns a tuple with the SemverConstraint field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSemverConstraint

`func (o *ChartDiscoveryResult) SetSemverConstraint(v string)`

SetSemverConstraint sets SemverConstraint field to given value.

### HasSemverConstraint

`func (o *ChartDiscoveryResult) HasSemverConstraint() bool`

HasSemverConstraint returns a boolean if a field has been set.

### GetVersions

`func (o *ChartDiscoveryResult) GetVersions() []string`

GetVersions returns the Versions field if non-nil, zero value otherwise.

### GetVersionsOk

`func (o *ChartDiscoveryResult) GetVersionsOk() (*[]string, bool)`

GetVersionsOk returns a tuple with the Versions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVersions

`func (o *ChartDiscoveryResult) SetVersions(v []string)`

SetVersions sets Versions field to given value.

### HasVersions

`func (o *ChartDiscoveryResult) HasVersions() bool`

HasVersions returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


