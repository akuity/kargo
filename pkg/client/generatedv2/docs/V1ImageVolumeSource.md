# V1ImageVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**PullPolicy** | Pointer to **string** | Policy for pulling OCI objects. Possible values are: Always: the kubelet always attempts to pull the reference. Container creation will fail If the pull fails. Never: the kubelet never pulls the reference and only uses a local image or artifact. Container creation will fail if the reference isn&#39;t present. IfNotPresent: the kubelet pulls if the reference isn&#39;t already present on disk. Container creation will fail if the reference isn&#39;t present and the pull fails. Defaults to Always if :latest tag is specified, or IfNotPresent otherwise. +optional | [optional] 
**Reference** | Pointer to **string** | Required: Image or artifact reference to be used. Behaves in the same way as pod.spec.containers[*].image. Pull secrets will be assembled in the same way as for the container image by looking up node credentials, SA image pull secrets, and pod spec image pull secrets. More info: https://kubernetes.io/docs/concepts/containers/images This field is optional to allow higher level config management to default or override container images in workload controllers like Deployments and StatefulSets. +optional | [optional] 

## Methods

### NewV1ImageVolumeSource

`func NewV1ImageVolumeSource() *V1ImageVolumeSource`

NewV1ImageVolumeSource instantiates a new V1ImageVolumeSource object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1ImageVolumeSourceWithDefaults

`func NewV1ImageVolumeSourceWithDefaults() *V1ImageVolumeSource`

NewV1ImageVolumeSourceWithDefaults instantiates a new V1ImageVolumeSource object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetPullPolicy

`func (o *V1ImageVolumeSource) GetPullPolicy() string`

GetPullPolicy returns the PullPolicy field if non-nil, zero value otherwise.

### GetPullPolicyOk

`func (o *V1ImageVolumeSource) GetPullPolicyOk() (*string, bool)`

GetPullPolicyOk returns a tuple with the PullPolicy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPullPolicy

`func (o *V1ImageVolumeSource) SetPullPolicy(v string)`

SetPullPolicy sets PullPolicy field to given value.

### HasPullPolicy

`func (o *V1ImageVolumeSource) HasPullPolicy() bool`

HasPullPolicy returns a boolean if a field has been set.

### GetReference

`func (o *V1ImageVolumeSource) GetReference() string`

GetReference returns the Reference field if non-nil, zero value otherwise.

### GetReferenceOk

`func (o *V1ImageVolumeSource) GetReferenceOk() (*string, bool)`

GetReferenceOk returns a tuple with the Reference field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReference

`func (o *V1ImageVolumeSource) SetReference(v string)`

SetReference sets Reference field to given value.

### HasReference

`func (o *V1ImageVolumeSource) HasReference() bool`

HasReference returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


