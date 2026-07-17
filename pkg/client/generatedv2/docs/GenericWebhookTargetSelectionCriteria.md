# GenericWebhookTargetSelectionCriteria

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**IndexSelector** | Pointer to [**IndexSelector**](IndexSelector.md) | IndexSelector is a selector used to identify cached target resources by cache key. If used with LabelSelector and/or Name, the results are the combined (logical AND) of all the criteria.  +optional | [optional] 
**Kind** | Pointer to **string** | Kind is the kind of the target resource.  +kubebuilder:validation:Enum&#x3D;Warehouse; | [optional] 
**LabelSelector** | Pointer to [**V1LabelSelector**](V1LabelSelector.md) | LabelSelector is a label selector to identify the target resources. If used with IndexSelector and/or Name, the results are the combined (logical AND) of all the criteria.  +optional | [optional] 
**Name** | Pointer to **string** | Name is the name of the target resource. If LabelSelector and/or IndexSelectors are also specified, the results are the combined (logical AND) of the criteria.  +optional | [optional] 

## Methods

### NewGenericWebhookTargetSelectionCriteria

`func NewGenericWebhookTargetSelectionCriteria() *GenericWebhookTargetSelectionCriteria`

NewGenericWebhookTargetSelectionCriteria instantiates a new GenericWebhookTargetSelectionCriteria object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewGenericWebhookTargetSelectionCriteriaWithDefaults

`func NewGenericWebhookTargetSelectionCriteriaWithDefaults() *GenericWebhookTargetSelectionCriteria`

NewGenericWebhookTargetSelectionCriteriaWithDefaults instantiates a new GenericWebhookTargetSelectionCriteria object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetIndexSelector

`func (o *GenericWebhookTargetSelectionCriteria) GetIndexSelector() IndexSelector`

GetIndexSelector returns the IndexSelector field if non-nil, zero value otherwise.

### GetIndexSelectorOk

`func (o *GenericWebhookTargetSelectionCriteria) GetIndexSelectorOk() (*IndexSelector, bool)`

GetIndexSelectorOk returns a tuple with the IndexSelector field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIndexSelector

`func (o *GenericWebhookTargetSelectionCriteria) SetIndexSelector(v IndexSelector)`

SetIndexSelector sets IndexSelector field to given value.

### HasIndexSelector

`func (o *GenericWebhookTargetSelectionCriteria) HasIndexSelector() bool`

HasIndexSelector returns a boolean if a field has been set.

### GetKind

`func (o *GenericWebhookTargetSelectionCriteria) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *GenericWebhookTargetSelectionCriteria) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *GenericWebhookTargetSelectionCriteria) SetKind(v string)`

SetKind sets Kind field to given value.

### HasKind

`func (o *GenericWebhookTargetSelectionCriteria) HasKind() bool`

HasKind returns a boolean if a field has been set.

### GetLabelSelector

`func (o *GenericWebhookTargetSelectionCriteria) GetLabelSelector() V1LabelSelector`

GetLabelSelector returns the LabelSelector field if non-nil, zero value otherwise.

### GetLabelSelectorOk

`func (o *GenericWebhookTargetSelectionCriteria) GetLabelSelectorOk() (*V1LabelSelector, bool)`

GetLabelSelectorOk returns a tuple with the LabelSelector field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLabelSelector

`func (o *GenericWebhookTargetSelectionCriteria) SetLabelSelector(v V1LabelSelector)`

SetLabelSelector sets LabelSelector field to given value.

### HasLabelSelector

`func (o *GenericWebhookTargetSelectionCriteria) HasLabelSelector() bool`

HasLabelSelector returns a boolean if a field has been set.

### GetName

`func (o *GenericWebhookTargetSelectionCriteria) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *GenericWebhookTargetSelectionCriteria) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *GenericWebhookTargetSelectionCriteria) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *GenericWebhookTargetSelectionCriteria) HasName() bool`

HasName returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


