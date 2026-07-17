# V1ClusterTrustBundleProjection

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**LabelSelector** | Pointer to [**V1LabelSelector**](V1LabelSelector.md) | Select all ClusterTrustBundles that match this label selector.  Only has effect if signerName is set.  Mutually-exclusive with name.  If unset, interpreted as \&quot;match nothing\&quot;.  If set but empty, interpreted as \&quot;match everything\&quot;. +optional | [optional] 
**Name** | Pointer to **string** | Select a single ClusterTrustBundle by object name.  Mutually-exclusive with signerName and labelSelector. +optional | [optional] 
**Optional** | Pointer to **bool** | If true, don&#39;t block pod startup if the referenced ClusterTrustBundle(s) aren&#39;t available.  If using name, then the named ClusterTrustBundle is allowed not to exist.  If using signerName, then the combination of signerName and labelSelector is allowed to match zero ClusterTrustBundles. +optional | [optional] 
**Path** | Pointer to **string** | Relative path from the volume root to write the bundle. | [optional] 
**SignerName** | Pointer to **string** | Select all ClusterTrustBundles that match this signer name. Mutually-exclusive with name.  The contents of all selected ClusterTrustBundles will be unified and deduplicated. +optional | [optional] 

## Methods

### NewV1ClusterTrustBundleProjection

`func NewV1ClusterTrustBundleProjection() *V1ClusterTrustBundleProjection`

NewV1ClusterTrustBundleProjection instantiates a new V1ClusterTrustBundleProjection object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1ClusterTrustBundleProjectionWithDefaults

`func NewV1ClusterTrustBundleProjectionWithDefaults() *V1ClusterTrustBundleProjection`

NewV1ClusterTrustBundleProjectionWithDefaults instantiates a new V1ClusterTrustBundleProjection object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetLabelSelector

`func (o *V1ClusterTrustBundleProjection) GetLabelSelector() V1LabelSelector`

GetLabelSelector returns the LabelSelector field if non-nil, zero value otherwise.

### GetLabelSelectorOk

`func (o *V1ClusterTrustBundleProjection) GetLabelSelectorOk() (*V1LabelSelector, bool)`

GetLabelSelectorOk returns a tuple with the LabelSelector field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLabelSelector

`func (o *V1ClusterTrustBundleProjection) SetLabelSelector(v V1LabelSelector)`

SetLabelSelector sets LabelSelector field to given value.

### HasLabelSelector

`func (o *V1ClusterTrustBundleProjection) HasLabelSelector() bool`

HasLabelSelector returns a boolean if a field has been set.

### GetName

`func (o *V1ClusterTrustBundleProjection) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *V1ClusterTrustBundleProjection) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *V1ClusterTrustBundleProjection) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *V1ClusterTrustBundleProjection) HasName() bool`

HasName returns a boolean if a field has been set.

### GetOptional

`func (o *V1ClusterTrustBundleProjection) GetOptional() bool`

GetOptional returns the Optional field if non-nil, zero value otherwise.

### GetOptionalOk

`func (o *V1ClusterTrustBundleProjection) GetOptionalOk() (*bool, bool)`

GetOptionalOk returns a tuple with the Optional field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOptional

`func (o *V1ClusterTrustBundleProjection) SetOptional(v bool)`

SetOptional sets Optional field to given value.

### HasOptional

`func (o *V1ClusterTrustBundleProjection) HasOptional() bool`

HasOptional returns a boolean if a field has been set.

### GetPath

`func (o *V1ClusterTrustBundleProjection) GetPath() string`

GetPath returns the Path field if non-nil, zero value otherwise.

### GetPathOk

`func (o *V1ClusterTrustBundleProjection) GetPathOk() (*string, bool)`

GetPathOk returns a tuple with the Path field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPath

`func (o *V1ClusterTrustBundleProjection) SetPath(v string)`

SetPath sets Path field to given value.

### HasPath

`func (o *V1ClusterTrustBundleProjection) HasPath() bool`

HasPath returns a boolean if a field has been set.

### GetSignerName

`func (o *V1ClusterTrustBundleProjection) GetSignerName() string`

GetSignerName returns the SignerName field if non-nil, zero value otherwise.

### GetSignerNameOk

`func (o *V1ClusterTrustBundleProjection) GetSignerNameOk() (*string, bool)`

GetSignerNameOk returns a tuple with the SignerName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSignerName

`func (o *V1ClusterTrustBundleProjection) SetSignerName(v string)`

SetSignerName sets SignerName field to given value.

### HasSignerName

`func (o *V1ClusterTrustBundleProjection) HasSignerName() bool`

HasSignerName returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


