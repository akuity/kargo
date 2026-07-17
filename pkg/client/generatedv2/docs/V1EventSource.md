# V1EventSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Component** | Pointer to **string** | Component from which the event is generated. +optional | [optional] 
**Host** | Pointer to **string** | Node name on which the event is generated. +optional | [optional] 

## Methods

### NewV1EventSource

`func NewV1EventSource() *V1EventSource`

NewV1EventSource instantiates a new V1EventSource object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1EventSourceWithDefaults

`func NewV1EventSourceWithDefaults() *V1EventSource`

NewV1EventSourceWithDefaults instantiates a new V1EventSource object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetComponent

`func (o *V1EventSource) GetComponent() string`

GetComponent returns the Component field if non-nil, zero value otherwise.

### GetComponentOk

`func (o *V1EventSource) GetComponentOk() (*string, bool)`

GetComponentOk returns a tuple with the Component field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetComponent

`func (o *V1EventSource) SetComponent(v string)`

SetComponent sets Component field to given value.

### HasComponent

`func (o *V1EventSource) HasComponent() bool`

HasComponent returns a boolean if a field has been set.

### GetHost

`func (o *V1EventSource) GetHost() string`

GetHost returns the Host field if non-nil, zero value otherwise.

### GetHostOk

`func (o *V1EventSource) GetHostOk() (*string, bool)`

GetHostOk returns a tuple with the Host field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHost

`func (o *V1EventSource) SetHost(v string)`

SetHost sets Host field to given value.

### HasHost

`func (o *V1EventSource) HasHost() bool`

HasHost returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


