# CurrentStage

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Since** | Pointer to **string** | Since is the time at which the Stage most recently started using the Freight. This can be used to calculate how long the Freight has been in use by the Stage. | [optional] 

## Methods

### NewCurrentStage

`func NewCurrentStage() *CurrentStage`

NewCurrentStage instantiates a new CurrentStage object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewCurrentStageWithDefaults

`func NewCurrentStageWithDefaults() *CurrentStage`

NewCurrentStageWithDefaults instantiates a new CurrentStage object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSince

`func (o *CurrentStage) GetSince() string`

GetSince returns the Since field if non-nil, zero value otherwise.

### GetSinceOk

`func (o *CurrentStage) GetSinceOk() (*string, bool)`

GetSinceOk returns a tuple with the Since field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSince

`func (o *CurrentStage) SetSince(v string)`

SetSince sets Since field to given value.

### HasSince

`func (o *CurrentStage) HasSince() bool`

HasSince returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


