# DiscoveredCommit

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Author** | Pointer to **string** | Author is the author of the commit. | [optional] 
**Branch** | Pointer to **string** | Branch is the branch in which the commit was found. This field is optional, and populated based on the CommitSelectionStrategy of the GitSubscription. | [optional] 
**Committer** | Pointer to **string** | Committer is the person who committed the commit. | [optional] 
**CreatorDate** | Pointer to **string** | CreatorDate is the commit creation date as specified by the commit, or the tagger date if the commit belongs to an annotated tag. | [optional] 
**Id** | Pointer to **string** | ID is the identifier of the commit. This typically is a SHA-1 hash.  +kubebuilder:validation:MinLength&#x3D;1 | [optional] 
**Subject** | Pointer to **string** | Subject is the subject of the commit (i.e. the first line of the commit message). | [optional] 
**Tag** | Pointer to **string** | Tag is the tag that resolved to this commit. This field is optional, and populated based on the CommitSelectionStrategy of the GitSubscription. | [optional] 

## Methods

### NewDiscoveredCommit

`func NewDiscoveredCommit() *DiscoveredCommit`

NewDiscoveredCommit instantiates a new DiscoveredCommit object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewDiscoveredCommitWithDefaults

`func NewDiscoveredCommitWithDefaults() *DiscoveredCommit`

NewDiscoveredCommitWithDefaults instantiates a new DiscoveredCommit object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAuthor

`func (o *DiscoveredCommit) GetAuthor() string`

GetAuthor returns the Author field if non-nil, zero value otherwise.

### GetAuthorOk

`func (o *DiscoveredCommit) GetAuthorOk() (*string, bool)`

GetAuthorOk returns a tuple with the Author field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAuthor

`func (o *DiscoveredCommit) SetAuthor(v string)`

SetAuthor sets Author field to given value.

### HasAuthor

`func (o *DiscoveredCommit) HasAuthor() bool`

HasAuthor returns a boolean if a field has been set.

### GetBranch

`func (o *DiscoveredCommit) GetBranch() string`

GetBranch returns the Branch field if non-nil, zero value otherwise.

### GetBranchOk

`func (o *DiscoveredCommit) GetBranchOk() (*string, bool)`

GetBranchOk returns a tuple with the Branch field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBranch

`func (o *DiscoveredCommit) SetBranch(v string)`

SetBranch sets Branch field to given value.

### HasBranch

`func (o *DiscoveredCommit) HasBranch() bool`

HasBranch returns a boolean if a field has been set.

### GetCommitter

`func (o *DiscoveredCommit) GetCommitter() string`

GetCommitter returns the Committer field if non-nil, zero value otherwise.

### GetCommitterOk

`func (o *DiscoveredCommit) GetCommitterOk() (*string, bool)`

GetCommitterOk returns a tuple with the Committer field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCommitter

`func (o *DiscoveredCommit) SetCommitter(v string)`

SetCommitter sets Committer field to given value.

### HasCommitter

`func (o *DiscoveredCommit) HasCommitter() bool`

HasCommitter returns a boolean if a field has been set.

### GetCreatorDate

`func (o *DiscoveredCommit) GetCreatorDate() string`

GetCreatorDate returns the CreatorDate field if non-nil, zero value otherwise.

### GetCreatorDateOk

`func (o *DiscoveredCommit) GetCreatorDateOk() (*string, bool)`

GetCreatorDateOk returns a tuple with the CreatorDate field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatorDate

`func (o *DiscoveredCommit) SetCreatorDate(v string)`

SetCreatorDate sets CreatorDate field to given value.

### HasCreatorDate

`func (o *DiscoveredCommit) HasCreatorDate() bool`

HasCreatorDate returns a boolean if a field has been set.

### GetId

`func (o *DiscoveredCommit) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *DiscoveredCommit) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *DiscoveredCommit) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *DiscoveredCommit) HasId() bool`

HasId returns a boolean if a field has been set.

### GetSubject

`func (o *DiscoveredCommit) GetSubject() string`

GetSubject returns the Subject field if non-nil, zero value otherwise.

### GetSubjectOk

`func (o *DiscoveredCommit) GetSubjectOk() (*string, bool)`

GetSubjectOk returns a tuple with the Subject field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSubject

`func (o *DiscoveredCommit) SetSubject(v string)`

SetSubject sets Subject field to given value.

### HasSubject

`func (o *DiscoveredCommit) HasSubject() bool`

HasSubject returns a boolean if a field has been set.

### GetTag

`func (o *DiscoveredCommit) GetTag() string`

GetTag returns the Tag field if non-nil, zero value otherwise.

### GetTagOk

`func (o *DiscoveredCommit) GetTagOk() (*string, bool)`

GetTagOk returns a tuple with the Tag field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTag

`func (o *DiscoveredCommit) SetTag(v string)`

SetTag sets Tag field to given value.

### HasTag

`func (o *DiscoveredCommit) HasTag() bool`

HasTag returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


