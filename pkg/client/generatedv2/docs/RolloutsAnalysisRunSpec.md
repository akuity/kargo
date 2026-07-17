# RolloutsAnalysisRunSpec

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Args** | Pointer to [**[]RolloutsArgument**](RolloutsArgument.md) |  | [optional] 
**DryRun** | Pointer to [**[]RolloutsDryRun**](RolloutsDryRun.md) |  | [optional] 
**MeasurementRetention** | Pointer to [**[]RolloutsMeasurementRetention**](RolloutsMeasurementRetention.md) |  | [optional] 
**Metrics** | Pointer to [**[]RolloutsMetric**](RolloutsMetric.md) |  | [optional] 
**Terminate** | Pointer to **bool** |  | [optional] 
**TtlStrategy** | Pointer to [**RolloutsTTLStrategy**](RolloutsTTLStrategy.md) |  | [optional] 

## Methods

### NewRolloutsAnalysisRunSpec

`func NewRolloutsAnalysisRunSpec() *RolloutsAnalysisRunSpec`

NewRolloutsAnalysisRunSpec instantiates a new RolloutsAnalysisRunSpec object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRolloutsAnalysisRunSpecWithDefaults

`func NewRolloutsAnalysisRunSpecWithDefaults() *RolloutsAnalysisRunSpec`

NewRolloutsAnalysisRunSpecWithDefaults instantiates a new RolloutsAnalysisRunSpec object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetArgs

`func (o *RolloutsAnalysisRunSpec) GetArgs() []RolloutsArgument`

GetArgs returns the Args field if non-nil, zero value otherwise.

### GetArgsOk

`func (o *RolloutsAnalysisRunSpec) GetArgsOk() (*[]RolloutsArgument, bool)`

GetArgsOk returns a tuple with the Args field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetArgs

`func (o *RolloutsAnalysisRunSpec) SetArgs(v []RolloutsArgument)`

SetArgs sets Args field to given value.

### HasArgs

`func (o *RolloutsAnalysisRunSpec) HasArgs() bool`

HasArgs returns a boolean if a field has been set.

### GetDryRun

`func (o *RolloutsAnalysisRunSpec) GetDryRun() []RolloutsDryRun`

GetDryRun returns the DryRun field if non-nil, zero value otherwise.

### GetDryRunOk

`func (o *RolloutsAnalysisRunSpec) GetDryRunOk() (*[]RolloutsDryRun, bool)`

GetDryRunOk returns a tuple with the DryRun field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDryRun

`func (o *RolloutsAnalysisRunSpec) SetDryRun(v []RolloutsDryRun)`

SetDryRun sets DryRun field to given value.

### HasDryRun

`func (o *RolloutsAnalysisRunSpec) HasDryRun() bool`

HasDryRun returns a boolean if a field has been set.

### GetMeasurementRetention

`func (o *RolloutsAnalysisRunSpec) GetMeasurementRetention() []RolloutsMeasurementRetention`

GetMeasurementRetention returns the MeasurementRetention field if non-nil, zero value otherwise.

### GetMeasurementRetentionOk

`func (o *RolloutsAnalysisRunSpec) GetMeasurementRetentionOk() (*[]RolloutsMeasurementRetention, bool)`

GetMeasurementRetentionOk returns a tuple with the MeasurementRetention field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMeasurementRetention

`func (o *RolloutsAnalysisRunSpec) SetMeasurementRetention(v []RolloutsMeasurementRetention)`

SetMeasurementRetention sets MeasurementRetention field to given value.

### HasMeasurementRetention

`func (o *RolloutsAnalysisRunSpec) HasMeasurementRetention() bool`

HasMeasurementRetention returns a boolean if a field has been set.

### GetMetrics

`func (o *RolloutsAnalysisRunSpec) GetMetrics() []RolloutsMetric`

GetMetrics returns the Metrics field if non-nil, zero value otherwise.

### GetMetricsOk

`func (o *RolloutsAnalysisRunSpec) GetMetricsOk() (*[]RolloutsMetric, bool)`

GetMetricsOk returns a tuple with the Metrics field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMetrics

`func (o *RolloutsAnalysisRunSpec) SetMetrics(v []RolloutsMetric)`

SetMetrics sets Metrics field to given value.

### HasMetrics

`func (o *RolloutsAnalysisRunSpec) HasMetrics() bool`

HasMetrics returns a boolean if a field has been set.

### GetTerminate

`func (o *RolloutsAnalysisRunSpec) GetTerminate() bool`

GetTerminate returns the Terminate field if non-nil, zero value otherwise.

### GetTerminateOk

`func (o *RolloutsAnalysisRunSpec) GetTerminateOk() (*bool, bool)`

GetTerminateOk returns a tuple with the Terminate field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTerminate

`func (o *RolloutsAnalysisRunSpec) SetTerminate(v bool)`

SetTerminate sets Terminate field to given value.

### HasTerminate

`func (o *RolloutsAnalysisRunSpec) HasTerminate() bool`

HasTerminate returns a boolean if a field has been set.

### GetTtlStrategy

`func (o *RolloutsAnalysisRunSpec) GetTtlStrategy() RolloutsTTLStrategy`

GetTtlStrategy returns the TtlStrategy field if non-nil, zero value otherwise.

### GetTtlStrategyOk

`func (o *RolloutsAnalysisRunSpec) GetTtlStrategyOk() (*RolloutsTTLStrategy, bool)`

GetTtlStrategyOk returns a tuple with the TtlStrategy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTtlStrategy

`func (o *RolloutsAnalysisRunSpec) SetTtlStrategy(v RolloutsTTLStrategy)`

SetTtlStrategy sets TtlStrategy field to given value.

### HasTtlStrategy

`func (o *RolloutsAnalysisRunSpec) HasTtlStrategy() bool`

HasTtlStrategy returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


