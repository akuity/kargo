# V1VolumeProjection

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ClusterTrustBundle** | Pointer to [**V1ClusterTrustBundleProjection**](V1ClusterTrustBundleProjection.md) | ClusterTrustBundle allows a pod to access the &#x60;.spec.trustBundle&#x60; field of ClusterTrustBundle objects in an auto-updating file.  Alpha, gated by the ClusterTrustBundleProjection feature gate.  ClusterTrustBundle objects can either be selected by name, or by the combination of signer name and a label selector.  Kubelet performs aggressive normalization of the PEM contents written into the pod filesystem.  Esoteric PEM features such as inter-block comments and block headers are stripped.  Certificates are deduplicated. The ordering of certificates within the file is arbitrary, and Kubelet may change the order over time.  +featureGate&#x3D;ClusterTrustBundleProjection +optional | [optional] 
**ConfigMap** | Pointer to [**V1ConfigMapProjection**](V1ConfigMapProjection.md) | configMap information about the configMap data to project +optional | [optional] 
**DownwardAPI** | Pointer to [**V1DownwardAPIProjection**](V1DownwardAPIProjection.md) | downwardAPI information about the downwardAPI data to project +optional | [optional] 
**PodCertificate** | Pointer to [**V1PodCertificateProjection**](V1PodCertificateProjection.md) | Projects an auto-rotating credential bundle (private key and certificate chain) that the pod can use either as a TLS client or server.  Kubelet generates a private key and uses it to send a PodCertificateRequest to the named signer.  Once the signer approves the request and issues a certificate chain, Kubelet writes the key and certificate chain to the pod filesystem.  The pod does not start until certificates have been issued for each podCertificate projected volume source in its spec.  Kubelet will begin trying to rotate the certificate at the time indicated by the signer using the PodCertificateRequest.Status.BeginRefreshAt timestamp.  Kubelet can write a single file, indicated by the credentialBundlePath field, or separate files, indicated by the keyPath and certificateChainPath fields.  The credential bundle is a single file in PEM format.  The first PEM entry is the private key (in PKCS#8 format), and the remaining PEM entries are the certificate chain issued by the signer (typically, signers will return their certificate chain in leaf-to-root order).  Prefer using the credential bundle format, since your application code can read it atomically.  If you use keyPath and certificateChainPath, your application must make two separate file reads. If these coincide with a certificate rotation, it is possible that the private key and leaf certificate you read may not correspond to each other.  Your application will need to check for this condition, and re-read until they are consistent.  The named signer controls chooses the format of the certificate it issues; consult the signer implementation&#39;s documentation to learn how to use the certificates it issues.  +featureGate&#x3D;PodCertificateProjection +optional | [optional] 
**Secret** | Pointer to [**V1SecretProjection**](V1SecretProjection.md) | secret information about the secret data to project +optional | [optional] 
**ServiceAccountToken** | Pointer to [**V1ServiceAccountTokenProjection**](V1ServiceAccountTokenProjection.md) | serviceAccountToken is information about the serviceAccountToken data to project +optional | [optional] 

## Methods

### NewV1VolumeProjection

`func NewV1VolumeProjection() *V1VolumeProjection`

NewV1VolumeProjection instantiates a new V1VolumeProjection object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1VolumeProjectionWithDefaults

`func NewV1VolumeProjectionWithDefaults() *V1VolumeProjection`

NewV1VolumeProjectionWithDefaults instantiates a new V1VolumeProjection object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetClusterTrustBundle

`func (o *V1VolumeProjection) GetClusterTrustBundle() V1ClusterTrustBundleProjection`

GetClusterTrustBundle returns the ClusterTrustBundle field if non-nil, zero value otherwise.

### GetClusterTrustBundleOk

`func (o *V1VolumeProjection) GetClusterTrustBundleOk() (*V1ClusterTrustBundleProjection, bool)`

GetClusterTrustBundleOk returns a tuple with the ClusterTrustBundle field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetClusterTrustBundle

`func (o *V1VolumeProjection) SetClusterTrustBundle(v V1ClusterTrustBundleProjection)`

SetClusterTrustBundle sets ClusterTrustBundle field to given value.

### HasClusterTrustBundle

`func (o *V1VolumeProjection) HasClusterTrustBundle() bool`

HasClusterTrustBundle returns a boolean if a field has been set.

### GetConfigMap

`func (o *V1VolumeProjection) GetConfigMap() V1ConfigMapProjection`

GetConfigMap returns the ConfigMap field if non-nil, zero value otherwise.

### GetConfigMapOk

`func (o *V1VolumeProjection) GetConfigMapOk() (*V1ConfigMapProjection, bool)`

GetConfigMapOk returns a tuple with the ConfigMap field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConfigMap

`func (o *V1VolumeProjection) SetConfigMap(v V1ConfigMapProjection)`

SetConfigMap sets ConfigMap field to given value.

### HasConfigMap

`func (o *V1VolumeProjection) HasConfigMap() bool`

HasConfigMap returns a boolean if a field has been set.

### GetDownwardAPI

`func (o *V1VolumeProjection) GetDownwardAPI() V1DownwardAPIProjection`

GetDownwardAPI returns the DownwardAPI field if non-nil, zero value otherwise.

### GetDownwardAPIOk

`func (o *V1VolumeProjection) GetDownwardAPIOk() (*V1DownwardAPIProjection, bool)`

GetDownwardAPIOk returns a tuple with the DownwardAPI field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDownwardAPI

`func (o *V1VolumeProjection) SetDownwardAPI(v V1DownwardAPIProjection)`

SetDownwardAPI sets DownwardAPI field to given value.

### HasDownwardAPI

`func (o *V1VolumeProjection) HasDownwardAPI() bool`

HasDownwardAPI returns a boolean if a field has been set.

### GetPodCertificate

`func (o *V1VolumeProjection) GetPodCertificate() V1PodCertificateProjection`

GetPodCertificate returns the PodCertificate field if non-nil, zero value otherwise.

### GetPodCertificateOk

`func (o *V1VolumeProjection) GetPodCertificateOk() (*V1PodCertificateProjection, bool)`

GetPodCertificateOk returns a tuple with the PodCertificate field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPodCertificate

`func (o *V1VolumeProjection) SetPodCertificate(v V1PodCertificateProjection)`

SetPodCertificate sets PodCertificate field to given value.

### HasPodCertificate

`func (o *V1VolumeProjection) HasPodCertificate() bool`

HasPodCertificate returns a boolean if a field has been set.

### GetSecret

`func (o *V1VolumeProjection) GetSecret() V1SecretProjection`

GetSecret returns the Secret field if non-nil, zero value otherwise.

### GetSecretOk

`func (o *V1VolumeProjection) GetSecretOk() (*V1SecretProjection, bool)`

GetSecretOk returns a tuple with the Secret field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecret

`func (o *V1VolumeProjection) SetSecret(v V1SecretProjection)`

SetSecret sets Secret field to given value.

### HasSecret

`func (o *V1VolumeProjection) HasSecret() bool`

HasSecret returns a boolean if a field has been set.

### GetServiceAccountToken

`func (o *V1VolumeProjection) GetServiceAccountToken() V1ServiceAccountTokenProjection`

GetServiceAccountToken returns the ServiceAccountToken field if non-nil, zero value otherwise.

### GetServiceAccountTokenOk

`func (o *V1VolumeProjection) GetServiceAccountTokenOk() (*V1ServiceAccountTokenProjection, bool)`

GetServiceAccountTokenOk returns a tuple with the ServiceAccountToken field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetServiceAccountToken

`func (o *V1VolumeProjection) SetServiceAccountToken(v V1ServiceAccountTokenProjection)`

SetServiceAccountToken sets ServiceAccountToken field to given value.

### HasServiceAccountToken

`func (o *V1VolumeProjection) HasServiceAccountToken() bool`

HasServiceAccountToken returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


