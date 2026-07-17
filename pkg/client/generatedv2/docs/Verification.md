# Verification

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**AnalysisRunMetadata** | Pointer to [**AnalysisRunMetadata**](AnalysisRunMetadata.md) | AnalysisRunMetadata contains optional metadata that should be applied to all AnalysisRuns. | [optional] 
**AnalysisTemplates** | Pointer to [**[]AnalysisTemplateReference**](AnalysisTemplateReference.md) | AnalysisTemplates is a list of AnalysisTemplates from which AnalysisRuns should be created to verify a Stage&#39;s current Freight is fit to be promoted downstream. | [optional] 
**Args** | Pointer to [**[]AnalysisRunArgument**](AnalysisRunArgument.md) | Args lists arguments that should be added to all AnalysisRuns. | [optional] 

## Methods

### NewVerification

`func NewVerification() *Verification`

NewVerification instantiates a new Verification object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewVerificationWithDefaults

`func NewVerificationWithDefaults() *Verification`

NewVerificationWithDefaults instantiates a new Verification object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAnalysisRunMetadata

`func (o *Verification) GetAnalysisRunMetadata() AnalysisRunMetadata`

GetAnalysisRunMetadata returns the AnalysisRunMetadata field if non-nil, zero value otherwise.

### GetAnalysisRunMetadataOk

`func (o *Verification) GetAnalysisRunMetadataOk() (*AnalysisRunMetadata, bool)`

GetAnalysisRunMetadataOk returns a tuple with the AnalysisRunMetadata field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAnalysisRunMetadata

`func (o *Verification) SetAnalysisRunMetadata(v AnalysisRunMetadata)`

SetAnalysisRunMetadata sets AnalysisRunMetadata field to given value.

### HasAnalysisRunMetadata

`func (o *Verification) HasAnalysisRunMetadata() bool`

HasAnalysisRunMetadata returns a boolean if a field has been set.

### GetAnalysisTemplates

`func (o *Verification) GetAnalysisTemplates() []AnalysisTemplateReference`

GetAnalysisTemplates returns the AnalysisTemplates field if non-nil, zero value otherwise.

### GetAnalysisTemplatesOk

`func (o *Verification) GetAnalysisTemplatesOk() (*[]AnalysisTemplateReference, bool)`

GetAnalysisTemplatesOk returns a tuple with the AnalysisTemplates field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAnalysisTemplates

`func (o *Verification) SetAnalysisTemplates(v []AnalysisTemplateReference)`

SetAnalysisTemplates sets AnalysisTemplates field to given value.

### HasAnalysisTemplates

`func (o *Verification) HasAnalysisTemplates() bool`

HasAnalysisTemplates returns a boolean if a field has been set.

### GetArgs

`func (o *Verification) GetArgs() []AnalysisRunArgument`

GetArgs returns the Args field if non-nil, zero value otherwise.

### GetArgsOk

`func (o *Verification) GetArgsOk() (*[]AnalysisRunArgument, bool)`

GetArgsOk returns a tuple with the Args field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetArgs

`func (o *Verification) SetArgs(v []AnalysisRunArgument)`

SetArgs sets Args field to given value.

### HasArgs

`func (o *Verification) HasArgs() bool`

HasArgs returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


