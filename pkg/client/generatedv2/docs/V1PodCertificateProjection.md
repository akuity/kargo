# V1PodCertificateProjection

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**CertificateChainPath** | Pointer to **string** | Write the certificate chain at this path in the projected volume.  Most applications should use credentialBundlePath.  When using keyPath and certificateChainPath, your application needs to check that the key and leaf certificate are consistent, because it is possible to read the files mid-rotation.  +optional | [optional] 
**CredentialBundlePath** | Pointer to **string** | Write the credential bundle at this path in the projected volume.  The credential bundle is a single file that contains multiple PEM blocks. The first PEM block is a PRIVATE KEY block, containing a PKCS#8 private key.  The remaining blocks are CERTIFICATE blocks, containing the issued certificate chain from the signer (leaf and any intermediates).  Using credentialBundlePath lets your Pod&#39;s application code make a single atomic read that retrieves a consistent key and certificate chain.  If you project them to separate files, your application code will need to additionally check that the leaf certificate was issued to the key.  +optional | [optional] 
**KeyPath** | Pointer to **string** | Write the key at this path in the projected volume.  Most applications should use credentialBundlePath.  When using keyPath and certificateChainPath, your application needs to check that the key and leaf certificate are consistent, because it is possible to read the files mid-rotation.  +optional | [optional] 
**KeyType** | Pointer to **string** | The type of keypair Kubelet will generate for the pod.  Valid values are \&quot;RSA3072\&quot;, \&quot;RSA4096\&quot;, \&quot;ECDSAP256\&quot;, \&quot;ECDSAP384\&quot;, \&quot;ECDSAP521\&quot;, and \&quot;ED25519\&quot;.  +required | [optional] 
**MaxExpirationSeconds** | Pointer to **int32** | maxExpirationSeconds is the maximum lifetime permitted for the certificate.  Kubelet copies this value verbatim into the PodCertificateRequests it generates for this projection.  If omitted, kube-apiserver will set it to 86400(24 hours). kube-apiserver will reject values shorter than 3600 (1 hour).  The maximum allowable value is 7862400 (91 days).  The signer implementation is then free to issue a certificate with any lifetime *shorter* than MaxExpirationSeconds, but no shorter than 3600 seconds (1 hour).  This constraint is enforced by kube-apiserver. &#x60;kubernetes.io&#x60; signers will never issue certificates with a lifetime longer than 24 hours.  +optional | [optional] 
**SignerName** | Pointer to **string** | Kubelet&#39;s generated CSRs will be addressed to this signer.  +required | [optional] 

## Methods

### NewV1PodCertificateProjection

`func NewV1PodCertificateProjection() *V1PodCertificateProjection`

NewV1PodCertificateProjection instantiates a new V1PodCertificateProjection object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1PodCertificateProjectionWithDefaults

`func NewV1PodCertificateProjectionWithDefaults() *V1PodCertificateProjection`

NewV1PodCertificateProjectionWithDefaults instantiates a new V1PodCertificateProjection object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetCertificateChainPath

`func (o *V1PodCertificateProjection) GetCertificateChainPath() string`

GetCertificateChainPath returns the CertificateChainPath field if non-nil, zero value otherwise.

### GetCertificateChainPathOk

`func (o *V1PodCertificateProjection) GetCertificateChainPathOk() (*string, bool)`

GetCertificateChainPathOk returns a tuple with the CertificateChainPath field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCertificateChainPath

`func (o *V1PodCertificateProjection) SetCertificateChainPath(v string)`

SetCertificateChainPath sets CertificateChainPath field to given value.

### HasCertificateChainPath

`func (o *V1PodCertificateProjection) HasCertificateChainPath() bool`

HasCertificateChainPath returns a boolean if a field has been set.

### GetCredentialBundlePath

`func (o *V1PodCertificateProjection) GetCredentialBundlePath() string`

GetCredentialBundlePath returns the CredentialBundlePath field if non-nil, zero value otherwise.

### GetCredentialBundlePathOk

`func (o *V1PodCertificateProjection) GetCredentialBundlePathOk() (*string, bool)`

GetCredentialBundlePathOk returns a tuple with the CredentialBundlePath field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCredentialBundlePath

`func (o *V1PodCertificateProjection) SetCredentialBundlePath(v string)`

SetCredentialBundlePath sets CredentialBundlePath field to given value.

### HasCredentialBundlePath

`func (o *V1PodCertificateProjection) HasCredentialBundlePath() bool`

HasCredentialBundlePath returns a boolean if a field has been set.

### GetKeyPath

`func (o *V1PodCertificateProjection) GetKeyPath() string`

GetKeyPath returns the KeyPath field if non-nil, zero value otherwise.

### GetKeyPathOk

`func (o *V1PodCertificateProjection) GetKeyPathOk() (*string, bool)`

GetKeyPathOk returns a tuple with the KeyPath field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKeyPath

`func (o *V1PodCertificateProjection) SetKeyPath(v string)`

SetKeyPath sets KeyPath field to given value.

### HasKeyPath

`func (o *V1PodCertificateProjection) HasKeyPath() bool`

HasKeyPath returns a boolean if a field has been set.

### GetKeyType

`func (o *V1PodCertificateProjection) GetKeyType() string`

GetKeyType returns the KeyType field if non-nil, zero value otherwise.

### GetKeyTypeOk

`func (o *V1PodCertificateProjection) GetKeyTypeOk() (*string, bool)`

GetKeyTypeOk returns a tuple with the KeyType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKeyType

`func (o *V1PodCertificateProjection) SetKeyType(v string)`

SetKeyType sets KeyType field to given value.

### HasKeyType

`func (o *V1PodCertificateProjection) HasKeyType() bool`

HasKeyType returns a boolean if a field has been set.

### GetMaxExpirationSeconds

`func (o *V1PodCertificateProjection) GetMaxExpirationSeconds() int32`

GetMaxExpirationSeconds returns the MaxExpirationSeconds field if non-nil, zero value otherwise.

### GetMaxExpirationSecondsOk

`func (o *V1PodCertificateProjection) GetMaxExpirationSecondsOk() (*int32, bool)`

GetMaxExpirationSecondsOk returns a tuple with the MaxExpirationSeconds field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMaxExpirationSeconds

`func (o *V1PodCertificateProjection) SetMaxExpirationSeconds(v int32)`

SetMaxExpirationSeconds sets MaxExpirationSeconds field to given value.

### HasMaxExpirationSeconds

`func (o *V1PodCertificateProjection) HasMaxExpirationSeconds() bool`

HasMaxExpirationSeconds returns a boolean if a field has been set.

### GetSignerName

`func (o *V1PodCertificateProjection) GetSignerName() string`

GetSignerName returns the SignerName field if non-nil, zero value otherwise.

### GetSignerNameOk

`func (o *V1PodCertificateProjection) GetSignerNameOk() (*string, bool)`

GetSignerNameOk returns a tuple with the SignerName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSignerName

`func (o *V1PodCertificateProjection) SetSignerName(v string)`

SetSignerName sets SignerName field to given value.

### HasSignerName

`func (o *V1PodCertificateProjection) HasSignerName() bool`

HasSignerName returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


