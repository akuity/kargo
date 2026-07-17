# V1PersistentVolumeClaimVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ClaimName** | Pointer to **string** | claimName is the name of a PersistentVolumeClaim in the same namespace as the pod using this volume. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims | [optional] 
**ReadOnly** | Pointer to **bool** | readOnly Will force the ReadOnly setting in VolumeMounts. Default false. +optional | [optional] 

## Methods

### NewV1PersistentVolumeClaimVolumeSource

`func NewV1PersistentVolumeClaimVolumeSource() *V1PersistentVolumeClaimVolumeSource`

NewV1PersistentVolumeClaimVolumeSource instantiates a new V1PersistentVolumeClaimVolumeSource object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1PersistentVolumeClaimVolumeSourceWithDefaults

`func NewV1PersistentVolumeClaimVolumeSourceWithDefaults() *V1PersistentVolumeClaimVolumeSource`

NewV1PersistentVolumeClaimVolumeSourceWithDefaults instantiates a new V1PersistentVolumeClaimVolumeSource object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetClaimName

`func (o *V1PersistentVolumeClaimVolumeSource) GetClaimName() string`

GetClaimName returns the ClaimName field if non-nil, zero value otherwise.

### GetClaimNameOk

`func (o *V1PersistentVolumeClaimVolumeSource) GetClaimNameOk() (*string, bool)`

GetClaimNameOk returns a tuple with the ClaimName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetClaimName

`func (o *V1PersistentVolumeClaimVolumeSource) SetClaimName(v string)`

SetClaimName sets ClaimName field to given value.

### HasClaimName

`func (o *V1PersistentVolumeClaimVolumeSource) HasClaimName() bool`

HasClaimName returns a boolean if a field has been set.

### GetReadOnly

`func (o *V1PersistentVolumeClaimVolumeSource) GetReadOnly() bool`

GetReadOnly returns the ReadOnly field if non-nil, zero value otherwise.

### GetReadOnlyOk

`func (o *V1PersistentVolumeClaimVolumeSource) GetReadOnlyOk() (*bool, bool)`

GetReadOnlyOk returns a tuple with the ReadOnly field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReadOnly

`func (o *V1PersistentVolumeClaimVolumeSource) SetReadOnly(v bool)`

SetReadOnly sets ReadOnly field to given value.

### HasReadOnly

`func (o *V1PersistentVolumeClaimVolumeSource) HasReadOnly() bool`

HasReadOnly returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


