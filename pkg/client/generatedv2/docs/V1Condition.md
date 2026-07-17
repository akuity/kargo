# V1Condition

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**LastTransitionTime** | **string** | lastTransitionTime is the last time the condition transitioned from one status to another. This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable. +required +kubebuilder:validation:Required +kubebuilder:validation:Type&#x3D;string +kubebuilder:validation:Format&#x3D;date-time | 
**Message** | **string** | message is a human readable message indicating details about the transition. This may be an empty string. +required +kubebuilder:validation:Required +kubebuilder:validation:MaxLength&#x3D;32768 | 
**ObservedGeneration** | Pointer to **int32** | observedGeneration represents the .metadata.generation that the condition was set based upon. For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date with respect to the current state of the instance. +optional +kubebuilder:validation:Minimum&#x3D;0 | [optional] 
**Reason** | **string** | reason contains a programmatic identifier indicating the reason for the condition&#39;s last transition. Producers of specific condition types may define expected values and meanings for this field, and whether the values are considered a guaranteed API. The value should be a CamelCase string. This field may not be empty. +required +kubebuilder:validation:Required +kubebuilder:validation:MaxLength&#x3D;1024 +kubebuilder:validation:MinLength&#x3D;1 +kubebuilder:validation:Pattern&#x3D;&#x60;^[A-Za-z]([A-Za-z0-9_,:]*[A-Za-z0-9_])?$&#x60; | 
**Status** | **string** | status of the condition, one of True, False, Unknown. +required +kubebuilder:validation:Required +kubebuilder:validation:Enum&#x3D;True;False;Unknown | 
**Type** | **string** | type of condition in CamelCase or in foo.example.com/CamelCase. --- Many .condition.type values are consistent across resources like Available, but because arbitrary conditions can be useful (see .node.status.conditions), the ability to deconflict is important. The regex it matches is (dns1123SubdomainFmt/)?(qualifiedNameFmt) +required +kubebuilder:validation:Required +kubebuilder:validation:Pattern&#x3D;&#x60;^([a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*_/)?(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])$&#x60; +kubebuilder:validation:MaxLength&#x3D;316 | 

## Methods

### NewV1Condition

`func NewV1Condition(lastTransitionTime string, message string, reason string, status string, type_ string, ) *V1Condition`

NewV1Condition instantiates a new V1Condition object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1ConditionWithDefaults

`func NewV1ConditionWithDefaults() *V1Condition`

NewV1ConditionWithDefaults instantiates a new V1Condition object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetLastTransitionTime

`func (o *V1Condition) GetLastTransitionTime() string`

GetLastTransitionTime returns the LastTransitionTime field if non-nil, zero value otherwise.

### GetLastTransitionTimeOk

`func (o *V1Condition) GetLastTransitionTimeOk() (*string, bool)`

GetLastTransitionTimeOk returns a tuple with the LastTransitionTime field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastTransitionTime

`func (o *V1Condition) SetLastTransitionTime(v string)`

SetLastTransitionTime sets LastTransitionTime field to given value.


### GetMessage

`func (o *V1Condition) GetMessage() string`

GetMessage returns the Message field if non-nil, zero value otherwise.

### GetMessageOk

`func (o *V1Condition) GetMessageOk() (*string, bool)`

GetMessageOk returns a tuple with the Message field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMessage

`func (o *V1Condition) SetMessage(v string)`

SetMessage sets Message field to given value.


### GetObservedGeneration

`func (o *V1Condition) GetObservedGeneration() int32`

GetObservedGeneration returns the ObservedGeneration field if non-nil, zero value otherwise.

### GetObservedGenerationOk

`func (o *V1Condition) GetObservedGenerationOk() (*int32, bool)`

GetObservedGenerationOk returns a tuple with the ObservedGeneration field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetObservedGeneration

`func (o *V1Condition) SetObservedGeneration(v int32)`

SetObservedGeneration sets ObservedGeneration field to given value.

### HasObservedGeneration

`func (o *V1Condition) HasObservedGeneration() bool`

HasObservedGeneration returns a boolean if a field has been set.

### GetReason

`func (o *V1Condition) GetReason() string`

GetReason returns the Reason field if non-nil, zero value otherwise.

### GetReasonOk

`func (o *V1Condition) GetReasonOk() (*string, bool)`

GetReasonOk returns a tuple with the Reason field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReason

`func (o *V1Condition) SetReason(v string)`

SetReason sets Reason field to given value.


### GetStatus

`func (o *V1Condition) GetStatus() string`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *V1Condition) GetStatusOk() (*string, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *V1Condition) SetStatus(v string)`

SetStatus sets Status field to given value.


### GetType

`func (o *V1Condition) GetType() string`

GetType returns the Type field if non-nil, zero value otherwise.

### GetTypeOk

`func (o *V1Condition) GetTypeOk() (*string, bool)`

GetTypeOk returns a tuple with the Type field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetType

`func (o *V1Condition) SetType(v string)`

SetType sets Type field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


