# ArtifactoryWebhookReceiverConfig

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**SecretRef** | [**V1LocalObjectReference**](V1LocalObjectReference.md) | SecretRef contains a reference to a Secret. For Project-scoped webhook receivers, the referenced Secret must be in the same namespace as the ProjectConfig.  For cluster-scoped webhook receivers, the referenced Secret must be in the designated \&quot;system resources\&quot; namespace.  The Secret&#39;s data map is expected to contain a &#x60;secret-token&#x60; key whose value is the shared secret used to authenticate the webhook requests sent by JFrog Artifactory. For more information please refer to the JFrog Artifactory documentation:   https://jfrog.com/help/r/jfrog-platform-administration-documentation/webhooks  +kubebuilder:validation:Required | 
**VirtualRepoName** | Pointer to **string** | VirtualRepoName is the name of an Artifactory virtual repository.  When unspecified, the Artifactory webhook receiver depends on the value of the webhook payload&#39;s &#x60;data.repo_key&#x60; field when inferring the URL of the repository from which the webhook originated, which will always be an Artifactory \&quot;local repository.\&quot; In cases where a Warehouse subscribes to such a repository indirectly via a \&quot;virtual repository,\&quot; there will be a discrepancy between the inferred (local) repository URL and the URL actually used by the subscription, which can prevent the receiver from identifying such a Warehouse as one in need of refreshing. When specified, the value of the VirtualRepoName field supersedes the value of the webhook payload&#39;s &#x60;data.repo_key&#x60; field to compensate for that discrepancy.  In practice, when using virtual repositories, a separate Artifactory webhook receiver should be configured for each, but one such receiver can handle inbound webhooks from any number of local repositories that are aggregated by that virtual repository. For example, if a virtual repository &#x60;proj-virtual&#x60; aggregates container images from all of the &#x60;proj&#x60; Artifactory project&#39;s local image repositories, with a single webhook configured to post to a single receiver configured for the &#x60;proj-virtual&#x60; virtual repository, an image pushed to &#x60;example.frog.io/proj-&lt;local-repo-name&gt;/&lt;path&gt;/image&#x60;, will cause that receiver to refresh all Warehouses subscribed to &#x60;example.frog.io/proj-virtual/&lt;path&gt;/image&#x60;.  +optional | [optional] 

## Methods

### NewArtifactoryWebhookReceiverConfig

`func NewArtifactoryWebhookReceiverConfig(secretRef V1LocalObjectReference, ) *ArtifactoryWebhookReceiverConfig`

NewArtifactoryWebhookReceiverConfig instantiates a new ArtifactoryWebhookReceiverConfig object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewArtifactoryWebhookReceiverConfigWithDefaults

`func NewArtifactoryWebhookReceiverConfigWithDefaults() *ArtifactoryWebhookReceiverConfig`

NewArtifactoryWebhookReceiverConfigWithDefaults instantiates a new ArtifactoryWebhookReceiverConfig object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSecretRef

`func (o *ArtifactoryWebhookReceiverConfig) GetSecretRef() V1LocalObjectReference`

GetSecretRef returns the SecretRef field if non-nil, zero value otherwise.

### GetSecretRefOk

`func (o *ArtifactoryWebhookReceiverConfig) GetSecretRefOk() (*V1LocalObjectReference, bool)`

GetSecretRefOk returns a tuple with the SecretRef field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecretRef

`func (o *ArtifactoryWebhookReceiverConfig) SetSecretRef(v V1LocalObjectReference)`

SetSecretRef sets SecretRef field to given value.


### GetVirtualRepoName

`func (o *ArtifactoryWebhookReceiverConfig) GetVirtualRepoName() string`

GetVirtualRepoName returns the VirtualRepoName field if non-nil, zero value otherwise.

### GetVirtualRepoNameOk

`func (o *ArtifactoryWebhookReceiverConfig) GetVirtualRepoNameOk() (*string, bool)`

GetVirtualRepoNameOk returns a tuple with the VirtualRepoName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVirtualRepoName

`func (o *ArtifactoryWebhookReceiverConfig) SetVirtualRepoName(v string)`

SetVirtualRepoName sets VirtualRepoName field to given value.

### HasVirtualRepoName

`func (o *ArtifactoryWebhookReceiverConfig) HasVirtualRepoName() bool`

HasVirtualRepoName returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


