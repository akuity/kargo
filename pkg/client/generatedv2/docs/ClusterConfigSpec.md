# ClusterConfigSpec

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**FreightLinks** | Pointer to [**[]DeepLink**](DeepLink.md) | FreightLinks defines deep links shown when viewing any Freight resource across all projects in the cluster. Project-level FreightLinks defined in ProjectConfig are shown in addition to these.  +optional | [optional] 
**GitClient** | Pointer to [**GitClientConfig**](GitClientConfig.md) | GitClient describes cluster-level configuration for Kargo&#39;s Git client, including committer identity and an optional signing key. If set, these values take precedence over any configuration provided at install time via the Helm chart. +optional | [optional] 
**StageLinks** | Pointer to [**[]DeepLink**](DeepLink.md) | StageLinks defines deep links shown when viewing any Stage resource across all projects in the cluster. Project-level StageLinks defined in ProjectConfig are shown in addition to these.  +optional | [optional] 
**WebhookReceivers** | Pointer to [**[]WebhookReceiverConfig**](WebhookReceiverConfig.md) | WebhookReceivers describes cluster-scoped webhook receivers used for processing events from various external platforms | [optional] 

## Methods

### NewClusterConfigSpec

`func NewClusterConfigSpec() *ClusterConfigSpec`

NewClusterConfigSpec instantiates a new ClusterConfigSpec object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewClusterConfigSpecWithDefaults

`func NewClusterConfigSpecWithDefaults() *ClusterConfigSpec`

NewClusterConfigSpecWithDefaults instantiates a new ClusterConfigSpec object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetFreightLinks

`func (o *ClusterConfigSpec) GetFreightLinks() []DeepLink`

GetFreightLinks returns the FreightLinks field if non-nil, zero value otherwise.

### GetFreightLinksOk

`func (o *ClusterConfigSpec) GetFreightLinksOk() (*[]DeepLink, bool)`

GetFreightLinksOk returns a tuple with the FreightLinks field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFreightLinks

`func (o *ClusterConfigSpec) SetFreightLinks(v []DeepLink)`

SetFreightLinks sets FreightLinks field to given value.

### HasFreightLinks

`func (o *ClusterConfigSpec) HasFreightLinks() bool`

HasFreightLinks returns a boolean if a field has been set.

### GetGitClient

`func (o *ClusterConfigSpec) GetGitClient() GitClientConfig`

GetGitClient returns the GitClient field if non-nil, zero value otherwise.

### GetGitClientOk

`func (o *ClusterConfigSpec) GetGitClientOk() (*GitClientConfig, bool)`

GetGitClientOk returns a tuple with the GitClient field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetGitClient

`func (o *ClusterConfigSpec) SetGitClient(v GitClientConfig)`

SetGitClient sets GitClient field to given value.

### HasGitClient

`func (o *ClusterConfigSpec) HasGitClient() bool`

HasGitClient returns a boolean if a field has been set.

### GetStageLinks

`func (o *ClusterConfigSpec) GetStageLinks() []DeepLink`

GetStageLinks returns the StageLinks field if non-nil, zero value otherwise.

### GetStageLinksOk

`func (o *ClusterConfigSpec) GetStageLinksOk() (*[]DeepLink, bool)`

GetStageLinksOk returns a tuple with the StageLinks field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStageLinks

`func (o *ClusterConfigSpec) SetStageLinks(v []DeepLink)`

SetStageLinks sets StageLinks field to given value.

### HasStageLinks

`func (o *ClusterConfigSpec) HasStageLinks() bool`

HasStageLinks returns a boolean if a field has been set.

### GetWebhookReceivers

`func (o *ClusterConfigSpec) GetWebhookReceivers() []WebhookReceiverConfig`

GetWebhookReceivers returns the WebhookReceivers field if non-nil, zero value otherwise.

### GetWebhookReceiversOk

`func (o *ClusterConfigSpec) GetWebhookReceiversOk() (*[]WebhookReceiverConfig, bool)`

GetWebhookReceiversOk returns a tuple with the WebhookReceivers field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWebhookReceivers

`func (o *ClusterConfigSpec) SetWebhookReceivers(v []WebhookReceiverConfig)`

SetWebhookReceivers sets WebhookReceivers field to given value.

### HasWebhookReceivers

`func (o *ClusterConfigSpec) HasWebhookReceivers() bool`

HasWebhookReceivers returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


