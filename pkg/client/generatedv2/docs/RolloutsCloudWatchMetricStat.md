# RolloutsCloudWatchMetricStat

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Metric** | Pointer to [**RolloutsCloudWatchMetricStatMetric**](RolloutsCloudWatchMetricStatMetric.md) |  | [optional] 
**Period** | Pointer to **interface{}** | Serializes as a bare integer or string | [optional] 
**Stat** | Pointer to **string** |  | [optional] 
**Unit** | Pointer to **string** |  | [optional] 

## Methods

### NewRolloutsCloudWatchMetricStat

`func NewRolloutsCloudWatchMetricStat() *RolloutsCloudWatchMetricStat`

NewRolloutsCloudWatchMetricStat instantiates a new RolloutsCloudWatchMetricStat object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRolloutsCloudWatchMetricStatWithDefaults

`func NewRolloutsCloudWatchMetricStatWithDefaults() *RolloutsCloudWatchMetricStat`

NewRolloutsCloudWatchMetricStatWithDefaults instantiates a new RolloutsCloudWatchMetricStat object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetMetric

`func (o *RolloutsCloudWatchMetricStat) GetMetric() RolloutsCloudWatchMetricStatMetric`

GetMetric returns the Metric field if non-nil, zero value otherwise.

### GetMetricOk

`func (o *RolloutsCloudWatchMetricStat) GetMetricOk() (*RolloutsCloudWatchMetricStatMetric, bool)`

GetMetricOk returns a tuple with the Metric field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMetric

`func (o *RolloutsCloudWatchMetricStat) SetMetric(v RolloutsCloudWatchMetricStatMetric)`

SetMetric sets Metric field to given value.

### HasMetric

`func (o *RolloutsCloudWatchMetricStat) HasMetric() bool`

HasMetric returns a boolean if a field has been set.

### GetPeriod

`func (o *RolloutsCloudWatchMetricStat) GetPeriod() interface{}`

GetPeriod returns the Period field if non-nil, zero value otherwise.

### GetPeriodOk

`func (o *RolloutsCloudWatchMetricStat) GetPeriodOk() (*interface{}, bool)`

GetPeriodOk returns a tuple with the Period field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPeriod

`func (o *RolloutsCloudWatchMetricStat) SetPeriod(v interface{})`

SetPeriod sets Period field to given value.

### HasPeriod

`func (o *RolloutsCloudWatchMetricStat) HasPeriod() bool`

HasPeriod returns a boolean if a field has been set.

### GetStat

`func (o *RolloutsCloudWatchMetricStat) GetStat() string`

GetStat returns the Stat field if non-nil, zero value otherwise.

### GetStatOk

`func (o *RolloutsCloudWatchMetricStat) GetStatOk() (*string, bool)`

GetStatOk returns a tuple with the Stat field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStat

`func (o *RolloutsCloudWatchMetricStat) SetStat(v string)`

SetStat sets Stat field to given value.

### HasStat

`func (o *RolloutsCloudWatchMetricStat) HasStat() bool`

HasStat returns a boolean if a field has been set.

### GetUnit

`func (o *RolloutsCloudWatchMetricStat) GetUnit() string`

GetUnit returns the Unit field if non-nil, zero value otherwise.

### GetUnitOk

`func (o *RolloutsCloudWatchMetricStat) GetUnitOk() (*string, bool)`

GetUnitOk returns a tuple with the Unit field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUnit

`func (o *RolloutsCloudWatchMetricStat) SetUnit(v string)`

SetUnit sets Unit field to given value.

### HasUnit

`func (o *RolloutsCloudWatchMetricStat) HasUnit() bool`

HasUnit returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


