# V1TopologySpreadConstraint

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**LabelSelector** | Pointer to [**V1LabelSelector**](V1LabelSelector.md) | LabelSelector is used to find matching pods. Pods that match this label selector are counted to determine the number of pods in their corresponding topology domain. +optional | [optional] 
**MatchLabelKeys** | Pointer to **[]string** | MatchLabelKeys is a set of pod label keys to select the pods over which spreading will be calculated. The keys are used to lookup values from the incoming pod labels, those key-value labels are ANDed with labelSelector to select the group of existing pods over which spreading will be calculated for the incoming pod. The same key is forbidden to exist in both MatchLabelKeys and LabelSelector. MatchLabelKeys cannot be set when LabelSelector isn&#39;t set. Keys that don&#39;t exist in the incoming pod labels will be ignored. A null or empty list means only match against labelSelector.  This is a beta field and requires the MatchLabelKeysInPodTopologySpread feature gate to be enabled (enabled by default). +listType&#x3D;atomic +optional | [optional] 
**MaxSkew** | Pointer to **int32** | MaxSkew describes the degree to which pods may be unevenly distributed. When &#x60;whenUnsatisfiable&#x3D;DoNotSchedule&#x60;, it is the maximum permitted difference between the number of matching pods in the target topology and the global minimum. The global minimum is the minimum number of matching pods in an eligible domain or zero if the number of eligible domains is less than MinDomains. For example, in a 3-zone cluster, MaxSkew is set to 1, and pods with the same labelSelector spread as 2/2/1: In this case, the global minimum is 1. +-------+-------+-------+ | zone1 | zone2 | zone3 | +-------+-------+-------+ |  P P  |  P P  |   P   | +-------+-------+-------+ - if MaxSkew is 1, incoming pod can only be scheduled to zone3 to become 2/2/2; scheduling it onto zone1(zone2) would make the ActualSkew(3-1) on zone1(zone2) violate MaxSkew(1). - if MaxSkew is 2, incoming pod can be scheduled onto any zone. When &#x60;whenUnsatisfiable&#x3D;ScheduleAnyway&#x60;, it is used to give higher precedence to topologies that satisfy it. It&#39;s a required field. Default value is 1 and 0 is not allowed. | [optional] 
**MinDomains** | Pointer to **int32** | MinDomains indicates a minimum number of eligible domains. When the number of eligible domains with matching topology keys is less than minDomains, Pod Topology Spread treats \&quot;global minimum\&quot; as 0, and then the calculation of Skew is performed. And when the number of eligible domains with matching topology keys equals or greater than minDomains, this value has no effect on scheduling. As a result, when the number of eligible domains is less than minDomains, scheduler won&#39;t schedule more than maxSkew Pods to those domains. If value is nil, the constraint behaves as if MinDomains is equal to 1. Valid values are integers greater than 0. When value is not nil, WhenUnsatisfiable must be DoNotSchedule.  For example, in a 3-zone cluster, MaxSkew is set to 2, MinDomains is set to 5 and pods with the same labelSelector spread as 2/2/2: +-------+-------+-------+ | zone1 | zone2 | zone3 | +-------+-------+-------+ |  P P  |  P P  |  P P  | +-------+-------+-------+ The number of domains is less than 5(MinDomains), so \&quot;global minimum\&quot; is treated as 0. In this situation, new pod with the same labelSelector cannot be scheduled, because computed skew will be 3(3 - 0) if new Pod is scheduled to any of the three zones, it will violate MaxSkew. +optional | [optional] 
**NodeAffinityPolicy** | Pointer to **string** | NodeAffinityPolicy indicates how we will treat Pod&#39;s nodeAffinity/nodeSelector when calculating pod topology spread skew. Options are: - Honor: only nodes matching nodeAffinity/nodeSelector are included in the calculations. - Ignore: nodeAffinity/nodeSelector are ignored. All nodes are included in the calculations.  If this value is nil, the behavior is equivalent to the Honor policy. +optional | [optional] 
**NodeTaintsPolicy** | Pointer to **string** | NodeTaintsPolicy indicates how we will treat node taints when calculating pod topology spread skew. Options are: - Honor: nodes without taints, along with tainted nodes for which the incoming pod has a toleration, are included. - Ignore: node taints are ignored. All nodes are included.  If this value is nil, the behavior is equivalent to the Ignore policy. +optional | [optional] 
**TopologyKey** | Pointer to **string** | TopologyKey is the key of node labels. Nodes that have a label with this key and identical values are considered to be in the same topology. We consider each &lt;key, value&gt; as a \&quot;bucket\&quot;, and try to put balanced number of pods into each bucket. We define a domain as a particular instance of a topology. Also, we define an eligible domain as a domain whose nodes meet the requirements of nodeAffinityPolicy and nodeTaintsPolicy. e.g. If TopologyKey is \&quot;kubernetes.io/hostname\&quot;, each Node is a domain of that topology. And, if TopologyKey is \&quot;topology.kubernetes.io/zone\&quot;, each zone is a domain of that topology. It&#39;s a required field. | [optional] 
**WhenUnsatisfiable** | Pointer to **string** | WhenUnsatisfiable indicates how to deal with a pod if it doesn&#39;t satisfy the spread constraint. - DoNotSchedule (default) tells the scheduler not to schedule it. - ScheduleAnyway tells the scheduler to schedule the pod in any location,   but giving higher precedence to topologies that would help reduce the   skew. A constraint is considered \&quot;Unsatisfiable\&quot; for an incoming pod if and only if every possible node assignment for that pod would violate \&quot;MaxSkew\&quot; on some topology. For example, in a 3-zone cluster, MaxSkew is set to 1, and pods with the same labelSelector spread as 3/1/1: +-------+-------+-------+ | zone1 | zone2 | zone3 | +-------+-------+-------+ | P P P |   P   |   P   | +-------+-------+-------+ If WhenUnsatisfiable is set to DoNotSchedule, incoming pod can only be scheduled to zone2(zone3) to become 3/2/1(3/1/2) as ActualSkew(2-1) on zone2(zone3) satisfies MaxSkew(1). In other words, the cluster can still be imbalanced, but scheduler won&#39;t make it *more* imbalanced. It&#39;s a required field. | [optional] 

## Methods

### NewV1TopologySpreadConstraint

`func NewV1TopologySpreadConstraint() *V1TopologySpreadConstraint`

NewV1TopologySpreadConstraint instantiates a new V1TopologySpreadConstraint object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1TopologySpreadConstraintWithDefaults

`func NewV1TopologySpreadConstraintWithDefaults() *V1TopologySpreadConstraint`

NewV1TopologySpreadConstraintWithDefaults instantiates a new V1TopologySpreadConstraint object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetLabelSelector

`func (o *V1TopologySpreadConstraint) GetLabelSelector() V1LabelSelector`

GetLabelSelector returns the LabelSelector field if non-nil, zero value otherwise.

### GetLabelSelectorOk

`func (o *V1TopologySpreadConstraint) GetLabelSelectorOk() (*V1LabelSelector, bool)`

GetLabelSelectorOk returns a tuple with the LabelSelector field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLabelSelector

`func (o *V1TopologySpreadConstraint) SetLabelSelector(v V1LabelSelector)`

SetLabelSelector sets LabelSelector field to given value.

### HasLabelSelector

`func (o *V1TopologySpreadConstraint) HasLabelSelector() bool`

HasLabelSelector returns a boolean if a field has been set.

### GetMatchLabelKeys

`func (o *V1TopologySpreadConstraint) GetMatchLabelKeys() []string`

GetMatchLabelKeys returns the MatchLabelKeys field if non-nil, zero value otherwise.

### GetMatchLabelKeysOk

`func (o *V1TopologySpreadConstraint) GetMatchLabelKeysOk() (*[]string, bool)`

GetMatchLabelKeysOk returns a tuple with the MatchLabelKeys field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMatchLabelKeys

`func (o *V1TopologySpreadConstraint) SetMatchLabelKeys(v []string)`

SetMatchLabelKeys sets MatchLabelKeys field to given value.

### HasMatchLabelKeys

`func (o *V1TopologySpreadConstraint) HasMatchLabelKeys() bool`

HasMatchLabelKeys returns a boolean if a field has been set.

### GetMaxSkew

`func (o *V1TopologySpreadConstraint) GetMaxSkew() int32`

GetMaxSkew returns the MaxSkew field if non-nil, zero value otherwise.

### GetMaxSkewOk

`func (o *V1TopologySpreadConstraint) GetMaxSkewOk() (*int32, bool)`

GetMaxSkewOk returns a tuple with the MaxSkew field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMaxSkew

`func (o *V1TopologySpreadConstraint) SetMaxSkew(v int32)`

SetMaxSkew sets MaxSkew field to given value.

### HasMaxSkew

`func (o *V1TopologySpreadConstraint) HasMaxSkew() bool`

HasMaxSkew returns a boolean if a field has been set.

### GetMinDomains

`func (o *V1TopologySpreadConstraint) GetMinDomains() int32`

GetMinDomains returns the MinDomains field if non-nil, zero value otherwise.

### GetMinDomainsOk

`func (o *V1TopologySpreadConstraint) GetMinDomainsOk() (*int32, bool)`

GetMinDomainsOk returns a tuple with the MinDomains field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMinDomains

`func (o *V1TopologySpreadConstraint) SetMinDomains(v int32)`

SetMinDomains sets MinDomains field to given value.

### HasMinDomains

`func (o *V1TopologySpreadConstraint) HasMinDomains() bool`

HasMinDomains returns a boolean if a field has been set.

### GetNodeAffinityPolicy

`func (o *V1TopologySpreadConstraint) GetNodeAffinityPolicy() string`

GetNodeAffinityPolicy returns the NodeAffinityPolicy field if non-nil, zero value otherwise.

### GetNodeAffinityPolicyOk

`func (o *V1TopologySpreadConstraint) GetNodeAffinityPolicyOk() (*string, bool)`

GetNodeAffinityPolicyOk returns a tuple with the NodeAffinityPolicy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNodeAffinityPolicy

`func (o *V1TopologySpreadConstraint) SetNodeAffinityPolicy(v string)`

SetNodeAffinityPolicy sets NodeAffinityPolicy field to given value.

### HasNodeAffinityPolicy

`func (o *V1TopologySpreadConstraint) HasNodeAffinityPolicy() bool`

HasNodeAffinityPolicy returns a boolean if a field has been set.

### GetNodeTaintsPolicy

`func (o *V1TopologySpreadConstraint) GetNodeTaintsPolicy() string`

GetNodeTaintsPolicy returns the NodeTaintsPolicy field if non-nil, zero value otherwise.

### GetNodeTaintsPolicyOk

`func (o *V1TopologySpreadConstraint) GetNodeTaintsPolicyOk() (*string, bool)`

GetNodeTaintsPolicyOk returns a tuple with the NodeTaintsPolicy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNodeTaintsPolicy

`func (o *V1TopologySpreadConstraint) SetNodeTaintsPolicy(v string)`

SetNodeTaintsPolicy sets NodeTaintsPolicy field to given value.

### HasNodeTaintsPolicy

`func (o *V1TopologySpreadConstraint) HasNodeTaintsPolicy() bool`

HasNodeTaintsPolicy returns a boolean if a field has been set.

### GetTopologyKey

`func (o *V1TopologySpreadConstraint) GetTopologyKey() string`

GetTopologyKey returns the TopologyKey field if non-nil, zero value otherwise.

### GetTopologyKeyOk

`func (o *V1TopologySpreadConstraint) GetTopologyKeyOk() (*string, bool)`

GetTopologyKeyOk returns a tuple with the TopologyKey field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTopologyKey

`func (o *V1TopologySpreadConstraint) SetTopologyKey(v string)`

SetTopologyKey sets TopologyKey field to given value.

### HasTopologyKey

`func (o *V1TopologySpreadConstraint) HasTopologyKey() bool`

HasTopologyKey returns a boolean if a field has been set.

### GetWhenUnsatisfiable

`func (o *V1TopologySpreadConstraint) GetWhenUnsatisfiable() string`

GetWhenUnsatisfiable returns the WhenUnsatisfiable field if non-nil, zero value otherwise.

### GetWhenUnsatisfiableOk

`func (o *V1TopologySpreadConstraint) GetWhenUnsatisfiableOk() (*string, bool)`

GetWhenUnsatisfiableOk returns a tuple with the WhenUnsatisfiable field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWhenUnsatisfiable

`func (o *V1TopologySpreadConstraint) SetWhenUnsatisfiable(v string)`

SetWhenUnsatisfiable sets WhenUnsatisfiable field to given value.

### HasWhenUnsatisfiable

`func (o *V1TopologySpreadConstraint) HasWhenUnsatisfiable() bool`

HasWhenUnsatisfiable returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


