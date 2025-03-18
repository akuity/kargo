package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"connectrpc.com/connect"
	"github.com/hashicorp/go-cleanhttp"

	v1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	"github.com/akuity/kargo/api/service/v1alpha1/svcv1alpha1connect"
)

const (
	// Set this to the address of the local Kargo API server.
	url = ""
	// Set the value of this token to a valid one for interacting with the above API server.
	token       = ""
	namespace   = ""
	analysisRun = ""
)

func main() {
	ctx := context.Background()
	httpClient := cleanhttp.DefaultClient()
	client := svcv1alpha1connect.NewKargoServiceClient(
		httpClient,
		url,
		connect.WithClientOptions(
			connect.WithInterceptors(
				&authInterceptor{
					credential: token,
				},
			),
		),
	)
	stream, err := client.GetAnalysisRunLogs(
		ctx,
		connect.NewRequest(
			&v1alpha1.GetAnalysisRunLogsRequest{
				Namespace: namespace,
				Name:      analysisRun,
			},
		),
	)
	if err != nil {
		log.Fatal(err)
	}
	for stream.Receive() {
		fmt.Print(stream.Msg().Line)
		<-time.After(100 * time.Millisecond)
	}
	if err = stream.Err(); err != nil {
		log.Fatal(err)
	}
}
