# V1ContainerPort

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ContainerPort** | Pointer to **int32** | Number of port to expose on the pod&#39;s IP address. This must be a valid port number, 0 &lt; x &lt; 65536. | [optional] 
**HostIP** | Pointer to **string** | What host IP to bind the external port to. +optional | [optional] 
**HostPort** | Pointer to **int32** | Number of port to expose on the host. If specified, this must be a valid port number, 0 &lt; x &lt; 65536. If HostNetwork is specified, this must match ContainerPort. Most containers do not need this. +optional | [optional] 
**Name** | Pointer to **string** | If specified, this must be an IANA_SVC_NAME and unique within the pod. Each named port in a pod must have a unique name. Name for the port that can be referred to by services. +optional | [optional] 
**Protocol** | Pointer to **string** | Protocol for port. Must be UDP, TCP, or SCTP. Defaults to \&quot;TCP\&quot;. +optional +default&#x3D;\&quot;TCP\&quot; | [optional] 

## Methods

### NewV1ContainerPort

`func NewV1ContainerPort() *V1ContainerPort`

NewV1ContainerPort instantiates a new V1ContainerPort object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1ContainerPortWithDefaults

`func NewV1ContainerPortWithDefaults() *V1ContainerPort`

NewV1ContainerPortWithDefaults instantiates a new V1ContainerPort object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetContainerPort

`func (o *V1ContainerPort) GetContainerPort() int32`

GetContainerPort returns the ContainerPort field if non-nil, zero value otherwise.

### GetContainerPortOk

`func (o *V1ContainerPort) GetContainerPortOk() (*int32, bool)`

GetContainerPortOk returns a tuple with the ContainerPort field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetContainerPort

`func (o *V1ContainerPort) SetContainerPort(v int32)`

SetContainerPort sets ContainerPort field to given value.

### HasContainerPort

`func (o *V1ContainerPort) HasContainerPort() bool`

HasContainerPort returns a boolean if a field has been set.

### GetHostIP

`func (o *V1ContainerPort) GetHostIP() string`

GetHostIP returns the HostIP field if non-nil, zero value otherwise.

### GetHostIPOk

`func (o *V1ContainerPort) GetHostIPOk() (*string, bool)`

GetHostIPOk returns a tuple with the HostIP field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHostIP

`func (o *V1ContainerPort) SetHostIP(v string)`

SetHostIP sets HostIP field to given value.

### HasHostIP

`func (o *V1ContainerPort) HasHostIP() bool`

HasHostIP returns a boolean if a field has been set.

### GetHostPort

`func (o *V1ContainerPort) GetHostPort() int32`

GetHostPort returns the HostPort field if non-nil, zero value otherwise.

### GetHostPortOk

`func (o *V1ContainerPort) GetHostPortOk() (*int32, bool)`

GetHostPortOk returns a tuple with the HostPort field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHostPort

`func (o *V1ContainerPort) SetHostPort(v int32)`

SetHostPort sets HostPort field to given value.

### HasHostPort

`func (o *V1ContainerPort) HasHostPort() bool`

HasHostPort returns a boolean if a field has been set.

### GetName

`func (o *V1ContainerPort) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *V1ContainerPort) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *V1ContainerPort) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *V1ContainerPort) HasName() bool`

HasName returns a boolean if a field has been set.

### GetProtocol

`func (o *V1ContainerPort) GetProtocol() string`

GetProtocol returns the Protocol field if non-nil, zero value otherwise.

### GetProtocolOk

`func (o *V1ContainerPort) GetProtocolOk() (*string, bool)`

GetProtocolOk returns a tuple with the Protocol field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProtocol

`func (o *V1ContainerPort) SetProtocol(v string)`

SetProtocol sets Protocol field to given value.

### HasProtocol

`func (o *V1ContainerPort) HasProtocol() bool`

HasProtocol returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


