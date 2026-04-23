package server

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/validation"
)

// fakeWatchSummariesStream captures events sent by the server so tests can
// inspect them.
type fakeWatchSummariesStream struct {
	events []*svcv1alpha1.WatchStageSummariesResponse
}

func (f *fakeWatchSummariesStream) Send(resp *svcv1alpha1.WatchStageSummariesResponse) error {
	f.events = append(f.events, resp)
	return nil
}

func TestWatchStageSummaries(t *testing.T) {
	const projectName = "kargo-demo"

	newStage := func(name, warehouseName string) *kargoapi.Stage {
		return &kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{Namespace: projectName, Name: name},
			Spec: kargoapi.StageSpec{
				RequestedFreight: []kargoapi.FreightRequest{{
					Origin:  kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: warehouseName},
					Sources: kargoapi.FreightSources{Direct: true},
				}},
			},
		}
	}

	testCases := map[string]struct {
		req       *svcv1alpha1.WatchStageSummariesRequest
		seed      []client.Object
		after     func(ctx context.Context, c client.Client)
		wantError bool
		assert    func(t *testing.T, events []*svcv1alpha1.WatchStageSummariesResponse)
	}{
		"empty project is rejected": {
			req:       &svcv1alpha1.WatchStageSummariesRequest{Project: ""},
			wantError: true,
		},
		"non-existing project is rejected": {
			req:       &svcv1alpha1.WatchStageSummariesRequest{Project: "does-not-exist"},
			wantError: true,
		},
		"emits stageSummary (not full Stage) for created objects": {
			req: &svcv1alpha1.WatchStageSummariesRequest{Project: projectName},
			after: func(ctx context.Context, c client.Client) {
				_ = c.Create(ctx, newStage("a", "wh-a"))
			},
			assert: func(t *testing.T, events []*svcv1alpha1.WatchStageSummariesResponse) {
				require.NotEmpty(t, events)
				evt := events[0]
				require.NotNil(t, evt.StageSummary)
				require.Equal(t, "a", evt.StageSummary.Metadata.Name)
				require.NotEmpty(t, evt.Type)
			},
		},
		"filter drops non-matching warehouse events": {
			req: &svcv1alpha1.WatchStageSummariesRequest{
				Project:        projectName,
				FreightOrigins: []string{"wh-a"},
			},
			after: func(ctx context.Context, c client.Client) {
				_ = c.Create(ctx, newStage("a", "wh-a"))
				_ = c.Create(ctx, newStage("b", "wh-other"))
			},
			assert: func(t *testing.T, events []*svcv1alpha1.WatchStageSummariesResponse) {
				for _, evt := range events {
					require.Equal(t, "a", evt.StageSummary.Metadata.Name)
				}
				require.NotEmpty(t, events)
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
			defer cancel()

			seed := append([]client.Object{
				&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Name:   projectName,
					Labels: map[string]string{kargoapi.LabelKeyProject: kargoapi.LabelValueTrue},
				}},
				&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: projectName}},
			}, tc.seed...)

			kubeClient, err := kubernetes.NewClient(
				ctx,
				&rest.Config{},
				kubernetes.ClientOptions{
					SkipAuthorization: true,
					NewInternalClient: func(
						context.Context, *rest.Config, *runtime.Scheme,
					) (client.WithWatch, error) {
						return fake.NewClientBuilder().
							WithScheme(mustNewScheme()).
							WithObjects(seed...).
							Build(), nil
					},
				},
			)
			require.NoError(t, err)

			svr := &server{client: kubeClient}
			svr.externalValidateProjectFn = validation.ValidateProject

			fakeStream := &fakeWatchSummariesStream{}

			errCh := make(chan error, 1)
			go func() {
				errCh <- runWatchStageSummariesForTest(ctx, svr, tc.req, fakeStream)
			}()

			// Let the watch start before triggering operations.
			time.Sleep(100 * time.Millisecond)
			if tc.after != nil {
				tc.after(ctx, svr.client.InternalClient())
			}
			time.Sleep(300 * time.Millisecond)
			cancel()
			err = <-errCh

			if tc.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tc.assert != nil {
				tc.assert(t, fakeStream.events)
			}
		})
	}
}

// runWatchStageSummariesForTest mirrors WatchStageSummaries but sends events
// to a test stream instead of a connect.ServerStream. Keeps test logic
// independent of connect's unexported stream constructors.
func runWatchStageSummariesForTest(
	ctx context.Context,
	s *server,
	req *svcv1alpha1.WatchStageSummariesRequest,
	out interface {
		Send(*svcv1alpha1.WatchStageSummariesResponse) error
	},
) error {
	if err := validateFieldNotEmpty("project", req.GetProject()); err != nil {
		return err
	}
	if err := s.validateProjectExists(ctx, req.GetProject()); err != nil {
		return err
	}
	want := warehouseNameSet(req.GetFreightOrigins())
	w, err := s.client.Watch(ctx, &kargoapi.StageList{}, client.InNamespace(req.GetProject()))
	if err != nil {
		return err
	}
	defer w.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case e, ok := <-w.ResultChan():
			if !ok {
				return nil
			}
			stage, ok := e.Object.(*kargoapi.Stage)
			if !ok {
				continue
			}
			if len(want) > 0 && !stageMatchesAnyWarehouse(stage, want) {
				continue
			}
			if err := out.Send(&svcv1alpha1.WatchStageSummariesResponse{
				StageSummary: stageToSummary(stage),
				Type:         string(e.Type),
			}); err != nil {
				return err
			}
		}
	}
}
