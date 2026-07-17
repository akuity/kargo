# RolloutsCloudWatchMetric

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Interval** | Pointer to **string** |  | [optional] 
**MetricDataQueries** | Pointer to [**[]RolloutsCloudWatchMetricDataQuery**](RolloutsCloudWatchMetricDataQuery.md) |  | [optional] 

## Methods

### NewRolloutsCloudWatchMetric

`func NewRolloutsCloudWatchMetric() *RolloutsCloudWatchMetric`

NewRolloutsCloudWatchMetric instantiates a new RolloutsCloudWatchMetric object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRolloutsCloudWatchMetricWithDefaults

`func NewRolloutsCloudWatchMetricWithDefaults() *RolloutsCloudWatchMetric`

NewRolloutsCloudWatchMetricWithDefaults instantiates a new RolloutsCloudWatchMetric object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetInterval

`func (o *RolloutsCloudWatchMetric) GetInterval() string`

GetInterval returns the Interval field if non-nil, zero value otherwise.

### GetIntervalOk

`func (o *RolloutsCloudWatchMetric) GetIntervalOk() (*string, bool)`

GetIntervalOk returns a tuple with the Interval field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInterval

`func (o *RolloutsCloudWatchMetric) SetInterval(v string)`

SetInterval sets Interval field to given value.

### HasInterval

`func (o *RolloutsCloudWatchMetric) HasInterval() bool`

HasInterval returns a boolean if a field has been set.

### GetMetricDataQueries

`func (o *RolloutsCloudWatchMetric) GetMetricDataQueries() []RolloutsCloudWatchMetricDataQuery`

GetMetricDataQueries returns the MetricDataQueries field if non-nil, zero value otherwise.

### GetMetricDataQueriesOk

`func (o *RolloutsCloudWatchMetric) GetMetricDataQueriesOk() (*[]RolloutsCloudWatchMetricDataQuery, bool)`

GetMetricDataQueriesOk returns a tuple with the MetricDataQueries field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMetricDataQueries

`func (o *RolloutsCloudWatchMetric) SetMetricDataQueries(v []RolloutsCloudWatchMetricDataQuery)`

SetMetricDataQueries sets MetricDataQueries field to given value.

### HasMetricDataQueries

`func (o *RolloutsCloudWatchMetric) HasMetricDataQueries() bool`

HasMetricDataQueries returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


