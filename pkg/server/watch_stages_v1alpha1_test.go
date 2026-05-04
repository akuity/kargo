package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	"github.com/akuity/kargo/api/service/v1alpha1/svcv1alpha1connect"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/validation"
)

func TestWatchStages(t *testing.T) {
	const projectName = "fake-project"

	stageDirectWH1 := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: projectName,
			Name:      "stage-direct-wh1",
		},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{{
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: "warehouse-1",
				},
				Sources: kargoapi.FreightSources{Direct: true},
			}},
		},
	}
	stageIndirectWH1 := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: projectName,
			Name:      "stage-indirect-wh1",
		},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{{
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: "warehouse-1",
				},
				Sources: kargoapi.FreightSources{Stages: []string{"stage-direct-wh1"}},
			}},
		},
	}
	stageWH2 := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: projectName,
			Name:      "stage-wh2",
		},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{{
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: "warehouse-2",
				},
				Sources: kargoapi.FreightSources{Direct: true},
			}},
		},
	}

	testCases := []struct {
		name        string
		req         *svcv1alpha1.WatchStagesRequest
		stageEvents []*kargoapi.Stage
		// expectedCount is the number of responses expected to be received.
		// The test cancels the stream context after receiving this many to avoid
		// waiting for the full timeout.
		expectedCount int
		errExpected   bool
		expectedCode  connect.Code
		assert        func(*testing.T, []*svcv1alpha1.WatchStagesResponse)
	}{
		{
			name: "empty project returns error",
			req: &svcv1alpha1.WatchStagesRequest{
				Project: "",
			},
			errExpected:  true,
			expectedCode: connect.CodeInvalidArgument,
		},
		{
			name: "non-existent project returns error",
			req: &svcv1alpha1.WatchStagesRequest{
				Project: "non-existent-project",
			},
			errExpected:  true,
			expectedCode: connect.CodeNotFound,
		},
		{
			name: "no warehouse filter passes all stage events",
			req: &svcv1alpha1.WatchStagesRequest{
				Project: projectName,
			},
			stageEvents:   []*kargoapi.Stage{stageDirectWH1, stageWH2},
			expectedCount: 2,
			assert: func(t *testing.T, responses []*svcv1alpha1.WatchStagesResponse) {
				require.Len(t, responses, 2)
				names := make([]string, len(responses))
				for i, r := range responses {
					names[i] = r.GetStage().GetName()
				}
				require.Contains(t, names, "stage-direct-wh1")
				require.Contains(t, names, "stage-wh2")
			},
		},
		{
			name: "warehouse filter passes direct and indirect subscribers, excludes others",
			req: &svcv1alpha1.WatchStagesRequest{
				Project:        projectName,
				FreightOrigins: []string{"warehouse-1"},
			},
			stageEvents:   []*kargoapi.Stage{stageDirectWH1, stageIndirectWH1, stageWH2},
			expectedCount: 2, // only warehouse-1 subscribers pass; stageWH2 is filtered
			assert: func(t *testing.T, responses []*svcv1alpha1.WatchStagesResponse) {
				require.Len(t, responses, 2)
				names := make([]string, len(responses))
				for i, r := range responses {
					names[i] = r.GetStage().GetName()
				}
				require.Contains(t, names, "stage-direct-wh1")
				require.Contains(t, names, "stage-indirect-wh1")
				require.NotContains(t, names, "stage-wh2")
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fakeClient := fake.NewClientBuilder().WithScheme(mustNewScheme()).Build()

			k8sClient, err := kubernetes.NewClient(
				t.Context(),
				&rest.Config{},
				kubernetes.ClientOptions{
					SkipAuthorization: true,
					NewInternalClient: func(
						context.Context,
						*rest.Config,
						*runtime.Scheme,
						string,
					) (client.WithWatch, error) {
						return fakeClient, nil
					},
				},
			)
			require.NoError(t, err)

			svr := &server{client: k8sClient}
			svr.externalValidateProjectFn = func(_ context.Context, _ client.Client, project string) error {
				if project != projectName {
					return validation.ErrProjectNotFound
				}
				return nil
			}

			mux := http.NewServeMux()
			mux.Handle(svcv1alpha1connect.NewKargoServiceHandler(svr))
			httpSrv := httptest.NewServer(mux)
			t.Cleanup(httpSrv.Close)

			cli := svcv1alpha1connect.NewKargoServiceClient(httpSrv.Client(), httpSrv.URL)

			ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
			defer cancel()

			// For streaming RPC: create events BEFORE calling WatchStages so the
			// goroutine is already running when the watch loop starts. The delay
			// gives the server time to register the watch before events arrive.
			// We skip this for error test cases which close the stream immediately.
			if len(tc.stageEvents) > 0 {
				go func() {
					time.Sleep(200 * time.Millisecond)
					for _, stage := range tc.stageEvents {
						s, ok := stage.DeepCopyObject().(client.Object)
						if !ok {
							return
						}
						_ = fakeClient.Create(ctx, s)
					}
				}()
			}

			// The connect streaming protocol sends HTTP response headers only after
			// the first Send(). For error cases this happens when the handler
			// closes the stream with an error; for success cases it happens when
			// the first matching event is sent. In both cases WatchStages returns
			// (stream, nil) — the actual error lives in stream.Err().
			stream, err := cli.WatchStages(ctx, connect.NewRequest(tc.req))
			require.NoError(t, err)

			if tc.errExpected {
				require.False(t, stream.Receive())
				require.Error(t, stream.Err())
				require.Equal(t, tc.expectedCode, connect.CodeOf(stream.Err()))
				return
			}

			var responses []*svcv1alpha1.WatchStagesResponse
			for stream.Receive() {
				responses = append(responses, stream.Msg())
				if len(responses) == tc.expectedCount {
					cancel()
					break
				}
			}

			tc.assert(t, responses)
		})
	}
}
