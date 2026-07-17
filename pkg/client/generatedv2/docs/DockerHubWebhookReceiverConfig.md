# DockerHubWebhookReceiverConfig

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**SecretRef** | [**V1LocalObjectReference**](V1LocalObjectReference.md) | SecretRef contains a reference to a Secret. For Project-scoped webhook receivers, the referenced Secret must be in the same namespace as the ProjectConfig.  The Secret&#39;s data map is expected to contain a &#x60;secret&#x60; key whose value does NOT need to be shared directly with Docker Hub when registering a webhook. It is used only by Kargo to create a complex, hard-to-guess URL, which implicitly serves as a shared secret. For more information about Docker Hub webhooks, please refer to the Docker documentation:   https://docs.docker.com/docker-hub/webhooks/  +kubebuilder:validation:Required | 

## Methods

### NewDockerHubWebhookReceiverConfig

`func NewDockerHubWebhookReceiverConfig(secretRef V1LocalObjectReference, ) *DockerHubWebhookReceiverConfig`

NewDockerHubWebhookReceiverConfig instantiates a new DockerHubWebhookReceiverConfig object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewDockerHubWebhookReceiverConfigWithDefaults

`func NewDockerHubWebhookReceiverConfigWithDefaults() *DockerHubWebhookReceiverConfig`

NewDockerHubWebhookReceiverConfigWithDefaults instantiates a new DockerHubWebhookReceiverConfig object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSecretRef

`func (o *DockerHubWebhookReceiverConfig) GetSecretRef() V1LocalObjectReference`

GetSecretRef returns the SecretRef field if non-nil, zero value otherwise.

### GetSecretRefOk

`func (o *DockerHubWebhookReceiverConfig) GetSecretRefOk() (*V1LocalObjectReference, bool)`

GetSecretRefOk returns a tuple with the SecretRef field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecretRef

`func (o *DockerHubWebhookReceiverConfig) SetSecretRef(v V1LocalObjectReference)`

SetSecretRef sets SecretRef field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


