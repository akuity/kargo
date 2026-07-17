# RolloutsCloudWatchMetricDataQuery

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Expression** | Pointer to **string** |  | [optional] 
**Id** | Pointer to **string** |  | [optional] 
**Label** | Pointer to **string** |  | [optional] 
**MetricStat** | Pointer to [**RolloutsCloudWatchMetricStat**](RolloutsCloudWatchMetricStat.md) |  | [optional] 
**Period** | Pointer to **interface{}** | Serializes as a bare integer or string | [optional] 
**ReturnData** | Pointer to **bool** |  | [optional] 

## Methods

### NewRolloutsCloudWatchMetricDataQuery

`func NewRolloutsCloudWatchMetricDataQuery() *RolloutsCloudWatchMetricDataQuery`

NewRolloutsCloudWatchMetricDataQuery instantiates a new RolloutsCloudWatchMetricDataQuery object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRolloutsCloudWatchMetricDataQueryWithDefaults

`func NewRolloutsCloudWatchMetricDataQueryWithDefaults() *RolloutsCloudWatchMetricDataQuery`

NewRolloutsCloudWatchMetricDataQueryWithDefaults instantiates a new RolloutsCloudWatchMetricDataQuery object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetExpression

`func (o *RolloutsCloudWatchMetricDataQuery) GetExpression() string`

GetExpression returns the Expression field if non-nil, zero value otherwise.

### GetExpressionOk

`func (o *RolloutsCloudWatchMetricDataQuery) GetExpressionOk() (*string, bool)`

GetExpressionOk returns a tuple with the Expression field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExpression

`func (o *RolloutsCloudWatchMetricDataQuery) SetExpression(v string)`

SetExpression sets Expression field to given value.

### HasExpression

`func (o *RolloutsCloudWatchMetricDataQuery) HasExpression() bool`

HasExpression returns a boolean if a field has been set.

### GetId

`func (o *RolloutsCloudWatchMetricDataQuery) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *RolloutsCloudWatchMetricDataQuery) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *RolloutsCloudWatchMetricDataQuery) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *RolloutsCloudWatchMetricDataQuery) HasId() bool`

HasId returns a boolean if a field has been set.

### GetLabel

`func (o *RolloutsCloudWatchMetricDataQuery) GetLabel() string`

GetLabel returns the Label field if non-nil, zero value otherwise.

### GetLabelOk

`func (o *RolloutsCloudWatchMetricDataQuery) GetLabelOk() (*string, bool)`

GetLabelOk returns a tuple with the Label field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLabel

`func (o *RolloutsCloudWatchMetricDataQuery) SetLabel(v string)`

SetLabel sets Label field to given value.

### HasLabel

`func (o *RolloutsCloudWatchMetricDataQuery) HasLabel() bool`

HasLabel returns a boolean if a field has been set.

### GetMetricStat

`func (o *RolloutsCloudWatchMetricDataQuery) GetMetricStat() RolloutsCloudWatchMetricStat`

GetMetricStat returns the MetricStat field if non-nil, zero value otherwise.

### GetMetricStatOk

`func (o *RolloutsCloudWatchMetricDataQuery) GetMetricStatOk() (*RolloutsCloudWatchMetricStat, bool)`

GetMetricStatOk returns a tuple with the MetricStat field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMetricStat

`func (o *RolloutsCloudWatchMetricDataQuery) SetMetricStat(v RolloutsCloudWatchMetricStat)`

SetMetricStat sets MetricStat field to given value.

### HasMetricStat

`func (o *RolloutsCloudWatchMetricDataQuery) HasMetricStat() bool`

HasMetricStat returns a boolean if a field has been set.

### GetPeriod

`func (o *RolloutsCloudWatchMetricDataQuery) GetPeriod() interface{}`

GetPeriod returns the Period field if non-nil, zero value otherwise.

### GetPeriodOk

`func (o *RolloutsCloudWatchMetricDataQuery) GetPeriodOk() (*interface{}, bool)`

GetPeriodOk returns a tuple with the Period field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPeriod

`func (o *RolloutsCloudWatchMetricDataQuery) SetPeriod(v interface{})`

SetPeriod sets Period field to given value.

### HasPeriod

`func (o *RolloutsCloudWatchMetricDataQuery) HasPeriod() bool`

HasPeriod returns a boolean if a field has been set.

### GetReturnData

`func (o *RolloutsCloudWatchMetricDataQuery) GetReturnData() bool`

GetReturnData returns the ReturnData field if non-nil, zero value otherwise.

### GetReturnDataOk

`func (o *RolloutsCloudWatchMetricDataQuery) GetReturnDataOk() (*bool, bool)`

GetReturnDataOk returns a tuple with the ReturnData field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReturnData

`func (o *RolloutsCloudWatchMetricDataQuery) SetReturnData(v bool)`

SetReturnData sets ReturnData field to given value.

### HasReturnData

`func (o *RolloutsCloudWatchMetricDataQuery) HasReturnData() bool`

HasReturnData returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


