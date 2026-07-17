# GetConfigResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ArgocdShards** | Pointer to [**map[string]ArgoCDShard**](ArgoCDShard.md) |  | [optional] 
**HasAnalysisRunLogsUrlTemplate** | Pointer to **bool** |  | [optional] 
**KargoNamespace** | Pointer to **string** |  | [optional] 
**SecretManagementEnabled** | Pointer to **bool** |  | [optional] 
**SharedResourcesNamespace** | Pointer to **string** |  | [optional] 
**SystemResourcesNamespace** | Pointer to **string** |  | [optional] 

## Methods

### NewGetConfigResponse

`func NewGetConfigResponse() *GetConfigResponse`

NewGetConfigResponse instantiates a new GetConfigResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewGetConfigResponseWithDefaults

`func NewGetConfigResponseWithDefaults() *GetConfigResponse`

NewGetConfigResponseWithDefaults instantiates a new GetConfigResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetArgocdShards

`func (o *GetConfigResponse) GetArgocdShards() map[string]ArgoCDShard`

GetArgocdShards returns the ArgocdShards field if non-nil, zero value otherwise.

### GetArgocdShardsOk

`func (o *GetConfigResponse) GetArgocdShardsOk() (*map[string]ArgoCDShard, bool)`

GetArgocdShardsOk returns a tuple with the ArgocdShards field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetArgocdShards

`func (o *GetConfigResponse) SetArgocdShards(v map[string]ArgoCDShard)`

SetArgocdShards sets ArgocdShards field to given value.

### HasArgocdShards

`func (o *GetConfigResponse) HasArgocdShards() bool`

HasArgocdShards returns a boolean if a field has been set.

### GetHasAnalysisRunLogsUrlTemplate

`func (o *GetConfigResponse) GetHasAnalysisRunLogsUrlTemplate() bool`

GetHasAnalysisRunLogsUrlTemplate returns the HasAnalysisRunLogsUrlTemplate field if non-nil, zero value otherwise.

### GetHasAnalysisRunLogsUrlTemplateOk

`func (o *GetConfigResponse) GetHasAnalysisRunLogsUrlTemplateOk() (*bool, bool)`

GetHasAnalysisRunLogsUrlTemplateOk returns a tuple with the HasAnalysisRunLogsUrlTemplate field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHasAnalysisRunLogsUrlTemplate

`func (o *GetConfigResponse) SetHasAnalysisRunLogsUrlTemplate(v bool)`

SetHasAnalysisRunLogsUrlTemplate sets HasAnalysisRunLogsUrlTemplate field to given value.

### HasHasAnalysisRunLogsUrlTemplate

`func (o *GetConfigResponse) HasHasAnalysisRunLogsUrlTemplate() bool`

HasHasAnalysisRunLogsUrlTemplate returns a boolean if a field has been set.

### GetKargoNamespace

`func (o *GetConfigResponse) GetKargoNamespace() string`

GetKargoNamespace returns the KargoNamespace field if non-nil, zero value otherwise.

### GetKargoNamespaceOk

`func (o *GetConfigResponse) GetKargoNamespaceOk() (*string, bool)`

GetKargoNamespaceOk returns a tuple with the KargoNamespace field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKargoNamespace

`func (o *GetConfigResponse) SetKargoNamespace(v string)`

SetKargoNamespace sets KargoNamespace field to given value.

### HasKargoNamespace

`func (o *GetConfigResponse) HasKargoNamespace() bool`

HasKargoNamespace returns a boolean if a field has been set.

### GetSecretManagementEnabled

`func (o *GetConfigResponse) GetSecretManagementEnabled() bool`

GetSecretManagementEnabled returns the SecretManagementEnabled field if non-nil, zero value otherwise.

### GetSecretManagementEnabledOk

`func (o *GetConfigResponse) GetSecretManagementEnabledOk() (*bool, bool)`

GetSecretManagementEnabledOk returns a tuple with the SecretManagementEnabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecretManagementEnabled

`func (o *GetConfigResponse) SetSecretManagementEnabled(v bool)`

SetSecretManagementEnabled sets SecretManagementEnabled field to given value.

### HasSecretManagementEnabled

`func (o *GetConfigResponse) HasSecretManagementEnabled() bool`

HasSecretManagementEnabled returns a boolean if a field has been set.

### GetSharedResourcesNamespace

`func (o *GetConfigResponse) GetSharedResourcesNamespace() string`

GetSharedResourcesNamespace returns the SharedResourcesNamespace field if non-nil, zero value otherwise.

### GetSharedResourcesNamespaceOk

`func (o *GetConfigResponse) GetSharedResourcesNamespaceOk() (*string, bool)`

GetSharedResourcesNamespaceOk returns a tuple with the SharedResourcesNamespace field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSharedResourcesNamespace

`func (o *GetConfigResponse) SetSharedResourcesNamespace(v string)`

SetSharedResourcesNamespace sets SharedResourcesNamespace field to given value.

### HasSharedResourcesNamespace

`func (o *GetConfigResponse) HasSharedResourcesNamespace() bool`

HasSharedResourcesNamespace returns a boolean if a field has been set.

### GetSystemResourcesNamespace

`func (o *GetConfigResponse) GetSystemResourcesNamespace() string`

GetSystemResourcesNamespace returns the SystemResourcesNamespace field if non-nil, zero value otherwise.

### GetSystemResourcesNamespaceOk

`func (o *GetConfigResponse) GetSystemResourcesNamespaceOk() (*string, bool)`

GetSystemResourcesNamespaceOk returns a tuple with the SystemResourcesNamespace field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSystemResourcesNamespace

`func (o *GetConfigResponse) SetSystemResourcesNamespace(v string)`

SetSystemResourcesNamespace sets SystemResourcesNamespace field to given value.

### HasSystemResourcesNamespace

`func (o *GetConfigResponse) HasSystemResourcesNamespace() bool`

HasSystemResourcesNamespace returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


