# \VerificationsAPI

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**AbortVerification**](VerificationsAPI.md#AbortVerification) | **Post** /v1beta1/projects/{project}/stages/{stage}/verification/abort | Abort a running Verification process
[**DeleteAnalysisTemplate**](VerificationsAPI.md#DeleteAnalysisTemplate) | **Delete** /v1beta1/projects/{project}/analysis-templates/{analysis-template} | Delete an AnalysisTemplate
[**DeleteClusterAnalysisTemplate**](VerificationsAPI.md#DeleteClusterAnalysisTemplate) | **Delete** /v1beta1/shared/cluster-analysis-templates/{cluster-analysis-template} | Delete a ClusterAnalysisTemplate
[**GetAnalysisRun**](VerificationsAPI.md#GetAnalysisRun) | **Get** /v1beta1/projects/{project}/analysis-runs/{analysis-run} | Retrieve an AnalysisRun
[**GetAnalysisRunLogs**](VerificationsAPI.md#GetAnalysisRunLogs) | **Get** /v1beta1/projects/{project}/analysis-runs/{analysis-run}/logs | Stream AnalysisRun logs
[**GetAnalysisTemplate**](VerificationsAPI.md#GetAnalysisTemplate) | **Get** /v1beta1/projects/{project}/analysis-templates/{analysis-template} | Retrieve an AnalysisTemplate
[**GetClusterAnalysisTemplate**](VerificationsAPI.md#GetClusterAnalysisTemplate) | **Get** /v1beta1/shared/cluster-analysis-templates/{cluster-analysis-template} | Retrieve a ClusterAnalysisTemplate
[**ListAnalysisTemplates**](VerificationsAPI.md#ListAnalysisTemplates) | **Get** /v1beta1/projects/{project}/analysis-templates | List AnalysisTemplates
[**ListClusterAnalysisTemplates**](VerificationsAPI.md#ListClusterAnalysisTemplates) | **Get** /v1beta1/shared/cluster-analysis-templates | List ClusterAnalysisTemplates
[**Reverify**](VerificationsAPI.md#Reverify) | **Post** /v1beta1/projects/{project}/stages/{stage}/verification | Reverify Freight



## AbortVerification

> AbortVerification(ctx, project, stage).Execute()

Abort a running Verification process



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
	r, err := apiClient.VerificationsAPI.AbortVerification(context.Background(), project, stage).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `VerificationsAPI.AbortVerification``: %v\n", err)
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

Other parameters are passed through a pointer to a apiAbortVerificationRequest struct via the builder pattern


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


## DeleteAnalysisTemplate

> DeleteAnalysisTemplate(ctx, project, analysisTemplate).Execute()

Delete an AnalysisTemplate



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
	analysisTemplate := "analysisTemplate_example" // string | AnalysisTemplate name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.VerificationsAPI.DeleteAnalysisTemplate(context.Background(), project, analysisTemplate).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `VerificationsAPI.DeleteAnalysisTemplate``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**analysisTemplate** | **string** | AnalysisTemplate name | 

### Other Parameters

Other parameters are passed through a pointer to a apiDeleteAnalysisTemplateRequest struct via the builder pattern


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


## DeleteClusterAnalysisTemplate

> DeleteClusterAnalysisTemplate(ctx, clusterAnalysisTemplate).Execute()

Delete a ClusterAnalysisTemplate



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
	clusterAnalysisTemplate := "clusterAnalysisTemplate_example" // string | ClusterAnalysisTemplate name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.VerificationsAPI.DeleteClusterAnalysisTemplate(context.Background(), clusterAnalysisTemplate).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `VerificationsAPI.DeleteClusterAnalysisTemplate``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**clusterAnalysisTemplate** | **string** | ClusterAnalysisTemplate name | 

### Other Parameters

Other parameters are passed through a pointer to a apiDeleteClusterAnalysisTemplateRequest struct via the builder pattern


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


## GetAnalysisRun

> RolloutsAnalysisRun GetAnalysisRun(ctx, project, analysisRun).Execute()

Retrieve an AnalysisRun



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
	analysisRun := "analysisRun_example" // string | AnalysisRun name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.VerificationsAPI.GetAnalysisRun(context.Background(), project, analysisRun).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `VerificationsAPI.GetAnalysisRun``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetAnalysisRun`: RolloutsAnalysisRun
	fmt.Fprintf(os.Stdout, "Response from `VerificationsAPI.GetAnalysisRun`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**analysisRun** | **string** | AnalysisRun name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetAnalysisRunRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



### Return type

[**RolloutsAnalysisRun**](RolloutsAnalysisRun.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetAnalysisRunLogs

> string GetAnalysisRunLogs(ctx, project, analysisRun).MetricName(metricName).ContainerName(containerName).Execute()

Stream AnalysisRun logs



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
	analysisRun := "analysisRun_example" // string | AnalysisRun name
	metricName := "metricName_example" // string | Metric name (optional)
	containerName := "containerName_example" // string | Container name (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.VerificationsAPI.GetAnalysisRunLogs(context.Background(), project, analysisRun).MetricName(metricName).ContainerName(containerName).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `VerificationsAPI.GetAnalysisRunLogs``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetAnalysisRunLogs`: string
	fmt.Fprintf(os.Stdout, "Response from `VerificationsAPI.GetAnalysisRunLogs`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**analysisRun** | **string** | AnalysisRun name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetAnalysisRunLogsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **metricName** | **string** | Metric name | 
 **containerName** | **string** | Container name | 

### Return type

**string**

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: text/event-stream

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetAnalysisTemplate

> RolloutsAnalysisTemplate GetAnalysisTemplate(ctx, project, analysisTemplate).Execute()

Retrieve an AnalysisTemplate



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
	analysisTemplate := "analysisTemplate_example" // string | AnalysisTemplate name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.VerificationsAPI.GetAnalysisTemplate(context.Background(), project, analysisTemplate).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `VerificationsAPI.GetAnalysisTemplate``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetAnalysisTemplate`: RolloutsAnalysisTemplate
	fmt.Fprintf(os.Stdout, "Response from `VerificationsAPI.GetAnalysisTemplate`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 
**analysisTemplate** | **string** | AnalysisTemplate name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetAnalysisTemplateRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



### Return type

[**RolloutsAnalysisTemplate**](RolloutsAnalysisTemplate.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetClusterAnalysisTemplate

> RolloutsClusterAnalysisTemplate GetClusterAnalysisTemplate(ctx, clusterAnalysisTemplate).Execute()

Retrieve a ClusterAnalysisTemplate



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
	clusterAnalysisTemplate := "clusterAnalysisTemplate_example" // string | ClusterAnalysisTemplate name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.VerificationsAPI.GetClusterAnalysisTemplate(context.Background(), clusterAnalysisTemplate).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `VerificationsAPI.GetClusterAnalysisTemplate``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetClusterAnalysisTemplate`: RolloutsClusterAnalysisTemplate
	fmt.Fprintf(os.Stdout, "Response from `VerificationsAPI.GetClusterAnalysisTemplate`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**clusterAnalysisTemplate** | **string** | ClusterAnalysisTemplate name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetClusterAnalysisTemplateRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**RolloutsClusterAnalysisTemplate**](RolloutsClusterAnalysisTemplate.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListAnalysisTemplates

> RolloutsAnalysisTemplateList ListAnalysisTemplates(ctx, project).Execute()

List AnalysisTemplates



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
	resp, r, err := apiClient.VerificationsAPI.ListAnalysisTemplates(context.Background(), project).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `VerificationsAPI.ListAnalysisTemplates``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListAnalysisTemplates`: RolloutsAnalysisTemplateList
	fmt.Fprintf(os.Stdout, "Response from `VerificationsAPI.ListAnalysisTemplates`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 

### Other Parameters

Other parameters are passed through a pointer to a apiListAnalysisTemplatesRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**RolloutsAnalysisTemplateList**](RolloutsAnalysisTemplateList.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListClusterAnalysisTemplates

> RolloutsClusterAnalysisTemplateList ListClusterAnalysisTemplates(ctx).Execute()

List ClusterAnalysisTemplates



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
	resp, r, err := apiClient.VerificationsAPI.ListClusterAnalysisTemplates(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `VerificationsAPI.ListClusterAnalysisTemplates``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListClusterAnalysisTemplates`: RolloutsClusterAnalysisTemplateList
	fmt.Fprintf(os.Stdout, "Response from `VerificationsAPI.ListClusterAnalysisTemplates`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiListClusterAnalysisTemplatesRequest struct via the builder pattern


### Return type

[**RolloutsClusterAnalysisTemplateList**](RolloutsClusterAnalysisTemplateList.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## Reverify

> Reverify(ctx, project, stage).Execute()

Reverify Freight



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
	r, err := apiClient.VerificationsAPI.Reverify(context.Background(), project, stage).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `VerificationsAPI.Reverify``: %v\n", err)
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

Other parameters are passed through a pointer to a apiReverifyRequest struct via the builder pattern


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

