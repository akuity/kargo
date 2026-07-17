# GetControllerHeartbeatsResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**DefaultController** | Pointer to **string** | DefaultController is the name of the default controller. The default controller is often unnamed, so an empty string is a valid value. This is included in the response to give clients a canonical identity to associate with Stages that have no explicit &#x60;spec.controller&#x60;. | [optional] 
**Heartbeats** | Pointer to [**map[string]Heartbeat**](Heartbeat.md) | Heartbeats is the most recent heartbeat from every controller that has reported in indexed by controller name. | [optional] 

## Methods

### NewGetControllerHeartbeatsResponse

`func NewGetControllerHeartbeatsResponse() *GetControllerHeartbeatsResponse`

NewGetControllerHeartbeatsResponse instantiates a new GetControllerHeartbeatsResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewGetControllerHeartbeatsResponseWithDefaults

`func NewGetControllerHeartbeatsResponseWithDefaults() *GetControllerHeartbeatsResponse`

NewGetControllerHeartbeatsResponseWithDefaults instantiates a new GetControllerHeartbeatsResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDefaultController

`func (o *GetControllerHeartbeatsResponse) GetDefaultController() string`

GetDefaultController returns the DefaultController field if non-nil, zero value otherwise.

### GetDefaultControllerOk

`func (o *GetControllerHeartbeatsResponse) GetDefaultControllerOk() (*string, bool)`

GetDefaultControllerOk returns a tuple with the DefaultController field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDefaultController

`func (o *GetControllerHeartbeatsResponse) SetDefaultController(v string)`

SetDefaultController sets DefaultController field to given value.

### HasDefaultController

`func (o *GetControllerHeartbeatsResponse) HasDefaultController() bool`

HasDefaultController returns a boolean if a field has been set.

### GetHeartbeats

`func (o *GetControllerHeartbeatsResponse) GetHeartbeats() map[string]Heartbeat`

GetHeartbeats returns the Heartbeats field if non-nil, zero value otherwise.

### GetHeartbeatsOk

`func (o *GetControllerHeartbeatsResponse) GetHeartbeatsOk() (*map[string]Heartbeat, bool)`

GetHeartbeatsOk returns a tuple with the Heartbeats field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHeartbeats

`func (o *GetControllerHeartbeatsResponse) SetHeartbeats(v map[string]Heartbeat)`

SetHeartbeats sets Heartbeats field to given value.

### HasHeartbeats

`func (o *GetControllerHeartbeatsResponse) HasHeartbeats() bool`

HasHeartbeats returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


