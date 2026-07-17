# V1NFSVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Path** | Pointer to **string** | path that is exported by the NFS server. More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs | [optional] 
**ReadOnly** | Pointer to **bool** | readOnly here will force the NFS export to be mounted with read-only permissions. Defaults to false. More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs +optional | [optional] 
**Server** | Pointer to **string** | server is the hostname or IP address of the NFS server. More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs | [optional] 

## Methods

### NewV1NFSVolumeSource

`func NewV1NFSVolumeSource() *V1NFSVolumeSource`

NewV1NFSVolumeSource instantiates a new V1NFSVolumeSource object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1NFSVolumeSourceWithDefaults

`func NewV1NFSVolumeSourceWithDefaults() *V1NFSVolumeSource`

NewV1NFSVolumeSourceWithDefaults instantiates a new V1NFSVolumeSource object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetPath

`func (o *V1NFSVolumeSource) GetPath() string`

GetPath returns the Path field if non-nil, zero value otherwise.

### GetPathOk

`func (o *V1NFSVolumeSource) GetPathOk() (*string, bool)`

GetPathOk returns a tuple with the Path field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPath

`func (o *V1NFSVolumeSource) SetPath(v string)`

SetPath sets Path field to given value.

### HasPath

`func (o *V1NFSVolumeSource) HasPath() bool`

HasPath returns a boolean if a field has been set.

### GetReadOnly

`func (o *V1NFSVolumeSource) GetReadOnly() bool`

GetReadOnly returns the ReadOnly field if non-nil, zero value otherwise.

### GetReadOnlyOk

`func (o *V1NFSVolumeSource) GetReadOnlyOk() (*bool, bool)`

GetReadOnlyOk returns a tuple with the ReadOnly field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReadOnly

`func (o *V1NFSVolumeSource) SetReadOnly(v bool)`

SetReadOnly sets ReadOnly field to given value.

### HasReadOnly

`func (o *V1NFSVolumeSource) HasReadOnly() bool`

HasReadOnly returns a boolean if a field has been set.

### GetServer

`func (o *V1NFSVolumeSource) GetServer() string`

GetServer returns the Server field if non-nil, zero value otherwise.

### GetServerOk

`func (o *V1NFSVolumeSource) GetServerOk() (*string, bool)`

GetServerOk returns a tuple with the Server field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetServer

`func (o *V1NFSVolumeSource) SetServer(v string)`

SetServer sets Server field to given value.

### HasServer

`func (o *V1NFSVolumeSource) HasServer() bool`

HasServer returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


