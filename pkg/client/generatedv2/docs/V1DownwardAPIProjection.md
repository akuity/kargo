# V1DownwardAPIProjection

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Items** | Pointer to [**[]V1DownwardAPIVolumeFile**](V1DownwardAPIVolumeFile.md) | Items is a list of DownwardAPIVolume file +optional +listType&#x3D;atomic | [optional] 

## Methods

### NewV1DownwardAPIProjection

`func NewV1DownwardAPIProjection() *V1DownwardAPIProjection`

NewV1DownwardAPIProjection instantiates a new V1DownwardAPIProjection object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1DownwardAPIProjectionWithDefaults

`func NewV1DownwardAPIProjectionWithDefaults() *V1DownwardAPIProjection`

NewV1DownwardAPIProjectionWithDefaults instantiates a new V1DownwardAPIProjection object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetItems

`func (o *V1DownwardAPIProjection) GetItems() []V1DownwardAPIVolumeFile`

GetItems returns the Items field if non-nil, zero value otherwise.

### GetItemsOk

`func (o *V1DownwardAPIProjection) GetItemsOk() (*[]V1DownwardAPIVolumeFile, bool)`

GetItemsOk returns a tuple with the Items field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetItems

`func (o *V1DownwardAPIProjection) SetItems(v []V1DownwardAPIVolumeFile)`

SetItems sets Items field to given value.

### HasItems

`func (o *V1DownwardAPIProjection) HasItems() bool`

HasItems returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


