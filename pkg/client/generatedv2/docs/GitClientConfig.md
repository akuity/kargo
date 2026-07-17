# GitClientConfig

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Email** | **string** | Email is the email address used for Git commits made by Kargo. +kubebuilder:validation:Required +kubebuilder:validation:MinLength&#x3D;1 +kubebuilder:validation:Format&#x3D;\&quot;email\&quot; | 
**Name** | **string** | Name is the name used for Git commits made by Kargo. +kubebuilder:validation:Required +kubebuilder:validation:MinLength&#x3D;1 | 
**SigningKeySecret** | Pointer to [**V1LocalObjectReference**](V1LocalObjectReference.md) | SigningKeySecret references a Secret in the system namespace containing a GPG signing key for commit signing. The Secret must contain a data key named \&quot;signingKey\&quot; with the GPG private key material. +optional | [optional] 

## Methods

### NewGitClientConfig

`func NewGitClientConfig(email string, name string, ) *GitClientConfig`

NewGitClientConfig instantiates a new GitClientConfig object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewGitClientConfigWithDefaults

`func NewGitClientConfigWithDefaults() *GitClientConfig`

NewGitClientConfigWithDefaults instantiates a new GitClientConfig object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetEmail

`func (o *GitClientConfig) GetEmail() string`

GetEmail returns the Email field if non-nil, zero value otherwise.

### GetEmailOk

`func (o *GitClientConfig) GetEmailOk() (*string, bool)`

GetEmailOk returns a tuple with the Email field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEmail

`func (o *GitClientConfig) SetEmail(v string)`

SetEmail sets Email field to given value.


### GetName

`func (o *GitClientConfig) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *GitClientConfig) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *GitClientConfig) SetName(v string)`

SetName sets Name field to given value.


### GetSigningKeySecret

`func (o *GitClientConfig) GetSigningKeySecret() V1LocalObjectReference`

GetSigningKeySecret returns the SigningKeySecret field if non-nil, zero value otherwise.

### GetSigningKeySecretOk

`func (o *GitClientConfig) GetSigningKeySecretOk() (*V1LocalObjectReference, bool)`

GetSigningKeySecretOk returns a tuple with the SigningKeySecret field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSigningKeySecret

`func (o *GitClientConfig) SetSigningKeySecret(v V1LocalObjectReference)`

SetSigningKeySecret sets SigningKeySecret field to given value.

### HasSigningKeySecret

`func (o *GitClientConfig) HasSigningKeySecret() bool`

HasSigningKeySecret returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


