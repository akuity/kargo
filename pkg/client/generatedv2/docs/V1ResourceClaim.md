# V1ResourceClaim

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | Pointer to **string** | Name must match the name of one entry in pod.spec.resourceClaims of the Pod where this field is used. It makes that resource available inside a container. | [optional] 
**Request** | Pointer to **string** | Request is the name chosen for a request in the referenced claim. If empty, everything from the claim is made available, otherwise only the result of this request.  +optional | [optional] 

## Methods

### NewV1ResourceClaim

`func NewV1ResourceClaim() *V1ResourceClaim`

NewV1ResourceClaim instantiates a new V1ResourceClaim object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1ResourceClaimWithDefaults

`func NewV1ResourceClaimWithDefaults() *V1ResourceClaim`

NewV1ResourceClaimWithDefaults instantiates a new V1ResourceClaim object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetName

`func (o *V1ResourceClaim) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *V1ResourceClaim) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *V1ResourceClaim) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *V1ResourceClaim) HasName() bool`

HasName returns a boolean if a field has been set.

### GetRequest

`func (o *V1ResourceClaim) GetRequest() string`

GetRequest returns the Request field if non-nil, zero value otherwise.

### GetRequestOk

`func (o *V1ResourceClaim) GetRequestOk() (*string, bool)`

GetRequestOk returns a tuple with the Request field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRequest

`func (o *V1ResourceClaim) SetRequest(v string)`

SetRequest sets Request field to given value.

### HasRequest

`func (o *V1ResourceClaim) HasRequest() bool`

HasRequest returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


