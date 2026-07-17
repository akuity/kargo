# \RbacAPI

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CreateProjectAPIToken**](RbacAPI.md#CreateProjectAPIToken) | **Post** /v1beta1/projects/{project}/roles/{role}/api-tokens | Create a project-level API token
[**CreateProjectRole**](RbacAPI.md#CreateProjectRole) | **Post** /v1beta1/projects/{project}/roles | Create a project-level Kargo Role virtual resource
[**CreateSystemAPIToken**](RbacAPI.md#CreateSystemAPIToken) | **Post** /v1beta1/system/roles/{role}/api-tokens | Create a system-level API token
[**DeleteProjectAPIToken**](RbacAPI.md#DeleteProjectAPIToken) | **Delete** /v1beta1/projects/{project}/api-tokens/{apitoken} | Delete a project-level API token
[**DeleteProjectRole**](RbacAPI.md#DeleteProjectRole) | **Delete** /v1beta1/projects/{project}/roles/{role} | Delete a project-level Kargo Role virtual resource
[**DeleteSystemAPIToken**](RbacAPI.md#DeleteSystemAPIToken) | **Delete** /v1beta1/system/api-tokens/{apitoken} | Delete a system-level API token
[**GetProjectAPIToken**](RbacAPI.md#GetProjectAPIToken) | **Get** /v1beta1/projects/{project}/api-tokens/{apitoken} | Retrieve a project-level API token
[**GetProjectRole**](RbacAPI.md#GetProjectRole) | **Get** /v1beta1/projects/{project}/roles/{role} | Retrieve a project-level Kargo Role virtual resource
[**GetSystemAPIToken**](RbacAPI.md#GetSystemAPIToken) | **Get** /v1beta1/system/api-tokens/{apitoken} | Retrieve a system-level API token
[**GetSystemRole**](RbacAPI.md#GetSystemRole) | **Get** /v1beta1/system/roles/{role} | Retrieve a system-level Kargo Role virtual resource
[**Grant**](RbacAPI.md#Grant) | **Post** /v1beta1/projects/{project}/roles/grants | Grant permissions
[**ListProjectAPITokens**](RbacAPI.md#ListProjectAPITokens) | **Get** /v1beta1/projects/{project}/api-tokens | List project-level API tokens
[**ListProjectRoles**](RbacAPI.md#ListProjectRoles) | **Get** /v1beta1/projects/{project}/roles | List project-level Kargo Role virtual resources
[**ListSystemAPITokens**](RbacAPI.md#ListSystemAPITokens) | **Get** /v1beta1/system/api-tokens | List system-level API tokens
[**ListSystemRoles**](RbacAPI.md#ListSystemRoles) | **Get** /v1beta1/system/roles | List system-level Kargo Role virtual resources
[**Revoke**](RbacAPI.md#Revoke) | **Post** /v1beta1/projects/{project}/roles/revocations | Revoke permissions
[**UpdateRole**](RbacAPI.md#UpdateRole) | **Put** /v1beta1/projects/{project}/roles/{role} | Update a project-level Kargo Role virtual resource



## CreateProjectAPIToken

> V1Secret CreateProjectAPIToken(ctx, project, role).Body(body).Execute()

Create a project-level API token



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
	role := "role_example" // string | Role name
	body := *openapiclient.NewCreateAPITokenRequest() // CreateAPITokenRequest | Token

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RbacAPI.CreateProjectAPIToken(context.Background(), project, role).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RbacAPI.CreateProjectAPIToken``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `CreateProjectAPIToken`: V1Secret
	fmt.Fprintf(os.Stdout, "Response from `RbacAPI.CreateProjectAPIToken`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**role** | **string** | Role name | 

### Other Parameters

Other parameters are passed through a pointer to a apiCreateProjectAPITokenRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **body** | [**CreateAPITokenRequest**](CreateAPITokenRequest.md) | Token | 

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


## CreateProjectRole

> RbacRole CreateProjectRole(ctx, project).Body(body).Execute()

Create a project-level Kargo Role virtual resource



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
	body := interface{}{ ... } // interface{} | Role resource (github.com/akuity/kargo/api/rbac/v1alpha1.Role)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RbacAPI.CreateProjectRole(context.Background(), project).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RbacAPI.CreateProjectRole``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `CreateProjectRole`: RbacRole
	fmt.Fprintf(os.Stdout, "Response from `RbacAPI.CreateProjectRole`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 

### Other Parameters

Other parameters are passed through a pointer to a apiCreateProjectRoleRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | **interface{}** | Role resource (github.com/akuity/kargo/api/rbac/v1alpha1.Role) | 

### Return type

[**RbacRole**](RbacRole.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## CreateSystemAPIToken

> V1Secret CreateSystemAPIToken(ctx, role).Body(body).Execute()

Create a system-level API token



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
	role := "role_example" // string | Role name
	body := *openapiclient.NewCreateAPITokenRequest() // CreateAPITokenRequest | Token

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RbacAPI.CreateSystemAPIToken(context.Background(), role).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RbacAPI.CreateSystemAPIToken``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `CreateSystemAPIToken`: V1Secret
	fmt.Fprintf(os.Stdout, "Response from `RbacAPI.CreateSystemAPIToken`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**role** | **string** | Role name | 

### Other Parameters

Other parameters are passed through a pointer to a apiCreateSystemAPITokenRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**CreateAPITokenRequest**](CreateAPITokenRequest.md) | Token | 

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


## DeleteProjectAPIToken

> DeleteProjectAPIToken(ctx, project, apitoken).Execute()

Delete a project-level API token



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
	apitoken := "apitoken_example" // string | API token name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.RbacAPI.DeleteProjectAPIToken(context.Background(), project, apitoken).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RbacAPI.DeleteProjectAPIToken``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**apitoken** | **string** | API token name | 

### Other Parameters

Other parameters are passed through a pointer to a apiDeleteProjectAPITokenRequest struct via the builder pattern


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


## DeleteProjectRole

> DeleteProjectRole(ctx, project, role).Execute()

Delete a project-level Kargo Role virtual resource



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
	role := "role_example" // string | Role name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.RbacAPI.DeleteProjectRole(context.Background(), project, role).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RbacAPI.DeleteProjectRole``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**role** | **string** | Role name | 

### Other Parameters

Other parameters are passed through a pointer to a apiDeleteProjectRoleRequest struct via the builder pattern


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


## DeleteSystemAPIToken

> DeleteSystemAPIToken(ctx, apitoken).Execute()

Delete a system-level API token



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
	apitoken := "apitoken_example" // string | API token name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.RbacAPI.DeleteSystemAPIToken(context.Background(), apitoken).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RbacAPI.DeleteSystemAPIToken``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**apitoken** | **string** | API token name | 

### Other Parameters

Other parameters are passed through a pointer to a apiDeleteSystemAPITokenRequest struct via the builder pattern


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


## GetProjectAPIToken

> V1Secret GetProjectAPIToken(ctx, project, apitoken).Execute()

Retrieve a project-level API token



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
	apitoken := "apitoken_example" // string | API token name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RbacAPI.GetProjectAPIToken(context.Background(), project, apitoken).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RbacAPI.GetProjectAPIToken``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetProjectAPIToken`: V1Secret
	fmt.Fprintf(os.Stdout, "Response from `RbacAPI.GetProjectAPIToken`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**apitoken** | **string** | API token name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetProjectAPITokenRequest struct via the builder pattern


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


## GetProjectRole

> interface{} GetProjectRole(ctx, project, role).Execute()

Retrieve a project-level Kargo Role virtual resource



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
	role := "role_example" // string | Role name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RbacAPI.GetProjectRole(context.Background(), project, role).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RbacAPI.GetProjectRole``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetProjectRole`: interface{}
	fmt.Fprintf(os.Stdout, "Response from `RbacAPI.GetProjectRole`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**role** | **string** | Role name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetProjectRoleRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



### Return type

**interface{}**

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetSystemAPIToken

> V1Secret GetSystemAPIToken(ctx, apitoken).Execute()

Retrieve a system-level API token



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
	apitoken := "apitoken_example" // string | API token name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RbacAPI.GetSystemAPIToken(context.Background(), apitoken).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RbacAPI.GetSystemAPIToken``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetSystemAPIToken`: V1Secret
	fmt.Fprintf(os.Stdout, "Response from `RbacAPI.GetSystemAPIToken`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**apitoken** | **string** | API token name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetSystemAPITokenRequest struct via the builder pattern


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


## GetSystemRole

> interface{} GetSystemRole(ctx, role).Execute()

Retrieve a system-level Kargo Role virtual resource



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
	role := "role_example" // string | Role name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RbacAPI.GetSystemRole(context.Background(), role).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RbacAPI.GetSystemRole``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetSystemRole`: interface{}
	fmt.Fprintf(os.Stdout, "Response from `RbacAPI.GetSystemRole`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**role** | **string** | Role name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetSystemRoleRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

**interface{}**

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## Grant

> RbacRole Grant(ctx, project).Body(body).Execute()

Grant permissions



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
	body := *openapiclient.NewGrantRequest() // GrantRequest | Grant request

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RbacAPI.Grant(context.Background(), project).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RbacAPI.Grant``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `Grant`: RbacRole
	fmt.Fprintf(os.Stdout, "Response from `RbacAPI.Grant`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGrantRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**GrantRequest**](GrantRequest.md) | Grant request | 

### Return type

[**RbacRole**](RbacRole.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListProjectAPITokens

> V1SecretList ListProjectAPITokens(ctx, project).Role(role).Execute()

List project-level API tokens



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
	role := "role_example" // string | Role name filter (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RbacAPI.ListProjectAPITokens(context.Background(), project).Role(role).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RbacAPI.ListProjectAPITokens``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListProjectAPITokens`: V1SecretList
	fmt.Fprintf(os.Stdout, "Response from `RbacAPI.ListProjectAPITokens`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 

### Other Parameters

Other parameters are passed through a pointer to a apiListProjectAPITokensRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **role** | **string** | Role name filter | 

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


## ListProjectRoles

> interface{} ListProjectRoles(ctx, project).Execute()

List project-level Kargo Role virtual resources



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
	resp, r, err := apiClient.RbacAPI.ListProjectRoles(context.Background(), project).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RbacAPI.ListProjectRoles``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListProjectRoles`: interface{}
	fmt.Fprintf(os.Stdout, "Response from `RbacAPI.ListProjectRoles`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 

### Other Parameters

Other parameters are passed through a pointer to a apiListProjectRolesRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

**interface{}**

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListSystemAPITokens

> V1SecretList ListSystemAPITokens(ctx).Role(role).Execute()

List system-level API tokens



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
	role := "role_example" // string | Role name filter (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RbacAPI.ListSystemAPITokens(context.Background()).Role(role).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RbacAPI.ListSystemAPITokens``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListSystemAPITokens`: V1SecretList
	fmt.Fprintf(os.Stdout, "Response from `RbacAPI.ListSystemAPITokens`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiListSystemAPITokensRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **role** | **string** | Role name filter | 

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


## ListSystemRoles

> interface{} ListSystemRoles(ctx).Execute()

List system-level Kargo Role virtual resources



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
	resp, r, err := apiClient.RbacAPI.ListSystemRoles(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RbacAPI.ListSystemRoles``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListSystemRoles`: interface{}
	fmt.Fprintf(os.Stdout, "Response from `RbacAPI.ListSystemRoles`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiListSystemRolesRequest struct via the builder pattern


### Return type

**interface{}**

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## Revoke

> RbacRole Revoke(ctx, project).Body(body).Execute()

Revoke permissions



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
	body := *openapiclient.NewRevokeRequest() // RevokeRequest | Revoke request

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RbacAPI.Revoke(context.Background(), project).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RbacAPI.Revoke``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `Revoke`: RbacRole
	fmt.Fprintf(os.Stdout, "Response from `RbacAPI.Revoke`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 

### Other Parameters

Other parameters are passed through a pointer to a apiRevokeRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**RevokeRequest**](RevokeRequest.md) | Revoke request | 

### Return type

[**RbacRole**](RbacRole.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## UpdateRole

> RbacRole UpdateRole(ctx, project, role).Body(body).Execute()

Update a project-level Kargo Role virtual resource



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
	role := "role_example" // string | Role name
	body := interface{}{ ... } // interface{} | Role resource (github.com/akuity/kargo/api/rbac/v1alpha1.Role)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RbacAPI.UpdateRole(context.Background(), project, role).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RbacAPI.UpdateRole``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `UpdateRole`: RbacRole
	fmt.Fprintf(os.Stdout, "Response from `RbacAPI.UpdateRole`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**role** | **string** | Role name | 

### Other Parameters

Other parameters are passed through a pointer to a apiUpdateRoleRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **body** | **interface{}** | Role resource (github.com/akuity/kargo/api/rbac/v1alpha1.Role) | 

### Return type

[**RbacRole**](RbacRole.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

