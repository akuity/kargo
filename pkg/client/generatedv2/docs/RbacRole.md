# RbacRole

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ApiVersion** | Pointer to **string** | APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources +optional | [optional] 
**Claims** | Pointer to [**[]Claim**](Claim.md) |  | [optional] 
**KargoManaged** | Pointer to **bool** |  | [optional] 
**Kind** | Pointer to **string** | Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds +optional | [optional] 
**Metadata** | Pointer to [**V1ObjectMeta**](V1ObjectMeta.md) |  | [optional] 
**Rules** | Pointer to [**[]V1PolicyRule**](V1PolicyRule.md) |  | [optional] 

## Methods

### NewRbacRole

`func NewRbacRole() *RbacRole`

NewRbacRole instantiates a new RbacRole object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRbacRoleWithDefaults

`func NewRbacRoleWithDefaults() *RbacRole`

NewRbacRoleWithDefaults instantiates a new RbacRole object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetApiVersion

`func (o *RbacRole) GetApiVersion() string`

GetApiVersion returns the ApiVersion field if non-nil, zero value otherwise.

### GetApiVersionOk

`func (o *RbacRole) GetApiVersionOk() (*string, bool)`

GetApiVersionOk returns a tuple with the ApiVersion field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetApiVersion

`func (o *RbacRole) SetApiVersion(v string)`

SetApiVersion sets ApiVersion field to given value.

### HasApiVersion

`func (o *RbacRole) HasApiVersion() bool`

HasApiVersion returns a boolean if a field has been set.

### GetClaims

`func (o *RbacRole) GetClaims() []Claim`

GetClaims returns the Claims field if non-nil, zero value otherwise.

### GetClaimsOk

`func (o *RbacRole) GetClaimsOk() (*[]Claim, bool)`

GetClaimsOk returns a tuple with the Claims field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetClaims

`func (o *RbacRole) SetClaims(v []Claim)`

SetClaims sets Claims field to given value.

### HasClaims

`func (o *RbacRole) HasClaims() bool`

HasClaims returns a boolean if a field has been set.

### GetKargoManaged

`func (o *RbacRole) GetKargoManaged() bool`

GetKargoManaged returns the KargoManaged field if non-nil, zero value otherwise.

### GetKargoManagedOk

`func (o *RbacRole) GetKargoManagedOk() (*bool, bool)`

GetKargoManagedOk returns a tuple with the KargoManaged field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKargoManaged

`func (o *RbacRole) SetKargoManaged(v bool)`

SetKargoManaged sets KargoManaged field to given value.

### HasKargoManaged

`func (o *RbacRole) HasKargoManaged() bool`

HasKargoManaged returns a boolean if a field has been set.

### GetKind

`func (o *RbacRole) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *RbacRole) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *RbacRole) SetKind(v string)`

SetKind sets Kind field to given value.

### HasKind

`func (o *RbacRole) HasKind() bool`

HasKind returns a boolean if a field has been set.

### GetMetadata

`func (o *RbacRole) GetMetadata() V1ObjectMeta`

GetMetadata returns the Metadata field if non-nil, zero value otherwise.

### GetMetadataOk

`func (o *RbacRole) GetMetadataOk() (*V1ObjectMeta, bool)`

GetMetadataOk returns a tuple with the Metadata field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMetadata

`func (o *RbacRole) SetMetadata(v V1ObjectMeta)`

SetMetadata sets Metadata field to given value.

### HasMetadata

`func (o *RbacRole) HasMetadata() bool`

HasMetadata returns a boolean if a field has been set.

### GetRules

`func (o *RbacRole) GetRules() []V1PolicyRule`

GetRules returns the Rules field if non-nil, zero value otherwise.

### GetRulesOk

`func (o *RbacRole) GetRulesOk() (*[]V1PolicyRule, bool)`

GetRulesOk returns a tuple with the Rules field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRules

`func (o *RbacRole) SetRules(v []V1PolicyRule)`

SetRules sets Rules field to given value.

### HasRules

`func (o *RbacRole) HasRules() bool`

HasRules returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


