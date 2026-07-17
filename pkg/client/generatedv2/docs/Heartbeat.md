# Heartbeat

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Controller** | Pointer to **string** | Controller is the name of the controller as reported in the Lease&#39;s kargoapi.LabelKeyController label. This may be empty if the controller that produced the heartbeat was unnamed. | [optional] 
**Status** | Pointer to [**HeartbeatStatus**](HeartbeatStatus.md) | Status is point-in-time liveness synthesized from a heartbeat record. | [optional] 
**Timestamp** | Pointer to **string** | Timestamp is the timestamp of the heartbeat. nil when the underlying record carried no parseable timestamp. | [optional] 

## Methods

### NewHeartbeat

`func NewHeartbeat() *Heartbeat`

NewHeartbeat instantiates a new Heartbeat object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewHeartbeatWithDefaults

`func NewHeartbeatWithDefaults() *Heartbeat`

NewHeartbeatWithDefaults instantiates a new Heartbeat object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetController

`func (o *Heartbeat) GetController() string`

GetController returns the Controller field if non-nil, zero value otherwise.

### GetControllerOk

`func (o *Heartbeat) GetControllerOk() (*string, bool)`

GetControllerOk returns a tuple with the Controller field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetController

`func (o *Heartbeat) SetController(v string)`

SetController sets Controller field to given value.

### HasController

`func (o *Heartbeat) HasController() bool`

HasController returns a boolean if a field has been set.

### GetStatus

`func (o *Heartbeat) GetStatus() HeartbeatStatus`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *Heartbeat) GetStatusOk() (*HeartbeatStatus, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *Heartbeat) SetStatus(v HeartbeatStatus)`

SetStatus sets Status field to given value.

### HasStatus

`func (o *Heartbeat) HasStatus() bool`

HasStatus returns a boolean if a field has been set.

### GetTimestamp

`func (o *Heartbeat) GetTimestamp() string`

GetTimestamp returns the Timestamp field if non-nil, zero value otherwise.

### GetTimestampOk

`func (o *Heartbeat) GetTimestampOk() (*string, bool)`

GetTimestampOk returns a tuple with the Timestamp field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTimestamp

`func (o *Heartbeat) SetTimestamp(v string)`

SetTimestamp sets Timestamp field to given value.

### HasTimestamp

`func (o *Heartbeat) HasTimestamp() bool`

HasTimestamp returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


