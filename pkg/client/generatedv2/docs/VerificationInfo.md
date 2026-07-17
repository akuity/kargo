# VerificationInfo

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Actor** | Pointer to **string** | Actor is the name of the entity that initiated or aborted the Verification process. | [optional] 
**AnalysisRun** | Pointer to [**AnalysisRunReference**](AnalysisRunReference.md) | AnalysisRun is a reference to the Argo Rollouts AnalysisRun that implements the Verification process. | [optional] 
**FinishTime** | Pointer to **string** | FinishTime is the time at which the Verification process finished. | [optional] 
**Id** | Pointer to **string** | ID is the identifier of the Verification process. | [optional] 
**Message** | Pointer to **string** | Message may contain additional information about why the verification process is in its current phase. | [optional] 
**Phase** | Pointer to **string** | Phase describes the current phase of the Verification process. Generally, this will be a reflection of the underlying AnalysisRun&#39;s phase, however, there are exceptions to this, such as in the case where an AnalysisRun cannot be launched successfully. | [optional] 
**StartTime** | Pointer to **string** | StartTime is the time at which the Verification process was started. | [optional] 

## Methods

### NewVerificationInfo

`func NewVerificationInfo() *VerificationInfo`

NewVerificationInfo instantiates a new VerificationInfo object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewVerificationInfoWithDefaults

`func NewVerificationInfoWithDefaults() *VerificationInfo`

NewVerificationInfoWithDefaults instantiates a new VerificationInfo object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetActor

`func (o *VerificationInfo) GetActor() string`

GetActor returns the Actor field if non-nil, zero value otherwise.

### GetActorOk

`func (o *VerificationInfo) GetActorOk() (*string, bool)`

GetActorOk returns a tuple with the Actor field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetActor

`func (o *VerificationInfo) SetActor(v string)`

SetActor sets Actor field to given value.

### HasActor

`func (o *VerificationInfo) HasActor() bool`

HasActor returns a boolean if a field has been set.

### GetAnalysisRun

`func (o *VerificationInfo) GetAnalysisRun() AnalysisRunReference`

GetAnalysisRun returns the AnalysisRun field if non-nil, zero value otherwise.

### GetAnalysisRunOk

`func (o *VerificationInfo) GetAnalysisRunOk() (*AnalysisRunReference, bool)`

GetAnalysisRunOk returns a tuple with the AnalysisRun field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAnalysisRun

`func (o *VerificationInfo) SetAnalysisRun(v AnalysisRunReference)`

SetAnalysisRun sets AnalysisRun field to given value.

### HasAnalysisRun

`func (o *VerificationInfo) HasAnalysisRun() bool`

HasAnalysisRun returns a boolean if a field has been set.

### GetFinishTime

`func (o *VerificationInfo) GetFinishTime() string`

GetFinishTime returns the FinishTime field if non-nil, zero value otherwise.

### GetFinishTimeOk

`func (o *VerificationInfo) GetFinishTimeOk() (*string, bool)`

GetFinishTimeOk returns a tuple with the FinishTime field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFinishTime

`func (o *VerificationInfo) SetFinishTime(v string)`

SetFinishTime sets FinishTime field to given value.

### HasFinishTime

`func (o *VerificationInfo) HasFinishTime() bool`

HasFinishTime returns a boolean if a field has been set.

### GetId

`func (o *VerificationInfo) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *VerificationInfo) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *VerificationInfo) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *VerificationInfo) HasId() bool`

HasId returns a boolean if a field has been set.

### GetMessage

`func (o *VerificationInfo) GetMessage() string`

GetMessage returns the Message field if non-nil, zero value otherwise.

### GetMessageOk

`func (o *VerificationInfo) GetMessageOk() (*string, bool)`

GetMessageOk returns a tuple with the Message field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMessage

`func (o *VerificationInfo) SetMessage(v string)`

SetMessage sets Message field to given value.

### HasMessage

`func (o *VerificationInfo) HasMessage() bool`

HasMessage returns a boolean if a field has been set.

### GetPhase

`func (o *VerificationInfo) GetPhase() string`

GetPhase returns the Phase field if non-nil, zero value otherwise.

### GetPhaseOk

`func (o *VerificationInfo) GetPhaseOk() (*string, bool)`

GetPhaseOk returns a tuple with the Phase field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPhase

`func (o *VerificationInfo) SetPhase(v string)`

SetPhase sets Phase field to given value.

### HasPhase

`func (o *VerificationInfo) HasPhase() bool`

HasPhase returns a boolean if a field has been set.

### GetStartTime

`func (o *VerificationInfo) GetStartTime() string`

GetStartTime returns the StartTime field if non-nil, zero value otherwise.

### GetStartTimeOk

`func (o *VerificationInfo) GetStartTimeOk() (*string, bool)`

GetStartTimeOk returns a tuple with the StartTime field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStartTime

`func (o *VerificationInfo) SetStartTime(v string)`

SetStartTime sets StartTime field to given value.

### HasStartTime

`func (o *VerificationInfo) HasStartTime() bool`

HasStartTime returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


