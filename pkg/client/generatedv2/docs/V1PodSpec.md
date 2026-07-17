# V1PodSpec

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ActiveDeadlineSeconds** | Pointer to **int32** | Optional duration in seconds the pod may be active on the node relative to StartTime before the system will actively try to mark it failed and kill associated containers. Value must be a positive integer. +optional | [optional] 
**Affinity** | Pointer to [**V1Affinity**](V1Affinity.md) | If specified, the pod&#39;s scheduling constraints +optional | [optional] 
**AutomountServiceAccountToken** | Pointer to **bool** | AutomountServiceAccountToken indicates whether a service account token should be automatically mounted. +optional | [optional] 
**Containers** | Pointer to [**[]V1Container**](V1Container.md) | List of containers belonging to the pod. Containers cannot currently be added or removed. There must be at least one container in a Pod. Cannot be updated. +patchMergeKey&#x3D;name +patchStrategy&#x3D;merge +listType&#x3D;map +listMapKey&#x3D;name | [optional] 
**DnsConfig** | Pointer to [**V1PodDNSConfig**](V1PodDNSConfig.md) | Specifies the DNS parameters of a pod. Parameters specified here will be merged to the generated DNS configuration based on DNSPolicy. +optional | [optional] 
**DnsPolicy** | Pointer to **string** | Set DNS policy for the pod. Defaults to \&quot;ClusterFirst\&quot;. Valid values are &#39;ClusterFirstWithHostNet&#39;, &#39;ClusterFirst&#39;, &#39;Default&#39; or &#39;None&#39;. DNS parameters given in DNSConfig will be merged with the policy selected with DNSPolicy. To have DNS options set along with hostNetwork, you have to specify DNS policy explicitly to &#39;ClusterFirstWithHostNet&#39;. +optional | [optional] 
**EnableServiceLinks** | Pointer to **bool** | EnableServiceLinks indicates whether information about services should be injected into pod&#39;s environment variables, matching the syntax of Docker links. Optional: Defaults to true. +optional | [optional] 
**EphemeralContainers** | Pointer to [**[]V1EphemeralContainer**](V1EphemeralContainer.md) | List of ephemeral containers run in this pod. Ephemeral containers may be run in an existing pod to perform user-initiated actions such as debugging. This list cannot be specified when creating a pod, and it cannot be modified by updating the pod spec. In order to add an ephemeral container to an existing pod, use the pod&#39;s ephemeralcontainers subresource. +optional +patchMergeKey&#x3D;name +patchStrategy&#x3D;merge +listType&#x3D;map +listMapKey&#x3D;name | [optional] 
**HostAliases** | Pointer to [**[]V1HostAlias**](V1HostAlias.md) | HostAliases is an optional list of hosts and IPs that will be injected into the pod&#39;s hosts file if specified. +optional +patchMergeKey&#x3D;ip +patchStrategy&#x3D;merge +listType&#x3D;map +listMapKey&#x3D;ip | [optional] 
**HostIPC** | Pointer to **bool** | Use the host&#39;s ipc namespace. Optional: Default to false. +k8s:conversion-gen&#x3D;false +optional | [optional] 
**HostNetwork** | Pointer to **bool** | Host networking requested for this pod. Use the host&#39;s network namespace. When using HostNetwork you should specify ports so the scheduler is aware. When &#x60;hostNetwork&#x60; is true, specified &#x60;hostPort&#x60; fields in port definitions must match &#x60;containerPort&#x60;, and unspecified &#x60;hostPort&#x60; fields in port definitions are defaulted to match &#x60;containerPort&#x60;. Default to false. +k8s:conversion-gen&#x3D;false +optional | [optional] 
**HostPID** | Pointer to **bool** | Use the host&#39;s pid namespace. Optional: Default to false. +k8s:conversion-gen&#x3D;false +optional | [optional] 
**HostUsers** | Pointer to **bool** | Use the host&#39;s user namespace. Optional: Default to true. If set to true or not present, the pod will be run in the host user namespace, useful for when the pod needs a feature only available to the host user namespace, such as loading a kernel module with CAP_SYS_MODULE. When set to false, a new userns is created for the pod. Setting false is useful for mitigating container breakout vulnerabilities even allowing users to run their containers as root without actually having root privileges on the host. This field is alpha-level and is only honored by servers that enable the UserNamespacesSupport feature. +k8s:conversion-gen&#x3D;false +optional | [optional] 
**Hostname** | Pointer to **string** | Specifies the hostname of the Pod If not specified, the pod&#39;s hostname will be set to a system-defined value. +optional | [optional] 
**HostnameOverride** | Pointer to **string** | HostnameOverride specifies an explicit override for the pod&#39;s hostname as perceived by the pod. This field only specifies the pod&#39;s hostname and does not affect its DNS records. When this field is set to a non-empty string: - It takes precedence over the values set in &#x60;hostname&#x60; and &#x60;subdomain&#x60;. - The Pod&#39;s hostname will be set to this value. - &#x60;setHostnameAsFQDN&#x60; must be nil or set to false. - &#x60;hostNetwork&#x60; must be set to false.  This field must be a valid DNS subdomain as defined in RFC 1123 and contain at most 64 characters. Requires the HostnameOverride feature gate to be enabled.  +featureGate&#x3D;HostnameOverride +optional | [optional] 
**ImagePullSecrets** | Pointer to [**[]V1LocalObjectReference**](V1LocalObjectReference.md) | ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec. If specified, these secrets will be passed to individual puller implementations for them to use. More info: https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod +optional +patchMergeKey&#x3D;name +patchStrategy&#x3D;merge +listType&#x3D;map +listMapKey&#x3D;name | [optional] 
**InitContainers** | Pointer to [**[]V1Container**](V1Container.md) | List of initialization containers belonging to the pod. Init containers are executed in order prior to containers being started. If any init container fails, the pod is considered to have failed and is handled according to its restartPolicy. The name for an init container or normal container must be unique among all containers. Init containers may not have Lifecycle actions, Readiness probes, Liveness probes, or Startup probes. The resourceRequirements of an init container are taken into account during scheduling by finding the highest request/limit for each resource type, and then using the max of that value or the sum of the normal containers. Limits are applied to init containers in a similar fashion. Init containers cannot currently be added or removed. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/init-containers/ +patchMergeKey&#x3D;name +patchStrategy&#x3D;merge +listType&#x3D;map +listMapKey&#x3D;name | [optional] 
**NodeName** | Pointer to **string** | NodeName indicates in which node this pod is scheduled. If empty, this pod is a candidate for scheduling by the scheduler defined in schedulerName. Once this field is set, the kubelet for this node becomes responsible for the lifecycle of this pod. This field should not be used to express a desire for the pod to be scheduled on a specific node. https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#nodename +optional | [optional] 
**NodeSelector** | Pointer to **map[string]string** | NodeSelector is a selector which must be true for the pod to fit on a node. Selector which must match a node&#39;s labels for the pod to be scheduled on that node. More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/ +optional +mapType&#x3D;atomic | [optional] 
**Os** | Pointer to [**V1PodOS**](V1PodOS.md) | Specifies the OS of the containers in the pod. Some pod and container fields are restricted if this is set.  If the OS field is set to linux, the following fields must be unset: -securityContext.windowsOptions  If the OS field is set to windows, following fields must be unset: - spec.hostPID - spec.hostIPC - spec.hostUsers - spec.resources - spec.securityContext.appArmorProfile - spec.securityContext.seLinuxOptions - spec.securityContext.seccompProfile - spec.securityContext.fsGroup - spec.securityContext.fsGroupChangePolicy - spec.securityContext.sysctls - spec.shareProcessNamespace - spec.securityContext.runAsUser - spec.securityContext.runAsGroup - spec.securityContext.supplementalGroups - spec.securityContext.supplementalGroupsPolicy - spec.containers[*].securityContext.appArmorProfile - spec.containers[*].securityContext.seLinuxOptions - spec.containers[*].securityContext.seccompProfile - spec.containers[*].securityContext.capabilities - spec.containers[*].securityContext.readOnlyRootFilesystem - spec.containers[*].securityContext.privileged - spec.containers[*].securityContext.allowPrivilegeEscalation - spec.containers[*].securityContext.procMount - spec.containers[*].securityContext.runAsUser - spec.containers[*].securityContext.runAsGroup +optional | [optional] 
**Overhead** | Pointer to **map[string]interface{}** | Overhead represents the resource overhead associated with running a pod for a given RuntimeClass. This field will be autopopulated at admission time by the RuntimeClass admission controller. If the RuntimeClass admission controller is enabled, overhead must not be set in Pod create requests. The RuntimeClass admission controller will reject Pod create requests which have the overhead already set. If RuntimeClass is configured and selected in the PodSpec, Overhead will be set to the value defined in the corresponding RuntimeClass, otherwise it will remain unset and treated as zero. More info: https://git.k8s.io/enhancements/keps/sig-node/688-pod-overhead/README.md +optional | [optional] 
**PreemptionPolicy** | Pointer to **string** | PreemptionPolicy is the Policy for preempting pods with lower priority. One of Never, PreemptLowerPriority. Defaults to PreemptLowerPriority if unset. +optional | [optional] 
**Priority** | Pointer to **int32** | The priority value. Various system components use this field to find the priority of the pod. When Priority Admission Controller is enabled, it prevents users from setting this field. The admission controller populates this field from PriorityClassName. The higher the value, the higher the priority. +optional | [optional] 
**PriorityClassName** | Pointer to **string** | If specified, indicates the pod&#39;s priority. \&quot;system-node-critical\&quot; and \&quot;system-cluster-critical\&quot; are two special keywords which indicate the highest priorities with the former being the highest priority. Any other name must be defined by creating a PriorityClass object with that name. If not specified, the pod priority will be default or zero if there is no default. +optional | [optional] 
**ReadinessGates** | Pointer to [**[]V1PodReadinessGate**](V1PodReadinessGate.md) | If specified, all readiness gates will be evaluated for pod readiness. A pod is ready when all its containers are ready AND all conditions specified in the readiness gates have status equal to \&quot;True\&quot; More info: https://git.k8s.io/enhancements/keps/sig-network/580-pod-readiness-gates +optional +listType&#x3D;atomic | [optional] 
**ResourceClaims** | Pointer to [**[]V1PodResourceClaim**](V1PodResourceClaim.md) | ResourceClaims defines which ResourceClaims must be allocated and reserved before the Pod is allowed to start. The resources will be made available to those containers which consume them by name.  This is an alpha field and requires enabling the DynamicResourceAllocation feature gate.  This field is immutable.  +patchMergeKey&#x3D;name +patchStrategy&#x3D;merge,retainKeys +listType&#x3D;map +listMapKey&#x3D;name +featureGate&#x3D;DynamicResourceAllocation +optional | [optional] 
**Resources** | Pointer to [**V1ResourceRequirements**](V1ResourceRequirements.md) | Resources is the total amount of CPU and Memory resources required by all containers in the pod. It supports specifying Requests and Limits for \&quot;cpu\&quot;, \&quot;memory\&quot; and \&quot;hugepages-\&quot; resource names only. ResourceClaims are not supported.  This field enables fine-grained control over resource allocation for the entire pod, allowing resource sharing among containers in a pod. TODO: For beta graduation, expand this comment with a detailed explanation.  This is an alpha field and requires enabling the PodLevelResources feature gate.  +featureGate&#x3D;PodLevelResources +optional | [optional] 
**RestartPolicy** | Pointer to **string** | Restart policy for all containers within the pod. One of Always, OnFailure, Never. In some contexts, only a subset of those values may be permitted. Default to Always. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#restart-policy +optional | [optional] 
**RuntimeClassName** | Pointer to **string** | RuntimeClassName refers to a RuntimeClass object in the node.k8s.io group, which should be used to run this pod.  If no RuntimeClass resource matches the named class, the pod will not be run. If unset or empty, the \&quot;legacy\&quot; RuntimeClass will be used, which is an implicit class with an empty definition that uses the default runtime handler. More info: https://git.k8s.io/enhancements/keps/sig-node/585-runtime-class +optional | [optional] 
**SchedulerName** | Pointer to **string** | If specified, the pod will be dispatched by specified scheduler. If not specified, the pod will be dispatched by default scheduler. +optional | [optional] 
**SchedulingGates** | Pointer to [**[]V1PodSchedulingGate**](V1PodSchedulingGate.md) | SchedulingGates is an opaque list of values that if specified will block scheduling the pod. If schedulingGates is not empty, the pod will stay in the SchedulingGated state and the scheduler will not attempt to schedule the pod.  SchedulingGates can only be set at pod creation time, and be removed only afterwards.  +patchMergeKey&#x3D;name +patchStrategy&#x3D;merge +listType&#x3D;map +listMapKey&#x3D;name +optional | [optional] 
**SecurityContext** | Pointer to [**V1PodSecurityContext**](V1PodSecurityContext.md) | SecurityContext holds pod-level security attributes and common container settings. Optional: Defaults to empty.  See type description for default values of each field. +optional | [optional] 
**ServiceAccount** | Pointer to **string** | DeprecatedServiceAccount is a deprecated alias for ServiceAccountName. Deprecated: Use serviceAccountName instead. +k8s:conversion-gen&#x3D;false +optional | [optional] 
**ServiceAccountName** | Pointer to **string** | ServiceAccountName is the name of the ServiceAccount to use to run this pod. More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/ +optional | [optional] 
**SetHostnameAsFQDN** | Pointer to **bool** | If true the pod&#39;s hostname will be configured as the pod&#39;s FQDN, rather than the leaf name (the default). In Linux containers, this means setting the FQDN in the hostname field of the kernel (the nodename field of struct utsname). In Windows containers, this means setting the registry value of hostname for the registry key HKEY_LOCAL_MACHINE\\\\SYSTEM\\\\CurrentControlSet\\\\Services\\\\Tcpip\\\\Parameters to FQDN. If a pod does not have FQDN, this has no effect. Default to false. +optional | [optional] 
**ShareProcessNamespace** | Pointer to **bool** | Share a single process namespace between all of the containers in a pod. When this is set containers will be able to view and signal processes from other containers in the same pod, and the first process in each container will not be assigned PID 1. HostPID and ShareProcessNamespace cannot both be set. Optional: Default to false. +k8s:conversion-gen&#x3D;false +optional | [optional] 
**Subdomain** | Pointer to **string** | If specified, the fully qualified Pod hostname will be \&quot;&lt;hostname&gt;.&lt;subdomain&gt;.&lt;pod namespace&gt;.svc.&lt;cluster domain&gt;\&quot;. If not specified, the pod will not have a domainname at all. +optional | [optional] 
**TerminationGracePeriodSeconds** | Pointer to **int32** | Optional duration in seconds the pod needs to terminate gracefully. May be decreased in delete request. Value must be non-negative integer. The value zero indicates stop immediately via the kill signal (no opportunity to shut down). If this value is nil, the default grace period will be used instead. The grace period is the duration in seconds after the processes running in the pod are sent a termination signal and the time when the processes are forcibly halted with a kill signal. Set this value longer than the expected cleanup time for your process. Defaults to 30 seconds. +optional | [optional] 
**Tolerations** | Pointer to [**[]V1Toleration**](V1Toleration.md) | If specified, the pod&#39;s tolerations. +optional +listType&#x3D;atomic | [optional] 
**TopologySpreadConstraints** | Pointer to [**[]V1TopologySpreadConstraint**](V1TopologySpreadConstraint.md) | TopologySpreadConstraints describes how a group of pods ought to spread across topology domains. Scheduler will schedule pods in a way which abides by the constraints. All topologySpreadConstraints are ANDed. +optional +patchMergeKey&#x3D;topologyKey +patchStrategy&#x3D;merge +listType&#x3D;map +listMapKey&#x3D;topologyKey +listMapKey&#x3D;whenUnsatisfiable | [optional] 
**Volumes** | Pointer to [**[]V1Volume**](V1Volume.md) | List of volumes that can be mounted by containers belonging to the pod. More info: https://kubernetes.io/docs/concepts/storage/volumes +optional +patchMergeKey&#x3D;name +patchStrategy&#x3D;merge,retainKeys +listType&#x3D;map +listMapKey&#x3D;name | [optional] 

## Methods

### NewV1PodSpec

`func NewV1PodSpec() *V1PodSpec`

NewV1PodSpec instantiates a new V1PodSpec object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1PodSpecWithDefaults

`func NewV1PodSpecWithDefaults() *V1PodSpec`

NewV1PodSpecWithDefaults instantiates a new V1PodSpec object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetActiveDeadlineSeconds

`func (o *V1PodSpec) GetActiveDeadlineSeconds() int32`

GetActiveDeadlineSeconds returns the ActiveDeadlineSeconds field if non-nil, zero value otherwise.

### GetActiveDeadlineSecondsOk

`func (o *V1PodSpec) GetActiveDeadlineSecondsOk() (*int32, bool)`

GetActiveDeadlineSecondsOk returns a tuple with the ActiveDeadlineSeconds field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetActiveDeadlineSeconds

`func (o *V1PodSpec) SetActiveDeadlineSeconds(v int32)`

SetActiveDeadlineSeconds sets ActiveDeadlineSeconds field to given value.

### HasActiveDeadlineSeconds

`func (o *V1PodSpec) HasActiveDeadlineSeconds() bool`

HasActiveDeadlineSeconds returns a boolean if a field has been set.

### GetAffinity

`func (o *V1PodSpec) GetAffinity() V1Affinity`

GetAffinity returns the Affinity field if non-nil, zero value otherwise.

### GetAffinityOk

`func (o *V1PodSpec) GetAffinityOk() (*V1Affinity, bool)`

GetAffinityOk returns a tuple with the Affinity field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAffinity

`func (o *V1PodSpec) SetAffinity(v V1Affinity)`

SetAffinity sets Affinity field to given value.

### HasAffinity

`func (o *V1PodSpec) HasAffinity() bool`

HasAffinity returns a boolean if a field has been set.

### GetAutomountServiceAccountToken

`func (o *V1PodSpec) GetAutomountServiceAccountToken() bool`

GetAutomountServiceAccountToken returns the AutomountServiceAccountToken field if non-nil, zero value otherwise.

### GetAutomountServiceAccountTokenOk

`func (o *V1PodSpec) GetAutomountServiceAccountTokenOk() (*bool, bool)`

GetAutomountServiceAccountTokenOk returns a tuple with the AutomountServiceAccountToken field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAutomountServiceAccountToken

`func (o *V1PodSpec) SetAutomountServiceAccountToken(v bool)`

SetAutomountServiceAccountToken sets AutomountServiceAccountToken field to given value.

### HasAutomountServiceAccountToken

`func (o *V1PodSpec) HasAutomountServiceAccountToken() bool`

HasAutomountServiceAccountToken returns a boolean if a field has been set.

### GetContainers

`func (o *V1PodSpec) GetContainers() []V1Container`

GetContainers returns the Containers field if non-nil, zero value otherwise.

### GetContainersOk

`func (o *V1PodSpec) GetContainersOk() (*[]V1Container, bool)`

GetContainersOk returns a tuple with the Containers field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetContainers

`func (o *V1PodSpec) SetContainers(v []V1Container)`

SetContainers sets Containers field to given value.

### HasContainers

`func (o *V1PodSpec) HasContainers() bool`

HasContainers returns a boolean if a field has been set.

### GetDnsConfig

`func (o *V1PodSpec) GetDnsConfig() V1PodDNSConfig`

GetDnsConfig returns the DnsConfig field if non-nil, zero value otherwise.

### GetDnsConfigOk

`func (o *V1PodSpec) GetDnsConfigOk() (*V1PodDNSConfig, bool)`

GetDnsConfigOk returns a tuple with the DnsConfig field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDnsConfig

`func (o *V1PodSpec) SetDnsConfig(v V1PodDNSConfig)`

SetDnsConfig sets DnsConfig field to given value.

### HasDnsConfig

`func (o *V1PodSpec) HasDnsConfig() bool`

HasDnsConfig returns a boolean if a field has been set.

### GetDnsPolicy

`func (o *V1PodSpec) GetDnsPolicy() string`

GetDnsPolicy returns the DnsPolicy field if non-nil, zero value otherwise.

### GetDnsPolicyOk

`func (o *V1PodSpec) GetDnsPolicyOk() (*string, bool)`

GetDnsPolicyOk returns a tuple with the DnsPolicy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDnsPolicy

`func (o *V1PodSpec) SetDnsPolicy(v string)`

SetDnsPolicy sets DnsPolicy field to given value.

### HasDnsPolicy

`func (o *V1PodSpec) HasDnsPolicy() bool`

HasDnsPolicy returns a boolean if a field has been set.

### GetEnableServiceLinks

`func (o *V1PodSpec) GetEnableServiceLinks() bool`

GetEnableServiceLinks returns the EnableServiceLinks field if non-nil, zero value otherwise.

### GetEnableServiceLinksOk

`func (o *V1PodSpec) GetEnableServiceLinksOk() (*bool, bool)`

GetEnableServiceLinksOk returns a tuple with the EnableServiceLinks field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEnableServiceLinks

`func (o *V1PodSpec) SetEnableServiceLinks(v bool)`

SetEnableServiceLinks sets EnableServiceLinks field to given value.

### HasEnableServiceLinks

`func (o *V1PodSpec) HasEnableServiceLinks() bool`

HasEnableServiceLinks returns a boolean if a field has been set.

### GetEphemeralContainers

`func (o *V1PodSpec) GetEphemeralContainers() []V1EphemeralContainer`

GetEphemeralContainers returns the EphemeralContainers field if non-nil, zero value otherwise.

### GetEphemeralContainersOk

`func (o *V1PodSpec) GetEphemeralContainersOk() (*[]V1EphemeralContainer, bool)`

GetEphemeralContainersOk returns a tuple with the EphemeralContainers field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEphemeralContainers

`func (o *V1PodSpec) SetEphemeralContainers(v []V1EphemeralContainer)`

SetEphemeralContainers sets EphemeralContainers field to given value.

### HasEphemeralContainers

`func (o *V1PodSpec) HasEphemeralContainers() bool`

HasEphemeralContainers returns a boolean if a field has been set.

### GetHostAliases

`func (o *V1PodSpec) GetHostAliases() []V1HostAlias`

GetHostAliases returns the HostAliases field if non-nil, zero value otherwise.

### GetHostAliasesOk

`func (o *V1PodSpec) GetHostAliasesOk() (*[]V1HostAlias, bool)`

GetHostAliasesOk returns a tuple with the HostAliases field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHostAliases

`func (o *V1PodSpec) SetHostAliases(v []V1HostAlias)`

SetHostAliases sets HostAliases field to given value.

### HasHostAliases

`func (o *V1PodSpec) HasHostAliases() bool`

HasHostAliases returns a boolean if a field has been set.

### GetHostIPC

`func (o *V1PodSpec) GetHostIPC() bool`

GetHostIPC returns the HostIPC field if non-nil, zero value otherwise.

### GetHostIPCOk

`func (o *V1PodSpec) GetHostIPCOk() (*bool, bool)`

GetHostIPCOk returns a tuple with the HostIPC field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHostIPC

`func (o *V1PodSpec) SetHostIPC(v bool)`

SetHostIPC sets HostIPC field to given value.

### HasHostIPC

`func (o *V1PodSpec) HasHostIPC() bool`

HasHostIPC returns a boolean if a field has been set.

### GetHostNetwork

`func (o *V1PodSpec) GetHostNetwork() bool`

GetHostNetwork returns the HostNetwork field if non-nil, zero value otherwise.

### GetHostNetworkOk

`func (o *V1PodSpec) GetHostNetworkOk() (*bool, bool)`

GetHostNetworkOk returns a tuple with the HostNetwork field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHostNetwork

`func (o *V1PodSpec) SetHostNetwork(v bool)`

SetHostNetwork sets HostNetwork field to given value.

### HasHostNetwork

`func (o *V1PodSpec) HasHostNetwork() bool`

HasHostNetwork returns a boolean if a field has been set.

### GetHostPID

`func (o *V1PodSpec) GetHostPID() bool`

GetHostPID returns the HostPID field if non-nil, zero value otherwise.

### GetHostPIDOk

`func (o *V1PodSpec) GetHostPIDOk() (*bool, bool)`

GetHostPIDOk returns a tuple with the HostPID field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHostPID

`func (o *V1PodSpec) SetHostPID(v bool)`

SetHostPID sets HostPID field to given value.

### HasHostPID

`func (o *V1PodSpec) HasHostPID() bool`

HasHostPID returns a boolean if a field has been set.

### GetHostUsers

`func (o *V1PodSpec) GetHostUsers() bool`

GetHostUsers returns the HostUsers field if non-nil, zero value otherwise.

### GetHostUsersOk

`func (o *V1PodSpec) GetHostUsersOk() (*bool, bool)`

GetHostUsersOk returns a tuple with the HostUsers field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHostUsers

`func (o *V1PodSpec) SetHostUsers(v bool)`

SetHostUsers sets HostUsers field to given value.

### HasHostUsers

`func (o *V1PodSpec) HasHostUsers() bool`

HasHostUsers returns a boolean if a field has been set.

### GetHostname

`func (o *V1PodSpec) GetHostname() string`

GetHostname returns the Hostname field if non-nil, zero value otherwise.

### GetHostnameOk

`func (o *V1PodSpec) GetHostnameOk() (*string, bool)`

GetHostnameOk returns a tuple with the Hostname field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHostname

`func (o *V1PodSpec) SetHostname(v string)`

SetHostname sets Hostname field to given value.

### HasHostname

`func (o *V1PodSpec) HasHostname() bool`

HasHostname returns a boolean if a field has been set.

### GetHostnameOverride

`func (o *V1PodSpec) GetHostnameOverride() string`

GetHostnameOverride returns the HostnameOverride field if non-nil, zero value otherwise.

### GetHostnameOverrideOk

`func (o *V1PodSpec) GetHostnameOverrideOk() (*string, bool)`

GetHostnameOverrideOk returns a tuple with the HostnameOverride field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHostnameOverride

`func (o *V1PodSpec) SetHostnameOverride(v string)`

SetHostnameOverride sets HostnameOverride field to given value.

### HasHostnameOverride

`func (o *V1PodSpec) HasHostnameOverride() bool`

HasHostnameOverride returns a boolean if a field has been set.

### GetImagePullSecrets

`func (o *V1PodSpec) GetImagePullSecrets() []V1LocalObjectReference`

GetImagePullSecrets returns the ImagePullSecrets field if non-nil, zero value otherwise.

### GetImagePullSecretsOk

`func (o *V1PodSpec) GetImagePullSecretsOk() (*[]V1LocalObjectReference, bool)`

GetImagePullSecretsOk returns a tuple with the ImagePullSecrets field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetImagePullSecrets

`func (o *V1PodSpec) SetImagePullSecrets(v []V1LocalObjectReference)`

SetImagePullSecrets sets ImagePullSecrets field to given value.

### HasImagePullSecrets

`func (o *V1PodSpec) HasImagePullSecrets() bool`

HasImagePullSecrets returns a boolean if a field has been set.

### GetInitContainers

`func (o *V1PodSpec) GetInitContainers() []V1Container`

GetInitContainers returns the InitContainers field if non-nil, zero value otherwise.

### GetInitContainersOk

`func (o *V1PodSpec) GetInitContainersOk() (*[]V1Container, bool)`

GetInitContainersOk returns a tuple with the InitContainers field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInitContainers

`func (o *V1PodSpec) SetInitContainers(v []V1Container)`

SetInitContainers sets InitContainers field to given value.

### HasInitContainers

`func (o *V1PodSpec) HasInitContainers() bool`

HasInitContainers returns a boolean if a field has been set.

### GetNodeName

`func (o *V1PodSpec) GetNodeName() string`

GetNodeName returns the NodeName field if non-nil, zero value otherwise.

### GetNodeNameOk

`func (o *V1PodSpec) GetNodeNameOk() (*string, bool)`

GetNodeNameOk returns a tuple with the NodeName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNodeName

`func (o *V1PodSpec) SetNodeName(v string)`

SetNodeName sets NodeName field to given value.

### HasNodeName

`func (o *V1PodSpec) HasNodeName() bool`

HasNodeName returns a boolean if a field has been set.

### GetNodeSelector

`func (o *V1PodSpec) GetNodeSelector() map[string]string`

GetNodeSelector returns the NodeSelector field if non-nil, zero value otherwise.

### GetNodeSelectorOk

`func (o *V1PodSpec) GetNodeSelectorOk() (*map[string]string, bool)`

GetNodeSelectorOk returns a tuple with the NodeSelector field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNodeSelector

`func (o *V1PodSpec) SetNodeSelector(v map[string]string)`

SetNodeSelector sets NodeSelector field to given value.

### HasNodeSelector

`func (o *V1PodSpec) HasNodeSelector() bool`

HasNodeSelector returns a boolean if a field has been set.

### GetOs

`func (o *V1PodSpec) GetOs() V1PodOS`

GetOs returns the Os field if non-nil, zero value otherwise.

### GetOsOk

`func (o *V1PodSpec) GetOsOk() (*V1PodOS, bool)`

GetOsOk returns a tuple with the Os field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOs

`func (o *V1PodSpec) SetOs(v V1PodOS)`

SetOs sets Os field to given value.

### HasOs

`func (o *V1PodSpec) HasOs() bool`

HasOs returns a boolean if a field has been set.

### GetOverhead

`func (o *V1PodSpec) GetOverhead() map[string]interface{}`

GetOverhead returns the Overhead field if non-nil, zero value otherwise.

### GetOverheadOk

`func (o *V1PodSpec) GetOverheadOk() (*map[string]interface{}, bool)`

GetOverheadOk returns a tuple with the Overhead field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOverhead

`func (o *V1PodSpec) SetOverhead(v map[string]interface{})`

SetOverhead sets Overhead field to given value.

### HasOverhead

`func (o *V1PodSpec) HasOverhead() bool`

HasOverhead returns a boolean if a field has been set.

### GetPreemptionPolicy

`func (o *V1PodSpec) GetPreemptionPolicy() string`

GetPreemptionPolicy returns the PreemptionPolicy field if non-nil, zero value otherwise.

### GetPreemptionPolicyOk

`func (o *V1PodSpec) GetPreemptionPolicyOk() (*string, bool)`

GetPreemptionPolicyOk returns a tuple with the PreemptionPolicy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPreemptionPolicy

`func (o *V1PodSpec) SetPreemptionPolicy(v string)`

SetPreemptionPolicy sets PreemptionPolicy field to given value.

### HasPreemptionPolicy

`func (o *V1PodSpec) HasPreemptionPolicy() bool`

HasPreemptionPolicy returns a boolean if a field has been set.

### GetPriority

`func (o *V1PodSpec) GetPriority() int32`

GetPriority returns the Priority field if non-nil, zero value otherwise.

### GetPriorityOk

`func (o *V1PodSpec) GetPriorityOk() (*int32, bool)`

GetPriorityOk returns a tuple with the Priority field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPriority

`func (o *V1PodSpec) SetPriority(v int32)`

SetPriority sets Priority field to given value.

### HasPriority

`func (o *V1PodSpec) HasPriority() bool`

HasPriority returns a boolean if a field has been set.

### GetPriorityClassName

`func (o *V1PodSpec) GetPriorityClassName() string`

GetPriorityClassName returns the PriorityClassName field if non-nil, zero value otherwise.

### GetPriorityClassNameOk

`func (o *V1PodSpec) GetPriorityClassNameOk() (*string, bool)`

GetPriorityClassNameOk returns a tuple with the PriorityClassName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPriorityClassName

`func (o *V1PodSpec) SetPriorityClassName(v string)`

SetPriorityClassName sets PriorityClassName field to given value.

### HasPriorityClassName

`func (o *V1PodSpec) HasPriorityClassName() bool`

HasPriorityClassName returns a boolean if a field has been set.

### GetReadinessGates

`func (o *V1PodSpec) GetReadinessGates() []V1PodReadinessGate`

GetReadinessGates returns the ReadinessGates field if non-nil, zero value otherwise.

### GetReadinessGatesOk

`func (o *V1PodSpec) GetReadinessGatesOk() (*[]V1PodReadinessGate, bool)`

GetReadinessGatesOk returns a tuple with the ReadinessGates field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReadinessGates

`func (o *V1PodSpec) SetReadinessGates(v []V1PodReadinessGate)`

SetReadinessGates sets ReadinessGates field to given value.

### HasReadinessGates

`func (o *V1PodSpec) HasReadinessGates() bool`

HasReadinessGates returns a boolean if a field has been set.

### GetResourceClaims

`func (o *V1PodSpec) GetResourceClaims() []V1PodResourceClaim`

GetResourceClaims returns the ResourceClaims field if non-nil, zero value otherwise.

### GetResourceClaimsOk

`func (o *V1PodSpec) GetResourceClaimsOk() (*[]V1PodResourceClaim, bool)`

GetResourceClaimsOk returns a tuple with the ResourceClaims field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResourceClaims

`func (o *V1PodSpec) SetResourceClaims(v []V1PodResourceClaim)`

SetResourceClaims sets ResourceClaims field to given value.

### HasResourceClaims

`func (o *V1PodSpec) HasResourceClaims() bool`

HasResourceClaims returns a boolean if a field has been set.

### GetResources

`func (o *V1PodSpec) GetResources() V1ResourceRequirements`

GetResources returns the Resources field if non-nil, zero value otherwise.

### GetResourcesOk

`func (o *V1PodSpec) GetResourcesOk() (*V1ResourceRequirements, bool)`

GetResourcesOk returns a tuple with the Resources field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResources

`func (o *V1PodSpec) SetResources(v V1ResourceRequirements)`

SetResources sets Resources field to given value.

### HasResources

`func (o *V1PodSpec) HasResources() bool`

HasResources returns a boolean if a field has been set.

### GetRestartPolicy

`func (o *V1PodSpec) GetRestartPolicy() string`

GetRestartPolicy returns the RestartPolicy field if non-nil, zero value otherwise.

### GetRestartPolicyOk

`func (o *V1PodSpec) GetRestartPolicyOk() (*string, bool)`

GetRestartPolicyOk returns a tuple with the RestartPolicy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRestartPolicy

`func (o *V1PodSpec) SetRestartPolicy(v string)`

SetRestartPolicy sets RestartPolicy field to given value.

### HasRestartPolicy

`func (o *V1PodSpec) HasRestartPolicy() bool`

HasRestartPolicy returns a boolean if a field has been set.

### GetRuntimeClassName

`func (o *V1PodSpec) GetRuntimeClassName() string`

GetRuntimeClassName returns the RuntimeClassName field if non-nil, zero value otherwise.

### GetRuntimeClassNameOk

`func (o *V1PodSpec) GetRuntimeClassNameOk() (*string, bool)`

GetRuntimeClassNameOk returns a tuple with the RuntimeClassName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRuntimeClassName

`func (o *V1PodSpec) SetRuntimeClassName(v string)`

SetRuntimeClassName sets RuntimeClassName field to given value.

### HasRuntimeClassName

`func (o *V1PodSpec) HasRuntimeClassName() bool`

HasRuntimeClassName returns a boolean if a field has been set.

### GetSchedulerName

`func (o *V1PodSpec) GetSchedulerName() string`

GetSchedulerName returns the SchedulerName field if non-nil, zero value otherwise.

### GetSchedulerNameOk

`func (o *V1PodSpec) GetSchedulerNameOk() (*string, bool)`

GetSchedulerNameOk returns a tuple with the SchedulerName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchedulerName

`func (o *V1PodSpec) SetSchedulerName(v string)`

SetSchedulerName sets SchedulerName field to given value.

### HasSchedulerName

`func (o *V1PodSpec) HasSchedulerName() bool`

HasSchedulerName returns a boolean if a field has been set.

### GetSchedulingGates

`func (o *V1PodSpec) GetSchedulingGates() []V1PodSchedulingGate`

GetSchedulingGates returns the SchedulingGates field if non-nil, zero value otherwise.

### GetSchedulingGatesOk

`func (o *V1PodSpec) GetSchedulingGatesOk() (*[]V1PodSchedulingGate, bool)`

GetSchedulingGatesOk returns a tuple with the SchedulingGates field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchedulingGates

`func (o *V1PodSpec) SetSchedulingGates(v []V1PodSchedulingGate)`

SetSchedulingGates sets SchedulingGates field to given value.

### HasSchedulingGates

`func (o *V1PodSpec) HasSchedulingGates() bool`

HasSchedulingGates returns a boolean if a field has been set.

### GetSecurityContext

`func (o *V1PodSpec) GetSecurityContext() V1PodSecurityContext`

GetSecurityContext returns the SecurityContext field if non-nil, zero value otherwise.

### GetSecurityContextOk

`func (o *V1PodSpec) GetSecurityContextOk() (*V1PodSecurityContext, bool)`

GetSecurityContextOk returns a tuple with the SecurityContext field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecurityContext

`func (o *V1PodSpec) SetSecurityContext(v V1PodSecurityContext)`

SetSecurityContext sets SecurityContext field to given value.

### HasSecurityContext

`func (o *V1PodSpec) HasSecurityContext() bool`

HasSecurityContext returns a boolean if a field has been set.

### GetServiceAccount

`func (o *V1PodSpec) GetServiceAccount() string`

GetServiceAccount returns the ServiceAccount field if non-nil, zero value otherwise.

### GetServiceAccountOk

`func (o *V1PodSpec) GetServiceAccountOk() (*string, bool)`

GetServiceAccountOk returns a tuple with the ServiceAccount field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetServiceAccount

`func (o *V1PodSpec) SetServiceAccount(v string)`

SetServiceAccount sets ServiceAccount field to given value.

### HasServiceAccount

`func (o *V1PodSpec) HasServiceAccount() bool`

HasServiceAccount returns a boolean if a field has been set.

### GetServiceAccountName

`func (o *V1PodSpec) GetServiceAccountName() string`

GetServiceAccountName returns the ServiceAccountName field if non-nil, zero value otherwise.

### GetServiceAccountNameOk

`func (o *V1PodSpec) GetServiceAccountNameOk() (*string, bool)`

GetServiceAccountNameOk returns a tuple with the ServiceAccountName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetServiceAccountName

`func (o *V1PodSpec) SetServiceAccountName(v string)`

SetServiceAccountName sets ServiceAccountName field to given value.

### HasServiceAccountName

`func (o *V1PodSpec) HasServiceAccountName() bool`

HasServiceAccountName returns a boolean if a field has been set.

### GetSetHostnameAsFQDN

`func (o *V1PodSpec) GetSetHostnameAsFQDN() bool`

GetSetHostnameAsFQDN returns the SetHostnameAsFQDN field if non-nil, zero value otherwise.

### GetSetHostnameAsFQDNOk

`func (o *V1PodSpec) GetSetHostnameAsFQDNOk() (*bool, bool)`

GetSetHostnameAsFQDNOk returns a tuple with the SetHostnameAsFQDN field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSetHostnameAsFQDN

`func (o *V1PodSpec) SetSetHostnameAsFQDN(v bool)`

SetSetHostnameAsFQDN sets SetHostnameAsFQDN field to given value.

### HasSetHostnameAsFQDN

`func (o *V1PodSpec) HasSetHostnameAsFQDN() bool`

HasSetHostnameAsFQDN returns a boolean if a field has been set.

### GetShareProcessNamespace

`func (o *V1PodSpec) GetShareProcessNamespace() bool`

GetShareProcessNamespace returns the ShareProcessNamespace field if non-nil, zero value otherwise.

### GetShareProcessNamespaceOk

`func (o *V1PodSpec) GetShareProcessNamespaceOk() (*bool, bool)`

GetShareProcessNamespaceOk returns a tuple with the ShareProcessNamespace field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetShareProcessNamespace

`func (o *V1PodSpec) SetShareProcessNamespace(v bool)`

SetShareProcessNamespace sets ShareProcessNamespace field to given value.

### HasShareProcessNamespace

`func (o *V1PodSpec) HasShareProcessNamespace() bool`

HasShareProcessNamespace returns a boolean if a field has been set.

### GetSubdomain

`func (o *V1PodSpec) GetSubdomain() string`

GetSubdomain returns the Subdomain field if non-nil, zero value otherwise.

### GetSubdomainOk

`func (o *V1PodSpec) GetSubdomainOk() (*string, bool)`

GetSubdomainOk returns a tuple with the Subdomain field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSubdomain

`func (o *V1PodSpec) SetSubdomain(v string)`

SetSubdomain sets Subdomain field to given value.

### HasSubdomain

`func (o *V1PodSpec) HasSubdomain() bool`

HasSubdomain returns a boolean if a field has been set.

### GetTerminationGracePeriodSeconds

`func (o *V1PodSpec) GetTerminationGracePeriodSeconds() int32`

GetTerminationGracePeriodSeconds returns the TerminationGracePeriodSeconds field if non-nil, zero value otherwise.

### GetTerminationGracePeriodSecondsOk

`func (o *V1PodSpec) GetTerminationGracePeriodSecondsOk() (*int32, bool)`

GetTerminationGracePeriodSecondsOk returns a tuple with the TerminationGracePeriodSeconds field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTerminationGracePeriodSeconds

`func (o *V1PodSpec) SetTerminationGracePeriodSeconds(v int32)`

SetTerminationGracePeriodSeconds sets TerminationGracePeriodSeconds field to given value.

### HasTerminationGracePeriodSeconds

`func (o *V1PodSpec) HasTerminationGracePeriodSeconds() bool`

HasTerminationGracePeriodSeconds returns a boolean if a field has been set.

### GetTolerations

`func (o *V1PodSpec) GetTolerations() []V1Toleration`

GetTolerations returns the Tolerations field if non-nil, zero value otherwise.

### GetTolerationsOk

`func (o *V1PodSpec) GetTolerationsOk() (*[]V1Toleration, bool)`

GetTolerationsOk returns a tuple with the Tolerations field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTolerations

`func (o *V1PodSpec) SetTolerations(v []V1Toleration)`

SetTolerations sets Tolerations field to given value.

### HasTolerations

`func (o *V1PodSpec) HasTolerations() bool`

HasTolerations returns a boolean if a field has been set.

### GetTopologySpreadConstraints

`func (o *V1PodSpec) GetTopologySpreadConstraints() []V1TopologySpreadConstraint`

GetTopologySpreadConstraints returns the TopologySpreadConstraints field if non-nil, zero value otherwise.

### GetTopologySpreadConstraintsOk

`func (o *V1PodSpec) GetTopologySpreadConstraintsOk() (*[]V1TopologySpreadConstraint, bool)`

GetTopologySpreadConstraintsOk returns a tuple with the TopologySpreadConstraints field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTopologySpreadConstraints

`func (o *V1PodSpec) SetTopologySpreadConstraints(v []V1TopologySpreadConstraint)`

SetTopologySpreadConstraints sets TopologySpreadConstraints field to given value.

### HasTopologySpreadConstraints

`func (o *V1PodSpec) HasTopologySpreadConstraints() bool`

HasTopologySpreadConstraints returns a boolean if a field has been set.

### GetVolumes

`func (o *V1PodSpec) GetVolumes() []V1Volume`

GetVolumes returns the Volumes field if non-nil, zero value otherwise.

### GetVolumesOk

`func (o *V1PodSpec) GetVolumesOk() (*[]V1Volume, bool)`

GetVolumesOk returns a tuple with the Volumes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVolumes

`func (o *V1PodSpec) SetVolumes(v []V1Volume)`

SetVolumes sets Volumes field to given value.

### HasVolumes

`func (o *V1PodSpec) HasVolumes() bool`

HasVolumes returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


