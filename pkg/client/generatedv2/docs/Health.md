# Health

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Config** | Pointer to **interface{}** | Config is the opaque configuration of all health checks performed on this Stage. | [optional] 
**Issues** | Pointer to **[]string** | Issues clarifies why a Stage in any state other than Healthy is in that state. This field will always be the empty when a Stage is Healthy. | [optional] 
**Output** | Pointer to **interface{}** | Output is the opaque output of all health checks performed on this Stage. | [optional] 
**Status** | Pointer to **string** | Status describes the health of the Stage. | [optional] 

## Methods

### NewHealth

`func NewHealth() *Health`

NewHealth instantiates a new Health object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewHealthWithDefaults

`func NewHealthWithDefaults() *Health`

NewHealthWithDefaults instantiates a new Health object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetConfig

`func (o *Health) GetConfig() interface{}`

GetConfig returns the Config field if non-nil, zero value otherwise.

### GetConfigOk

`func (o *Health) GetConfigOk() (*interface{}, bool)`

GetConfigOk returns a tuple with the Config field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConfig

`func (o *Health) SetConfig(v interface{})`

SetConfig sets Config field to given value.

### HasConfig

`func (o *Health) HasConfig() bool`

HasConfig returns a boolean if a field has been set.

### GetIssues

`func (o *Health) GetIssues() []string`

GetIssues returns the Issues field if non-nil, zero value otherwise.

### GetIssuesOk

`func (o *Health) GetIssuesOk() (*[]string, bool)`

GetIssuesOk returns a tuple with the Issues field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIssues

`func (o *Health) SetIssues(v []string)`

SetIssues sets Issues field to given value.

### HasIssues

`func (o *Health) HasIssues() bool`

HasIssues returns a boolean if a field has been set.

### GetOutput

`func (o *Health) GetOutput() interface{}`

GetOutput returns the Output field if non-nil, zero value otherwise.

### GetOutputOk

`func (o *Health) GetOutputOk() (*interface{}, bool)`

GetOutputOk returns a tuple with the Output field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOutput

`func (o *Health) SetOutput(v interface{})`

SetOutput sets Output field to given value.

### HasOutput

`func (o *Health) HasOutput() bool`

HasOutput returns a boolean if a field has been set.

### GetStatus

`func (o *Health) GetStatus() string`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *Health) GetStatusOk() (*string, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *Health) SetStatus(v string)`

SetStatus sets Status field to given value.

### HasStatus

`func (o *Health) HasStatus() bool`

HasStatus returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


