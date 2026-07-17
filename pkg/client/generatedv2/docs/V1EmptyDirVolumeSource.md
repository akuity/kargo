# V1EmptyDirVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Medium** | Pointer to **string** | medium represents what type of storage medium should back this directory. The default is \&quot;\&quot; which means to use the node&#39;s default medium. Must be an empty string (default) or Memory. More info: https://kubernetes.io/docs/concepts/storage/volumes#emptydir +optional | [optional] 
**SizeLimit** | Pointer to **interface{}** | sizeLimit is the total amount of local storage required for this EmptyDir volume. The size limit is also applicable for memory medium. The maximum usage on memory medium EmptyDir would be the minimum value between the SizeLimit specified here and the sum of memory limits of all containers in a pod. The default is nil which means that the limit is undefined. More info: https://kubernetes.io/docs/concepts/storage/volumes#emptydir +optional | [optional] 

## Methods

### NewV1EmptyDirVolumeSource

`func NewV1EmptyDirVolumeSource() *V1EmptyDirVolumeSource`

NewV1EmptyDirVolumeSource instantiates a new V1EmptyDirVolumeSource object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1EmptyDirVolumeSourceWithDefaults

`func NewV1EmptyDirVolumeSourceWithDefaults() *V1EmptyDirVolumeSource`

NewV1EmptyDirVolumeSourceWithDefaults instantiates a new V1EmptyDirVolumeSource object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetMedium

`func (o *V1EmptyDirVolumeSource) GetMedium() string`

GetMedium returns the Medium field if non-nil, zero value otherwise.

### GetMediumOk

`func (o *V1EmptyDirVolumeSource) GetMediumOk() (*string, bool)`

GetMediumOk returns a tuple with the Medium field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMedium

`func (o *V1EmptyDirVolumeSource) SetMedium(v string)`

SetMedium sets Medium field to given value.

### HasMedium

`func (o *V1EmptyDirVolumeSource) HasMedium() bool`

HasMedium returns a boolean if a field has been set.

### GetSizeLimit

`func (o *V1EmptyDirVolumeSource) GetSizeLimit() interface{}`

GetSizeLimit returns the SizeLimit field if non-nil, zero value otherwise.

### GetSizeLimitOk

`func (o *V1EmptyDirVolumeSource) GetSizeLimitOk() (*interface{}, bool)`

GetSizeLimitOk returns a tuple with the SizeLimit field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSizeLimit

`func (o *V1EmptyDirVolumeSource) SetSizeLimit(v interface{})`

SetSizeLimit sets SizeLimit field to given value.

### HasSizeLimit

`func (o *V1EmptyDirVolumeSource) HasSizeLimit() bool`

HasSizeLimit returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


