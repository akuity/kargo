# V1LabelSelectorRequirement

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Key** | Pointer to **string** | key is the label key that the selector applies to. | [optional] 
**Operator** | Pointer to **string** | operator represents a key&#39;s relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist. | [optional] 
**Values** | Pointer to **[]string** | values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch. +optional +listType&#x3D;atomic | [optional] 

## Methods

### NewV1LabelSelectorRequirement

`func NewV1LabelSelectorRequirement() *V1LabelSelectorRequirement`

NewV1LabelSelectorRequirement instantiates a new V1LabelSelectorRequirement object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1LabelSelectorRequirementWithDefaults

`func NewV1LabelSelectorRequirementWithDefaults() *V1LabelSelectorRequirement`

NewV1LabelSelectorRequirementWithDefaults instantiates a new V1LabelSelectorRequirement object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetKey

`func (o *V1LabelSelectorRequirement) GetKey() string`

GetKey returns the Key field if non-nil, zero value otherwise.

### GetKeyOk

`func (o *V1LabelSelectorRequirement) GetKeyOk() (*string, bool)`

GetKeyOk returns a tuple with the Key field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKey

`func (o *V1LabelSelectorRequirement) SetKey(v string)`

SetKey sets Key field to given value.

### HasKey

`func (o *V1LabelSelectorRequirement) HasKey() bool`

HasKey returns a boolean if a field has been set.

### GetOperator

`func (o *V1LabelSelectorRequirement) GetOperator() string`

GetOperator returns the Operator field if non-nil, zero value otherwise.

### GetOperatorOk

`func (o *V1LabelSelectorRequirement) GetOperatorOk() (*string, bool)`

GetOperatorOk returns a tuple with the Operator field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOperator

`func (o *V1LabelSelectorRequirement) SetOperator(v string)`

SetOperator sets Operator field to given value.

### HasOperator

`func (o *V1LabelSelectorRequirement) HasOperator() bool`

HasOperator returns a boolean if a field has been set.

### GetValues

`func (o *V1LabelSelectorRequirement) GetValues() []string`

GetValues returns the Values field if non-nil, zero value otherwise.

### GetValuesOk

`func (o *V1LabelSelectorRequirement) GetValuesOk() (*[]string, bool)`

GetValuesOk returns a tuple with the Values field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetValues

`func (o *V1LabelSelectorRequirement) SetValues(v []string)`

SetValues sets Values field to given value.

### HasValues

`func (o *V1LabelSelectorRequirement) HasValues() bool`

HasValues returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


