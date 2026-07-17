# WebhookReceiverConfig

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Artifactory** | Pointer to [**ArtifactoryWebhookReceiverConfig**](ArtifactoryWebhookReceiverConfig.md) | Artifactory contains the configuration for a webhook receiver that is compatible with JFrog Artifactory payloads. | [optional] 
**Azure** | Pointer to [**AzureWebhookReceiverConfig**](AzureWebhookReceiverConfig.md) | Azure contains the configuration for a webhook receiver that is compatible with Azure Container Registry (ACR) and Azure DevOps payloads. | [optional] 
**Bitbucket** | Pointer to [**BitbucketWebhookReceiverConfig**](BitbucketWebhookReceiverConfig.md) | Bitbucket contains the configuration for a webhook receiver that is compatible with Bitbucket payloads. | [optional] 
**Dockerhub** | Pointer to [**DockerHubWebhookReceiverConfig**](DockerHubWebhookReceiverConfig.md) | DockerHub contains the configuration for a webhook receiver that is compatible with DockerHub payloads. | [optional] 
**Generic** | Pointer to [**GenericWebhookReceiverConfig**](GenericWebhookReceiverConfig.md) | Generic contains the configuration for a generic webhook receiver. | [optional] 
**Gitea** | Pointer to [**GiteaWebhookReceiverConfig**](GiteaWebhookReceiverConfig.md) | Gitea contains the configuration for a webhook receiver that is compatible with Gitea payloads. | [optional] 
**Github** | Pointer to [**GitHubWebhookReceiverConfig**](GitHubWebhookReceiverConfig.md) | GitHub contains the configuration for a webhook receiver that is compatible with GitHub payloads. | [optional] 
**Gitlab** | Pointer to [**GitLabWebhookReceiverConfig**](GitLabWebhookReceiverConfig.md) | GitLab contains the configuration for a webhook receiver that is compatible with GitLab payloads. | [optional] 
**Harbor** | Pointer to [**HarborWebhookReceiverConfig**](HarborWebhookReceiverConfig.md) | Harbor contains the configuration for a webhook receiver that is compatible with Harbor payloads. | [optional] 
**Name** | **string** | Name is the name of the webhook receiver.  +kubebuilder:validation:Required +kubebuilder:validation:MinLength&#x3D;1 +kubebuilder:validation:MaxLength&#x3D;253 +kubebuilder:validation:Pattern&#x3D;&#x60;^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$&#x60; +akuity:test-kubebuilder-pattern&#x3D;KubernetesName | 
**Quay** | Pointer to [**QuayWebhookReceiverConfig**](QuayWebhookReceiverConfig.md) | Quay contains the configuration for a webhook receiver that is compatible with Quay payloads. | [optional] 

## Methods

### NewWebhookReceiverConfig

`func NewWebhookReceiverConfig(name string, ) *WebhookReceiverConfig`

NewWebhookReceiverConfig instantiates a new WebhookReceiverConfig object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewWebhookReceiverConfigWithDefaults

`func NewWebhookReceiverConfigWithDefaults() *WebhookReceiverConfig`

NewWebhookReceiverConfigWithDefaults instantiates a new WebhookReceiverConfig object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetArtifactory

`func (o *WebhookReceiverConfig) GetArtifactory() ArtifactoryWebhookReceiverConfig`

GetArtifactory returns the Artifactory field if non-nil, zero value otherwise.

### GetArtifactoryOk

`func (o *WebhookReceiverConfig) GetArtifactoryOk() (*ArtifactoryWebhookReceiverConfig, bool)`

GetArtifactoryOk returns a tuple with the Artifactory field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetArtifactory

`func (o *WebhookReceiverConfig) SetArtifactory(v ArtifactoryWebhookReceiverConfig)`

SetArtifactory sets Artifactory field to given value.

### HasArtifactory

`func (o *WebhookReceiverConfig) HasArtifactory() bool`

HasArtifactory returns a boolean if a field has been set.

### GetAzure

`func (o *WebhookReceiverConfig) GetAzure() AzureWebhookReceiverConfig`

GetAzure returns the Azure field if non-nil, zero value otherwise.

### GetAzureOk

`func (o *WebhookReceiverConfig) GetAzureOk() (*AzureWebhookReceiverConfig, bool)`

GetAzureOk returns a tuple with the Azure field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAzure

`func (o *WebhookReceiverConfig) SetAzure(v AzureWebhookReceiverConfig)`

SetAzure sets Azure field to given value.

### HasAzure

`func (o *WebhookReceiverConfig) HasAzure() bool`

HasAzure returns a boolean if a field has been set.

### GetBitbucket

`func (o *WebhookReceiverConfig) GetBitbucket() BitbucketWebhookReceiverConfig`

GetBitbucket returns the Bitbucket field if non-nil, zero value otherwise.

### GetBitbucketOk

`func (o *WebhookReceiverConfig) GetBitbucketOk() (*BitbucketWebhookReceiverConfig, bool)`

GetBitbucketOk returns a tuple with the Bitbucket field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBitbucket

`func (o *WebhookReceiverConfig) SetBitbucket(v BitbucketWebhookReceiverConfig)`

SetBitbucket sets Bitbucket field to given value.

### HasBitbucket

`func (o *WebhookReceiverConfig) HasBitbucket() bool`

HasBitbucket returns a boolean if a field has been set.

### GetDockerhub

`func (o *WebhookReceiverConfig) GetDockerhub() DockerHubWebhookReceiverConfig`

GetDockerhub returns the Dockerhub field if non-nil, zero value otherwise.

### GetDockerhubOk

`func (o *WebhookReceiverConfig) GetDockerhubOk() (*DockerHubWebhookReceiverConfig, bool)`

GetDockerhubOk returns a tuple with the Dockerhub field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDockerhub

`func (o *WebhookReceiverConfig) SetDockerhub(v DockerHubWebhookReceiverConfig)`

SetDockerhub sets Dockerhub field to given value.

### HasDockerhub

`func (o *WebhookReceiverConfig) HasDockerhub() bool`

HasDockerhub returns a boolean if a field has been set.

### GetGeneric

`func (o *WebhookReceiverConfig) GetGeneric() GenericWebhookReceiverConfig`

GetGeneric returns the Generic field if non-nil, zero value otherwise.

### GetGenericOk

`func (o *WebhookReceiverConfig) GetGenericOk() (*GenericWebhookReceiverConfig, bool)`

GetGenericOk returns a tuple with the Generic field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetGeneric

`func (o *WebhookReceiverConfig) SetGeneric(v GenericWebhookReceiverConfig)`

SetGeneric sets Generic field to given value.

### HasGeneric

`func (o *WebhookReceiverConfig) HasGeneric() bool`

HasGeneric returns a boolean if a field has been set.

### GetGitea

`func (o *WebhookReceiverConfig) GetGitea() GiteaWebhookReceiverConfig`

GetGitea returns the Gitea field if non-nil, zero value otherwise.

### GetGiteaOk

`func (o *WebhookReceiverConfig) GetGiteaOk() (*GiteaWebhookReceiverConfig, bool)`

GetGiteaOk returns a tuple with the Gitea field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetGitea

`func (o *WebhookReceiverConfig) SetGitea(v GiteaWebhookReceiverConfig)`

SetGitea sets Gitea field to given value.

### HasGitea

`func (o *WebhookReceiverConfig) HasGitea() bool`

HasGitea returns a boolean if a field has been set.

### GetGithub

`func (o *WebhookReceiverConfig) GetGithub() GitHubWebhookReceiverConfig`

GetGithub returns the Github field if non-nil, zero value otherwise.

### GetGithubOk

`func (o *WebhookReceiverConfig) GetGithubOk() (*GitHubWebhookReceiverConfig, bool)`

GetGithubOk returns a tuple with the Github field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetGithub

`func (o *WebhookReceiverConfig) SetGithub(v GitHubWebhookReceiverConfig)`

SetGithub sets Github field to given value.

### HasGithub

`func (o *WebhookReceiverConfig) HasGithub() bool`

HasGithub returns a boolean if a field has been set.

### GetGitlab

`func (o *WebhookReceiverConfig) GetGitlab() GitLabWebhookReceiverConfig`

GetGitlab returns the Gitlab field if non-nil, zero value otherwise.

### GetGitlabOk

`func (o *WebhookReceiverConfig) GetGitlabOk() (*GitLabWebhookReceiverConfig, bool)`

GetGitlabOk returns a tuple with the Gitlab field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetGitlab

`func (o *WebhookReceiverConfig) SetGitlab(v GitLabWebhookReceiverConfig)`

SetGitlab sets Gitlab field to given value.

### HasGitlab

`func (o *WebhookReceiverConfig) HasGitlab() bool`

HasGitlab returns a boolean if a field has been set.

### GetHarbor

`func (o *WebhookReceiverConfig) GetHarbor() HarborWebhookReceiverConfig`

GetHarbor returns the Harbor field if non-nil, zero value otherwise.

### GetHarborOk

`func (o *WebhookReceiverConfig) GetHarborOk() (*HarborWebhookReceiverConfig, bool)`

GetHarborOk returns a tuple with the Harbor field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHarbor

`func (o *WebhookReceiverConfig) SetHarbor(v HarborWebhookReceiverConfig)`

SetHarbor sets Harbor field to given value.

### HasHarbor

`func (o *WebhookReceiverConfig) HasHarbor() bool`

HasHarbor returns a boolean if a field has been set.

### GetName

`func (o *WebhookReceiverConfig) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *WebhookReceiverConfig) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *WebhookReceiverConfig) SetName(v string)`

SetName sets Name field to given value.


### GetQuay

`func (o *WebhookReceiverConfig) GetQuay() QuayWebhookReceiverConfig`

GetQuay returns the Quay field if non-nil, zero value otherwise.

### GetQuayOk

`func (o *WebhookReceiverConfig) GetQuayOk() (*QuayWebhookReceiverConfig, bool)`

GetQuayOk returns a tuple with the Quay field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetQuay

`func (o *WebhookReceiverConfig) SetQuay(v QuayWebhookReceiverConfig)`

SetQuay sets Quay field to given value.

### HasQuay

`func (o *WebhookReceiverConfig) HasQuay() bool`

HasQuay returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


