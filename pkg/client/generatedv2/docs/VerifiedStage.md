# VerifiedStage

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**LongestSoak** | Pointer to **string** | LongestCompletedSoak represents the longest definite time interval wherein the Freight was in CONTINUOUS use by the Stage. This value is updated as Freight EXITS the Stage. If the Freight is currently in use by the Stage, the time elapsed since the Freight ENTERED the Stage is its current soak time, which may exceed the value of this field. | [optional] 
**VerifiedAt** | Pointer to **string** | VerifiedAt is the time at which the Freight was verified in the Stage. | [optional] 

## Methods

### NewVerifiedStage

`func NewVerifiedStage() *VerifiedStage`

NewVerifiedStage instantiates a new VerifiedStage object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewVerifiedStageWithDefaults

`func NewVerifiedStageWithDefaults() *VerifiedStage`

NewVerifiedStageWithDefaults instantiates a new VerifiedStage object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetLongestSoak

`func (o *VerifiedStage) GetLongestSoak() string`

GetLongestSoak returns the LongestSoak field if non-nil, zero value otherwise.

### GetLongestSoakOk

`func (o *VerifiedStage) GetLongestSoakOk() (*string, bool)`

GetLongestSoakOk returns a tuple with the LongestSoak field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLongestSoak

`func (o *VerifiedStage) SetLongestSoak(v string)`

SetLongestSoak sets LongestSoak field to given value.

### HasLongestSoak

`func (o *VerifiedStage) HasLongestSoak() bool`

HasLongestSoak returns a boolean if a field has been set.

### GetVerifiedAt

`func (o *VerifiedStage) GetVerifiedAt() string`

GetVerifiedAt returns the VerifiedAt field if non-nil, zero value otherwise.

### GetVerifiedAtOk

`func (o *VerifiedStage) GetVerifiedAtOk() (*string, bool)`

GetVerifiedAtOk returns a tuple with the VerifiedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVerifiedAt

`func (o *VerifiedStage) SetVerifiedAt(v string)`

SetVerifiedAt sets VerifiedAt field to given value.

### HasVerifiedAt

`func (o *VerifiedStage) HasVerifiedAt() bool`

HasVerifiedAt returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


