# IndexSelector

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**MatchIndices** | Pointer to [**[]IndexSelectorRequirement**](IndexSelectorRequirement.md) | MatchIndices is a list of index selector requirements.  +kubebuilder:validation:MinItems&#x3D;1 | [optional] 

## Methods

### NewIndexSelector

`func NewIndexSelector() *IndexSelector`

NewIndexSelector instantiates a new IndexSelector object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewIndexSelectorWithDefaults

`func NewIndexSelectorWithDefaults() *IndexSelector`

NewIndexSelectorWithDefaults instantiates a new IndexSelector object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetMatchIndices

`func (o *IndexSelector) GetMatchIndices() []IndexSelectorRequirement`

GetMatchIndices returns the MatchIndices field if non-nil, zero value otherwise.

### GetMatchIndicesOk

`func (o *IndexSelector) GetMatchIndicesOk() (*[]IndexSelectorRequirement, bool)`

GetMatchIndicesOk returns a tuple with the MatchIndices field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMatchIndices

`func (o *IndexSelector) SetMatchIndices(v []IndexSelectorRequirement)`

SetMatchIndices sets MatchIndices field to given value.

### HasMatchIndices

`func (o *IndexSelector) HasMatchIndices() bool`

HasMatchIndices returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


