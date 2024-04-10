package freight

import (
	"context"
	"errors"
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
	"k8s.io/apiserver/pkg/authentication/serviceaccount"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	fakekubeclient "github.com/akuity/kargo/internal/kubeclient/fake"
	libWebhook "github.com/akuity/kargo/internal/webhook"
)

func TestNewWebhook(t *testing.T) {
	kubeClient := fake.NewClientBuilder().Build()
	w := newWebhook(
		libWebhook.Config{},
		kubeClient,
		&fakekubeclient.EventRecorder{},
	)
	require.NotNil(t, w.freightAliasGenerator)
	// Assert that all overridable behaviors were initialized to a default:
	require.NotNil(t, w.admissionRequestFromContextFn)
	require.NotNil(t, w.getAvailableFreightAliasFn)
	require.NotNil(t, w.validateProjectFn)
	require.NotNil(t, w.listFreightFn)
	require.NotNil(t, w.listStagesFn)
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
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error getting admission request from context",
				)
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
				require.Error(t, err)
				require.Contains(t, err.Error(), "get available freight alias")
				require.Contains(t, err.Error(), "something went wrong")
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
				require.NotEmpty(t, freight.Name)
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
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"freight must contain at least one commit, image, or chart",
				)
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
		assertions func(*testing.T, *fakekubeclient.EventRecorder, error)
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
			assertions: func(t *testing.T, _ *fakekubeclient.EventRecorder, err error) {
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
			assertions: func(t *testing.T, _ *fakekubeclient.EventRecorder, err error) {
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
			assertions: func(t *testing.T, _ *fakekubeclient.EventRecorder, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "is invalid")
				require.Contains(t, err.Error(), "freight is immutable")
			},
		},

		{
			name: "attempt to mutate warehouse field",
			setup: func() (*kargoapi.Freight, *kargoapi.Freight) {
				oldFreight := &kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-namespace",
					},
					Warehouse: "fake-warehouse",
				}
				oldFreight.Name = oldFreight.GenerateID()
				newFreight := oldFreight.DeepCopy()
				newFreight.Warehouse = "another-fake-warehouse"
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
			assertions: func(t *testing.T, _ *fakekubeclient.EventRecorder, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "is invalid")
				require.Contains(t, err.Error(), "freight is immutable")
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
			assertions: func(t *testing.T, r *fakekubeclient.EventRecorder, err error) {
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
			assertions: func(t *testing.T, r *fakekubeclient.EventRecorder, err error) {
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
			assertions: func(t *testing.T, r *fakekubeclient.EventRecorder, err error) {
				require.NoError(t, err)
				require.Empty(t, r.Events)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			oldFreight, newFreight := testCase.setup()

			recorder := fakekubeclient.NewEventRecorder(1)
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
								CurrentFreight: &kargoapi.FreightReference{
									Name: "fake-id",
								},
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
