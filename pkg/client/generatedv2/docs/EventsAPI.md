# \EventsAPI

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**ListProjectEvents**](EventsAPI.md#ListProjectEvents) | **Get** /v1beta1/projects/{project}/events | List project-level Kubernetes Events



## ListProjectEvents

> V1EventList ListProjectEvents(ctx, project).Execute()

List project-level Kubernetes Events



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
	resp, r, err := apiClient.EventsAPI.ListProjectEvents(context.Background(), project).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `EventsAPI.ListProjectEvents``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListProjectEvents`: V1EventList
	fmt.Fprintf(os.Stdout, "Response from `EventsAPI.ListProjectEvents`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**project** | **string** | Project name | 

### Other Parameters

Other parameters are passed through a pointer to a apiListProjectEventsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**V1EventList**](V1EventList.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

