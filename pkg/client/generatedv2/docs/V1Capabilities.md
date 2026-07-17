# V1Capabilities

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Add** | Pointer to **[]string** | Added capabilities +optional +listType&#x3D;atomic | [optional] 
**Drop** | Pointer to **[]string** | Removed capabilities +optional +listType&#x3D;atomic | [optional] 

## Methods

### NewV1Capabilities

`func NewV1Capabilities() *V1Capabilities`

NewV1Capabilities instantiates a new V1Capabilities object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1CapabilitiesWithDefaults

`func NewV1CapabilitiesWithDefaults() *V1Capabilities`

NewV1CapabilitiesWithDefaults instantiates a new V1Capabilities object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAdd

`func (o *V1Capabilities) GetAdd() []string`

GetAdd returns the Add field if non-nil, zero value otherwise.

### GetAddOk

`func (o *V1Capabilities) GetAddOk() (*[]string, bool)`

GetAddOk returns a tuple with the Add field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAdd

`func (o *V1Capabilities) SetAdd(v []string)`

SetAdd sets Add field to given value.

### HasAdd

`func (o *V1Capabilities) HasAdd() bool`

HasAdd returns a boolean if a field has been set.

### GetDrop

`func (o *V1Capabilities) GetDrop() []string`

GetDrop returns the Drop field if non-nil, zero value otherwise.

### GetDropOk

`func (o *V1Capabilities) GetDropOk() (*[]string, bool)`

GetDropOk returns a tuple with the Drop field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDrop

`func (o *V1Capabilities) SetDrop(v []string)`

SetDrop sets Drop field to given value.

### HasDrop

`func (o *V1Capabilities) HasDrop() bool`

HasDrop returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


