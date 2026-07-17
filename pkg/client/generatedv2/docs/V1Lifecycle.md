# V1Lifecycle

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**PostStart** | Pointer to [**V1LifecycleHandler**](V1LifecycleHandler.md) | PostStart is called immediately after a container is created. If the handler fails, the container is terminated and restarted according to its restart policy. Other management of the container blocks until the hook completes. More info: https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#container-hooks +optional | [optional] 
**PreStop** | Pointer to [**V1LifecycleHandler**](V1LifecycleHandler.md) | PreStop is called immediately before a container is terminated due to an API request or management event such as liveness/startup probe failure, preemption, resource contention, etc. The handler is not called if the container crashes or exits. The Pod&#39;s termination grace period countdown begins before the PreStop hook is executed. Regardless of the outcome of the handler, the container will eventually terminate within the Pod&#39;s termination grace period (unless delayed by finalizers). Other management of the container blocks until the hook completes or until the termination grace period is reached. More info: https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#container-hooks +optional | [optional] 
**StopSignal** | Pointer to **string** | StopSignal defines which signal will be sent to a container when it is being stopped. If not specified, the default is defined by the container runtime in use. StopSignal can only be set for Pods with a non-empty .spec.os.name +optional | [optional] 

## Methods

### NewV1Lifecycle

`func NewV1Lifecycle() *V1Lifecycle`

NewV1Lifecycle instantiates a new V1Lifecycle object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1LifecycleWithDefaults

`func NewV1LifecycleWithDefaults() *V1Lifecycle`

NewV1LifecycleWithDefaults instantiates a new V1Lifecycle object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetPostStart

`func (o *V1Lifecycle) GetPostStart() V1LifecycleHandler`

GetPostStart returns the PostStart field if non-nil, zero value otherwise.

### GetPostStartOk

`func (o *V1Lifecycle) GetPostStartOk() (*V1LifecycleHandler, bool)`

GetPostStartOk returns a tuple with the PostStart field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPostStart

`func (o *V1Lifecycle) SetPostStart(v V1LifecycleHandler)`

SetPostStart sets PostStart field to given value.

### HasPostStart

`func (o *V1Lifecycle) HasPostStart() bool`

HasPostStart returns a boolean if a field has been set.

### GetPreStop

`func (o *V1Lifecycle) GetPreStop() V1LifecycleHandler`

GetPreStop returns the PreStop field if non-nil, zero value otherwise.

### GetPreStopOk

`func (o *V1Lifecycle) GetPreStopOk() (*V1LifecycleHandler, bool)`

GetPreStopOk returns a tuple with the PreStop field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPreStop

`func (o *V1Lifecycle) SetPreStop(v V1LifecycleHandler)`

SetPreStop sets PreStop field to given value.

### HasPreStop

`func (o *V1Lifecycle) HasPreStop() bool`

HasPreStop returns a boolean if a field has been set.

### GetStopSignal

`func (o *V1Lifecycle) GetStopSignal() string`

GetStopSignal returns the StopSignal field if non-nil, zero value otherwise.

### GetStopSignalOk

`func (o *V1Lifecycle) GetStopSignalOk() (*string, bool)`

GetStopSignalOk returns a tuple with the StopSignal field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStopSignal

`func (o *V1Lifecycle) SetStopSignal(v string)`

SetStopSignal sets StopSignal field to given value.

### HasStopSignal

`func (o *V1Lifecycle) HasStopSignal() bool`

HasStopSignal returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


