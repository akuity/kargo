# UserClaims

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Claims** | Pointer to [**[]Claim**](Claim.md) |  | [optional] 

## Methods

### NewUserClaims

`func NewUserClaims() *UserClaims`

NewUserClaims instantiates a new UserClaims object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewUserClaimsWithDefaults

`func NewUserClaimsWithDefaults() *UserClaims`

NewUserClaimsWithDefaults instantiates a new UserClaims object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetClaims

`func (o *UserClaims) GetClaims() []Claim`

GetClaims returns the Claims field if non-nil, zero value otherwise.

### GetClaimsOk

`func (o *UserClaims) GetClaimsOk() (*[]Claim, bool)`

GetClaimsOk returns a tuple with the Claims field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetClaims

`func (o *UserClaims) SetClaims(v []Claim)`

SetClaims sets Claims field to given value.

### HasClaims

`func (o *UserClaims) HasClaims() bool`

HasClaims returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


