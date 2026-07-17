# V1PodResourceClaim

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | Pointer to **string** | Name uniquely identifies this resource claim inside the pod. This must be a DNS_LABEL. | [optional] 
**ResourceClaimName** | Pointer to **string** | ResourceClaimName is the name of a ResourceClaim object in the same namespace as this pod.  Exactly one of ResourceClaimName and ResourceClaimTemplateName must be set. | [optional] 
**ResourceClaimTemplateName** | Pointer to **string** | ResourceClaimTemplateName is the name of a ResourceClaimTemplate object in the same namespace as this pod.  The template will be used to create a new ResourceClaim, which will be bound to this pod. When this pod is deleted, the ResourceClaim will also be deleted. The pod name and resource name, along with a generated component, will be used to form a unique name for the ResourceClaim, which will be recorded in pod.status.resourceClaimStatuses.  This field is immutable and no changes will be made to the corresponding ResourceClaim by the control plane after creating the ResourceClaim.  Exactly one of ResourceClaimName and ResourceClaimTemplateName must be set. | [optional] 

## Methods

### NewV1PodResourceClaim

`func NewV1PodResourceClaim() *V1PodResourceClaim`

NewV1PodResourceClaim instantiates a new V1PodResourceClaim object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1PodResourceClaimWithDefaults

`func NewV1PodResourceClaimWithDefaults() *V1PodResourceClaim`

NewV1PodResourceClaimWithDefaults instantiates a new V1PodResourceClaim object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetName

`func (o *V1PodResourceClaim) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *V1PodResourceClaim) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *V1PodResourceClaim) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *V1PodResourceClaim) HasName() bool`

HasName returns a boolean if a field has been set.

### GetResourceClaimName

`func (o *V1PodResourceClaim) GetResourceClaimName() string`

GetResourceClaimName returns the ResourceClaimName field if non-nil, zero value otherwise.

### GetResourceClaimNameOk

`func (o *V1PodResourceClaim) GetResourceClaimNameOk() (*string, bool)`

GetResourceClaimNameOk returns a tuple with the ResourceClaimName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResourceClaimName

`func (o *V1PodResourceClaim) SetResourceClaimName(v string)`

SetResourceClaimName sets ResourceClaimName field to given value.

### HasResourceClaimName

`func (o *V1PodResourceClaim) HasResourceClaimName() bool`

HasResourceClaimName returns a boolean if a field has been set.

### GetResourceClaimTemplateName

`func (o *V1PodResourceClaim) GetResourceClaimTemplateName() string`

GetResourceClaimTemplateName returns the ResourceClaimTemplateName field if non-nil, zero value otherwise.

### GetResourceClaimTemplateNameOk

`func (o *V1PodResourceClaim) GetResourceClaimTemplateNameOk() (*string, bool)`

GetResourceClaimTemplateNameOk returns a tuple with the ResourceClaimTemplateName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResourceClaimTemplateName

`func (o *V1PodResourceClaim) SetResourceClaimTemplateName(v string)`

SetResourceClaimTemplateName sets ResourceClaimTemplateName field to given value.

### HasResourceClaimTemplateName

`func (o *V1PodResourceClaim) HasResourceClaimTemplateName() bool`

HasResourceClaimTemplateName returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


