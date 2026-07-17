# PkgServerQueryFreightsResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Groups** | Pointer to [**map[string]PkgServerFreightGroupList**](PkgServerFreightGroupList.md) |  | [optional] 
**ResourceVersion** | Pointer to **string** | ResourceVersion is the Kubernetes list resourceVersion clients use to seed a follow-up Freight watch so the API server does not replay every existing Freight as an ADDED event. It is empty for the stage-scoped query, whose result is assembled from multiple sources rather than a single watchable namespace list. | [optional] 

## Methods

### NewPkgServerQueryFreightsResponse

`func NewPkgServerQueryFreightsResponse() *PkgServerQueryFreightsResponse`

NewPkgServerQueryFreightsResponse instantiates a new PkgServerQueryFreightsResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPkgServerQueryFreightsResponseWithDefaults

`func NewPkgServerQueryFreightsResponseWithDefaults() *PkgServerQueryFreightsResponse`

NewPkgServerQueryFreightsResponseWithDefaults instantiates a new PkgServerQueryFreightsResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetGroups

`func (o *PkgServerQueryFreightsResponse) GetGroups() map[string]PkgServerFreightGroupList`

GetGroups returns the Groups field if non-nil, zero value otherwise.

### GetGroupsOk

`func (o *PkgServerQueryFreightsResponse) GetGroupsOk() (*map[string]PkgServerFreightGroupList, bool)`

GetGroupsOk returns a tuple with the Groups field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetGroups

`func (o *PkgServerQueryFreightsResponse) SetGroups(v map[string]PkgServerFreightGroupList)`

SetGroups sets Groups field to given value.

### HasGroups

`func (o *PkgServerQueryFreightsResponse) HasGroups() bool`

HasGroups returns a boolean if a field has been set.

### GetResourceVersion

`func (o *PkgServerQueryFreightsResponse) GetResourceVersion() string`

GetResourceVersion returns the ResourceVersion field if non-nil, zero value otherwise.

### GetResourceVersionOk

`func (o *PkgServerQueryFreightsResponse) GetResourceVersionOk() (*string, bool)`

GetResourceVersionOk returns a tuple with the ResourceVersion field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResourceVersion

`func (o *PkgServerQueryFreightsResponse) SetResourceVersion(v string)`

SetResourceVersion sets ResourceVersion field to given value.

### HasResourceVersion

`func (o *PkgServerQueryFreightsResponse) HasResourceVersion() bool`

HasResourceVersion returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


