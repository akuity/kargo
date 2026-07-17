# WebhookReceiverDetails

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | Pointer to **string** | Name is the name of the webhook receiver. | [optional] 
**Path** | Pointer to **string** | Path is the path to the receiver&#39;s webhook endpoint. | [optional] 
**Url** | Pointer to **string** | URL includes the full address of the receiver&#39;s webhook endpoint. | [optional] 

## Methods

### NewWebhookReceiverDetails

`func NewWebhookReceiverDetails() *WebhookReceiverDetails`

NewWebhookReceiverDetails instantiates a new WebhookReceiverDetails object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewWebhookReceiverDetailsWithDefaults

`func NewWebhookReceiverDetailsWithDefaults() *WebhookReceiverDetails`

NewWebhookReceiverDetailsWithDefaults instantiates a new WebhookReceiverDetails object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetName

`func (o *WebhookReceiverDetails) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *WebhookReceiverDetails) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *WebhookReceiverDetails) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *WebhookReceiverDetails) HasName() bool`

HasName returns a boolean if a field has been set.

### GetPath

`func (o *WebhookReceiverDetails) GetPath() string`

GetPath returns the Path field if non-nil, zero value otherwise.

### GetPathOk

`func (o *WebhookReceiverDetails) GetPathOk() (*string, bool)`

GetPathOk returns a tuple with the Path field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPath

`func (o *WebhookReceiverDetails) SetPath(v string)`

SetPath sets Path field to given value.

### HasPath

`func (o *WebhookReceiverDetails) HasPath() bool`

HasPath returns a boolean if a field has been set.

### GetUrl

`func (o *WebhookReceiverDetails) GetUrl() string`

GetUrl returns the Url field if non-nil, zero value otherwise.

### GetUrlOk

`func (o *WebhookReceiverDetails) GetUrlOk() (*string, bool)`

GetUrlOk returns a tuple with the Url field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUrl

`func (o *WebhookReceiverDetails) SetUrl(v string)`

SetUrl sets Url field to given value.

### HasUrl

`func (o *WebhookReceiverDetails) HasUrl() bool`

HasUrl returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


