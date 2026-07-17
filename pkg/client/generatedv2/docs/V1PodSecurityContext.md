# V1PodSecurityContext

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**AppArmorProfile** | Pointer to [**V1AppArmorProfile**](V1AppArmorProfile.md) | appArmorProfile is the AppArmor options to use by the containers in this pod. Note that this field cannot be set when spec.os.name is windows. +optional | [optional] 
**FsGroup** | Pointer to **int32** | A special supplemental group that applies to all containers in a pod. Some volume types allow the Kubelet to change the ownership of that volume to be owned by the pod:  1. The owning GID will be the FSGroup 2. The setgid bit is set (new files created in the volume will be owned by FSGroup) 3. The permission bits are OR&#39;d with rw-rw----  If unset, the Kubelet will not modify the ownership and permissions of any volume. Note that this field cannot be set when spec.os.name is windows. +optional | [optional] 
**FsGroupChangePolicy** | Pointer to **string** | fsGroupChangePolicy defines behavior of changing ownership and permission of the volume before being exposed inside Pod. This field will only apply to volume types which support fsGroup based ownership(and permissions). It will have no effect on ephemeral volume types such as: secret, configmaps and emptydir. Valid values are \&quot;OnRootMismatch\&quot; and \&quot;Always\&quot;. If not specified, \&quot;Always\&quot; is used. Note that this field cannot be set when spec.os.name is windows. +optional | [optional] 
**RunAsGroup** | Pointer to **int32** | The GID to run the entrypoint of the container process. Uses runtime default if unset. May also be set in SecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence for that container. Note that this field cannot be set when spec.os.name is windows. +optional | [optional] 
**RunAsNonRoot** | Pointer to **bool** | Indicates that the container must run as a non-root user. If true, the Kubelet will validate the image at runtime to ensure that it does not run as UID 0 (root) and fail to start the container if it does. If unset or false, no such validation will be performed. May also be set in SecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. +optional | [optional] 
**RunAsUser** | Pointer to **int32** | The UID to run the entrypoint of the container process. Defaults to user specified in image metadata if unspecified. May also be set in SecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence for that container. Note that this field cannot be set when spec.os.name is windows. +optional | [optional] 
**SeLinuxChangePolicy** | Pointer to **string** | seLinuxChangePolicy defines how the container&#39;s SELinux label is applied to all volumes used by the Pod. It has no effect on nodes that do not support SELinux or to volumes does not support SELinux. Valid values are \&quot;MountOption\&quot; and \&quot;Recursive\&quot;.  \&quot;Recursive\&quot; means relabeling of all files on all Pod volumes by the container runtime. This may be slow for large volumes, but allows mixing privileged and unprivileged Pods sharing the same volume on the same node.  \&quot;MountOption\&quot; mounts all eligible Pod volumes with &#x60;-o context&#x60; mount option. This requires all Pods that share the same volume to use the same SELinux label. It is not possible to share the same volume among privileged and unprivileged Pods. Eligible volumes are in-tree FibreChannel and iSCSI volumes, and all CSI volumes whose CSI driver announces SELinux support by setting spec.seLinuxMount: true in their CSIDriver instance. Other volumes are always re-labelled recursively. \&quot;MountOption\&quot; value is allowed only when SELinuxMount feature gate is enabled.  If not specified and SELinuxMount feature gate is enabled, \&quot;MountOption\&quot; is used. If not specified and SELinuxMount feature gate is disabled, \&quot;MountOption\&quot; is used for ReadWriteOncePod volumes and \&quot;Recursive\&quot; for all other volumes.  This field affects only Pods that have SELinux label set, either in PodSecurityContext or in SecurityContext of all containers.  All Pods that use the same volume should use the same seLinuxChangePolicy, otherwise some pods can get stuck in ContainerCreating state. Note that this field cannot be set when spec.os.name is windows. +featureGate&#x3D;SELinuxChangePolicy +optional | [optional] 
**SeLinuxOptions** | Pointer to [**V1SELinuxOptions**](V1SELinuxOptions.md) | The SELinux context to be applied to all containers. If unspecified, the container runtime will allocate a random SELinux context for each container.  May also be set in SecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence for that container. Note that this field cannot be set when spec.os.name is windows. +optional | [optional] 
**SeccompProfile** | Pointer to [**V1SeccompProfile**](V1SeccompProfile.md) | The seccomp options to use by the containers in this pod. Note that this field cannot be set when spec.os.name is windows. +optional | [optional] 
**SupplementalGroups** | Pointer to **[]int32** | A list of groups applied to the first process run in each container, in addition to the container&#39;s primary GID and fsGroup (if specified).  If the SupplementalGroupsPolicy feature is enabled, the supplementalGroupsPolicy field determines whether these are in addition to or instead of any group memberships defined in the container image. If unspecified, no additional groups are added, though group memberships defined in the container image may still be used, depending on the supplementalGroupsPolicy field. Note that this field cannot be set when spec.os.name is windows. +optional +listType&#x3D;atomic | [optional] 
**SupplementalGroupsPolicy** | Pointer to **string** | Defines how supplemental groups of the first container processes are calculated. Valid values are \&quot;Merge\&quot; and \&quot;Strict\&quot;. If not specified, \&quot;Merge\&quot; is used. (Alpha) Using the field requires the SupplementalGroupsPolicy feature gate to be enabled and the container runtime must implement support for this feature. Note that this field cannot be set when spec.os.name is windows. TODO: update the default value to \&quot;Merge\&quot; when spec.os.name is not windows in v1.34 +featureGate&#x3D;SupplementalGroupsPolicy +optional | [optional] 
**Sysctls** | Pointer to [**[]V1Sysctl**](V1Sysctl.md) | Sysctls hold a list of namespaced sysctls used for the pod. Pods with unsupported sysctls (by the container runtime) might fail to launch. Note that this field cannot be set when spec.os.name is windows. +optional +listType&#x3D;atomic | [optional] 
**WindowsOptions** | Pointer to [**V1WindowsSecurityContextOptions**](V1WindowsSecurityContextOptions.md) | The Windows specific settings applied to all containers. If unspecified, the options within a container&#39;s SecurityContext will be used. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. Note that this field cannot be set when spec.os.name is linux. +optional | [optional] 

## Methods

### NewV1PodSecurityContext

`func NewV1PodSecurityContext() *V1PodSecurityContext`

NewV1PodSecurityContext instantiates a new V1PodSecurityContext object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1PodSecurityContextWithDefaults

`func NewV1PodSecurityContextWithDefaults() *V1PodSecurityContext`

NewV1PodSecurityContextWithDefaults instantiates a new V1PodSecurityContext object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAppArmorProfile

`func (o *V1PodSecurityContext) GetAppArmorProfile() V1AppArmorProfile`

GetAppArmorProfile returns the AppArmorProfile field if non-nil, zero value otherwise.

### GetAppArmorProfileOk

`func (o *V1PodSecurityContext) GetAppArmorProfileOk() (*V1AppArmorProfile, bool)`

GetAppArmorProfileOk returns a tuple with the AppArmorProfile field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAppArmorProfile

`func (o *V1PodSecurityContext) SetAppArmorProfile(v V1AppArmorProfile)`

SetAppArmorProfile sets AppArmorProfile field to given value.

### HasAppArmorProfile

`func (o *V1PodSecurityContext) HasAppArmorProfile() bool`

HasAppArmorProfile returns a boolean if a field has been set.

### GetFsGroup

`func (o *V1PodSecurityContext) GetFsGroup() int32`

GetFsGroup returns the FsGroup field if non-nil, zero value otherwise.

### GetFsGroupOk

`func (o *V1PodSecurityContext) GetFsGroupOk() (*int32, bool)`

GetFsGroupOk returns a tuple with the FsGroup field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFsGroup

`func (o *V1PodSecurityContext) SetFsGroup(v int32)`

SetFsGroup sets FsGroup field to given value.

### HasFsGroup

`func (o *V1PodSecurityContext) HasFsGroup() bool`

HasFsGroup returns a boolean if a field has been set.

### GetFsGroupChangePolicy

`func (o *V1PodSecurityContext) GetFsGroupChangePolicy() string`

GetFsGroupChangePolicy returns the FsGroupChangePolicy field if non-nil, zero value otherwise.

### GetFsGroupChangePolicyOk

`func (o *V1PodSecurityContext) GetFsGroupChangePolicyOk() (*string, bool)`

GetFsGroupChangePolicyOk returns a tuple with the FsGroupChangePolicy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFsGroupChangePolicy

`func (o *V1PodSecurityContext) SetFsGroupChangePolicy(v string)`

SetFsGroupChangePolicy sets FsGroupChangePolicy field to given value.

### HasFsGroupChangePolicy

`func (o *V1PodSecurityContext) HasFsGroupChangePolicy() bool`

HasFsGroupChangePolicy returns a boolean if a field has been set.

### GetRunAsGroup

`func (o *V1PodSecurityContext) GetRunAsGroup() int32`

GetRunAsGroup returns the RunAsGroup field if non-nil, zero value otherwise.

### GetRunAsGroupOk

`func (o *V1PodSecurityContext) GetRunAsGroupOk() (*int32, bool)`

GetRunAsGroupOk returns a tuple with the RunAsGroup field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRunAsGroup

`func (o *V1PodSecurityContext) SetRunAsGroup(v int32)`

SetRunAsGroup sets RunAsGroup field to given value.

### HasRunAsGroup

`func (o *V1PodSecurityContext) HasRunAsGroup() bool`

HasRunAsGroup returns a boolean if a field has been set.

### GetRunAsNonRoot

`func (o *V1PodSecurityContext) GetRunAsNonRoot() bool`

GetRunAsNonRoot returns the RunAsNonRoot field if non-nil, zero value otherwise.

### GetRunAsNonRootOk

`func (o *V1PodSecurityContext) GetRunAsNonRootOk() (*bool, bool)`

GetRunAsNonRootOk returns a tuple with the RunAsNonRoot field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRunAsNonRoot

`func (o *V1PodSecurityContext) SetRunAsNonRoot(v bool)`

SetRunAsNonRoot sets RunAsNonRoot field to given value.

### HasRunAsNonRoot

`func (o *V1PodSecurityContext) HasRunAsNonRoot() bool`

HasRunAsNonRoot returns a boolean if a field has been set.

### GetRunAsUser

`func (o *V1PodSecurityContext) GetRunAsUser() int32`

GetRunAsUser returns the RunAsUser field if non-nil, zero value otherwise.

### GetRunAsUserOk

`func (o *V1PodSecurityContext) GetRunAsUserOk() (*int32, bool)`

GetRunAsUserOk returns a tuple with the RunAsUser field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRunAsUser

`func (o *V1PodSecurityContext) SetRunAsUser(v int32)`

SetRunAsUser sets RunAsUser field to given value.

### HasRunAsUser

`func (o *V1PodSecurityContext) HasRunAsUser() bool`

HasRunAsUser returns a boolean if a field has been set.

### GetSeLinuxChangePolicy

`func (o *V1PodSecurityContext) GetSeLinuxChangePolicy() string`

GetSeLinuxChangePolicy returns the SeLinuxChangePolicy field if non-nil, zero value otherwise.

### GetSeLinuxChangePolicyOk

`func (o *V1PodSecurityContext) GetSeLinuxChangePolicyOk() (*string, bool)`

GetSeLinuxChangePolicyOk returns a tuple with the SeLinuxChangePolicy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSeLinuxChangePolicy

`func (o *V1PodSecurityContext) SetSeLinuxChangePolicy(v string)`

SetSeLinuxChangePolicy sets SeLinuxChangePolicy field to given value.

### HasSeLinuxChangePolicy

`func (o *V1PodSecurityContext) HasSeLinuxChangePolicy() bool`

HasSeLinuxChangePolicy returns a boolean if a field has been set.

### GetSeLinuxOptions

`func (o *V1PodSecurityContext) GetSeLinuxOptions() V1SELinuxOptions`

GetSeLinuxOptions returns the SeLinuxOptions field if non-nil, zero value otherwise.

### GetSeLinuxOptionsOk

`func (o *V1PodSecurityContext) GetSeLinuxOptionsOk() (*V1SELinuxOptions, bool)`

GetSeLinuxOptionsOk returns a tuple with the SeLinuxOptions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSeLinuxOptions

`func (o *V1PodSecurityContext) SetSeLinuxOptions(v V1SELinuxOptions)`

SetSeLinuxOptions sets SeLinuxOptions field to given value.

### HasSeLinuxOptions

`func (o *V1PodSecurityContext) HasSeLinuxOptions() bool`

HasSeLinuxOptions returns a boolean if a field has been set.

### GetSeccompProfile

`func (o *V1PodSecurityContext) GetSeccompProfile() V1SeccompProfile`

GetSeccompProfile returns the SeccompProfile field if non-nil, zero value otherwise.

### GetSeccompProfileOk

`func (o *V1PodSecurityContext) GetSeccompProfileOk() (*V1SeccompProfile, bool)`

GetSeccompProfileOk returns a tuple with the SeccompProfile field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSeccompProfile

`func (o *V1PodSecurityContext) SetSeccompProfile(v V1SeccompProfile)`

SetSeccompProfile sets SeccompProfile field to given value.

### HasSeccompProfile

`func (o *V1PodSecurityContext) HasSeccompProfile() bool`

HasSeccompProfile returns a boolean if a field has been set.

### GetSupplementalGroups

`func (o *V1PodSecurityContext) GetSupplementalGroups() []int32`

GetSupplementalGroups returns the SupplementalGroups field if non-nil, zero value otherwise.

### GetSupplementalGroupsOk

`func (o *V1PodSecurityContext) GetSupplementalGroupsOk() (*[]int32, bool)`

GetSupplementalGroupsOk returns a tuple with the SupplementalGroups field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSupplementalGroups

`func (o *V1PodSecurityContext) SetSupplementalGroups(v []int32)`

SetSupplementalGroups sets SupplementalGroups field to given value.

### HasSupplementalGroups

`func (o *V1PodSecurityContext) HasSupplementalGroups() bool`

HasSupplementalGroups returns a boolean if a field has been set.

### GetSupplementalGroupsPolicy

`func (o *V1PodSecurityContext) GetSupplementalGroupsPolicy() string`

GetSupplementalGroupsPolicy returns the SupplementalGroupsPolicy field if non-nil, zero value otherwise.

### GetSupplementalGroupsPolicyOk

`func (o *V1PodSecurityContext) GetSupplementalGroupsPolicyOk() (*string, bool)`

GetSupplementalGroupsPolicyOk returns a tuple with the SupplementalGroupsPolicy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSupplementalGroupsPolicy

`func (o *V1PodSecurityContext) SetSupplementalGroupsPolicy(v string)`

SetSupplementalGroupsPolicy sets SupplementalGroupsPolicy field to given value.

### HasSupplementalGroupsPolicy

`func (o *V1PodSecurityContext) HasSupplementalGroupsPolicy() bool`

HasSupplementalGroupsPolicy returns a boolean if a field has been set.

### GetSysctls

`func (o *V1PodSecurityContext) GetSysctls() []V1Sysctl`

GetSysctls returns the Sysctls field if non-nil, zero value otherwise.

### GetSysctlsOk

`func (o *V1PodSecurityContext) GetSysctlsOk() (*[]V1Sysctl, bool)`

GetSysctlsOk returns a tuple with the Sysctls field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSysctls

`func (o *V1PodSecurityContext) SetSysctls(v []V1Sysctl)`

SetSysctls sets Sysctls field to given value.

### HasSysctls

`func (o *V1PodSecurityContext) HasSysctls() bool`

HasSysctls returns a boolean if a field has been set.

### GetWindowsOptions

`func (o *V1PodSecurityContext) GetWindowsOptions() V1WindowsSecurityContextOptions`

GetWindowsOptions returns the WindowsOptions field if non-nil, zero value otherwise.

### GetWindowsOptionsOk

`func (o *V1PodSecurityContext) GetWindowsOptionsOk() (*V1WindowsSecurityContextOptions, bool)`

GetWindowsOptionsOk returns a tuple with the WindowsOptions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWindowsOptions

`func (o *V1PodSecurityContext) SetWindowsOptions(v V1WindowsSecurityContextOptions)`

SetWindowsOptions sets WindowsOptions field to given value.

### HasWindowsOptions

`func (o *V1PodSecurityContext) HasWindowsOptions() bool`

HasWindowsOptions returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


