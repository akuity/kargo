# GitDiscoveryResult

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Commits** | Pointer to [**[]DiscoveredCommit**](DiscoveredCommit.md) | Commits is a list of commits discovered by the Warehouse for the GitSubscription. An empty list indicates that the discovery operation was successful, but no commits matching the GitSubscription criteria were found.  +optional | [optional] 
**ObservedRefs** | Pointer to [**GitDiscoveryRefs**](GitDiscoveryRefs.md) | ObservedRefs records the raw remote ref state observed at the most recent successful discovery, after name-based filtering but before path filtering or commit selection. The Warehouse uses it to short-circuit discovery: at the start of a reconcile, a single git ls-remote call yields the current ref state, and if it matches this field, nothing relevant has moved and the previously selected Commits remain valid -- so an expensive clone and history walk can be skipped entirely. This field is optional; when absent (e.g. on a Warehouse that predates this feature), discovery falls through to a full clone and repopulates it.  +optional | [optional] 
**RepoURL** | Pointer to **string** | RepoURL is the repository URL of the GitSubscription.  TODO(v1.13.0): Remove SSH/SCP-style URL support from this pattern.  +kubebuilder:validation:MinLength&#x3D;1 +kubebuilder:validation:Pattern&#x3D;&#x60;(?:^(ssh|https?)://(?:([\\w-]+)(:(.+))?@)?([\\w-]+(?:\\.[\\w-]+)*)(?::(\\d{1,5}))?(/.*)$)|(?:^([\\w-]+)@([\\w+]+(?:\\.[\\w-]+)*):(/?.*))&#x60; +akuity:test-kubebuilder-pattern&#x3D;GitRepoURLPattern | [optional] 

## Methods

### NewGitDiscoveryResult

`func NewGitDiscoveryResult() *GitDiscoveryResult`

NewGitDiscoveryResult instantiates a new GitDiscoveryResult object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewGitDiscoveryResultWithDefaults

`func NewGitDiscoveryResultWithDefaults() *GitDiscoveryResult`

NewGitDiscoveryResultWithDefaults instantiates a new GitDiscoveryResult object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetCommits

`func (o *GitDiscoveryResult) GetCommits() []DiscoveredCommit`

GetCommits returns the Commits field if non-nil, zero value otherwise.

### GetCommitsOk

`func (o *GitDiscoveryResult) GetCommitsOk() (*[]DiscoveredCommit, bool)`

GetCommitsOk returns a tuple with the Commits field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCommits

`func (o *GitDiscoveryResult) SetCommits(v []DiscoveredCommit)`

SetCommits sets Commits field to given value.

### HasCommits

`func (o *GitDiscoveryResult) HasCommits() bool`

HasCommits returns a boolean if a field has been set.

### GetObservedRefs

`func (o *GitDiscoveryResult) GetObservedRefs() GitDiscoveryRefs`

GetObservedRefs returns the ObservedRefs field if non-nil, zero value otherwise.

### GetObservedRefsOk

`func (o *GitDiscoveryResult) GetObservedRefsOk() (*GitDiscoveryRefs, bool)`

GetObservedRefsOk returns a tuple with the ObservedRefs field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetObservedRefs

`func (o *GitDiscoveryResult) SetObservedRefs(v GitDiscoveryRefs)`

SetObservedRefs sets ObservedRefs field to given value.

### HasObservedRefs

`func (o *GitDiscoveryResult) HasObservedRefs() bool`

HasObservedRefs returns a boolean if a field has been set.

### GetRepoURL

`func (o *GitDiscoveryResult) GetRepoURL() string`

GetRepoURL returns the RepoURL field if non-nil, zero value otherwise.

### GetRepoURLOk

`func (o *GitDiscoveryResult) GetRepoURLOk() (*string, bool)`

GetRepoURLOk returns a tuple with the RepoURL field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRepoURL

`func (o *GitDiscoveryResult) SetRepoURL(v string)`

SetRepoURL sets RepoURL field to given value.

### HasRepoURL

`func (o *GitDiscoveryResult) HasRepoURL() bool`

HasRepoURL returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


