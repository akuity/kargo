# RolloutsMetric

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ConsecutiveErrorLimit** | Pointer to **interface{}** | Serializes as a bare integer or string | [optional] 
**ConsecutiveSuccessLimit** | Pointer to **interface{}** | Serializes as a bare integer or string | [optional] 
**Count** | Pointer to **interface{}** | Serializes as a bare integer or string | [optional] 
**FailureCondition** | Pointer to **string** |  | [optional] 
**FailureLimit** | Pointer to **interface{}** | Serializes as a bare integer or string | [optional] 
**InconclusiveLimit** | Pointer to **interface{}** | Serializes as a bare integer or string | [optional] 
**InitialDelay** | Pointer to **string** |  | [optional] 
**Interval** | Pointer to **string** |  | [optional] 
**Name** | Pointer to **string** |  | [optional] 
**Provider** | Pointer to [**RolloutsMetricProvider**](RolloutsMetricProvider.md) |  | [optional] 
**SuccessCondition** | Pointer to **string** |  | [optional] 

## Methods

### NewRolloutsMetric

`func NewRolloutsMetric() *RolloutsMetric`

NewRolloutsMetric instantiates a new RolloutsMetric object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRolloutsMetricWithDefaults

`func NewRolloutsMetricWithDefaults() *RolloutsMetric`

NewRolloutsMetricWithDefaults instantiates a new RolloutsMetric object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetConsecutiveErrorLimit

`func (o *RolloutsMetric) GetConsecutiveErrorLimit() interface{}`

GetConsecutiveErrorLimit returns the ConsecutiveErrorLimit field if non-nil, zero value otherwise.

### GetConsecutiveErrorLimitOk

`func (o *RolloutsMetric) GetConsecutiveErrorLimitOk() (*interface{}, bool)`

GetConsecutiveErrorLimitOk returns a tuple with the ConsecutiveErrorLimit field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConsecutiveErrorLimit

`func (o *RolloutsMetric) SetConsecutiveErrorLimit(v interface{})`

SetConsecutiveErrorLimit sets ConsecutiveErrorLimit field to given value.

### HasConsecutiveErrorLimit

`func (o *RolloutsMetric) HasConsecutiveErrorLimit() bool`

HasConsecutiveErrorLimit returns a boolean if a field has been set.

### GetConsecutiveSuccessLimit

`func (o *RolloutsMetric) GetConsecutiveSuccessLimit() interface{}`

GetConsecutiveSuccessLimit returns the ConsecutiveSuccessLimit field if non-nil, zero value otherwise.

### GetConsecutiveSuccessLimitOk

`func (o *RolloutsMetric) GetConsecutiveSuccessLimitOk() (*interface{}, bool)`

GetConsecutiveSuccessLimitOk returns a tuple with the ConsecutiveSuccessLimit field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConsecutiveSuccessLimit

`func (o *RolloutsMetric) SetConsecutiveSuccessLimit(v interface{})`

SetConsecutiveSuccessLimit sets ConsecutiveSuccessLimit field to given value.

### HasConsecutiveSuccessLimit

`func (o *RolloutsMetric) HasConsecutiveSuccessLimit() bool`

HasConsecutiveSuccessLimit returns a boolean if a field has been set.

### GetCount

`func (o *RolloutsMetric) GetCount() interface{}`

GetCount returns the Count field if non-nil, zero value otherwise.

### GetCountOk

`func (o *RolloutsMetric) GetCountOk() (*interface{}, bool)`

GetCountOk returns a tuple with the Count field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCount

`func (o *RolloutsMetric) SetCount(v interface{})`

SetCount sets Count field to given value.

### HasCount

`func (o *RolloutsMetric) HasCount() bool`

HasCount returns a boolean if a field has been set.

### GetFailureCondition

`func (o *RolloutsMetric) GetFailureCondition() string`

GetFailureCondition returns the FailureCondition field if non-nil, zero value otherwise.

### GetFailureConditionOk

`func (o *RolloutsMetric) GetFailureConditionOk() (*string, bool)`

GetFailureConditionOk returns a tuple with the FailureCondition field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFailureCondition

`func (o *RolloutsMetric) SetFailureCondition(v string)`

SetFailureCondition sets FailureCondition field to given value.

### HasFailureCondition

`func (o *RolloutsMetric) HasFailureCondition() bool`

HasFailureCondition returns a boolean if a field has been set.

### GetFailureLimit

`func (o *RolloutsMetric) GetFailureLimit() interface{}`

GetFailureLimit returns the FailureLimit field if non-nil, zero value otherwise.

### GetFailureLimitOk

`func (o *RolloutsMetric) GetFailureLimitOk() (*interface{}, bool)`

GetFailureLimitOk returns a tuple with the FailureLimit field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFailureLimit

`func (o *RolloutsMetric) SetFailureLimit(v interface{})`

SetFailureLimit sets FailureLimit field to given value.

### HasFailureLimit

`func (o *RolloutsMetric) HasFailureLimit() bool`

HasFailureLimit returns a boolean if a field has been set.

### GetInconclusiveLimit

`func (o *RolloutsMetric) GetInconclusiveLimit() interface{}`

GetInconclusiveLimit returns the InconclusiveLimit field if non-nil, zero value otherwise.

### GetInconclusiveLimitOk

`func (o *RolloutsMetric) GetInconclusiveLimitOk() (*interface{}, bool)`

GetInconclusiveLimitOk returns a tuple with the InconclusiveLimit field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInconclusiveLimit

`func (o *RolloutsMetric) SetInconclusiveLimit(v interface{})`

SetInconclusiveLimit sets InconclusiveLimit field to given value.

### HasInconclusiveLimit

`func (o *RolloutsMetric) HasInconclusiveLimit() bool`

HasInconclusiveLimit returns a boolean if a field has been set.

### GetInitialDelay

`func (o *RolloutsMetric) GetInitialDelay() string`

GetInitialDelay returns the InitialDelay field if non-nil, zero value otherwise.

### GetInitialDelayOk

`func (o *RolloutsMetric) GetInitialDelayOk() (*string, bool)`

GetInitialDelayOk returns a tuple with the InitialDelay field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInitialDelay

`func (o *RolloutsMetric) SetInitialDelay(v string)`

SetInitialDelay sets InitialDelay field to given value.

### HasInitialDelay

`func (o *RolloutsMetric) HasInitialDelay() bool`

HasInitialDelay returns a boolean if a field has been set.

### GetInterval

`func (o *RolloutsMetric) GetInterval() string`

GetInterval returns the Interval field if non-nil, zero value otherwise.

### GetIntervalOk

`func (o *RolloutsMetric) GetIntervalOk() (*string, bool)`

GetIntervalOk returns a tuple with the Interval field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInterval

`func (o *RolloutsMetric) SetInterval(v string)`

SetInterval sets Interval field to given value.

### HasInterval

`func (o *RolloutsMetric) HasInterval() bool`

HasInterval returns a boolean if a field has been set.

### GetName

`func (o *RolloutsMetric) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *RolloutsMetric) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *RolloutsMetric) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *RolloutsMetric) HasName() bool`

HasName returns a boolean if a field has been set.

### GetProvider

`func (o *RolloutsMetric) GetProvider() RolloutsMetricProvider`

GetProvider returns the Provider field if non-nil, zero value otherwise.

### GetProviderOk

`func (o *RolloutsMetric) GetProviderOk() (*RolloutsMetricProvider, bool)`

GetProviderOk returns a tuple with the Provider field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProvider

`func (o *RolloutsMetric) SetProvider(v RolloutsMetricProvider)`

SetProvider sets Provider field to given value.

### HasProvider

`func (o *RolloutsMetric) HasProvider() bool`

HasProvider returns a boolean if a field has been set.

### GetSuccessCondition

`func (o *RolloutsMetric) GetSuccessCondition() string`

GetSuccessCondition returns the SuccessCondition field if non-nil, zero value otherwise.

### GetSuccessConditionOk

`func (o *RolloutsMetric) GetSuccessConditionOk() (*string, bool)`

GetSuccessConditionOk returns a tuple with the SuccessCondition field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSuccessCondition

`func (o *RolloutsMetric) SetSuccessCondition(v string)`

SetSuccessCondition sets SuccessCondition field to given value.

### HasSuccessCondition

`func (o *RolloutsMetric) HasSuccessCondition() bool`

HasSuccessCondition returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


