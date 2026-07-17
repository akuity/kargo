# ProjectConfigSpec

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**FreightLinks** | Pointer to [**[]DeepLink**](DeepLink.md) | FreightLinks defines deep links shown when viewing Freight resources within this project. These are shown in addition to any cluster-level FreightLinks defined in ClusterConfig.  +optional | [optional] 
**PromotionPolicies** | Pointer to [**[]PromotionPolicy**](PromotionPolicy.md) | PromotionPolicies defines policies governing the promotion of Freight to specific Stages within the Project. | [optional] 
**StageLinks** | Pointer to [**[]DeepLink**](DeepLink.md) | StageLinks defines deep links shown when viewing Stage resources within this project. These are shown in addition to any cluster-level StageLinks defined in ClusterConfig.  +optional | [optional] 
**WebhookReceivers** | Pointer to [**[]WebhookReceiverConfig**](WebhookReceiverConfig.md) | WebhookReceivers describes Project-specific webhook receivers used for processing events from various external platforms | [optional] 

## Methods

### NewProjectConfigSpec

`func NewProjectConfigSpec() *ProjectConfigSpec`

NewProjectConfigSpec instantiates a new ProjectConfigSpec object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewProjectConfigSpecWithDefaults

`func NewProjectConfigSpecWithDefaults() *ProjectConfigSpec`

NewProjectConfigSpecWithDefaults instantiates a new ProjectConfigSpec object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetFreightLinks

`func (o *ProjectConfigSpec) GetFreightLinks() []DeepLink`

GetFreightLinks returns the FreightLinks field if non-nil, zero value otherwise.

### GetFreightLinksOk

`func (o *ProjectConfigSpec) GetFreightLinksOk() (*[]DeepLink, bool)`

GetFreightLinksOk returns a tuple with the FreightLinks field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFreightLinks

`func (o *ProjectConfigSpec) SetFreightLinks(v []DeepLink)`

SetFreightLinks sets FreightLinks field to given value.

### HasFreightLinks

`func (o *ProjectConfigSpec) HasFreightLinks() bool`

HasFreightLinks returns a boolean if a field has been set.

### GetPromotionPolicies

`func (o *ProjectConfigSpec) GetPromotionPolicies() []PromotionPolicy`

GetPromotionPolicies returns the PromotionPolicies field if non-nil, zero value otherwise.

### GetPromotionPoliciesOk

`func (o *ProjectConfigSpec) GetPromotionPoliciesOk() (*[]PromotionPolicy, bool)`

GetPromotionPoliciesOk returns a tuple with the PromotionPolicies field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPromotionPolicies

`func (o *ProjectConfigSpec) SetPromotionPolicies(v []PromotionPolicy)`

SetPromotionPolicies sets PromotionPolicies field to given value.

### HasPromotionPolicies

`func (o *ProjectConfigSpec) HasPromotionPolicies() bool`

HasPromotionPolicies returns a boolean if a field has been set.

### GetStageLinks

`func (o *ProjectConfigSpec) GetStageLinks() []DeepLink`

GetStageLinks returns the StageLinks field if non-nil, zero value otherwise.

### GetStageLinksOk

`func (o *ProjectConfigSpec) GetStageLinksOk() (*[]DeepLink, bool)`

GetStageLinksOk returns a tuple with the StageLinks field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStageLinks

`func (o *ProjectConfigSpec) SetStageLinks(v []DeepLink)`

SetStageLinks sets StageLinks field to given value.

### HasStageLinks

`func (o *ProjectConfigSpec) HasStageLinks() bool`

HasStageLinks returns a boolean if a field has been set.

### GetWebhookReceivers

`func (o *ProjectConfigSpec) GetWebhookReceivers() []WebhookReceiverConfig`

GetWebhookReceivers returns the WebhookReceivers field if non-nil, zero value otherwise.

### GetWebhookReceiversOk

`func (o *ProjectConfigSpec) GetWebhookReceiversOk() (*[]WebhookReceiverConfig, bool)`

GetWebhookReceiversOk returns a tuple with the WebhookReceivers field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWebhookReceivers

`func (o *ProjectConfigSpec) SetWebhookReceivers(v []WebhookReceiverConfig)`

SetWebhookReceivers sets WebhookReceivers field to given value.

### HasWebhookReceivers

`func (o *ProjectConfigSpec) HasWebhookReceivers() bool`

HasWebhookReceivers returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


