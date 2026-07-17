# GitLabWebhookReceiverConfig

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**SecretRef** | [**V1LocalObjectReference**](V1LocalObjectReference.md) | SecretRef contains a reference to a Secret. For Project-scoped webhook receivers, the referenced Secret must be in the same namespace as the ProjectConfig.  For cluster-scoped webhook receivers, the referenced Secret must be in the designated \&quot;system resources\&quot; namespace.  The secret is expected to contain a &#x60;secret-token&#x60; key containing the shared secret specified when registering the webhook in GitLab. For more information about this token, please refer to the GitLab documentation:   https://docs.gitlab.com/user/project/integrations/webhooks/  +kubebuilder:validation:Required | 

## Methods

### NewGitLabWebhookReceiverConfig

`func NewGitLabWebhookReceiverConfig(secretRef V1LocalObjectReference, ) *GitLabWebhookReceiverConfig`

NewGitLabWebhookReceiverConfig instantiates a new GitLabWebhookReceiverConfig object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewGitLabWebhookReceiverConfigWithDefaults

`func NewGitLabWebhookReceiverConfigWithDefaults() *GitLabWebhookReceiverConfig`

NewGitLabWebhookReceiverConfigWithDefaults instantiates a new GitLabWebhookReceiverConfig object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSecretRef

`func (o *GitLabWebhookReceiverConfig) GetSecretRef() V1LocalObjectReference`

GetSecretRef returns the SecretRef field if non-nil, zero value otherwise.

### GetSecretRefOk

`func (o *GitLabWebhookReceiverConfig) GetSecretRefOk() (*V1LocalObjectReference, bool)`

GetSecretRefOk returns a tuple with the SecretRef field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecretRef

`func (o *GitLabWebhookReceiverConfig) SetSecretRef(v V1LocalObjectReference)`

SetSecretRef sets SecretRef field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


