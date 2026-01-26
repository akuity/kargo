package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	k8sevent "github.com/akuity/kargo/pkg/event/kubernetes"
	fakeevent "github.com/akuity/kargo/pkg/kubernetes/event/fake"
	"github.com/akuity/kargo/pkg/server/config"
)

func TestPromoteDownstream(t *testing.T) {
	testStageSpec := kargoapi.StageSpec{
		RequestedFreight: []kargoapi.FreightRequest{{
			Origin: kargoapi.FreightOrigin{
				Kind: kargoapi.FreightOriginKindWarehouse,
				Name: "fake-warehouse",
			},
			Sources: kargoapi.FreightSources{
				Stages: []string{"fake-upstream-stage"},
			},
		}},
	}

	testCases := []struct {
		name       string
		req        *svcv1alpha1.PromoteDownstreamRequest
		server     *server
		assertions func(
			*testing.T,
			*fakeevent.EventRecorder,
			*connect.Response[svcv1alpha1.PromoteDownstreamResponse],
			error,
		)
	}{
		{
			name:   "input validation error",
			server: &server{},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				_ *connect.Response[svcv1alpha1.PromoteDownstreamResponse],
				err error,
			) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeInvalidArgument, connErr.Code())
			},
		},
		{
			name: "error validating project",
			req: &svcv1alpha1.PromoteDownstreamRequest{
				Project: "fake-project",
				Stage:   "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				_ *connect.Response[svcv1alpha1.PromoteDownstreamResponse],
				err error,
			) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error getting Stage",
			req: &svcv1alpha1.PromoteDownstreamRequest{
				Project: "fake-project",
				Stage:   "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				_ *connect.Response[svcv1alpha1.PromoteDownstreamResponse],
				err error,
			) {
				require.ErrorContains(t, err, "get stage: something went wrong")
			},
		},
		{
			name: "Stage not found",
			req: &svcv1alpha1.PromoteDownstreamRequest{
				Project: "fake-project",
				Stage:   "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return nil, nil
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				_ *connect.Response[svcv1alpha1.PromoteDownstreamResponse],
				err error,
			) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeNotFound, connErr.Code())
				require.Contains(t, connErr.Message(), "Stage")
				require.Contains(t, connErr.Message(), "not found in namespace")
			},
		},
		{
			name: "error getting Freight",
			req: &svcv1alpha1.PromoteDownstreamRequest{
				Project: "fake-project",
				Stage:   "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{
						Spec: testStageSpec,
					}, nil
				},
				getFreightByNameOrAliasFn: func(
					context.Context,
					client.Client,
					string, string, string,
				) (*kargoapi.Freight, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				_ *connect.Response[svcv1alpha1.PromoteDownstreamResponse],
				err error,
			) {
				require.ErrorContains(t, err, "get freight: something went wrong")
			},
		},
		{
			name: "Freight not found",
			req: &svcv1alpha1.PromoteDownstreamRequest{
				Project: "fake-project",
				Stage:   "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{
						Spec: testStageSpec,
					}, nil
				},
				getFreightByNameOrAliasFn: func(
					context.Context,
					client.Client,
					string, string, string,
				) (*kargoapi.Freight, error) {
					return nil, nil
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				_ *connect.Response[svcv1alpha1.PromoteDownstreamResponse],
				err error,
			) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeNotFound, connErr.Code())
				require.Contains(t, connErr.Message(), "freight")
				require.Contains(t, connErr.Message(), "not found in namespace")
			},
		},
		{
			name: "error finding downstream Stages",
			req: &svcv1alpha1.PromoteDownstreamRequest{
				Project: "fake-project",
				Stage:   "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "fake-project",
							Name:      "fake-stage",
						},
						Spec: testStageSpec,
					}, nil
				},
				getFreightByNameOrAliasFn: func(
					context.Context,
					client.Client,
					string, string, string,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						Status: kargoapi.FreightStatus{
							VerifiedIn: map[string]kargoapi.VerifiedStage{
								"fake-stage": {},
							},
						},
					}, nil
				},
				findDownstreamStagesFn: func(
					context.Context,
					*kargoapi.Stage,
					kargoapi.FreightOrigin,
				) ([]kargoapi.Stage, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				_ *connect.Response[svcv1alpha1.PromoteDownstreamResponse],
				err error,
			) {
				require.ErrorContains(t, err, "find downstream stages: something went wrong")
			},
		},
		{
			name: "no downstream Stages found",
			req: &svcv1alpha1.PromoteDownstreamRequest{
				Project: "fake-project",
				Stage:   "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "fake-project",
							Name:      "fake-stage",
						},
						Spec: testStageSpec,
					}, nil
				},
				getFreightByNameOrAliasFn: func(
					context.Context,
					client.Client,
					string, string, string,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						Status: kargoapi.FreightStatus{
							VerifiedIn: map[string]kargoapi.VerifiedStage{
								"fake-stage": {},
							},
						},
					}, nil
				},
				findDownstreamStagesFn: func(
					context.Context,
					*kargoapi.Stage,
					kargoapi.FreightOrigin,
				) ([]kargoapi.Stage, error) {
					return nil, nil
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				_ *connect.Response[svcv1alpha1.PromoteDownstreamResponse],
				err error,
			) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeNotFound, connErr.Code())
				require.Contains(t, connErr.Message(), "stage")
				require.Contains(t, connErr.Message(), "has no downstream stages")
			},
		},
		{
			name: "promoting not authorized",
			req: &svcv1alpha1.PromoteDownstreamRequest{
				Project: "fake-project",
				Stage:   "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "fake-project",
							Name:      "fake-stage",
						},
						Spec: testStageSpec,
					}, nil
				},
				getFreightByNameOrAliasFn: func(
					context.Context,
					client.Client,
					string,
					string,
					string,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						Status: kargoapi.FreightStatus{
							VerifiedIn: map[string]kargoapi.VerifiedStage{
								"fake-stage": {},
							},
						},
					}, nil
				},
				findDownstreamStagesFn: func(
					context.Context,
					*kargoapi.Stage,
					kargoapi.FreightOrigin,
				) ([]kargoapi.Stage, error) {
					return []kargoapi.Stage{{}}, nil
				},
				authorizeFn: func(
					context.Context,
					string,
					schema.GroupVersionResource,
					string,
					client.ObjectKey,
				) error {
					return errors.New("not authorized")
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				_ *connect.Response[svcv1alpha1.PromoteDownstreamResponse],
				err error,
			) {
				require.ErrorContains(t, err, "not authorized")
			},
		},
		{
			name: "Freight not available",
			req: &svcv1alpha1.PromoteDownstreamRequest{
				Project: "fake-project",
				Stage:   "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{
						Spec: testStageSpec,
					}, nil
				},
				getFreightByNameOrAliasFn: func(
					context.Context,
					client.Client,
					string,
					string,
					string,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				findDownstreamStagesFn: func(
					context.Context,
					*kargoapi.Stage,
					kargoapi.FreightOrigin,
				) ([]kargoapi.Stage, error) {
					return []kargoapi.Stage{{}}, nil
				},
				authorizeFn: func(
					context.Context,
					string,
					schema.GroupVersionResource,
					string,
					client.ObjectKey,
				) error {
					return nil
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				_ *connect.Response[svcv1alpha1.PromoteDownstreamResponse],
				err error,
			) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeInvalidArgument, connErr.Code())
				require.Contains(t, connErr.Message(), "Freight")
				require.Contains(t, connErr.Message(), "is not available to downstream Stage")
			},
		},
		{
			name: "error creating Promotion",
			req: &svcv1alpha1.PromoteDownstreamRequest{
				Project: "fake-project",
				Stage:   "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "fake-project",
							Name:      "fake-stage",
						},
						Spec: testStageSpec,
					}, nil
				},
				getFreightByNameOrAliasFn: func(
					context.Context,
					client.Client,
					string, string, string,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "fake-project",
							Name:      "fake-freight",
						},
						Status: kargoapi.FreightStatus{
							VerifiedIn: map[string]kargoapi.VerifiedStage{
								"fake-stage": {},
							},
						},
					}, nil
				},
				findDownstreamStagesFn: func(
					context.Context,
					*kargoapi.Stage,
					kargoapi.FreightOrigin,
				) ([]kargoapi.Stage, error) {
					return []kargoapi.Stage{
						{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "fake-project",
								Name:      "fake-downstream-stage",
							},
							Spec: kargoapi.StageSpec{
								RequestedFreight: []kargoapi.FreightRequest{{
									Sources: kargoapi.FreightSources{
										Stages: []string{"fake-stage"},
									},
								}},
								PromotionTemplate: &kargoapi.PromotionTemplate{
									Spec: kargoapi.PromotionTemplateSpec{
										Steps: []kargoapi.PromotionStep{{}},
									},
								},
							},
						},
					}, nil
				},
				authorizeFn: func(
					context.Context,
					string,
					schema.GroupVersionResource,
					string,
					client.ObjectKey,
				) error {
					return nil
				},
				createPromotionFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				_ *connect.Response[svcv1alpha1.PromoteDownstreamResponse],
				err error,
			) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeInternal, connErr.Code())
				require.Contains(t, connErr.Message(), "something went wrong")
			},
		},
		{
			name: "success",
			req: &svcv1alpha1.PromoteDownstreamRequest{
				Project: "fake-project",
				Stage:   "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "fake-project",
							Name:      "fake-stage",
						},
						Spec: testStageSpec,
					}, nil
				},
				getFreightByNameOrAliasFn: func(
					context.Context,
					client.Client,
					string, string, string,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "fake-project",
							Name:      "fake-freight",
						},
						Status: kargoapi.FreightStatus{
							VerifiedIn: map[string]kargoapi.VerifiedStage{
								"fake-stage": {},
							},
						},
					}, nil
				},
				findDownstreamStagesFn: func(
					context.Context,
					*kargoapi.Stage,
					kargoapi.FreightOrigin,
				) ([]kargoapi.Stage, error) {
					return []kargoapi.Stage{
						{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "fake-project",
								Name:      "fake-downstream-stage",
							},
							Spec: kargoapi.StageSpec{
								RequestedFreight: []kargoapi.FreightRequest{{
									Sources: kargoapi.FreightSources{
										Stages: []string{"fake-stage"},
									},
								}},
								PromotionTemplate: &kargoapi.PromotionTemplate{
									Spec: kargoapi.PromotionTemplateSpec{
										Steps: []kargoapi.PromotionStep{{}},
									},
								},
							},
						},
					}, nil
				},
				authorizeFn: func(
					context.Context,
					string,
					schema.GroupVersionResource,
					string,
					client.ObjectKey,
				) error {
					return nil
				},
				createPromotionFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return nil
				},
			},
			assertions: func(
				t *testing.T,
				recorder *fakeevent.EventRecorder,
				res *connect.Response[svcv1alpha1.PromoteDownstreamResponse],
				err error,
			) {
				require.NoError(t, err)
				require.NotNil(t, res)
				require.NotEmpty(t, res.Msg.GetPromotions())
				require.Len(t, recorder.Events, 1)
				event := <-recorder.Events
				require.Equal(t, corev1.EventTypeNormal, event.EventType)
				require.Equal(t, string(kargoapi.EventTypePromotionCreated), event.Reason)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := fakeevent.NewEventRecorder(1)
			testCase.server.sender = k8sevent.NewEventSender(recorder)
			resp, err := testCase.server.PromoteDownstream(
				context.Background(),
				connect.NewRequest(testCase.req),
			)
			testCase.assertions(t, recorder, resp, err)
		})
	}
}

func Test_server_promoteDownstream(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testWarehouse := &kargoapi.Warehouse{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-warehouse",
			Namespace: testProject.Name,
		},
	}
	testFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-freight",
			Namespace: testProject.Name,
			Labels: map[string]string{
				kargoapi.LabelKeyAlias: "fake-alias",
			},
		},
		Origin: kargoapi.FreightOrigin{
			Kind: kargoapi.FreightOriginKindWarehouse,
			Name: testWarehouse.Name,
		},
		Status: kargoapi.FreightStatus{
			// Freight must be verified in upstream stage to be available to downstream
			VerifiedIn: map[string]kargoapi.VerifiedStage{
				"fake-stage": {},
			},
		},
	}
	testStage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-stage",
			Namespace: testProject.Name,
		},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{
				{
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: testWarehouse.Name,
					},
					Sources: kargoapi.FreightSources{
						Direct: true,
					},
				},
			},
		},
	}
	testDownstreamStage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-downstream-stage",
			Namespace: testProject.Name,
		},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{
				{
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: testWarehouse.Name,
					},
					Sources: kargoapi.FreightSources{
						Stages: []string{testStage.Name},
					},
				},
			},
			PromotionTemplate: &kargoapi.PromotionTemplate{
				Spec: kargoapi.PromotionTemplateSpec{
					Steps: []kargoapi.PromotionStep{
						{Uses: "fake-step"},
					},
				},
			},
		},
	}

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodPost, "/v1beta1/projects/"+testProject.Name+"/stages/"+testStage.Name+"/promotions/downstream",
		[]restTestCase{
			{
				name:          "Project not found",
				clientBuilder: fake.NewClientBuilder(),
				body: mustJSONBody(promoteDownstreamRequest{
					Freight: testFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "Stage not found",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				body: mustJSONBody(promoteDownstreamRequest{
					Freight: testFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "Freight not found",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testStage),
				body: mustJSONBody(promoteDownstreamRequest{
					Freight: "nonexistent-freight",
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "No downstream stages",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testStage, testFreight),
				body: mustJSONBody(promoteDownstreamRequest{
					Freight: testFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "Successfully promote downstream",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testStage,
					testDownstreamStage,
					testFreight,
				),
				body: mustJSONBody(promoteDownstreamRequest{
					Freight: testFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)

					// Verify a Promotion was created for the downstream stage
					promos := &kargoapi.PromotionList{}
					err := c.List(t.Context(), promos, client.InNamespace(testProject.Name))
					require.NoError(t, err)
					require.Len(t, promos.Items, 1)
					require.Equal(t, testDownstreamStage.Name, promos.Items[0].Spec.Stage)
					require.Equal(t, testFreight.Name, promos.Items[0].Spec.Freight)
				},
			},
		},
	)
}
