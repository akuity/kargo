# GenericWebhookReceiverConfig

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Actions** | Pointer to [**[]GenericWebhookAction**](GenericWebhookAction.md) | Actions is a list of actions to be performed when a webhook event is received.  +kubebuilder:validation:MinItems&#x3D;1 | [optional] 
**SecretRef** | [**V1LocalObjectReference**](V1LocalObjectReference.md) | SecretRef contains a reference to a Secret. For Project-scoped webhook receivers, the referenced Secret must be in the same namespace as the ProjectConfig.  For cluster-scoped webhook receivers, the referenced Secret must be in the designated \&quot;system resources\&quot; namespace.  The Secret&#39;s data map is expected to contain a &#x60;secret&#x60; key whose value does NOT need to be shared directly with the sender. It is used only by Kargo to create a complex, hard-to-guess URL, which implicitly serves as a shared secret.  +kubebuilder:validation:Required | 

## Methods

### NewGenericWebhookReceiverConfig

`func NewGenericWebhookReceiverConfig(secretRef V1LocalObjectReference, ) *GenericWebhookReceiverConfig`

NewGenericWebhookReceiverConfig instantiates a new GenericWebhookReceiverConfig object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewGenericWebhookReceiverConfigWithDefaults

`func NewGenericWebhookReceiverConfigWithDefaults() *GenericWebhookReceiverConfig`

NewGenericWebhookReceiverConfigWithDefaults instantiates a new GenericWebhookReceiverConfig object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetActions

`func (o *GenericWebhookReceiverConfig) GetActions() []GenericWebhookAction`

GetActions returns the Actions field if non-nil, zero value otherwise.

### GetActionsOk

`func (o *GenericWebhookReceiverConfig) GetActionsOk() (*[]GenericWebhookAction, bool)`

GetActionsOk returns a tuple with the Actions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetActions

`func (o *GenericWebhookReceiverConfig) SetActions(v []GenericWebhookAction)`

SetActions sets Actions field to given value.

### HasActions

`func (o *GenericWebhookReceiverConfig) HasActions() bool`

HasActions returns a boolean if a field has been set.

### GetSecretRef

`func (o *GenericWebhookReceiverConfig) GetSecretRef() V1LocalObjectReference`

GetSecretRef returns the SecretRef field if non-nil, zero value otherwise.

### GetSecretRefOk

`func (o *GenericWebhookReceiverConfig) GetSecretRefOk() (*V1LocalObjectReference, bool)`

GetSecretRefOk returns a tuple with the SecretRef field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecretRef

`func (o *GenericWebhookReceiverConfig) SetSecretRef(v V1LocalObjectReference)`

SetSecretRef sets SecretRef field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


