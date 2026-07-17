# V1ServiceAccountTokenProjection

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Audience** | Pointer to **string** | audience is the intended audience of the token. A recipient of a token must identify itself with an identifier specified in the audience of the token, and otherwise should reject the token. The audience defaults to the identifier of the apiserver. +optional | [optional] 
**ExpirationSeconds** | Pointer to **int32** | expirationSeconds is the requested duration of validity of the service account token. As the token approaches expiration, the kubelet volume plugin will proactively rotate the service account token. The kubelet will start trying to rotate the token if the token is older than 80 percent of its time to live or if the token is older than 24 hours.Defaults to 1 hour and must be at least 10 minutes. +optional | [optional] 
**Path** | Pointer to **string** | path is the path relative to the mount point of the file to project the token into. | [optional] 

## Methods

### NewV1ServiceAccountTokenProjection

`func NewV1ServiceAccountTokenProjection() *V1ServiceAccountTokenProjection`

NewV1ServiceAccountTokenProjection instantiates a new V1ServiceAccountTokenProjection object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1ServiceAccountTokenProjectionWithDefaults

`func NewV1ServiceAccountTokenProjectionWithDefaults() *V1ServiceAccountTokenProjection`

NewV1ServiceAccountTokenProjectionWithDefaults instantiates a new V1ServiceAccountTokenProjection object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAudience

`func (o *V1ServiceAccountTokenProjection) GetAudience() string`

GetAudience returns the Audience field if non-nil, zero value otherwise.

### GetAudienceOk

`func (o *V1ServiceAccountTokenProjection) GetAudienceOk() (*string, bool)`

GetAudienceOk returns a tuple with the Audience field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAudience

`func (o *V1ServiceAccountTokenProjection) SetAudience(v string)`

SetAudience sets Audience field to given value.

### HasAudience

`func (o *V1ServiceAccountTokenProjection) HasAudience() bool`

HasAudience returns a boolean if a field has been set.

### GetExpirationSeconds

`func (o *V1ServiceAccountTokenProjection) GetExpirationSeconds() int32`

GetExpirationSeconds returns the ExpirationSeconds field if non-nil, zero value otherwise.

### GetExpirationSecondsOk

`func (o *V1ServiceAccountTokenProjection) GetExpirationSecondsOk() (*int32, bool)`

GetExpirationSecondsOk returns a tuple with the ExpirationSeconds field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExpirationSeconds

`func (o *V1ServiceAccountTokenProjection) SetExpirationSeconds(v int32)`

SetExpirationSeconds sets ExpirationSeconds field to given value.

### HasExpirationSeconds

`func (o *V1ServiceAccountTokenProjection) HasExpirationSeconds() bool`

HasExpirationSeconds returns a boolean if a field has been set.

### GetPath

`func (o *V1ServiceAccountTokenProjection) GetPath() string`

GetPath returns the Path field if non-nil, zero value otherwise.

### GetPathOk

`func (o *V1ServiceAccountTokenProjection) GetPathOk() (*string, bool)`

GetPathOk returns a tuple with the Path field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPath

`func (o *V1ServiceAccountTokenProjection) SetPath(v string)`

SetPath sets Path field to given value.

### HasPath

`func (o *V1ServiceAccountTokenProjection) HasPath() bool`

HasPath returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


