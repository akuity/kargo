# \ResourcesAPI

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CreateResource**](ResourcesAPI.md#CreateResource) | **Post** /v1beta1/resources | Create resources
[**DeleteResource**](ResourcesAPI.md#DeleteResource) | **Delete** /v1beta1/resources | Delete resources
[**UpdateResource**](ResourcesAPI.md#UpdateResource) | **Put** /v1beta1/resources | Update resources



## CreateResource

> CreateResourceResponse CreateResource(ctx).Manifest(manifest).Execute()

Create resources



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/akuity/kargo"
)

func main() {
	manifest := "manifest_example" // string | YAML or JSON manifest(s)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.ResourcesAPI.CreateResource(context.Background()).Manifest(manifest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `ResourcesAPI.CreateResource``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `CreateResource`: CreateResourceResponse
	fmt.Fprintf(os.Stdout, "Response from `ResourcesAPI.CreateResource`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiCreateResourceRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **manifest** | **string** | YAML or JSON manifest(s) | 

### Return type

[**CreateResourceResponse**](CreateResourceResponse.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: text/plain
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DeleteResource

> DeleteResourceResponse DeleteResource(ctx).Manifest(manifest).Execute()

Delete resources



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/akuity/kargo"
)

func main() {
	manifest := "manifest_example" // string | YAML or JSON manifest(s)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.ResourcesAPI.DeleteResource(context.Background()).Manifest(manifest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `ResourcesAPI.DeleteResource``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `DeleteResource`: DeleteResourceResponse
	fmt.Fprintf(os.Stdout, "Response from `ResourcesAPI.DeleteResource`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiDeleteResourceRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **manifest** | **string** | YAML or JSON manifest(s) | 

### Return type

[**DeleteResourceResponse**](DeleteResourceResponse.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: text/plain
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## UpdateResource

> CreateOrUpdateResourceResponse UpdateResource(ctx).Manifest(manifest).Upsert(upsert).Execute()

Update resources



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/akuity/kargo"
)

func main() {
	manifest := "manifest_example" // string | YAML or JSON manifest(s)
	upsert := true // bool | If true, create a resource if it does not exist (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.ResourcesAPI.UpdateResource(context.Background()).Manifest(manifest).Upsert(upsert).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `ResourcesAPI.UpdateResource``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `UpdateResource`: CreateOrUpdateResourceResponse
	fmt.Fprintf(os.Stdout, "Response from `ResourcesAPI.UpdateResource`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiUpdateResourceRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **manifest** | **string** | YAML or JSON manifest(s) | 
 **upsert** | **bool** | If true, create a resource if it does not exist | 

### Return type

[**CreateOrUpdateResourceResponse**](CreateOrUpdateResourceResponse.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: text/plain
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

