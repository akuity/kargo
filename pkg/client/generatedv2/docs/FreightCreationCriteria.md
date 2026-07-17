# FreightCreationCriteria

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Expression** | Pointer to **string** | Expression is an expr-lang expression that must evaluate to true for Freight to be created automatically from new artifacts following discovery. | [optional] 

## Methods

### NewFreightCreationCriteria

`func NewFreightCreationCriteria() *FreightCreationCriteria`

NewFreightCreationCriteria instantiates a new FreightCreationCriteria object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewFreightCreationCriteriaWithDefaults

`func NewFreightCreationCriteriaWithDefaults() *FreightCreationCriteria`

NewFreightCreationCriteriaWithDefaults instantiates a new FreightCreationCriteria object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetExpression

`func (o *FreightCreationCriteria) GetExpression() string`

GetExpression returns the Expression field if non-nil, zero value otherwise.

### GetExpressionOk

`func (o *FreightCreationCriteria) GetExpressionOk() (*string, bool)`

GetExpressionOk returns a tuple with the Expression field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExpression

`func (o *FreightCreationCriteria) SetExpression(v string)`

SetExpression sets Expression field to given value.

### HasExpression

`func (o *FreightCreationCriteria) HasExpression() bool`

HasExpression returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


