# V1GRPCAction

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Port** | Pointer to **int32** | Port number of the gRPC service. Number must be in the range 1 to 65535. | [optional] 
**Service** | Pointer to **string** | Service is the name of the service to place in the gRPC HealthCheckRequest (see https://github.com/grpc/grpc/blob/master/doc/health-checking.md).  If this is not specified, the default behavior is defined by gRPC. +optional +default&#x3D;\&quot;\&quot; | [optional] 

## Methods

### NewV1GRPCAction

`func NewV1GRPCAction() *V1GRPCAction`

NewV1GRPCAction instantiates a new V1GRPCAction object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1GRPCActionWithDefaults

`func NewV1GRPCActionWithDefaults() *V1GRPCAction`

NewV1GRPCActionWithDefaults instantiates a new V1GRPCAction object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetPort

`func (o *V1GRPCAction) GetPort() int32`

GetPort returns the Port field if non-nil, zero value otherwise.

### GetPortOk

`func (o *V1GRPCAction) GetPortOk() (*int32, bool)`

GetPortOk returns a tuple with the Port field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPort

`func (o *V1GRPCAction) SetPort(v int32)`

SetPort sets Port field to given value.

### HasPort

`func (o *V1GRPCAction) HasPort() bool`

HasPort returns a boolean if a field has been set.

### GetService

`func (o *V1GRPCAction) GetService() string`

GetService returns the Service field if non-nil, zero value otherwise.

### GetServiceOk

`func (o *V1GRPCAction) GetServiceOk() (*string, bool)`

GetServiceOk returns a tuple with the Service field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetService

`func (o *V1GRPCAction) SetService(v string)`

SetService sets Service field to given value.

### HasService

`func (o *V1GRPCAction) HasService() bool`

HasService returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


