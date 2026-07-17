# PromotionTaskReference

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Kind** | Pointer to **string** | Kind is the type of the PromotionTask. Can be either PromotionTask or ClusterPromotionTask, default is PromotionTask.  +kubebuilder:validation:Optional +kubebuilder:validation:Enum&#x3D;PromotionTask;ClusterPromotionTask | [optional] 
**Name** | **string** | Name is the name of the (Cluster)PromotionTask.  +kubebuilder:validation:Required +kubebuilder:validation:MinLength&#x3D;1 +kubebuilder:validation:MaxLength&#x3D;253 +kubebuilder:validation:Pattern&#x3D;&#x60;^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$&#x60; +akuity:test-kubebuilder-pattern&#x3D;KubernetesName | 

## Methods

### NewPromotionTaskReference

`func NewPromotionTaskReference(name string, ) *PromotionTaskReference`

NewPromotionTaskReference instantiates a new PromotionTaskReference object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPromotionTaskReferenceWithDefaults

`func NewPromotionTaskReferenceWithDefaults() *PromotionTaskReference`

NewPromotionTaskReferenceWithDefaults instantiates a new PromotionTaskReference object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetKind

`func (o *PromotionTaskReference) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *PromotionTaskReference) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *PromotionTaskReference) SetKind(v string)`

SetKind sets Kind field to given value.

### HasKind

`func (o *PromotionTaskReference) HasKind() bool`

HasKind returns a boolean if a field has been set.

### GetName

`func (o *PromotionTaskReference) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *PromotionTaskReference) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *PromotionTaskReference) SetName(v string)`

SetName sets Name field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


