# V1PodDNSConfig

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Nameservers** | Pointer to **[]string** | A list of DNS name server IP addresses. This will be appended to the base nameservers generated from DNSPolicy. Duplicated nameservers will be removed. +optional +listType&#x3D;atomic | [optional] 
**Options** | Pointer to [**[]V1PodDNSConfigOption**](V1PodDNSConfigOption.md) | A list of DNS resolver options. This will be merged with the base options generated from DNSPolicy. Duplicated entries will be removed. Resolution options given in Options will override those that appear in the base DNSPolicy. +optional +listType&#x3D;atomic | [optional] 
**Searches** | Pointer to **[]string** | A list of DNS search domains for host-name lookup. This will be appended to the base search paths generated from DNSPolicy. Duplicated search paths will be removed. +optional +listType&#x3D;atomic | [optional] 

## Methods

### NewV1PodDNSConfig

`func NewV1PodDNSConfig() *V1PodDNSConfig`

NewV1PodDNSConfig instantiates a new V1PodDNSConfig object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1PodDNSConfigWithDefaults

`func NewV1PodDNSConfigWithDefaults() *V1PodDNSConfig`

NewV1PodDNSConfigWithDefaults instantiates a new V1PodDNSConfig object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetNameservers

`func (o *V1PodDNSConfig) GetNameservers() []string`

GetNameservers returns the Nameservers field if non-nil, zero value otherwise.

### GetNameserversOk

`func (o *V1PodDNSConfig) GetNameserversOk() (*[]string, bool)`

GetNameserversOk returns a tuple with the Nameservers field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNameservers

`func (o *V1PodDNSConfig) SetNameservers(v []string)`

SetNameservers sets Nameservers field to given value.

### HasNameservers

`func (o *V1PodDNSConfig) HasNameservers() bool`

HasNameservers returns a boolean if a field has been set.

### GetOptions

`func (o *V1PodDNSConfig) GetOptions() []V1PodDNSConfigOption`

GetOptions returns the Options field if non-nil, zero value otherwise.

### GetOptionsOk

`func (o *V1PodDNSConfig) GetOptionsOk() (*[]V1PodDNSConfigOption, bool)`

GetOptionsOk returns a tuple with the Options field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOptions

`func (o *V1PodDNSConfig) SetOptions(v []V1PodDNSConfigOption)`

SetOptions sets Options field to given value.

### HasOptions

`func (o *V1PodDNSConfig) HasOptions() bool`

HasOptions returns a boolean if a field has been set.

### GetSearches

`func (o *V1PodDNSConfig) GetSearches() []string`

GetSearches returns the Searches field if non-nil, zero value otherwise.

### GetSearchesOk

`func (o *V1PodDNSConfig) GetSearchesOk() (*[]string, bool)`

GetSearchesOk returns a tuple with the Searches field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSearches

`func (o *V1PodDNSConfig) SetSearches(v []string)`

SetSearches sets Searches field to given value.

### HasSearches

`func (o *V1PodDNSConfig) HasSearches() bool`

HasSearches returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


