# FreightStatus

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ApprovedFor** | Pointer to [**map[string]ApprovedStage**](ApprovedStage.md) | ApprovedFor describes the Stages for which this Freight has been approved preemptively/manually by a user. This is useful for hotfixes, where one might wish to promote a piece of Freight to a given Stage without transiting the entire pipeline. | [optional] 
**CurrentlyIn** | Pointer to [**map[string]CurrentStage**](CurrentStage.md) | CurrentlyIn describes the Stages in which this Freight is currently in use. | [optional] 
**Metadata** | Pointer to **map[string]interface{}** | Metadata is a map of arbitrary metadata associated with the Freight. This is useful for storing additional information about the Freight or Promotion that can be shared across steps or stages. | [optional] 
**VerifiedIn** | Pointer to [**map[string]VerifiedStage**](VerifiedStage.md) | VerifiedIn describes the Stages in which this Freight has been verified through promotion and subsequent health checks. | [optional] 

## Methods

### NewFreightStatus

`func NewFreightStatus() *FreightStatus`

NewFreightStatus instantiates a new FreightStatus object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewFreightStatusWithDefaults

`func NewFreightStatusWithDefaults() *FreightStatus`

NewFreightStatusWithDefaults instantiates a new FreightStatus object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetApprovedFor

`func (o *FreightStatus) GetApprovedFor() map[string]ApprovedStage`

GetApprovedFor returns the ApprovedFor field if non-nil, zero value otherwise.

### GetApprovedForOk

`func (o *FreightStatus) GetApprovedForOk() (*map[string]ApprovedStage, bool)`

GetApprovedForOk returns a tuple with the ApprovedFor field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetApprovedFor

`func (o *FreightStatus) SetApprovedFor(v map[string]ApprovedStage)`

SetApprovedFor sets ApprovedFor field to given value.

### HasApprovedFor

`func (o *FreightStatus) HasApprovedFor() bool`

HasApprovedFor returns a boolean if a field has been set.

### GetCurrentlyIn

`func (o *FreightStatus) GetCurrentlyIn() map[string]CurrentStage`

GetCurrentlyIn returns the CurrentlyIn field if non-nil, zero value otherwise.

### GetCurrentlyInOk

`func (o *FreightStatus) GetCurrentlyInOk() (*map[string]CurrentStage, bool)`

GetCurrentlyInOk returns a tuple with the CurrentlyIn field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCurrentlyIn

`func (o *FreightStatus) SetCurrentlyIn(v map[string]CurrentStage)`

SetCurrentlyIn sets CurrentlyIn field to given value.

### HasCurrentlyIn

`func (o *FreightStatus) HasCurrentlyIn() bool`

HasCurrentlyIn returns a boolean if a field has been set.

### GetMetadata

`func (o *FreightStatus) GetMetadata() map[string]interface{}`

GetMetadata returns the Metadata field if non-nil, zero value otherwise.

### GetMetadataOk

`func (o *FreightStatus) GetMetadataOk() (*map[string]interface{}, bool)`

GetMetadataOk returns a tuple with the Metadata field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMetadata

`func (o *FreightStatus) SetMetadata(v map[string]interface{})`

SetMetadata sets Metadata field to given value.

### HasMetadata

`func (o *FreightStatus) HasMetadata() bool`

HasMetadata returns a boolean if a field has been set.

### GetVerifiedIn

`func (o *FreightStatus) GetVerifiedIn() map[string]VerifiedStage`

GetVerifiedIn returns the VerifiedIn field if non-nil, zero value otherwise.

### GetVerifiedInOk

`func (o *FreightStatus) GetVerifiedInOk() (*map[string]VerifiedStage, bool)`

GetVerifiedInOk returns a tuple with the VerifiedIn field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVerifiedIn

`func (o *FreightStatus) SetVerifiedIn(v map[string]VerifiedStage)`

SetVerifiedIn sets VerifiedIn field to given value.

### HasVerifiedIn

`func (o *FreightStatus) HasVerifiedIn() bool`

HasVerifiedIn returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


