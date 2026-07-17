# V1NodeSelectorTerm

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**MatchExpressions** | Pointer to [**[]V1NodeSelectorRequirement**](V1NodeSelectorRequirement.md) | A list of node selector requirements by node&#39;s labels. +optional +listType&#x3D;atomic | [optional] 
**MatchFields** | Pointer to [**[]V1NodeSelectorRequirement**](V1NodeSelectorRequirement.md) | A list of node selector requirements by node&#39;s fields. +optional +listType&#x3D;atomic | [optional] 

## Methods

### NewV1NodeSelectorTerm

`func NewV1NodeSelectorTerm() *V1NodeSelectorTerm`

NewV1NodeSelectorTerm instantiates a new V1NodeSelectorTerm object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1NodeSelectorTermWithDefaults

`func NewV1NodeSelectorTermWithDefaults() *V1NodeSelectorTerm`

NewV1NodeSelectorTermWithDefaults instantiates a new V1NodeSelectorTerm object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetMatchExpressions

`func (o *V1NodeSelectorTerm) GetMatchExpressions() []V1NodeSelectorRequirement`

GetMatchExpressions returns the MatchExpressions field if non-nil, zero value otherwise.

### GetMatchExpressionsOk

`func (o *V1NodeSelectorTerm) GetMatchExpressionsOk() (*[]V1NodeSelectorRequirement, bool)`

GetMatchExpressionsOk returns a tuple with the MatchExpressions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMatchExpressions

`func (o *V1NodeSelectorTerm) SetMatchExpressions(v []V1NodeSelectorRequirement)`

SetMatchExpressions sets MatchExpressions field to given value.

### HasMatchExpressions

`func (o *V1NodeSelectorTerm) HasMatchExpressions() bool`

HasMatchExpressions returns a boolean if a field has been set.

### GetMatchFields

`func (o *V1NodeSelectorTerm) GetMatchFields() []V1NodeSelectorRequirement`

GetMatchFields returns the MatchFields field if non-nil, zero value otherwise.

### GetMatchFieldsOk

`func (o *V1NodeSelectorTerm) GetMatchFieldsOk() (*[]V1NodeSelectorRequirement, bool)`

GetMatchFieldsOk returns a tuple with the MatchFields field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMatchFields

`func (o *V1NodeSelectorTerm) SetMatchFields(v []V1NodeSelectorRequirement)`

SetMatchFields sets MatchFields field to given value.

### HasMatchFields

`func (o *V1NodeSelectorTerm) HasMatchFields() bool`

HasMatchFields returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


