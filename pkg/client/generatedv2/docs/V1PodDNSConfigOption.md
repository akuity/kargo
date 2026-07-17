# V1PodDNSConfigOption

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | Pointer to **string** | Name is this DNS resolver option&#39;s name. Required. | [optional] 
**Value** | Pointer to **string** | Value is this DNS resolver option&#39;s value. +optional | [optional] 

## Methods

### NewV1PodDNSConfigOption

`func NewV1PodDNSConfigOption() *V1PodDNSConfigOption`

NewV1PodDNSConfigOption instantiates a new V1PodDNSConfigOption object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1PodDNSConfigOptionWithDefaults

`func NewV1PodDNSConfigOptionWithDefaults() *V1PodDNSConfigOption`

NewV1PodDNSConfigOptionWithDefaults instantiates a new V1PodDNSConfigOption object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetName

`func (o *V1PodDNSConfigOption) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *V1PodDNSConfigOption) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *V1PodDNSConfigOption) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *V1PodDNSConfigOption) HasName() bool`

HasName returns a boolean if a field has been set.

### GetValue

`func (o *V1PodDNSConfigOption) GetValue() string`

GetValue returns the Value field if non-nil, zero value otherwise.

### GetValueOk

`func (o *V1PodDNSConfigOption) GetValueOk() (*string, bool)`

GetValueOk returns a tuple with the Value field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetValue

`func (o *V1PodDNSConfigOption) SetValue(v string)`

SetValue sets Value field to given value.

### HasValue

`func (o *V1PodDNSConfigOption) HasValue() bool`

HasValue returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


