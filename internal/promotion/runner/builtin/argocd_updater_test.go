package builtin

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/internal/kubeclient"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_newArgocdUpdater(t *testing.T) {
	runner := newArgocdUpdater(fake.NewFakeClient())
	require.NotNil(t, runner)
	require.Equal(t, "argocd-update", runner.Name())
	require.NotNil(t, runner.schemaLoader)
	require.NotNil(t, runner.getAuthorizedApplicationFn)
	require.NotNil(t, runner.buildDesiredSourcesFn)
	require.NotNil(t, runner.mustPerformUpdateFn)
	require.NotNil(t, runner.syncApplicationFn)
	require.NotNil(t, runner.applyArgoCDSourceUpdateFn)
	require.NotNil(t, runner.argoCDAppPatchFn)
	require.NotNil(t, runner.logAppEventFn)
}

func Test_argoCDUpdater_validate(t *testing.T) {
	testCases := []struct {
		name             string
		config           promotion.Config
		expectedProblems []string
	}{
		{
			name:   "apps not specified",
			config: promotion.Config{},
			expectedProblems: []string{
				"(root): apps is required",
			},
		},
		{
			name: "apps is empty array",
			config: promotion.Config{
				"apps": []promotion.Config{},
			},
			expectedProblems: []string{
				"apps: Array must have at least 1 items",
			},
		},
		{
			name: "app name not specified",
			config: promotion.Config{
				"apps": []promotion.Config{{}},
			},
			expectedProblems: []string{
				"apps.0: name is required",
			},
		},
		{
			name: "app name is empty string",
			config: promotion.Config{
				"apps": []promotion.Config{{
					"name": "",
				}},
			},
			expectedProblems: []string{
				"apps.0.name: String length must be greater than or equal to 1",
			},
		},
		{
			name: "app namespace is empty string",
			config: promotion.Config{
				"apps": []promotion.Config{{
					"namespace": "",
				}},
			},
			expectedProblems: []string{
				"apps.0.namespace: String length must be greater than or equal to 1",
			},
		},
		{
			name: "app sources is empty array",
			config: promotion.Config{
				"apps": []promotion.Config{{
					"sources": []promotion.Config{},
				}},
			},
			expectedProblems: []string{
				"apps.0.sources: Array must have at least 1 items",
			},
		},
		{
			name: "source repoURL not specified",
			config: promotion.Config{
				"apps": []promotion.Config{{
					"sources": []promotion.Config{{}},
				}},
			},
			expectedProblems: []string{
				"apps.0.sources.0: repoURL is required",
			},
		},
		{
			name: "source repoURL is empty string",
			config: promotion.Config{
				"apps": []promotion.Config{{
					"sources": []promotion.Config{{
						"repoURL": "",
					}},
				}},
			},
			expectedProblems: []string{
				"apps.0.sources.0.repoURL: String length must be greater than or equal to 1",
			},
		},
		{
			name: "targetRevision=true with desiredCommitFromStep and desiredRevision unspecified",
			config: promotion.Config{
				"apps": []promotion.Config{{
					"sources": []promotion.Config{{
						"updateTargetRevision": true,
					}},
				}},
			},
			expectedProblems: []string{
				"apps.0.sources.0: Must validate one and only one schema",
			},
		},
		{
			name: "helm images is empty array",
			config: promotion.Config{
				"apps": []promotion.Config{{
					"sources": []promotion.Config{{
						"helm": promotion.Config{
							"images": []promotion.Config{},
						},
					}},
				}},
			},
			expectedProblems: []string{
				"apps.0.sources.0.helm.images: Array must have at least 1 items",
			},
		},
		{
			name: "helm images update key is not specified",
			config: promotion.Config{
				"apps": []promotion.Config{{
					"sources": []promotion.Config{{
						"helm": promotion.Config{
							"images": []promotion.Config{{}},
						},
					}},
				}},
			},
			expectedProblems: []string{
				"apps.0.sources.0.helm.images.0: key is required",
			},
		},
		{
			name: "helm images update key is empty string",
			config: promotion.Config{
				"apps": []promotion.Config{{
					"sources": []promotion.Config{{
						"helm": promotion.Config{
							"images": []promotion.Config{{
								"key": "",
							}},
						},
					}},
				}},
			},
			expectedProblems: []string{
				"apps.0.sources.0.helm.images.0.key: String length must be greater than or equal to 1",
			},
		},
		{
			name: "helm images update value is not specified",
			config: promotion.Config{
				"apps": []promotion.Config{{
					"sources": []promotion.Config{{
						"helm": promotion.Config{
							"images": []promotion.Config{{}},
						},
					}},
				}},
			},
			expectedProblems: []string{
				"apps.0.sources.0.helm.images.0: value is required",
			},
		},
		{
			name: "kustomize images is empty array",
			config: promotion.Config{
				"apps": []promotion.Config{{
					"sources": []promotion.Config{{
						"kustomize": promotion.Config{
							"images": []promotion.Config{},
						},
					}},
				}},
			},
			expectedProblems: []string{
				"apps.0.sources.0.kustomize.images: Array must have at least 1 items",
			},
		},
		{
			name: "kustomize images update newName is empty string",
			config: promotion.Config{
				"apps": []promotion.Config{{
					"sources": []promotion.Config{{
						"kustomize": promotion.Config{
							"images": []promotion.Config{{
								"newName": "",
							}},
						},
					}},
				}},
			},
			expectedProblems: []string{
				"apps.0.sources.0.kustomize.images.0.newName: String length must be greater than or equal to 1",
			},
		},
		{
			name: "kustomize images update repoURL is not specified",
			config: promotion.Config{
				"apps": []promotion.Config{{
					"sources": []promotion.Config{{
						"kustomize": promotion.Config{
							"images": []promotion.Config{{}},
						},
					}},
				}},
			},
			expectedProblems: []string{
				"apps.0.sources.0.kustomize.images.0: repoURL is required",
			},
		},
		{
			name: "kustomize images update repoURL is empty string",
			config: promotion.Config{
				"apps": []promotion.Config{{
					"sources": []promotion.Config{{
						"kustomize": promotion.Config{
							"images": []promotion.Config{{
								"repoURL": "",
							}},
						},
					}},
				}},
			},
			expectedProblems: []string{
				"apps.0.sources.0.kustomize.images.0.repoURL: String length must be greater than or equal to 1",
			},
		},
		{
			name: "kustomize images digest and tag are both specified",
			// These are meant to be mutually exclusive.
			config: promotion.Config{
				"apps": []promotion.Config{{
					"sources": []promotion.Config{{
						"kustomize": promotion.Config{
							"images": []promotion.Config{{
								"digest": "fake-digest",
								"tag":    "fake-tag",
							}},
						},
					}},
				}},
			},
			expectedProblems: []string{
				"apps.0.sources.0.kustomize.images.0: Must validate one and only one schema (oneOf)",
			},
		},
		{
			name: "valid kitchen sink",
			config: promotion.Config{
				"apps": []promotion.Config{{
					"name":      "app",
					"namespace": "argocd",
					"sources": []promotion.Config{{
						"repoURL":              "fake-git-url",
						"desiredRevision":      "fake-commit",
						"updateTargetRevision": true,
						"helm": promotion.Config{
							"images": []promotion.Config{
								{
									"key":   "another-fake-key",
									"value": "fake-tag",
								},
							},
						},
						"kustomize": promotion.Config{
							"images": []promotion.Config{
								{
									"repoURL": "another-fake-image",
									"digest":  "fake-digest",
								},
								{
									"repoURL": "yet-another-fake-image",
									"tag":     "fake-tag",
								},
							},
						},
					}},
				}},
			},
		},
	}

	runner := newArgocdUpdater(nil)

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := runner.validate(testCase.config)
			if len(testCase.expectedProblems) == 0 {
				require.NoError(t, err)
			} else {
				for _, problem := range testCase.expectedProblems {
					require.ErrorContains(t, err, problem)
				}
			}
		})
	}
}

func Test_argoCDUpdater_run(t *testing.T) {
	testCases := []struct {
		name       string
		runner     *argocdUpdater
		stepCfg    builtin.ArgoCDUpdateConfig
		assertions func(*testing.T, promotion.StepResult, error)
	}{
		{
			name:    "argo cd integration disabled",
			runner:  &argocdUpdater{},
			stepCfg: builtin.ArgoCDUpdateConfig{},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.Equal(t, kargoapi.PromotionStepPhaseErrored, res.Status)
				require.ErrorContains(
					t, err, "Argo CD integration is disabled on this controller",
				)
			},
		},
		{
			name: "error retrieving authorized application",
			runner: &argocdUpdater{
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationFn: func(
					context.Context,
					*promotion.StepContext,
					client.ObjectKey,
				) (*v1alpha1.Application, error) {
					return nil, errors.New("something went wrong")
				},
			},
			stepCfg: builtin.ArgoCDUpdateConfig{
				Apps: []builtin.ArgoCDAppUpdate{{}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.Equal(t, kargoapi.PromotionStepPhaseErrored, res.Status)
				require.ErrorContains(t, err, "error getting Argo CD Application")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error determining if update is necessary",
			runner: &argocdUpdater{
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationFn: func(
					context.Context,
					*promotion.StepContext,
					client.ObjectKey,
				) (*v1alpha1.Application, error) {
					return &argocd.Application{}, nil
				},
				mustPerformUpdateFn: func(
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
					*argocd.Application,
				) (argocd.OperationPhase, bool, error) {
					return "", false, errors.New("something went wrong")
				},
			},
			stepCfg: builtin.ArgoCDUpdateConfig{
				Apps: []builtin.ArgoCDAppUpdate{{}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.Equal(t, kargoapi.PromotionStepPhaseErrored, res.Status)
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "determination error can be solved by applying update",
			runner: &argocdUpdater{
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationFn: func(
					context.Context,
					*promotion.StepContext,
					client.ObjectKey,
				) (*v1alpha1.Application, error) {
					return &argocd.Application{}, nil
				},
				buildDesiredSourcesFn: func(
					*builtin.ArgoCDAppUpdate,
					[]string,
					*argocd.Application,
				) (argocd.ApplicationSources, error) {
					return []argocd.ApplicationSource{{}}, nil
				},
				mustPerformUpdateFn: func(
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
					*argocd.Application,
				) (argocd.OperationPhase, bool, error) {
					return "", true, errors.New("something went wrong")
				},
				syncApplicationFn: func(
					context.Context,
					*promotion.StepContext,
					*argocd.Application,
					argocd.ApplicationSources,
				) error {
					return nil
				},
			},
			stepCfg: builtin.ArgoCDUpdateConfig{
				Apps: []builtin.ArgoCDAppUpdate{{}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.Equal(t, kargoapi.PromotionStepPhaseRunning, res.Status)
				require.NoError(t, err)
			},
		},
		{
			name: "must wait for update to complete",
			runner: &argocdUpdater{
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationFn: func(
					context.Context,
					*promotion.StepContext,
					client.ObjectKey,
				) (*v1alpha1.Application, error) {
					return &argocd.Application{}, nil
				},
				mustPerformUpdateFn: func(
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
					*argocd.Application,
				) (argocd.OperationPhase, bool, error) {
					return argocd.OperationRunning, false, nil
				},
			},
			stepCfg: builtin.ArgoCDUpdateConfig{
				Apps: []builtin.ArgoCDAppUpdate{{}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.Equal(t, kargoapi.PromotionStepPhaseRunning, res.Status)
				require.NoError(t, err)
			},
		},
		{
			name: "must wait for operation from different user to complete",
			runner: &argocdUpdater{
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationFn: func(
					context.Context,
					*promotion.StepContext,
					client.ObjectKey,
				) (*v1alpha1.Application, error) {
					return &argocd.Application{}, nil
				},
				mustPerformUpdateFn: func(
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
					*argocd.Application,
				) (argocd.OperationPhase, bool, error) {
					return argocd.OperationRunning, false, fmt.Errorf("waiting for operation to complete")
				},
			},
			stepCfg: builtin.ArgoCDUpdateConfig{
				Apps: []builtin.ArgoCDAppUpdate{{}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.Equal(t, kargoapi.PromotionStepPhaseRunning, res.Status)
				require.NoError(t, err)
			},
		},
		{
			name: "error building desired sources",
			runner: &argocdUpdater{
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationFn: func(
					context.Context,
					*promotion.StepContext,
					client.ObjectKey,
				) (*v1alpha1.Application, error) {
					return &argocd.Application{}, nil
				},
				mustPerformUpdateFn: func(
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
					*argocd.Application,
				) (argocd.OperationPhase, bool, error) {
					return "", true, nil
				},
				buildDesiredSourcesFn: func(
					*builtin.ArgoCDAppUpdate,
					[]string,
					*argocd.Application,
				) (argocd.ApplicationSources, error) {
					return nil, errors.New("something went wrong")
				},
			},
			stepCfg: builtin.ArgoCDUpdateConfig{
				Apps: []builtin.ArgoCDAppUpdate{{}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.Equal(t, kargoapi.PromotionStepPhaseErrored, res.Status)
				require.ErrorContains(t, err, "error building desired sources for Argo CD Application")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error applying update",
			runner: &argocdUpdater{
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationFn: func(
					context.Context,
					*promotion.StepContext,
					client.ObjectKey,
				) (*v1alpha1.Application, error) {
					return &argocd.Application{}, nil
				},
				mustPerformUpdateFn: func(
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
					*argocd.Application,
				) (argocd.OperationPhase, bool, error) {
					return "", true, nil
				},
				buildDesiredSourcesFn: func(
					*builtin.ArgoCDAppUpdate,
					[]string,
					*argocd.Application,
				) (argocd.ApplicationSources, error) {
					return []argocd.ApplicationSource{{}}, nil
				},
				syncApplicationFn: func(
					context.Context,
					*promotion.StepContext,
					*argocd.Application,
					argocd.ApplicationSources,
				) error {
					return errors.New("something went wrong")
				},
			},
			stepCfg: builtin.ArgoCDUpdateConfig{
				Apps: []builtin.ArgoCDAppUpdate{{}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.Equal(t, kargoapi.PromotionStepPhaseErrored, res.Status)
				require.ErrorContains(t, err, "error syncing Argo CD Application")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "failed and pending update",
			runner: &argocdUpdater{
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationFn: func(
					context.Context,
					*promotion.StepContext,
					client.ObjectKey,
				) (*v1alpha1.Application, error) {
					return &argocd.Application{}, nil
				},
				mustPerformUpdateFn: func() func(
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
					*argocd.Application,
				) (argocd.OperationPhase, bool, error) {
					var count uint
					return func(
						*promotion.StepContext,
						*builtin.ArgoCDAppUpdate,
						*argocd.Application,
					) (argocd.OperationPhase, bool, error) {
						count++
						if count > 1 {
							return argocd.OperationFailed, false, nil
						}
						return "", true, nil
					}
				}(),
				buildDesiredSourcesFn: func(
					*builtin.ArgoCDAppUpdate,
					[]string,
					*argocd.Application,
				) (argocd.ApplicationSources, error) {
					return []argocd.ApplicationSource{{}}, nil
				},
				syncApplicationFn: func(
					context.Context,
					*promotion.StepContext,
					*argocd.Application,
					argocd.ApplicationSources,
				) error {
					return nil
				},
			},
			stepCfg: builtin.ArgoCDUpdateConfig{
				Apps: []builtin.ArgoCDAppUpdate{{}, {}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.Equal(t, kargoapi.PromotionStepPhaseErrored, res.Status)
				require.NoError(t, err)
			},
		},
		{
			name: "operation phase aggregation error",
			runner: &argocdUpdater{
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationFn: func(
					context.Context,
					*promotion.StepContext,
					client.ObjectKey,
				) (*v1alpha1.Application, error) {
					return &argocd.Application{}, nil
				},
				mustPerformUpdateFn: func(
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
					*argocd.Application,
				) (argocd.OperationPhase, bool, error) {
					return "Unknown", false, nil
				},
			},
			stepCfg: builtin.ArgoCDUpdateConfig{
				Apps: []builtin.ArgoCDAppUpdate{{}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.Equal(t, kargoapi.PromotionStepPhaseErrored, res.Status)
				require.ErrorContains(t, err, "could not determine promotion step status")
			},
		},
		{
			name: "completed",
			runner: &argocdUpdater{
				getAuthorizedApplicationFn: func(
					context.Context,
					*promotion.StepContext,
					client.ObjectKey,
				) (*v1alpha1.Application, error) {
					return &argocd.Application{}, nil
				},
				mustPerformUpdateFn: func(
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
					*argocd.Application,
				) (argocd.OperationPhase, bool, error) {
					return argocd.OperationSucceeded, false, nil
				},
				argocdClient: fake.NewFakeClient(),
			},
			stepCfg: builtin.ArgoCDUpdateConfig{
				Apps: []builtin.ArgoCDAppUpdate{{}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.Equal(t, kargoapi.PromotionStepPhaseSucceeded, res.Status)
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			res, err := testCase.runner.run(
				context.Background(),
				&promotion.StepContext{},
				testCase.stepCfg,
			)
			testCase.assertions(t, res, err)
		})
	}
}

func Test_argoCDUpdater_buildDesiredSources(t *testing.T) {
	testCases := []struct {
		name             string
		runner           *argocdUpdater
		update           *builtin.ArgoCDAppUpdate
		desiredRevisions []string
		app              *argocd.Application
		assertions       func(
			t *testing.T,
			desiredSources argocd.ApplicationSources,
			err error,
		)
	}{
		{
			name: "applies updates to sources",
			runner: &argocdUpdater{
				applyArgoCDSourceUpdateFn: func(
					update *builtin.ArgoCDAppSourceUpdate,
					_ string,
					src argocd.ApplicationSource,
				) (argocd.ApplicationSource, bool) {
					if update.RepoURL == "fake-chart-url" && update.Chart == "fake-chart" &&
						src.RepoURL == "fake-chart-url" && src.Chart == "fake-chart" {
						src.TargetRevision = "fake-version"
						return src, true
					}
					if update.RepoURL == "fake-git-url" && src.RepoURL == "fake-git-url" {
						src.TargetRevision = "fake-commit"
						return src, true
					}
					return src, false
				},
			},
			update: &builtin.ArgoCDAppUpdate{
				Sources: []builtin.ArgoCDAppSourceUpdate{
					{
						RepoURL: "fake-chart-url",
						Chart:   "fake-chart",
					},
					{
						RepoURL: "fake-git-url",
					},
				},
			},
			desiredRevisions: []string{"fake-version", "fake-commit"},
			app: &argocd.Application{
				Spec: argocd.ApplicationSpec{
					Sources: []argocd.ApplicationSource{
						{
							RepoURL: "fake-chart-url",
							Chart:   "fake-chart",
						},
						{
							RepoURL: "fake-git-url",
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				desiredSources argocd.ApplicationSources,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, 2, len(desiredSources))
				require.Equal(t, "fake-version", desiredSources[0].TargetRevision)
				require.Equal(t, "fake-commit", desiredSources[1].TargetRevision)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			desiredSources, err := testCase.runner.buildDesiredSources(
				testCase.update,
				testCase.desiredRevisions,
				testCase.app,
			)
			testCase.assertions(t, desiredSources, err)
		})
	}
}

func Test_argoCDUpdater_mustPerformUpdate(t *testing.T) {
	testPromotionID := "fake-promotion"
	testCases := []struct {
		name              string
		modifyApplication func(*argocd.Application)
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
			name: "running operation initiated by different user",
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
				require.ErrorContains(t, err, "current operation was initiated by")
				require.ErrorContains(t, err, "and not by")
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
			name: "running operation initiated for incorrect Promotion",
			modifyApplication: func(app *argocd.Application) {
				app.Status.OperationState = &argocd.OperationState{
					Phase: argocd.OperationRunning,
					Operation: argocd.Operation{
						InitiatedBy: argocd.OperationInitiator{
							Username: applicationOperationInitiator,
						},
						Info: []*argocd.Info{{
							Name:  promotionInfoKey,
							Value: "wrong-freight-collection",
						}},
					},
				}
			},
			assertions: func(t *testing.T, phase argocd.OperationPhase, mustUpdate bool, err error) {
				require.ErrorContains(t, err, "current operation was not initiated for")
				require.ErrorContains(t, err, "waiting for operation to complete")
				require.Equal(t, argocd.OperationRunning, phase)
				require.False(t, mustUpdate)
			},
		},
		{
			name: "completed operation initiated for incorrect Promotion",
			modifyApplication: func(app *argocd.Application) {
				app.Status.OperationState = &argocd.OperationState{
					Phase: argocd.OperationSucceeded,
					Operation: argocd.Operation{
						InitiatedBy: argocd.OperationInitiator{
							Username: applicationOperationInitiator,
						},
						Info: []*argocd.Info{{
							Name:  promotionInfoKey,
							Value: "wrong-freight-collection",
						}},
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
			name: "running operation",
			modifyApplication: func(app *argocd.Application) {
				app.Status.OperationState = &argocd.OperationState{
					Phase: argocd.OperationRunning,
					Operation: argocd.Operation{
						InitiatedBy: argocd.OperationInitiator{
							Username: applicationOperationInitiator,
						},
						Info: []*argocd.Info{{
							Name:  promotionInfoKey,
							Value: testPromotionID,
						}},
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
			name: "unable to determine desired revisions",
			modifyApplication: func(app *argocd.Application) {
				app.Status.OperationState = &argocd.OperationState{
					Phase: argocd.OperationSucceeded,
					Operation: argocd.Operation{
						InitiatedBy: argocd.OperationInitiator{
							Username: applicationOperationInitiator,
						},
						Info: []*argocd.Info{{
							Name:  promotionInfoKey,
							Value: testPromotionID,
						}},
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
						Info: []*argocd.Info{{
							Name:  promotionInfoKey,
							Value: testPromotionID,
						}},
					},
				}
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
						Info: []*argocd.Info{{
							Name:  promotionInfoKey,
							Value: testPromotionID,
						}},
					},
					SyncResult: &argocd.SyncOperationResult{
						Revision: "other-fake-revision",
					},
				}
			},
			assertions: func(t *testing.T, phase argocd.OperationPhase, mustUpdate bool, err error) {
				require.ErrorContains(t, err, "sync result revisions")
				require.ErrorContains(t, err, "do not match desired revisions")
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
						Info: []*argocd.Info{{
							Name:  promotionInfoKey,
							Value: testPromotionID,
						}},
					},
					SyncResult: &argocd.SyncOperationResult{
						Revision: "fake-revision",
					},
				}
			},
			assertions: func(t *testing.T, phase argocd.OperationPhase, mustUpdate bool, err error) {
				require.NoError(t, err)
				require.Equal(t, argocd.OperationSucceeded, phase)
				require.False(t, mustUpdate)
			},
		},
	}

	runner := &argocdUpdater{}

	for _, testCase := range testCases {
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

			stepCfg := &builtin.ArgoCDUpdateConfig{
				Apps: []builtin.ArgoCDAppUpdate{{
					Sources: []builtin.ArgoCDAppSourceUpdate{{
						RepoURL:         "https://github.com/universe/42",
						DesiredRevision: "fake-revision",
					}},
				}},
			}

			phase, mustUpdate, err := runner.mustPerformUpdate(
				&promotion.StepContext{Promotion: testPromotionID},
				&stepCfg.Apps[0],
				app,
			)
			testCase.assertions(t, phase, mustUpdate, err)
		})
	}
}

func Test_argoCDUpdater_syncApplication(t *testing.T) {
	testCases := []struct {
		name           string
		runner         *argocdUpdater
		app            *argocd.Application
		desiredSources argocd.ApplicationSources
		assertions     func(*testing.T, error)
	}{
		{
			name: "error patching Application",
			runner: &argocdUpdater{
				argoCDAppPatchFn: func(
					context.Context,
					kubeclient.ObjectWithKind,
					kubeclient.UnstructuredPatchFn,
				) error {
					return errors.New("something went wrong")
				},
			},
			app: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-name",
					Namespace: "fake-namespace",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAuthorizedStage: "fake-namespace:fake-name",
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
			runner: &argocdUpdater{
				argoCDAppPatchFn: func(
					context.Context,
					kubeclient.ObjectWithKind,
					kubeclient.UnstructuredPatchFn,
				) error {
					return nil
				},
				logAppEventFn: func(
					context.Context,
					*argocd.Application,
					string,
					string,
					string,
				) {
				},
			},
			app: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-name",
					Namespace: "fake-namespace",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAuthorizedStage: "fake-namespace:fake-name",
					},
				},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}

	stepCtx := &promotion.StepContext{
		Freight: kargoapi.FreightCollection{},
	}
	// Tamper with the freight collection ID for testing purposes
	stepCtx.Freight.ID = "fake-freight-collection-id"

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				testCase.runner.syncApplication(
					context.Background(),
					stepCtx,
					testCase.app,
					testCase.desiredSources,
				),
			)
		})
	}
}

func Test_argoCDUpdater_logAppEvent(t *testing.T) {
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
			runner := &argocdUpdater{
				argocdClient: c,
			}
			runner.logAppEvent(
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

func Test_argoCDUpdater_getAuthorizedApplication(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, argocd.AddToScheme(scheme))

	testCases := []struct {
		name        string
		app         *argocd.Application
		interceptor interceptor.Funcs
		assertions  func(*testing.T, *argocd.Application, error)
	}{
		{
			name: "error getting Application",
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
			name: "Application not found",
			assertions: func(t *testing.T, app *argocd.Application, err error) {
				require.ErrorContains(t, err, "unable to find Argo CD Application")
				require.Nil(t, app)
			},
		},
		{
			name: "Application not authorized for Stage",
			app: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-app",
					Namespace: "fake-namespace",
				},
			},
			assertions: func(t *testing.T, app *argocd.Application, err error) {
				require.ErrorContains(t, err, "does not permit mutation by Kargo Stage")
				require.Nil(t, app)
			},
		},
		{
			name: "success",
			app: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-app",
					Namespace: "fake-namespace",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAuthorizedStage: "fake-namespace:fake-stage",
					},
				},
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

			if testCase.app != nil {
				c.WithObjects(testCase.app)
			}

			runner := &argocdUpdater{
				argocdClient: c.Build(),
			}
			app, err := runner.getAuthorizedApplication(
				context.Background(),
				&promotion.StepContext{
					Project: "fake-namespace",
					Stage:   "fake-stage",
				},
				client.ObjectKey{
					Namespace: "fake-namespace",
					Name:      "fake-app",
				},
			)
			testCase.assertions(t, app, err)
		})
	}
}

func Test_argoCDUpdater_authorizeArgoCDAppUpdate(t *testing.T) {
	const (
		permErr           = "does not permit mutation"
		parseErr          = "unable to parse"
		deprecatedGlobErr = "deprecated glob expression"
	)

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
					kargoapi.AnnotationKeyAuthorizedStage: "bogus",
				},
			},
			errMsg: parseErr,
		},
		{
			name: "mutation is not allowed",
			appMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					kargoapi.AnnotationKeyAuthorizedStage: "ns-nope:name-nope",
				},
			},
			errMsg: permErr,
		},
		{
			name: "mutation is allowed",
			appMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					kargoapi.AnnotationKeyAuthorizedStage: "ns-yep:name-yep",
				},
			},
		},
		{
			name: "wildcard namespace with full name",
			appMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					kargoapi.AnnotationKeyAuthorizedStage: "*:name-yep",
				},
			},
			errMsg: deprecatedGlobErr,
		},
		{
			name: "full namespace with wildcard name",
			appMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					kargoapi.AnnotationKeyAuthorizedStage: "ns-yep:*",
				},
			},
			errMsg: deprecatedGlobErr,
		},
	}

	runner := &argocdUpdater{}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := runner.authorizeArgoCDAppUpdate(
				&promotion.StepContext{
					Project: "ns-yep",
					Stage:   "name-yep",
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

func Test_argoCDUpdater_applyArgoCDSourceUpdate(t *testing.T) {
	testCases := []struct {
		name            string
		source          argocd.ApplicationSource
		update          builtin.ArgoCDAppSourceUpdate
		desiredRevision string
		assertions      func(
			t *testing.T,
			originalSource argocd.ApplicationSource,
			updated bool,
			updatedSource argocd.ApplicationSource,
		)
	}{
		{
			name: "update doesn't apply to this source",
			source: argocd.ApplicationSource{
				RepoURL: "fake-url",
			},
			update: builtin.ArgoCDAppSourceUpdate{
				RepoURL: "different-fake-url",
			},
			assertions: func(
				t *testing.T,
				originalSource argocd.ApplicationSource,
				updated bool,
				updatedSource argocd.ApplicationSource,
			) {
				require.False(t, updated)
				// Source should be entirely unchanged
				require.Equal(t, originalSource, updatedSource)
			},
		},

		{
			name: "update target revision (git)",
			source: argocd.ApplicationSource{
				RepoURL: "fake-url",
			},
			update: builtin.ArgoCDAppSourceUpdate{
				RepoURL:              "fake-url",
				UpdateTargetRevision: true,
			},
			desiredRevision: "fake-commit",
			assertions: func(
				t *testing.T,
				originalSource argocd.ApplicationSource,
				updated bool,
				updatedSource argocd.ApplicationSource,
			) {
				require.True(t, updated)
				// TargetRevision should be updated
				require.Equal(t, "fake-commit", updatedSource.TargetRevision)
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
			update: builtin.ArgoCDAppSourceUpdate{
				RepoURL:              "fake-url",
				Chart:                "fake-chart",
				UpdateTargetRevision: true,
			},
			desiredRevision: "fake-version",
			assertions: func(
				t *testing.T,
				originalSource argocd.ApplicationSource,
				updated bool,
				updatedSource argocd.ApplicationSource,
			) {
				require.True(t, updated)
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
			update: builtin.ArgoCDAppSourceUpdate{
				RepoURL: "fake-url",
				Kustomize: &builtin.ArgoCDKustomizeImageUpdates{
					Images: []builtin.ArgoCDKustomizeImageUpdate{{
						RepoURL: "fake-image-url",
						Tag:     "fake-tag",
					}},
				},
			},
			assertions: func(
				t *testing.T,
				originalSource argocd.ApplicationSource,
				updated bool,
				updatedSource argocd.ApplicationSource,
			) {
				require.True(t, updated)
				// Kustomize attributes should be updated
				require.NotNil(t, updatedSource.Kustomize)
				require.Equal(
					t,
					argocd.KustomizeImages{
						"fake-image-url:fake-tag",
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
			update: builtin.ArgoCDAppSourceUpdate{
				RepoURL: "fake-url",
				Helm: &builtin.ArgoCDHelmParameterUpdates{
					Images: []builtin.ArgoCDHelmImageUpdate{
						{
							Key:   "image",
							Value: "fake-image-url:fake-tag",
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				originalSource argocd.ApplicationSource,
				updated bool,
				updatedSource argocd.ApplicationSource,
			) {
				require.True(t, updated)
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

	runner := &argocdUpdater{}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			stepCfg := &builtin.ArgoCDUpdateConfig{
				Apps: []builtin.ArgoCDAppUpdate{{
					Sources: []builtin.ArgoCDAppSourceUpdate{testCase.update},
				}},
			}
			updatedSource, updated := runner.applyArgoCDSourceUpdate(
				&stepCfg.Apps[0].Sources[0],
				testCase.desiredRevision,
				testCase.source,
			)
			testCase.assertions(t, testCase.source, updated, updatedSource)
		})
	}
}

func Test_argoCDUpdater_buildKustomizeImagesForAppSource(t *testing.T) {
	stepCfg := &builtin.ArgoCDUpdateConfig{
		Apps: []builtin.ArgoCDAppUpdate{{
			Sources: []builtin.ArgoCDAppSourceUpdate{{
				Kustomize: &builtin.ArgoCDKustomizeImageUpdates{
					Images: []builtin.ArgoCDKustomizeImageUpdate{
						{
							RepoURL: "yet-another-fake-url",
							Digest:  "fake-digest",
						},
						{
							RepoURL: "still-another-fake-url",
							Tag:     "fake-tag",
						},
						{
							RepoURL: "another-fake-url",
							NewName: "another-fake-name",
							Tag:     "another-fake-tag",
						},
					},
				},
				RepoURL: "https://github.com/universe/42",
			}},
		}},
	}

	result := (&argocdUpdater{}).buildKustomizeImagesForAppSource(stepCfg.Apps[0].Sources[0].Kustomize)
	require.Equal(
		t,
		argocd.KustomizeImages{
			"yet-another-fake-url@fake-digest",
			"still-another-fake-url:fake-tag",
			"another-fake-url=another-fake-name:another-fake-tag",
		},
		result,
	)
}

func Test_argoCDUpdater_buildHelmParamChangesForAppSource(t *testing.T) {
	stepCfg := &builtin.ArgoCDUpdateConfig{
		Apps: []builtin.ArgoCDAppUpdate{{
			Sources: []builtin.ArgoCDAppSourceUpdate{{
				Helm: &builtin.ArgoCDHelmParameterUpdates{
					Images: []builtin.ArgoCDHelmImageUpdate{
						{
							Key:   "first-fake-key",
							Value: "fake-value",
						},
						{
							Key:   "second-fake-key",
							Value: "another-fake-value",
						},
					},
				},
			}},
		}},
	}

	result := (&argocdUpdater{}).buildHelmParamChangesForAppSource(stepCfg.Apps[0].Sources[0].Helm)
	require.Equal(
		t,
		map[string]string{
			"first-fake-key":  "fake-value",
			"second-fake-key": "another-fake-value",
		},
		result,
	)
}

func Test_argoCDUpdater_recursiveMerge(t *testing.T) {
	testCases := []struct {
		name     string
		src      any
		dst      any
		expected any
	}{
		{
			name: "merge maps",
			src: map[string]any{
				"key1": "value1",
				"key2": map[string]any{
					"subkey1": "subvalue1",
					"subkey2": true,
				},
			},
			dst: map[string]any{
				"key1": "old_value1",
				"key2": map[string]any{
					"subkey2": false,
					"subkey3": "subvalue3",
				},
			},
			expected: map[string]any{
				"key1": "value1",
				"key2": map[string]any{
					"subkey1": "subvalue1",
					"subkey2": true,
					"subkey3": "subvalue3",
				},
			},
		},
		{
			name: "merge arrays",
			src: []any{
				"value1",
				map[string]any{
					"key1": "subvalue1",
				},
				true,
			},
			dst: []any{
				"old_value1",
				map[string]any{
					"key1": "old_subvalue1",
					"key2": "subvalue2",
				},
				false,
			},
			expected: []any{
				"value1",
				map[string]any{
					"key1": "subvalue1",
					"key2": "subvalue2",
				},
				true,
			},
		},
		{
			name:     "merge incompatible types (map to array)",
			src:      map[string]any{"key1": "value1"},
			dst:      []any{"old_value1"},
			expected: map[string]any{"key1": "value1"},
		},
		{
			name:     "merge incompatible types (array to map)",
			src:      []any{"value1"},
			dst:      map[string]any{"key1": "old_value1"},
			expected: []any{"value1"},
		},
		{
			name:     "overwrite types (string to int)",
			src:      "value1",
			dst:      42,
			expected: "value1",
		},
		{
			name:     "overwrite types (int to string)",
			src:      true,
			dst:      "old_value1",
			expected: true,
		},
		{
			name:     "overwrite value with nil",
			src:      nil,
			dst:      map[string]any{"key1": "old_value1"},
			expected: nil,
		},
		{
			name:     "overwrite nil with value",
			src:      map[string]any{"key1": "value1"},
			dst:      nil,
			expected: map[string]any{"key1": "value1"},
		},
	}

	runner := &argocdUpdater{}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := runner.recursiveMerge(tc.src, tc.dst)
			require.Equal(t, tc.expected, result)
		})
	}
}
