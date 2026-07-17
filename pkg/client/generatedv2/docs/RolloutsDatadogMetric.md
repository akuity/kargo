# RolloutsDatadogMetric

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Aggregator** | Pointer to **string** |  | [optional] 
**ApiVersion** | Pointer to **string** |  | [optional] 
**Formula** | Pointer to **string** |  | [optional] 
**Interval** | Pointer to **string** |  | [optional] 
**Queries** | Pointer to **map[string]string** |  | [optional] 
**Query** | Pointer to **string** |  | [optional] 
**SecretRef** | Pointer to [**RolloutsSecretRef**](RolloutsSecretRef.md) |  | [optional] 

## Methods

### NewRolloutsDatadogMetric

`func NewRolloutsDatadogMetric() *RolloutsDatadogMetric`

NewRolloutsDatadogMetric instantiates a new RolloutsDatadogMetric object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRolloutsDatadogMetricWithDefaults

`func NewRolloutsDatadogMetricWithDefaults() *RolloutsDatadogMetric`

NewRolloutsDatadogMetricWithDefaults instantiates a new RolloutsDatadogMetric object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAggregator

`func (o *RolloutsDatadogMetric) GetAggregator() string`

GetAggregator returns the Aggregator field if non-nil, zero value otherwise.

### GetAggregatorOk

`func (o *RolloutsDatadogMetric) GetAggregatorOk() (*string, bool)`

GetAggregatorOk returns a tuple with the Aggregator field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAggregator

`func (o *RolloutsDatadogMetric) SetAggregator(v string)`

SetAggregator sets Aggregator field to given value.

### HasAggregator

`func (o *RolloutsDatadogMetric) HasAggregator() bool`

HasAggregator returns a boolean if a field has been set.

### GetApiVersion

`func (o *RolloutsDatadogMetric) GetApiVersion() string`

GetApiVersion returns the ApiVersion field if non-nil, zero value otherwise.

### GetApiVersionOk

`func (o *RolloutsDatadogMetric) GetApiVersionOk() (*string, bool)`

GetApiVersionOk returns a tuple with the ApiVersion field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetApiVersion

`func (o *RolloutsDatadogMetric) SetApiVersion(v string)`

SetApiVersion sets ApiVersion field to given value.

### HasApiVersion

`func (o *RolloutsDatadogMetric) HasApiVersion() bool`

HasApiVersion returns a boolean if a field has been set.

### GetFormula

`func (o *RolloutsDatadogMetric) GetFormula() string`

GetFormula returns the Formula field if non-nil, zero value otherwise.

### GetFormulaOk

`func (o *RolloutsDatadogMetric) GetFormulaOk() (*string, bool)`

GetFormulaOk returns a tuple with the Formula field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFormula

`func (o *RolloutsDatadogMetric) SetFormula(v string)`

SetFormula sets Formula field to given value.

### HasFormula

`func (o *RolloutsDatadogMetric) HasFormula() bool`

HasFormula returns a boolean if a field has been set.

### GetInterval

`func (o *RolloutsDatadogMetric) GetInterval() string`

GetInterval returns the Interval field if non-nil, zero value otherwise.

### GetIntervalOk

`func (o *RolloutsDatadogMetric) GetIntervalOk() (*string, bool)`

GetIntervalOk returns a tuple with the Interval field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInterval

`func (o *RolloutsDatadogMetric) SetInterval(v string)`

SetInterval sets Interval field to given value.

### HasInterval

`func (o *RolloutsDatadogMetric) HasInterval() bool`

HasInterval returns a boolean if a field has been set.

### GetQueries

`func (o *RolloutsDatadogMetric) GetQueries() map[string]string`

GetQueries returns the Queries field if non-nil, zero value otherwise.

### GetQueriesOk

`func (o *RolloutsDatadogMetric) GetQueriesOk() (*map[string]string, bool)`

GetQueriesOk returns a tuple with the Queries field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetQueries

`func (o *RolloutsDatadogMetric) SetQueries(v map[string]string)`

SetQueries sets Queries field to given value.

### HasQueries

`func (o *RolloutsDatadogMetric) HasQueries() bool`

HasQueries returns a boolean if a field has been set.

### GetQuery

`func (o *RolloutsDatadogMetric) GetQuery() string`

GetQuery returns the Query field if non-nil, zero value otherwise.

### GetQueryOk

`func (o *RolloutsDatadogMetric) GetQueryOk() (*string, bool)`

GetQueryOk returns a tuple with the Query field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetQuery

`func (o *RolloutsDatadogMetric) SetQuery(v string)`

SetQuery sets Query field to given value.

### HasQuery

`func (o *RolloutsDatadogMetric) HasQuery() bool`

HasQuery returns a boolean if a field has been set.

### GetSecretRef

`func (o *RolloutsDatadogMetric) GetSecretRef() RolloutsSecretRef`

GetSecretRef returns the SecretRef field if non-nil, zero value otherwise.

### GetSecretRefOk

`func (o *RolloutsDatadogMetric) GetSecretRefOk() (*RolloutsSecretRef, bool)`

GetSecretRefOk returns a tuple with the SecretRef field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecretRef

`func (o *RolloutsDatadogMetric) SetSecretRef(v RolloutsSecretRef)`

SetSecretRef sets SecretRef field to given value.

### HasSecretRef

`func (o *RolloutsDatadogMetric) HasSecretRef() bool`

HasSecretRef returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


