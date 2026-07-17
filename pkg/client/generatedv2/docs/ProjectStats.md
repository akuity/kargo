# ProjectStats

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Stages** | Pointer to [**StageStats**](StageStats.md) | Stages contains a summary of the collective state of the Project&#39;s Stages. | [optional] 
**Warehouses** | Pointer to [**WarehouseStats**](WarehouseStats.md) | Warehouses contains a summary of the collective state of the Project&#39;s Warehouses. | [optional] 

## Methods

### NewProjectStats

`func NewProjectStats() *ProjectStats`

NewProjectStats instantiates a new ProjectStats object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewProjectStatsWithDefaults

`func NewProjectStatsWithDefaults() *ProjectStats`

NewProjectStatsWithDefaults instantiates a new ProjectStats object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetStages

`func (o *ProjectStats) GetStages() StageStats`

GetStages returns the Stages field if non-nil, zero value otherwise.

### GetStagesOk

`func (o *ProjectStats) GetStagesOk() (*StageStats, bool)`

GetStagesOk returns a tuple with the Stages field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStages

`func (o *ProjectStats) SetStages(v StageStats)`

SetStages sets Stages field to given value.

### HasStages

`func (o *ProjectStats) HasStages() bool`

HasStages returns a boolean if a field has been set.

### GetWarehouses

`func (o *ProjectStats) GetWarehouses() WarehouseStats`

GetWarehouses returns the Warehouses field if non-nil, zero value otherwise.

### GetWarehousesOk

`func (o *ProjectStats) GetWarehousesOk() (*WarehouseStats, bool)`

GetWarehousesOk returns a tuple with the Warehouses field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWarehouses

`func (o *ProjectStats) SetWarehouses(v WarehouseStats)`

SetWarehouses sets Warehouses field to given value.

### HasWarehouses

`func (o *ProjectStats) HasWarehouses() bool`

HasWarehouses returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


