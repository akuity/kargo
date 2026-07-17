# RolloutsCloudWatchMetricStatMetric

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Dimensions** | Pointer to [**[]RolloutsCloudWatchMetricStatMetricDimension**](RolloutsCloudWatchMetricStatMetricDimension.md) |  | [optional] 
**MetricName** | Pointer to **string** |  | [optional] 
**Namespace** | Pointer to **string** |  | [optional] 

## Methods

### NewRolloutsCloudWatchMetricStatMetric

`func NewRolloutsCloudWatchMetricStatMetric() *RolloutsCloudWatchMetricStatMetric`

NewRolloutsCloudWatchMetricStatMetric instantiates a new RolloutsCloudWatchMetricStatMetric object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRolloutsCloudWatchMetricStatMetricWithDefaults

`func NewRolloutsCloudWatchMetricStatMetricWithDefaults() *RolloutsCloudWatchMetricStatMetric`

NewRolloutsCloudWatchMetricStatMetricWithDefaults instantiates a new RolloutsCloudWatchMetricStatMetric object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDimensions

`func (o *RolloutsCloudWatchMetricStatMetric) GetDimensions() []RolloutsCloudWatchMetricStatMetricDimension`

GetDimensions returns the Dimensions field if non-nil, zero value otherwise.

### GetDimensionsOk

`func (o *RolloutsCloudWatchMetricStatMetric) GetDimensionsOk() (*[]RolloutsCloudWatchMetricStatMetricDimension, bool)`

GetDimensionsOk returns a tuple with the Dimensions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDimensions

`func (o *RolloutsCloudWatchMetricStatMetric) SetDimensions(v []RolloutsCloudWatchMetricStatMetricDimension)`

SetDimensions sets Dimensions field to given value.

### HasDimensions

`func (o *RolloutsCloudWatchMetricStatMetric) HasDimensions() bool`

HasDimensions returns a boolean if a field has been set.

### GetMetricName

`func (o *RolloutsCloudWatchMetricStatMetric) GetMetricName() string`

GetMetricName returns the MetricName field if non-nil, zero value otherwise.

### GetMetricNameOk

`func (o *RolloutsCloudWatchMetricStatMetric) GetMetricNameOk() (*string, bool)`

GetMetricNameOk returns a tuple with the MetricName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMetricName

`func (o *RolloutsCloudWatchMetricStatMetric) SetMetricName(v string)`

SetMetricName sets MetricName field to given value.

### HasMetricName

`func (o *RolloutsCloudWatchMetricStatMetric) HasMetricName() bool`

HasMetricName returns a boolean if a field has been set.

### GetNamespace

`func (o *RolloutsCloudWatchMetricStatMetric) GetNamespace() string`

GetNamespace returns the Namespace field if non-nil, zero value otherwise.

### GetNamespaceOk

`func (o *RolloutsCloudWatchMetricStatMetric) GetNamespaceOk() (*string, bool)`

GetNamespaceOk returns a tuple with the Namespace field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNamespace

`func (o *RolloutsCloudWatchMetricStatMetric) SetNamespace(v string)`

SetNamespace sets Namespace field to given value.

### HasNamespace

`func (o *RolloutsCloudWatchMetricStatMetric) HasNamespace() bool`

HasNamespace returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


