# AutoRollbackConfig

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**OnPromotion** | Pointer to **[]string** | OnPromotion is the list of terminal Promotion phases that should trigger an automated rollback. Only Failed and Errored are accepted. Note that unsuccessful promotions (as opposed to unsuccessful verifications) may not necessarily indicate a problem with the Freight, since promotions might fail due to transient issues with the deployment itself (network, credential expirations, etc...). Defaults to [].  +optional +listType&#x3D;set +kubebuilder:validation:MaxItems&#x3D;2 +kubebuilder:validation:XValidation:message&#x3D;\&quot;onPromotion[0] must be Failed or Errored\&quot;,rule&#x3D;\&quot;self.size() &#x3D;&#x3D; 0 || self[0] &#x3D;&#x3D; &#39;Failed&#39; || self[0] &#x3D;&#x3D; &#39;Errored&#39;\&quot; +kubebuilder:validation:XValidation:message&#x3D;\&quot;onPromotion[1] must be Failed or Errored\&quot;,rule&#x3D;\&quot;self.size() &lt;&#x3D; 1 || self[1] &#x3D;&#x3D; &#39;Failed&#39; || self[1] &#x3D;&#x3D; &#39;Errored&#39;\&quot; | [optional] 
**OnVerification** | Pointer to **[]string** | OnVerification is the list of terminal verification phases that should trigger an automated rollback. Only Failed and Error are accepted (note: \&quot;Error\&quot;, not \&quot;Errored\&quot; as in onPromotion). When absent or empty, defaults to [Failed].  +optional +listType&#x3D;set +kubebuilder:validation:MaxItems&#x3D;2 +kubebuilder:validation:XValidation:message&#x3D;\&quot;onVerification[0] must be Failed or Error\&quot;,rule&#x3D;\&quot;self.size() &#x3D;&#x3D; 0 || self[0] &#x3D;&#x3D; &#39;Failed&#39; || self[0] &#x3D;&#x3D; &#39;Error&#39;\&quot; +kubebuilder:validation:XValidation:message&#x3D;\&quot;onVerification[1] must be Failed or Error\&quot;,rule&#x3D;\&quot;self.size() &lt;&#x3D; 1 || self[1] &#x3D;&#x3D; &#39;Failed&#39; || self[1] &#x3D;&#x3D; &#39;Error&#39;\&quot; | [optional] 

## Methods

### NewAutoRollbackConfig

`func NewAutoRollbackConfig() *AutoRollbackConfig`

NewAutoRollbackConfig instantiates a new AutoRollbackConfig object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAutoRollbackConfigWithDefaults

`func NewAutoRollbackConfigWithDefaults() *AutoRollbackConfig`

NewAutoRollbackConfigWithDefaults instantiates a new AutoRollbackConfig object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetOnPromotion

`func (o *AutoRollbackConfig) GetOnPromotion() []string`

GetOnPromotion returns the OnPromotion field if non-nil, zero value otherwise.

### GetOnPromotionOk

`func (o *AutoRollbackConfig) GetOnPromotionOk() (*[]string, bool)`

GetOnPromotionOk returns a tuple with the OnPromotion field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOnPromotion

`func (o *AutoRollbackConfig) SetOnPromotion(v []string)`

SetOnPromotion sets OnPromotion field to given value.

### HasOnPromotion

`func (o *AutoRollbackConfig) HasOnPromotion() bool`

HasOnPromotion returns a boolean if a field has been set.

### GetOnVerification

`func (o *AutoRollbackConfig) GetOnVerification() []string`

GetOnVerification returns the OnVerification field if non-nil, zero value otherwise.

### GetOnVerificationOk

`func (o *AutoRollbackConfig) GetOnVerificationOk() (*[]string, bool)`

GetOnVerificationOk returns a tuple with the OnVerification field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOnVerification

`func (o *AutoRollbackConfig) SetOnVerification(v []string)`

SetOnVerification sets OnVerification field to given value.

### HasOnVerification

`func (o *AutoRollbackConfig) HasOnVerification() bool`

HasOnVerification returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


