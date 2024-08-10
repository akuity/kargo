package freight

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	authnv1 "k8s.io/api/authentication/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/authentication/serviceaccount"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	fakeevent "github.com/akuity/kargo/internal/kubernetes/event/fake"
	libWebhook "github.com/akuity/kargo/internal/webhook"
)

func TestNewWebhook(t *testing.T) {
	kubeClient := fake.NewClientBuilder().Build()
	w := newWebhook(
		libWebhook.Config{},
		kubeClient,
		&fakeevent.EventRecorder{},
	)
	require.NotNil(t, w.freightAliasGenerator)
	// Assert that all overridable behaviors were initialized to a default:
	require.NotNil(t, w.admissionRequestFromContextFn)
	require.NotNil(t, w.getAvailableFreightAliasFn)
	require.NotNil(t, w.validateProjectFn)
	require.NotNil(t, w.listFreightFn)
	require.NotNil(t, w.listStagesFn)
	require.NotNil(t, w.getWarehouseFn)
	require.NotNil(t, w.validateFreightArtifactsFn)
	require.NotNil(t, w.isRequestFromKargoControlplaneFn)
}

func TestDefault(t *testing.T) {
	testCases := []struct {
		name       string
		op         admissionv1.Operation
		webhook    *webhook
		freight    *kargoapi.Freight
		assertions func(*testing.T, *kargoapi.Freight, error)
	}{
		{
			name:    "error getting request from context",
			webhook: &webhook{},
			freight: &kargoapi.Freight{},
			assertions: func(t *testing.T, _ *kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "error getting admission request from context")
			},
		},
		{
			name:    "sync alias label to non-empty alias field",
			op:      admissionv1.Create,
			webhook: &webhook{},
			freight: &kargoapi.Freight{
				Alias: "fake-alias",
			},
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, freight.Name)
				require.Equal(t, "fake-alias", freight.Alias)
				require.Equal(t, "fake-alias", freight.Labels[kargoapi.AliasLabelKey])
			},
		},
		{
			name: "error getting available alias",
			op:   admissionv1.Create,
			webhook: &webhook{
				getAvailableFreightAliasFn: func(context.Context) (string, error) {
					return "", errors.New("something went wrong")
				},
			},
			freight: &kargoapi.Freight{},
			assertions: func(t *testing.T, _ *kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "get available freight alias")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "success getting available alias",
			op:   admissionv1.Create,
			webhook: &webhook{
				getAvailableFreightAliasFn: func(context.Context) (string, error) {
					return "fake-alias", nil
				},
			},
			freight: &kargoapi.Freight{},
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, freight.Name)
				require.Equal(t, "fake-alias", freight.Alias)
				require.Equal(t, "fake-alias", freight.Labels[kargoapi.AliasLabelKey])
			},
		},
		{
			name:    "create with empty name",
			op:      admissionv1.Create,
			webhook: &webhook{},
			freight: &kargoapi.Freight{
				Alias: "fake-alias",
			},
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, freight.Name)
			},
		},
		{
			name:    "update with empty alias",
			op:      admissionv1.Update,
			webhook: &webhook{},
			freight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						kargoapi.AliasLabelKey: "fake-alias",
					},
				},
			},
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Empty(t, freight.Alias)
				_, ok := freight.Labels[kargoapi.AliasLabelKey]
				require.False(t, ok)
			},
		},
	}
	for _, testCase := range testCases {
		ctx := context.Background()
		if testCase.op != "" {
			ctx = admission.NewContextWithRequest(
				ctx,
				admission.Request{
					AdmissionRequest: admissionv1.AdmissionRequest{
						Operation: testCase.op,
					},
				},
			)
		}
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.webhook.Default(ctx, testCase.freight)
			testCase.assertions(t, testCase.freight, err)
		})
	}
}

func TestValidateCreate(t *testing.T) {
	testCases := []struct {
		name       string
		webhook    *webhook
		freight    kargoapi.Freight
		assertions func(*testing.T, error)
	}{
		{
			name: "error validating project",
			webhook: &webhook{
				validateProjectFn: func(
					context.Context,
					client.Client,
					schema.GroupKind,
					client.Object,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
			},
		},
		{
			name: "error listing freight",
			freight: kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						kargoapi.AliasLabelKey: "fake-alias",
					},
				},
			},
			webhook: &webhook{
				validateProjectFn: func(
					context.Context,
					client.Client,
					schema.GroupKind,
					client.Object,
				) error {
					return nil
				},
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				statusErr, ok := err.(*apierrors.StatusError)
				require.True(t, ok)
				require.Equal(
					t,
					int32(http.StatusInternalServerError),
					statusErr.Status().Code,
				)
			},
		},
		{
			name: "alias already in use",
			freight: kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						kargoapi.AliasLabelKey: "fake-alias",
					},
				},
			},
			webhook: &webhook{
				validateProjectFn: func(
					context.Context,
					client.Client,
					schema.GroupKind,
					client.Object,
				) error {
					return nil
				},
				listFreightFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					freight, ok := objList.(*kargoapi.FreightList)
					require.True(t, ok)
					freight.Items = []kargoapi.Freight{{}}
					return nil
				},
			},
			assertions: func(t *testing.T, err error) {
				statusErr, ok := err.(*apierrors.StatusError)
				require.True(t, ok)
				require.Equal(t, int32(http.StatusConflict), statusErr.Status().Code)
			},
		},
		{
			name: "no artifacts",
			webhook: &webhook{
				validateProjectFn: func(
					context.Context,
					client.Client,
					schema.GroupKind,
					client.Object,
				) error {
					return nil
				},
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(
					t, err, "freight must contain at least one commit, image, or chart",
				)
			},
		},
		{
			name: "error getting warehouse",
			webhook: &webhook{
				validateProjectFn: func(
					context.Context,
					client.Client,
					schema.GroupKind,
					client.Object,
				) error {
					return nil
				},
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
				getWarehouseFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Warehouse, error) {
					return nil, fmt.Errorf("something went wrong")
				},
			},
			freight: kargoapi.Freight{
				Commits: []kargoapi.GitCommit{{}},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "warehouse does not exist",
			webhook: &webhook{
				validateProjectFn: func(
					context.Context,
					client.Client,
					schema.GroupKind,
					client.Object,
				) error {
					return nil
				},
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
				getWarehouseFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Warehouse, error) {
					return nil, nil
				},
			},
			freight: kargoapi.Freight{
				Commits: []kargoapi.GitCommit{{}},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "warehouse does not exist")
			},
		},
		{
			name: "artifact validation error",
			webhook: &webhook{
				validateProjectFn: func(
					context.Context,
					client.Client,
					schema.GroupKind,
					client.Object,
				) error {
					return nil
				},
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
				getWarehouseFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Warehouse, error) {
					return &kargoapi.Warehouse{}, nil
				},
				validateFreightArtifactsFn: func(
					*kargoapi.Freight,
					*kargoapi.Warehouse,
				) error {
					return errors.New("something went wrong")
				},
			},
			freight: kargoapi.Freight{
				Commits: []kargoapi.GitCommit{{}},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "success",
			webhook: &webhook{
				validateProjectFn: func(
					context.Context,
					client.Client,
					schema.GroupKind,
					client.Object,
				) error {
					return nil
				},
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
				getWarehouseFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Warehouse, error) {
					return &kargoapi.Warehouse{}, nil
				},
				validateFreightArtifactsFn: func(
					*kargoapi.Freight,
					*kargoapi.Warehouse,
				) error {
					return nil
				},
			},
			freight: kargoapi.Freight{
				Commits: []kargoapi.GitCommit{{}},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		tc := testCase // Avoid implicit memory aliasing
		t.Run(testCase.name, func(t *testing.T) {
			_, err := tc.webhook.ValidateCreate(context.Background(), &tc.freight)
			tc.assertions(t, err)
		})
	}
}

func TestValidateUpdate(t *testing.T) {
	testCases := []struct {
		name       string
		webhook    *webhook
		userInfo   *authnv1.UserInfo
		setup      func() (*kargoapi.Freight, *kargoapi.Freight)
		assertions func(*testing.T, *fakeevent.EventRecorder, error)
	}{
		{
			name: "error listing freight",
			setup: func() (*kargoapi.Freight, *kargoapi.Freight) {
				return &kargoapi.Freight{}, &kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							kargoapi.AliasLabelKey: "fake-alias",
						},
					},
				}
			},
			webhook: &webhook{
				validateProjectFn: func(
					context.Context,
					client.Client,
					schema.GroupKind,
					client.Object,
				) error {
					return nil
				},
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *fakeevent.EventRecorder, err error) {
				statusErr, ok := err.(*apierrors.StatusError)
				require.True(t, ok)
				require.Equal(
					t,
					int32(http.StatusInternalServerError),
					statusErr.Status().Code,
				)
			},
		},
		{
			name: "alias already in use",
			setup: func() (*kargoapi.Freight, *kargoapi.Freight) {
				return &kargoapi.Freight{}, &kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							kargoapi.AliasLabelKey: "fake-alias",
						},
					},
				}
			},
			webhook: &webhook{
				validateProjectFn: func(
					context.Context,
					client.Client,
					schema.GroupKind,
					client.Object,
				) error {
					return nil
				},
				listFreightFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					freight, ok := objList.(*kargoapi.FreightList)
					require.True(t, ok)
					freight.Items = []kargoapi.Freight{{}, {}}
					return nil
				},
			},
			assertions: func(t *testing.T, _ *fakeevent.EventRecorder, err error) {
				statusErr, ok := err.(*apierrors.StatusError)
				require.True(t, ok)
				require.Equal(t, int32(http.StatusConflict), statusErr.Status().Code)
			},
		},

		{
			name: "attempt to mutate artifacts",
			setup: func() (*kargoapi.Freight, *kargoapi.Freight) {
				oldFreight := &kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-namespace",
					},
					Commits: []kargoapi.GitCommit{
						{
							RepoURL: "fake-repo-url",
							ID:      "fake-commit-id",
						},
					},
				}
				oldFreight.Name = oldFreight.GenerateID()
				newFreight := oldFreight.DeepCopy()
				newFreight.Commits[0].ID = "another-fake-commit-id"
				return oldFreight, newFreight
			},
			webhook: &webhook{
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
			},
			assertions: func(t *testing.T, _ *fakeevent.EventRecorder, err error) {
				require.ErrorContains(t, err, "is invalid")
				require.ErrorContains(t, err, "Freight is immutable")
			},
		},

		{
			name: "attempt to mutate origin field",
			setup: func() (*kargoapi.Freight, *kargoapi.Freight) {
				oldFreight := &kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-namespace",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "fake-warehouse",
					},
				}
				oldFreight.Name = oldFreight.GenerateID()
				newFreight := oldFreight.DeepCopy()
				newFreight.Origin = kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: "another-fake-warehouse",
				}
				return oldFreight, newFreight
			},
			webhook: &webhook{
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
			},
			assertions: func(t *testing.T, _ *fakeevent.EventRecorder, err error) {
				require.ErrorContains(t, err, "is invalid")
				require.ErrorContains(t, err, "Freight is immutable")
			},
		},
		{
			name: "update without mutation",
			setup: func() (*kargoapi.Freight, *kargoapi.Freight) {
				oldFreight := &kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-name",
						Namespace: "fake-namespace",
					},
					Commits: []kargoapi.GitCommit{
						{
							RepoURL: "fake-repo-url",
							ID:      "fake-commit-id",
						},
					},
				}
				oldFreight.Name = oldFreight.GenerateID()
				newFreight := oldFreight.DeepCopy()
				return oldFreight, newFreight
			},
			webhook: &webhook{
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
				admissionRequestFromContextFn: admission.RequestFromContext,
				isRequestFromKargoControlplaneFn: libWebhook.IsRequestFromKargoControlplane(
					regexp.MustCompile("^system:serviceaccount:kargo:(kargo-api|kargo-controller)$"),
				),
			},
			userInfo: &authnv1.UserInfo{
				Username: "fake-user",
			},
			assertions: func(t *testing.T, r *fakeevent.EventRecorder, err error) {
				require.NoError(t, err)
				// Recorder should not record non-freight approval events
				require.Empty(t, r.Events)
			},
		},
		{
			name: "record approval event from non-controlplane",
			setup: func() (*kargoapi.Freight, *kargoapi.Freight) {
				oldFreight := &kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-name",
						Namespace: "fake-namespace",
					},
					Commits: []kargoapi.GitCommit{
						{
							RepoURL: "fake-repo-url",
							ID:      "fake-commit-id",
						},
					},
					Status: kargoapi.FreightStatus{},
				}
				oldFreight.Name = oldFreight.GenerateID()
				newFreight := oldFreight.DeepCopy()
				newFreight.Status.ApprovedFor = map[string]kargoapi.ApprovedStage{
					"fake-stage": {},
				}
				return oldFreight, newFreight
			},
			webhook: &webhook{
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
				admissionRequestFromContextFn: admission.RequestFromContext,
				isRequestFromKargoControlplaneFn: libWebhook.IsRequestFromKargoControlplane(
					regexp.MustCompile("^system:serviceaccount:kargo:(kargo-api|kargo-controller)$"),
				),
			},
			userInfo: &authnv1.UserInfo{
				Username: "fake-user",
			},
			assertions: func(t *testing.T, r *fakeevent.EventRecorder, err error) {
				require.NoError(t, err)
				require.Len(t, r.Events, 1)
				event := <-r.Events
				require.Equal(t, kargoapi.EventReasonFreightApproved, event.Reason)
			},
		},
		{
			name: "skip recording approval event from controlplane",
			setup: func() (*kargoapi.Freight, *kargoapi.Freight) {
				oldFreight := &kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-name",
						Namespace: "fake-namespace",
					},
					Commits: []kargoapi.GitCommit{
						{
							RepoURL: "fake-repo-url",
							ID:      "fake-commit-id",
						},
					},
					Status: kargoapi.FreightStatus{},
				}
				oldFreight.Name = oldFreight.GenerateID()
				newFreight := oldFreight.DeepCopy()
				newFreight.Status.ApprovedFor = map[string]kargoapi.ApprovedStage{
					"fake-stage": {},
				}
				return oldFreight, newFreight
			},
			webhook: &webhook{
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
				admissionRequestFromContextFn: admission.RequestFromContext,
				isRequestFromKargoControlplaneFn: libWebhook.IsRequestFromKargoControlplane(
					regexp.MustCompile("^system:serviceaccount:kargo:(kargo-api|kargo-controller)$"),
				),
			},
			userInfo: &authnv1.UserInfo{
				Username: serviceaccount.ServiceAccountUsernamePrefix + "kargo:kargo-api",
			},
			assertions: func(t *testing.T, r *fakeevent.EventRecorder, err error) {
				require.NoError(t, err)
				require.Empty(t, r.Events)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			oldFreight, newFreight := testCase.setup()

			recorder := fakeevent.NewEventRecorder(1)
			testCase.webhook.recorder = recorder

			var req admission.Request
			if testCase.userInfo != nil {
				req.UserInfo = *testCase.userInfo
			}
			ctx := admission.NewContextWithRequest(context.Background(), req)

			_, err := testCase.webhook.ValidateUpdate(
				ctx,
				oldFreight,
				newFreight,
			)
			testCase.assertions(t, recorder, err)
		})
	}
}

func TestValidateDelete(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	testCases := map[string]struct {
		input     *kargoapi.Freight
		webhook   *webhook
		shouldErr bool
	}{
		"idle freight": {
			input: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Name: "fake-freight",
				},
			},
			webhook: &webhook{
				listStagesFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
			},
			shouldErr: false,
		},
		"in-use freight": {
			input: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Name: "fake-freight",
				},
			},
			webhook: &webhook{
				listStagesFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					stages, ok := objList.(*kargoapi.StageList)
					require.True(t, ok)
					stages.Items = []kargoapi.Stage{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "fake-stage",
							},
							Status: kargoapi.StageStatus{
								FreightHistory: kargoapi.FreightHistory{{
									ID: "fake-id",
								}},
							},
						},
					}
					return nil
				},
			},
			shouldErr: true,
		},
	}
	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := tc.webhook.ValidateDelete(context.Background(), tc.input)
			if tc.shouldErr {
				require.Error(t, err)
				require.True(t, apierrors.IsForbidden(err))
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateFreightArtifacts(t *testing.T) {
	testCases := []struct {
		name       string
		freight    *kargoapi.Freight
		warehouse  *kargoapi.Warehouse
		assertions func(*testing.T, error)
	}{
		{
			name: "Freight missing artifact",
			freight: &kargoapi.Freight{
				Commits: []kargoapi.GitCommit{},
			},
			warehouse: &kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					Subscriptions: []kargoapi.RepoSubscription{
						{
							Git: &kargoapi.GitSubscription{
								RepoURL: "fake-repo-url",
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "no artifact found for subscription")
			},
		},
		{
			name: "Freight with duplicate Git artifact for Warehouse subscription",
			freight: &kargoapi.Freight{
				Commits: []kargoapi.GitCommit{
					{
						RepoURL: "fake-repo-url",
					},
					{
						RepoURL: "fake-repo-url",
					},
				},
			},
			warehouse: &kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					Subscriptions: []kargoapi.RepoSubscription{
						{
							Git: &kargoapi.GitSubscription{
								RepoURL: "fake-repo-url",
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "multiple artifacts found for subscription")
			},
		},
		{
			name: "Freight with duplicate image artifact for Warehouse subscription",
			freight: &kargoapi.Freight{
				Images: []kargoapi.Image{
					{
						RepoURL: "fake-repo-url",
					},
					{
						RepoURL: "fake-repo-url",
					},
				},
			},
			warehouse: &kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					Subscriptions: []kargoapi.RepoSubscription{
						{
							Image: &kargoapi.ImageSubscription{
								RepoURL: "fake-repo-url",
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "multiple artifacts found for subscription")
			},
		},
		{
			name: "Freight with multiple chart artifacts for Warehouse subscription",
			freight: &kargoapi.Freight{
				Charts: []kargoapi.Chart{
					{
						RepoURL: "fake-repo-url",
						Name:    "fake-name",
					},
					{
						RepoURL: "fake-repo-url",
						Name:    "fake-name",
					},
				},
			},
			warehouse: &kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					Subscriptions: []kargoapi.RepoSubscription{
						{
							Chart: &kargoapi.ChartSubscription{
								RepoURL: "fake-repo-url",
								Name:    "fake-name",
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "multiple artifacts found for subscription")
			},
		},
		{
			name: "Freight with Git commit not matching Warehouse subscription",
			freight: &kargoapi.Freight{
				Commits: []kargoapi.GitCommit{
					{
						RepoURL: "fake-repo-url",
					},
				},
			},
			warehouse: &kargoapi.Warehouse{},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "no subscription found for Git repository in Warehouse")
			},
		},
		{
			name: "Freight with image repository not matching Warehouse subscription",
			freight: &kargoapi.Freight{
				Images: []kargoapi.Image{
					{
						RepoURL: "fake-repo-url",
					},
				},
			},
			warehouse: &kargoapi.Warehouse{},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "no subscription found for image repository in Warehouse")
			},
		},
		{
			name: "Freight with Helm chart not matching Warehouse subscription",
			freight: &kargoapi.Freight{
				Charts: []kargoapi.Chart{
					{
						RepoURL: "fake-repo-url",
						Name:    "fake-name",
					},
				},
			},
			warehouse: &kargoapi.Warehouse{},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "no subscription found for Helm chart in Warehouse")
			},
		},
		{
			name: "success",
			freight: &kargoapi.Freight{
				Commits: []kargoapi.GitCommit{
					{
						RepoURL: "fake-git-repo-url",
					},
					{
						RepoURL: "fake-another-git-repo-url",
					},
				},
				Images: []kargoapi.Image{
					{
						RepoURL: "fake-image-repo-url",
					},
					{
						RepoURL: "fake-another-image-repo-url",
					},
				},
				Charts: []kargoapi.Chart{
					{
						RepoURL: "fake-chart-repo-url",
						Name:    "fake-chart-name",
					},
					{
						RepoURL: "fake-chart-repo-url",
						Name:    "fake-another-chart-name",
					},
					{
						RepoURL: "fake-another-chart-repo-url",
					},
				},
			},
			warehouse: &kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					Subscriptions: []kargoapi.RepoSubscription{
						{
							Git: &kargoapi.GitSubscription{
								RepoURL: "fake-git-repo-url",
							},
						},
						{
							Git: &kargoapi.GitSubscription{
								RepoURL: "fake-another-git-repo-url",
							},
						},
						{
							Image: &kargoapi.ImageSubscription{
								RepoURL: "fake-image-repo-url",
							},
						},
						{
							Image: &kargoapi.ImageSubscription{
								RepoURL: "fake-another-image-repo-url",
							},
						},
						{
							Chart: &kargoapi.ChartSubscription{
								RepoURL: "fake-chart-repo-url",
								Name:    "fake-chart-name",
							},
						},
						{
							Chart: &kargoapi.ChartSubscription{
								RepoURL: "fake-chart-repo-url",
								Name:    "fake-another-chart-name",
							},
						},
						{
							Chart: &kargoapi.ChartSubscription{
								RepoURL: "fake-another-chart-repo-url",
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := validateFreightArtifacts(testCase.freight, testCase.warehouse)
			testCase.assertions(t, err)
		})
	}
}

func TestCompareFreight(t *testing.T) {
	tests := []struct {
		name       string
		old        *kargoapi.Freight
		new        *kargoapi.Freight
		assertions func(*testing.T, *kargoapi.Freight, *field.Path, any, bool)
	}{
		{
			name: "Equal Freights",
			old: &kargoapi.Freight{
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: "warehouse1",
				},
				Commits: []kargoapi.GitCommit{{ID: "commit1"}}, Images: []kargoapi.Image{{RepoURL: "image1"}},
				Charts: []kargoapi.Chart{{Name: "chart1"}},
			},
			new: &kargoapi.Freight{
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: "warehouse1",
				},
				Commits: []kargoapi.GitCommit{{ID: "commit1"}},
				Images:  []kargoapi.Image{{RepoURL: "image1"}}, Charts: []kargoapi.Chart{{Name: "chart1"}},
			},
			assertions: func(t *testing.T, _ *kargoapi.Freight, path *field.Path, val any, eq bool) {
				require.Nil(t, path)
				require.Nil(t, val)
				require.True(t, eq)
			},
		},
		{
			name: "different origin",
			old: &kargoapi.Freight{
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: "warehouse1",
				},
			},
			new: &kargoapi.Freight{
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: "warehouse2",
				},
			},
			assertions: func(t *testing.T, freight *kargoapi.Freight, path *field.Path, val any, eq bool) {
				require.Equal(t, field.NewPath("origin"), path)
				require.Equal(t, freight.Origin, val)
				require.False(t, eq)
			},
		},
		{
			name: "different number of commits",
			old:  &kargoapi.Freight{Commits: []kargoapi.GitCommit{{ID: "commit1"}}},
			new:  &kargoapi.Freight{Commits: []kargoapi.GitCommit{{ID: "commit1"}, {ID: "commit2"}}},
			assertions: func(t *testing.T, freight *kargoapi.Freight, path *field.Path, val any, eq bool) {
				require.Equal(t, field.NewPath("commits"), path)
				require.Equal(t, freight.Commits, val)
				require.False(t, eq)
			},
		},
		{
			name: "different commit contents",
			old:  &kargoapi.Freight{Commits: []kargoapi.GitCommit{{ID: "commit1"}, {ID: "commit2"}}},
			new:  &kargoapi.Freight{Commits: []kargoapi.GitCommit{{ID: "commit1"}, {ID: "commit3"}}},
			assertions: func(t *testing.T, freight *kargoapi.Freight, path *field.Path, val any, eq bool) {
				require.Equal(t, field.NewPath("commits").Index(1), path)
				require.Equal(t, freight.Commits[1], val)
				require.False(t, eq)
			},
		},
		{
			name: "different number of images",
			old:  &kargoapi.Freight{Images: []kargoapi.Image{{RepoURL: "image1"}}},
			new:  &kargoapi.Freight{Images: []kargoapi.Image{{RepoURL: "image1"}, {RepoURL: "image2"}}},
			assertions: func(t *testing.T, freight *kargoapi.Freight, path *field.Path, val any, eq bool) {
				require.Equal(t, field.NewPath("images"), path)
				require.Equal(t, freight.Images, val)
				require.False(t, eq)
			},
		},
		{
			name: "different image contents",
			old:  &kargoapi.Freight{Images: []kargoapi.Image{{RepoURL: "image1"}}},
			new:  &kargoapi.Freight{Images: []kargoapi.Image{{RepoURL: "image2"}}},
			assertions: func(t *testing.T, freight *kargoapi.Freight, path *field.Path, val any, eq bool) {
				require.Equal(t, field.NewPath("images").Index(0), path)
				require.Equal(t, freight.Images[0], val)
				require.False(t, eq)
			},
		},
		{
			name: "different number of charts",
			old:  &kargoapi.Freight{Charts: []kargoapi.Chart{{Name: "chart1"}}},
			new:  &kargoapi.Freight{Charts: []kargoapi.Chart{{Name: "chart1"}, {Name: "chart2"}}},
			assertions: func(t *testing.T, freight *kargoapi.Freight, path *field.Path, val any, eq bool) {
				require.Equal(t, field.NewPath("charts"), path)
				require.Equal(t, freight.Charts, val)
				require.False(t, eq)
			},
		},
		{
			name: "different chart contents",
			old:  &kargoapi.Freight{Charts: []kargoapi.Chart{{Name: "chart1"}, {Name: "chart2"}, {Name: "chart3"}}},
			new:  &kargoapi.Freight{Charts: []kargoapi.Chart{{Name: "chart1"}, {Name: "chart2"}, {Name: "chart4"}}},
			assertions: func(t *testing.T, freight *kargoapi.Freight, path *field.Path, val any, eq bool) {
				require.Equal(t, field.NewPath("charts").Index(2), path)
				require.Equal(t, freight.Charts[2], val)
				require.False(t, eq)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, val, eq := compareFreight(tt.old, tt.new)
			tt.assertions(t, tt.new, path, val, eq)
		})
	}
}
