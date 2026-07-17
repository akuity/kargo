# V1SecurityContext

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**AllowPrivilegeEscalation** | Pointer to **bool** | AllowPrivilegeEscalation controls whether a process can gain more privileges than its parent process. This bool directly controls if the no_new_privs flag will be set on the container process. AllowPrivilegeEscalation is true always when the container is: 1) run as Privileged 2) has CAP_SYS_ADMIN Note that this field cannot be set when spec.os.name is windows. +optional | [optional] 
**AppArmorProfile** | Pointer to [**V1AppArmorProfile**](V1AppArmorProfile.md) | appArmorProfile is the AppArmor options to use by this container. If set, this profile overrides the pod&#39;s appArmorProfile. Note that this field cannot be set when spec.os.name is windows. +optional | [optional] 
**Capabilities** | Pointer to [**V1Capabilities**](V1Capabilities.md) | The capabilities to add/drop when running containers. Defaults to the default set of capabilities granted by the container runtime. Note that this field cannot be set when spec.os.name is windows. +optional | [optional] 
**Privileged** | Pointer to **bool** | Run container in privileged mode. Processes in privileged containers are essentially equivalent to root on the host. Defaults to false. Note that this field cannot be set when spec.os.name is windows. +optional | [optional] 
**ProcMount** | Pointer to **string** | procMount denotes the type of proc mount to use for the containers. The default value is Default which uses the container runtime defaults for readonly paths and masked paths. This requires the ProcMountType feature flag to be enabled. Note that this field cannot be set when spec.os.name is windows. +optional | [optional] 
**ReadOnlyRootFilesystem** | Pointer to **bool** | Whether this container has a read-only root filesystem. Default is false. Note that this field cannot be set when spec.os.name is windows. +optional | [optional] 
**RunAsGroup** | Pointer to **int32** | The GID to run the entrypoint of the container process. Uses runtime default if unset. May also be set in PodSecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. Note that this field cannot be set when spec.os.name is windows. +optional | [optional] 
**RunAsNonRoot** | Pointer to **bool** | Indicates that the container must run as a non-root user. If true, the Kubelet will validate the image at runtime to ensure that it does not run as UID 0 (root) and fail to start the container if it does. If unset or false, no such validation will be performed. May also be set in PodSecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. +optional | [optional] 
**RunAsUser** | Pointer to **int32** | The UID to run the entrypoint of the container process. Defaults to user specified in image metadata if unspecified. May also be set in PodSecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. Note that this field cannot be set when spec.os.name is windows. +optional | [optional] 
**SeLinuxOptions** | Pointer to [**V1SELinuxOptions**](V1SELinuxOptions.md) | The SELinux context to be applied to the container. If unspecified, the container runtime will allocate a random SELinux context for each container.  May also be set in PodSecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. Note that this field cannot be set when spec.os.name is windows. +optional | [optional] 
**SeccompProfile** | Pointer to [**V1SeccompProfile**](V1SeccompProfile.md) | The seccomp options to use by this container. If seccomp options are provided at both the pod &amp; container level, the container options override the pod options. Note that this field cannot be set when spec.os.name is windows. +optional | [optional] 
**WindowsOptions** | Pointer to [**V1WindowsSecurityContextOptions**](V1WindowsSecurityContextOptions.md) | The Windows specific settings applied to all containers. If unspecified, the options from the PodSecurityContext will be used. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. Note that this field cannot be set when spec.os.name is linux. +optional | [optional] 

## Methods

### NewV1SecurityContext

`func NewV1SecurityContext() *V1SecurityContext`

NewV1SecurityContext instantiates a new V1SecurityContext object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1SecurityContextWithDefaults

`func NewV1SecurityContextWithDefaults() *V1SecurityContext`

NewV1SecurityContextWithDefaults instantiates a new V1SecurityContext object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAllowPrivilegeEscalation

`func (o *V1SecurityContext) GetAllowPrivilegeEscalation() bool`

GetAllowPrivilegeEscalation returns the AllowPrivilegeEscalation field if non-nil, zero value otherwise.

### GetAllowPrivilegeEscalationOk

`func (o *V1SecurityContext) GetAllowPrivilegeEscalationOk() (*bool, bool)`

GetAllowPrivilegeEscalationOk returns a tuple with the AllowPrivilegeEscalation field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAllowPrivilegeEscalation

`func (o *V1SecurityContext) SetAllowPrivilegeEscalation(v bool)`

SetAllowPrivilegeEscalation sets AllowPrivilegeEscalation field to given value.

### HasAllowPrivilegeEscalation

`func (o *V1SecurityContext) HasAllowPrivilegeEscalation() bool`

HasAllowPrivilegeEscalation returns a boolean if a field has been set.

### GetAppArmorProfile

`func (o *V1SecurityContext) GetAppArmorProfile() V1AppArmorProfile`

GetAppArmorProfile returns the AppArmorProfile field if non-nil, zero value otherwise.

### GetAppArmorProfileOk

`func (o *V1SecurityContext) GetAppArmorProfileOk() (*V1AppArmorProfile, bool)`

GetAppArmorProfileOk returns a tuple with the AppArmorProfile field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAppArmorProfile

`func (o *V1SecurityContext) SetAppArmorProfile(v V1AppArmorProfile)`

SetAppArmorProfile sets AppArmorProfile field to given value.

### HasAppArmorProfile

`func (o *V1SecurityContext) HasAppArmorProfile() bool`

HasAppArmorProfile returns a boolean if a field has been set.

### GetCapabilities

`func (o *V1SecurityContext) GetCapabilities() V1Capabilities`

GetCapabilities returns the Capabilities field if non-nil, zero value otherwise.

### GetCapabilitiesOk

`func (o *V1SecurityContext) GetCapabilitiesOk() (*V1Capabilities, bool)`

GetCapabilitiesOk returns a tuple with the Capabilities field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCapabilities

`func (o *V1SecurityContext) SetCapabilities(v V1Capabilities)`

SetCapabilities sets Capabilities field to given value.

### HasCapabilities

`func (o *V1SecurityContext) HasCapabilities() bool`

HasCapabilities returns a boolean if a field has been set.

### GetPrivileged

`func (o *V1SecurityContext) GetPrivileged() bool`

GetPrivileged returns the Privileged field if non-nil, zero value otherwise.

### GetPrivilegedOk

`func (o *V1SecurityContext) GetPrivilegedOk() (*bool, bool)`

GetPrivilegedOk returns a tuple with the Privileged field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPrivileged

`func (o *V1SecurityContext) SetPrivileged(v bool)`

SetPrivileged sets Privileged field to given value.

### HasPrivileged

`func (o *V1SecurityContext) HasPrivileged() bool`

HasPrivileged returns a boolean if a field has been set.

### GetProcMount

`func (o *V1SecurityContext) GetProcMount() string`

GetProcMount returns the ProcMount field if non-nil, zero value otherwise.

### GetProcMountOk

`func (o *V1SecurityContext) GetProcMountOk() (*string, bool)`

GetProcMountOk returns a tuple with the ProcMount field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProcMount

`func (o *V1SecurityContext) SetProcMount(v string)`

SetProcMount sets ProcMount field to given value.

### HasProcMount

`func (o *V1SecurityContext) HasProcMount() bool`

HasProcMount returns a boolean if a field has been set.

### GetReadOnlyRootFilesystem

`func (o *V1SecurityContext) GetReadOnlyRootFilesystem() bool`

GetReadOnlyRootFilesystem returns the ReadOnlyRootFilesystem field if non-nil, zero value otherwise.

### GetReadOnlyRootFilesystemOk

`func (o *V1SecurityContext) GetReadOnlyRootFilesystemOk() (*bool, bool)`

GetReadOnlyRootFilesystemOk returns a tuple with the ReadOnlyRootFilesystem field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReadOnlyRootFilesystem

`func (o *V1SecurityContext) SetReadOnlyRootFilesystem(v bool)`

SetReadOnlyRootFilesystem sets ReadOnlyRootFilesystem field to given value.

### HasReadOnlyRootFilesystem

`func (o *V1SecurityContext) HasReadOnlyRootFilesystem() bool`

HasReadOnlyRootFilesystem returns a boolean if a field has been set.

### GetRunAsGroup

`func (o *V1SecurityContext) GetRunAsGroup() int32`

GetRunAsGroup returns the RunAsGroup field if non-nil, zero value otherwise.

### GetRunAsGroupOk

`func (o *V1SecurityContext) GetRunAsGroupOk() (*int32, bool)`

GetRunAsGroupOk returns a tuple with the RunAsGroup field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRunAsGroup

`func (o *V1SecurityContext) SetRunAsGroup(v int32)`

SetRunAsGroup sets RunAsGroup field to given value.

### HasRunAsGroup

`func (o *V1SecurityContext) HasRunAsGroup() bool`

HasRunAsGroup returns a boolean if a field has been set.

### GetRunAsNonRoot

`func (o *V1SecurityContext) GetRunAsNonRoot() bool`

GetRunAsNonRoot returns the RunAsNonRoot field if non-nil, zero value otherwise.

### GetRunAsNonRootOk

`func (o *V1SecurityContext) GetRunAsNonRootOk() (*bool, bool)`

GetRunAsNonRootOk returns a tuple with the RunAsNonRoot field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRunAsNonRoot

`func (o *V1SecurityContext) SetRunAsNonRoot(v bool)`

SetRunAsNonRoot sets RunAsNonRoot field to given value.

### HasRunAsNonRoot

`func (o *V1SecurityContext) HasRunAsNonRoot() bool`

HasRunAsNonRoot returns a boolean if a field has been set.

### GetRunAsUser

`func (o *V1SecurityContext) GetRunAsUser() int32`

GetRunAsUser returns the RunAsUser field if non-nil, zero value otherwise.

### GetRunAsUserOk

`func (o *V1SecurityContext) GetRunAsUserOk() (*int32, bool)`

GetRunAsUserOk returns a tuple with the RunAsUser field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRunAsUser

`func (o *V1SecurityContext) SetRunAsUser(v int32)`

SetRunAsUser sets RunAsUser field to given value.

### HasRunAsUser

`func (o *V1SecurityContext) HasRunAsUser() bool`

HasRunAsUser returns a boolean if a field has been set.

### GetSeLinuxOptions

`func (o *V1SecurityContext) GetSeLinuxOptions() V1SELinuxOptions`

GetSeLinuxOptions returns the SeLinuxOptions field if non-nil, zero value otherwise.

### GetSeLinuxOptionsOk

`func (o *V1SecurityContext) GetSeLinuxOptionsOk() (*V1SELinuxOptions, bool)`

GetSeLinuxOptionsOk returns a tuple with the SeLinuxOptions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSeLinuxOptions

`func (o *V1SecurityContext) SetSeLinuxOptions(v V1SELinuxOptions)`

SetSeLinuxOptions sets SeLinuxOptions field to given value.

### HasSeLinuxOptions

`func (o *V1SecurityContext) HasSeLinuxOptions() bool`

HasSeLinuxOptions returns a boolean if a field has been set.

### GetSeccompProfile

`func (o *V1SecurityContext) GetSeccompProfile() V1SeccompProfile`

GetSeccompProfile returns the SeccompProfile field if non-nil, zero value otherwise.

### GetSeccompProfileOk

`func (o *V1SecurityContext) GetSeccompProfileOk() (*V1SeccompProfile, bool)`

GetSeccompProfileOk returns a tuple with the SeccompProfile field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSeccompProfile

`func (o *V1SecurityContext) SetSeccompProfile(v V1SeccompProfile)`

SetSeccompProfile sets SeccompProfile field to given value.

### HasSeccompProfile

`func (o *V1SecurityContext) HasSeccompProfile() bool`

HasSeccompProfile returns a boolean if a field has been set.

### GetWindowsOptions

`func (o *V1SecurityContext) GetWindowsOptions() V1WindowsSecurityContextOptions`

GetWindowsOptions returns the WindowsOptions field if non-nil, zero value otherwise.

### GetWindowsOptionsOk

`func (o *V1SecurityContext) GetWindowsOptionsOk() (*V1WindowsSecurityContextOptions, bool)`

GetWindowsOptionsOk returns a tuple with the WindowsOptions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWindowsOptions

`func (o *V1SecurityContext) SetWindowsOptions(v V1WindowsSecurityContextOptions)`

SetWindowsOptions sets WindowsOptions field to given value.

### HasWindowsOptions

`func (o *V1SecurityContext) HasWindowsOptions() bool`

HasWindowsOptions returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


