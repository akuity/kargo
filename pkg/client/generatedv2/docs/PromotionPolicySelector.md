# PromotionPolicySelector

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**MatchExpressions** | Pointer to [**[]V1LabelSelectorRequirement**](V1LabelSelectorRequirement.md) | matchExpressions is a list of label selector requirements. The requirements are ANDed. +optional +listType&#x3D;atomic | [optional] 
**MatchLabels** | Pointer to **map[string]string** | matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is \&quot;key\&quot;, the operator is \&quot;In\&quot;, and the values array contains only \&quot;value\&quot;. The requirements are ANDed. +optional | [optional] 
**Name** | Pointer to **string** | Name is the name of the resource to which this policy applies.  It can be an exact name, a regex pattern (with prefix \&quot;regex:\&quot;), or a glob pattern (with prefix \&quot;glob:\&quot;).  When both Name and LabelSelector are specified, the Name is ANDed with the LabelSelector. I.e., the resource must match both the Name and LabelSelector to be selected by this policy.  NOTE: Using a specific exact name is the most secure option. Pattern matching via regex or glob can be exploited by users with permissions to match promotion policies that weren&#39;t intended to apply to their resources. For example, a user could create a resource with a name deliberately crafted to match the pattern, potentially bypassing intended promotion controls.  +optional | [optional] 

## Methods

### NewPromotionPolicySelector

`func NewPromotionPolicySelector() *PromotionPolicySelector`

NewPromotionPolicySelector instantiates a new PromotionPolicySelector object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPromotionPolicySelectorWithDefaults

`func NewPromotionPolicySelectorWithDefaults() *PromotionPolicySelector`

NewPromotionPolicySelectorWithDefaults instantiates a new PromotionPolicySelector object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetMatchExpressions

`func (o *PromotionPolicySelector) GetMatchExpressions() []V1LabelSelectorRequirement`

GetMatchExpressions returns the MatchExpressions field if non-nil, zero value otherwise.

### GetMatchExpressionsOk

`func (o *PromotionPolicySelector) GetMatchExpressionsOk() (*[]V1LabelSelectorRequirement, bool)`

GetMatchExpressionsOk returns a tuple with the MatchExpressions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMatchExpressions

`func (o *PromotionPolicySelector) SetMatchExpressions(v []V1LabelSelectorRequirement)`

SetMatchExpressions sets MatchExpressions field to given value.

### HasMatchExpressions

`func (o *PromotionPolicySelector) HasMatchExpressions() bool`

HasMatchExpressions returns a boolean if a field has been set.

### GetMatchLabels

`func (o *PromotionPolicySelector) GetMatchLabels() map[string]string`

GetMatchLabels returns the MatchLabels field if non-nil, zero value otherwise.

### GetMatchLabelsOk

`func (o *PromotionPolicySelector) GetMatchLabelsOk() (*map[string]string, bool)`

GetMatchLabelsOk returns a tuple with the MatchLabels field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMatchLabels

`func (o *PromotionPolicySelector) SetMatchLabels(v map[string]string)`

SetMatchLabels sets MatchLabels field to given value.

### HasMatchLabels

`func (o *PromotionPolicySelector) HasMatchLabels() bool`

HasMatchLabels returns a boolean if a field has been set.

### GetName

`func (o *PromotionPolicySelector) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *PromotionPolicySelector) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *PromotionPolicySelector) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *PromotionPolicySelector) HasName() bool`

HasName returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


