# V1ContainerResizePolicy

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ResourceName** | Pointer to **string** | Name of the resource to which this resource resize policy applies. Supported values: cpu, memory. | [optional] 
**RestartPolicy** | Pointer to **string** | Restart policy to apply when specified resource is resized. If not specified, it defaults to NotRequired. | [optional] 

## Methods

### NewV1ContainerResizePolicy

`func NewV1ContainerResizePolicy() *V1ContainerResizePolicy`

NewV1ContainerResizePolicy instantiates a new V1ContainerResizePolicy object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1ContainerResizePolicyWithDefaults

`func NewV1ContainerResizePolicyWithDefaults() *V1ContainerResizePolicy`

NewV1ContainerResizePolicyWithDefaults instantiates a new V1ContainerResizePolicy object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetResourceName

`func (o *V1ContainerResizePolicy) GetResourceName() string`

GetResourceName returns the ResourceName field if non-nil, zero value otherwise.

### GetResourceNameOk

`func (o *V1ContainerResizePolicy) GetResourceNameOk() (*string, bool)`

GetResourceNameOk returns a tuple with the ResourceName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResourceName

`func (o *V1ContainerResizePolicy) SetResourceName(v string)`

SetResourceName sets ResourceName field to given value.

### HasResourceName

`func (o *V1ContainerResizePolicy) HasResourceName() bool`

HasResourceName returns a boolean if a field has been set.

### GetRestartPolicy

`func (o *V1ContainerResizePolicy) GetRestartPolicy() string`

GetRestartPolicy returns the RestartPolicy field if non-nil, zero value otherwise.

### GetRestartPolicyOk

`func (o *V1ContainerResizePolicy) GetRestartPolicyOk() (*string, bool)`

GetRestartPolicyOk returns a tuple with the RestartPolicy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRestartPolicy

`func (o *V1ContainerResizePolicy) SetRestartPolicy(v string)`

SetRestartPolicy sets RestartPolicy field to given value.

### HasRestartPolicy

`func (o *V1ContainerResizePolicy) HasRestartPolicy() bool`

HasRestartPolicy returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


