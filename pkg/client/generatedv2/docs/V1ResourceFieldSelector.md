# V1ResourceFieldSelector

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ContainerName** | Pointer to **string** | Container name: required for volumes, optional for env vars +optional | [optional] 
**Divisor** | Pointer to **interface{}** | Specifies the output format of the exposed resources, defaults to \&quot;1\&quot; +optional | [optional] 
**Resource** | Pointer to **string** | Required: resource to select | [optional] 

## Methods

### NewV1ResourceFieldSelector

`func NewV1ResourceFieldSelector() *V1ResourceFieldSelector`

NewV1ResourceFieldSelector instantiates a new V1ResourceFieldSelector object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1ResourceFieldSelectorWithDefaults

`func NewV1ResourceFieldSelectorWithDefaults() *V1ResourceFieldSelector`

NewV1ResourceFieldSelectorWithDefaults instantiates a new V1ResourceFieldSelector object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetContainerName

`func (o *V1ResourceFieldSelector) GetContainerName() string`

GetContainerName returns the ContainerName field if non-nil, zero value otherwise.

### GetContainerNameOk

`func (o *V1ResourceFieldSelector) GetContainerNameOk() (*string, bool)`

GetContainerNameOk returns a tuple with the ContainerName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetContainerName

`func (o *V1ResourceFieldSelector) SetContainerName(v string)`

SetContainerName sets ContainerName field to given value.

### HasContainerName

`func (o *V1ResourceFieldSelector) HasContainerName() bool`

HasContainerName returns a boolean if a field has been set.

### GetDivisor

`func (o *V1ResourceFieldSelector) GetDivisor() interface{}`

GetDivisor returns the Divisor field if non-nil, zero value otherwise.

### GetDivisorOk

`func (o *V1ResourceFieldSelector) GetDivisorOk() (*interface{}, bool)`

GetDivisorOk returns a tuple with the Divisor field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDivisor

`func (o *V1ResourceFieldSelector) SetDivisor(v interface{})`

SetDivisor sets Divisor field to given value.

### HasDivisor

`func (o *V1ResourceFieldSelector) HasDivisor() bool`

HasDivisor returns a boolean if a field has been set.

### GetResource

`func (o *V1ResourceFieldSelector) GetResource() string`

GetResource returns the Resource field if non-nil, zero value otherwise.

### GetResourceOk

`func (o *V1ResourceFieldSelector) GetResourceOk() (*string, bool)`

GetResourceOk returns a tuple with the Resource field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResource

`func (o *V1ResourceFieldSelector) SetResource(v string)`

SetResource sets Resource field to given value.

### HasResource

`func (o *V1ResourceFieldSelector) HasResource() bool`

HasResource returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


