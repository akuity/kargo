# IndexSelectorRequirement

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Key** | Pointer to **string** | Key is the key of the index.  +kubebuilder:validation:Enum&#x3D;subscribedURLs;receiverPaths | [optional] 
**Operator** | Pointer to **string** | Operator indicates the operation that should be used to evaluate whether the selection requirement is satisfied.  kubebuilder:validation:Enum&#x3D;Equal;NotEqual; | [optional] 
**Value** | Pointer to **string** | Value can be a static string or an expression that will be evaluated.  kubebuilder:validation:Required | [optional] 

## Methods

### NewIndexSelectorRequirement

`func NewIndexSelectorRequirement() *IndexSelectorRequirement`

NewIndexSelectorRequirement instantiates a new IndexSelectorRequirement object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewIndexSelectorRequirementWithDefaults

`func NewIndexSelectorRequirementWithDefaults() *IndexSelectorRequirement`

NewIndexSelectorRequirementWithDefaults instantiates a new IndexSelectorRequirement object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetKey

`func (o *IndexSelectorRequirement) GetKey() string`

GetKey returns the Key field if non-nil, zero value otherwise.

### GetKeyOk

`func (o *IndexSelectorRequirement) GetKeyOk() (*string, bool)`

GetKeyOk returns a tuple with the Key field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKey

`func (o *IndexSelectorRequirement) SetKey(v string)`

SetKey sets Key field to given value.

### HasKey

`func (o *IndexSelectorRequirement) HasKey() bool`

HasKey returns a boolean if a field has been set.

### GetOperator

`func (o *IndexSelectorRequirement) GetOperator() string`

GetOperator returns the Operator field if non-nil, zero value otherwise.

### GetOperatorOk

`func (o *IndexSelectorRequirement) GetOperatorOk() (*string, bool)`

GetOperatorOk returns a tuple with the Operator field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOperator

`func (o *IndexSelectorRequirement) SetOperator(v string)`

SetOperator sets Operator field to given value.

### HasOperator

`func (o *IndexSelectorRequirement) HasOperator() bool`

HasOperator returns a boolean if a field has been set.

### GetValue

`func (o *IndexSelectorRequirement) GetValue() string`

GetValue returns the Value field if non-nil, zero value otherwise.

### GetValueOk

`func (o *IndexSelectorRequirement) GetValueOk() (*string, bool)`

GetValueOk returns a tuple with the Value field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetValue

`func (o *IndexSelectorRequirement) SetValue(v string)`

SetValue sets Value field to given value.

### HasValue

`func (o *IndexSelectorRequirement) HasValue() bool`

HasValue returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


