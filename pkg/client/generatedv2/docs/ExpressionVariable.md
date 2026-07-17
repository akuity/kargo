# ExpressionVariable

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | Pointer to **string** | Name is the name of the variable.  +kubebuilder:validation:MinLength&#x3D;1 +kubebuilder:validation:Pattern&#x3D;^[a-zA-Z_]\\w*$ | [optional] 
**Value** | Pointer to **string** | Value is the value of the variable. It is allowed to utilize expressions in the value. See https://docs.kargo.io/user-guide/reference-docs/expressions for details. | [optional] 

## Methods

### NewExpressionVariable

`func NewExpressionVariable() *ExpressionVariable`

NewExpressionVariable instantiates a new ExpressionVariable object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewExpressionVariableWithDefaults

`func NewExpressionVariableWithDefaults() *ExpressionVariable`

NewExpressionVariableWithDefaults instantiates a new ExpressionVariable object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetName

`func (o *ExpressionVariable) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *ExpressionVariable) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *ExpressionVariable) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *ExpressionVariable) HasName() bool`

HasName returns a boolean if a field has been set.

### GetValue

`func (o *ExpressionVariable) GetValue() string`

GetValue returns the Value field if non-nil, zero value otherwise.

### GetValueOk

`func (o *ExpressionVariable) GetValueOk() (*string, bool)`

GetValueOk returns a tuple with the Value field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetValue

`func (o *ExpressionVariable) SetValue(v string)`

SetValue sets Value field to given value.

### HasValue

`func (o *ExpressionVariable) HasValue() bool`

HasValue returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


