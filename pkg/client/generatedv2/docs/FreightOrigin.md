# FreightOrigin

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Kind** | **string** | Kind is the kind of resource from which Freight may have originated. At present, this can only be \&quot;Warehouse\&quot;.  +kubebuilder:validation:Required | 
**Name** | **string** | Name is the name of the resource of the kind indicated by the Kind field from which Freight may originate.  +kubebuilder:validation:Required | 

## Methods

### NewFreightOrigin

`func NewFreightOrigin(kind string, name string, ) *FreightOrigin`

NewFreightOrigin instantiates a new FreightOrigin object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewFreightOriginWithDefaults

`func NewFreightOriginWithDefaults() *FreightOrigin`

NewFreightOriginWithDefaults instantiates a new FreightOrigin object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetKind

`func (o *FreightOrigin) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *FreightOrigin) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *FreightOrigin) SetKind(v string)`

SetKind sets Kind field to given value.


### GetName

`func (o *FreightOrigin) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *FreightOrigin) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *FreightOrigin) SetName(v string)`

SetName sets Name field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


