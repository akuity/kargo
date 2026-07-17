# HealthCheckStep

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Config** | Pointer to **interface{}** | Config is the configuration for the directive. | [optional] 
**Uses** | Pointer to **string** | Uses identifies a runner that can execute this step.  +kubebuilder:validation:MinLength&#x3D;1 | [optional] 

## Methods

### NewHealthCheckStep

`func NewHealthCheckStep() *HealthCheckStep`

NewHealthCheckStep instantiates a new HealthCheckStep object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewHealthCheckStepWithDefaults

`func NewHealthCheckStepWithDefaults() *HealthCheckStep`

NewHealthCheckStepWithDefaults instantiates a new HealthCheckStep object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetConfig

`func (o *HealthCheckStep) GetConfig() interface{}`

GetConfig returns the Config field if non-nil, zero value otherwise.

### GetConfigOk

`func (o *HealthCheckStep) GetConfigOk() (*interface{}, bool)`

GetConfigOk returns a tuple with the Config field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConfig

`func (o *HealthCheckStep) SetConfig(v interface{})`

SetConfig sets Config field to given value.

### HasConfig

`func (o *HealthCheckStep) HasConfig() bool`

HasConfig returns a boolean if a field has been set.

### GetUses

`func (o *HealthCheckStep) GetUses() string`

GetUses returns the Uses field if non-nil, zero value otherwise.

### GetUsesOk

`func (o *HealthCheckStep) GetUsesOk() (*string, bool)`

GetUsesOk returns a tuple with the Uses field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUses

`func (o *HealthCheckStep) SetUses(v string)`

SetUses sets Uses field to given value.

### HasUses

`func (o *HealthCheckStep) HasUses() bool`

HasUses returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


