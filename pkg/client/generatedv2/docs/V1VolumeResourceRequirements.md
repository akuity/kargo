# V1VolumeResourceRequirements

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Limits** | Pointer to **map[string]interface{}** | Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/ +optional | [optional] 
**Requests** | Pointer to **map[string]interface{}** | Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. Requests cannot exceed Limits. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/ +optional | [optional] 

## Methods

### NewV1VolumeResourceRequirements

`func NewV1VolumeResourceRequirements() *V1VolumeResourceRequirements`

NewV1VolumeResourceRequirements instantiates a new V1VolumeResourceRequirements object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1VolumeResourceRequirementsWithDefaults

`func NewV1VolumeResourceRequirementsWithDefaults() *V1VolumeResourceRequirements`

NewV1VolumeResourceRequirementsWithDefaults instantiates a new V1VolumeResourceRequirements object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetLimits

`func (o *V1VolumeResourceRequirements) GetLimits() map[string]interface{}`

GetLimits returns the Limits field if non-nil, zero value otherwise.

### GetLimitsOk

`func (o *V1VolumeResourceRequirements) GetLimitsOk() (*map[string]interface{}, bool)`

GetLimitsOk returns a tuple with the Limits field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLimits

`func (o *V1VolumeResourceRequirements) SetLimits(v map[string]interface{})`

SetLimits sets Limits field to given value.

### HasLimits

`func (o *V1VolumeResourceRequirements) HasLimits() bool`

HasLimits returns a boolean if a field has been set.

### GetRequests

`func (o *V1VolumeResourceRequirements) GetRequests() map[string]interface{}`

GetRequests returns the Requests field if non-nil, zero value otherwise.

### GetRequestsOk

`func (o *V1VolumeResourceRequirements) GetRequestsOk() (*map[string]interface{}, bool)`

GetRequestsOk returns a tuple with the Requests field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRequests

`func (o *V1VolumeResourceRequirements) SetRequests(v map[string]interface{})`

SetRequests sets Requests field to given value.

### HasRequests

`func (o *V1VolumeResourceRequirements) HasRequests() bool`

HasRequests returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


