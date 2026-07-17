# DeepLink

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Description** | Pointer to **string** | Description is an optional human-readable summary shown alongside the link.  +optional | [optional] 
**If** | Pointer to **string** | If is an optional expression condition. When set, the link is only shown when the expression evaluates to true.  +optional | [optional] 
**Title** | Pointer to **string** | Title is the display label for the link.  +kubebuilder:validation:MinLength&#x3D;1 | [optional] 
**Url** | Pointer to **string** | URL is an expression that resolves to the link&#39;s href.  +kubebuilder:validation:MinLength&#x3D;1 | [optional] 

## Methods

### NewDeepLink

`func NewDeepLink() *DeepLink`

NewDeepLink instantiates a new DeepLink object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewDeepLinkWithDefaults

`func NewDeepLinkWithDefaults() *DeepLink`

NewDeepLinkWithDefaults instantiates a new DeepLink object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDescription

`func (o *DeepLink) GetDescription() string`

GetDescription returns the Description field if non-nil, zero value otherwise.

### GetDescriptionOk

`func (o *DeepLink) GetDescriptionOk() (*string, bool)`

GetDescriptionOk returns a tuple with the Description field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDescription

`func (o *DeepLink) SetDescription(v string)`

SetDescription sets Description field to given value.

### HasDescription

`func (o *DeepLink) HasDescription() bool`

HasDescription returns a boolean if a field has been set.

### GetIf

`func (o *DeepLink) GetIf() string`

GetIf returns the If field if non-nil, zero value otherwise.

### GetIfOk

`func (o *DeepLink) GetIfOk() (*string, bool)`

GetIfOk returns a tuple with the If field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIf

`func (o *DeepLink) SetIf(v string)`

SetIf sets If field to given value.

### HasIf

`func (o *DeepLink) HasIf() bool`

HasIf returns a boolean if a field has been set.

### GetTitle

`func (o *DeepLink) GetTitle() string`

GetTitle returns the Title field if non-nil, zero value otherwise.

### GetTitleOk

`func (o *DeepLink) GetTitleOk() (*string, bool)`

GetTitleOk returns a tuple with the Title field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTitle

`func (o *DeepLink) SetTitle(v string)`

SetTitle sets Title field to given value.

### HasTitle

`func (o *DeepLink) HasTitle() bool`

HasTitle returns a boolean if a field has been set.

### GetUrl

`func (o *DeepLink) GetUrl() string`

GetUrl returns the Url field if non-nil, zero value otherwise.

### GetUrlOk

`func (o *DeepLink) GetUrlOk() (*string, bool)`

GetUrlOk returns a tuple with the Url field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUrl

`func (o *DeepLink) SetUrl(v string)`

SetUrl sets Url field to given value.

### HasUrl

`func (o *DeepLink) HasUrl() bool`

HasUrl returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


