# RolloutsKayentaMetric

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Address** | Pointer to **string** |  | [optional] 
**Application** | Pointer to **string** |  | [optional] 
**CanaryConfigName** | Pointer to **string** |  | [optional] 
**ConfigurationAccountName** | Pointer to **string** |  | [optional] 
**Lookback** | Pointer to **bool** |  | [optional] 
**MetricsAccountName** | Pointer to **string** |  | [optional] 
**Scopes** | Pointer to [**[]RolloutsKayentaScope**](RolloutsKayentaScope.md) |  | [optional] 
**StorageAccountName** | Pointer to **string** |  | [optional] 
**Threshold** | Pointer to [**RolloutsKayentaThreshold**](RolloutsKayentaThreshold.md) |  | [optional] 

## Methods

### NewRolloutsKayentaMetric

`func NewRolloutsKayentaMetric() *RolloutsKayentaMetric`

NewRolloutsKayentaMetric instantiates a new RolloutsKayentaMetric object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRolloutsKayentaMetricWithDefaults

`func NewRolloutsKayentaMetricWithDefaults() *RolloutsKayentaMetric`

NewRolloutsKayentaMetricWithDefaults instantiates a new RolloutsKayentaMetric object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAddress

`func (o *RolloutsKayentaMetric) GetAddress() string`

GetAddress returns the Address field if non-nil, zero value otherwise.

### GetAddressOk

`func (o *RolloutsKayentaMetric) GetAddressOk() (*string, bool)`

GetAddressOk returns a tuple with the Address field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAddress

`func (o *RolloutsKayentaMetric) SetAddress(v string)`

SetAddress sets Address field to given value.

### HasAddress

`func (o *RolloutsKayentaMetric) HasAddress() bool`

HasAddress returns a boolean if a field has been set.

### GetApplication

`func (o *RolloutsKayentaMetric) GetApplication() string`

GetApplication returns the Application field if non-nil, zero value otherwise.

### GetApplicationOk

`func (o *RolloutsKayentaMetric) GetApplicationOk() (*string, bool)`

GetApplicationOk returns a tuple with the Application field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetApplication

`func (o *RolloutsKayentaMetric) SetApplication(v string)`

SetApplication sets Application field to given value.

### HasApplication

`func (o *RolloutsKayentaMetric) HasApplication() bool`

HasApplication returns a boolean if a field has been set.

### GetCanaryConfigName

`func (o *RolloutsKayentaMetric) GetCanaryConfigName() string`

GetCanaryConfigName returns the CanaryConfigName field if non-nil, zero value otherwise.

### GetCanaryConfigNameOk

`func (o *RolloutsKayentaMetric) GetCanaryConfigNameOk() (*string, bool)`

GetCanaryConfigNameOk returns a tuple with the CanaryConfigName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCanaryConfigName

`func (o *RolloutsKayentaMetric) SetCanaryConfigName(v string)`

SetCanaryConfigName sets CanaryConfigName field to given value.

### HasCanaryConfigName

`func (o *RolloutsKayentaMetric) HasCanaryConfigName() bool`

HasCanaryConfigName returns a boolean if a field has been set.

### GetConfigurationAccountName

`func (o *RolloutsKayentaMetric) GetConfigurationAccountName() string`

GetConfigurationAccountName returns the ConfigurationAccountName field if non-nil, zero value otherwise.

### GetConfigurationAccountNameOk

`func (o *RolloutsKayentaMetric) GetConfigurationAccountNameOk() (*string, bool)`

GetConfigurationAccountNameOk returns a tuple with the ConfigurationAccountName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConfigurationAccountName

`func (o *RolloutsKayentaMetric) SetConfigurationAccountName(v string)`

SetConfigurationAccountName sets ConfigurationAccountName field to given value.

### HasConfigurationAccountName

`func (o *RolloutsKayentaMetric) HasConfigurationAccountName() bool`

HasConfigurationAccountName returns a boolean if a field has been set.

### GetLookback

`func (o *RolloutsKayentaMetric) GetLookback() bool`

GetLookback returns the Lookback field if non-nil, zero value otherwise.

### GetLookbackOk

`func (o *RolloutsKayentaMetric) GetLookbackOk() (*bool, bool)`

GetLookbackOk returns a tuple with the Lookback field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLookback

`func (o *RolloutsKayentaMetric) SetLookback(v bool)`

SetLookback sets Lookback field to given value.

### HasLookback

`func (o *RolloutsKayentaMetric) HasLookback() bool`

HasLookback returns a boolean if a field has been set.

### GetMetricsAccountName

`func (o *RolloutsKayentaMetric) GetMetricsAccountName() string`

GetMetricsAccountName returns the MetricsAccountName field if non-nil, zero value otherwise.

### GetMetricsAccountNameOk

`func (o *RolloutsKayentaMetric) GetMetricsAccountNameOk() (*string, bool)`

GetMetricsAccountNameOk returns a tuple with the MetricsAccountName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMetricsAccountName

`func (o *RolloutsKayentaMetric) SetMetricsAccountName(v string)`

SetMetricsAccountName sets MetricsAccountName field to given value.

### HasMetricsAccountName

`func (o *RolloutsKayentaMetric) HasMetricsAccountName() bool`

HasMetricsAccountName returns a boolean if a field has been set.

### GetScopes

`func (o *RolloutsKayentaMetric) GetScopes() []RolloutsKayentaScope`

GetScopes returns the Scopes field if non-nil, zero value otherwise.

### GetScopesOk

`func (o *RolloutsKayentaMetric) GetScopesOk() (*[]RolloutsKayentaScope, bool)`

GetScopesOk returns a tuple with the Scopes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetScopes

`func (o *RolloutsKayentaMetric) SetScopes(v []RolloutsKayentaScope)`

SetScopes sets Scopes field to given value.

### HasScopes

`func (o *RolloutsKayentaMetric) HasScopes() bool`

HasScopes returns a boolean if a field has been set.

### GetStorageAccountName

`func (o *RolloutsKayentaMetric) GetStorageAccountName() string`

GetStorageAccountName returns the StorageAccountName field if non-nil, zero value otherwise.

### GetStorageAccountNameOk

`func (o *RolloutsKayentaMetric) GetStorageAccountNameOk() (*string, bool)`

GetStorageAccountNameOk returns a tuple with the StorageAccountName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStorageAccountName

`func (o *RolloutsKayentaMetric) SetStorageAccountName(v string)`

SetStorageAccountName sets StorageAccountName field to given value.

### HasStorageAccountName

`func (o *RolloutsKayentaMetric) HasStorageAccountName() bool`

HasStorageAccountName returns a boolean if a field has been set.

### GetThreshold

`func (o *RolloutsKayentaMetric) GetThreshold() RolloutsKayentaThreshold`

GetThreshold returns the Threshold field if non-nil, zero value otherwise.

### GetThresholdOk

`func (o *RolloutsKayentaMetric) GetThresholdOk() (*RolloutsKayentaThreshold, bool)`

GetThresholdOk returns a tuple with the Threshold field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetThreshold

`func (o *RolloutsKayentaMetric) SetThreshold(v RolloutsKayentaThreshold)`

SetThreshold sets Threshold field to given value.

### HasThreshold

`func (o *RolloutsKayentaMetric) HasThreshold() bool`

HasThreshold returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


