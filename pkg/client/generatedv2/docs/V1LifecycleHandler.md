# V1LifecycleHandler

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Exec** | Pointer to [**V1ExecAction**](V1ExecAction.md) | Exec specifies a command to execute in the container. +optional | [optional] 
**HttpGet** | Pointer to [**V1HTTPGetAction**](V1HTTPGetAction.md) | HTTPGet specifies an HTTP GET request to perform. +optional | [optional] 
**Sleep** | Pointer to [**V1SleepAction**](V1SleepAction.md) | Sleep represents a duration that the container should sleep. +featureGate&#x3D;PodLifecycleSleepAction +optional | [optional] 
**TcpSocket** | Pointer to [**V1TCPSocketAction**](V1TCPSocketAction.md) | Deprecated. TCPSocket is NOT supported as a LifecycleHandler and kept for backward compatibility. There is no validation of this field and lifecycle hooks will fail at runtime when it is specified. +optional | [optional] 

## Methods

### NewV1LifecycleHandler

`func NewV1LifecycleHandler() *V1LifecycleHandler`

NewV1LifecycleHandler instantiates a new V1LifecycleHandler object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1LifecycleHandlerWithDefaults

`func NewV1LifecycleHandlerWithDefaults() *V1LifecycleHandler`

NewV1LifecycleHandlerWithDefaults instantiates a new V1LifecycleHandler object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetExec

`func (o *V1LifecycleHandler) GetExec() V1ExecAction`

GetExec returns the Exec field if non-nil, zero value otherwise.

### GetExecOk

`func (o *V1LifecycleHandler) GetExecOk() (*V1ExecAction, bool)`

GetExecOk returns a tuple with the Exec field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExec

`func (o *V1LifecycleHandler) SetExec(v V1ExecAction)`

SetExec sets Exec field to given value.

### HasExec

`func (o *V1LifecycleHandler) HasExec() bool`

HasExec returns a boolean if a field has been set.

### GetHttpGet

`func (o *V1LifecycleHandler) GetHttpGet() V1HTTPGetAction`

GetHttpGet returns the HttpGet field if non-nil, zero value otherwise.

### GetHttpGetOk

`func (o *V1LifecycleHandler) GetHttpGetOk() (*V1HTTPGetAction, bool)`

GetHttpGetOk returns a tuple with the HttpGet field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHttpGet

`func (o *V1LifecycleHandler) SetHttpGet(v V1HTTPGetAction)`

SetHttpGet sets HttpGet field to given value.

### HasHttpGet

`func (o *V1LifecycleHandler) HasHttpGet() bool`

HasHttpGet returns a boolean if a field has been set.

### GetSleep

`func (o *V1LifecycleHandler) GetSleep() V1SleepAction`

GetSleep returns the Sleep field if non-nil, zero value otherwise.

### GetSleepOk

`func (o *V1LifecycleHandler) GetSleepOk() (*V1SleepAction, bool)`

GetSleepOk returns a tuple with the Sleep field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSleep

`func (o *V1LifecycleHandler) SetSleep(v V1SleepAction)`

SetSleep sets Sleep field to given value.

### HasSleep

`func (o *V1LifecycleHandler) HasSleep() bool`

HasSleep returns a boolean if a field has been set.

### GetTcpSocket

`func (o *V1LifecycleHandler) GetTcpSocket() V1TCPSocketAction`

GetTcpSocket returns the TcpSocket field if non-nil, zero value otherwise.

### GetTcpSocketOk

`func (o *V1LifecycleHandler) GetTcpSocketOk() (*V1TCPSocketAction, bool)`

GetTcpSocketOk returns a tuple with the TcpSocket field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTcpSocket

`func (o *V1LifecycleHandler) SetTcpSocket(v V1TCPSocketAction)`

SetTcpSocket sets TcpSocket field to given value.

### HasTcpSocket

`func (o *V1LifecycleHandler) HasTcpSocket() bool`

HasTcpSocket returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


