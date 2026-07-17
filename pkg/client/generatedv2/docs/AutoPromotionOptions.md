# AutoPromotionOptions

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**SelectionPolicy** | Pointer to **string** | SelectionPolicy specifies the rules for identifying new Freight that is eligible for auto-promotion to this Stage. This field is optional. When left unspecified, the field is implicitly treated as if its value were \&quot;NewestFreight\&quot;.  Accepted Values:  - \&quot;NewestFreight\&quot;: The newest Freight that is available to the Stage is   eligible for auto-promotion.  - \&quot;MatchUpstream\&quot;: Only the Freight currently used immediately upstream   from this Stage is eligible for auto-promotion. This policy may only   be applied when the Stage has exactly one upstream Stage. | [optional] 

## Methods

### NewAutoPromotionOptions

`func NewAutoPromotionOptions() *AutoPromotionOptions`

NewAutoPromotionOptions instantiates a new AutoPromotionOptions object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAutoPromotionOptionsWithDefaults

`func NewAutoPromotionOptionsWithDefaults() *AutoPromotionOptions`

NewAutoPromotionOptionsWithDefaults instantiates a new AutoPromotionOptions object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSelectionPolicy

`func (o *AutoPromotionOptions) GetSelectionPolicy() string`

GetSelectionPolicy returns the SelectionPolicy field if non-nil, zero value otherwise.

### GetSelectionPolicyOk

`func (o *AutoPromotionOptions) GetSelectionPolicyOk() (*string, bool)`

GetSelectionPolicyOk returns a tuple with the SelectionPolicy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSelectionPolicy

`func (o *AutoPromotionOptions) SetSelectionPolicy(v string)`

SetSelectionPolicy sets SelectionPolicy field to given value.

### HasSelectionPolicy

`func (o *AutoPromotionOptions) HasSelectionPolicy() bool`

HasSelectionPolicy returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


