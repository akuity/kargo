# VersionInfo

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**BuildDate** | Pointer to **string** | BuildDate is the date/time on which the application was built. | [optional] 
**Compiler** | Pointer to **string** | Compiler indicates what Go compiler was used for the build. | [optional] 
**GitCommit** | Pointer to **string** | GitCommit is the ID (sha) of the last commit to the application&#39;s source code that is included in this build. | [optional] 
**GitTreeDirty** | Pointer to **bool** | GitTreeDirty is true if the application&#39;s source code contained uncommitted changes at the time it was built; otherwise it is false. | [optional] 
**GoVersion** | Pointer to **string** | GoVersion is the version of Go that was used to build the application. | [optional] 
**Platform** | Pointer to **string** | Platform indicates the OS and CPU architecture for which the application was built. | [optional] 
**Version** | Pointer to **string** | Version is a human-friendly version string. | [optional] 

## Methods

### NewVersionInfo

`func NewVersionInfo() *VersionInfo`

NewVersionInfo instantiates a new VersionInfo object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewVersionInfoWithDefaults

`func NewVersionInfoWithDefaults() *VersionInfo`

NewVersionInfoWithDefaults instantiates a new VersionInfo object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetBuildDate

`func (o *VersionInfo) GetBuildDate() string`

GetBuildDate returns the BuildDate field if non-nil, zero value otherwise.

### GetBuildDateOk

`func (o *VersionInfo) GetBuildDateOk() (*string, bool)`

GetBuildDateOk returns a tuple with the BuildDate field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBuildDate

`func (o *VersionInfo) SetBuildDate(v string)`

SetBuildDate sets BuildDate field to given value.

### HasBuildDate

`func (o *VersionInfo) HasBuildDate() bool`

HasBuildDate returns a boolean if a field has been set.

### GetCompiler

`func (o *VersionInfo) GetCompiler() string`

GetCompiler returns the Compiler field if non-nil, zero value otherwise.

### GetCompilerOk

`func (o *VersionInfo) GetCompilerOk() (*string, bool)`

GetCompilerOk returns a tuple with the Compiler field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCompiler

`func (o *VersionInfo) SetCompiler(v string)`

SetCompiler sets Compiler field to given value.

### HasCompiler

`func (o *VersionInfo) HasCompiler() bool`

HasCompiler returns a boolean if a field has been set.

### GetGitCommit

`func (o *VersionInfo) GetGitCommit() string`

GetGitCommit returns the GitCommit field if non-nil, zero value otherwise.

### GetGitCommitOk

`func (o *VersionInfo) GetGitCommitOk() (*string, bool)`

GetGitCommitOk returns a tuple with the GitCommit field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetGitCommit

`func (o *VersionInfo) SetGitCommit(v string)`

SetGitCommit sets GitCommit field to given value.

### HasGitCommit

`func (o *VersionInfo) HasGitCommit() bool`

HasGitCommit returns a boolean if a field has been set.

### GetGitTreeDirty

`func (o *VersionInfo) GetGitTreeDirty() bool`

GetGitTreeDirty returns the GitTreeDirty field if non-nil, zero value otherwise.

### GetGitTreeDirtyOk

`func (o *VersionInfo) GetGitTreeDirtyOk() (*bool, bool)`

GetGitTreeDirtyOk returns a tuple with the GitTreeDirty field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetGitTreeDirty

`func (o *VersionInfo) SetGitTreeDirty(v bool)`

SetGitTreeDirty sets GitTreeDirty field to given value.

### HasGitTreeDirty

`func (o *VersionInfo) HasGitTreeDirty() bool`

HasGitTreeDirty returns a boolean if a field has been set.

### GetGoVersion

`func (o *VersionInfo) GetGoVersion() string`

GetGoVersion returns the GoVersion field if non-nil, zero value otherwise.

### GetGoVersionOk

`func (o *VersionInfo) GetGoVersionOk() (*string, bool)`

GetGoVersionOk returns a tuple with the GoVersion field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetGoVersion

`func (o *VersionInfo) SetGoVersion(v string)`

SetGoVersion sets GoVersion field to given value.

### HasGoVersion

`func (o *VersionInfo) HasGoVersion() bool`

HasGoVersion returns a boolean if a field has been set.

### GetPlatform

`func (o *VersionInfo) GetPlatform() string`

GetPlatform returns the Platform field if non-nil, zero value otherwise.

### GetPlatformOk

`func (o *VersionInfo) GetPlatformOk() (*string, bool)`

GetPlatformOk returns a tuple with the Platform field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPlatform

`func (o *VersionInfo) SetPlatform(v string)`

SetPlatform sets Platform field to given value.

### HasPlatform

`func (o *VersionInfo) HasPlatform() bool`

HasPlatform returns a boolean if a field has been set.

### GetVersion

`func (o *VersionInfo) GetVersion() string`

GetVersion returns the Version field if non-nil, zero value otherwise.

### GetVersionOk

`func (o *VersionInfo) GetVersionOk() (*string, bool)`

GetVersionOk returns a tuple with the Version field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVersion

`func (o *VersionInfo) SetVersion(v string)`

SetVersion sets Version field to given value.

### HasVersion

`func (o *VersionInfo) HasVersion() bool`

HasVersion returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


