# StepExecutionMetadata

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Alias** | Pointer to **string** | Alias is the alias of the step. | [optional] 
**ContinueOnError** | Pointer to **bool** | ContinueOnError is a boolean value that, if set to true, will cause the Promotion to continue executing the next step even if this step fails. It also will not permit this failure to impact the overall status of the Promotion. | [optional] 
**ErrorCount** | Pointer to **int32** | ErrorCount tracks consecutive failed attempts to execute the step. | [optional] 
**FinishedAt** | Pointer to **string** | FinishedAt is the time at which the final attempt to execute the step completed. | [optional] 
**Message** | Pointer to **string** | Message is a display message about the step, including any errors. | [optional] 
**StartedAt** | Pointer to **string** | StartedAt is the time at which the first attempt to execute the step began. | [optional] 
**Status** | Pointer to **string** | Status is the high-level outcome of the step. | [optional] 

## Methods

### NewStepExecutionMetadata

`func NewStepExecutionMetadata() *StepExecutionMetadata`

NewStepExecutionMetadata instantiates a new StepExecutionMetadata object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewStepExecutionMetadataWithDefaults

`func NewStepExecutionMetadataWithDefaults() *StepExecutionMetadata`

NewStepExecutionMetadataWithDefaults instantiates a new StepExecutionMetadata object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAlias

`func (o *StepExecutionMetadata) GetAlias() string`

GetAlias returns the Alias field if non-nil, zero value otherwise.

### GetAliasOk

`func (o *StepExecutionMetadata) GetAliasOk() (*string, bool)`

GetAliasOk returns a tuple with the Alias field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAlias

`func (o *StepExecutionMetadata) SetAlias(v string)`

SetAlias sets Alias field to given value.

### HasAlias

`func (o *StepExecutionMetadata) HasAlias() bool`

HasAlias returns a boolean if a field has been set.

### GetContinueOnError

`func (o *StepExecutionMetadata) GetContinueOnError() bool`

GetContinueOnError returns the ContinueOnError field if non-nil, zero value otherwise.

### GetContinueOnErrorOk

`func (o *StepExecutionMetadata) GetContinueOnErrorOk() (*bool, bool)`

GetContinueOnErrorOk returns a tuple with the ContinueOnError field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetContinueOnError

`func (o *StepExecutionMetadata) SetContinueOnError(v bool)`

SetContinueOnError sets ContinueOnError field to given value.

### HasContinueOnError

`func (o *StepExecutionMetadata) HasContinueOnError() bool`

HasContinueOnError returns a boolean if a field has been set.

### GetErrorCount

`func (o *StepExecutionMetadata) GetErrorCount() int32`

GetErrorCount returns the ErrorCount field if non-nil, zero value otherwise.

### GetErrorCountOk

`func (o *StepExecutionMetadata) GetErrorCountOk() (*int32, bool)`

GetErrorCountOk returns a tuple with the ErrorCount field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetErrorCount

`func (o *StepExecutionMetadata) SetErrorCount(v int32)`

SetErrorCount sets ErrorCount field to given value.

### HasErrorCount

`func (o *StepExecutionMetadata) HasErrorCount() bool`

HasErrorCount returns a boolean if a field has been set.

### GetFinishedAt

`func (o *StepExecutionMetadata) GetFinishedAt() string`

GetFinishedAt returns the FinishedAt field if non-nil, zero value otherwise.

### GetFinishedAtOk

`func (o *StepExecutionMetadata) GetFinishedAtOk() (*string, bool)`

GetFinishedAtOk returns a tuple with the FinishedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFinishedAt

`func (o *StepExecutionMetadata) SetFinishedAt(v string)`

SetFinishedAt sets FinishedAt field to given value.

### HasFinishedAt

`func (o *StepExecutionMetadata) HasFinishedAt() bool`

HasFinishedAt returns a boolean if a field has been set.

### GetMessage

`func (o *StepExecutionMetadata) GetMessage() string`

GetMessage returns the Message field if non-nil, zero value otherwise.

### GetMessageOk

`func (o *StepExecutionMetadata) GetMessageOk() (*string, bool)`

GetMessageOk returns a tuple with the Message field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMessage

`func (o *StepExecutionMetadata) SetMessage(v string)`

SetMessage sets Message field to given value.

### HasMessage

`func (o *StepExecutionMetadata) HasMessage() bool`

HasMessage returns a boolean if a field has been set.

### GetStartedAt

`func (o *StepExecutionMetadata) GetStartedAt() string`

GetStartedAt returns the StartedAt field if non-nil, zero value otherwise.

### GetStartedAtOk

`func (o *StepExecutionMetadata) GetStartedAtOk() (*string, bool)`

GetStartedAtOk returns a tuple with the StartedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStartedAt

`func (o *StepExecutionMetadata) SetStartedAt(v string)`

SetStartedAt sets StartedAt field to given value.

### HasStartedAt

`func (o *StepExecutionMetadata) HasStartedAt() bool`

HasStartedAt returns a boolean if a field has been set.

### GetStatus

`func (o *StepExecutionMetadata) GetStatus() string`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *StepExecutionMetadata) GetStatusOk() (*string, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *StepExecutionMetadata) SetStatus(v string)`

SetStatus sets Status field to given value.

### HasStatus

`func (o *StepExecutionMetadata) HasStatus() bool`

HasStatus returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


