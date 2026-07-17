# AnalysisRunArgument

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | **string** | Name is the name of the argument.  +kubebuilder:validation:Required | 
**Value** | **string** | Value is the value of the argument.  +kubebuilder:validation:Required | 

## Methods

### NewAnalysisRunArgument

`func NewAnalysisRunArgument(name string, value string, ) *AnalysisRunArgument`

NewAnalysisRunArgument instantiates a new AnalysisRunArgument object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAnalysisRunArgumentWithDefaults

`func NewAnalysisRunArgumentWithDefaults() *AnalysisRunArgument`

NewAnalysisRunArgumentWithDefaults instantiates a new AnalysisRunArgument object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetName

`func (o *AnalysisRunArgument) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *AnalysisRunArgument) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *AnalysisRunArgument) SetName(v string)`

SetName sets Name field to given value.


### GetValue

`func (o *AnalysisRunArgument) GetValue() string`

GetValue returns the Value field if non-nil, zero value otherwise.

### GetValueOk

`func (o *AnalysisRunArgument) GetValueOk() (*string, bool)`

GetValueOk returns a tuple with the Value field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetValue

`func (o *AnalysisRunArgument) SetValue(v string)`

SetValue sets Value field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


