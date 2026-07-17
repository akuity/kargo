# FreightSources

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**AutoPromotionOptions** | Pointer to [**AutoPromotionOptions**](AutoPromotionOptions.md) | AutoPromotionOptions specifies options pertaining to auto-promotion. These settings have no effect if auto-promotion is not enabled for this Stage at the ProjectConfig level. | [optional] 
**AvailabilityStrategy** | Pointer to **string** | AvailabilityStrategy specifies the semantics for how requested Freight is made available to the Stage. This field is optional. When left unspecified, the field is implicitly treated as if its value were \&quot;OneOf\&quot;.  Accepted Values:  - \&quot;All\&quot;: Freight must be verified and, if applicable, soaked in all   upstream Stages to be considered available for promotion. - \&quot;OneOf\&quot;: Freight must be verified and, if applicable, soaked in at least    one upstream Stage to be considered available for promotion. - \&quot;\&quot;: Treated the same as \&quot;OneOf\&quot;.  +kubebuilder:validation:Optional | [optional] 
**Direct** | Pointer to **bool** | Direct indicates the requested Freight may be obtained directly from the Warehouse from which it originated. If this field&#39;s value is false, then the value of the Stages field must be non-empty. i.e. Between the two fields, at least one source must be specified. | [optional] 
**RequiredSoakTime** | Pointer to **string** | RequiredSoakTime specifies a minimum duration for which the requested Freight must have continuously occupied (\&quot;soaked in\&quot;) in an upstream Stage before becoming available for promotion to this Stage. This is an optional field. If nil or zero, no soak time is required. Any soak time requirement is in ADDITION to the requirement that Freight be verified in an upstream Stage to become available for promotion to this Stage, although a manual approval for promotion to this Stage will supersede any soak time requirement.  +kubebuilder:validation:Type&#x3D;string +kubebuilder:validation:Pattern&#x3D;&#x60;^([0-9]+(\\.[0-9]+)?(s|m|h))+$&#x60; +akuity:test-kubebuilder-pattern&#x3D;Duration | [optional] 
**Stages** | Pointer to **[]string** | Stages identifies other \&quot;upstream\&quot; Stages as potential sources of the requested Freight. If this field&#39;s value is empty, then the value of the Direct field must be true. i.e. Between the two fields, at least on source must be specified. | [optional] 

## Methods

### NewFreightSources

`func NewFreightSources() *FreightSources`

NewFreightSources instantiates a new FreightSources object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewFreightSourcesWithDefaults

`func NewFreightSourcesWithDefaults() *FreightSources`

NewFreightSourcesWithDefaults instantiates a new FreightSources object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAutoPromotionOptions

`func (o *FreightSources) GetAutoPromotionOptions() AutoPromotionOptions`

GetAutoPromotionOptions returns the AutoPromotionOptions field if non-nil, zero value otherwise.

### GetAutoPromotionOptionsOk

`func (o *FreightSources) GetAutoPromotionOptionsOk() (*AutoPromotionOptions, bool)`

GetAutoPromotionOptionsOk returns a tuple with the AutoPromotionOptions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAutoPromotionOptions

`func (o *FreightSources) SetAutoPromotionOptions(v AutoPromotionOptions)`

SetAutoPromotionOptions sets AutoPromotionOptions field to given value.

### HasAutoPromotionOptions

`func (o *FreightSources) HasAutoPromotionOptions() bool`

HasAutoPromotionOptions returns a boolean if a field has been set.

### GetAvailabilityStrategy

`func (o *FreightSources) GetAvailabilityStrategy() string`

GetAvailabilityStrategy returns the AvailabilityStrategy field if non-nil, zero value otherwise.

### GetAvailabilityStrategyOk

`func (o *FreightSources) GetAvailabilityStrategyOk() (*string, bool)`

GetAvailabilityStrategyOk returns a tuple with the AvailabilityStrategy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAvailabilityStrategy

`func (o *FreightSources) SetAvailabilityStrategy(v string)`

SetAvailabilityStrategy sets AvailabilityStrategy field to given value.

### HasAvailabilityStrategy

`func (o *FreightSources) HasAvailabilityStrategy() bool`

HasAvailabilityStrategy returns a boolean if a field has been set.

### GetDirect

`func (o *FreightSources) GetDirect() bool`

GetDirect returns the Direct field if non-nil, zero value otherwise.

### GetDirectOk

`func (o *FreightSources) GetDirectOk() (*bool, bool)`

GetDirectOk returns a tuple with the Direct field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDirect

`func (o *FreightSources) SetDirect(v bool)`

SetDirect sets Direct field to given value.

### HasDirect

`func (o *FreightSources) HasDirect() bool`

HasDirect returns a boolean if a field has been set.

### GetRequiredSoakTime

`func (o *FreightSources) GetRequiredSoakTime() string`

GetRequiredSoakTime returns the RequiredSoakTime field if non-nil, zero value otherwise.

### GetRequiredSoakTimeOk

`func (o *FreightSources) GetRequiredSoakTimeOk() (*string, bool)`

GetRequiredSoakTimeOk returns a tuple with the RequiredSoakTime field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRequiredSoakTime

`func (o *FreightSources) SetRequiredSoakTime(v string)`

SetRequiredSoakTime sets RequiredSoakTime field to given value.

### HasRequiredSoakTime

`func (o *FreightSources) HasRequiredSoakTime() bool`

HasRequiredSoakTime returns a boolean if a field has been set.

### GetStages

`func (o *FreightSources) GetStages() []string`

GetStages returns the Stages field if non-nil, zero value otherwise.

### GetStagesOk

`func (o *FreightSources) GetStagesOk() (*[]string, bool)`

GetStagesOk returns a tuple with the Stages field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStages

`func (o *FreightSources) SetStages(v []string)`

SetStages sets Stages field to given value.

### HasStages

`func (o *FreightSources) HasStages() bool`

HasStages returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


