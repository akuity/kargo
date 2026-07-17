# V1NodeSelector

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**NodeSelectorTerms** | Pointer to [**[]V1NodeSelectorTerm**](V1NodeSelectorTerm.md) | Required. A list of node selector terms. The terms are ORed. +listType&#x3D;atomic | [optional] 

## Methods

### NewV1NodeSelector

`func NewV1NodeSelector() *V1NodeSelector`

NewV1NodeSelector instantiates a new V1NodeSelector object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1NodeSelectorWithDefaults

`func NewV1NodeSelectorWithDefaults() *V1NodeSelector`

NewV1NodeSelectorWithDefaults instantiates a new V1NodeSelector object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetNodeSelectorTerms

`func (o *V1NodeSelector) GetNodeSelectorTerms() []V1NodeSelectorTerm`

GetNodeSelectorTerms returns the NodeSelectorTerms field if non-nil, zero value otherwise.

### GetNodeSelectorTermsOk

`func (o *V1NodeSelector) GetNodeSelectorTermsOk() (*[]V1NodeSelectorTerm, bool)`

GetNodeSelectorTermsOk returns a tuple with the NodeSelectorTerms field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNodeSelectorTerms

`func (o *V1NodeSelector) SetNodeSelectorTerms(v []V1NodeSelectorTerm)`

SetNodeSelectorTerms sets NodeSelectorTerms field to given value.

### HasNodeSelectorTerms

`func (o *V1NodeSelector) HasNodeSelectorTerms() bool`

HasNodeSelectorTerms returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


