# RolloutsMetricProvider

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**CloudWatch** | Pointer to [**RolloutsCloudWatchMetric**](RolloutsCloudWatchMetric.md) |  | [optional] 
**Datadog** | Pointer to [**RolloutsDatadogMetric**](RolloutsDatadogMetric.md) |  | [optional] 
**Graphite** | Pointer to [**RolloutsGraphiteMetric**](RolloutsGraphiteMetric.md) |  | [optional] 
**Influxdb** | Pointer to [**RolloutsInfluxdbMetric**](RolloutsInfluxdbMetric.md) |  | [optional] 
**Job** | Pointer to [**RolloutsJobMetric**](RolloutsJobMetric.md) |  | [optional] 
**Kayenta** | Pointer to [**RolloutsKayentaMetric**](RolloutsKayentaMetric.md) |  | [optional] 
**NewRelic** | Pointer to [**RolloutsNewRelicMetric**](RolloutsNewRelicMetric.md) |  | [optional] 
**Plugin** | Pointer to **map[string]string** |  | [optional] 
**Prometheus** | Pointer to [**RolloutsPrometheusMetric**](RolloutsPrometheusMetric.md) |  | [optional] 
**Skywalking** | Pointer to [**RolloutsSkyWalkingMetric**](RolloutsSkyWalkingMetric.md) |  | [optional] 
**Wavefront** | Pointer to [**RolloutsWavefrontMetric**](RolloutsWavefrontMetric.md) |  | [optional] 
**Web** | Pointer to [**RolloutsWebMetric**](RolloutsWebMetric.md) |  | [optional] 

## Methods

### NewRolloutsMetricProvider

`func NewRolloutsMetricProvider() *RolloutsMetricProvider`

NewRolloutsMetricProvider instantiates a new RolloutsMetricProvider object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRolloutsMetricProviderWithDefaults

`func NewRolloutsMetricProviderWithDefaults() *RolloutsMetricProvider`

NewRolloutsMetricProviderWithDefaults instantiates a new RolloutsMetricProvider object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetCloudWatch

`func (o *RolloutsMetricProvider) GetCloudWatch() RolloutsCloudWatchMetric`

GetCloudWatch returns the CloudWatch field if non-nil, zero value otherwise.

### GetCloudWatchOk

`func (o *RolloutsMetricProvider) GetCloudWatchOk() (*RolloutsCloudWatchMetric, bool)`

GetCloudWatchOk returns a tuple with the CloudWatch field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCloudWatch

`func (o *RolloutsMetricProvider) SetCloudWatch(v RolloutsCloudWatchMetric)`

SetCloudWatch sets CloudWatch field to given value.

### HasCloudWatch

`func (o *RolloutsMetricProvider) HasCloudWatch() bool`

HasCloudWatch returns a boolean if a field has been set.

### GetDatadog

`func (o *RolloutsMetricProvider) GetDatadog() RolloutsDatadogMetric`

GetDatadog returns the Datadog field if non-nil, zero value otherwise.

### GetDatadogOk

`func (o *RolloutsMetricProvider) GetDatadogOk() (*RolloutsDatadogMetric, bool)`

GetDatadogOk returns a tuple with the Datadog field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDatadog

`func (o *RolloutsMetricProvider) SetDatadog(v RolloutsDatadogMetric)`

SetDatadog sets Datadog field to given value.

### HasDatadog

`func (o *RolloutsMetricProvider) HasDatadog() bool`

HasDatadog returns a boolean if a field has been set.

### GetGraphite

`func (o *RolloutsMetricProvider) GetGraphite() RolloutsGraphiteMetric`

GetGraphite returns the Graphite field if non-nil, zero value otherwise.

### GetGraphiteOk

`func (o *RolloutsMetricProvider) GetGraphiteOk() (*RolloutsGraphiteMetric, bool)`

GetGraphiteOk returns a tuple with the Graphite field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetGraphite

`func (o *RolloutsMetricProvider) SetGraphite(v RolloutsGraphiteMetric)`

SetGraphite sets Graphite field to given value.

### HasGraphite

`func (o *RolloutsMetricProvider) HasGraphite() bool`

HasGraphite returns a boolean if a field has been set.

### GetInfluxdb

`func (o *RolloutsMetricProvider) GetInfluxdb() RolloutsInfluxdbMetric`

GetInfluxdb returns the Influxdb field if non-nil, zero value otherwise.

### GetInfluxdbOk

`func (o *RolloutsMetricProvider) GetInfluxdbOk() (*RolloutsInfluxdbMetric, bool)`

GetInfluxdbOk returns a tuple with the Influxdb field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInfluxdb

`func (o *RolloutsMetricProvider) SetInfluxdb(v RolloutsInfluxdbMetric)`

SetInfluxdb sets Influxdb field to given value.

### HasInfluxdb

`func (o *RolloutsMetricProvider) HasInfluxdb() bool`

HasInfluxdb returns a boolean if a field has been set.

### GetJob

`func (o *RolloutsMetricProvider) GetJob() RolloutsJobMetric`

GetJob returns the Job field if non-nil, zero value otherwise.

### GetJobOk

`func (o *RolloutsMetricProvider) GetJobOk() (*RolloutsJobMetric, bool)`

GetJobOk returns a tuple with the Job field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetJob

`func (o *RolloutsMetricProvider) SetJob(v RolloutsJobMetric)`

SetJob sets Job field to given value.

### HasJob

`func (o *RolloutsMetricProvider) HasJob() bool`

HasJob returns a boolean if a field has been set.

### GetKayenta

`func (o *RolloutsMetricProvider) GetKayenta() RolloutsKayentaMetric`

GetKayenta returns the Kayenta field if non-nil, zero value otherwise.

### GetKayentaOk

`func (o *RolloutsMetricProvider) GetKayentaOk() (*RolloutsKayentaMetric, bool)`

GetKayentaOk returns a tuple with the Kayenta field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKayenta

`func (o *RolloutsMetricProvider) SetKayenta(v RolloutsKayentaMetric)`

SetKayenta sets Kayenta field to given value.

### HasKayenta

`func (o *RolloutsMetricProvider) HasKayenta() bool`

HasKayenta returns a boolean if a field has been set.

### GetNewRelic

`func (o *RolloutsMetricProvider) GetNewRelic() RolloutsNewRelicMetric`

GetNewRelic returns the NewRelic field if non-nil, zero value otherwise.

### GetNewRelicOk

`func (o *RolloutsMetricProvider) GetNewRelicOk() (*RolloutsNewRelicMetric, bool)`

GetNewRelicOk returns a tuple with the NewRelic field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNewRelic

`func (o *RolloutsMetricProvider) SetNewRelic(v RolloutsNewRelicMetric)`

SetNewRelic sets NewRelic field to given value.

### HasNewRelic

`func (o *RolloutsMetricProvider) HasNewRelic() bool`

HasNewRelic returns a boolean if a field has been set.

### GetPlugin

`func (o *RolloutsMetricProvider) GetPlugin() map[string]string`

GetPlugin returns the Plugin field if non-nil, zero value otherwise.

### GetPluginOk

`func (o *RolloutsMetricProvider) GetPluginOk() (*map[string]string, bool)`

GetPluginOk returns a tuple with the Plugin field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPlugin

`func (o *RolloutsMetricProvider) SetPlugin(v map[string]string)`

SetPlugin sets Plugin field to given value.

### HasPlugin

`func (o *RolloutsMetricProvider) HasPlugin() bool`

HasPlugin returns a boolean if a field has been set.

### GetPrometheus

`func (o *RolloutsMetricProvider) GetPrometheus() RolloutsPrometheusMetric`

GetPrometheus returns the Prometheus field if non-nil, zero value otherwise.

### GetPrometheusOk

`func (o *RolloutsMetricProvider) GetPrometheusOk() (*RolloutsPrometheusMetric, bool)`

GetPrometheusOk returns a tuple with the Prometheus field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPrometheus

`func (o *RolloutsMetricProvider) SetPrometheus(v RolloutsPrometheusMetric)`

SetPrometheus sets Prometheus field to given value.

### HasPrometheus

`func (o *RolloutsMetricProvider) HasPrometheus() bool`

HasPrometheus returns a boolean if a field has been set.

### GetSkywalking

`func (o *RolloutsMetricProvider) GetSkywalking() RolloutsSkyWalkingMetric`

GetSkywalking returns the Skywalking field if non-nil, zero value otherwise.

### GetSkywalkingOk

`func (o *RolloutsMetricProvider) GetSkywalkingOk() (*RolloutsSkyWalkingMetric, bool)`

GetSkywalkingOk returns a tuple with the Skywalking field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSkywalking

`func (o *RolloutsMetricProvider) SetSkywalking(v RolloutsSkyWalkingMetric)`

SetSkywalking sets Skywalking field to given value.

### HasSkywalking

`func (o *RolloutsMetricProvider) HasSkywalking() bool`

HasSkywalking returns a boolean if a field has been set.

### GetWavefront

`func (o *RolloutsMetricProvider) GetWavefront() RolloutsWavefrontMetric`

GetWavefront returns the Wavefront field if non-nil, zero value otherwise.

### GetWavefrontOk

`func (o *RolloutsMetricProvider) GetWavefrontOk() (*RolloutsWavefrontMetric, bool)`

GetWavefrontOk returns a tuple with the Wavefront field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWavefront

`func (o *RolloutsMetricProvider) SetWavefront(v RolloutsWavefrontMetric)`

SetWavefront sets Wavefront field to given value.

### HasWavefront

`func (o *RolloutsMetricProvider) HasWavefront() bool`

HasWavefront returns a boolean if a field has been set.

### GetWeb

`func (o *RolloutsMetricProvider) GetWeb() RolloutsWebMetric`

GetWeb returns the Web field if non-nil, zero value otherwise.

### GetWebOk

`func (o *RolloutsMetricProvider) GetWebOk() (*RolloutsWebMetric, bool)`

GetWebOk returns a tuple with the Web field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWeb

`func (o *RolloutsMetricProvider) SetWeb(v RolloutsWebMetric)`

SetWeb sets Web field to given value.

### HasWeb

`func (o *RolloutsMetricProvider) HasWeb() bool`

HasWeb returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


