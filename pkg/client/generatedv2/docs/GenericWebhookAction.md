# GenericWebhookAction

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Action** | Pointer to **string** | ActionType indicates the type of action to be performed. &#x60;Refresh&#x60; is the only currently supported action.  +kubebuilder:validation:Enum&#x3D;Refresh; | [optional] 
**Parameters** | Pointer to **map[string]string** | Parameters contains additional, action-specific parameters. Values may be static or extracted from the request using expressions.  +optional | [optional] 
**TargetSelectionCriteria** | Pointer to [**[]GenericWebhookTargetSelectionCriteria**](GenericWebhookTargetSelectionCriteria.md) | TargetSelectionCriteria is a list of selection criteria for the resources on which the action should be performed.  +kubebuilder:validation:MinItems&#x3D;1 | [optional] 
**WhenExpression** | Pointer to **string** | WhenExpression defines criteria that a request must meet to run this action.  +optional | [optional] 

## Methods

### NewGenericWebhookAction

`func NewGenericWebhookAction() *GenericWebhookAction`

NewGenericWebhookAction instantiates a new GenericWebhookAction object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewGenericWebhookActionWithDefaults

`func NewGenericWebhookActionWithDefaults() *GenericWebhookAction`

NewGenericWebhookActionWithDefaults instantiates a new GenericWebhookAction object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAction

`func (o *GenericWebhookAction) GetAction() string`

GetAction returns the Action field if non-nil, zero value otherwise.

### GetActionOk

`func (o *GenericWebhookAction) GetActionOk() (*string, bool)`

GetActionOk returns a tuple with the Action field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAction

`func (o *GenericWebhookAction) SetAction(v string)`

SetAction sets Action field to given value.

### HasAction

`func (o *GenericWebhookAction) HasAction() bool`

HasAction returns a boolean if a field has been set.

### GetParameters

`func (o *GenericWebhookAction) GetParameters() map[string]string`

GetParameters returns the Parameters field if non-nil, zero value otherwise.

### GetParametersOk

`func (o *GenericWebhookAction) GetParametersOk() (*map[string]string, bool)`

GetParametersOk returns a tuple with the Parameters field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetParameters

`func (o *GenericWebhookAction) SetParameters(v map[string]string)`

SetParameters sets Parameters field to given value.

### HasParameters

`func (o *GenericWebhookAction) HasParameters() bool`

HasParameters returns a boolean if a field has been set.

### GetTargetSelectionCriteria

`func (o *GenericWebhookAction) GetTargetSelectionCriteria() []GenericWebhookTargetSelectionCriteria`

GetTargetSelectionCriteria returns the TargetSelectionCriteria field if non-nil, zero value otherwise.

### GetTargetSelectionCriteriaOk

`func (o *GenericWebhookAction) GetTargetSelectionCriteriaOk() (*[]GenericWebhookTargetSelectionCriteria, bool)`

GetTargetSelectionCriteriaOk returns a tuple with the TargetSelectionCriteria field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTargetSelectionCriteria

`func (o *GenericWebhookAction) SetTargetSelectionCriteria(v []GenericWebhookTargetSelectionCriteria)`

SetTargetSelectionCriteria sets TargetSelectionCriteria field to given value.

### HasTargetSelectionCriteria

`func (o *GenericWebhookAction) HasTargetSelectionCriteria() bool`

HasTargetSelectionCriteria returns a boolean if a field has been set.

### GetWhenExpression

`func (o *GenericWebhookAction) GetWhenExpression() string`

GetWhenExpression returns the WhenExpression field if non-nil, zero value otherwise.

### GetWhenExpressionOk

`func (o *GenericWebhookAction) GetWhenExpressionOk() (*string, bool)`

GetWhenExpressionOk returns a tuple with the WhenExpression field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWhenExpression

`func (o *GenericWebhookAction) SetWhenExpression(v string)`

SetWhenExpression sets WhenExpression field to given value.

### HasWhenExpression

`func (o *GenericWebhookAction) HasWhenExpression() bool`

HasWhenExpression returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


