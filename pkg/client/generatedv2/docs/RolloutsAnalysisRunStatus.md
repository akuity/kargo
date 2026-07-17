# RolloutsAnalysisRunStatus

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**CompletedAt** | Pointer to **string** |  | [optional] 
**DryRunSummary** | Pointer to [**RolloutsRunSummary**](RolloutsRunSummary.md) |  | [optional] 
**Message** | Pointer to **string** |  | [optional] 
**MetricResults** | Pointer to [**[]RolloutsMetricResult**](RolloutsMetricResult.md) |  | [optional] 
**Phase** | Pointer to **string** |  | [optional] 
**RunSummary** | Pointer to [**RolloutsRunSummary**](RolloutsRunSummary.md) |  | [optional] 
**StartedAt** | Pointer to **string** |  | [optional] 

## Methods

### NewRolloutsAnalysisRunStatus

`func NewRolloutsAnalysisRunStatus() *RolloutsAnalysisRunStatus`

NewRolloutsAnalysisRunStatus instantiates a new RolloutsAnalysisRunStatus object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRolloutsAnalysisRunStatusWithDefaults

`func NewRolloutsAnalysisRunStatusWithDefaults() *RolloutsAnalysisRunStatus`

NewRolloutsAnalysisRunStatusWithDefaults instantiates a new RolloutsAnalysisRunStatus object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetCompletedAt

`func (o *RolloutsAnalysisRunStatus) GetCompletedAt() string`

GetCompletedAt returns the CompletedAt field if non-nil, zero value otherwise.

### GetCompletedAtOk

`func (o *RolloutsAnalysisRunStatus) GetCompletedAtOk() (*string, bool)`

GetCompletedAtOk returns a tuple with the CompletedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCompletedAt

`func (o *RolloutsAnalysisRunStatus) SetCompletedAt(v string)`

SetCompletedAt sets CompletedAt field to given value.

### HasCompletedAt

`func (o *RolloutsAnalysisRunStatus) HasCompletedAt() bool`

HasCompletedAt returns a boolean if a field has been set.

### GetDryRunSummary

`func (o *RolloutsAnalysisRunStatus) GetDryRunSummary() RolloutsRunSummary`

GetDryRunSummary returns the DryRunSummary field if non-nil, zero value otherwise.

### GetDryRunSummaryOk

`func (o *RolloutsAnalysisRunStatus) GetDryRunSummaryOk() (*RolloutsRunSummary, bool)`

GetDryRunSummaryOk returns a tuple with the DryRunSummary field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDryRunSummary

`func (o *RolloutsAnalysisRunStatus) SetDryRunSummary(v RolloutsRunSummary)`

SetDryRunSummary sets DryRunSummary field to given value.

### HasDryRunSummary

`func (o *RolloutsAnalysisRunStatus) HasDryRunSummary() bool`

HasDryRunSummary returns a boolean if a field has been set.

### GetMessage

`func (o *RolloutsAnalysisRunStatus) GetMessage() string`

GetMessage returns the Message field if non-nil, zero value otherwise.

### GetMessageOk

`func (o *RolloutsAnalysisRunStatus) GetMessageOk() (*string, bool)`

GetMessageOk returns a tuple with the Message field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMessage

`func (o *RolloutsAnalysisRunStatus) SetMessage(v string)`

SetMessage sets Message field to given value.

### HasMessage

`func (o *RolloutsAnalysisRunStatus) HasMessage() bool`

HasMessage returns a boolean if a field has been set.

### GetMetricResults

`func (o *RolloutsAnalysisRunStatus) GetMetricResults() []RolloutsMetricResult`

GetMetricResults returns the MetricResults field if non-nil, zero value otherwise.

### GetMetricResultsOk

`func (o *RolloutsAnalysisRunStatus) GetMetricResultsOk() (*[]RolloutsMetricResult, bool)`

GetMetricResultsOk returns a tuple with the MetricResults field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMetricResults

`func (o *RolloutsAnalysisRunStatus) SetMetricResults(v []RolloutsMetricResult)`

SetMetricResults sets MetricResults field to given value.

### HasMetricResults

`func (o *RolloutsAnalysisRunStatus) HasMetricResults() bool`

HasMetricResults returns a boolean if a field has been set.

### GetPhase

`func (o *RolloutsAnalysisRunStatus) GetPhase() string`

GetPhase returns the Phase field if non-nil, zero value otherwise.

### GetPhaseOk

`func (o *RolloutsAnalysisRunStatus) GetPhaseOk() (*string, bool)`

GetPhaseOk returns a tuple with the Phase field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPhase

`func (o *RolloutsAnalysisRunStatus) SetPhase(v string)`

SetPhase sets Phase field to given value.

### HasPhase

`func (o *RolloutsAnalysisRunStatus) HasPhase() bool`

HasPhase returns a boolean if a field has been set.

### GetRunSummary

`func (o *RolloutsAnalysisRunStatus) GetRunSummary() RolloutsRunSummary`

GetRunSummary returns the RunSummary field if non-nil, zero value otherwise.

### GetRunSummaryOk

`func (o *RolloutsAnalysisRunStatus) GetRunSummaryOk() (*RolloutsRunSummary, bool)`

GetRunSummaryOk returns a tuple with the RunSummary field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRunSummary

`func (o *RolloutsAnalysisRunStatus) SetRunSummary(v RolloutsRunSummary)`

SetRunSummary sets RunSummary field to given value.

### HasRunSummary

`func (o *RolloutsAnalysisRunStatus) HasRunSummary() bool`

HasRunSummary returns a boolean if a field has been set.

### GetStartedAt

`func (o *RolloutsAnalysisRunStatus) GetStartedAt() string`

GetStartedAt returns the StartedAt field if non-nil, zero value otherwise.

### GetStartedAtOk

`func (o *RolloutsAnalysisRunStatus) GetStartedAtOk() (*string, bool)`

GetStartedAtOk returns a tuple with the StartedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStartedAt

`func (o *RolloutsAnalysisRunStatus) SetStartedAt(v string)`

SetStartedAt sets StartedAt field to given value.

### HasStartedAt

`func (o *RolloutsAnalysisRunStatus) HasStartedAt() bool`

HasStartedAt returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


