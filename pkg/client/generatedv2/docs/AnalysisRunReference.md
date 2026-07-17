# AnalysisRunReference

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | Pointer to **string** | Name is the name of the AnalysisRun. | [optional] 
**Namespace** | Pointer to **string** | Namespace is the namespace of the AnalysisRun. | [optional] 
**Phase** | Pointer to **string** | Phase is the last observed phase of the AnalysisRun referenced by Name. | [optional] 

## Methods

### NewAnalysisRunReference

`func NewAnalysisRunReference() *AnalysisRunReference`

NewAnalysisRunReference instantiates a new AnalysisRunReference object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAnalysisRunReferenceWithDefaults

`func NewAnalysisRunReferenceWithDefaults() *AnalysisRunReference`

NewAnalysisRunReferenceWithDefaults instantiates a new AnalysisRunReference object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetName

`func (o *AnalysisRunReference) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *AnalysisRunReference) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *AnalysisRunReference) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *AnalysisRunReference) HasName() bool`

HasName returns a boolean if a field has been set.

### GetNamespace

`func (o *AnalysisRunReference) GetNamespace() string`

GetNamespace returns the Namespace field if non-nil, zero value otherwise.

### GetNamespaceOk

`func (o *AnalysisRunReference) GetNamespaceOk() (*string, bool)`

GetNamespaceOk returns a tuple with the Namespace field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNamespace

`func (o *AnalysisRunReference) SetNamespace(v string)`

SetNamespace sets Namespace field to given value.

### HasNamespace

`func (o *AnalysisRunReference) HasNamespace() bool`

HasNamespace returns a boolean if a field has been set.

### GetPhase

`func (o *AnalysisRunReference) GetPhase() string`

GetPhase returns the Phase field if non-nil, zero value otherwise.

### GetPhaseOk

`func (o *AnalysisRunReference) GetPhaseOk() (*string, bool)`

GetPhaseOk returns a tuple with the Phase field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPhase

`func (o *AnalysisRunReference) SetPhase(v string)`

SetPhase sets Phase field to given value.

### HasPhase

`func (o *AnalysisRunReference) HasPhase() bool`

HasPhase returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


