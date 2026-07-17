# V1EphemeralVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**VolumeClaimTemplate** | Pointer to [**V1PersistentVolumeClaimTemplate**](V1PersistentVolumeClaimTemplate.md) | Will be used to create a stand-alone PVC to provision the volume. The pod in which this EphemeralVolumeSource is embedded will be the owner of the PVC, i.e. the PVC will be deleted together with the pod.  The name of the PVC will be &#x60;&lt;pod name&gt;-&lt;volume name&gt;&#x60; where &#x60;&lt;volume name&gt;&#x60; is the name from the &#x60;PodSpec.Volumes&#x60; array entry. Pod validation will reject the pod if the concatenated name is not valid for a PVC (for example, too long).  An existing PVC with that name that is not owned by the pod will *not* be used for the pod to avoid using an unrelated volume by mistake. Starting the pod is then blocked until the unrelated PVC is removed. If such a pre-created PVC is meant to be used by the pod, the PVC has to updated with an owner reference to the pod once the pod exists. Normally this should not be necessary, but it may be useful when manually reconstructing a broken cluster.  This field is read-only and no changes will be made by Kubernetes to the PVC after it has been created.  Required, must not be nil. | [optional] 

## Methods

### NewV1EphemeralVolumeSource

`func NewV1EphemeralVolumeSource() *V1EphemeralVolumeSource`

NewV1EphemeralVolumeSource instantiates a new V1EphemeralVolumeSource object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1EphemeralVolumeSourceWithDefaults

`func NewV1EphemeralVolumeSourceWithDefaults() *V1EphemeralVolumeSource`

NewV1EphemeralVolumeSourceWithDefaults instantiates a new V1EphemeralVolumeSource object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetVolumeClaimTemplate

`func (o *V1EphemeralVolumeSource) GetVolumeClaimTemplate() V1PersistentVolumeClaimTemplate`

GetVolumeClaimTemplate returns the VolumeClaimTemplate field if non-nil, zero value otherwise.

### GetVolumeClaimTemplateOk

`func (o *V1EphemeralVolumeSource) GetVolumeClaimTemplateOk() (*V1PersistentVolumeClaimTemplate, bool)`

GetVolumeClaimTemplateOk returns a tuple with the VolumeClaimTemplate field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVolumeClaimTemplate

`func (o *V1EphemeralVolumeSource) SetVolumeClaimTemplate(v V1PersistentVolumeClaimTemplate)`

SetVolumeClaimTemplate sets VolumeClaimTemplate field to given value.

### HasVolumeClaimTemplate

`func (o *V1EphemeralVolumeSource) HasVolumeClaimTemplate() bool`

HasVolumeClaimTemplate returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


