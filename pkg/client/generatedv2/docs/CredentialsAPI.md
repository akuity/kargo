# \CredentialsAPI

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CreateProjectGenericCredentials**](CredentialsAPI.md#CreateProjectGenericCredentials) | **Post** /v1beta1/projects/{project}/generic-credentials | Create project-level generic credentials
[**CreateProjectRepoCredentials**](CredentialsAPI.md#CreateProjectRepoCredentials) | **Post** /v1beta1/projects/{project}/repo-credentials | Create project-level repository credentials
[**CreateSharedGenericCredentials**](CredentialsAPI.md#CreateSharedGenericCredentials) | **Post** /v1beta1/shared/generic-credentials | Create shared generic credentials
[**CreateSharedRepoCredentials**](CredentialsAPI.md#CreateSharedRepoCredentials) | **Post** /v1beta1/shared/repo-credentials | Create shared repository credentials
[**CreateSystemGenericCredentials**](CredentialsAPI.md#CreateSystemGenericCredentials) | **Post** /v1beta1/system/generic-credentials | Create system-level generic credentials
[**DeleteProjectGenericCredentials**](CredentialsAPI.md#DeleteProjectGenericCredentials) | **Delete** /v1beta1/projects/{project}/generic-credentials/{generic-credentials} | Delete project-level generic credentials
[**DeleteProjectRepoCredentials**](CredentialsAPI.md#DeleteProjectRepoCredentials) | **Delete** /v1beta1/projects/{project}/repo-credentials/{repo-credentials} | Delete project-level repository credentials
[**DeleteSharedGenericCredentials**](CredentialsAPI.md#DeleteSharedGenericCredentials) | **Delete** /v1beta1/shared/generic-credentials/{generic-credentials} | Delete shared generic credentials
[**DeleteSharedRepoCredentials**](CredentialsAPI.md#DeleteSharedRepoCredentials) | **Delete** /v1beta1/shared/repo-credentials/{repo-credentials} | Delete shared repository credentials
[**DeleteSystemGenericCredentials**](CredentialsAPI.md#DeleteSystemGenericCredentials) | **Delete** /v1beta1/system/generic-credentials/{generic-credentials} | Delete system-level generic credentials
[**GetProjectGenericCredentials**](CredentialsAPI.md#GetProjectGenericCredentials) | **Get** /v1beta1/projects/{project}/generic-credentials/{generic-credentials} | Retrieve project-level generic credentials
[**GetProjectRepoCredentials**](CredentialsAPI.md#GetProjectRepoCredentials) | **Get** /v1beta1/projects/{project}/repo-credentials/{repo-credentials} | Retrieve project-level repository credentials
[**GetSharedGenericCredentials**](CredentialsAPI.md#GetSharedGenericCredentials) | **Get** /v1beta1/shared/generic-credentials/{generic-credentials} | Retrieve shared generic credentials
[**GetSharedRepoCredentials**](CredentialsAPI.md#GetSharedRepoCredentials) | **Get** /v1beta1/shared/repo-credentials/{repo-credentials} | Retrieve shared repository credentials
[**GetSystemGenericCredentials**](CredentialsAPI.md#GetSystemGenericCredentials) | **Get** /v1beta1/system/generic-credentials/{generic-credentials} | Retrieve system-level generic credentials
[**ListProjectGenericCredentials**](CredentialsAPI.md#ListProjectGenericCredentials) | **Get** /v1beta1/projects/{project}/generic-credentials | List project-level generic credentials
[**ListProjectRepoCredentials**](CredentialsAPI.md#ListProjectRepoCredentials) | **Get** /v1beta1/projects/{project}/repo-credentials | List project-level repository credentials
[**ListSharedGenericCredentials**](CredentialsAPI.md#ListSharedGenericCredentials) | **Get** /v1beta1/shared/generic-credentials | List shared generic credentials
[**ListSharedRepoCredentials**](CredentialsAPI.md#ListSharedRepoCredentials) | **Get** /v1beta1/shared/repo-credentials | List shared repository credentials
[**ListSystemGenericCredentials**](CredentialsAPI.md#ListSystemGenericCredentials) | **Get** /v1beta1/system/generic-credentials | List system-level generic credentials
[**PatchProjectGenericCredentials**](CredentialsAPI.md#PatchProjectGenericCredentials) | **Patch** /v1beta1/projects/{project}/generic-credentials/{generic-credentials} | Patch project-level generic credentials
[**PatchProjectRepoCredentials**](CredentialsAPI.md#PatchProjectRepoCredentials) | **Patch** /v1beta1/projects/{project}/repo-credentials/{repo-credentials} | Patch project-level repository credentials
[**PatchSharedGenericCredentials**](CredentialsAPI.md#PatchSharedGenericCredentials) | **Patch** /v1beta1/shared/generic-credentials/{generic-credentials} | Patch shared generic credentials
[**PatchSharedRepoCredentials**](CredentialsAPI.md#PatchSharedRepoCredentials) | **Patch** /v1beta1/shared/repo-credentials/{repo-credentials} | Patch shared repository credentials
[**PatchSystemGenericCredentials**](CredentialsAPI.md#PatchSystemGenericCredentials) | **Patch** /v1beta1/system/generic-credentials/{generic-credentials} | Patch system-level generic credentials
[**UpdateProjectGenericCredentials**](CredentialsAPI.md#UpdateProjectGenericCredentials) | **Put** /v1beta1/projects/{project}/generic-credentials/{generic-credentials} | Replace project-level generic credentials
[**UpdateProjectRepoCredentials**](CredentialsAPI.md#UpdateProjectRepoCredentials) | **Put** /v1beta1/projects/{project}/repo-credentials/{repo-credentials} | Replace project-level repository credentials
[**UpdateSharedGenericCredentials**](CredentialsAPI.md#UpdateSharedGenericCredentials) | **Put** /v1beta1/shared/generic-credentials/{generic-credentials} | Replace shared generic credentials
[**UpdateSharedRepoCredentials**](CredentialsAPI.md#UpdateSharedRepoCredentials) | **Put** /v1beta1/shared/repo-credentials/{repo-credentials} | Replace shared repository credentials
[**UpdateSystemGenericCredentials**](CredentialsAPI.md#UpdateSystemGenericCredentials) | **Put** /v1beta1/system/generic-credentials/{generic-credentials} | Replace system-level generic credentials



## CreateProjectGenericCredentials

> V1Secret CreateProjectGenericCredentials(ctx, project).Body(body).Execute()

Create project-level generic credentials



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
	project := "project_example" // string | Project name
	body := *openapiclient.NewCreateGenericCredentialsRequest() // CreateGenericCredentialsRequest | Generic credentials

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CredentialsAPI.CreateProjectGenericCredentials(context.Background(), project).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CredentialsAPI.CreateProjectGenericCredentials``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `CreateProjectGenericCredentials`: V1Secret
	fmt.Fprintf(os.Stdout, "Response from `CredentialsAPI.CreateProjectGenericCredentials`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 

### Other Parameters

Other parameters are passed through a pointer to a apiCreateProjectGenericCredentialsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**CreateGenericCredentialsRequest**](CreateGenericCredentialsRequest.md) | Generic credentials | 

### Return type

[**V1Secret**](V1Secret.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## CreateProjectRepoCredentials

> V1Secret CreateProjectRepoCredentials(ctx, project).Body(body).Execute()

Create project-level repository credentials



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
	project := "project_example" // string | Project name
	body := *openapiclient.NewCreateRepoCredentialsRequest() // CreateRepoCredentialsRequest | Credentials

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CredentialsAPI.CreateProjectRepoCredentials(context.Background(), project).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CredentialsAPI.CreateProjectRepoCredentials``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `CreateProjectRepoCredentials`: V1Secret
	fmt.Fprintf(os.Stdout, "Response from `CredentialsAPI.CreateProjectRepoCredentials`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 

### Other Parameters

Other parameters are passed through a pointer to a apiCreateProjectRepoCredentialsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**CreateRepoCredentialsRequest**](CreateRepoCredentialsRequest.md) | Credentials | 

### Return type

[**V1Secret**](V1Secret.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## CreateSharedGenericCredentials

> V1Secret CreateSharedGenericCredentials(ctx).Body(body).Execute()

Create shared generic credentials



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
	body := *openapiclient.NewCreateGenericCredentialsRequest() // CreateGenericCredentialsRequest | Generic credentials

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CredentialsAPI.CreateSharedGenericCredentials(context.Background()).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CredentialsAPI.CreateSharedGenericCredentials``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `CreateSharedGenericCredentials`: V1Secret
	fmt.Fprintf(os.Stdout, "Response from `CredentialsAPI.CreateSharedGenericCredentials`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiCreateSharedGenericCredentialsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**CreateGenericCredentialsRequest**](CreateGenericCredentialsRequest.md) | Generic credentials | 

### Return type

[**V1Secret**](V1Secret.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## CreateSharedRepoCredentials

> V1Secret CreateSharedRepoCredentials(ctx).Body(body).Execute()

Create shared repository credentials



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
	body := *openapiclient.NewCreateRepoCredentialsRequest() // CreateRepoCredentialsRequest | Credentials

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CredentialsAPI.CreateSharedRepoCredentials(context.Background()).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CredentialsAPI.CreateSharedRepoCredentials``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `CreateSharedRepoCredentials`: V1Secret
	fmt.Fprintf(os.Stdout, "Response from `CredentialsAPI.CreateSharedRepoCredentials`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiCreateSharedRepoCredentialsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**CreateRepoCredentialsRequest**](CreateRepoCredentialsRequest.md) | Credentials | 

### Return type

[**V1Secret**](V1Secret.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## CreateSystemGenericCredentials

> V1Secret CreateSystemGenericCredentials(ctx).Body(body).Execute()

Create system-level generic credentials



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
	body := *openapiclient.NewCreateGenericCredentialsRequest() // CreateGenericCredentialsRequest | Generic credentials

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CredentialsAPI.CreateSystemGenericCredentials(context.Background()).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CredentialsAPI.CreateSystemGenericCredentials``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `CreateSystemGenericCredentials`: V1Secret
	fmt.Fprintf(os.Stdout, "Response from `CredentialsAPI.CreateSystemGenericCredentials`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiCreateSystemGenericCredentialsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**CreateGenericCredentialsRequest**](CreateGenericCredentialsRequest.md) | Generic credentials | 

### Return type

[**V1Secret**](V1Secret.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DeleteProjectGenericCredentials

> DeleteProjectGenericCredentials(ctx, project, genericCredentials).Execute()

Delete project-level generic credentials



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
	project := "project_example" // string | Project name
	genericCredentials := "genericCredentials_example" // string | Generic credentials name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.CredentialsAPI.DeleteProjectGenericCredentials(context.Background(), project, genericCredentials).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CredentialsAPI.DeleteProjectGenericCredentials``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**genericCredentials** | **string** | Generic credentials name | 

### Other Parameters

Other parameters are passed through a pointer to a apiDeleteProjectGenericCredentialsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



### Return type

 (empty response body)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DeleteProjectRepoCredentials

> DeleteProjectRepoCredentials(ctx, project, repoCredentials).Execute()

Delete project-level repository credentials



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
	project := "project_example" // string | Project name
	repoCredentials := "repoCredentials_example" // string | Credentials name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.CredentialsAPI.DeleteProjectRepoCredentials(context.Background(), project, repoCredentials).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CredentialsAPI.DeleteProjectRepoCredentials``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**repoCredentials** | **string** | Credentials name | 

### Other Parameters

Other parameters are passed through a pointer to a apiDeleteProjectRepoCredentialsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



### Return type

 (empty response body)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DeleteSharedGenericCredentials

> DeleteSharedGenericCredentials(ctx, genericCredentials).Execute()

Delete shared generic credentials



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
	genericCredentials := "genericCredentials_example" // string | Generic credentials name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.CredentialsAPI.DeleteSharedGenericCredentials(context.Background(), genericCredentials).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CredentialsAPI.DeleteSharedGenericCredentials``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**genericCredentials** | **string** | Generic credentials name | 

### Other Parameters

Other parameters are passed through a pointer to a apiDeleteSharedGenericCredentialsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

 (empty response body)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DeleteSharedRepoCredentials

> DeleteSharedRepoCredentials(ctx, repoCredentials).Execute()

Delete shared repository credentials



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
	repoCredentials := "repoCredentials_example" // string | Credentials name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.CredentialsAPI.DeleteSharedRepoCredentials(context.Background(), repoCredentials).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CredentialsAPI.DeleteSharedRepoCredentials``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**repoCredentials** | **string** | Credentials name | 

### Other Parameters

Other parameters are passed through a pointer to a apiDeleteSharedRepoCredentialsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

 (empty response body)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DeleteSystemGenericCredentials

> DeleteSystemGenericCredentials(ctx, genericCredentials).Execute()

Delete system-level generic credentials



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
	genericCredentials := "genericCredentials_example" // string | Generic credentials name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.CredentialsAPI.DeleteSystemGenericCredentials(context.Background(), genericCredentials).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CredentialsAPI.DeleteSystemGenericCredentials``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**genericCredentials** | **string** | Generic credentials name | 

### Other Parameters

Other parameters are passed through a pointer to a apiDeleteSystemGenericCredentialsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

 (empty response body)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetProjectGenericCredentials

> V1Secret GetProjectGenericCredentials(ctx, project, genericCredentials).Execute()

Retrieve project-level generic credentials



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
	project := "project_example" // string | Project name
	genericCredentials := "genericCredentials_example" // string | Credentials name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CredentialsAPI.GetProjectGenericCredentials(context.Background(), project, genericCredentials).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CredentialsAPI.GetProjectGenericCredentials``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetProjectGenericCredentials`: V1Secret
	fmt.Fprintf(os.Stdout, "Response from `CredentialsAPI.GetProjectGenericCredentials`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**genericCredentials** | **string** | Credentials name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetProjectGenericCredentialsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



### Return type

[**V1Secret**](V1Secret.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetProjectRepoCredentials

> V1Secret GetProjectRepoCredentials(ctx, project, repoCredentials).Execute()

Retrieve project-level repository credentials



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
	project := "project_example" // string | Project name
	repoCredentials := "repoCredentials_example" // string | Credentials name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CredentialsAPI.GetProjectRepoCredentials(context.Background(), project, repoCredentials).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CredentialsAPI.GetProjectRepoCredentials``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetProjectRepoCredentials`: V1Secret
	fmt.Fprintf(os.Stdout, "Response from `CredentialsAPI.GetProjectRepoCredentials`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**repoCredentials** | **string** | Credentials name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetProjectRepoCredentialsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



### Return type

[**V1Secret**](V1Secret.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetSharedGenericCredentials

> V1Secret GetSharedGenericCredentials(ctx, genericCredentials).Execute()

Retrieve shared generic credentials



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
	genericCredentials := "genericCredentials_example" // string | Credentials name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CredentialsAPI.GetSharedGenericCredentials(context.Background(), genericCredentials).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CredentialsAPI.GetSharedGenericCredentials``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetSharedGenericCredentials`: V1Secret
	fmt.Fprintf(os.Stdout, "Response from `CredentialsAPI.GetSharedGenericCredentials`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**genericCredentials** | **string** | Credentials name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetSharedGenericCredentialsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**V1Secret**](V1Secret.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetSharedRepoCredentials

> V1Secret GetSharedRepoCredentials(ctx, repoCredentials).Execute()

Retrieve shared repository credentials



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
	repoCredentials := "repoCredentials_example" // string | Credentials name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CredentialsAPI.GetSharedRepoCredentials(context.Background(), repoCredentials).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CredentialsAPI.GetSharedRepoCredentials``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetSharedRepoCredentials`: V1Secret
	fmt.Fprintf(os.Stdout, "Response from `CredentialsAPI.GetSharedRepoCredentials`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**repoCredentials** | **string** | Credentials name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetSharedRepoCredentialsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**V1Secret**](V1Secret.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetSystemGenericCredentials

> V1Secret GetSystemGenericCredentials(ctx, genericCredentials).Execute()

Retrieve system-level generic credentials



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
	genericCredentials := "genericCredentials_example" // string | Credentials name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CredentialsAPI.GetSystemGenericCredentials(context.Background(), genericCredentials).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CredentialsAPI.GetSystemGenericCredentials``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetSystemGenericCredentials`: V1Secret
	fmt.Fprintf(os.Stdout, "Response from `CredentialsAPI.GetSystemGenericCredentials`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**genericCredentials** | **string** | Credentials name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetSystemGenericCredentialsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**V1Secret**](V1Secret.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListProjectGenericCredentials

> V1SecretList ListProjectGenericCredentials(ctx, project).Execute()

List project-level generic credentials



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
	project := "project_example" // string | Project name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CredentialsAPI.ListProjectGenericCredentials(context.Background(), project).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CredentialsAPI.ListProjectGenericCredentials``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListProjectGenericCredentials`: V1SecretList
	fmt.Fprintf(os.Stdout, "Response from `CredentialsAPI.ListProjectGenericCredentials`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 

### Other Parameters

Other parameters are passed through a pointer to a apiListProjectGenericCredentialsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**V1SecretList**](V1SecretList.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListProjectRepoCredentials

> V1SecretList ListProjectRepoCredentials(ctx, project).Execute()

List project-level repository credentials



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
	project := "project_example" // string | Project name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CredentialsAPI.ListProjectRepoCredentials(context.Background(), project).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CredentialsAPI.ListProjectRepoCredentials``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListProjectRepoCredentials`: V1SecretList
	fmt.Fprintf(os.Stdout, "Response from `CredentialsAPI.ListProjectRepoCredentials`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 

### Other Parameters

Other parameters are passed through a pointer to a apiListProjectRepoCredentialsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**V1SecretList**](V1SecretList.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListSharedGenericCredentials

> V1SecretList ListSharedGenericCredentials(ctx).Execute()

List shared generic credentials



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

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CredentialsAPI.ListSharedGenericCredentials(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CredentialsAPI.ListSharedGenericCredentials``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListSharedGenericCredentials`: V1SecretList
	fmt.Fprintf(os.Stdout, "Response from `CredentialsAPI.ListSharedGenericCredentials`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiListSharedGenericCredentialsRequest struct via the builder pattern


### Return type

[**V1SecretList**](V1SecretList.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListSharedRepoCredentials

> V1SecretList ListSharedRepoCredentials(ctx).Execute()

List shared repository credentials



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

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CredentialsAPI.ListSharedRepoCredentials(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CredentialsAPI.ListSharedRepoCredentials``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListSharedRepoCredentials`: V1SecretList
	fmt.Fprintf(os.Stdout, "Response from `CredentialsAPI.ListSharedRepoCredentials`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiListSharedRepoCredentialsRequest struct via the builder pattern


### Return type

[**V1SecretList**](V1SecretList.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListSystemGenericCredentials

> V1SecretList ListSystemGenericCredentials(ctx).Execute()

List system-level generic credentials



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

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CredentialsAPI.ListSystemGenericCredentials(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CredentialsAPI.ListSystemGenericCredentials``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListSystemGenericCredentials`: V1SecretList
	fmt.Fprintf(os.Stdout, "Response from `CredentialsAPI.ListSystemGenericCredentials`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiListSystemGenericCredentialsRequest struct via the builder pattern


### Return type

[**V1SecretList**](V1SecretList.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PatchProjectGenericCredentials

> V1Secret PatchProjectGenericCredentials(ctx, project, genericCredentials).Body(body).Execute()

Patch project-level generic credentials



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
	project := "project_example" // string | Project name
	genericCredentials := "genericCredentials_example" // string | Generic credentials name
	body := *openapiclient.NewPatchGenericCredentialsRequest() // PatchGenericCredentialsRequest | GenericCredentials patch

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CredentialsAPI.PatchProjectGenericCredentials(context.Background(), project, genericCredentials).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CredentialsAPI.PatchProjectGenericCredentials``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PatchProjectGenericCredentials`: V1Secret
	fmt.Fprintf(os.Stdout, "Response from `CredentialsAPI.PatchProjectGenericCredentials`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**genericCredentials** | **string** | Generic credentials name | 

### Other Parameters

Other parameters are passed through a pointer to a apiPatchProjectGenericCredentialsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **body** | [**PatchGenericCredentialsRequest**](PatchGenericCredentialsRequest.md) | GenericCredentials patch | 

### Return type

[**V1Secret**](V1Secret.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PatchProjectRepoCredentials

> V1Secret PatchProjectRepoCredentials(ctx, project, repoCredentials).Body(body).Execute()

Patch project-level repository credentials



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
	project := "project_example" // string | Project name
	repoCredentials := "repoCredentials_example" // string | Repo credentials name
	body := *openapiclient.NewPatchRepoCredentialsRequest() // PatchRepoCredentialsRequest | Credentials

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CredentialsAPI.PatchProjectRepoCredentials(context.Background(), project, repoCredentials).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CredentialsAPI.PatchProjectRepoCredentials``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PatchProjectRepoCredentials`: V1Secret
	fmt.Fprintf(os.Stdout, "Response from `CredentialsAPI.PatchProjectRepoCredentials`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**repoCredentials** | **string** | Repo credentials name | 

### Other Parameters

Other parameters are passed through a pointer to a apiPatchProjectRepoCredentialsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **body** | [**PatchRepoCredentialsRequest**](PatchRepoCredentialsRequest.md) | Credentials | 

### Return type

[**V1Secret**](V1Secret.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PatchSharedGenericCredentials

> V1Secret PatchSharedGenericCredentials(ctx, genericCredentials).Body(body).Execute()

Patch shared generic credentials



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
	genericCredentials := "genericCredentials_example" // string | Generic credentials name
	body := *openapiclient.NewPatchGenericCredentialsRequest() // PatchGenericCredentialsRequest | GenericCredentials patch

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CredentialsAPI.PatchSharedGenericCredentials(context.Background(), genericCredentials).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CredentialsAPI.PatchSharedGenericCredentials``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PatchSharedGenericCredentials`: V1Secret
	fmt.Fprintf(os.Stdout, "Response from `CredentialsAPI.PatchSharedGenericCredentials`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**genericCredentials** | **string** | Generic credentials name | 

### Other Parameters

Other parameters are passed through a pointer to a apiPatchSharedGenericCredentialsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**PatchGenericCredentialsRequest**](PatchGenericCredentialsRequest.md) | GenericCredentials patch | 

### Return type

[**V1Secret**](V1Secret.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PatchSharedRepoCredentials

> V1Secret PatchSharedRepoCredentials(ctx, repoCredentials).Body(body).Execute()

Patch shared repository credentials



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
	repoCredentials := "repoCredentials_example" // string | Repo credentials name
	body := *openapiclient.NewPatchRepoCredentialsRequest() // PatchRepoCredentialsRequest | Credentials

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CredentialsAPI.PatchSharedRepoCredentials(context.Background(), repoCredentials).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CredentialsAPI.PatchSharedRepoCredentials``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PatchSharedRepoCredentials`: V1Secret
	fmt.Fprintf(os.Stdout, "Response from `CredentialsAPI.PatchSharedRepoCredentials`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**repoCredentials** | **string** | Repo credentials name | 

### Other Parameters

Other parameters are passed through a pointer to a apiPatchSharedRepoCredentialsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**PatchRepoCredentialsRequest**](PatchRepoCredentialsRequest.md) | Credentials | 

### Return type

[**V1Secret**](V1Secret.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PatchSystemGenericCredentials

> V1Secret PatchSystemGenericCredentials(ctx, genericCredentials).Body(body).Execute()

Patch system-level generic credentials



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
	genericCredentials := "genericCredentials_example" // string | Generic credentials name
	body := *openapiclient.NewPatchGenericCredentialsRequest() // PatchGenericCredentialsRequest | GenericCredentials patch

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CredentialsAPI.PatchSystemGenericCredentials(context.Background(), genericCredentials).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CredentialsAPI.PatchSystemGenericCredentials``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PatchSystemGenericCredentials`: V1Secret
	fmt.Fprintf(os.Stdout, "Response from `CredentialsAPI.PatchSystemGenericCredentials`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**genericCredentials** | **string** | Generic credentials name | 

### Other Parameters

Other parameters are passed through a pointer to a apiPatchSystemGenericCredentialsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**PatchGenericCredentialsRequest**](PatchGenericCredentialsRequest.md) | GenericCredentials patch | 

### Return type

[**V1Secret**](V1Secret.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## UpdateProjectGenericCredentials

> V1Secret UpdateProjectGenericCredentials(ctx, project, genericCredentials).Body(body).Execute()

Replace project-level generic credentials



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
	project := "project_example" // string | Project name
	genericCredentials := "genericCredentials_example" // string | Generic credentials name
	body := *openapiclient.NewUpdateGenericCredentialsRequest() // UpdateGenericCredentialsRequest | GenericCredentials

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CredentialsAPI.UpdateProjectGenericCredentials(context.Background(), project, genericCredentials).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CredentialsAPI.UpdateProjectGenericCredentials``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `UpdateProjectGenericCredentials`: V1Secret
	fmt.Fprintf(os.Stdout, "Response from `CredentialsAPI.UpdateProjectGenericCredentials`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**genericCredentials** | **string** | Generic credentials name | 

### Other Parameters

Other parameters are passed through a pointer to a apiUpdateProjectGenericCredentialsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **body** | [**UpdateGenericCredentialsRequest**](UpdateGenericCredentialsRequest.md) | GenericCredentials | 

### Return type

[**V1Secret**](V1Secret.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## UpdateProjectRepoCredentials

> V1Secret UpdateProjectRepoCredentials(ctx, project, repoCredentials).Body(body).Execute()

Replace project-level repository credentials



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
	project := "project_example" // string | Project name
	repoCredentials := "repoCredentials_example" // string | Repo credentials name
	body := *openapiclient.NewUpdateRepoCredentialsRequest() // UpdateRepoCredentialsRequest | Credentials

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CredentialsAPI.UpdateProjectRepoCredentials(context.Background(), project, repoCredentials).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CredentialsAPI.UpdateProjectRepoCredentials``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `UpdateProjectRepoCredentials`: V1Secret
	fmt.Fprintf(os.Stdout, "Response from `CredentialsAPI.UpdateProjectRepoCredentials`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**repoCredentials** | **string** | Repo credentials name | 

### Other Parameters

Other parameters are passed through a pointer to a apiUpdateProjectRepoCredentialsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **body** | [**UpdateRepoCredentialsRequest**](UpdateRepoCredentialsRequest.md) | Credentials | 

### Return type

[**V1Secret**](V1Secret.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## UpdateSharedGenericCredentials

> V1Secret UpdateSharedGenericCredentials(ctx, genericCredentials).Body(body).Execute()

Replace shared generic credentials



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
	genericCredentials := "genericCredentials_example" // string | Generic credentials name
	body := *openapiclient.NewUpdateGenericCredentialsRequest() // UpdateGenericCredentialsRequest | GenericCredentials

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CredentialsAPI.UpdateSharedGenericCredentials(context.Background(), genericCredentials).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CredentialsAPI.UpdateSharedGenericCredentials``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `UpdateSharedGenericCredentials`: V1Secret
	fmt.Fprintf(os.Stdout, "Response from `CredentialsAPI.UpdateSharedGenericCredentials`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**genericCredentials** | **string** | Generic credentials name | 

### Other Parameters

Other parameters are passed through a pointer to a apiUpdateSharedGenericCredentialsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**UpdateGenericCredentialsRequest**](UpdateGenericCredentialsRequest.md) | GenericCredentials | 

### Return type

[**V1Secret**](V1Secret.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## UpdateSharedRepoCredentials

> V1Secret UpdateSharedRepoCredentials(ctx, repoCredentials).Body(body).Execute()

Replace shared repository credentials



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
	repoCredentials := "repoCredentials_example" // string | Repo credentials name
	body := *openapiclient.NewUpdateRepoCredentialsRequest() // UpdateRepoCredentialsRequest | Credentials

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CredentialsAPI.UpdateSharedRepoCredentials(context.Background(), repoCredentials).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CredentialsAPI.UpdateSharedRepoCredentials``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `UpdateSharedRepoCredentials`: V1Secret
	fmt.Fprintf(os.Stdout, "Response from `CredentialsAPI.UpdateSharedRepoCredentials`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**repoCredentials** | **string** | Repo credentials name | 

### Other Parameters

Other parameters are passed through a pointer to a apiUpdateSharedRepoCredentialsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**UpdateRepoCredentialsRequest**](UpdateRepoCredentialsRequest.md) | Credentials | 

### Return type

[**V1Secret**](V1Secret.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## UpdateSystemGenericCredentials

> V1Secret UpdateSystemGenericCredentials(ctx, genericCredentials).Body(body).Execute()

Replace system-level generic credentials



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
	genericCredentials := "genericCredentials_example" // string | Generic credentials name
	body := *openapiclient.NewUpdateGenericCredentialsRequest() // UpdateGenericCredentialsRequest | GenericCredentials

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CredentialsAPI.UpdateSystemGenericCredentials(context.Background(), genericCredentials).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CredentialsAPI.UpdateSystemGenericCredentials``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `UpdateSystemGenericCredentials`: V1Secret
	fmt.Fprintf(os.Stdout, "Response from `CredentialsAPI.UpdateSystemGenericCredentials`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**genericCredentials** | **string** | Generic credentials name | 

### Other Parameters

Other parameters are passed through a pointer to a apiUpdateSystemGenericCredentialsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**UpdateGenericCredentialsRequest**](UpdateGenericCredentialsRequest.md) | GenericCredentials | 

### Return type

[**V1Secret**](V1Secret.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

