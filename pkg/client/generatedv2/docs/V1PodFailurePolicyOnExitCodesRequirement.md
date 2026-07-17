# V1PodFailurePolicyOnExitCodesRequirement

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ContainerName** | Pointer to **string** | Restricts the check for exit codes to the container with the specified name. When null, the rule applies to all containers. When specified, it should match one the container or initContainer names in the pod template. +optional | [optional] 
**Operator** | Pointer to [**V1PodFailurePolicyOnExitCodesOperator**](V1PodFailurePolicyOnExitCodesOperator.md) | Represents the relationship between the container exit code(s) and the specified values. Containers completed with success (exit code 0) are excluded from the requirement check. Possible values are:  - In: the requirement is satisfied if at least one container exit code   (might be multiple if there are multiple containers not restricted   by the &#39;containerName&#39; field) is in the set of specified values. - NotIn: the requirement is satisfied if at least one container exit code   (might be multiple if there are multiple containers not restricted   by the &#39;containerName&#39; field) is not in the set of specified values. Additional values are considered to be added in the future. Clients should react to an unknown operator by assuming the requirement is not satisfied. | [optional] 
**Values** | Pointer to **[]int32** | Specifies the set of values. Each returned container exit code (might be multiple in case of multiple containers) is checked against this set of values with respect to the operator. The list of values must be ordered and must not contain duplicates. Value &#39;0&#39; cannot be used for the In operator. At least one element is required. At most 255 elements are allowed. +listType&#x3D;set | [optional] 

## Methods

### NewV1PodFailurePolicyOnExitCodesRequirement

`func NewV1PodFailurePolicyOnExitCodesRequirement() *V1PodFailurePolicyOnExitCodesRequirement`

NewV1PodFailurePolicyOnExitCodesRequirement instantiates a new V1PodFailurePolicyOnExitCodesRequirement object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1PodFailurePolicyOnExitCodesRequirementWithDefaults

`func NewV1PodFailurePolicyOnExitCodesRequirementWithDefaults() *V1PodFailurePolicyOnExitCodesRequirement`

NewV1PodFailurePolicyOnExitCodesRequirementWithDefaults instantiates a new V1PodFailurePolicyOnExitCodesRequirement object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetContainerName

`func (o *V1PodFailurePolicyOnExitCodesRequirement) GetContainerName() string`

GetContainerName returns the ContainerName field if non-nil, zero value otherwise.

### GetContainerNameOk

`func (o *V1PodFailurePolicyOnExitCodesRequirement) GetContainerNameOk() (*string, bool)`

GetContainerNameOk returns a tuple with the ContainerName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetContainerName

`func (o *V1PodFailurePolicyOnExitCodesRequirement) SetContainerName(v string)`

SetContainerName sets ContainerName field to given value.

### HasContainerName

`func (o *V1PodFailurePolicyOnExitCodesRequirement) HasContainerName() bool`

HasContainerName returns a boolean if a field has been set.

### GetOperator

`func (o *V1PodFailurePolicyOnExitCodesRequirement) GetOperator() V1PodFailurePolicyOnExitCodesOperator`

GetOperator returns the Operator field if non-nil, zero value otherwise.

### GetOperatorOk

`func (o *V1PodFailurePolicyOnExitCodesRequirement) GetOperatorOk() (*V1PodFailurePolicyOnExitCodesOperator, bool)`

GetOperatorOk returns a tuple with the Operator field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOperator

`func (o *V1PodFailurePolicyOnExitCodesRequirement) SetOperator(v V1PodFailurePolicyOnExitCodesOperator)`

SetOperator sets Operator field to given value.

### HasOperator

`func (o *V1PodFailurePolicyOnExitCodesRequirement) HasOperator() bool`

HasOperator returns a boolean if a field has been set.

### GetValues

`func (o *V1PodFailurePolicyOnExitCodesRequirement) GetValues() []int32`

GetValues returns the Values field if non-nil, zero value otherwise.

### GetValuesOk

`func (o *V1PodFailurePolicyOnExitCodesRequirement) GetValuesOk() (*[]int32, bool)`

GetValuesOk returns a tuple with the Values field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetValues

`func (o *V1PodFailurePolicyOnExitCodesRequirement) SetValues(v []int32)`

SetValues sets Values field to given value.

### HasValues

`func (o *V1PodFailurePolicyOnExitCodesRequirement) HasValues() bool`

HasValues returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


