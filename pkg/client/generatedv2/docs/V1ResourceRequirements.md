# V1ResourceRequirements

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Claims** | Pointer to [**[]V1ResourceClaim**](V1ResourceClaim.md) | Claims lists the names of resources, defined in spec.resourceClaims, that are used by this container.  This field depends on the DynamicResourceAllocation feature gate.  This field is immutable. It can only be set for containers.  +listType&#x3D;map +listMapKey&#x3D;name +featureGate&#x3D;DynamicResourceAllocation +optional | [optional] 
**Limits** | Pointer to **map[string]interface{}** | Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/ +optional | [optional] 
**Requests** | Pointer to **map[string]interface{}** | Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. Requests cannot exceed Limits. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/ +optional | [optional] 

## Methods

### NewV1ResourceRequirements

`func NewV1ResourceRequirements() *V1ResourceRequirements`

NewV1ResourceRequirements instantiates a new V1ResourceRequirements object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1ResourceRequirementsWithDefaults

`func NewV1ResourceRequirementsWithDefaults() *V1ResourceRequirements`

NewV1ResourceRequirementsWithDefaults instantiates a new V1ResourceRequirements object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetClaims

`func (o *V1ResourceRequirements) GetClaims() []V1ResourceClaim`

GetClaims returns the Claims field if non-nil, zero value otherwise.

### GetClaimsOk

`func (o *V1ResourceRequirements) GetClaimsOk() (*[]V1ResourceClaim, bool)`

GetClaimsOk returns a tuple with the Claims field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetClaims

`func (o *V1ResourceRequirements) SetClaims(v []V1ResourceClaim)`

SetClaims sets Claims field to given value.

### HasClaims

`func (o *V1ResourceRequirements) HasClaims() bool`

HasClaims returns a boolean if a field has been set.

### GetLimits

`func (o *V1ResourceRequirements) GetLimits() map[string]interface{}`

GetLimits returns the Limits field if non-nil, zero value otherwise.

### GetLimitsOk

`func (o *V1ResourceRequirements) GetLimitsOk() (*map[string]interface{}, bool)`

GetLimitsOk returns a tuple with the Limits field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLimits

`func (o *V1ResourceRequirements) SetLimits(v map[string]interface{})`

SetLimits sets Limits field to given value.

### HasLimits

`func (o *V1ResourceRequirements) HasLimits() bool`

HasLimits returns a boolean if a field has been set.

### GetRequests

`func (o *V1ResourceRequirements) GetRequests() map[string]interface{}`

GetRequests returns the Requests field if non-nil, zero value otherwise.

### GetRequestsOk

`func (o *V1ResourceRequirements) GetRequestsOk() (*map[string]interface{}, bool)`

GetRequestsOk returns a tuple with the Requests field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRequests

`func (o *V1ResourceRequirements) SetRequests(v map[string]interface{})`

SetRequests sets Requests field to given value.

### HasRequests

`func (o *V1ResourceRequirements) HasRequests() bool`

HasRequests returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


