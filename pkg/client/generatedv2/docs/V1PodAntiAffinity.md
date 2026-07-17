# V1PodAntiAffinity

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**PreferredDuringSchedulingIgnoredDuringExecution** | Pointer to [**[]V1WeightedPodAffinityTerm**](V1WeightedPodAffinityTerm.md) | The scheduler will prefer to schedule pods to nodes that satisfy the anti-affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling anti-affinity expressions, etc.), compute a sum by iterating through the elements of this field and subtracting \&quot;weight\&quot; from the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred. +optional +listType&#x3D;atomic | [optional] 
**RequiredDuringSchedulingIgnoredDuringExecution** | Pointer to [**[]V1PodAffinityTerm**](V1PodAffinityTerm.md) | If the anti-affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the anti-affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied. +optional +listType&#x3D;atomic | [optional] 

## Methods

### NewV1PodAntiAffinity

`func NewV1PodAntiAffinity() *V1PodAntiAffinity`

NewV1PodAntiAffinity instantiates a new V1PodAntiAffinity object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1PodAntiAffinityWithDefaults

`func NewV1PodAntiAffinityWithDefaults() *V1PodAntiAffinity`

NewV1PodAntiAffinityWithDefaults instantiates a new V1PodAntiAffinity object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetPreferredDuringSchedulingIgnoredDuringExecution

`func (o *V1PodAntiAffinity) GetPreferredDuringSchedulingIgnoredDuringExecution() []V1WeightedPodAffinityTerm`

GetPreferredDuringSchedulingIgnoredDuringExecution returns the PreferredDuringSchedulingIgnoredDuringExecution field if non-nil, zero value otherwise.

### GetPreferredDuringSchedulingIgnoredDuringExecutionOk

`func (o *V1PodAntiAffinity) GetPreferredDuringSchedulingIgnoredDuringExecutionOk() (*[]V1WeightedPodAffinityTerm, bool)`

GetPreferredDuringSchedulingIgnoredDuringExecutionOk returns a tuple with the PreferredDuringSchedulingIgnoredDuringExecution field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPreferredDuringSchedulingIgnoredDuringExecution

`func (o *V1PodAntiAffinity) SetPreferredDuringSchedulingIgnoredDuringExecution(v []V1WeightedPodAffinityTerm)`

SetPreferredDuringSchedulingIgnoredDuringExecution sets PreferredDuringSchedulingIgnoredDuringExecution field to given value.

### HasPreferredDuringSchedulingIgnoredDuringExecution

`func (o *V1PodAntiAffinity) HasPreferredDuringSchedulingIgnoredDuringExecution() bool`

HasPreferredDuringSchedulingIgnoredDuringExecution returns a boolean if a field has been set.

### GetRequiredDuringSchedulingIgnoredDuringExecution

`func (o *V1PodAntiAffinity) GetRequiredDuringSchedulingIgnoredDuringExecution() []V1PodAffinityTerm`

GetRequiredDuringSchedulingIgnoredDuringExecution returns the RequiredDuringSchedulingIgnoredDuringExecution field if non-nil, zero value otherwise.

### GetRequiredDuringSchedulingIgnoredDuringExecutionOk

`func (o *V1PodAntiAffinity) GetRequiredDuringSchedulingIgnoredDuringExecutionOk() (*[]V1PodAffinityTerm, bool)`

GetRequiredDuringSchedulingIgnoredDuringExecutionOk returns a tuple with the RequiredDuringSchedulingIgnoredDuringExecution field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRequiredDuringSchedulingIgnoredDuringExecution

`func (o *V1PodAntiAffinity) SetRequiredDuringSchedulingIgnoredDuringExecution(v []V1PodAffinityTerm)`

SetRequiredDuringSchedulingIgnoredDuringExecution sets RequiredDuringSchedulingIgnoredDuringExecution field to given value.

### HasRequiredDuringSchedulingIgnoredDuringExecution

`func (o *V1PodAntiAffinity) HasRequiredDuringSchedulingIgnoredDuringExecution() bool`

HasRequiredDuringSchedulingIgnoredDuringExecution returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


