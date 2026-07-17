# GitCommit

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Author** | Pointer to **string** | Author is the author of the commit. | [optional] 
**Branch** | Pointer to **string** | Branch denotes the branch of the repository where this commit was found. | [optional] 
**Committer** | Pointer to **string** | Committer is the person who committed the commit. | [optional] 
**Id** | Pointer to **string** | ID is the ID of a specific commit in the Git repository specified by RepoURL. | [optional] 
**Message** | Pointer to **string** | Message is the message associated with the commit. At present, this only contains the first line (subject) of the commit message. | [optional] 
**RepoURL** | Pointer to **string** | RepoURL is the URL of a Git repository. | [optional] 
**Tag** | Pointer to **string** | Tag denotes a tag in the repository that matched selection criteria and resolved to this commit. | [optional] 

## Methods

### NewGitCommit

`func NewGitCommit() *GitCommit`

NewGitCommit instantiates a new GitCommit object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewGitCommitWithDefaults

`func NewGitCommitWithDefaults() *GitCommit`

NewGitCommitWithDefaults instantiates a new GitCommit object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAuthor

`func (o *GitCommit) GetAuthor() string`

GetAuthor returns the Author field if non-nil, zero value otherwise.

### GetAuthorOk

`func (o *GitCommit) GetAuthorOk() (*string, bool)`

GetAuthorOk returns a tuple with the Author field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAuthor

`func (o *GitCommit) SetAuthor(v string)`

SetAuthor sets Author field to given value.

### HasAuthor

`func (o *GitCommit) HasAuthor() bool`

HasAuthor returns a boolean if a field has been set.

### GetBranch

`func (o *GitCommit) GetBranch() string`

GetBranch returns the Branch field if non-nil, zero value otherwise.

### GetBranchOk

`func (o *GitCommit) GetBranchOk() (*string, bool)`

GetBranchOk returns a tuple with the Branch field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBranch

`func (o *GitCommit) SetBranch(v string)`

SetBranch sets Branch field to given value.

### HasBranch

`func (o *GitCommit) HasBranch() bool`

HasBranch returns a boolean if a field has been set.

### GetCommitter

`func (o *GitCommit) GetCommitter() string`

GetCommitter returns the Committer field if non-nil, zero value otherwise.

### GetCommitterOk

`func (o *GitCommit) GetCommitterOk() (*string, bool)`

GetCommitterOk returns a tuple with the Committer field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCommitter

`func (o *GitCommit) SetCommitter(v string)`

SetCommitter sets Committer field to given value.

### HasCommitter

`func (o *GitCommit) HasCommitter() bool`

HasCommitter returns a boolean if a field has been set.

### GetId

`func (o *GitCommit) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *GitCommit) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *GitCommit) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *GitCommit) HasId() bool`

HasId returns a boolean if a field has been set.

### GetMessage

`func (o *GitCommit) GetMessage() string`

GetMessage returns the Message field if non-nil, zero value otherwise.

### GetMessageOk

`func (o *GitCommit) GetMessageOk() (*string, bool)`

GetMessageOk returns a tuple with the Message field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMessage

`func (o *GitCommit) SetMessage(v string)`

SetMessage sets Message field to given value.

### HasMessage

`func (o *GitCommit) HasMessage() bool`

HasMessage returns a boolean if a field has been set.

### GetRepoURL

`func (o *GitCommit) GetRepoURL() string`

GetRepoURL returns the RepoURL field if non-nil, zero value otherwise.

### GetRepoURLOk

`func (o *GitCommit) GetRepoURLOk() (*string, bool)`

GetRepoURLOk returns a tuple with the RepoURL field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRepoURL

`func (o *GitCommit) SetRepoURL(v string)`

SetRepoURL sets RepoURL field to given value.

### HasRepoURL

`func (o *GitCommit) HasRepoURL() bool`

HasRepoURL returns a boolean if a field has been set.

### GetTag

`func (o *GitCommit) GetTag() string`

GetTag returns the Tag field if non-nil, zero value otherwise.

### GetTagOk

`func (o *GitCommit) GetTagOk() (*string, bool)`

GetTagOk returns a tuple with the Tag field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTag

`func (o *GitCommit) SetTag(v string)`

SetTag sets Tag field to given value.

### HasTag

`func (o *GitCommit) HasTag() bool`

HasTag returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


