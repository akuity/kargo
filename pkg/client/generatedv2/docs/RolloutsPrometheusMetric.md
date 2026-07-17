# RolloutsPrometheusMetric

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Address** | Pointer to **string** |  | [optional] 
**Authentication** | Pointer to [**RolloutsAuthentication**](RolloutsAuthentication.md) |  | [optional] 
**Headers** | Pointer to [**[]RolloutsWebMetricHeader**](RolloutsWebMetricHeader.md) |  | [optional] 
**Insecure** | Pointer to **bool** |  | [optional] 
**Query** | Pointer to **string** |  | [optional] 
**RangeQuery** | Pointer to [**RolloutsPrometheusRangeQueryArgs**](RolloutsPrometheusRangeQueryArgs.md) |  | [optional] 
**Timeout** | Pointer to **int32** |  | [optional] 

## Methods

### NewRolloutsPrometheusMetric

`func NewRolloutsPrometheusMetric() *RolloutsPrometheusMetric`

NewRolloutsPrometheusMetric instantiates a new RolloutsPrometheusMetric object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRolloutsPrometheusMetricWithDefaults

`func NewRolloutsPrometheusMetricWithDefaults() *RolloutsPrometheusMetric`

NewRolloutsPrometheusMetricWithDefaults instantiates a new RolloutsPrometheusMetric object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAddress

`func (o *RolloutsPrometheusMetric) GetAddress() string`

GetAddress returns the Address field if non-nil, zero value otherwise.

### GetAddressOk

`func (o *RolloutsPrometheusMetric) GetAddressOk() (*string, bool)`

GetAddressOk returns a tuple with the Address field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAddress

`func (o *RolloutsPrometheusMetric) SetAddress(v string)`

SetAddress sets Address field to given value.

### HasAddress

`func (o *RolloutsPrometheusMetric) HasAddress() bool`

HasAddress returns a boolean if a field has been set.

### GetAuthentication

`func (o *RolloutsPrometheusMetric) GetAuthentication() RolloutsAuthentication`

GetAuthentication returns the Authentication field if non-nil, zero value otherwise.

### GetAuthenticationOk

`func (o *RolloutsPrometheusMetric) GetAuthenticationOk() (*RolloutsAuthentication, bool)`

GetAuthenticationOk returns a tuple with the Authentication field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAuthentication

`func (o *RolloutsPrometheusMetric) SetAuthentication(v RolloutsAuthentication)`

SetAuthentication sets Authentication field to given value.

### HasAuthentication

`func (o *RolloutsPrometheusMetric) HasAuthentication() bool`

HasAuthentication returns a boolean if a field has been set.

### GetHeaders

`func (o *RolloutsPrometheusMetric) GetHeaders() []RolloutsWebMetricHeader`

GetHeaders returns the Headers field if non-nil, zero value otherwise.

### GetHeadersOk

`func (o *RolloutsPrometheusMetric) GetHeadersOk() (*[]RolloutsWebMetricHeader, bool)`

GetHeadersOk returns a tuple with the Headers field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHeaders

`func (o *RolloutsPrometheusMetric) SetHeaders(v []RolloutsWebMetricHeader)`

SetHeaders sets Headers field to given value.

### HasHeaders

`func (o *RolloutsPrometheusMetric) HasHeaders() bool`

HasHeaders returns a boolean if a field has been set.

### GetInsecure

`func (o *RolloutsPrometheusMetric) GetInsecure() bool`

GetInsecure returns the Insecure field if non-nil, zero value otherwise.

### GetInsecureOk

`func (o *RolloutsPrometheusMetric) GetInsecureOk() (*bool, bool)`

GetInsecureOk returns a tuple with the Insecure field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInsecure

`func (o *RolloutsPrometheusMetric) SetInsecure(v bool)`

SetInsecure sets Insecure field to given value.

### HasInsecure

`func (o *RolloutsPrometheusMetric) HasInsecure() bool`

HasInsecure returns a boolean if a field has been set.

### GetQuery

`func (o *RolloutsPrometheusMetric) GetQuery() string`

GetQuery returns the Query field if non-nil, zero value otherwise.

### GetQueryOk

`func (o *RolloutsPrometheusMetric) GetQueryOk() (*string, bool)`

GetQueryOk returns a tuple with the Query field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetQuery

`func (o *RolloutsPrometheusMetric) SetQuery(v string)`

SetQuery sets Query field to given value.

### HasQuery

`func (o *RolloutsPrometheusMetric) HasQuery() bool`

HasQuery returns a boolean if a field has been set.

### GetRangeQuery

`func (o *RolloutsPrometheusMetric) GetRangeQuery() RolloutsPrometheusRangeQueryArgs`

GetRangeQuery returns the RangeQuery field if non-nil, zero value otherwise.

### GetRangeQueryOk

`func (o *RolloutsPrometheusMetric) GetRangeQueryOk() (*RolloutsPrometheusRangeQueryArgs, bool)`

GetRangeQueryOk returns a tuple with the RangeQuery field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRangeQuery

`func (o *RolloutsPrometheusMetric) SetRangeQuery(v RolloutsPrometheusRangeQueryArgs)`

SetRangeQuery sets RangeQuery field to given value.

### HasRangeQuery

`func (o *RolloutsPrometheusMetric) HasRangeQuery() bool`

HasRangeQuery returns a boolean if a field has been set.

### GetTimeout

`func (o *RolloutsPrometheusMetric) GetTimeout() int32`

GetTimeout returns the Timeout field if non-nil, zero value otherwise.

### GetTimeoutOk

`func (o *RolloutsPrometheusMetric) GetTimeoutOk() (*int32, bool)`

GetTimeoutOk returns a tuple with the Timeout field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTimeout

`func (o *RolloutsPrometheusMetric) SetTimeout(v int32)`

SetTimeout sets Timeout field to given value.

### HasTimeout

`func (o *RolloutsPrometheusMetric) HasTimeout() bool`

HasTimeout returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


