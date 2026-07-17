# V1EnvVar

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | Pointer to **string** | Name of the environment variable. May consist of any printable ASCII characters except &#39;&#x3D;&#39;. | [optional] 
**Value** | Pointer to **string** | Variable references $(VAR_NAME) are expanded using the previously defined environment variables in the container and any service environment variables. If a variable cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. \&quot;$$(VAR_NAME)\&quot; will produce the string literal \&quot;$(VAR_NAME)\&quot;. Escaped references will never be expanded, regardless of whether the variable exists or not. Defaults to \&quot;\&quot;. +optional | [optional] 
**ValueFrom** | Pointer to [**V1EnvVarSource**](V1EnvVarSource.md) | Source for the environment variable&#39;s value. Cannot be used if value is not empty. +optional | [optional] 

## Methods

### NewV1EnvVar

`func NewV1EnvVar() *V1EnvVar`

NewV1EnvVar instantiates a new V1EnvVar object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1EnvVarWithDefaults

`func NewV1EnvVarWithDefaults() *V1EnvVar`

NewV1EnvVarWithDefaults instantiates a new V1EnvVar object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetName

`func (o *V1EnvVar) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *V1EnvVar) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *V1EnvVar) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *V1EnvVar) HasName() bool`

HasName returns a boolean if a field has been set.

### GetValue

`func (o *V1EnvVar) GetValue() string`

GetValue returns the Value field if non-nil, zero value otherwise.

### GetValueOk

`func (o *V1EnvVar) GetValueOk() (*string, bool)`

GetValueOk returns a tuple with the Value field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetValue

`func (o *V1EnvVar) SetValue(v string)`

SetValue sets Value field to given value.

### HasValue

`func (o *V1EnvVar) HasValue() bool`

HasValue returns a boolean if a field has been set.

### GetValueFrom

`func (o *V1EnvVar) GetValueFrom() V1EnvVarSource`

GetValueFrom returns the ValueFrom field if non-nil, zero value otherwise.

### GetValueFromOk

`func (o *V1EnvVar) GetValueFromOk() (*V1EnvVarSource, bool)`

GetValueFromOk returns a tuple with the ValueFrom field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetValueFrom

`func (o *V1EnvVar) SetValueFrom(v V1EnvVarSource)`

SetValueFrom sets ValueFrom field to given value.

### HasValueFrom

`func (o *V1EnvVar) HasValueFrom() bool`

HasValueFrom returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


