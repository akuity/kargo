# ProjectStatus

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Conditions** | Pointer to [**[]V1Condition**](V1Condition.md) | Conditions contains the last observations of the Project&#39;s current state. +patchMergeKey&#x3D;type +patchStrategy&#x3D;merge +listType&#x3D;map +listMapKey&#x3D;type | [optional] 
**Stats** | Pointer to [**ProjectStats**](ProjectStats.md) | Stats contains a summary of the collective state of a Project&#39;s constituent resources. | [optional] 

## Methods

### NewProjectStatus

`func NewProjectStatus() *ProjectStatus`

NewProjectStatus instantiates a new ProjectStatus object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewProjectStatusWithDefaults

`func NewProjectStatusWithDefaults() *ProjectStatus`

NewProjectStatusWithDefaults instantiates a new ProjectStatus object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetConditions

`func (o *ProjectStatus) GetConditions() []V1Condition`

GetConditions returns the Conditions field if non-nil, zero value otherwise.

### GetConditionsOk

`func (o *ProjectStatus) GetConditionsOk() (*[]V1Condition, bool)`

GetConditionsOk returns a tuple with the Conditions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConditions

`func (o *ProjectStatus) SetConditions(v []V1Condition)`

SetConditions sets Conditions field to given value.

### HasConditions

`func (o *ProjectStatus) HasConditions() bool`

HasConditions returns a boolean if a field has been set.

### GetStats

`func (o *ProjectStatus) GetStats() ProjectStats`

GetStats returns the Stats field if non-nil, zero value otherwise.

### GetStatsOk

`func (o *ProjectStatus) GetStatsOk() (*ProjectStats, bool)`

GetStatsOk returns a tuple with the Stats field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStats

`func (o *ProjectStatus) SetStats(v ProjectStats)`

SetStats sets Stats field to given value.

### HasStats

`func (o *ProjectStatus) HasStats() bool`

HasStats returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


