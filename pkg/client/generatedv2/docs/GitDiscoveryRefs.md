# GitDiscoveryRefs

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**BranchHead** | Pointer to **string** | BranchHead is the unfiltered commit ID at the tip of the subscribed branch. It is populated for branch-based selection strategies (NewestFromBranch). Because it records the branch tip before any path filtering, an unchanged value guarantees the path-filtered selection cannot have changed either.  +optional | [optional] 
**Tags** | Pointer to [**[]DiscoveredRef**](DiscoveredRef.md) | Tags is the set of tags that satisfied the GitSubscription&#39;s name-based filters (semver and/or regex), paired with the commit IDs they reference, sorted by tag name for a stable comparison. It is populated for tag-based selection strategies (NewestTag, SemVer, Lexical). Path filtering is applied later, during selection, and does not affect this set.  +optional | [optional] 

## Methods

### NewGitDiscoveryRefs

`func NewGitDiscoveryRefs() *GitDiscoveryRefs`

NewGitDiscoveryRefs instantiates a new GitDiscoveryRefs object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewGitDiscoveryRefsWithDefaults

`func NewGitDiscoveryRefsWithDefaults() *GitDiscoveryRefs`

NewGitDiscoveryRefsWithDefaults instantiates a new GitDiscoveryRefs object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetBranchHead

`func (o *GitDiscoveryRefs) GetBranchHead() string`

GetBranchHead returns the BranchHead field if non-nil, zero value otherwise.

### GetBranchHeadOk

`func (o *GitDiscoveryRefs) GetBranchHeadOk() (*string, bool)`

GetBranchHeadOk returns a tuple with the BranchHead field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBranchHead

`func (o *GitDiscoveryRefs) SetBranchHead(v string)`

SetBranchHead sets BranchHead field to given value.

### HasBranchHead

`func (o *GitDiscoveryRefs) HasBranchHead() bool`

HasBranchHead returns a boolean if a field has been set.

### GetTags

`func (o *GitDiscoveryRefs) GetTags() []DiscoveredRef`

GetTags returns the Tags field if non-nil, zero value otherwise.

### GetTagsOk

`func (o *GitDiscoveryRefs) GetTagsOk() (*[]DiscoveredRef, bool)`

GetTagsOk returns a tuple with the Tags field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTags

`func (o *GitDiscoveryRefs) SetTags(v []DiscoveredRef)`

SetTags sets Tags field to given value.

### HasTags

`func (o *GitDiscoveryRefs) HasTags() bool`

HasTags returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


