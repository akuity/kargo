# WarehouseSpec

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**FreightCreationCriteria** | Pointer to [**FreightCreationCriteria**](FreightCreationCriteria.md) | FreightCreationCriteria defines criteria that must be satisfied for Freight to be created automatically from new artifacts following discovery. This field has no effect when the FreightCreationPolicy is &#x60;Manual&#x60;.  +kubebuilder:validation:Optional | [optional] 
**FreightCreationPolicy** | Pointer to **string** | FreightCreationPolicy describes how Freight is created by this Warehouse. This field is optional. When left unspecified, the field is implicitly treated as if its value were \&quot;Automatic\&quot;.  Accepted values:  - \&quot;Automatic\&quot;: New Freight is created automatically when any new artifact   is discovered. - \&quot;Manual\&quot;: New Freight is never created automatically.  +kubebuilder:default&#x3D;Automatic +kubebuilder:validation:Optional | [optional] 
**Interval** | Pointer to **string** | Interval is the reconciliation interval for this Warehouse. On each reconciliation, the Warehouse will discover new artifacts and optionally produce new Freight. This field is optional. When left unspecified, the field is implicitly treated as if its value were \&quot;5m0s\&quot;.  +kubebuilder:validation:Type&#x3D;string +kubebuilder:validation:Pattern&#x3D;&#x60;^([0-9]+(\\.[0-9]+)?(s|m|h))+$&#x60; +kubebuilder:default&#x3D;\&quot;5m0s\&quot; +akuity:test-kubebuilder-pattern&#x3D;Duration | [optional] 
**Shard** | Pointer to **string** | Shard is the name of the shard that this Warehouse belongs to. This is an optional field. If not specified, the Warehouse will belong to the default shard. A defaulting webhook will sync this field with the value of the kargo.akuity.io/shard label. When the shard label is not present or differs from the value of this field, the defaulting webhook will set the label to the value of this field. If the shard label is present and this field is empty, the defaulting webhook will set the value of this field to the value of the shard label. | [optional] 
**Subscriptions** | Pointer to **[]interface{}** | Subscriptions describes sources of artifacts to be included in Freight produced by this Warehouse.  +kubebuilder:validation:MinItems&#x3D;1 | [optional] 

## Methods

### NewWarehouseSpec

`func NewWarehouseSpec() *WarehouseSpec`

NewWarehouseSpec instantiates a new WarehouseSpec object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewWarehouseSpecWithDefaults

`func NewWarehouseSpecWithDefaults() *WarehouseSpec`

NewWarehouseSpecWithDefaults instantiates a new WarehouseSpec object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetFreightCreationCriteria

`func (o *WarehouseSpec) GetFreightCreationCriteria() FreightCreationCriteria`

GetFreightCreationCriteria returns the FreightCreationCriteria field if non-nil, zero value otherwise.

### GetFreightCreationCriteriaOk

`func (o *WarehouseSpec) GetFreightCreationCriteriaOk() (*FreightCreationCriteria, bool)`

GetFreightCreationCriteriaOk returns a tuple with the FreightCreationCriteria field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFreightCreationCriteria

`func (o *WarehouseSpec) SetFreightCreationCriteria(v FreightCreationCriteria)`

SetFreightCreationCriteria sets FreightCreationCriteria field to given value.

### HasFreightCreationCriteria

`func (o *WarehouseSpec) HasFreightCreationCriteria() bool`

HasFreightCreationCriteria returns a boolean if a field has been set.

### GetFreightCreationPolicy

`func (o *WarehouseSpec) GetFreightCreationPolicy() string`

GetFreightCreationPolicy returns the FreightCreationPolicy field if non-nil, zero value otherwise.

### GetFreightCreationPolicyOk

`func (o *WarehouseSpec) GetFreightCreationPolicyOk() (*string, bool)`

GetFreightCreationPolicyOk returns a tuple with the FreightCreationPolicy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFreightCreationPolicy

`func (o *WarehouseSpec) SetFreightCreationPolicy(v string)`

SetFreightCreationPolicy sets FreightCreationPolicy field to given value.

### HasFreightCreationPolicy

`func (o *WarehouseSpec) HasFreightCreationPolicy() bool`

HasFreightCreationPolicy returns a boolean if a field has been set.

### GetInterval

`func (o *WarehouseSpec) GetInterval() string`

GetInterval returns the Interval field if non-nil, zero value otherwise.

### GetIntervalOk

`func (o *WarehouseSpec) GetIntervalOk() (*string, bool)`

GetIntervalOk returns a tuple with the Interval field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInterval

`func (o *WarehouseSpec) SetInterval(v string)`

SetInterval sets Interval field to given value.

### HasInterval

`func (o *WarehouseSpec) HasInterval() bool`

HasInterval returns a boolean if a field has been set.

### GetShard

`func (o *WarehouseSpec) GetShard() string`

GetShard returns the Shard field if non-nil, zero value otherwise.

### GetShardOk

`func (o *WarehouseSpec) GetShardOk() (*string, bool)`

GetShardOk returns a tuple with the Shard field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetShard

`func (o *WarehouseSpec) SetShard(v string)`

SetShard sets Shard field to given value.

### HasShard

`func (o *WarehouseSpec) HasShard() bool`

HasShard returns a boolean if a field has been set.

### GetSubscriptions

`func (o *WarehouseSpec) GetSubscriptions() []interface{}`

GetSubscriptions returns the Subscriptions field if non-nil, zero value otherwise.

### GetSubscriptionsOk

`func (o *WarehouseSpec) GetSubscriptionsOk() (*[]interface{}, bool)`

GetSubscriptionsOk returns a tuple with the Subscriptions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSubscriptions

`func (o *WarehouseSpec) SetSubscriptions(v []interface{})`

SetSubscriptions sets Subscriptions field to given value.

### HasSubscriptions

`func (o *WarehouseSpec) HasSubscriptions() bool`

HasSubscriptions returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


