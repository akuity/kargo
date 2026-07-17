# V1GitRepoVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Directory** | Pointer to **string** | directory is the target directory name. Must not contain or start with &#39;..&#39;.  If &#39;.&#39; is supplied, the volume directory will be the git repository.  Otherwise, if specified, the volume will contain the git repository in the subdirectory with the given name. +optional | [optional] 
**Repository** | Pointer to **string** | repository is the URL | [optional] 
**Revision** | Pointer to **string** | revision is the commit hash for the specified revision. +optional | [optional] 

## Methods

### NewV1GitRepoVolumeSource

`func NewV1GitRepoVolumeSource() *V1GitRepoVolumeSource`

NewV1GitRepoVolumeSource instantiates a new V1GitRepoVolumeSource object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1GitRepoVolumeSourceWithDefaults

`func NewV1GitRepoVolumeSourceWithDefaults() *V1GitRepoVolumeSource`

NewV1GitRepoVolumeSourceWithDefaults instantiates a new V1GitRepoVolumeSource object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDirectory

`func (o *V1GitRepoVolumeSource) GetDirectory() string`

GetDirectory returns the Directory field if non-nil, zero value otherwise.

### GetDirectoryOk

`func (o *V1GitRepoVolumeSource) GetDirectoryOk() (*string, bool)`

GetDirectoryOk returns a tuple with the Directory field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDirectory

`func (o *V1GitRepoVolumeSource) SetDirectory(v string)`

SetDirectory sets Directory field to given value.

### HasDirectory

`func (o *V1GitRepoVolumeSource) HasDirectory() bool`

HasDirectory returns a boolean if a field has been set.

### GetRepository

`func (o *V1GitRepoVolumeSource) GetRepository() string`

GetRepository returns the Repository field if non-nil, zero value otherwise.

### GetRepositoryOk

`func (o *V1GitRepoVolumeSource) GetRepositoryOk() (*string, bool)`

GetRepositoryOk returns a tuple with the Repository field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRepository

`func (o *V1GitRepoVolumeSource) SetRepository(v string)`

SetRepository sets Repository field to given value.

### HasRepository

`func (o *V1GitRepoVolumeSource) HasRepository() bool`

HasRepository returns a boolean if a field has been set.

### GetRevision

`func (o *V1GitRepoVolumeSource) GetRevision() string`

GetRevision returns the Revision field if non-nil, zero value otherwise.

### GetRevisionOk

`func (o *V1GitRepoVolumeSource) GetRevisionOk() (*string, bool)`

GetRevisionOk returns a tuple with the Revision field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRevision

`func (o *V1GitRepoVolumeSource) SetRevision(v string)`

SetRevision sets Revision field to given value.

### HasRevision

`func (o *V1GitRepoVolumeSource) HasRevision() bool`

HasRevision returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


