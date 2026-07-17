# V1PreferredSchedulingTerm

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Preference** | Pointer to [**V1NodeSelectorTerm**](V1NodeSelectorTerm.md) | A node selector term, associated with the corresponding weight. | [optional] 
**Weight** | Pointer to **int32** | Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100. | [optional] 

## Methods

### NewV1PreferredSchedulingTerm

`func NewV1PreferredSchedulingTerm() *V1PreferredSchedulingTerm`

NewV1PreferredSchedulingTerm instantiates a new V1PreferredSchedulingTerm object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1PreferredSchedulingTermWithDefaults

`func NewV1PreferredSchedulingTermWithDefaults() *V1PreferredSchedulingTerm`

NewV1PreferredSchedulingTermWithDefaults instantiates a new V1PreferredSchedulingTerm object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetPreference

`func (o *V1PreferredSchedulingTerm) GetPreference() V1NodeSelectorTerm`

GetPreference returns the Preference field if non-nil, zero value otherwise.

### GetPreferenceOk

`func (o *V1PreferredSchedulingTerm) GetPreferenceOk() (*V1NodeSelectorTerm, bool)`

GetPreferenceOk returns a tuple with the Preference field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPreference

`func (o *V1PreferredSchedulingTerm) SetPreference(v V1NodeSelectorTerm)`

SetPreference sets Preference field to given value.

### HasPreference

`func (o *V1PreferredSchedulingTerm) HasPreference() bool`

HasPreference returns a boolean if a field has been set.

### GetWeight

`func (o *V1PreferredSchedulingTerm) GetWeight() int32`

GetWeight returns the Weight field if non-nil, zero value otherwise.

### GetWeightOk

`func (o *V1PreferredSchedulingTerm) GetWeightOk() (*int32, bool)`

GetWeightOk returns a tuple with the Weight field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWeight

`func (o *V1PreferredSchedulingTerm) SetWeight(v int32)`

SetWeight sets Weight field to given value.

### HasWeight

`func (o *V1PreferredSchedulingTerm) HasWeight() bool`

HasWeight returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


