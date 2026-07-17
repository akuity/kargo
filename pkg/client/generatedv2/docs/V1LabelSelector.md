# V1LabelSelector

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**MatchExpressions** | Pointer to [**[]V1LabelSelectorRequirement**](V1LabelSelectorRequirement.md) | matchExpressions is a list of label selector requirements. The requirements are ANDed. +optional +listType&#x3D;atomic | [optional] 
**MatchLabels** | Pointer to **map[string]string** | matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is \&quot;key\&quot;, the operator is \&quot;In\&quot;, and the values array contains only \&quot;value\&quot;. The requirements are ANDed. +optional | [optional] 

## Methods

### NewV1LabelSelector

`func NewV1LabelSelector() *V1LabelSelector`

NewV1LabelSelector instantiates a new V1LabelSelector object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1LabelSelectorWithDefaults

`func NewV1LabelSelectorWithDefaults() *V1LabelSelector`

NewV1LabelSelectorWithDefaults instantiates a new V1LabelSelector object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetMatchExpressions

`func (o *V1LabelSelector) GetMatchExpressions() []V1LabelSelectorRequirement`

GetMatchExpressions returns the MatchExpressions field if non-nil, zero value otherwise.

### GetMatchExpressionsOk

`func (o *V1LabelSelector) GetMatchExpressionsOk() (*[]V1LabelSelectorRequirement, bool)`

GetMatchExpressionsOk returns a tuple with the MatchExpressions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMatchExpressions

`func (o *V1LabelSelector) SetMatchExpressions(v []V1LabelSelectorRequirement)`

SetMatchExpressions sets MatchExpressions field to given value.

### HasMatchExpressions

`func (o *V1LabelSelector) HasMatchExpressions() bool`

HasMatchExpressions returns a boolean if a field has been set.

### GetMatchLabels

`func (o *V1LabelSelector) GetMatchLabels() map[string]string`

GetMatchLabels returns the MatchLabels field if non-nil, zero value otherwise.

### GetMatchLabelsOk

`func (o *V1LabelSelector) GetMatchLabelsOk() (*map[string]string, bool)`

GetMatchLabelsOk returns a tuple with the MatchLabels field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMatchLabels

`func (o *V1LabelSelector) SetMatchLabels(v map[string]string)`

SetMatchLabels sets MatchLabels field to given value.

### HasMatchLabels

`func (o *V1LabelSelector) HasMatchLabels() bool`

HasMatchLabels returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


