# V1SuccessPolicyRule

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**SucceededCount** | Pointer to **int32** | succeededCount specifies the minimal required size of the actual set of the succeeded indexes for the Job. When succeededCount is used along with succeededIndexes, the check is constrained only to the set of indexes specified by succeededIndexes. For example, given that succeededIndexes is \&quot;1-4\&quot;, succeededCount is \&quot;3\&quot;, and completed indexes are \&quot;1\&quot;, \&quot;3\&quot;, and \&quot;5\&quot;, the Job isn&#39;t declared as succeeded because only \&quot;1\&quot; and \&quot;3\&quot; indexes are considered in that rules. When this field is null, this doesn&#39;t default to any value and is never evaluated at any time. When specified it needs to be a positive integer.  +optional | [optional] 
**SucceededIndexes** | Pointer to **string** | succeededIndexes specifies the set of indexes which need to be contained in the actual set of the succeeded indexes for the Job. The list of indexes must be within 0 to \&quot;.spec.completions-1\&quot; and must not contain duplicates. At least one element is required. The indexes are represented as intervals separated by commas. The intervals can be a decimal integer or a pair of decimal integers separated by a hyphen. The number are listed in represented by the first and last element of the series, separated by a hyphen. For example, if the completed indexes are 1, 3, 4, 5 and 7, they are represented as \&quot;1,3-5,7\&quot;. When this field is null, this field doesn&#39;t default to any value and is never evaluated at any time.  +optional | [optional] 

## Methods

### NewV1SuccessPolicyRule

`func NewV1SuccessPolicyRule() *V1SuccessPolicyRule`

NewV1SuccessPolicyRule instantiates a new V1SuccessPolicyRule object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1SuccessPolicyRuleWithDefaults

`func NewV1SuccessPolicyRuleWithDefaults() *V1SuccessPolicyRule`

NewV1SuccessPolicyRuleWithDefaults instantiates a new V1SuccessPolicyRule object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSucceededCount

`func (o *V1SuccessPolicyRule) GetSucceededCount() int32`

GetSucceededCount returns the SucceededCount field if non-nil, zero value otherwise.

### GetSucceededCountOk

`func (o *V1SuccessPolicyRule) GetSucceededCountOk() (*int32, bool)`

GetSucceededCountOk returns a tuple with the SucceededCount field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSucceededCount

`func (o *V1SuccessPolicyRule) SetSucceededCount(v int32)`

SetSucceededCount sets SucceededCount field to given value.

### HasSucceededCount

`func (o *V1SuccessPolicyRule) HasSucceededCount() bool`

HasSucceededCount returns a boolean if a field has been set.

### GetSucceededIndexes

`func (o *V1SuccessPolicyRule) GetSucceededIndexes() string`

GetSucceededIndexes returns the SucceededIndexes field if non-nil, zero value otherwise.

### GetSucceededIndexesOk

`func (o *V1SuccessPolicyRule) GetSucceededIndexesOk() (*string, bool)`

GetSucceededIndexesOk returns a tuple with the SucceededIndexes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSucceededIndexes

`func (o *V1SuccessPolicyRule) SetSucceededIndexes(v string)`

SetSucceededIndexes sets SucceededIndexes field to given value.

### HasSucceededIndexes

`func (o *V1SuccessPolicyRule) HasSucceededIndexes() bool`

HasSucceededIndexes returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


