# AnalysisTemplateReference

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Kind** | Pointer to **string** | Kind is the type of the AnalysisTemplate. Can be either AnalysisTemplate or ClusterAnalysisTemplate, default is AnalysisTemplate.  +kubebuilder:validation:Optional +kubebuilder:validation:Enum&#x3D;AnalysisTemplate;ClusterAnalysisTemplate | [optional] 
**Name** | **string** | Name is the name of the AnalysisTemplate in the same project/namespace as the Stage.  +kubebuilder:validation:Required | 

## Methods

### NewAnalysisTemplateReference

`func NewAnalysisTemplateReference(name string, ) *AnalysisTemplateReference`

NewAnalysisTemplateReference instantiates a new AnalysisTemplateReference object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAnalysisTemplateReferenceWithDefaults

`func NewAnalysisTemplateReferenceWithDefaults() *AnalysisTemplateReference`

NewAnalysisTemplateReferenceWithDefaults instantiates a new AnalysisTemplateReference object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetKind

`func (o *AnalysisTemplateReference) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *AnalysisTemplateReference) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *AnalysisTemplateReference) SetKind(v string)`

SetKind sets Kind field to given value.

### HasKind

`func (o *AnalysisTemplateReference) HasKind() bool`

HasKind returns a boolean if a field has been set.

### GetName

`func (o *AnalysisTemplateReference) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *AnalysisTemplateReference) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *AnalysisTemplateReference) SetName(v string)`

SetName sets Name field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


