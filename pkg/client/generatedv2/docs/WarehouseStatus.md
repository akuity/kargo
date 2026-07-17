# WarehouseStatus

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Conditions** | Pointer to [**[]V1Condition**](V1Condition.md) | Conditions contains the last observations of the Warehouse&#39;s current state. +patchMergeKey&#x3D;type +patchStrategy&#x3D;merge +listType&#x3D;map +listMapKey&#x3D;type | [optional] 
**DiscoveredArtifacts** | Pointer to [**DiscoveredArtifacts**](DiscoveredArtifacts.md) | DiscoveredArtifacts holds the artifacts discovered by the Warehouse. | [optional] 
**LastFreightID** | Pointer to **string** | LastFreightID is a reference to the system-assigned identifier (name) of the most recent Freight produced by the Warehouse. | [optional] 
**LastHandledRefresh** | Pointer to **string** | LastHandledRefresh holds the value of the most recent AnnotationKeyRefresh annotation that was handled by the controller. This field can be used to determine whether the request to refresh the resource has been handled. +optional | [optional] 
**ObservedGeneration** | Pointer to **int32** | ObservedGeneration represents the .metadata.generation that this Warehouse was reconciled against. | [optional] 

## Methods

### NewWarehouseStatus

`func NewWarehouseStatus() *WarehouseStatus`

NewWarehouseStatus instantiates a new WarehouseStatus object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewWarehouseStatusWithDefaults

`func NewWarehouseStatusWithDefaults() *WarehouseStatus`

NewWarehouseStatusWithDefaults instantiates a new WarehouseStatus object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetConditions

`func (o *WarehouseStatus) GetConditions() []V1Condition`

GetConditions returns the Conditions field if non-nil, zero value otherwise.

### GetConditionsOk

`func (o *WarehouseStatus) GetConditionsOk() (*[]V1Condition, bool)`

GetConditionsOk returns a tuple with the Conditions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConditions

`func (o *WarehouseStatus) SetConditions(v []V1Condition)`

SetConditions sets Conditions field to given value.

### HasConditions

`func (o *WarehouseStatus) HasConditions() bool`

HasConditions returns a boolean if a field has been set.

### GetDiscoveredArtifacts

`func (o *WarehouseStatus) GetDiscoveredArtifacts() DiscoveredArtifacts`

GetDiscoveredArtifacts returns the DiscoveredArtifacts field if non-nil, zero value otherwise.

### GetDiscoveredArtifactsOk

`func (o *WarehouseStatus) GetDiscoveredArtifactsOk() (*DiscoveredArtifacts, bool)`

GetDiscoveredArtifactsOk returns a tuple with the DiscoveredArtifacts field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDiscoveredArtifacts

`func (o *WarehouseStatus) SetDiscoveredArtifacts(v DiscoveredArtifacts)`

SetDiscoveredArtifacts sets DiscoveredArtifacts field to given value.

### HasDiscoveredArtifacts

`func (o *WarehouseStatus) HasDiscoveredArtifacts() bool`

HasDiscoveredArtifacts returns a boolean if a field has been set.

### GetLastFreightID

`func (o *WarehouseStatus) GetLastFreightID() string`

GetLastFreightID returns the LastFreightID field if non-nil, zero value otherwise.

### GetLastFreightIDOk

`func (o *WarehouseStatus) GetLastFreightIDOk() (*string, bool)`

GetLastFreightIDOk returns a tuple with the LastFreightID field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastFreightID

`func (o *WarehouseStatus) SetLastFreightID(v string)`

SetLastFreightID sets LastFreightID field to given value.

### HasLastFreightID

`func (o *WarehouseStatus) HasLastFreightID() bool`

HasLastFreightID returns a boolean if a field has been set.

### GetLastHandledRefresh

`func (o *WarehouseStatus) GetLastHandledRefresh() string`

GetLastHandledRefresh returns the LastHandledRefresh field if non-nil, zero value otherwise.

### GetLastHandledRefreshOk

`func (o *WarehouseStatus) GetLastHandledRefreshOk() (*string, bool)`

GetLastHandledRefreshOk returns a tuple with the LastHandledRefresh field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastHandledRefresh

`func (o *WarehouseStatus) SetLastHandledRefresh(v string)`

SetLastHandledRefresh sets LastHandledRefresh field to given value.

### HasLastHandledRefresh

`func (o *WarehouseStatus) HasLastHandledRefresh() bool`

HasLastHandledRefresh returns a boolean if a field has been set.

### GetObservedGeneration

`func (o *WarehouseStatus) GetObservedGeneration() int32`

GetObservedGeneration returns the ObservedGeneration field if non-nil, zero value otherwise.

### GetObservedGenerationOk

`func (o *WarehouseStatus) GetObservedGenerationOk() (*int32, bool)`

GetObservedGenerationOk returns a tuple with the ObservedGeneration field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetObservedGeneration

`func (o *WarehouseStatus) SetObservedGeneration(v int32)`

SetObservedGeneration sets ObservedGeneration field to given value.

### HasObservedGeneration

`func (o *WarehouseStatus) HasObservedGeneration() bool`

HasObservedGeneration returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


