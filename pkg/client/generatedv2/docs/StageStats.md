# StageStats

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Count** | Pointer to **int32** | Count contains the total number of Stages in the Project. | [optional] 
**Health** | Pointer to [**HealthStats**](HealthStats.md) | Health contains a summary of the collective health of a Project&#39;s Stages. | [optional] 

## Methods

### NewStageStats

`func NewStageStats() *StageStats`

NewStageStats instantiates a new StageStats object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewStageStatsWithDefaults

`func NewStageStatsWithDefaults() *StageStats`

NewStageStatsWithDefaults instantiates a new StageStats object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetCount

`func (o *StageStats) GetCount() int32`

GetCount returns the Count field if non-nil, zero value otherwise.

### GetCountOk

`func (o *StageStats) GetCountOk() (*int32, bool)`

GetCountOk returns a tuple with the Count field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCount

`func (o *StageStats) SetCount(v int32)`

SetCount sets Count field to given value.

### HasCount

`func (o *StageStats) HasCount() bool`

HasCount returns a boolean if a field has been set.

### GetHealth

`func (o *StageStats) GetHealth() HealthStats`

GetHealth returns the Health field if non-nil, zero value otherwise.

### GetHealthOk

`func (o *StageStats) GetHealthOk() (*HealthStats, bool)`

GetHealthOk returns a tuple with the Health field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHealth

`func (o *StageStats) SetHealth(v HealthStats)`

SetHealth sets Health field to given value.

### HasHealth

`func (o *StageStats) HasHealth() bool`

HasHealth returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


