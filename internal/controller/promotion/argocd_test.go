package promotion

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
)

func TestNewArgoCDMechanism(t *testing.T) {
	pm := newArgoCDMechanism(
		fake.NewClientBuilder().Build(),
	)
	apm, ok := pm.(*argoCDMechanism)
	require.True(t, ok)
	require.NotNil(t, apm.mustPerformUpdateFn)
	require.NotNil(t, apm.doSingleUpdateFn)
	require.NotNil(t, apm.getArgoCDAppFn)
	require.NotNil(t, apm.applyArgoCDSourceUpdateFn)
	require.NotNil(t, apm.argoCDAppPatchFn)
}

func TestArgoCDGetName(t *testing.T) {
	require.NotEmpty(t, (&argoCDMechanism{}).GetName())
}

func TestArgoCDPromote(t *testing.T) {
	testCases := []struct {
		name       string
		promoMech  *argoCDMechanism
		stage      *kargoapi.Stage
		newFreight kargoapi.FreightReference
		assertions func(
			t *testing.T,
			newStatus *kargoapi.PromotionStatus,
			newFreightIn kargoapi.FreightReference,
			newFreightOut kargoapi.FreightReference,
			err error,
		)
	}{
		{
			name:      "no updates",
			promoMech: &argoCDMechanism{},
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.PromotionStatus,
				newFreightIn kargoapi.FreightReference,
				newFreightOut kargoapi.FreightReference,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
		{
			name:      "argo cd integration disabled",
			promoMech: &argoCDMechanism{},
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{
						ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{
							{},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.PromotionStatus,
				_ kargoapi.FreightReference,
				_ kargoapi.FreightReference,
				err error,
			) {
				require.ErrorContains(
					t, err, "Argo CD integration is disabled on this controller",
				)
			},
		},
		{
			name: "error determining if update is necessary",
			promoMech: &argoCDMechanism{
				argocdClient: fake.NewClientBuilder().Build(),
				mustPerformUpdateFn: func(
					context.Context,
					kargoapi.ArgoCDAppUpdate,
					kargoapi.FreightReference,
				) (argocd.OperationPhase, bool, error) {
					return "", false, errors.New("something went wrong")
				},
			},
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{
						ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{
							{},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.PromotionStatus,
				newFreightIn kargoapi.FreightReference,
				newFreightOut kargoapi.FreightReference,
				err error,
			) {
				require.ErrorContains(t, err, "something went wrong")
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
		{
			name: "determination error can be solved by applying update",
			promoMech: &argoCDMechanism{
				argocdClient: fake.NewClientBuilder().Build(),
				mustPerformUpdateFn: func(
					context.Context,
					kargoapi.ArgoCDAppUpdate,
					kargoapi.FreightReference,
				) (argocd.OperationPhase, bool, error) {
					return "", true, fmt.Errorf("something went wrong")
				},
				doSingleUpdateFn: func(
					context.Context,
					metav1.ObjectMeta,
					kargoapi.ArgoCDAppUpdate,
					kargoapi.FreightReference,
				) error {
					return nil
				},
			},
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{
						ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{
							{},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				status *kargoapi.PromotionStatus,
				_ kargoapi.FreightReference,
				_ kargoapi.FreightReference,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionPhaseRunning, status.Phase)
			},
		},
		{
			name: "must wait for update to complete",
			promoMech: &argoCDMechanism{
				argocdClient: fake.NewClientBuilder().Build(),
				mustPerformUpdateFn: func(
					context.Context,
					kargoapi.ArgoCDAppUpdate,
					kargoapi.FreightReference,
				) (argocd.OperationPhase, bool, error) {
					return argocd.OperationRunning, false, nil
				},
			},
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{
						ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{
							{},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				status *kargoapi.PromotionStatus,
				newFreightIn kargoapi.FreightReference,
				newFreightOut kargoapi.FreightReference,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionPhaseRunning, status.Phase)
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
		{
			name: "must wait for operation from different user to complete",
			promoMech: &argoCDMechanism{
				argocdClient: fake.NewClientBuilder().Build(),
				mustPerformUpdateFn: func(
					context.Context,
					kargoapi.ArgoCDAppUpdate,
					kargoapi.FreightReference,
				) (argocd.OperationPhase, bool, error) {
					return argocd.OperationRunning, false, fmt.Errorf("waiting for operation to complete")
				},
			},
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{
						ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{
							{},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				status *kargoapi.PromotionStatus,
				newFreightIn kargoapi.FreightReference,
				newFreightOut kargoapi.FreightReference,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionPhaseRunning, status.Phase)
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
		{
			name: "error applying update",
			promoMech: &argoCDMechanism{
				argocdClient: fake.NewClientBuilder().Build(),
				mustPerformUpdateFn: func(
					context.Context,
					kargoapi.ArgoCDAppUpdate,
					kargoapi.FreightReference,
				) (argocd.OperationPhase, bool, error) {
					return "", true, nil
				},
				doSingleUpdateFn: func(
					context.Context,
					metav1.ObjectMeta,
					kargoapi.ArgoCDAppUpdate,
					kargoapi.FreightReference,
				) error {
					return errors.New("something went wrong")
				},
			},
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{
						ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{
							{},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.PromotionStatus,
				newFreightIn kargoapi.FreightReference,
				newFreightOut kargoapi.FreightReference,
				err error,
			) {
				require.Error(t, err)
				require.Equal(
					t,
					"something went wrong",
					err.Error(),
				)
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
		{
			name: "failed and pending update",
			promoMech: &argoCDMechanism{
				argocdClient: fake.NewClientBuilder().Build(),
				mustPerformUpdateFn: func() func(
					context.Context,
					kargoapi.ArgoCDAppUpdate,
					kargoapi.FreightReference,
				) (argocd.OperationPhase, bool, error) {
					var count uint
					return func(
						context.Context,
						kargoapi.ArgoCDAppUpdate,
						kargoapi.FreightReference,
					) (argocd.OperationPhase, bool, error) {
						count++
						if count > 1 {
							return argocd.OperationFailed, false, nil
						}
						return "", true, nil
					}
				}(),
				doSingleUpdateFn: func(
					context.Context,
					metav1.ObjectMeta,
					kargoapi.ArgoCDAppUpdate,
					kargoapi.FreightReference,
				) error {
					return nil
				},
			},
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{
						ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{
							{},
							{},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				status *kargoapi.PromotionStatus,
				newFreightIn kargoapi.FreightReference,
				newFreightOut kargoapi.FreightReference,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionPhaseFailed, status.Phase)
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
		{
			name: "operation phase aggregation error",
			promoMech: &argoCDMechanism{
				argocdClient: fake.NewClientBuilder().Build(),
				mustPerformUpdateFn: func(
					context.Context,
					kargoapi.ArgoCDAppUpdate,
					kargoapi.FreightReference,
				) (argocd.OperationPhase, bool, error) {
					return "Unknown", false, nil
				},
			},
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{
						ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{
							{},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.PromotionStatus,
				newFreightIn kargoapi.FreightReference,
				newFreightOut kargoapi.FreightReference,
				err error,
			) {
				require.ErrorContains(t, err, "could not determine promotion phase from operation phases")
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
		{
			name: "completed",
			promoMech: &argoCDMechanism{
				argocdClient: fake.NewClientBuilder().Build(),
				mustPerformUpdateFn: func(
					context.Context,
					kargoapi.ArgoCDAppUpdate,
					kargoapi.FreightReference,
				) (argocd.OperationPhase, bool, error) {
					return argocd.OperationSucceeded, false, nil
				},
			},
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{
						ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{
							{},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				status *kargoapi.PromotionStatus,
				newFreightIn kargoapi.FreightReference,
				newFreightOut kargoapi.FreightReference,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionPhaseSucceeded, status.Phase)
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			logger := logrus.New()
			logger.Out = io.Discard

			newStatus, newFreightOut, err := testCase.promoMech.Promote(
				logging.ContextWithLogger(context.TODO(), logger.WithFields(nil)),
				testCase.stage,
				&kargoapi.Promotion{},
				testCase.newFreight,
			)
			testCase.assertions(t, newStatus, testCase.newFreight, newFreightOut, err)
		})
	}
}

func TestArgoCDMustPerformUpdate(t *testing.T) {
	testCases := []struct {
		name              string
		modifyApplication func(*argocd.Application)
		newFreight        kargoapi.FreightReference
		interceptor       interceptor.Funcs
		assertions        func(t *testing.T, phase argocd.OperationPhase, mustUpdate bool, err error)
	}{
		{
			name: "error getting Argo CD App",
			interceptor: interceptor.Funcs{
				Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, phase argocd.OperationPhase, mustUpdate bool, err error) {
				require.ErrorContains(t, err, "error finding Argo CD Application")
				require.ErrorContains(t, err, "something went wrong")
				require.Empty(t, phase)
				require.False(t, mustUpdate)
			},
		},
		{
			name: "Argo CD App not found",
			modifyApplication: func(app *argocd.Application) {
				app.ObjectMeta = metav1.ObjectMeta{}
			},
			assertions: func(t *testing.T, phase argocd.OperationPhase, mustUpdate bool, err error) {
				require.ErrorContains(t, err, "unable to find Argo CD Application")
				require.Empty(t, phase)
				require.False(t, mustUpdate)
			},
		},
		{
			name: "no operation state",
			assertions: func(t *testing.T, phase argocd.OperationPhase, mustUpdate bool, err error) {
				require.NoError(t, err)
				require.Empty(t, phase)
				require.True(t, mustUpdate)
			},
		},
		{
			name: "pending operation initiated by different user",
			modifyApplication: func(app *argocd.Application) {
				app.Status.OperationState = &argocd.OperationState{
					Phase: argocd.OperationRunning,
					Operation: argocd.Operation{
						InitiatedBy: argocd.OperationInitiator{
							Username: "someone-else",
						},
					},
				}
			},
			assertions: func(t *testing.T, phase argocd.OperationPhase, mustUpdate bool, err error) {
				require.ErrorContains(t, err, "current operation was not initiated by")
				require.ErrorContains(t, err, "waiting for operation to complete")
				require.Equal(t, argocd.OperationRunning, phase)
				require.False(t, mustUpdate)
			},
		},
		{
			name: "completed operation initiated by different user",
			modifyApplication: func(app *argocd.Application) {
				app.Status.OperationState = &argocd.OperationState{
					Phase: argocd.OperationSucceeded,
					Operation: argocd.Operation{
						InitiatedBy: argocd.OperationInitiator{
							Username: "someone-else",
						},
					},
				}
			},
			assertions: func(t *testing.T, phase argocd.OperationPhase, mustUpdate bool, err error) {
				require.NoError(t, err)
				require.True(t, mustUpdate)
				require.Empty(t, phase)
			},
		},
		{
			name: "pending operation initiated by us",
			modifyApplication: func(app *argocd.Application) {
				app.Status.OperationState = &argocd.OperationState{
					Phase: argocd.OperationRunning,
					Operation: argocd.Operation{
						InitiatedBy: argocd.OperationInitiator{
							Username: applicationOperationInitiator,
						},
					},
				}
			},
			assertions: func(t *testing.T, phase argocd.OperationPhase, mustUpdate bool, err error) {
				require.NoError(t, err)
				require.False(t, mustUpdate)
				require.Equal(t, argocd.OperationRunning, phase)
			},
		},
		{
			name: "unable to determine desired revision",
			modifyApplication: func(app *argocd.Application) {
				app.Status.OperationState = &argocd.OperationState{
					Phase: argocd.OperationSucceeded,
					Operation: argocd.Operation{
						InitiatedBy: argocd.OperationInitiator{
							Username: applicationOperationInitiator,
						},
					},
				}
			},
			assertions: func(t *testing.T, phase argocd.OperationPhase, mustUpdate bool, err error) {
				require.ErrorContains(t, err, "unable to determine desired revision")
				require.Empty(t, phase)
				require.False(t, mustUpdate)
			},
		},
		{
			name: "no sync result",
			modifyApplication: func(app *argocd.Application) {
				app.Spec.Source = &argocd.ApplicationSource{
					RepoURL: "https://github.com/universe/42",
				}
				app.Status.OperationState = &argocd.OperationState{
					Phase: argocd.OperationSucceeded,
					Operation: argocd.Operation{
						InitiatedBy: argocd.OperationInitiator{
							Username: applicationOperationInitiator,
						},
					},
				}
			},
			newFreight: kargoapi.FreightReference{
				Commits: []kargoapi.GitCommit{
					{
						RepoURL:           "https://github.com/universe/42",
						HealthCheckCommit: "fake-revision",
					},
				},
			},
			assertions: func(t *testing.T, phase argocd.OperationPhase, mustUpdate bool, err error) {
				require.ErrorContains(t, err, "operation completed without a sync result")
				require.Empty(t, phase)
				require.True(t, mustUpdate)
			},
		},
		{
			name: "desired revision does not match operation state",
			modifyApplication: func(app *argocd.Application) {
				app.Spec.Source = &argocd.ApplicationSource{
					RepoURL: "https://github.com/universe/42",
				}
				app.Status.OperationState = &argocd.OperationState{
					Phase: argocd.OperationSucceeded,
					Operation: argocd.Operation{
						InitiatedBy: argocd.OperationInitiator{
							Username: applicationOperationInitiator,
						},
					},
					SyncResult: &argocd.SyncOperationResult{
						Revision: "other-fake-revision",
					},
				}
			},
			newFreight: kargoapi.FreightReference{
				Commits: []kargoapi.GitCommit{
					{
						RepoURL: "https://github.com/universe/42",
						ID:      "fake-revision",
					},
				},
			},
			assertions: func(t *testing.T, phase argocd.OperationPhase, mustUpdate bool, err error) {
				require.ErrorContains(t, err, "does not match desired revision")
				require.Empty(t, phase)
				require.True(t, mustUpdate)
			},
		},
		{
			name: "operation completed",
			modifyApplication: func(app *argocd.Application) {
				app.Spec.Source = &argocd.ApplicationSource{
					RepoURL: "https://github.com/universe/42",
				}
				app.Status.OperationState = &argocd.OperationState{
					Phase: argocd.OperationSucceeded,
					Operation: argocd.Operation{
						InitiatedBy: argocd.OperationInitiator{
							Username: applicationOperationInitiator,
						},
					},
					SyncResult: &argocd.SyncOperationResult{
						Revision: "fake-revision",
					},
				}
			},
			newFreight: kargoapi.FreightReference{
				Commits: []kargoapi.GitCommit{
					{
						RepoURL: "https://github.com/universe/42",
						ID:      "fake-revision",
					},
				},
			},
			assertions: func(t *testing.T, phase argocd.OperationPhase, mustUpdate bool, err error) {
				require.NoError(t, err)
				require.Equal(t, argocd.OperationSucceeded, phase)
				require.False(t, mustUpdate)
			},
		},
	}

	for _, testCase := range testCases {
		scheme := runtime.NewScheme()
		require.NoError(t, argocd.AddToScheme(scheme))

		t.Run(testCase.name, func(t *testing.T) {
			app := &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-name",
					Namespace: "fake-namespace",
				},
			}
			if testCase.modifyApplication != nil {
				testCase.modifyApplication(app)
			}

			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(app).
				WithInterceptorFuncs(testCase.interceptor).
				Build()

			mechanism := newArgoCDMechanism(c)
			argocdMech, ok := mechanism.(*argoCDMechanism)
			require.True(t, ok)

			phase, mustUpdate, err := argocdMech.mustPerformUpdate(
				context.Background(),
				kargoapi.ArgoCDAppUpdate{
					AppName:      "fake-name",
					AppNamespace: "fake-namespace",
				},
				testCase.newFreight,
			)
			testCase.assertions(t, phase, mustUpdate, err)
		})
	}
}

func TestArgoCDDoSingleUpdate(t *testing.T) {
	testCases := []struct {
		name       string
		promoMech  *argoCDMechanism
		stageMeta  metav1.ObjectMeta
		update     kargoapi.ArgoCDAppUpdate
		assertions func(*testing.T, error)
	}{
		{
			name: "error getting Argo CD App",
			promoMech: &argoCDMechanism{
				getArgoCDAppFn: func(
					context.Context,
					string,
					string,
				) (*argocd.Application, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error finding Argo CD Application")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "Argo CD App not found",
			promoMech: &argoCDMechanism{
				getArgoCDAppFn: func(
					context.Context,
					string,
					string,
				) (*argocd.Application, error) {
					return nil, nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unable to find Argo CD Application")
			},
		},
		{
			name: "update not authorized",
			promoMech: &argoCDMechanism{
				getArgoCDAppFn: func(
					context.Context,
					string,
					string,
				) (*argocd.Application, error) {
					return &argocd.Application{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fake-name",
							Namespace: "fake-namespace",
							// The annotations that would permit this are missing
						},
					}, nil
				},
			},
			stageMeta: metav1.ObjectMeta{
				Name:      "fake-name",
				Namespace: "fake-namespace",
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "does not permit mutation by")
			},
		},
		{
			name: "error updating app.Spec.Source",
			promoMech: &argoCDMechanism{
				getArgoCDAppFn: func(
					context.Context,
					string,
					string,
				) (*argocd.Application, error) {
					return &argocd.Application{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fake-name",
							Namespace: "fake-namespace",
							Annotations: map[string]string{
								authorizedStageAnnotationKey: "fake-namespace:fake-name",
							},
						},
						Spec: argocd.ApplicationSpec{
							Source: &argocd.ApplicationSource{},
						},
					}, nil
				},
				applyArgoCDSourceUpdateFn: func(
					argocd.ApplicationSource,
					kargoapi.FreightReference,
					kargoapi.ArgoCDSourceUpdate,
				) (argocd.ApplicationSource, error) {
					return argocd.ApplicationSource{}, errors.New("something went wrong")
				},
			},
			stageMeta: metav1.ObjectMeta{
				Name:      "fake-name",
				Namespace: "fake-namespace",
			},
			update: kargoapi.ArgoCDAppUpdate{
				SourceUpdates: []kargoapi.ArgoCDSourceUpdate{
					{},
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error updating source of Argo CD Application")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error updating app.Spec.Sources",
			promoMech: &argoCDMechanism{
				getArgoCDAppFn: func(
					context.Context,
					string,
					string,
				) (*argocd.Application, error) {
					return &argocd.Application{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fake-name",
							Namespace: "fake-namespace",
							Annotations: map[string]string{
								authorizedStageAnnotationKey: "fake-namespace:fake-name",
							},
						},
						Spec: argocd.ApplicationSpec{
							Sources: []argocd.ApplicationSource{
								{},
							},
						},
					}, nil
				},
				applyArgoCDSourceUpdateFn: func(
					argocd.ApplicationSource,
					kargoapi.FreightReference,
					kargoapi.ArgoCDSourceUpdate,
				) (argocd.ApplicationSource, error) {
					return argocd.ApplicationSource{}, errors.New("something went wrong")
				},
			},
			stageMeta: metav1.ObjectMeta{
				Name:      "fake-name",
				Namespace: "fake-namespace",
			},
			update: kargoapi.ArgoCDAppUpdate{
				SourceUpdates: []kargoapi.ArgoCDSourceUpdate{
					{},
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error updating source(s) of Argo CD Application")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error patching Application",
			promoMech: &argoCDMechanism{
				getArgoCDAppFn: func(
					context.Context,
					string,
					string,
				) (*argocd.Application, error) {
					return &argocd.Application{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fake-name",
							Namespace: "fake-namespace",
							Annotations: map[string]string{
								authorizedStageAnnotationKey: "fake-namespace:fake-name",
							},
						},
					}, nil
				},
				argoCDAppPatchFn: func(
					context.Context,
					client.Object,
					client.Patch,
					...client.PatchOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			stageMeta: metav1.ObjectMeta{
				Name:      "fake-name",
				Namespace: "fake-namespace",
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error patching Argo CD Application")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "success",
			promoMech: &argoCDMechanism{
				getArgoCDAppFn: func(
					context.Context,
					string,
					string,
				) (*argocd.Application, error) {
					return &argocd.Application{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fake-name",
							Namespace: "fake-namespace",
							Annotations: map[string]string{
								authorizedStageAnnotationKey: "fake-namespace:fake-name",
							},
						},
					}, nil
				},
				argoCDAppPatchFn: func(
					context.Context,
					client.Object,
					client.Patch,
					...client.PatchOption,
				) error {
					return nil
				},
			},
			stageMeta: metav1.ObjectMeta{
				Name:      "fake-name",
				Namespace: "fake-namespace",
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				testCase.promoMech.doSingleUpdate(
					context.Background(),
					testCase.stageMeta,
					testCase.update,
					kargoapi.FreightReference{},
				),
			)
		})
	}
}

func TestAuthorizeArgoCDAppUpdate(t *testing.T) {
	permErr := "does not permit mutation"
	parseErr := "unable to parse"
	invalidGlobErr := "invalid glob expression"
	testCases := []struct {
		name    string
		appMeta metav1.ObjectMeta
		errMsg  string
	}{
		{
			name:    "annotations are nil",
			appMeta: metav1.ObjectMeta{},
			errMsg:  permErr,
		},
		{
			name: "annotation is missing",
			appMeta: metav1.ObjectMeta{
				Annotations: map[string]string{},
			},
			errMsg: permErr,
		},
		{
			name: "annotation cannot be parsed",
			appMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					authorizedStageAnnotationKey: "bogus",
				},
			},
			errMsg: parseErr,
		},
		{
			name: "mutation is not allowed",
			appMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					authorizedStageAnnotationKey: "ns-nope:name-nope",
				},
			},
			errMsg: permErr,
		},
		{
			name: "mutation is allowed",
			appMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					authorizedStageAnnotationKey: "ns-yep:name-yep",
				},
			},
		},
		{
			name: "wildcard namespace with full name",
			appMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					authorizedStageAnnotationKey: "*:name-yep",
				},
			},
		},
		{
			name: "full namespace with wildcard name",
			appMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					authorizedStageAnnotationKey: "ns-yep:*",
				},
			},
		},
		{
			name: "partial wildcards in namespace and name",
			appMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					authorizedStageAnnotationKey: "*-ye*:*-y*",
				},
			},
		},
		{
			name: "wildcards do not match",
			appMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					authorizedStageAnnotationKey: "*-nope:*-nope",
				},
			},
			errMsg: permErr,
		},
		{
			name: "invalid namespace glob",
			appMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					authorizedStageAnnotationKey: "*[:*",
				},
			},
			errMsg: invalidGlobErr,
		},
		{
			name: "invalid name glob",
			appMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					authorizedStageAnnotationKey: "*:*[",
				},
			},
			errMsg: invalidGlobErr,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := authorizeArgoCDAppUpdate(
				metav1.ObjectMeta{
					Name:      "name-yep",
					Namespace: "ns-yep",
				},
				testCase.appMeta,
			)
			if testCase.errMsg == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, testCase.errMsg)
			}
		})
	}
}

func TestApplyArgoCDSourceUpdate(t *testing.T) {
	testCases := []struct {
		name       string
		source     argocd.ApplicationSource
		newFreight kargoapi.FreightReference
		update     kargoapi.ArgoCDSourceUpdate
		assertions func(
			t *testing.T,
			originalSource argocd.ApplicationSource,
			updatedSource argocd.ApplicationSource,
			err error,
		)
	}{
		{
			name: "update doesn't apply to this source",
			source: argocd.ApplicationSource{
				RepoURL: "fake-url",
			},
			update: kargoapi.ArgoCDSourceUpdate{
				RepoURL: "different-fake-url",
			},
			assertions: func(
				t *testing.T,
				originalSource argocd.ApplicationSource,
				updatedSource argocd.ApplicationSource,
				err error,
			) {
				require.NoError(t, err)
				// Source should be entirely unchanged
				require.Equal(t, originalSource, updatedSource)
			},
		},

		{
			name: "update target revision (git)",
			source: argocd.ApplicationSource{
				RepoURL: "fake-url",
			},
			newFreight: kargoapi.FreightReference{
				Commits: []kargoapi.GitCommit{
					{
						RepoURL: "fake-url",
						ID:      "fake-commit",
					},
				},
			},
			update: kargoapi.ArgoCDSourceUpdate{
				RepoURL:              "fake-url",
				UpdateTargetRevision: true,
			},
			assertions: func(
				t *testing.T,
				originalSource argocd.ApplicationSource,
				updatedSource argocd.ApplicationSource,
				err error,
			) {
				require.NoError(t, err)
				// TargetRevision should be updated
				require.Equal(t, "fake-commit", updatedSource.TargetRevision)
				// Everything else should be unchanged
				updatedSource.TargetRevision = originalSource.TargetRevision
				require.Equal(t, originalSource, updatedSource)
			},
		},

		{
			name: "update target revision (git with tag)",
			source: argocd.ApplicationSource{
				RepoURL: "fake-url",
			},
			newFreight: kargoapi.FreightReference{
				Commits: []kargoapi.GitCommit{
					{
						RepoURL: "fake-url",
						ID:      "fake-commit",
						Tag:     "fake-tag",
					},
				},
			},
			update: kargoapi.ArgoCDSourceUpdate{
				RepoURL:              "fake-url",
				UpdateTargetRevision: true,
			},
			assertions: func(
				t *testing.T,
				originalSource argocd.ApplicationSource,
				updatedSource argocd.ApplicationSource,
				err error,
			) {
				require.NoError(t, err)
				// TargetRevision should be updated
				require.Equal(t, "fake-tag", updatedSource.TargetRevision)
				// Everything else should be unchanged
				updatedSource.TargetRevision = originalSource.TargetRevision
				require.Equal(t, originalSource, updatedSource)
			},
		},

		{
			name: "update target revision (helm chart)",
			source: argocd.ApplicationSource{
				RepoURL: "fake-url",
				Chart:   "fake-chart",
			},
			newFreight: kargoapi.FreightReference{
				Charts: []kargoapi.Chart{
					{
						RepoURL: "fake-url",
						Name:    "fake-chart",
						Version: "fake-version",
					},
				},
			},
			update: kargoapi.ArgoCDSourceUpdate{
				RepoURL:              "fake-url",
				Chart:                "fake-chart",
				UpdateTargetRevision: true,
			},
			assertions: func(
				t *testing.T,
				originalSource argocd.ApplicationSource,
				updatedSource argocd.ApplicationSource,
				err error,
			) {
				require.NoError(t, err)
				// TargetRevision should be updated
				require.Equal(t, "fake-version", updatedSource.TargetRevision)
				// Everything else should be unchanged
				updatedSource.TargetRevision = originalSource.TargetRevision
				require.Equal(t, originalSource, updatedSource)
			},
		},

		{
			name: "update images with kustomize",
			source: argocd.ApplicationSource{
				RepoURL: "fake-url",
			},
			newFreight: kargoapi.FreightReference{
				Images: []kargoapi.Image{
					{
						RepoURL: "fake-image-url",
						Tag:     "fake-tag",
					},
					{
						// This one should not be updated because it's not a match for
						// anything in the update instructions
						RepoURL: "another-fake-image-url",
						Tag:     "another-fake-tag",
					},
				},
			},
			update: kargoapi.ArgoCDSourceUpdate{
				RepoURL: "fake-url",
				Kustomize: &kargoapi.ArgoCDKustomize{
					Images: []kargoapi.ArgoCDKustomizeImageUpdate{
						{
							Image: "fake-image-url",
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				originalSource argocd.ApplicationSource,
				updatedSource argocd.ApplicationSource,
				err error,
			) {
				require.NoError(t, err)
				// Kustomize attributes should be updated
				require.NotNil(t, updatedSource.Kustomize)
				require.Equal(
					t,
					argocd.KustomizeImages{
						argocd.KustomizeImage("fake-image-url=fake-image-url:fake-tag"),
					},
					updatedSource.Kustomize.Images,
				)
				// Everything else should be unchanged
				updatedSource.Kustomize = originalSource.Kustomize
				require.Equal(t, originalSource, updatedSource)
			},
		},

		{
			name: "update images with helm",
			source: argocd.ApplicationSource{
				RepoURL: "fake-url",
			},
			newFreight: kargoapi.FreightReference{
				Images: []kargoapi.Image{
					{
						RepoURL: "fake-image-url",
						Tag:     "fake-tag",
					},
					{
						// This one should not be updated because it's not a match for
						// anything in the update instructions
						RepoURL: "another-fake-image-url",
						Tag:     "another-fake-tag",
					},
				},
			},
			update: kargoapi.ArgoCDSourceUpdate{
				RepoURL: "fake-url",
				Helm: &kargoapi.ArgoCDHelm{
					Images: []kargoapi.ArgoCDHelmImageUpdate{
						{
							Image: "fake-image-url",
							Key:   "image",
							Value: kargoapi.ImageUpdateValueTypeImageAndTag,
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				originalSource argocd.ApplicationSource,
				updatedSource argocd.ApplicationSource,
				err error,
			) {
				require.NoError(t, err)
				// Helm attributes should be updated
				require.NotNil(t, updatedSource.Helm)
				require.NotNil(t, updatedSource.Helm.Parameters)
				require.Equal(
					t,
					[]argocd.HelmParameter{
						{
							Name:  "image",
							Value: "fake-image-url:fake-tag",
						},
					},
					updatedSource.Helm.Parameters,
				)
				// Everything else should be unchanged
				updatedSource.Helm = originalSource.Helm
				require.Equal(t, originalSource, updatedSource)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			updatedSource, err := applyArgoCDSourceUpdate(
				testCase.source,
				testCase.newFreight,
				testCase.update,
			)
			testCase.assertions(t, testCase.source, updatedSource, err)
		})
	}
}

func TestBuildKustomizeImagesForArgoCDAppSource(t *testing.T) {
	images := []kargoapi.Image{
		{
			RepoURL: "fake-url",
			Tag:     "fake-tag",
			Digest:  "fake-digest",
		},
		{
			RepoURL: "another-fake-url",
			Tag:     "another-fake-tag",
			Digest:  "another-fake-digest",
		},
	}
	imageUpdates := []kargoapi.ArgoCDKustomizeImageUpdate{
		{Image: "fake-url"},
		{
			Image:     "another-fake-url",
			UseDigest: true,
		},
		{Image: "image-that-is-not-in-list"},
	}
	result := buildKustomizeImagesForArgoCDAppSource(images, imageUpdates)
	require.Equal(
		t,
		argocd.KustomizeImages{
			"fake-url=fake-url:fake-tag",
			"another-fake-url=another-fake-url@another-fake-digest",
		},
		result,
	)
}

func TestBuildHelmParamChangesForArgoCDAppSource(t *testing.T) {
	images := []kargoapi.Image{
		{
			RepoURL: "fake-url",
			Tag:     "fake-tag",
			Digest:  "fake-digest",
		},
		{
			RepoURL: "second-fake-url",
			Tag:     "second-fake-tag",
			Digest:  "second-fake-digest",
		},
		{
			RepoURL: "third-fake-url",
			Tag:     "third-fake-tag",
			Digest:  "third-fake-digest",
		},
		{
			RepoURL: "fourth-fake-url",
			Tag:     "fourth-fake-tag",
			Digest:  "fourth-fake-digest",
		},
	}
	imageUpdates := []kargoapi.ArgoCDHelmImageUpdate{
		{
			Image: "fake-url",
			Key:   "fake-key",
			Value: kargoapi.ImageUpdateValueTypeImageAndTag,
		},
		{
			Image: "second-fake-url",
			Key:   "second-fake-key",
			Value: kargoapi.ImageUpdateValueTypeTag,
		},
		{
			Image: "third-fake-url",
			Key:   "third-fake-key",
			Value: kargoapi.ImageUpdateValueTypeImageAndDigest,
		},
		{
			Image: "fourth-fake-url",
			Key:   "fourth-fake-key",
			Value: kargoapi.ImageUpdateValueTypeDigest,
		},
		{
			Image: "image-that-is-not-in-list",
			Key:   "fake-key",
			Value: "Tag",
		},
	}
	result := buildHelmParamChangesForArgoCDAppSource(images, imageUpdates)
	require.Equal(
		t,
		map[string]string{
			"fake-key":        "fake-url:fake-tag",
			"second-fake-key": "second-fake-tag",
			"third-fake-key":  "third-fake-url@third-fake-digest",
			"fourth-fake-key": "fourth-fake-digest",
		},
		result,
	)
}
