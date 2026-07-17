# V1EphemeralContainer

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Args** | Pointer to **[]string** | Arguments to the entrypoint. The image&#39;s CMD is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container&#39;s environment. If a variable cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. \&quot;$$(VAR_NAME)\&quot; will produce the string literal \&quot;$(VAR_NAME)\&quot;. Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell +optional +listType&#x3D;atomic | [optional] 
**Command** | Pointer to **[]string** | Entrypoint array. Not executed within a shell. The image&#39;s ENTRYPOINT is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container&#39;s environment. If a variable cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. \&quot;$$(VAR_NAME)\&quot; will produce the string literal \&quot;$(VAR_NAME)\&quot;. Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell +optional +listType&#x3D;atomic | [optional] 
**Env** | Pointer to [**[]V1EnvVar**](V1EnvVar.md) | List of environment variables to set in the container. Cannot be updated. +optional +patchMergeKey&#x3D;name +patchStrategy&#x3D;merge +listType&#x3D;map +listMapKey&#x3D;name | [optional] 
**EnvFrom** | Pointer to [**[]V1EnvFromSource**](V1EnvFromSource.md) | List of sources to populate environment variables in the container. The keys defined within a source may consist of any printable ASCII characters except &#39;&#x3D;&#39;. When a key exists in multiple sources, the value associated with the last source will take precedence. Values defined by an Env with a duplicate key will take precedence. Cannot be updated. +optional +listType&#x3D;atomic | [optional] 
**Image** | Pointer to **string** | Container image name. More info: https://kubernetes.io/docs/concepts/containers/images | [optional] 
**ImagePullPolicy** | Pointer to **string** | Image pull policy. One of Always, Never, IfNotPresent. Defaults to Always if :latest tag is specified, or IfNotPresent otherwise. Cannot be updated. More info: https://kubernetes.io/docs/concepts/containers/images#updating-images +optional | [optional] 
**Lifecycle** | Pointer to [**V1Lifecycle**](V1Lifecycle.md) | Lifecycle is not allowed for ephemeral containers. +optional | [optional] 
**LivenessProbe** | Pointer to [**V1Probe**](V1Probe.md) | Probes are not allowed for ephemeral containers. +optional | [optional] 
**Name** | Pointer to **string** | Name of the ephemeral container specified as a DNS_LABEL. This name must be unique among all containers, init containers and ephemeral containers. | [optional] 
**Ports** | Pointer to [**[]V1ContainerPort**](V1ContainerPort.md) | Ports are not allowed for ephemeral containers. +optional +patchMergeKey&#x3D;containerPort +patchStrategy&#x3D;merge +listType&#x3D;map +listMapKey&#x3D;containerPort +listMapKey&#x3D;protocol | [optional] 
**ReadinessProbe** | Pointer to [**V1Probe**](V1Probe.md) | Probes are not allowed for ephemeral containers. +optional | [optional] 
**ResizePolicy** | Pointer to [**[]V1ContainerResizePolicy**](V1ContainerResizePolicy.md) | Resources resize policy for the container. +featureGate&#x3D;InPlacePodVerticalScaling +optional +listType&#x3D;atomic | [optional] 
**Resources** | Pointer to [**V1ResourceRequirements**](V1ResourceRequirements.md) | Resources are not allowed for ephemeral containers. Ephemeral containers use spare resources already allocated to the pod. +optional | [optional] 
**RestartPolicy** | Pointer to **string** | Restart policy for the container to manage the restart behavior of each container within a pod. You cannot set this field on ephemeral containers. +featureGate&#x3D;SidecarContainers +optional | [optional] 
**RestartPolicyRules** | Pointer to [**[]V1ContainerRestartRule**](V1ContainerRestartRule.md) | Represents a list of rules to be checked to determine if the container should be restarted on exit. You cannot set this field on ephemeral containers. +featureGate&#x3D;ContainerRestartRules +optional +listType&#x3D;atomic | [optional] 
**SecurityContext** | Pointer to [**V1SecurityContext**](V1SecurityContext.md) | Optional: SecurityContext defines the security options the ephemeral container should be run with. If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext. +optional | [optional] 
**StartupProbe** | Pointer to [**V1Probe**](V1Probe.md) | Probes are not allowed for ephemeral containers. +optional | [optional] 
**Stdin** | Pointer to **bool** | Whether this container should allocate a buffer for stdin in the container runtime. If this is not set, reads from stdin in the container will always result in EOF. Default is false. +optional | [optional] 
**StdinOnce** | Pointer to **bool** | Whether the container runtime should close the stdin channel after it has been opened by a single attach. When stdin is true the stdin stream will remain open across multiple attach sessions. If stdinOnce is set to true, stdin is opened on container start, is empty until the first client attaches to stdin, and then remains open and accepts data until the client disconnects, at which time stdin is closed and remains closed until the container is restarted. If this flag is false, a container processes that reads from stdin will never receive an EOF. Default is false +optional | [optional] 
**TargetContainerName** | Pointer to **string** | If set, the name of the container from PodSpec that this ephemeral container targets. The ephemeral container will be run in the namespaces (IPC, PID, etc) of this container. If not set then the ephemeral container uses the namespaces configured in the Pod spec.  The container runtime must implement support for this feature. If the runtime does not support namespace targeting then the result of setting this field is undefined. +optional | [optional] 
**TerminationMessagePath** | Pointer to **string** | Optional: Path at which the file to which the container&#39;s termination message will be written is mounted into the container&#39;s filesystem. Message written is intended to be brief final status, such as an assertion failure message. Will be truncated by the node if greater than 4096 bytes. The total message length across all containers will be limited to 12kb. Defaults to /dev/termination-log. Cannot be updated. +optional | [optional] 
**TerminationMessagePolicy** | Pointer to **string** | Indicate how the termination message should be populated. File will use the contents of terminationMessagePath to populate the container status message on both success and failure. FallbackToLogsOnError will use the last chunk of container log output if the termination message file is empty and the container exited with an error. The log output is limited to 2048 bytes or 80 lines, whichever is smaller. Defaults to File. Cannot be updated. +optional | [optional] 
**Tty** | Pointer to **bool** | Whether this container should allocate a TTY for itself, also requires &#39;stdin&#39; to be true. Default is false. +optional | [optional] 
**VolumeDevices** | Pointer to [**[]V1VolumeDevice**](V1VolumeDevice.md) | volumeDevices is the list of block devices to be used by the container. +patchMergeKey&#x3D;devicePath +patchStrategy&#x3D;merge +listType&#x3D;map +listMapKey&#x3D;devicePath +optional | [optional] 
**VolumeMounts** | Pointer to [**[]V1VolumeMount**](V1VolumeMount.md) | Pod volumes to mount into the container&#39;s filesystem. Subpath mounts are not allowed for ephemeral containers. Cannot be updated. +optional +patchMergeKey&#x3D;mountPath +patchStrategy&#x3D;merge +listType&#x3D;map +listMapKey&#x3D;mountPath | [optional] 
**WorkingDir** | Pointer to **string** | Container&#39;s working directory. If not specified, the container runtime&#39;s default will be used, which might be configured in the container image. Cannot be updated. +optional | [optional] 

## Methods

### NewV1EphemeralContainer

`func NewV1EphemeralContainer() *V1EphemeralContainer`

NewV1EphemeralContainer instantiates a new V1EphemeralContainer object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1EphemeralContainerWithDefaults

`func NewV1EphemeralContainerWithDefaults() *V1EphemeralContainer`

NewV1EphemeralContainerWithDefaults instantiates a new V1EphemeralContainer object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetArgs

`func (o *V1EphemeralContainer) GetArgs() []string`

GetArgs returns the Args field if non-nil, zero value otherwise.

### GetArgsOk

`func (o *V1EphemeralContainer) GetArgsOk() (*[]string, bool)`

GetArgsOk returns a tuple with the Args field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetArgs

`func (o *V1EphemeralContainer) SetArgs(v []string)`

SetArgs sets Args field to given value.

### HasArgs

`func (o *V1EphemeralContainer) HasArgs() bool`

HasArgs returns a boolean if a field has been set.

### GetCommand

`func (o *V1EphemeralContainer) GetCommand() []string`

GetCommand returns the Command field if non-nil, zero value otherwise.

### GetCommandOk

`func (o *V1EphemeralContainer) GetCommandOk() (*[]string, bool)`

GetCommandOk returns a tuple with the Command field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCommand

`func (o *V1EphemeralContainer) SetCommand(v []string)`

SetCommand sets Command field to given value.

### HasCommand

`func (o *V1EphemeralContainer) HasCommand() bool`

HasCommand returns a boolean if a field has been set.

### GetEnv

`func (o *V1EphemeralContainer) GetEnv() []V1EnvVar`

GetEnv returns the Env field if non-nil, zero value otherwise.

### GetEnvOk

`func (o *V1EphemeralContainer) GetEnvOk() (*[]V1EnvVar, bool)`

GetEnvOk returns a tuple with the Env field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEnv

`func (o *V1EphemeralContainer) SetEnv(v []V1EnvVar)`

SetEnv sets Env field to given value.

### HasEnv

`func (o *V1EphemeralContainer) HasEnv() bool`

HasEnv returns a boolean if a field has been set.

### GetEnvFrom

`func (o *V1EphemeralContainer) GetEnvFrom() []V1EnvFromSource`

GetEnvFrom returns the EnvFrom field if non-nil, zero value otherwise.

### GetEnvFromOk

`func (o *V1EphemeralContainer) GetEnvFromOk() (*[]V1EnvFromSource, bool)`

GetEnvFromOk returns a tuple with the EnvFrom field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEnvFrom

`func (o *V1EphemeralContainer) SetEnvFrom(v []V1EnvFromSource)`

SetEnvFrom sets EnvFrom field to given value.

### HasEnvFrom

`func (o *V1EphemeralContainer) HasEnvFrom() bool`

HasEnvFrom returns a boolean if a field has been set.

### GetImage

`func (o *V1EphemeralContainer) GetImage() string`

GetImage returns the Image field if non-nil, zero value otherwise.

### GetImageOk

`func (o *V1EphemeralContainer) GetImageOk() (*string, bool)`

GetImageOk returns a tuple with the Image field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetImage

`func (o *V1EphemeralContainer) SetImage(v string)`

SetImage sets Image field to given value.

### HasImage

`func (o *V1EphemeralContainer) HasImage() bool`

HasImage returns a boolean if a field has been set.

### GetImagePullPolicy

`func (o *V1EphemeralContainer) GetImagePullPolicy() string`

GetImagePullPolicy returns the ImagePullPolicy field if non-nil, zero value otherwise.

### GetImagePullPolicyOk

`func (o *V1EphemeralContainer) GetImagePullPolicyOk() (*string, bool)`

GetImagePullPolicyOk returns a tuple with the ImagePullPolicy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetImagePullPolicy

`func (o *V1EphemeralContainer) SetImagePullPolicy(v string)`

SetImagePullPolicy sets ImagePullPolicy field to given value.

### HasImagePullPolicy

`func (o *V1EphemeralContainer) HasImagePullPolicy() bool`

HasImagePullPolicy returns a boolean if a field has been set.

### GetLifecycle

`func (o *V1EphemeralContainer) GetLifecycle() V1Lifecycle`

GetLifecycle returns the Lifecycle field if non-nil, zero value otherwise.

### GetLifecycleOk

`func (o *V1EphemeralContainer) GetLifecycleOk() (*V1Lifecycle, bool)`

GetLifecycleOk returns a tuple with the Lifecycle field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLifecycle

`func (o *V1EphemeralContainer) SetLifecycle(v V1Lifecycle)`

SetLifecycle sets Lifecycle field to given value.

### HasLifecycle

`func (o *V1EphemeralContainer) HasLifecycle() bool`

HasLifecycle returns a boolean if a field has been set.

### GetLivenessProbe

`func (o *V1EphemeralContainer) GetLivenessProbe() V1Probe`

GetLivenessProbe returns the LivenessProbe field if non-nil, zero value otherwise.

### GetLivenessProbeOk

`func (o *V1EphemeralContainer) GetLivenessProbeOk() (*V1Probe, bool)`

GetLivenessProbeOk returns a tuple with the LivenessProbe field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLivenessProbe

`func (o *V1EphemeralContainer) SetLivenessProbe(v V1Probe)`

SetLivenessProbe sets LivenessProbe field to given value.

### HasLivenessProbe

`func (o *V1EphemeralContainer) HasLivenessProbe() bool`

HasLivenessProbe returns a boolean if a field has been set.

### GetName

`func (o *V1EphemeralContainer) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *V1EphemeralContainer) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *V1EphemeralContainer) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *V1EphemeralContainer) HasName() bool`

HasName returns a boolean if a field has been set.

### GetPorts

`func (o *V1EphemeralContainer) GetPorts() []V1ContainerPort`

GetPorts returns the Ports field if non-nil, zero value otherwise.

### GetPortsOk

`func (o *V1EphemeralContainer) GetPortsOk() (*[]V1ContainerPort, bool)`

GetPortsOk returns a tuple with the Ports field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPorts

`func (o *V1EphemeralContainer) SetPorts(v []V1ContainerPort)`

SetPorts sets Ports field to given value.

### HasPorts

`func (o *V1EphemeralContainer) HasPorts() bool`

HasPorts returns a boolean if a field has been set.

### GetReadinessProbe

`func (o *V1EphemeralContainer) GetReadinessProbe() V1Probe`

GetReadinessProbe returns the ReadinessProbe field if non-nil, zero value otherwise.

### GetReadinessProbeOk

`func (o *V1EphemeralContainer) GetReadinessProbeOk() (*V1Probe, bool)`

GetReadinessProbeOk returns a tuple with the ReadinessProbe field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReadinessProbe

`func (o *V1EphemeralContainer) SetReadinessProbe(v V1Probe)`

SetReadinessProbe sets ReadinessProbe field to given value.

### HasReadinessProbe

`func (o *V1EphemeralContainer) HasReadinessProbe() bool`

HasReadinessProbe returns a boolean if a field has been set.

### GetResizePolicy

`func (o *V1EphemeralContainer) GetResizePolicy() []V1ContainerResizePolicy`

GetResizePolicy returns the ResizePolicy field if non-nil, zero value otherwise.

### GetResizePolicyOk

`func (o *V1EphemeralContainer) GetResizePolicyOk() (*[]V1ContainerResizePolicy, bool)`

GetResizePolicyOk returns a tuple with the ResizePolicy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResizePolicy

`func (o *V1EphemeralContainer) SetResizePolicy(v []V1ContainerResizePolicy)`

SetResizePolicy sets ResizePolicy field to given value.

### HasResizePolicy

`func (o *V1EphemeralContainer) HasResizePolicy() bool`

HasResizePolicy returns a boolean if a field has been set.

### GetResources

`func (o *V1EphemeralContainer) GetResources() V1ResourceRequirements`

GetResources returns the Resources field if non-nil, zero value otherwise.

### GetResourcesOk

`func (o *V1EphemeralContainer) GetResourcesOk() (*V1ResourceRequirements, bool)`

GetResourcesOk returns a tuple with the Resources field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResources

`func (o *V1EphemeralContainer) SetResources(v V1ResourceRequirements)`

SetResources sets Resources field to given value.

### HasResources

`func (o *V1EphemeralContainer) HasResources() bool`

HasResources returns a boolean if a field has been set.

### GetRestartPolicy

`func (o *V1EphemeralContainer) GetRestartPolicy() string`

GetRestartPolicy returns the RestartPolicy field if non-nil, zero value otherwise.

### GetRestartPolicyOk

`func (o *V1EphemeralContainer) GetRestartPolicyOk() (*string, bool)`

GetRestartPolicyOk returns a tuple with the RestartPolicy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRestartPolicy

`func (o *V1EphemeralContainer) SetRestartPolicy(v string)`

SetRestartPolicy sets RestartPolicy field to given value.

### HasRestartPolicy

`func (o *V1EphemeralContainer) HasRestartPolicy() bool`

HasRestartPolicy returns a boolean if a field has been set.

### GetRestartPolicyRules

`func (o *V1EphemeralContainer) GetRestartPolicyRules() []V1ContainerRestartRule`

GetRestartPolicyRules returns the RestartPolicyRules field if non-nil, zero value otherwise.

### GetRestartPolicyRulesOk

`func (o *V1EphemeralContainer) GetRestartPolicyRulesOk() (*[]V1ContainerRestartRule, bool)`

GetRestartPolicyRulesOk returns a tuple with the RestartPolicyRules field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRestartPolicyRules

`func (o *V1EphemeralContainer) SetRestartPolicyRules(v []V1ContainerRestartRule)`

SetRestartPolicyRules sets RestartPolicyRules field to given value.

### HasRestartPolicyRules

`func (o *V1EphemeralContainer) HasRestartPolicyRules() bool`

HasRestartPolicyRules returns a boolean if a field has been set.

### GetSecurityContext

`func (o *V1EphemeralContainer) GetSecurityContext() V1SecurityContext`

GetSecurityContext returns the SecurityContext field if non-nil, zero value otherwise.

### GetSecurityContextOk

`func (o *V1EphemeralContainer) GetSecurityContextOk() (*V1SecurityContext, bool)`

GetSecurityContextOk returns a tuple with the SecurityContext field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecurityContext

`func (o *V1EphemeralContainer) SetSecurityContext(v V1SecurityContext)`

SetSecurityContext sets SecurityContext field to given value.

### HasSecurityContext

`func (o *V1EphemeralContainer) HasSecurityContext() bool`

HasSecurityContext returns a boolean if a field has been set.

### GetStartupProbe

`func (o *V1EphemeralContainer) GetStartupProbe() V1Probe`

GetStartupProbe returns the StartupProbe field if non-nil, zero value otherwise.

### GetStartupProbeOk

`func (o *V1EphemeralContainer) GetStartupProbeOk() (*V1Probe, bool)`

GetStartupProbeOk returns a tuple with the StartupProbe field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStartupProbe

`func (o *V1EphemeralContainer) SetStartupProbe(v V1Probe)`

SetStartupProbe sets StartupProbe field to given value.

### HasStartupProbe

`func (o *V1EphemeralContainer) HasStartupProbe() bool`

HasStartupProbe returns a boolean if a field has been set.

### GetStdin

`func (o *V1EphemeralContainer) GetStdin() bool`

GetStdin returns the Stdin field if non-nil, zero value otherwise.

### GetStdinOk

`func (o *V1EphemeralContainer) GetStdinOk() (*bool, bool)`

GetStdinOk returns a tuple with the Stdin field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStdin

`func (o *V1EphemeralContainer) SetStdin(v bool)`

SetStdin sets Stdin field to given value.

### HasStdin

`func (o *V1EphemeralContainer) HasStdin() bool`

HasStdin returns a boolean if a field has been set.

### GetStdinOnce

`func (o *V1EphemeralContainer) GetStdinOnce() bool`

GetStdinOnce returns the StdinOnce field if non-nil, zero value otherwise.

### GetStdinOnceOk

`func (o *V1EphemeralContainer) GetStdinOnceOk() (*bool, bool)`

GetStdinOnceOk returns a tuple with the StdinOnce field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStdinOnce

`func (o *V1EphemeralContainer) SetStdinOnce(v bool)`

SetStdinOnce sets StdinOnce field to given value.

### HasStdinOnce

`func (o *V1EphemeralContainer) HasStdinOnce() bool`

HasStdinOnce returns a boolean if a field has been set.

### GetTargetContainerName

`func (o *V1EphemeralContainer) GetTargetContainerName() string`

GetTargetContainerName returns the TargetContainerName field if non-nil, zero value otherwise.

### GetTargetContainerNameOk

`func (o *V1EphemeralContainer) GetTargetContainerNameOk() (*string, bool)`

GetTargetContainerNameOk returns a tuple with the TargetContainerName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTargetContainerName

`func (o *V1EphemeralContainer) SetTargetContainerName(v string)`

SetTargetContainerName sets TargetContainerName field to given value.

### HasTargetContainerName

`func (o *V1EphemeralContainer) HasTargetContainerName() bool`

HasTargetContainerName returns a boolean if a field has been set.

### GetTerminationMessagePath

`func (o *V1EphemeralContainer) GetTerminationMessagePath() string`

GetTerminationMessagePath returns the TerminationMessagePath field if non-nil, zero value otherwise.

### GetTerminationMessagePathOk

`func (o *V1EphemeralContainer) GetTerminationMessagePathOk() (*string, bool)`

GetTerminationMessagePathOk returns a tuple with the TerminationMessagePath field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTerminationMessagePath

`func (o *V1EphemeralContainer) SetTerminationMessagePath(v string)`

SetTerminationMessagePath sets TerminationMessagePath field to given value.

### HasTerminationMessagePath

`func (o *V1EphemeralContainer) HasTerminationMessagePath() bool`

HasTerminationMessagePath returns a boolean if a field has been set.

### GetTerminationMessagePolicy

`func (o *V1EphemeralContainer) GetTerminationMessagePolicy() string`

GetTerminationMessagePolicy returns the TerminationMessagePolicy field if non-nil, zero value otherwise.

### GetTerminationMessagePolicyOk

`func (o *V1EphemeralContainer) GetTerminationMessagePolicyOk() (*string, bool)`

GetTerminationMessagePolicyOk returns a tuple with the TerminationMessagePolicy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTerminationMessagePolicy

`func (o *V1EphemeralContainer) SetTerminationMessagePolicy(v string)`

SetTerminationMessagePolicy sets TerminationMessagePolicy field to given value.

### HasTerminationMessagePolicy

`func (o *V1EphemeralContainer) HasTerminationMessagePolicy() bool`

HasTerminationMessagePolicy returns a boolean if a field has been set.

### GetTty

`func (o *V1EphemeralContainer) GetTty() bool`

GetTty returns the Tty field if non-nil, zero value otherwise.

### GetTtyOk

`func (o *V1EphemeralContainer) GetTtyOk() (*bool, bool)`

GetTtyOk returns a tuple with the Tty field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTty

`func (o *V1EphemeralContainer) SetTty(v bool)`

SetTty sets Tty field to given value.

### HasTty

`func (o *V1EphemeralContainer) HasTty() bool`

HasTty returns a boolean if a field has been set.

### GetVolumeDevices

`func (o *V1EphemeralContainer) GetVolumeDevices() []V1VolumeDevice`

GetVolumeDevices returns the VolumeDevices field if non-nil, zero value otherwise.

### GetVolumeDevicesOk

`func (o *V1EphemeralContainer) GetVolumeDevicesOk() (*[]V1VolumeDevice, bool)`

GetVolumeDevicesOk returns a tuple with the VolumeDevices field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVolumeDevices

`func (o *V1EphemeralContainer) SetVolumeDevices(v []V1VolumeDevice)`

SetVolumeDevices sets VolumeDevices field to given value.

### HasVolumeDevices

`func (o *V1EphemeralContainer) HasVolumeDevices() bool`

HasVolumeDevices returns a boolean if a field has been set.

### GetVolumeMounts

`func (o *V1EphemeralContainer) GetVolumeMounts() []V1VolumeMount`

GetVolumeMounts returns the VolumeMounts field if non-nil, zero value otherwise.

### GetVolumeMountsOk

`func (o *V1EphemeralContainer) GetVolumeMountsOk() (*[]V1VolumeMount, bool)`

GetVolumeMountsOk returns a tuple with the VolumeMounts field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVolumeMounts

`func (o *V1EphemeralContainer) SetVolumeMounts(v []V1VolumeMount)`

SetVolumeMounts sets VolumeMounts field to given value.

### HasVolumeMounts

`func (o *V1EphemeralContainer) HasVolumeMounts() bool`

HasVolumeMounts returns a boolean if a field has been set.

### GetWorkingDir

`func (o *V1EphemeralContainer) GetWorkingDir() string`

GetWorkingDir returns the WorkingDir field if non-nil, zero value otherwise.

### GetWorkingDirOk

`func (o *V1EphemeralContainer) GetWorkingDirOk() (*string, bool)`

GetWorkingDirOk returns a tuple with the WorkingDir field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWorkingDir

`func (o *V1EphemeralContainer) SetWorkingDir(v string)`

SetWorkingDir sets WorkingDir field to given value.

### HasWorkingDir

`func (o *V1EphemeralContainer) HasWorkingDir() bool`

HasWorkingDir returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


