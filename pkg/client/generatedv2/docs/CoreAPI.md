# \CoreAPI

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**AbortPromotion**](CoreAPI.md#AbortPromotion) | **Post** /v1beta1/projects/{project}/promotions/{promotion}/abort | Abort a Promotion
[**ApproveFreight**](CoreAPI.md#ApproveFreight) | **Post** /v1beta1/projects/{project}/freight/{freight-name-or-alias}/approve | Approve Freight for promotion to a Stage
[**CreateProjectConfigMap**](CoreAPI.md#CreateProjectConfigMap) | **Post** /v1beta1/projects/{project}/configmaps | Create a project-level ConfigMap
[**CreateSharedConfigMap**](CoreAPI.md#CreateSharedConfigMap) | **Post** /v1beta1/shared/configmaps | Create a shared ConfigMap
[**CreateSystemConfigMap**](CoreAPI.md#CreateSystemConfigMap) | **Post** /v1beta1/system/configmaps | Create a system-level ConfigMap
[**DeleteFreight**](CoreAPI.md#DeleteFreight) | **Delete** /v1beta1/projects/{project}/freight/{freight-name-or-alias} | Delete a Freight resource
[**DeleteProject**](CoreAPI.md#DeleteProject) | **Delete** /v1beta1/projects/{project} | Delete a Project
[**DeleteProjectConfig**](CoreAPI.md#DeleteProjectConfig) | **Delete** /v1beta1/projects/{project}/config | Delete a ProjectConfig resource
[**DeleteProjectConfigMap**](CoreAPI.md#DeleteProjectConfigMap) | **Delete** /v1beta1/projects/{project}/configmaps/{configmap} | Delete a project-level ConfigMap
[**DeleteSharedConfigMap**](CoreAPI.md#DeleteSharedConfigMap) | **Delete** /v1beta1/shared/configmaps/{configmap} | Delete a shared ConfigMap
[**DeleteStage**](CoreAPI.md#DeleteStage) | **Delete** /v1beta1/projects/{project}/stages/{stage} | Delete a Stage
[**DeleteSystemConfigMap**](CoreAPI.md#DeleteSystemConfigMap) | **Delete** /v1beta1/system/configmaps/{configmap} | Delete a system-level ConfigMap
[**DeleteWarehouse**](CoreAPI.md#DeleteWarehouse) | **Delete** /v1beta1/projects/{project}/warehouses/{warehouse} | Delete a Warehouse
[**GetClusterPromotionTask**](CoreAPI.md#GetClusterPromotionTask) | **Get** /v1beta1/shared/cluster-promotion-tasks/{cluster-promotion-task} | Retrieve a ClusterPromotionTask
[**GetFreight**](CoreAPI.md#GetFreight) | **Get** /v1beta1/projects/{project}/freight/{freight-name-or-alias} | Retrieve a Freight resource
[**GetFreightLinks**](CoreAPI.md#GetFreightLinks) | **Get** /v1beta1/projects/{project}/freight/{freight-name-or-alias}/links | Retrieve deep links for a Freight resource
[**GetProject**](CoreAPI.md#GetProject) | **Get** /v1beta1/projects/{project} | Retrieve a Project resource
[**GetProjectConfig**](CoreAPI.md#GetProjectConfig) | **Get** /v1beta1/projects/{project}/config | Retrieve ProjectConfig
[**GetProjectConfigMap**](CoreAPI.md#GetProjectConfigMap) | **Get** /v1beta1/projects/{project}/configmaps/{configmap} | Retrieve a project-level ConfigMap
[**GetPromotion**](CoreAPI.md#GetPromotion) | **Get** /v1beta1/projects/{project}/promotions/{promotion} | Retrieve a Promotion
[**GetPromotionTask**](CoreAPI.md#GetPromotionTask) | **Get** /v1beta1/projects/{project}/promotion-tasks/{promotion-task} | Retrieve a PromotionTask
[**GetSharedConfigMap**](CoreAPI.md#GetSharedConfigMap) | **Get** /v1beta1/shared/configmaps/{configmap} | Retrieve a shared ConfigMap
[**GetStage**](CoreAPI.md#GetStage) | **Get** /v1beta1/projects/{project}/stages/{stage} | Retrieve a Stage
[**GetStageLinks**](CoreAPI.md#GetStageLinks) | **Get** /v1beta1/projects/{project}/stages/{stage}/links | Retrieve deep links for a Stage resource
[**GetSystemConfigMap**](CoreAPI.md#GetSystemConfigMap) | **Get** /v1beta1/system/configmaps/{configmap} | Retrieve a system-level ConfigMap
[**GetWarehouse**](CoreAPI.md#GetWarehouse) | **Get** /v1beta1/projects/{project}/warehouses/{warehouse} | Retrieve a Warehouse
[**ListClusterPromotionTasks**](CoreAPI.md#ListClusterPromotionTasks) | **Get** /v1beta1/shared/cluster-promotion-tasks | List ClusterPromotionTasks
[**ListImages**](CoreAPI.md#ListImages) | **Get** /v1beta1/projects/{project}/images | List container images
[**ListProjectConfigMaps**](CoreAPI.md#ListProjectConfigMaps) | **Get** /v1beta1/projects/{project}/configmaps | List project-level ConfigMaps
[**ListProjects**](CoreAPI.md#ListProjects) | **Get** /v1beta1/projects | List projects
[**ListPromotionTasks**](CoreAPI.md#ListPromotionTasks) | **Get** /v1beta1/projects/{project}/promotion-tasks | List PromotionTasks
[**ListPromotions**](CoreAPI.md#ListPromotions) | **Get** /v1beta1/projects/{project}/promotions | List Promotions
[**ListSharedConfigMaps**](CoreAPI.md#ListSharedConfigMaps) | **Get** /v1beta1/shared/configmaps | List shared ConfigMaps
[**ListStages**](CoreAPI.md#ListStages) | **Get** /v1beta1/projects/{project}/stages | List Stages
[**ListSystemConfigMaps**](CoreAPI.md#ListSystemConfigMaps) | **Get** /v1beta1/system/configmaps | List system-level ConfigMaps
[**ListWarehouses**](CoreAPI.md#ListWarehouses) | **Get** /v1beta1/projects/{project}/warehouses | List Warehouses
[**PatchFreightAlias**](CoreAPI.md#PatchFreightAlias) | **Patch** /v1beta1/projects/{project}/freight/{freight-name-or-alias}/alias | Patch a Freight resource&#39;s alias
[**PatchProjectConfigMap**](CoreAPI.md#PatchProjectConfigMap) | **Patch** /v1beta1/projects/{project}/configmaps/{configmap} | Patch a project-level ConfigMap
[**PatchSharedConfigMap**](CoreAPI.md#PatchSharedConfigMap) | **Patch** /v1beta1/shared/configmaps/{configmap} | Patch a shared ConfigMap
[**PatchSystemConfigMap**](CoreAPI.md#PatchSystemConfigMap) | **Patch** /v1beta1/system/configmaps/{configmap} | Patch a system-level ConfigMap
[**PromoteDownstream**](CoreAPI.md#PromoteDownstream) | **Post** /v1beta1/projects/{project}/stages/{stage}/promotions/downstream | Promote downstream
[**PromoteToStage**](CoreAPI.md#PromoteToStage) | **Post** /v1beta1/projects/{project}/stages/{stage}/promotions | Promote to Stage
[**QueryFreightsRest**](CoreAPI.md#QueryFreightsRest) | **Get** /v1beta1/projects/{project}/freight | Query Freight
[**RefreshProjectConfig**](CoreAPI.md#RefreshProjectConfig) | **Post** /v1beta1/projects/{project}/config/refresh | Refresh ProjectConfig
[**RefreshPromotion**](CoreAPI.md#RefreshPromotion) | **Post** /v1beta1/projects/{project}/promotions/{promotion}/refresh | Refresh a Promotion
[**RefreshStage**](CoreAPI.md#RefreshStage) | **Post** /v1beta1/projects/{project}/stages/{stage}/refresh | Refresh a Stage
[**RefreshWarehouse**](CoreAPI.md#RefreshWarehouse) | **Post** /v1beta1/projects/{project}/warehouses/{warehouse}/refresh | Refresh a Warehouse
[**UpdateProjectConfigMap**](CoreAPI.md#UpdateProjectConfigMap) | **Put** /v1beta1/projects/{project}/configmaps/{configmap} | Replace a project-level ConfigMap
[**UpdateSharedConfigMap**](CoreAPI.md#UpdateSharedConfigMap) | **Put** /v1beta1/shared/configmaps/{configmap} | Replace a shared ConfigMap
[**UpdateSystemConfigMap**](CoreAPI.md#UpdateSystemConfigMap) | **Put** /v1beta1/system/configmaps/{configmap} | Replace a system-level ConfigMap



## AbortPromotion

> AbortPromotion(ctx, project, promotion).Execute()

Abort a Promotion



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
	promotion := "promotion_example" // string | Promotion name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.CoreAPI.AbortPromotion(context.Background(), project, promotion).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.AbortPromotion``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**promotion** | **string** | Promotion name | 

### Other Parameters

Other parameters are passed through a pointer to a apiAbortPromotionRequest struct via the builder pattern


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


## ApproveFreight

> ApproveFreight(ctx, project, freightNameOrAlias).Stage(stage).Execute()

Approve Freight for promotion to a Stage



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
	freightNameOrAlias := "freightNameOrAlias_example" // string | Freight name or alias
	stage := "stage_example" // string | Stage name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.CoreAPI.ApproveFreight(context.Background(), project, freightNameOrAlias).Stage(stage).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.ApproveFreight``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**freightNameOrAlias** | **string** | Freight name or alias | 

### Other Parameters

Other parameters are passed through a pointer to a apiApproveFreightRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **stage** | **string** | Stage name | 

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


## CreateProjectConfigMap

> V1ConfigMap CreateProjectConfigMap(ctx, project).Body(body).Execute()

Create a project-level ConfigMap



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
	body := *openapiclient.NewCreateConfigMapRequest() // CreateConfigMapRequest | ConfigMap

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CoreAPI.CreateProjectConfigMap(context.Background(), project).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.CreateProjectConfigMap``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `CreateProjectConfigMap`: V1ConfigMap
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.CreateProjectConfigMap`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 

### Other Parameters

Other parameters are passed through a pointer to a apiCreateProjectConfigMapRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**CreateConfigMapRequest**](CreateConfigMapRequest.md) | ConfigMap | 

### Return type

[**V1ConfigMap**](V1ConfigMap.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## CreateSharedConfigMap

> V1ConfigMap CreateSharedConfigMap(ctx).Body(body).Execute()

Create a shared ConfigMap



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
	body := *openapiclient.NewCreateConfigMapRequest() // CreateConfigMapRequest | ConfigMap

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CoreAPI.CreateSharedConfigMap(context.Background()).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.CreateSharedConfigMap``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `CreateSharedConfigMap`: V1ConfigMap
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.CreateSharedConfigMap`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiCreateSharedConfigMapRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**CreateConfigMapRequest**](CreateConfigMapRequest.md) | ConfigMap | 

### Return type

[**V1ConfigMap**](V1ConfigMap.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## CreateSystemConfigMap

> V1ConfigMap CreateSystemConfigMap(ctx).Body(body).Execute()

Create a system-level ConfigMap



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
	body := *openapiclient.NewCreateConfigMapRequest() // CreateConfigMapRequest | ConfigMap

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CoreAPI.CreateSystemConfigMap(context.Background()).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.CreateSystemConfigMap``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `CreateSystemConfigMap`: V1ConfigMap
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.CreateSystemConfigMap`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiCreateSystemConfigMapRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**CreateConfigMapRequest**](CreateConfigMapRequest.md) | ConfigMap | 

### Return type

[**V1ConfigMap**](V1ConfigMap.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DeleteFreight

> DeleteFreight(ctx, project, freightNameOrAlias).Execute()

Delete a Freight resource



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
	freightNameOrAlias := "freightNameOrAlias_example" // string | Freight name or alias

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.CoreAPI.DeleteFreight(context.Background(), project, freightNameOrAlias).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.DeleteFreight``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**freightNameOrAlias** | **string** | Freight name or alias | 

### Other Parameters

Other parameters are passed through a pointer to a apiDeleteFreightRequest struct via the builder pattern


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


## DeleteProject

> DeleteProject(ctx, project).Execute()

Delete a Project



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
	r, err := apiClient.CoreAPI.DeleteProject(context.Background(), project).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.DeleteProject``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 

### Other Parameters

Other parameters are passed through a pointer to a apiDeleteProjectRequest struct via the builder pattern


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


## DeleteProjectConfig

> DeleteProjectConfig(ctx, project).Execute()

Delete a ProjectConfig resource



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
	r, err := apiClient.CoreAPI.DeleteProjectConfig(context.Background(), project).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.DeleteProjectConfig``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 

### Other Parameters

Other parameters are passed through a pointer to a apiDeleteProjectConfigRequest struct via the builder pattern


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


## DeleteProjectConfigMap

> DeleteProjectConfigMap(ctx, project, configmap).Execute()

Delete a project-level ConfigMap



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
	configmap := "configmap_example" // string | ConfigMap name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.CoreAPI.DeleteProjectConfigMap(context.Background(), project, configmap).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.DeleteProjectConfigMap``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**configmap** | **string** | ConfigMap name | 

### Other Parameters

Other parameters are passed through a pointer to a apiDeleteProjectConfigMapRequest struct via the builder pattern


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


## DeleteSharedConfigMap

> DeleteSharedConfigMap(ctx, configmap).Execute()

Delete a shared ConfigMap



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
	configmap := "configmap_example" // string | ConfigMap name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.CoreAPI.DeleteSharedConfigMap(context.Background(), configmap).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.DeleteSharedConfigMap``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**configmap** | **string** | ConfigMap name | 

### Other Parameters

Other parameters are passed through a pointer to a apiDeleteSharedConfigMapRequest struct via the builder pattern


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


## DeleteStage

> DeleteStage(ctx, project, stage).Execute()

Delete a Stage



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
	stage := "stage_example" // string | Stage name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.CoreAPI.DeleteStage(context.Background(), project, stage).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.DeleteStage``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**stage** | **string** | Stage name | 

### Other Parameters

Other parameters are passed through a pointer to a apiDeleteStageRequest struct via the builder pattern


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


## DeleteSystemConfigMap

> DeleteSystemConfigMap(ctx, configmap).Execute()

Delete a system-level ConfigMap



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
	configmap := "configmap_example" // string | ConfigMap name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.CoreAPI.DeleteSystemConfigMap(context.Background(), configmap).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.DeleteSystemConfigMap``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**configmap** | **string** | ConfigMap name | 

### Other Parameters

Other parameters are passed through a pointer to a apiDeleteSystemConfigMapRequest struct via the builder pattern


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


## DeleteWarehouse

> DeleteWarehouse(ctx, project, warehouse).Execute()

Delete a Warehouse



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
	warehouse := "warehouse_example" // string | Warehouse name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.CoreAPI.DeleteWarehouse(context.Background(), project, warehouse).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.DeleteWarehouse``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**warehouse** | **string** | Warehouse name | 

### Other Parameters

Other parameters are passed through a pointer to a apiDeleteWarehouseRequest struct via the builder pattern


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


## GetClusterPromotionTask

> ClusterPromotionTask GetClusterPromotionTask(ctx, clusterPromotionTask).Execute()

Retrieve a ClusterPromotionTask



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
	clusterPromotionTask := "clusterPromotionTask_example" // string | ClusterPromotionTask name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CoreAPI.GetClusterPromotionTask(context.Background(), clusterPromotionTask).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.GetClusterPromotionTask``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetClusterPromotionTask`: ClusterPromotionTask
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.GetClusterPromotionTask`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**clusterPromotionTask** | **string** | ClusterPromotionTask name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetClusterPromotionTaskRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**ClusterPromotionTask**](ClusterPromotionTask.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetFreight

> Freight GetFreight(ctx, project, freightNameOrAlias).Execute()

Retrieve a Freight resource



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
	freightNameOrAlias := "freightNameOrAlias_example" // string | Freight name or alias

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CoreAPI.GetFreight(context.Background(), project, freightNameOrAlias).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.GetFreight``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetFreight`: Freight
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.GetFreight`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**freightNameOrAlias** | **string** | Freight name or alias | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetFreightRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



### Return type

[**Freight**](Freight.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetFreightLinks

> GetFreightLinksResponse GetFreightLinks(ctx, project, freightNameOrAlias).Execute()

Retrieve deep links for a Freight resource



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
	freightNameOrAlias := "freightNameOrAlias_example" // string | Freight name or alias

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CoreAPI.GetFreightLinks(context.Background(), project, freightNameOrAlias).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.GetFreightLinks``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetFreightLinks`: GetFreightLinksResponse
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.GetFreightLinks`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**freightNameOrAlias** | **string** | Freight name or alias | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetFreightLinksRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



### Return type

[**GetFreightLinksResponse**](GetFreightLinksResponse.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetProject

> Project GetProject(ctx, project).Execute()

Retrieve a Project resource



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
	resp, r, err := apiClient.CoreAPI.GetProject(context.Background(), project).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.GetProject``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetProject`: Project
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.GetProject`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetProjectRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**Project**](Project.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetProjectConfig

> ProjectConfig GetProjectConfig(ctx, project).Execute()

Retrieve ProjectConfig



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
	resp, r, err := apiClient.CoreAPI.GetProjectConfig(context.Background(), project).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.GetProjectConfig``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetProjectConfig`: ProjectConfig
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.GetProjectConfig`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetProjectConfigRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**ProjectConfig**](ProjectConfig.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetProjectConfigMap

> V1ConfigMap GetProjectConfigMap(ctx, project, configmap).Execute()

Retrieve a project-level ConfigMap



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
	configmap := "configmap_example" // string | ConfigMap name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CoreAPI.GetProjectConfigMap(context.Background(), project, configmap).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.GetProjectConfigMap``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetProjectConfigMap`: V1ConfigMap
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.GetProjectConfigMap`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**configmap** | **string** | ConfigMap name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetProjectConfigMapRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



### Return type

[**V1ConfigMap**](V1ConfigMap.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetPromotion

> Promotion GetPromotion(ctx, project, promotion).Execute()

Retrieve a Promotion



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
	promotion := "promotion_example" // string | Promotion name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CoreAPI.GetPromotion(context.Background(), project, promotion).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.GetPromotion``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetPromotion`: Promotion
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.GetPromotion`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**promotion** | **string** | Promotion name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetPromotionRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



### Return type

[**Promotion**](Promotion.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetPromotionTask

> PromotionTask GetPromotionTask(ctx, project, promotionTask).Execute()

Retrieve a PromotionTask



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
	promotionTask := "promotionTask_example" // string | PromotionTask name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CoreAPI.GetPromotionTask(context.Background(), project, promotionTask).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.GetPromotionTask``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetPromotionTask`: PromotionTask
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.GetPromotionTask`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**promotionTask** | **string** | PromotionTask name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetPromotionTaskRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



### Return type

[**PromotionTask**](PromotionTask.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetSharedConfigMap

> V1ConfigMap GetSharedConfigMap(ctx, configmap).Execute()

Retrieve a shared ConfigMap



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
	configmap := "configmap_example" // string | ConfigMap name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CoreAPI.GetSharedConfigMap(context.Background(), configmap).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.GetSharedConfigMap``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetSharedConfigMap`: V1ConfigMap
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.GetSharedConfigMap`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**configmap** | **string** | ConfigMap name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetSharedConfigMapRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**V1ConfigMap**](V1ConfigMap.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetStage

> Stage GetStage(ctx, project, stage).Execute()

Retrieve a Stage



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
	stage := "stage_example" // string | Stage name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CoreAPI.GetStage(context.Background(), project, stage).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.GetStage``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetStage`: Stage
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.GetStage`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**stage** | **string** | Stage name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetStageRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



### Return type

[**Stage**](Stage.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetStageLinks

> GetStageLinksResponse GetStageLinks(ctx, project, stage).Execute()

Retrieve deep links for a Stage resource



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
	stage := "stage_example" // string | Stage name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CoreAPI.GetStageLinks(context.Background(), project, stage).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.GetStageLinks``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetStageLinks`: GetStageLinksResponse
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.GetStageLinks`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**stage** | **string** | Stage name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetStageLinksRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



### Return type

[**GetStageLinksResponse**](GetStageLinksResponse.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetSystemConfigMap

> V1ConfigMap GetSystemConfigMap(ctx, configmap).Execute()

Retrieve a system-level ConfigMap



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
	configmap := "configmap_example" // string | ConfigMap name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CoreAPI.GetSystemConfigMap(context.Background(), configmap).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.GetSystemConfigMap``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetSystemConfigMap`: V1ConfigMap
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.GetSystemConfigMap`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**configmap** | **string** | ConfigMap name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetSystemConfigMapRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**V1ConfigMap**](V1ConfigMap.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetWarehouse

> Warehouse GetWarehouse(ctx, project, warehouse).Execute()

Retrieve a Warehouse



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
	warehouse := "warehouse_example" // string | Warehouse name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CoreAPI.GetWarehouse(context.Background(), project, warehouse).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.GetWarehouse``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetWarehouse`: Warehouse
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.GetWarehouse`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**warehouse** | **string** | Warehouse name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetWarehouseRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



### Return type

[**Warehouse**](Warehouse.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListClusterPromotionTasks

> ClusterPromotionTaskList ListClusterPromotionTasks(ctx).Execute()

List ClusterPromotionTasks



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
	resp, r, err := apiClient.CoreAPI.ListClusterPromotionTasks(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.ListClusterPromotionTasks``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListClusterPromotionTasks`: ClusterPromotionTaskList
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.ListClusterPromotionTasks`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiListClusterPromotionTasksRequest struct via the builder pattern


### Return type

[**ClusterPromotionTaskList**](ClusterPromotionTaskList.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListImages

> map[string]TagMap ListImages(ctx, project).Execute()

List container images



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
	resp, r, err := apiClient.CoreAPI.ListImages(context.Background(), project).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.ListImages``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListImages`: map[string]TagMap
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.ListImages`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 

### Other Parameters

Other parameters are passed through a pointer to a apiListImagesRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**map[string]TagMap**](TagMap.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListProjectConfigMaps

> V1ConfigMapList ListProjectConfigMaps(ctx, project).Execute()

List project-level ConfigMaps



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
	resp, r, err := apiClient.CoreAPI.ListProjectConfigMaps(context.Background(), project).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.ListProjectConfigMaps``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListProjectConfigMaps`: V1ConfigMapList
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.ListProjectConfigMaps`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 

### Other Parameters

Other parameters are passed through a pointer to a apiListProjectConfigMapsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**V1ConfigMapList**](V1ConfigMapList.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListProjects

> ListProjectsResponse ListProjects(ctx).Mine(mine).Filter(filter).Uid(uid).PageSize(pageSize).Page(page).Execute()

List projects



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
	mine := true // bool | Only return Projects whose namespaces are mapped to the user's ServiceAccounts. (optional)
	filter := "filter_example" // string | Case-insensitive substring filter applied to the Project name. (optional)
	uid := []string{"Inner_example"} // []string | Return only Projects whose UID matches one of the given values. (optional)
	pageSize := int32(56) // int32 | Maximum number of Projects to return. Defaults to all matching Projects. (optional)
	page := int32(56) // int32 | Zero-indexed page number used together with pageSize. (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CoreAPI.ListProjects(context.Background()).Mine(mine).Filter(filter).Uid(uid).PageSize(pageSize).Page(page).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.ListProjects``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListProjects`: ListProjectsResponse
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.ListProjects`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiListProjectsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **mine** | **bool** | Only return Projects whose namespaces are mapped to the user&#39;s ServiceAccounts. | 
 **filter** | **string** | Case-insensitive substring filter applied to the Project name. | 
 **uid** | **[]string** | Return only Projects whose UID matches one of the given values. | 
 **pageSize** | **int32** | Maximum number of Projects to return. Defaults to all matching Projects. | 
 **page** | **int32** | Zero-indexed page number used together with pageSize. | 

### Return type

[**ListProjectsResponse**](ListProjectsResponse.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListPromotionTasks

> PromotionTaskList ListPromotionTasks(ctx, project).Execute()

List PromotionTasks



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
	resp, r, err := apiClient.CoreAPI.ListPromotionTasks(context.Background(), project).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.ListPromotionTasks``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListPromotionTasks`: PromotionTaskList
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.ListPromotionTasks`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 

### Other Parameters

Other parameters are passed through a pointer to a apiListPromotionTasksRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**PromotionTaskList**](PromotionTaskList.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListPromotions

> PromotionList ListPromotions(ctx, project).Stage(stage).Execute()

List Promotions



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
	stage := "stage_example" // string | Stage filter (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CoreAPI.ListPromotions(context.Background(), project).Stage(stage).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.ListPromotions``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListPromotions`: PromotionList
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.ListPromotions`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 

### Other Parameters

Other parameters are passed through a pointer to a apiListPromotionsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **stage** | **string** | Stage filter | 

### Return type

[**PromotionList**](PromotionList.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListSharedConfigMaps

> V1ConfigMapList ListSharedConfigMaps(ctx).Execute()

List shared ConfigMaps



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
	resp, r, err := apiClient.CoreAPI.ListSharedConfigMaps(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.ListSharedConfigMaps``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListSharedConfigMaps`: V1ConfigMapList
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.ListSharedConfigMaps`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiListSharedConfigMapsRequest struct via the builder pattern


### Return type

[**V1ConfigMapList**](V1ConfigMapList.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListStages

> StageList ListStages(ctx, project).FreightOrigins(freightOrigins).Execute()

List Stages



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
	freightOrigins := []string{"Inner_example"} // []string | Warehouse names to filter Stages by (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CoreAPI.ListStages(context.Background(), project).FreightOrigins(freightOrigins).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.ListStages``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListStages`: StageList
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.ListStages`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 

### Other Parameters

Other parameters are passed through a pointer to a apiListStagesRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **freightOrigins** | **[]string** | Warehouse names to filter Stages by | 

### Return type

[**StageList**](StageList.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListSystemConfigMaps

> V1ConfigMapList ListSystemConfigMaps(ctx).Execute()

List system-level ConfigMaps



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
	resp, r, err := apiClient.CoreAPI.ListSystemConfigMaps(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.ListSystemConfigMaps``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListSystemConfigMaps`: V1ConfigMapList
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.ListSystemConfigMaps`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiListSystemConfigMapsRequest struct via the builder pattern


### Return type

[**V1ConfigMapList**](V1ConfigMapList.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListWarehouses

> WarehouseList ListWarehouses(ctx, project).Execute()

List Warehouses



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
	resp, r, err := apiClient.CoreAPI.ListWarehouses(context.Background(), project).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.ListWarehouses``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListWarehouses`: WarehouseList
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.ListWarehouses`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 

### Other Parameters

Other parameters are passed through a pointer to a apiListWarehousesRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**WarehouseList**](WarehouseList.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PatchFreightAlias

> PatchFreightAlias(ctx, project, freightNameOrAlias).NewAlias(newAlias).Execute()

Patch a Freight resource's alias



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
	freightNameOrAlias := "freightNameOrAlias_example" // string | Freight name or alias
	newAlias := "newAlias_example" // string | New alias

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.CoreAPI.PatchFreightAlias(context.Background(), project, freightNameOrAlias).NewAlias(newAlias).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.PatchFreightAlias``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**freightNameOrAlias** | **string** | Freight name or alias | 

### Other Parameters

Other parameters are passed through a pointer to a apiPatchFreightAliasRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **newAlias** | **string** | New alias | 

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


## PatchProjectConfigMap

> V1ConfigMap PatchProjectConfigMap(ctx, project, configmap).Body(body).Execute()

Patch a project-level ConfigMap



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
	configmap := "configmap_example" // string | ConfigMap name
	body := *openapiclient.NewPatchConfigMapRequest() // PatchConfigMapRequest | ConfigMap patch

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CoreAPI.PatchProjectConfigMap(context.Background(), project, configmap).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.PatchProjectConfigMap``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PatchProjectConfigMap`: V1ConfigMap
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.PatchProjectConfigMap`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**configmap** | **string** | ConfigMap name | 

### Other Parameters

Other parameters are passed through a pointer to a apiPatchProjectConfigMapRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **body** | [**PatchConfigMapRequest**](PatchConfigMapRequest.md) | ConfigMap patch | 

### Return type

[**V1ConfigMap**](V1ConfigMap.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PatchSharedConfigMap

> V1ConfigMap PatchSharedConfigMap(ctx, configmap).Body(body).Execute()

Patch a shared ConfigMap



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
	configmap := "configmap_example" // string | ConfigMap name
	body := *openapiclient.NewPatchConfigMapRequest() // PatchConfigMapRequest | ConfigMap patch

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CoreAPI.PatchSharedConfigMap(context.Background(), configmap).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.PatchSharedConfigMap``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PatchSharedConfigMap`: V1ConfigMap
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.PatchSharedConfigMap`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**configmap** | **string** | ConfigMap name | 

### Other Parameters

Other parameters are passed through a pointer to a apiPatchSharedConfigMapRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**PatchConfigMapRequest**](PatchConfigMapRequest.md) | ConfigMap patch | 

### Return type

[**V1ConfigMap**](V1ConfigMap.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PatchSystemConfigMap

> V1ConfigMap PatchSystemConfigMap(ctx, configmap).Body(body).Execute()

Patch a system-level ConfigMap



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
	configmap := "configmap_example" // string | ConfigMap name
	body := *openapiclient.NewPatchConfigMapRequest() // PatchConfigMapRequest | ConfigMap patch

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CoreAPI.PatchSystemConfigMap(context.Background(), configmap).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.PatchSystemConfigMap``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PatchSystemConfigMap`: V1ConfigMap
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.PatchSystemConfigMap`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**configmap** | **string** | ConfigMap name | 

### Other Parameters

Other parameters are passed through a pointer to a apiPatchSystemConfigMapRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**PatchConfigMapRequest**](PatchConfigMapRequest.md) | ConfigMap patch | 

### Return type

[**V1ConfigMap**](V1ConfigMap.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PromoteDownstream

> interface{} PromoteDownstream(ctx, project, stage).Body(body).Execute()

Promote downstream



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
	stage := "stage_example" // string | Stage name
	body := *openapiclient.NewPromoteDownstreamRequest() // PromoteDownstreamRequest | Promote request

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CoreAPI.PromoteDownstream(context.Background(), project, stage).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.PromoteDownstream``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PromoteDownstream`: interface{}
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.PromoteDownstream`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**stage** | **string** | Stage name | 

### Other Parameters

Other parameters are passed through a pointer to a apiPromoteDownstreamRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **body** | [**PromoteDownstreamRequest**](PromoteDownstreamRequest.md) | Promote request | 

### Return type

**interface{}**

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PromoteToStage

> Promotion PromoteToStage(ctx, project, stage).Body(body).Execute()

Promote to Stage



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
	stage := "stage_example" // string | Stage name
	body := *openapiclient.NewPromoteToStageRequest() // PromoteToStageRequest | Promote request

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CoreAPI.PromoteToStage(context.Background(), project, stage).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.PromoteToStage``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PromoteToStage`: Promotion
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.PromoteToStage`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**stage** | **string** | Stage name | 

### Other Parameters

Other parameters are passed through a pointer to a apiPromoteToStageRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **body** | [**PromoteToStageRequest**](PromoteToStageRequest.md) | Promote request | 

### Return type

[**Promotion**](Promotion.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## QueryFreightsRest

> PkgServerQueryFreightsResponse QueryFreightsRest(ctx, project).Stage(stage).Origins(origins).Group(group).GroupBy(groupBy).OrderBy(orderBy).Reverse(reverse).Execute()

Query Freight



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
	stage := "stage_example" // string | Stage name to get available freight for (optional)
	origins := []string{"Inner_example"} // []string | Warehouse names to get freight from (optional)
	group := "group_example" // string | Group filter (optional)
	groupBy := "groupBy_example" // string | Group by (image_repo, git_repo, chart_repo) (optional)
	orderBy := "orderBy_example" // string | Order by (first_seen, tag) (optional)
	reverse := true // bool | Reverse order (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CoreAPI.QueryFreightsRest(context.Background(), project).Stage(stage).Origins(origins).Group(group).GroupBy(groupBy).OrderBy(orderBy).Reverse(reverse).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.QueryFreightsRest``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `QueryFreightsRest`: PkgServerQueryFreightsResponse
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.QueryFreightsRest`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 

### Other Parameters

Other parameters are passed through a pointer to a apiQueryFreightsRestRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **stage** | **string** | Stage name to get available freight for | 
 **origins** | **[]string** | Warehouse names to get freight from | 
 **group** | **string** | Group filter | 
 **groupBy** | **string** | Group by (image_repo, git_repo, chart_repo) | 
 **orderBy** | **string** | Order by (first_seen, tag) | 
 **reverse** | **bool** | Reverse order | 

### Return type

[**PkgServerQueryFreightsResponse**](PkgServerQueryFreightsResponse.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## RefreshProjectConfig

> RefreshProjectConfig(ctx, project).Execute()

Refresh ProjectConfig



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
	r, err := apiClient.CoreAPI.RefreshProjectConfig(context.Background(), project).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.RefreshProjectConfig``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 

### Other Parameters

Other parameters are passed through a pointer to a apiRefreshProjectConfigRequest struct via the builder pattern


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


## RefreshPromotion

> RefreshPromotion(ctx, project, promotion).Execute()

Refresh a Promotion



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
	promotion := "promotion_example" // string | Promotion name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.CoreAPI.RefreshPromotion(context.Background(), project, promotion).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.RefreshPromotion``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**promotion** | **string** | Promotion name | 

### Other Parameters

Other parameters are passed through a pointer to a apiRefreshPromotionRequest struct via the builder pattern


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


## RefreshStage

> RefreshStage(ctx, project, stage).Execute()

Refresh a Stage



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
	stage := "stage_example" // string | Stage name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.CoreAPI.RefreshStage(context.Background(), project, stage).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.RefreshStage``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**stage** | **string** | Stage name | 

### Other Parameters

Other parameters are passed through a pointer to a apiRefreshStageRequest struct via the builder pattern


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


## RefreshWarehouse

> RefreshWarehouse(ctx, project, warehouse).Execute()

Refresh a Warehouse



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
	warehouse := "warehouse_example" // string | Warehouse name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.CoreAPI.RefreshWarehouse(context.Background(), project, warehouse).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.RefreshWarehouse``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**warehouse** | **string** | Warehouse name | 

### Other Parameters

Other parameters are passed through a pointer to a apiRefreshWarehouseRequest struct via the builder pattern


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


## UpdateProjectConfigMap

> V1ConfigMap UpdateProjectConfigMap(ctx, project, configmap).Body(body).Execute()

Replace a project-level ConfigMap



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
	configmap := "configmap_example" // string | ConfigMap name
	body := *openapiclient.NewUpdateConfigMapRequest() // UpdateConfigMapRequest | ConfigMap

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CoreAPI.UpdateProjectConfigMap(context.Background(), project, configmap).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.UpdateProjectConfigMap``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `UpdateProjectConfigMap`: V1ConfigMap
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.UpdateProjectConfigMap`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**configmap** | **string** | ConfigMap name | 

### Other Parameters

Other parameters are passed through a pointer to a apiUpdateProjectConfigMapRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **body** | [**UpdateConfigMapRequest**](UpdateConfigMapRequest.md) | ConfigMap | 

### Return type

[**V1ConfigMap**](V1ConfigMap.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## UpdateSharedConfigMap

> V1ConfigMap UpdateSharedConfigMap(ctx, configmap).Body(body).Execute()

Replace a shared ConfigMap



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
	configmap := "configmap_example" // string | ConfigMap name
	body := *openapiclient.NewUpdateConfigMapRequest() // UpdateConfigMapRequest | ConfigMap

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CoreAPI.UpdateSharedConfigMap(context.Background(), configmap).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.UpdateSharedConfigMap``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `UpdateSharedConfigMap`: V1ConfigMap
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.UpdateSharedConfigMap`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**configmap** | **string** | ConfigMap name | 

### Other Parameters

Other parameters are passed through a pointer to a apiUpdateSharedConfigMapRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**UpdateConfigMapRequest**](UpdateConfigMapRequest.md) | ConfigMap | 

### Return type

[**V1ConfigMap**](V1ConfigMap.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## UpdateSystemConfigMap

> V1ConfigMap UpdateSystemConfigMap(ctx, configmap).Body(body).Execute()

Replace a system-level ConfigMap



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
	configmap := "configmap_example" // string | ConfigMap name
	body := *openapiclient.NewUpdateConfigMapRequest() // UpdateConfigMapRequest | ConfigMap

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CoreAPI.UpdateSystemConfigMap(context.Background(), configmap).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CoreAPI.UpdateSystemConfigMap``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `UpdateSystemConfigMap`: V1ConfigMap
	fmt.Fprintf(os.Stdout, "Response from `CoreAPI.UpdateSystemConfigMap`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**configmap** | **string** | ConfigMap name | 

### Other Parameters

Other parameters are passed through a pointer to a apiUpdateSystemConfigMapRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**UpdateConfigMapRequest**](UpdateConfigMapRequest.md) | ConfigMap | 

### Return type

[**V1ConfigMap**](V1ConfigMap.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

