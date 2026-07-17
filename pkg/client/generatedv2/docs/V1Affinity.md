# V1Affinity

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**NodeAffinity** | Pointer to [**V1NodeAffinity**](V1NodeAffinity.md) | Describes node affinity scheduling rules for the pod. +optional | [optional] 
**PodAffinity** | Pointer to [**V1PodAffinity**](V1PodAffinity.md) | Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)). +optional | [optional] 
**PodAntiAffinity** | Pointer to [**V1PodAntiAffinity**](V1PodAntiAffinity.md) | Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)). +optional | [optional] 

## Methods

### NewV1Affinity

`func NewV1Affinity() *V1Affinity`

NewV1Affinity instantiates a new V1Affinity object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1AffinityWithDefaults

`func NewV1AffinityWithDefaults() *V1Affinity`

NewV1AffinityWithDefaults instantiates a new V1Affinity object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetNodeAffinity

`func (o *V1Affinity) GetNodeAffinity() V1NodeAffinity`

GetNodeAffinity returns the NodeAffinity field if non-nil, zero value otherwise.

### GetNodeAffinityOk

`func (o *V1Affinity) GetNodeAffinityOk() (*V1NodeAffinity, bool)`

GetNodeAffinityOk returns a tuple with the NodeAffinity field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNodeAffinity

`func (o *V1Affinity) SetNodeAffinity(v V1NodeAffinity)`

SetNodeAffinity sets NodeAffinity field to given value.

### HasNodeAffinity

`func (o *V1Affinity) HasNodeAffinity() bool`

HasNodeAffinity returns a boolean if a field has been set.

### GetPodAffinity

`func (o *V1Affinity) GetPodAffinity() V1PodAffinity`

GetPodAffinity returns the PodAffinity field if non-nil, zero value otherwise.

### GetPodAffinityOk

`func (o *V1Affinity) GetPodAffinityOk() (*V1PodAffinity, bool)`

GetPodAffinityOk returns a tuple with the PodAffinity field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPodAffinity

`func (o *V1Affinity) SetPodAffinity(v V1PodAffinity)`

SetPodAffinity sets PodAffinity field to given value.

### HasPodAffinity

`func (o *V1Affinity) HasPodAffinity() bool`

HasPodAffinity returns a boolean if a field has been set.

### GetPodAntiAffinity

`func (o *V1Affinity) GetPodAntiAffinity() V1PodAntiAffinity`

GetPodAntiAffinity returns the PodAntiAffinity field if non-nil, zero value otherwise.

### GetPodAntiAffinityOk

`func (o *V1Affinity) GetPodAntiAffinityOk() (*V1PodAntiAffinity, bool)`

GetPodAntiAffinityOk returns a tuple with the PodAntiAffinity field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPodAntiAffinity

`func (o *V1Affinity) SetPodAntiAffinity(v V1PodAntiAffinity)`

SetPodAntiAffinity sets PodAntiAffinity field to given value.

### HasPodAntiAffinity

`func (o *V1Affinity) HasPodAntiAffinity() bool`

HasPodAntiAffinity returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


