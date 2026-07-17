# RolloutsMetricResult

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ConsecutiveError** | Pointer to **int32** |  | [optional] 
**ConsecutiveSuccess** | Pointer to **int32** |  | [optional] 
**Count** | Pointer to **int32** |  | [optional] 
**DryRun** | Pointer to **bool** |  | [optional] 
**Error** | Pointer to **int32** |  | [optional] 
**Failed** | Pointer to **int32** |  | [optional] 
**Inconclusive** | Pointer to **int32** |  | [optional] 
**Measurements** | Pointer to [**[]RolloutsMeasurement**](RolloutsMeasurement.md) |  | [optional] 
**Message** | Pointer to **string** |  | [optional] 
**Metadata** | Pointer to **map[string]string** |  | [optional] 
**Name** | Pointer to **string** |  | [optional] 
**Phase** | Pointer to **string** |  | [optional] 
**Successful** | Pointer to **int32** |  | [optional] 

## Methods

### NewRolloutsMetricResult

`func NewRolloutsMetricResult() *RolloutsMetricResult`

NewRolloutsMetricResult instantiates a new RolloutsMetricResult object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRolloutsMetricResultWithDefaults

`func NewRolloutsMetricResultWithDefaults() *RolloutsMetricResult`

NewRolloutsMetricResultWithDefaults instantiates a new RolloutsMetricResult object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetConsecutiveError

`func (o *RolloutsMetricResult) GetConsecutiveError() int32`

GetConsecutiveError returns the ConsecutiveError field if non-nil, zero value otherwise.

### GetConsecutiveErrorOk

`func (o *RolloutsMetricResult) GetConsecutiveErrorOk() (*int32, bool)`

GetConsecutiveErrorOk returns a tuple with the ConsecutiveError field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConsecutiveError

`func (o *RolloutsMetricResult) SetConsecutiveError(v int32)`

SetConsecutiveError sets ConsecutiveError field to given value.

### HasConsecutiveError

`func (o *RolloutsMetricResult) HasConsecutiveError() bool`

HasConsecutiveError returns a boolean if a field has been set.

### GetConsecutiveSuccess

`func (o *RolloutsMetricResult) GetConsecutiveSuccess() int32`

GetConsecutiveSuccess returns the ConsecutiveSuccess field if non-nil, zero value otherwise.

### GetConsecutiveSuccessOk

`func (o *RolloutsMetricResult) GetConsecutiveSuccessOk() (*int32, bool)`

GetConsecutiveSuccessOk returns a tuple with the ConsecutiveSuccess field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConsecutiveSuccess

`func (o *RolloutsMetricResult) SetConsecutiveSuccess(v int32)`

SetConsecutiveSuccess sets ConsecutiveSuccess field to given value.

### HasConsecutiveSuccess

`func (o *RolloutsMetricResult) HasConsecutiveSuccess() bool`

HasConsecutiveSuccess returns a boolean if a field has been set.

### GetCount

`func (o *RolloutsMetricResult) GetCount() int32`

GetCount returns the Count field if non-nil, zero value otherwise.

### GetCountOk

`func (o *RolloutsMetricResult) GetCountOk() (*int32, bool)`

GetCountOk returns a tuple with the Count field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCount

`func (o *RolloutsMetricResult) SetCount(v int32)`

SetCount sets Count field to given value.

### HasCount

`func (o *RolloutsMetricResult) HasCount() bool`

HasCount returns a boolean if a field has been set.

### GetDryRun

`func (o *RolloutsMetricResult) GetDryRun() bool`

GetDryRun returns the DryRun field if non-nil, zero value otherwise.

### GetDryRunOk

`func (o *RolloutsMetricResult) GetDryRunOk() (*bool, bool)`

GetDryRunOk returns a tuple with the DryRun field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDryRun

`func (o *RolloutsMetricResult) SetDryRun(v bool)`

SetDryRun sets DryRun field to given value.

### HasDryRun

`func (o *RolloutsMetricResult) HasDryRun() bool`

HasDryRun returns a boolean if a field has been set.

### GetError

`func (o *RolloutsMetricResult) GetError() int32`

GetError returns the Error field if non-nil, zero value otherwise.

### GetErrorOk

`func (o *RolloutsMetricResult) GetErrorOk() (*int32, bool)`

GetErrorOk returns a tuple with the Error field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetError

`func (o *RolloutsMetricResult) SetError(v int32)`

SetError sets Error field to given value.

### HasError

`func (o *RolloutsMetricResult) HasError() bool`

HasError returns a boolean if a field has been set.

### GetFailed

`func (o *RolloutsMetricResult) GetFailed() int32`

GetFailed returns the Failed field if non-nil, zero value otherwise.

### GetFailedOk

`func (o *RolloutsMetricResult) GetFailedOk() (*int32, bool)`

GetFailedOk returns a tuple with the Failed field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFailed

`func (o *RolloutsMetricResult) SetFailed(v int32)`

SetFailed sets Failed field to given value.

### HasFailed

`func (o *RolloutsMetricResult) HasFailed() bool`

HasFailed returns a boolean if a field has been set.

### GetInconclusive

`func (o *RolloutsMetricResult) GetInconclusive() int32`

GetInconclusive returns the Inconclusive field if non-nil, zero value otherwise.

### GetInconclusiveOk

`func (o *RolloutsMetricResult) GetInconclusiveOk() (*int32, bool)`

GetInconclusiveOk returns a tuple with the Inconclusive field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInconclusive

`func (o *RolloutsMetricResult) SetInconclusive(v int32)`

SetInconclusive sets Inconclusive field to given value.

### HasInconclusive

`func (o *RolloutsMetricResult) HasInconclusive() bool`

HasInconclusive returns a boolean if a field has been set.

### GetMeasurements

`func (o *RolloutsMetricResult) GetMeasurements() []RolloutsMeasurement`

GetMeasurements returns the Measurements field if non-nil, zero value otherwise.

### GetMeasurementsOk

`func (o *RolloutsMetricResult) GetMeasurementsOk() (*[]RolloutsMeasurement, bool)`

GetMeasurementsOk returns a tuple with the Measurements field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMeasurements

`func (o *RolloutsMetricResult) SetMeasurements(v []RolloutsMeasurement)`

SetMeasurements sets Measurements field to given value.

### HasMeasurements

`func (o *RolloutsMetricResult) HasMeasurements() bool`

HasMeasurements returns a boolean if a field has been set.

### GetMessage

`func (o *RolloutsMetricResult) GetMessage() string`

GetMessage returns the Message field if non-nil, zero value otherwise.

### GetMessageOk

`func (o *RolloutsMetricResult) GetMessageOk() (*string, bool)`

GetMessageOk returns a tuple with the Message field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMessage

`func (o *RolloutsMetricResult) SetMessage(v string)`

SetMessage sets Message field to given value.

### HasMessage

`func (o *RolloutsMetricResult) HasMessage() bool`

HasMessage returns a boolean if a field has been set.

### GetMetadata

`func (o *RolloutsMetricResult) GetMetadata() map[string]string`

GetMetadata returns the Metadata field if non-nil, zero value otherwise.

### GetMetadataOk

`func (o *RolloutsMetricResult) GetMetadataOk() (*map[string]string, bool)`

GetMetadataOk returns a tuple with the Metadata field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMetadata

`func (o *RolloutsMetricResult) SetMetadata(v map[string]string)`

SetMetadata sets Metadata field to given value.

### HasMetadata

`func (o *RolloutsMetricResult) HasMetadata() bool`

HasMetadata returns a boolean if a field has been set.

### GetName

`func (o *RolloutsMetricResult) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *RolloutsMetricResult) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *RolloutsMetricResult) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *RolloutsMetricResult) HasName() bool`

HasName returns a boolean if a field has been set.

### GetPhase

`func (o *RolloutsMetricResult) GetPhase() string`

GetPhase returns the Phase field if non-nil, zero value otherwise.

### GetPhaseOk

`func (o *RolloutsMetricResult) GetPhaseOk() (*string, bool)`

GetPhaseOk returns a tuple with the Phase field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPhase

`func (o *RolloutsMetricResult) SetPhase(v string)`

SetPhase sets Phase field to given value.

### HasPhase

`func (o *RolloutsMetricResult) HasPhase() bool`

HasPhase returns a boolean if a field has been set.

### GetSuccessful

`func (o *RolloutsMetricResult) GetSuccessful() int32`

GetSuccessful returns the Successful field if non-nil, zero value otherwise.

### GetSuccessfulOk

`func (o *RolloutsMetricResult) GetSuccessfulOk() (*int32, bool)`

GetSuccessfulOk returns a tuple with the Successful field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSuccessful

`func (o *RolloutsMetricResult) SetSuccessful(v int32)`

SetSuccessful sets Successful field to given value.

### HasSuccessful

`func (o *RolloutsMetricResult) HasSuccessful() bool`

HasSuccessful returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


