# PromotionStepRetry

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ErrorThreshold** | Pointer to **int32** | ErrorThreshold is the number of consecutive times the step must fail (for any reason) before retries are abandoned and the entire Promotion is marked as failed.  If this field is set to 0, the effective default will be a step-specific one. If no step-specific default exists (i.e. is also 0), the effective default will be the system-wide default of 1.  A value of 1 will cause the Promotion to be marked as failed after just a single failure; i.e. no retries will be attempted.  There is no option to specify an infinite number of retries using a value such as -1.  In a future release, Kargo is likely to become capable of distinguishing between recoverable and non-recoverable step failures. At that time, it is planned that unrecoverable failures will not be subject to this threshold and will immediately cause the Promotion to be marked as failed without further condition. | [optional] 
**Timeout** | Pointer to **string** | Timeout is the soft maximum interval in which a step that returns a Running status (which typically indicates it&#39;s waiting for something to happen) may be retried.  The maximum is a soft one because the check for whether the interval has elapsed occurs AFTER the step has run. This effectively means a step may run ONCE beyond the close of the interval.  If this field is set to nil, the effective default will be a step-specific one. If no step-specific default exists (i.e. is also nil), the effective default will be the system-wide default of 0.  A value of 0 will cause the step to be retried indefinitely unless the ErrorThreshold is reached. | [optional] 

## Methods

### NewPromotionStepRetry

`func NewPromotionStepRetry() *PromotionStepRetry`

NewPromotionStepRetry instantiates a new PromotionStepRetry object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPromotionStepRetryWithDefaults

`func NewPromotionStepRetryWithDefaults() *PromotionStepRetry`

NewPromotionStepRetryWithDefaults instantiates a new PromotionStepRetry object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetErrorThreshold

`func (o *PromotionStepRetry) GetErrorThreshold() int32`

GetErrorThreshold returns the ErrorThreshold field if non-nil, zero value otherwise.

### GetErrorThresholdOk

`func (o *PromotionStepRetry) GetErrorThresholdOk() (*int32, bool)`

GetErrorThresholdOk returns a tuple with the ErrorThreshold field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetErrorThreshold

`func (o *PromotionStepRetry) SetErrorThreshold(v int32)`

SetErrorThreshold sets ErrorThreshold field to given value.

### HasErrorThreshold

`func (o *PromotionStepRetry) HasErrorThreshold() bool`

HasErrorThreshold returns a boolean if a field has been set.

### GetTimeout

`func (o *PromotionStepRetry) GetTimeout() string`

GetTimeout returns the Timeout field if non-nil, zero value otherwise.

### GetTimeoutOk

`func (o *PromotionStepRetry) GetTimeoutOk() (*string, bool)`

GetTimeoutOk returns a tuple with the Timeout field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTimeout

`func (o *PromotionStepRetry) SetTimeout(v string)`

SetTimeout sets Timeout field to given value.

### HasTimeout

`func (o *PromotionStepRetry) HasTimeout() bool`

HasTimeout returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


