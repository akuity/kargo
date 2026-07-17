# HealthStats

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Healthy** | Pointer to **int32** | Healthy contains the number of resources that are explicitly healthy. | [optional] 

## Methods

### NewHealthStats

`func NewHealthStats() *HealthStats`

NewHealthStats instantiates a new HealthStats object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewHealthStatsWithDefaults

`func NewHealthStatsWithDefaults() *HealthStats`

NewHealthStatsWithDefaults instantiates a new HealthStats object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetHealthy

`func (o *HealthStats) GetHealthy() int32`

GetHealthy returns the Healthy field if non-nil, zero value otherwise.

### GetHealthyOk

`func (o *HealthStats) GetHealthyOk() (*int32, bool)`

GetHealthyOk returns a tuple with the Healthy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHealthy

`func (o *HealthStats) SetHealthy(v int32)`

SetHealthy sets Healthy field to given value.

### HasHealthy

`func (o *HealthStats) HasHealthy() bool`

HasHealthy returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


