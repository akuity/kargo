package directives

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
)

func Test_newArgocdUpdater(t *testing.T) {
	runner := newArgocdUpdater()
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
		config           Config
		expectedProblems []string
	}{
		{
			name:   "apps not specified",
			config: Config{},
			expectedProblems: []string{
				"(root): apps is required",
			},
		},
		{
			name: "apps is empty array",
			config: Config{
				"apps": []Config{},
			},
			expectedProblems: []string{
				"apps: Array must have at least 1 items",
			},
		},
		{
			name: "app name not specified",
			config: Config{
				"apps": []Config{{}},
			},
			expectedProblems: []string{
				"apps.0: name is required",
			},
		},
		{
			name: "app name is empty string",
			config: Config{
				"apps": []Config{{
					"name": "",
				}},
			},
			expectedProblems: []string{
				"apps.0.name: String length must be greater than or equal to 1",
			},
		},
		{
			name: "app namespace is empty string",
			config: Config{
				"apps": []Config{{
					"namespace": "",
				}},
			},
			expectedProblems: []string{
				"apps.0.namespace: String length must be greater than or equal to 1",
			},
		},
		{
			name: "app sources is empty array",
			config: Config{
				"apps": []Config{{
					"sources": []Config{},
				}},
			},
			expectedProblems: []string{
				"apps.0.sources: Array must have at least 1 items",
			},
		},
		{
			name: "source repoURL not specified",
			config: Config{
				"apps": []Config{{
					"sources": []Config{{}},
				}},
			},
			expectedProblems: []string{
				"apps.0.sources.0: repoURL is required",
			},
		},
		{
			name: "source repoURL is empty string",
			config: Config{
				"apps": []Config{{
					"sources": []Config{{
						"repoURL": "",
					}},
				}},
			},
			expectedProblems: []string{
				"apps.0.sources.0.repoURL: String length must be greater than or equal to 1",
			},
		},
		{
			name: "helm images is empty array",
			config: Config{
				"apps": []Config{{
					"sources": []Config{{
						"helm": Config{
							"images": []Config{},
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
			config: Config{
				"apps": []Config{{
					"sources": []Config{{
						"helm": Config{
							"images": []Config{{}},
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
			config: Config{
				"apps": []Config{{
					"sources": []Config{{
						"helm": Config{
							"images": []Config{{
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
			name: "helm images update repoURL is not specified",
			config: Config{
				"apps": []Config{{
					"sources": []Config{{
						"helm": Config{
							"images": []Config{{}},
						},
					}},
				}},
			},
			expectedProblems: []string{
				"apps.0.sources.0.helm.images.0: repoURL is required",
			},
		},
		{
			name: "helm images update repoURL is empty string",
			config: Config{
				"apps": []Config{{
					"sources": []Config{{
						"helm": Config{
							"images": []Config{{
								"repoURL": "",
							}},
						},
					}},
				}},
			},
			expectedProblems: []string{
				"apps.0.sources.0.helm.images.0.repoURL: String length must be greater than or equal to 1",
			},
		},
		{
			name: "helm images update value is not specified",
			config: Config{
				"apps": []Config{{
					"sources": []Config{{
						"helm": Config{
							"images": []Config{{}},
						},
					}},
				}},
			},
			expectedProblems: []string{
				"apps.0.sources.0.helm.images.0: value is required",
			},
		},
		{
			name: "helm images update value is invalid",
			config: Config{
				"apps": []Config{{
					"sources": []Config{{
						"helm": Config{
							"images": []Config{{
								"value": "bogus",
							}},
						},
					}},
				}},
			},
			expectedProblems: []string{
				"apps.0.sources.0.helm.images.0.value must be one of the following",
			},
		},
		{
			name: "kustomize images is empty array",
			config: Config{
				"apps": []Config{{
					"sources": []Config{{
						"kustomize": Config{
							"images": []Config{},
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
			config: Config{
				"apps": []Config{{
					"sources": []Config{{
						"kustomize": Config{
							"images": []Config{{
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
			config: Config{
				"apps": []Config{{
					"sources": []Config{{
						"kustomize": Config{
							"images": []Config{{}},
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
			config: Config{
				"apps": []Config{{
					"sources": []Config{{
						"kustomize": Config{
							"images": []Config{{
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
			name: "valid kitchen sink",
			config: Config{
				"apps": []Config{{
					"name":      "app",
					"namespace": "argocd",
					"sources": []Config{{
						"repoURL":              "fake-git-url",
						"updateTargetRevision": true,
						"helm": Config{
							"images": []Config{{
								"repoURL": "fake-image-url",
								"key":     "fake-key",
								"value":   Tag,
							}},
						},
						"kustomize": Config{
							"images": []Config{{
								"repoURL":   "fake-image-url",
								"newName":   "fake-new-name",
								"useDigest": true,
							}},
						},
					}},
				}},
			},
		},
	}

	runner := newArgocdUpdater()

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

func Test_argoCDUpdater_runPromotionStep(t *testing.T) {
	testCases := []struct {
		name       string
		runner     *argocdUpdater
		stepCtx    *PromotionStepContext
		stepCfg    ArgoCDUpdateConfig
		assertions func(*testing.T, PromotionStepResult, error)
	}{
		{
			name:    "argo cd integration disabled",
			runner:  &argocdUpdater{},
			stepCtx: &PromotionStepContext{},
			stepCfg: ArgoCDUpdateConfig{},
			assertions: func(t *testing.T, res PromotionStepResult, err error) {
				require.Equal(t, PromotionStatusFailure, res.Status)
				require.ErrorContains(
					t, err, "Argo CD integration is disabled on this controller",
				)
			},
		},
		{
			name: "error retrieving authorized application",
			runner: &argocdUpdater{
				getAuthorizedApplicationFn: func(
					context.Context,
					*PromotionStepContext,
					client.ObjectKey,
				) (*v1alpha1.Application, error) {
					return nil, errors.New("something went wrong")
				},
			},
			stepCtx: &PromotionStepContext{
				ArgoCDClient: fake.NewFakeClient(),
			},
			stepCfg: ArgoCDUpdateConfig{
				Apps: []ArgoCDAppUpdate{{}},
			},
			assertions: func(t *testing.T, res PromotionStepResult, err error) {
				require.Equal(t, PromotionStatusFailure, res.Status)
				require.ErrorContains(t, err, "error getting Argo CD Application")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error building desired sources",
			runner: &argocdUpdater{
				getAuthorizedApplicationFn: func(
					context.Context,
					*PromotionStepContext,
					client.ObjectKey,
				) (*v1alpha1.Application, error) {
					return &argocd.Application{}, nil
				},
				buildDesiredSourcesFn: func(
					context.Context,
					*PromotionStepContext,
					*ArgoCDUpdateConfig,
					*ArgoCDAppUpdate,
					[]string,
					*argocd.Application,
				) (argocd.ApplicationSources, error) {
					return nil, errors.New("something went wrong")
				},
			},
			stepCtx: &PromotionStepContext{
				ArgoCDClient: fake.NewFakeClient(),
			},
			stepCfg: ArgoCDUpdateConfig{
				Apps: []ArgoCDAppUpdate{{}},
			},
			assertions: func(t *testing.T, res PromotionStepResult, err error) {
				require.Equal(t, PromotionStatusFailure, res.Status)
				require.ErrorContains(t, err, "error building desired sources for Argo CD Application")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error determining if update is necessary",
			runner: &argocdUpdater{
				getAuthorizedApplicationFn: func(
					context.Context,
					*PromotionStepContext,
					client.ObjectKey,
				) (*v1alpha1.Application, error) {
					return &argocd.Application{}, nil
				},
				buildDesiredSourcesFn: func(
					context.Context,
					*PromotionStepContext,
					*ArgoCDUpdateConfig,
					*ArgoCDAppUpdate,
					[]string,
					*argocd.Application,
				) (argocd.ApplicationSources, error) {
					return []argocd.ApplicationSource{{}}, nil
				},
				mustPerformUpdateFn: func(
					context.Context,
					*PromotionStepContext,
					*ArgoCDUpdateConfig,
					*ArgoCDAppUpdate,
					*argocd.Application,
					argocd.ApplicationSources,
				) (argocd.OperationPhase, bool, error) {
					return "", false, errors.New("something went wrong")
				},
			},
			stepCtx: &PromotionStepContext{
				ArgoCDClient: fake.NewFakeClient(),
			},
			stepCfg: ArgoCDUpdateConfig{
				Apps: []ArgoCDAppUpdate{{}},
			},
			assertions: func(t *testing.T, res PromotionStepResult, err error) {
				require.Equal(t, PromotionStatusFailure, res.Status)
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "determination error can be solved by applying update",
			runner: &argocdUpdater{
				getAuthorizedApplicationFn: func(
					context.Context,
					*PromotionStepContext,
					client.ObjectKey,
				) (*v1alpha1.Application, error) {
					return &argocd.Application{}, nil
				},
				buildDesiredSourcesFn: func(
					context.Context,
					*PromotionStepContext,
					*ArgoCDUpdateConfig,
					*ArgoCDAppUpdate,
					[]string,
					*argocd.Application,
				) (argocd.ApplicationSources, error) {
					return []argocd.ApplicationSource{{}}, nil
				},
				mustPerformUpdateFn: func(
					context.Context,
					*PromotionStepContext,
					*ArgoCDUpdateConfig,
					*ArgoCDAppUpdate,
					*argocd.Application,
					argocd.ApplicationSources,
				) (argocd.OperationPhase, bool, error) {
					return "", true, errors.New("something went wrong")
				},
				syncApplicationFn: func(
					context.Context,
					*PromotionStepContext,
					*argocd.Application,
					argocd.ApplicationSources,
				) error {
					return nil
				},
			},
			stepCtx: &PromotionStepContext{
				ArgoCDClient: fake.NewFakeClient(),
			},
			stepCfg: ArgoCDUpdateConfig{
				Apps: []ArgoCDAppUpdate{{}},
			},
			assertions: func(t *testing.T, res PromotionStepResult, err error) {
				require.Equal(t, PromotionStatusPending, res.Status)
				require.NoError(t, err)
			},
		},
		{
			name: "must wait for update to complete",
			runner: &argocdUpdater{
				getAuthorizedApplicationFn: func(
					context.Context,
					*PromotionStepContext,
					client.ObjectKey,
				) (*v1alpha1.Application, error) {
					return &argocd.Application{}, nil
				},
				buildDesiredSourcesFn: func(
					context.Context,
					*PromotionStepContext,
					*ArgoCDUpdateConfig,
					*ArgoCDAppUpdate,
					[]string,
					*argocd.Application,
				) (argocd.ApplicationSources, error) {
					return []argocd.ApplicationSource{{}}, nil
				},
				mustPerformUpdateFn: func(
					context.Context,
					*PromotionStepContext,
					*ArgoCDUpdateConfig,
					*ArgoCDAppUpdate,
					*argocd.Application,
					argocd.ApplicationSources,
				) (argocd.OperationPhase, bool, error) {
					return argocd.OperationRunning, false, nil
				},
			},
			stepCtx: &PromotionStepContext{
				ArgoCDClient: fake.NewFakeClient(),
			},
			stepCfg: ArgoCDUpdateConfig{
				Apps: []ArgoCDAppUpdate{{}},
			},
			assertions: func(t *testing.T, res PromotionStepResult, err error) {
				require.Equal(t, PromotionStatusPending, res.Status)
				require.NoError(t, err)
			},
		},
		{
			name: "must wait for operation from different user to complete",
			runner: &argocdUpdater{
				getAuthorizedApplicationFn: func(
					context.Context,
					*PromotionStepContext,
					client.ObjectKey,
				) (*v1alpha1.Application, error) {
					return &argocd.Application{}, nil
				},
				buildDesiredSourcesFn: func(
					context.Context,
					*PromotionStepContext,
					*ArgoCDUpdateConfig,
					*ArgoCDAppUpdate,
					[]string,
					*argocd.Application,
				) (argocd.ApplicationSources, error) {
					return []argocd.ApplicationSource{{}}, nil
				},
				mustPerformUpdateFn: func(
					context.Context,
					*PromotionStepContext,
					*ArgoCDUpdateConfig,
					*ArgoCDAppUpdate,
					*argocd.Application,
					argocd.ApplicationSources,
				) (argocd.OperationPhase, bool, error) {
					return argocd.OperationRunning, false, fmt.Errorf("waiting for operation to complete")
				},
			},
			stepCtx: &PromotionStepContext{
				ArgoCDClient: fake.NewFakeClient(),
			},
			stepCfg: ArgoCDUpdateConfig{
				Apps: []ArgoCDAppUpdate{{}},
			},
			assertions: func(t *testing.T, res PromotionStepResult, err error) {
				require.Equal(t, PromotionStatusPending, res.Status)
				require.NoError(t, err)
			},
		},
		{
			name: "error applying update",
			runner: &argocdUpdater{
				getAuthorizedApplicationFn: func(
					context.Context,
					*PromotionStepContext,
					client.ObjectKey,
				) (*v1alpha1.Application, error) {
					return &argocd.Application{}, nil
				},
				buildDesiredSourcesFn: func(
					context.Context,
					*PromotionStepContext,
					*ArgoCDUpdateConfig,
					*ArgoCDAppUpdate,
					[]string,
					*argocd.Application,
				) (argocd.ApplicationSources, error) {
					return []argocd.ApplicationSource{{}}, nil
				},
				mustPerformUpdateFn: func(
					context.Context,
					*PromotionStepContext,
					*ArgoCDUpdateConfig,
					*ArgoCDAppUpdate,
					*argocd.Application,
					argocd.ApplicationSources,
				) (argocd.OperationPhase, bool, error) {
					return "", true, nil
				},
				syncApplicationFn: func(
					context.Context,
					*PromotionStepContext,
					*argocd.Application,
					argocd.ApplicationSources,
				) error {
					return errors.New("something went wrong")
				},
			},
			stepCtx: &PromotionStepContext{
				ArgoCDClient: fake.NewFakeClient(),
			},
			stepCfg: ArgoCDUpdateConfig{
				Apps: []ArgoCDAppUpdate{{}},
			},
			assertions: func(t *testing.T, res PromotionStepResult, err error) {
				require.Equal(t, PromotionStatusFailure, res.Status)
				require.ErrorContains(t, err, "error syncing Argo CD Application")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "failed and pending update",
			runner: &argocdUpdater{
				getAuthorizedApplicationFn: func(
					context.Context,
					*PromotionStepContext,
					client.ObjectKey,
				) (*v1alpha1.Application, error) {
					return &argocd.Application{}, nil
				},
				buildDesiredSourcesFn: func(
					context.Context,
					*PromotionStepContext,
					*ArgoCDUpdateConfig,
					*ArgoCDAppUpdate,
					[]string,
					*argocd.Application,
				) (argocd.ApplicationSources, error) {
					return []argocd.ApplicationSource{{}}, nil
				},
				mustPerformUpdateFn: func() func(
					context.Context,
					*PromotionStepContext,
					*ArgoCDUpdateConfig,
					*ArgoCDAppUpdate,
					*argocd.Application,
					argocd.ApplicationSources,
				) (argocd.OperationPhase, bool, error) {
					var count uint
					return func(
						context.Context,
						*PromotionStepContext,
						*ArgoCDUpdateConfig,
						*ArgoCDAppUpdate,
						*argocd.Application,
						argocd.ApplicationSources,
					) (argocd.OperationPhase, bool, error) {
						count++
						if count > 1 {
							return argocd.OperationFailed, false, nil
						}
						return "", true, nil
					}
				}(),
				syncApplicationFn: func(
					context.Context,
					*PromotionStepContext,
					*argocd.Application,
					argocd.ApplicationSources,
				) error {
					return nil
				},
			},
			stepCtx: &PromotionStepContext{
				ArgoCDClient: fake.NewFakeClient(),
			},
			stepCfg: ArgoCDUpdateConfig{
				Apps: []ArgoCDAppUpdate{
					{},
					{},
				},
			},
			assertions: func(t *testing.T, res PromotionStepResult, err error) {
				require.Equal(t, PromotionStatusFailure, res.Status)
				require.NoError(t, err)
			},
		},
		{
			name: "operation phase aggregation error",
			runner: &argocdUpdater{
				getAuthorizedApplicationFn: func(
					context.Context,
					*PromotionStepContext,
					client.ObjectKey,
				) (*v1alpha1.Application, error) {
					return &argocd.Application{}, nil
				},
				buildDesiredSourcesFn: func(
					context.Context,
					*PromotionStepContext,
					*ArgoCDUpdateConfig,
					*ArgoCDAppUpdate,
					[]string,
					*argocd.Application,
				) (argocd.ApplicationSources, error) {
					return []argocd.ApplicationSource{{}}, nil
				},
				mustPerformUpdateFn: func(
					context.Context,
					*PromotionStepContext,
					*ArgoCDUpdateConfig,
					*ArgoCDAppUpdate,
					*argocd.Application,
					argocd.ApplicationSources,
				) (argocd.OperationPhase, bool, error) {
					return "Unknown", false, nil
				},
			},
			stepCtx: &PromotionStepContext{
				ArgoCDClient: fake.NewFakeClient(),
			},
			stepCfg: ArgoCDUpdateConfig{
				Apps: []ArgoCDAppUpdate{{}},
			},
			assertions: func(t *testing.T, res PromotionStepResult, err error) {
				require.Equal(t, PromotionStatusFailure, res.Status)
				require.ErrorContains(t, err, "could not determine promotion step status")
			},
		},
		{
			name: "completed",
			runner: &argocdUpdater{
				getAuthorizedApplicationFn: func(
					context.Context,
					*PromotionStepContext,
					client.ObjectKey,
				) (*v1alpha1.Application, error) {
					return &argocd.Application{}, nil
				},
				buildDesiredSourcesFn: func(
					context.Context,
					*PromotionStepContext,
					*ArgoCDUpdateConfig,
					*ArgoCDAppUpdate,
					[]string,
					*argocd.Application,
				) (argocd.ApplicationSources, error) {
					return []argocd.ApplicationSource{{}}, nil
				},
				mustPerformUpdateFn: func(
					context.Context,
					*PromotionStepContext,
					*ArgoCDUpdateConfig,
					*ArgoCDAppUpdate,
					*argocd.Application,
					argocd.ApplicationSources,
				) (argocd.OperationPhase, bool, error) {
					return argocd.OperationSucceeded, false, nil
				},
			},
			stepCtx: &PromotionStepContext{
				ArgoCDClient: fake.NewFakeClient(),
			},
			stepCfg: ArgoCDUpdateConfig{
				Apps: []ArgoCDAppUpdate{{}},
			},
			assertions: func(t *testing.T, res PromotionStepResult, err error) {
				require.Equal(t, PromotionStatusSuccess, res.Status)
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			res, err := testCase.runner.runPromotionStep(
				context.Background(),
				testCase.stepCtx,
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
		update           *ArgoCDAppUpdate
		desiredRevisions []string
		app              *argocd.Application
		assertions       func(
			t *testing.T,
			desiredSources argocd.ApplicationSources,
			err error,
		)
	}{
		{
			name: "error applying update to source",
			runner: &argocdUpdater{
				applyArgoCDSourceUpdateFn: func(
					context.Context,
					*PromotionStepContext,
					*ArgoCDUpdateConfig,
					*ArgoCDAppSourceUpdate,
					string,
					argocd.ApplicationSource,
				) (argocd.ApplicationSource, error) {
					return argocd.ApplicationSource{}, errors.New("something went wrong")
				},
			},
			update: &ArgoCDAppUpdate{
				Sources: []ArgoCDAppSourceUpdate{{}},
			},
			desiredRevisions: []string{"fake-version"},
			app: &argocd.Application{
				Spec: argocd.ApplicationSpec{
					Source: &argocd.ApplicationSource{},
				},
			},
			assertions: func(
				t *testing.T,
				desiredSources argocd.ApplicationSources,
				err error,
			) {
				require.ErrorContains(t, err, "something went wrong")
				require.Nil(t, desiredSources)
			},
		},
		{
			name: "applies updates to sources",
			runner: &argocdUpdater{
				applyArgoCDSourceUpdateFn: func(
					_ context.Context,
					_ *PromotionStepContext,
					_ *ArgoCDUpdateConfig,
					_ *ArgoCDAppSourceUpdate,
					_ string,
					src argocd.ApplicationSource,
				) (argocd.ApplicationSource, error) {
					if src.RepoURL == "fake-chart-url" && src.Chart == "fake-chart" {
						src.TargetRevision = "fake-version"
						return src, nil
					}
					if src.RepoURL == "fake-git-url" && src.Chart == "" {
						src.TargetRevision = "fake-commit"
						return src, nil
					}
					return src, nil
				},
			},
			update: &ArgoCDAppUpdate{
				Sources: []ArgoCDAppSourceUpdate{{}, {}},
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
				context.Background(),
				&PromotionStepContext{},
				&ArgoCDUpdateConfig{},
				testCase.update,
				testCase.desiredRevisions,
				testCase.app,
			)
			testCase.assertions(t, desiredSources, err)
		})
	}
}

func Test_argoCDUpdater_mustPerformUpdate(t *testing.T) {
	testFreightCollectionID := "fake-freight-collection"
	testOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse",
	}
	testCases := []struct {
		name              string
		modifyApplication func(*argocd.Application)
		newFreight        []kargoapi.FreightReference
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
			name: "running operation initiated for incorrect freight collection",
			modifyApplication: func(app *argocd.Application) {
				app.Status.OperationState = &argocd.OperationState{
					Phase: argocd.OperationRunning,
					Operation: argocd.Operation{
						InitiatedBy: argocd.OperationInitiator{
							Username: applicationOperationInitiator,
						},
						Info: []*argocd.Info{{
							Name:  freightCollectionInfoKey,
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
			name: "completed operation initiated for incorrect freight collection",
			modifyApplication: func(app *argocd.Application) {
				app.Status.OperationState = &argocd.OperationState{
					Phase: argocd.OperationSucceeded,
					Operation: argocd.Operation{
						InitiatedBy: argocd.OperationInitiator{
							Username: applicationOperationInitiator,
						},
						Info: []*argocd.Info{{
							Name:  freightCollectionInfoKey,
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
							Name:  freightCollectionInfoKey,
							Value: testFreightCollectionID,
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
							Name:  freightCollectionInfoKey,
							Value: testFreightCollectionID,
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
							Name:  freightCollectionInfoKey,
							Value: testFreightCollectionID,
						}},
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
						Info: []*argocd.Info{{
							Name:  freightCollectionInfoKey,
							Value: testFreightCollectionID,
						}},
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
				require.ErrorContains(t, err, "sync result revisions")
				require.ErrorContains(t, err, "do not match desired revisions")
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
						Info: []*argocd.Info{{
							Name:  freightCollectionInfoKey,
							Value: testFreightCollectionID,
						}},
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
						Info: []*argocd.Info{{
							Name:  freightCollectionInfoKey,
							Value: testFreightCollectionID,
						}},
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

			freight := kargoapi.FreightCollection{}
			for _, ref := range testCase.newFreight {
				freight.UpdateOrPush(ref)
			}
			// Tamper with the freight collection ID for testing purposes
			freight.ID = testFreightCollectionID

			stepCfg := &ArgoCDUpdateConfig{
				Apps: []ArgoCDAppUpdate{{
					Sources: []ArgoCDAppSourceUpdate{{
						FromOrigin: &AppFromOrigin{
							Kind: Kind(testOrigin.Kind),
							Name: testOrigin.Name,
						},
						RepoURL: "https://github.com/universe/42",
					}},
				}},
			}

			phase, mustUpdate, err := runner.mustPerformUpdate(
				context.Background(),
				&PromotionStepContext{
					Freight: freight,
				},
				stepCfg,
				&stepCfg.Apps[0],
				app,
				testCase.desiredSources,
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
					*PromotionStepContext,
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
					*PromotionStepContext,
					kubeclient.ObjectWithKind,
					kubeclient.UnstructuredPatchFn,
				) error {
					return nil
				},
				logAppEventFn: func(
					context.Context,
					*PromotionStepContext,
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

	stepCtx := &PromotionStepContext{
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

	runner := &argocdUpdater{}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			c := fake.NewFakeClient()
			runner.logAppEvent(
				context.Background(),
				&PromotionStepContext{
					ArgoCDClient: c,
				},
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

	runner := &argocdUpdater{}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithInterceptorFuncs(testCase.interceptor)

			if testCase.app != nil {
				c.WithObjects(testCase.app)
			}

			app, err := runner.getAuthorizedApplication(
				context.Background(),
				&PromotionStepContext{
					Project:      "fake-namespace",
					Stage:        "fake-stage",
					ArgoCDClient: c.Build(),
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
				&PromotionStepContext{
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
	testOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse",
	}
	testCases := []struct {
		name            string
		source          argocd.ApplicationSource
		freight         []kargoapi.FreightReference
		update          ArgoCDAppSourceUpdate
		desiredRevision string
		assertions      func(
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
			update: ArgoCDAppSourceUpdate{
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
			update: ArgoCDAppSourceUpdate{
				RepoURL:              "fake-url",
				UpdateTargetRevision: true,
			},
			desiredRevision: "fake-commit",
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
			name: "update target revision (helm chart)",
			source: argocd.ApplicationSource{
				RepoURL: "fake-url",
				Chart:   "fake-chart",
			},
			update: ArgoCDAppSourceUpdate{
				RepoURL:              "fake-url",
				Chart:                "fake-chart",
				UpdateTargetRevision: true,
			},
			desiredRevision: "fake-version",
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
				Images: []kargoapi.Image{{
					RepoURL: "fake-image-url",
					Tag:     "fake-tag",
				}},
			}},
			update: ArgoCDAppSourceUpdate{
				RepoURL: "fake-url",
				Kustomize: &ArgoCDKustomizeImageUpdates{
					Images: []ArgoCDKustomizeImageUpdate{{
						RepoURL: "fake-image-url",
					}},
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
						argocd.KustomizeImage("fake-image-url:fake-tag"),
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
				Images: []kargoapi.Image{{
					RepoURL: "fake-image-url",
					Tag:     "fake-tag",
				}},
			}},
			update: ArgoCDAppSourceUpdate{
				RepoURL: "fake-url",
				Helm: &ArgoCDHelmParameterUpdates{
					Images: []ArgoCDHelmImageUpdate{
						{
							RepoURL: "fake-image-url",
							Key:     "image",
							Value:   ImageAndTag,
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

	runner := &argocdUpdater{}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			freight := kargoapi.FreightCollection{}
			for _, ref := range testCase.freight {
				freight.UpdateOrPush(ref)
			}
			stepCfg := &ArgoCDUpdateConfig{
				Apps: []ArgoCDAppUpdate{{
					FromOrigin: &AppFromOrigin{
						Kind: Kind(testOrigin.Kind),
						Name: testOrigin.Name,
					},
					Sources: []ArgoCDAppSourceUpdate{testCase.update},
				}},
			}
			updatedSource, err := runner.applyArgoCDSourceUpdate(
				context.Background(),
				&PromotionStepContext{
					Freight: freight,
				},
				stepCfg,
				&stepCfg.Apps[0].Sources[0],
				testCase.desiredRevision,
				testCase.source,
			)
			testCase.assertions(t, testCase.source, updatedSource, err)
		})
	}
}

func Test_argoCDUpdater_buildKustomizeImagesForAppSource(t *testing.T) {
	testOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse",
	}

	freight := kargoapi.FreightCollection{}
	freight.UpdateOrPush(kargoapi.FreightReference{
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
	})

	stepCfg := &ArgoCDUpdateConfig{
		Apps: []ArgoCDAppUpdate{{
			Sources: []ArgoCDAppSourceUpdate{{
				Kustomize: &ArgoCDKustomizeImageUpdates{
					FromOrigin: &AppFromOrigin{
						Kind: Kind(testOrigin.Kind),
						Name: testOrigin.Name,
					},
					Images: []ArgoCDKustomizeImageUpdate{
						{RepoURL: "fake-url"},
						{
							RepoURL:   "another-fake-url",
							UseDigest: true,
						},
						{RepoURL: "image-that-is-not-in-list"},
					},
				},
				RepoURL: "https://github.com/universe/42",
			}},
		}},
	}

	result, err := (&argocdUpdater{}).
		buildKustomizeImagesForAppSource(
			context.Background(),
			&PromotionStepContext{
				Freight: freight,
			},
			stepCfg,
			stepCfg.Apps[0].Sources[0].Kustomize,
		)
	require.NoError(t, err)
	require.Equal(
		t,
		argocd.KustomizeImages{
			"fake-url:fake-tag",
			"another-fake-url@another-fake-digest",
		},
		result,
	)
}

func Test_argoCDUpdater_buildHelmParamChangesForAppSource(t *testing.T) {
	testOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse",
	}

	freight := kargoapi.FreightCollection{}
	freight.UpdateOrPush(kargoapi.FreightReference{
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
	})

	stepCfg := &ArgoCDUpdateConfig{
		Apps: []ArgoCDAppUpdate{{
			Sources: []ArgoCDAppSourceUpdate{{
				Helm: &ArgoCDHelmParameterUpdates{
					FromOrigin: &AppFromOrigin{
						Kind: Kind(testOrigin.Kind),
						Name: testOrigin.Name,
					},
					Images: []ArgoCDHelmImageUpdate{
						{
							RepoURL: "fake-url",
							Key:     "fake-key",
							Value:   ImageAndTag,
						},
						{
							RepoURL: "second-fake-url",
							Key:     "second-fake-key",
							Value:   Tag,
						},
						{
							RepoURL: "third-fake-url",
							Key:     "third-fake-key",
							Value:   ImageAndDigest,
						},
						{
							RepoURL: "fourth-fake-url",
							Key:     "fourth-fake-key",
							Value:   Digest,
						},
						{
							RepoURL: "image-that-is-not-in-list",
							Key:     "fake-key",
							Value:   Tag,
						},
					},
				},
			}},
		}},
	}

	result, err := (&argocdUpdater{}).
		buildHelmParamChangesForAppSource(
			context.Background(),
			&PromotionStepContext{
				Freight: freight,
			},
			stepCfg,
			stepCfg.Apps[0].Sources[0].Helm,
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
