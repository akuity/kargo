# V1PodAffinityTerm

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**LabelSelector** | Pointer to [**V1LabelSelector**](V1LabelSelector.md) | A label query over a set of resources, in this case pods. If it&#39;s null, this PodAffinityTerm matches with no Pods. +optional | [optional] 
**MatchLabelKeys** | Pointer to **[]string** | MatchLabelKeys is a set of pod label keys to select which pods will be taken into consideration. The keys are used to lookup values from the incoming pod labels, those key-value labels are merged with &#x60;labelSelector&#x60; as &#x60;key in (value)&#x60; to select the group of existing pods which pods will be taken into consideration for the incoming pod&#39;s pod (anti) affinity. Keys that don&#39;t exist in the incoming pod labels will be ignored. The default value is empty. The same key is forbidden to exist in both matchLabelKeys and labelSelector. Also, matchLabelKeys cannot be set when labelSelector isn&#39;t set.  +listType&#x3D;atomic +optional | [optional] 
**MismatchLabelKeys** | Pointer to **[]string** | MismatchLabelKeys is a set of pod label keys to select which pods will be taken into consideration. The keys are used to lookup values from the incoming pod labels, those key-value labels are merged with &#x60;labelSelector&#x60; as &#x60;key notin (value)&#x60; to select the group of existing pods which pods will be taken into consideration for the incoming pod&#39;s pod (anti) affinity. Keys that don&#39;t exist in the incoming pod labels will be ignored. The default value is empty. The same key is forbidden to exist in both mismatchLabelKeys and labelSelector. Also, mismatchLabelKeys cannot be set when labelSelector isn&#39;t set.  +listType&#x3D;atomic +optional | [optional] 
**NamespaceSelector** | Pointer to [**V1LabelSelector**](V1LabelSelector.md) | A label query over the set of namespaces that the term applies to. The term is applied to the union of the namespaces selected by this field and the ones listed in the namespaces field. null selector and null or empty namespaces list means \&quot;this pod&#39;s namespace\&quot;. An empty selector ({}) matches all namespaces. +optional | [optional] 
**Namespaces** | Pointer to **[]string** | namespaces specifies a static list of namespace names that the term applies to. The term is applied to the union of the namespaces listed in this field and the ones selected by namespaceSelector. null or empty namespaces list and null namespaceSelector means \&quot;this pod&#39;s namespace\&quot;. +optional +listType&#x3D;atomic | [optional] 
**TopologyKey** | Pointer to **string** | This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed. | [optional] 

## Methods

### NewV1PodAffinityTerm

`func NewV1PodAffinityTerm() *V1PodAffinityTerm`

NewV1PodAffinityTerm instantiates a new V1PodAffinityTerm object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1PodAffinityTermWithDefaults

`func NewV1PodAffinityTermWithDefaults() *V1PodAffinityTerm`

NewV1PodAffinityTermWithDefaults instantiates a new V1PodAffinityTerm object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetLabelSelector

`func (o *V1PodAffinityTerm) GetLabelSelector() V1LabelSelector`

GetLabelSelector returns the LabelSelector field if non-nil, zero value otherwise.

### GetLabelSelectorOk

`func (o *V1PodAffinityTerm) GetLabelSelectorOk() (*V1LabelSelector, bool)`

GetLabelSelectorOk returns a tuple with the LabelSelector field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLabelSelector

`func (o *V1PodAffinityTerm) SetLabelSelector(v V1LabelSelector)`

SetLabelSelector sets LabelSelector field to given value.

### HasLabelSelector

`func (o *V1PodAffinityTerm) HasLabelSelector() bool`

HasLabelSelector returns a boolean if a field has been set.

### GetMatchLabelKeys

`func (o *V1PodAffinityTerm) GetMatchLabelKeys() []string`

GetMatchLabelKeys returns the MatchLabelKeys field if non-nil, zero value otherwise.

### GetMatchLabelKeysOk

`func (o *V1PodAffinityTerm) GetMatchLabelKeysOk() (*[]string, bool)`

GetMatchLabelKeysOk returns a tuple with the MatchLabelKeys field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMatchLabelKeys

`func (o *V1PodAffinityTerm) SetMatchLabelKeys(v []string)`

SetMatchLabelKeys sets MatchLabelKeys field to given value.

### HasMatchLabelKeys

`func (o *V1PodAffinityTerm) HasMatchLabelKeys() bool`

HasMatchLabelKeys returns a boolean if a field has been set.

### GetMismatchLabelKeys

`func (o *V1PodAffinityTerm) GetMismatchLabelKeys() []string`

GetMismatchLabelKeys returns the MismatchLabelKeys field if non-nil, zero value otherwise.

### GetMismatchLabelKeysOk

`func (o *V1PodAffinityTerm) GetMismatchLabelKeysOk() (*[]string, bool)`

GetMismatchLabelKeysOk returns a tuple with the MismatchLabelKeys field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMismatchLabelKeys

`func (o *V1PodAffinityTerm) SetMismatchLabelKeys(v []string)`

SetMismatchLabelKeys sets MismatchLabelKeys field to given value.

### HasMismatchLabelKeys

`func (o *V1PodAffinityTerm) HasMismatchLabelKeys() bool`

HasMismatchLabelKeys returns a boolean if a field has been set.

### GetNamespaceSelector

`func (o *V1PodAffinityTerm) GetNamespaceSelector() V1LabelSelector`

GetNamespaceSelector returns the NamespaceSelector field if non-nil, zero value otherwise.

### GetNamespaceSelectorOk

`func (o *V1PodAffinityTerm) GetNamespaceSelectorOk() (*V1LabelSelector, bool)`

GetNamespaceSelectorOk returns a tuple with the NamespaceSelector field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNamespaceSelector

`func (o *V1PodAffinityTerm) SetNamespaceSelector(v V1LabelSelector)`

SetNamespaceSelector sets NamespaceSelector field to given value.

### HasNamespaceSelector

`func (o *V1PodAffinityTerm) HasNamespaceSelector() bool`

HasNamespaceSelector returns a boolean if a field has been set.

### GetNamespaces

`func (o *V1PodAffinityTerm) GetNamespaces() []string`

GetNamespaces returns the Namespaces field if non-nil, zero value otherwise.

### GetNamespacesOk

`func (o *V1PodAffinityTerm) GetNamespacesOk() (*[]string, bool)`

GetNamespacesOk returns a tuple with the Namespaces field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNamespaces

`func (o *V1PodAffinityTerm) SetNamespaces(v []string)`

SetNamespaces sets Namespaces field to given value.

### HasNamespaces

`func (o *V1PodAffinityTerm) HasNamespaces() bool`

HasNamespaces returns a boolean if a field has been set.

### GetTopologyKey

`func (o *V1PodAffinityTerm) GetTopologyKey() string`

GetTopologyKey returns the TopologyKey field if non-nil, zero value otherwise.

### GetTopologyKeyOk

`func (o *V1PodAffinityTerm) GetTopologyKeyOk() (*string, bool)`

GetTopologyKeyOk returns a tuple with the TopologyKey field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTopologyKey

`func (o *V1PodAffinityTerm) SetTopologyKey(v string)`

SetTopologyKey sets TopologyKey field to given value.

### HasTopologyKey

`func (o *V1PodAffinityTerm) HasTopologyKey() bool`

HasTopologyKey returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


