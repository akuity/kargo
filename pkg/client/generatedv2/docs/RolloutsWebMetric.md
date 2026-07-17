# RolloutsWebMetric

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Authentication** | Pointer to [**RolloutsAuthentication**](RolloutsAuthentication.md) |  | [optional] 
**Body** | Pointer to **string** |  | [optional] 
**Headers** | Pointer to [**[]RolloutsWebMetricHeader**](RolloutsWebMetricHeader.md) |  | [optional] 
**Insecure** | Pointer to **bool** |  | [optional] 
**JsonBody** | Pointer to **[]int32** |  | [optional] 
**JsonPath** | Pointer to **string** |  | [optional] 
**Method** | Pointer to **string** |  | [optional] 
**TimeoutSeconds** | Pointer to **int32** |  | [optional] 
**Url** | Pointer to **string** | URL is the address of the web metric | [optional] 

## Methods

### NewRolloutsWebMetric

`func NewRolloutsWebMetric() *RolloutsWebMetric`

NewRolloutsWebMetric instantiates a new RolloutsWebMetric object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRolloutsWebMetricWithDefaults

`func NewRolloutsWebMetricWithDefaults() *RolloutsWebMetric`

NewRolloutsWebMetricWithDefaults instantiates a new RolloutsWebMetric object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAuthentication

`func (o *RolloutsWebMetric) GetAuthentication() RolloutsAuthentication`

GetAuthentication returns the Authentication field if non-nil, zero value otherwise.

### GetAuthenticationOk

`func (o *RolloutsWebMetric) GetAuthenticationOk() (*RolloutsAuthentication, bool)`

GetAuthenticationOk returns a tuple with the Authentication field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAuthentication

`func (o *RolloutsWebMetric) SetAuthentication(v RolloutsAuthentication)`

SetAuthentication sets Authentication field to given value.

### HasAuthentication

`func (o *RolloutsWebMetric) HasAuthentication() bool`

HasAuthentication returns a boolean if a field has been set.

### GetBody

`func (o *RolloutsWebMetric) GetBody() string`

GetBody returns the Body field if non-nil, zero value otherwise.

### GetBodyOk

`func (o *RolloutsWebMetric) GetBodyOk() (*string, bool)`

GetBodyOk returns a tuple with the Body field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBody

`func (o *RolloutsWebMetric) SetBody(v string)`

SetBody sets Body field to given value.

### HasBody

`func (o *RolloutsWebMetric) HasBody() bool`

HasBody returns a boolean if a field has been set.

### GetHeaders

`func (o *RolloutsWebMetric) GetHeaders() []RolloutsWebMetricHeader`

GetHeaders returns the Headers field if non-nil, zero value otherwise.

### GetHeadersOk

`func (o *RolloutsWebMetric) GetHeadersOk() (*[]RolloutsWebMetricHeader, bool)`

GetHeadersOk returns a tuple with the Headers field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHeaders

`func (o *RolloutsWebMetric) SetHeaders(v []RolloutsWebMetricHeader)`

SetHeaders sets Headers field to given value.

### HasHeaders

`func (o *RolloutsWebMetric) HasHeaders() bool`

HasHeaders returns a boolean if a field has been set.

### GetInsecure

`func (o *RolloutsWebMetric) GetInsecure() bool`

GetInsecure returns the Insecure field if non-nil, zero value otherwise.

### GetInsecureOk

`func (o *RolloutsWebMetric) GetInsecureOk() (*bool, bool)`

GetInsecureOk returns a tuple with the Insecure field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInsecure

`func (o *RolloutsWebMetric) SetInsecure(v bool)`

SetInsecure sets Insecure field to given value.

### HasInsecure

`func (o *RolloutsWebMetric) HasInsecure() bool`

HasInsecure returns a boolean if a field has been set.

### GetJsonBody

`func (o *RolloutsWebMetric) GetJsonBody() []int32`

GetJsonBody returns the JsonBody field if non-nil, zero value otherwise.

### GetJsonBodyOk

`func (o *RolloutsWebMetric) GetJsonBodyOk() (*[]int32, bool)`

GetJsonBodyOk returns a tuple with the JsonBody field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetJsonBody

`func (o *RolloutsWebMetric) SetJsonBody(v []int32)`

SetJsonBody sets JsonBody field to given value.

### HasJsonBody

`func (o *RolloutsWebMetric) HasJsonBody() bool`

HasJsonBody returns a boolean if a field has been set.

### GetJsonPath

`func (o *RolloutsWebMetric) GetJsonPath() string`

GetJsonPath returns the JsonPath field if non-nil, zero value otherwise.

### GetJsonPathOk

`func (o *RolloutsWebMetric) GetJsonPathOk() (*string, bool)`

GetJsonPathOk returns a tuple with the JsonPath field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetJsonPath

`func (o *RolloutsWebMetric) SetJsonPath(v string)`

SetJsonPath sets JsonPath field to given value.

### HasJsonPath

`func (o *RolloutsWebMetric) HasJsonPath() bool`

HasJsonPath returns a boolean if a field has been set.

### GetMethod

`func (o *RolloutsWebMetric) GetMethod() string`

GetMethod returns the Method field if non-nil, zero value otherwise.

### GetMethodOk

`func (o *RolloutsWebMetric) GetMethodOk() (*string, bool)`

GetMethodOk returns a tuple with the Method field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMethod

`func (o *RolloutsWebMetric) SetMethod(v string)`

SetMethod sets Method field to given value.

### HasMethod

`func (o *RolloutsWebMetric) HasMethod() bool`

HasMethod returns a boolean if a field has been set.

### GetTimeoutSeconds

`func (o *RolloutsWebMetric) GetTimeoutSeconds() int32`

GetTimeoutSeconds returns the TimeoutSeconds field if non-nil, zero value otherwise.

### GetTimeoutSecondsOk

`func (o *RolloutsWebMetric) GetTimeoutSecondsOk() (*int32, bool)`

GetTimeoutSecondsOk returns a tuple with the TimeoutSeconds field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTimeoutSeconds

`func (o *RolloutsWebMetric) SetTimeoutSeconds(v int32)`

SetTimeoutSeconds sets TimeoutSeconds field to given value.

### HasTimeoutSeconds

`func (o *RolloutsWebMetric) HasTimeoutSeconds() bool`

HasTimeoutSeconds returns a boolean if a field has been set.

### GetUrl

`func (o *RolloutsWebMetric) GetUrl() string`

GetUrl returns the Url field if non-nil, zero value otherwise.

### GetUrlOk

`func (o *RolloutsWebMetric) GetUrlOk() (*string, bool)`

GetUrlOk returns a tuple with the Url field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUrl

`func (o *RolloutsWebMetric) SetUrl(v string)`

SetUrl sets Url field to given value.

### HasUrl

`func (o *RolloutsWebMetric) HasUrl() bool`

HasUrl returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


