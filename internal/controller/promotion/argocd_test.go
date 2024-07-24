package promotion

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libargocd "github.com/akuity/kargo/internal/argocd"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
)

func TestNewArgoCDMechanism(t *testing.T) {
	pm := newArgoCDMechanism(fake.NewFakeClient(), fake.NewFakeClient())
	apm, ok := pm.(*argoCDMechanism)
	require.True(t, ok)
	require.Equal(t, "Argo CD promotion mechanism", apm.GetName())
	require.NotNil(t, apm.kargoClient)
	require.NotNil(t, apm.argocdClient)
	require.NotNil(t, apm.buildDesiredSourcesFn)
	require.NotNil(t, apm.mustPerformUpdateFn)
	require.NotNil(t, apm.updateApplicationSourcesFn)
	require.NotNil(t, apm.getAuthorizedApplicationFn)
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
		newFreight []kargoapi.FreightReference
		assertions func(
			t *testing.T,
			newStatus *kargoapi.PromotionStatus,
			newFreightIn []kargoapi.FreightReference,
			newFreightOut []kargoapi.FreightReference,
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
				newFreightIn []kargoapi.FreightReference,
				newFreightOut []kargoapi.FreightReference,
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
				_ []kargoapi.FreightReference,
				_ []kargoapi.FreightReference,
				err error,
			) {
				require.ErrorContains(
					t, err, "Argo CD integration is disabled on this controller",
				)
			},
		},
		{
			name: "error retrieving authorized application",
			promoMech: &argoCDMechanism{
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationFn: func(
					context.Context,
					string,
					string,
					metav1.ObjectMeta,
				) (*argocd.Application, error) {
					return nil, errors.New("something went wrong")
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
				newFreightIn []kargoapi.FreightReference,
				newFreightOut []kargoapi.FreightReference,
				err error,
			) {
				require.ErrorContains(t, err, "something went wrong")
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
		{
			name: "error building desired sources",
			promoMech: &argoCDMechanism{
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationFn: func(
					context.Context,
					string,
					string,
					metav1.ObjectMeta,
				) (*argocd.Application, error) {
					return &argocd.Application{}, nil
				},
				buildDesiredSourcesFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.ArgoCDAppUpdate,
					*argocd.Application,
					[]kargoapi.FreightReference,
				) (*argocd.ApplicationSource, argocd.ApplicationSources, error) {
					return nil, nil, errors.New("something went wrong")
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
				newFreightIn []kargoapi.FreightReference,
				newFreightOut []kargoapi.FreightReference,
				err error,
			) {
				require.ErrorContains(t, err, "something went wrong")
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
		{
			name: "error determining if update is necessary",
			promoMech: &argoCDMechanism{
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationFn: func(
					context.Context,
					string,
					string,
					metav1.ObjectMeta,
				) (*argocd.Application, error) {
					return &argocd.Application{}, nil
				},
				buildDesiredSourcesFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.ArgoCDAppUpdate,
					*argocd.Application,
					[]kargoapi.FreightReference,
				) (*argocd.ApplicationSource, argocd.ApplicationSources, error) {
					return nil, nil, nil
				},
				mustPerformUpdateFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.ArgoCDAppUpdate,
					*argocd.Application,
					[]kargoapi.FreightReference,
					*argocd.ApplicationSource,
					argocd.ApplicationSources,
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
				newFreightIn []kargoapi.FreightReference,
				newFreightOut []kargoapi.FreightReference,
				err error,
			) {
				require.ErrorContains(t, err, "something went wrong")
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
		{
			name: "determination error can be solved by applying update",
			promoMech: &argoCDMechanism{
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationFn: func(
					context.Context,
					string,
					string,
					metav1.ObjectMeta,
				) (*argocd.Application, error) {
					return &argocd.Application{}, nil
				},
				buildDesiredSourcesFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.ArgoCDAppUpdate,
					*argocd.Application,
					[]kargoapi.FreightReference,
				) (*argocd.ApplicationSource, argocd.ApplicationSources, error) {
					return nil, nil, nil
				},
				mustPerformUpdateFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.ArgoCDAppUpdate,
					*argocd.Application,
					[]kargoapi.FreightReference,
					*argocd.ApplicationSource,
					argocd.ApplicationSources,
				) (argocd.OperationPhase, bool, error) {
					return "", true, fmt.Errorf("something went wrong")
				},
				updateApplicationSourcesFn: func(
					context.Context,
					*argocd.Application,
					*argocd.ApplicationSource,
					argocd.ApplicationSources,
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
				_ []kargoapi.FreightReference,
				_ []kargoapi.FreightReference,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionPhaseRunning, status.Phase)
			},
		},
		{
			name: "must wait for update to complete",
			promoMech: &argoCDMechanism{
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationFn: func(
					context.Context,
					string,
					string,
					metav1.ObjectMeta,
				) (*argocd.Application, error) {
					return &argocd.Application{}, nil
				},
				buildDesiredSourcesFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.ArgoCDAppUpdate,
					*argocd.Application,
					[]kargoapi.FreightReference,
				) (*argocd.ApplicationSource, argocd.ApplicationSources, error) {
					return nil, nil, nil
				},
				mustPerformUpdateFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.ArgoCDAppUpdate,
					*argocd.Application,
					[]kargoapi.FreightReference,
					*argocd.ApplicationSource,
					argocd.ApplicationSources,
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
				newFreightIn []kargoapi.FreightReference,
				newFreightOut []kargoapi.FreightReference,
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
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationFn: func(
					context.Context,
					string,
					string,
					metav1.ObjectMeta,
				) (*argocd.Application, error) {
					return &argocd.Application{}, nil
				},
				buildDesiredSourcesFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.ArgoCDAppUpdate,
					*argocd.Application,
					[]kargoapi.FreightReference,
				) (*argocd.ApplicationSource, argocd.ApplicationSources, error) {
					return nil, nil, nil
				},
				mustPerformUpdateFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.ArgoCDAppUpdate,
					*argocd.Application,
					[]kargoapi.FreightReference,
					*argocd.ApplicationSource,
					argocd.ApplicationSources,
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
				newFreightIn []kargoapi.FreightReference,
				newFreightOut []kargoapi.FreightReference,
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
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationFn: func(
					context.Context,
					string,
					string,
					metav1.ObjectMeta,
				) (*argocd.Application, error) {
					return &argocd.Application{}, nil
				},
				buildDesiredSourcesFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.ArgoCDAppUpdate,
					*argocd.Application,
					[]kargoapi.FreightReference,
				) (*argocd.ApplicationSource, argocd.ApplicationSources, error) {
					return nil, nil, nil
				},
				mustPerformUpdateFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.ArgoCDAppUpdate,
					*argocd.Application,
					[]kargoapi.FreightReference,
					*argocd.ApplicationSource,
					argocd.ApplicationSources,
				) (argocd.OperationPhase, bool, error) {
					return "", true, nil
				},
				updateApplicationSourcesFn: func(
					context.Context,
					*argocd.Application,
					*argocd.ApplicationSource,
					argocd.ApplicationSources,
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
				newFreightIn []kargoapi.FreightReference,
				newFreightOut []kargoapi.FreightReference,
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
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationFn: func(
					context.Context,
					string,
					string,
					metav1.ObjectMeta,
				) (*argocd.Application, error) {
					return &argocd.Application{}, nil
				},
				buildDesiredSourcesFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.ArgoCDAppUpdate,
					*argocd.Application,
					[]kargoapi.FreightReference,
				) (*argocd.ApplicationSource, argocd.ApplicationSources, error) {
					return nil, nil, nil
				},
				mustPerformUpdateFn: func() func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.ArgoCDAppUpdate,
					*argocd.Application,
					[]kargoapi.FreightReference,
					*argocd.ApplicationSource,
					argocd.ApplicationSources,
				) (argocd.OperationPhase, bool, error) {
					var count uint
					return func(
						context.Context,
						*kargoapi.Stage,
						*kargoapi.ArgoCDAppUpdate,
						*argocd.Application,
						[]kargoapi.FreightReference,
						*argocd.ApplicationSource,
						argocd.ApplicationSources,
					) (argocd.OperationPhase, bool, error) {
						count++
						if count > 1 {
							return argocd.OperationFailed, false, nil
						}
						return "", true, nil
					}
				}(),
				updateApplicationSourcesFn: func(
					context.Context,
					*argocd.Application,
					*argocd.ApplicationSource,
					argocd.ApplicationSources,
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
				newFreightIn []kargoapi.FreightReference,
				newFreightOut []kargoapi.FreightReference,
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
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationFn: func(
					context.Context,
					string,
					string,
					metav1.ObjectMeta,
				) (*argocd.Application, error) {
					return &argocd.Application{}, nil
				},
				buildDesiredSourcesFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.ArgoCDAppUpdate,
					*argocd.Application,
					[]kargoapi.FreightReference,
				) (*argocd.ApplicationSource, argocd.ApplicationSources, error) {
					return nil, nil, nil
				},
				mustPerformUpdateFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.ArgoCDAppUpdate,
					*argocd.Application,
					[]kargoapi.FreightReference,
					*argocd.ApplicationSource,
					argocd.ApplicationSources,
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
				newFreightIn []kargoapi.FreightReference,
				newFreightOut []kargoapi.FreightReference,
				err error,
			) {
				require.ErrorContains(t, err, "could not determine promotion phase from operation phases")
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
		{
			name: "completed",
			promoMech: &argoCDMechanism{
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationFn: func(
					context.Context,
					string,
					string,
					metav1.ObjectMeta,
				) (*argocd.Application, error) {
					return &argocd.Application{}, nil
				},
				buildDesiredSourcesFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.ArgoCDAppUpdate,
					*argocd.Application,
					[]kargoapi.FreightReference,
				) (*argocd.ApplicationSource, argocd.ApplicationSources, error) {
					return nil, nil, nil
				},
				mustPerformUpdateFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.ArgoCDAppUpdate,
					*argocd.Application,
					[]kargoapi.FreightReference,
					*argocd.ApplicationSource,
					argocd.ApplicationSources,
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
				newFreightIn []kargoapi.FreightReference,
				newFreightOut []kargoapi.FreightReference,
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
			newStatus, newFreightOut, err := testCase.promoMech.Promote(
				logging.ContextWithLogger(
					context.Background(),
					logging.Wrap(logr.Discard()),
				),
				testCase.stage,
				&kargoapi.Promotion{},
				testCase.newFreight,
			)
			testCase.assertions(t, newStatus, testCase.newFreight, newFreightOut, err)
		})
	}
}

func TestArgoCDBuildDesiredSources(t *testing.T) {
	testCases := []struct {
		name              string
		reconciler        *argoCDMechanism
		modifyApplication func(*argocd.Application)
		update            kargoapi.ArgoCDAppUpdate
		assertions        func(
			t *testing.T,
			oldSource, newSource *argocd.ApplicationSource,
			oldSources, newSources argocd.ApplicationSources,
			err error,
		)
	}{
		{
			name: "applies updates to source",
			reconciler: &argoCDMechanism{
				applyArgoCDSourceUpdateFn: func(
					_ context.Context,
					_ *kargoapi.Stage,
					_ *kargoapi.ArgoCDSourceUpdate,
					src argocd.ApplicationSource,
					_ []kargoapi.FreightReference,
				) (argocd.ApplicationSource, error) {
					if src.RepoURL == "updated-url" {
						src.TargetRevision = "updated-revision"
						return src, nil
					}
					if src.RepoURL == "" {
						src.RepoURL = "updated-url"
					}
					return src, nil
				},
			},
			modifyApplication: func(app *argocd.Application) {
				app.Spec.Source = &argocd.ApplicationSource{}
			},
			update: kargoapi.ArgoCDAppUpdate{
				SourceUpdates: []kargoapi.ArgoCDSourceUpdate{
					{}, {},
				},
			},
			assertions: func(
				t *testing.T,
				oldSource, newSource *argocd.ApplicationSource,
				oldSources, newSources argocd.ApplicationSources,
				err error,
			) {
				require.NoError(t, err)
				require.True(t, oldSources.Equals(newSources))

				require.False(t, oldSource.Equals(newSource))
				require.Equal(t, "updated-url", newSource.RepoURL)
				require.Equal(t, "updated-revision", newSource.TargetRevision)
			},
		},
		{
			name: "error applying update to source",
			reconciler: &argoCDMechanism{
				applyArgoCDSourceUpdateFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.ArgoCDSourceUpdate,
					argocd.ApplicationSource,
					[]kargoapi.FreightReference,
				) (argocd.ApplicationSource, error) {
					return argocd.ApplicationSource{}, errors.New("something went wrong")
				},
			},
			modifyApplication: func(app *argocd.Application) {
				app.Spec.Source = &argocd.ApplicationSource{}
			},
			update: kargoapi.ArgoCDAppUpdate{
				SourceUpdates: []kargoapi.ArgoCDSourceUpdate{
					{},
				},
			},
			assertions: func(
				t *testing.T,
				_, newSource *argocd.ApplicationSource,
				_, newSources argocd.ApplicationSources,
				err error,
			) {
				require.ErrorContains(t, err, "something went wrong")
				require.Nil(t, newSource)
				require.Nil(t, newSources)
			},
		},
		{
			name: "applies updates to sources",
			reconciler: &argoCDMechanism{
				applyArgoCDSourceUpdateFn: func(
					_ context.Context,
					_ *kargoapi.Stage,
					_ *kargoapi.ArgoCDSourceUpdate,
					src argocd.ApplicationSource,
					_ []kargoapi.FreightReference,
				) (argocd.ApplicationSource, error) {
					if src.RepoURL == "url-1" {
						src.TargetRevision = "updated-revision-1"
						return src, nil
					}
					if src.RepoURL == "url-2" {
						src.TargetRevision = "updated-revision-2"
						return src, nil
					}
					return src, nil
				},
			},
			modifyApplication: func(app *argocd.Application) {
				app.Spec.Sources = argocd.ApplicationSources{
					{
						RepoURL: "url-1",
					},
					{
						RepoURL: "url-2",
					},
				}
			},
			update: kargoapi.ArgoCDAppUpdate{
				SourceUpdates: []kargoapi.ArgoCDSourceUpdate{
					{},
				},
			},
			assertions: func(
				t *testing.T,
				oldSource, newSource *argocd.ApplicationSource,
				oldSources, newSources argocd.ApplicationSources,
				err error,
			) {
				require.NoError(t, err)
				require.True(t, oldSource.Equals(newSource))
				require.False(t, oldSources.Equals(newSources))

				require.Equal(t, 2, len(newSources))
				require.Equal(t, "updated-revision-1", newSources[0].TargetRevision)
				require.Equal(t, "updated-revision-2", newSources[1].TargetRevision)
			},
		},
		{
			name: "error applying update to sources",
			reconciler: &argoCDMechanism{
				applyArgoCDSourceUpdateFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.ArgoCDSourceUpdate,
					argocd.ApplicationSource,
					[]kargoapi.FreightReference,
				) (argocd.ApplicationSource, error) {
					return argocd.ApplicationSource{}, errors.New("something went wrong")
				},
			},
			modifyApplication: func(app *argocd.Application) {
				app.Spec.Sources = argocd.ApplicationSources{
					{},
				}
			},
			update: kargoapi.ArgoCDAppUpdate{
				SourceUpdates: []kargoapi.ArgoCDSourceUpdate{
					{},
				},
			},
			assertions: func(
				t *testing.T,
				_, newSource *argocd.ApplicationSource,
				_, newSources argocd.ApplicationSources,
				err error,
			) {
				require.ErrorContains(t, err, "something went wrong")
				require.Nil(t, newSource)
				require.Nil(t, newSources)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			app := &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-app",
					Namespace: "fake-namespace",
				},
			}
			if testCase.modifyApplication != nil {
				testCase.modifyApplication(app)
			}

			stage := &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{
						ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{
							testCase.update,
						},
					},
				},
			}

			oldSource, oldSources := app.Spec.Source.DeepCopy(), app.Spec.Sources.DeepCopy()
			newSource, newSources, err := testCase.reconciler.buildDesiredSources(
				context.Background(),
				stage,
				&stage.Spec.PromotionMechanisms.ArgoCDAppUpdates[0],
				app,
				[]kargoapi.FreightReference{},
			)
			testCase.assertions(t, oldSource, newSource, oldSources, newSources, err)
		})
	}
}

func TestArgoCDMustPerformUpdate(t *testing.T) {
	testOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse",
	}
	testCases := []struct {
		name              string
		modifyApplication func(*argocd.Application)
		newFreight        []kargoapi.FreightReference
		desiredSource     *argocd.ApplicationSource
		desiredSources    argocd.ApplicationSources
		assertions        func(t *testing.T, phase argocd.OperationPhase, mustUpdate bool, err error)
	}{
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
					SyncResult: &argocd.SyncOperationResult{},
				}
			},
			assertions: func(t *testing.T, phase argocd.OperationPhase, mustUpdate bool, err error) {
				require.NoError(t, err)
				require.Equal(t, argocd.OperationSucceeded, phase)
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
			newFreight: []kargoapi.FreightReference{{
				Commits: []kargoapi.GitCommit{
					{
						RepoURL:           "https://github.com/universe/42",
						HealthCheckCommit: "fake-revision",
					},
				},
			}},
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
			newFreight: []kargoapi.FreightReference{{
				Origin: testOrigin,
				Commits: []kargoapi.GitCommit{
					{
						RepoURL: "https://github.com/universe/42",
						ID:      "fake-revision",
					},
				},
			}},
			assertions: func(t *testing.T, phase argocd.OperationPhase, mustUpdate bool, err error) {
				require.ErrorContains(t, err, "does not match desired revision")
				require.Empty(t, phase)
				require.True(t, mustUpdate)
			},
		},
		{
			name: "desired source does not match operation state",
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
						Source: argocd.ApplicationSource{
							RepoURL: "https://github.com/different/universe",
						},
					},
				}
			},
			desiredSource: &argocd.ApplicationSource{
				RepoURL: "http://github.com/universe/42",
			},
			assertions: func(t *testing.T, phase argocd.OperationPhase, mustUpdate bool, err error) {
				require.ErrorContains(t, err, "does not match desired source")
				require.Empty(t, phase)
				require.True(t, mustUpdate)
			},
		},
		{
			name: "desired sources do not match operation state",
			modifyApplication: func(app *argocd.Application) {
				app.Spec.Sources = argocd.ApplicationSources{
					{
						RepoURL: "https://github.com/universe/42",
					},
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
						Sources: argocd.ApplicationSources{
							{
								RepoURL: "https://github.com/different/universe",
							},
						},
					},
				}
			},
			desiredSource: &argocd.ApplicationSource{},
			desiredSources: argocd.ApplicationSources{
				{
					RepoURL: "https://github.com/universe/42",
				},
			},
			assertions: func(t *testing.T, phase argocd.OperationPhase, mustUpdate bool, err error) {
				require.ErrorContains(t, err, "does not match desired source")
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
			newFreight: []kargoapi.FreightReference{{
				Commits: []kargoapi.GitCommit{
					{
						RepoURL: "https://github.com/universe/42",
						ID:      "fake-revision",
					},
				},
			}},
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

			mechanism := newArgoCDMechanism(
				fake.NewFakeClient(),
				fake.NewClientBuilder().WithScheme(scheme).Build(),
			)
			argocdMech, ok := mechanism.(*argoCDMechanism)
			require.True(t, ok)

			stage := &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{
						ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{{
							SourceUpdates: []kargoapi.ArgoCDSourceUpdate{{
								Origin:  &testOrigin,
								RepoURL: "https://github.com/universe/42",
							}},
						}},
					},
				},
			}

			phase, mustUpdate, err := argocdMech.mustPerformUpdate(
				context.Background(),
				stage,
				&stage.Spec.PromotionMechanisms.ArgoCDAppUpdates[0],
				app,
				testCase.newFreight,
				testCase.desiredSource,
				testCase.desiredSources,
			)
			testCase.assertions(t, phase, mustUpdate, err)
		})
	}
}

func TestArgoCDUpdateApplicationSources(t *testing.T) {
	scheme := runtime.NewScheme()
	c := fake.NewClientBuilder().WithScheme(scheme)
	err := argocd.AddToScheme(scheme)
	require.NoError(t, err)
	proj := &argocd.AppProject{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fake-project",
		},
	}
	c.WithObjects(proj)

	testCases := []struct {
		name           string
		promoMech      *argoCDMechanism
		app            *argocd.Application
		desiredSource  *argocd.ApplicationSource
		desiredSources argocd.ApplicationSources
		assertions     func(*testing.T, error)
	}{
		{
			name: "error patching Application",
			promoMech: &argoCDMechanism{
				argoCDAppPatchFn: func(
					context.Context,
					client.Object,
					client.Patch,
					...client.PatchOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			app: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-name",
					Namespace: "fake-namespace",
					Annotations: map[string]string{
						authorizedStageAnnotationKey: "fake-namespace:fake-name",
					},
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error patching Argo CD Application")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "success",
			promoMech: &argoCDMechanism{
				argoCDAppPatchFn: func(
					context.Context,
					client.Object,
					client.Patch,
					...client.PatchOption,
				) error {
					return nil
				},
				logAppEventFn: func(context.Context, *argocd.Application, string, string, string) {},
			},
			app: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-name",
					Namespace: "fake-namespace",
					Annotations: map[string]string{
						authorizedStageAnnotationKey: "fake-namespace:fake-name",
					},
				},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		testCase.promoMech.argocdClient = c.Build()
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				testCase.promoMech.updateApplicationSources(
					context.Background(),
					testCase.app,
					testCase.desiredSource,
					testCase.desiredSources,
				),
			)
		})
	}
}

func TestLogAppEvent(t *testing.T) {
	testCases := []struct {
		name         string
		app          *argocd.Application
		user         string
		eventReason  string
		eventMessage string
		assertions   func(*testing.T, client.Client, *argocd.Application)
	}{
		{
			name: "success",
			app: &argocd.Application{
				TypeMeta: metav1.TypeMeta{
					Kind: "Application",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "fake-name",
					Namespace:       "fake-namespace",
					UID:             "fake-uid",
					ResourceVersion: "fake-resource-version",
				},
			},
			user:         "fake-user",
			eventReason:  "fake-reason",
			eventMessage: "fake-message",
			assertions: func(t *testing.T, c client.Client, app *argocd.Application) {
				events := &corev1.EventList{}
				require.NoError(t, c.List(context.Background(), events))
				require.Len(t, events.Items, 1)

				event := events.Items[0]
				require.Equal(t, corev1.ObjectReference{
					APIVersion:      argocd.GroupVersion.String(),
					Kind:            app.TypeMeta.Kind,
					Name:            app.ObjectMeta.Name,
					Namespace:       app.ObjectMeta.Namespace,
					UID:             app.ObjectMeta.UID,
					ResourceVersion: app.ObjectMeta.ResourceVersion,
				}, event.InvolvedObject)
				require.NotNil(t, event.FirstTimestamp)
				require.NotNil(t, event.LastTimestamp)
				require.Equal(t, 1, int(event.Count))
				require.Equal(t, corev1.EventTypeNormal, event.Type)
				require.Equal(t, "fake-reason", event.Reason)
				require.Equal(t, "fake-user fake-message", event.Message)
			},
		},
		{
			name: "unknown user",
			app: &argocd.Application{
				TypeMeta: metav1.TypeMeta{
					Kind: "Application",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "fake-name",
					Namespace:       "fake-namespace",
					UID:             "fake-uid",
					ResourceVersion: "fake-resource-version",
				},
			},
			eventReason:  "fake-reason",
			eventMessage: "fake-message",
			assertions: func(t *testing.T, c client.Client, _ *argocd.Application) {
				events := &corev1.EventList{}
				require.NoError(t, c.List(context.Background(), events))
				require.Len(t, events.Items, 1)

				event := events.Items[0]
				require.Equal(t, "Unknown user fake-message", event.Message)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			c := fake.NewFakeClient()
			(&argoCDMechanism{argocdClient: c}).logAppEvent(
				context.Background(),
				testCase.app,
				testCase.user,
				testCase.eventReason,
				testCase.eventMessage,
			)
			testCase.assertions(t, c, testCase.app)
		})
	}
}

func TestArgoCDGetAuthorizedApplication(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, argocd.AddToScheme(scheme))

	testCases := []struct {
		name         string
		obj          *argocd.Application
		appName      string
		appNamespace string
		interceptor  interceptor.Funcs
		stageMeta    metav1.ObjectMeta
		assertions   func(*testing.T, *argocd.Application, error)
	}{
		{
			name:         "error getting Application",
			appNamespace: "fake-namespace",
			appName:      "fake-name",
			interceptor: interceptor.Funcs{
				Get: func(
					context.Context,
					client.WithWatch,
					client.ObjectKey,
					client.Object,
					...client.GetOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, app *argocd.Application, err error) {
				require.ErrorContains(t, err, "error finding Argo CD Application")
				require.ErrorContains(t, err, "something went wrong")
				require.Nil(t, app)
			},
		},
		{
			name:         "Application not found",
			appNamespace: "fake-namespace",
			appName:      "fake-name",
			assertions: func(t *testing.T, app *argocd.Application, err error) {
				require.ErrorContains(t, err, "unable to find Argo CD Application")
				require.Nil(t, app)
			},
		},
		{
			name:         "Application not authorized for Stage",
			appNamespace: "fake-namespace",
			appName:      "fake-name",
			obj: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-name",
					Namespace: "fake-namespace",
				},
			},
			assertions: func(t *testing.T, app *argocd.Application, err error) {
				require.ErrorContains(t, err, "does not permit mutation by Kargo Stage")
				require.Nil(t, app)
			},
		},
		{
			name:         "success",
			appNamespace: "fake-namespace",
			appName:      "fake-name",
			obj: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-name",
					Namespace: "fake-namespace",
					Annotations: map[string]string{
						authorizedStageAnnotationKey: "fake-namespace:fake-stage",
					},
				},
			},
			stageMeta: metav1.ObjectMeta{
				Name:      "fake-stage",
				Namespace: "fake-namespace",
			},
			assertions: func(t *testing.T, app *argocd.Application, err error) {
				require.NoError(t, err)
				require.NotNil(t, app)
			},
		},
		{
			name:    "success with default namespace",
			appName: "fake-name",
			obj: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-name",
					Namespace: libargocd.Namespace(),
					Annotations: map[string]string{
						authorizedStageAnnotationKey: "*:fake-stage",
					},
				},
			},
			stageMeta: metav1.ObjectMeta{
				Name:      "fake-stage",
				Namespace: "fake-namespace",
			},
			assertions: func(t *testing.T, app *argocd.Application, err error) {
				require.NoError(t, err)
				require.NotNil(t, app)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithInterceptorFuncs(testCase.interceptor)

			if testCase.obj != nil {
				c.WithObjects(testCase.obj)
			}

			app, err := (&argoCDMechanism{argocdClient: c.Build()}).getAuthorizedApplication(
				context.Background(),
				testCase.appNamespace,
				testCase.appName,
				testCase.stageMeta,
			)
			testCase.assertions(t, app, err)
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
	testOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse",
	}
	testCases := []struct {
		name       string
		source     argocd.ApplicationSource
		freight    []kargoapi.FreightReference
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
			freight: []kargoapi.FreightReference{{
				Origin: testOrigin,
				Commits: []kargoapi.GitCommit{
					{
						RepoURL: "fake-url",
						ID:      "fake-commit",
					},
				},
			}},
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
			freight: []kargoapi.FreightReference{{
				Origin: testOrigin,
				Commits: []kargoapi.GitCommit{
					{
						RepoURL: "fake-url",
						ID:      "fake-commit",
						Tag:     "fake-tag",
					},
				},
			}},
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
			freight: []kargoapi.FreightReference{{
				Origin: testOrigin,
				Charts: []kargoapi.Chart{
					{
						RepoURL: "fake-url",
						Name:    "fake-chart",
						Version: "fake-version",
					},
				},
			}},
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
			freight: []kargoapi.FreightReference{{
				Origin: testOrigin,
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
			}},
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
			freight: []kargoapi.FreightReference{{
				Origin: testOrigin,
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
			}},
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
		stage := &kargoapi.Stage{
			Spec: kargoapi.StageSpec{
				PromotionMechanisms: &kargoapi.PromotionMechanisms{
					ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{{
						Origin:        &testOrigin,
						SourceUpdates: []kargoapi.ArgoCDSourceUpdate{testCase.update},
					}},
				},
			},
		}
		mech := &argoCDMechanism{}
		t.Run(testCase.name, func(t *testing.T) {
			updatedSource, err := mech.applyArgoCDSourceUpdate(
				context.Background(),
				stage,
				&stage.Spec.PromotionMechanisms.ArgoCDAppUpdates[0].SourceUpdates[0],
				testCase.source,
				testCase.freight,
			)
			testCase.assertions(t, testCase.source, updatedSource, err)
		})
	}
}

func TestBuildKustomizeImagesForArgoCDAppSource(t *testing.T) {
	testOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse",
	}
	freight := []kargoapi.FreightReference{{
		Origin: testOrigin,
		Images: []kargoapi.Image{
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
		},
	}}
	stage := &kargoapi.Stage{
		Spec: kargoapi.StageSpec{
			PromotionMechanisms: &kargoapi.PromotionMechanisms{
				ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{{
					Origin: &testOrigin,
					SourceUpdates: []kargoapi.ArgoCDSourceUpdate{{
						Kustomize: &kargoapi.ArgoCDKustomize{
							Images: []kargoapi.ArgoCDKustomizeImageUpdate{
								{Image: "fake-url"},
								{
									Image:     "another-fake-url",
									UseDigest: true,
								},
								{Image: "image-that-is-not-in-list"},
							},
						},
					}},
				}},
			},
		},
	}
	mech := &argoCDMechanism{}
	result, err := mech.buildKustomizeImagesForArgoCDAppSource(
		context.Background(),
		stage,
		stage.Spec.PromotionMechanisms.ArgoCDAppUpdates[0].SourceUpdates[0].Kustomize,
		freight,
	)
	require.NoError(t, err)
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
	testOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse",
	}
	freight := []kargoapi.FreightReference{{
		Origin: testOrigin,
		Images: []kargoapi.Image{
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
		},
	}}
	stage := &kargoapi.Stage{
		Spec: kargoapi.StageSpec{
			PromotionMechanisms: &kargoapi.PromotionMechanisms{
				ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{{
					Origin: &testOrigin,
					SourceUpdates: []kargoapi.ArgoCDSourceUpdate{{
						Helm: &kargoapi.ArgoCDHelm{
							Images: []kargoapi.ArgoCDHelmImageUpdate{
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
							},
						},
					}},
				}},
			},
		},
	}

	mech := &argoCDMechanism{}
	result, err := mech.buildHelmParamChangesForArgoCDAppSource(
		context.Background(),
		stage,
		stage.Spec.PromotionMechanisms.ArgoCDAppUpdates[0].SourceUpdates[0].Helm,
		freight,
	)
	require.NoError(t, err)
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
