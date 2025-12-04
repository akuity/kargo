package builtin

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	argocd "github.com/akuity/kargo/pkg/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/pkg/kubeclient"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_newArgocdUpdater(t *testing.T) {
	r := newArgocdUpdater(promotion.StepRunnerCapabilities{
		ArgoCDClient: fake.NewFakeClient(),
	})
	runner, ok := r.(*argocdUpdater)
	require.True(t, ok)
	assert.NotNil(t, runner.argocdClient)
	assert.NotNil(t, runner.schemaLoader)
	assert.NotNil(t, runner.getAuthorizedApplicationsFn)
	assert.NotNil(t, runner.buildLabelSelectorFn)
	assert.NotNil(t, runner.buildDesiredSourcesFn)
	assert.NotNil(t, runner.mustPerformUpdateFn)
	assert.NotNil(t, runner.syncApplicationFn)
	assert.NotNil(t, runner.applyArgoCDSourceUpdateFn)
	assert.NotNil(t, runner.argoCDAppPatchFn)
	assert.NotNil(t, runner.logAppEventFn)
}

func Test_argoCDUpdater_convert(t *testing.T) {
	tests := []validationTestCase{
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
			name: "app name and selector not specified",
			config: promotion.Config{
				"apps": []promotion.Config{{}},
			},
			expectedProblems: []string{
				"apps.0: Must validate one and only one schema (oneOf)",
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
			name: "app name and selector both specified",
			config: promotion.Config{
				"apps": []promotion.Config{{
					"name": "my-app",
					"selector": promotion.Config{
						"matchLabels": promotion.Config{
							"env": "prod",
						},
					},
				}},
			},
			expectedProblems: []string{
				"apps.0: Must validate one and only one schema (oneOf)",
			},
		},
		{
			name: "app selector is empty",
			config: promotion.Config{
				"apps": []promotion.Config{{
					"selector": promotion.Config{},
				}},
			},
			expectedProblems: []string{
				"apps.0.selector: Must validate at least one schema (anyOf)",
			},
		},
		{
			name: "app selector matchLabels is empty object",
			config: promotion.Config{
				"apps": []promotion.Config{{
					"selector": promotion.Config{
						"matchLabels": promotion.Config{},
					},
				}},
			},
			expectedProblems: []string{
				"apps.0.selector.matchLabels: Must have at least 1 properties",
			},
		},
		{
			name: "app selector matchExpressions is empty array",
			config: promotion.Config{
				"apps": []promotion.Config{{
					"selector": promotion.Config{
						"matchExpressions": []promotion.Config{},
					},
				}},
			},
			expectedProblems: []string{
				"apps.0.selector.matchExpressions: Array must have at least 1 items",
			},
		},
		{
			name: "app selector matchExpression missing key",
			config: promotion.Config{
				"apps": []promotion.Config{{
					"selector": promotion.Config{
						"matchExpressions": []promotion.Config{{
							"operator": "In",
						}},
					},
				}},
			},
			expectedProblems: []string{
				"apps.0.selector.matchExpressions.0: key is required",
			},
		},
		{
			name: "app selector matchExpression missing operator",
			config: promotion.Config{
				"apps": []promotion.Config{{
					"selector": promotion.Config{
						"matchExpressions": []promotion.Config{{
							"key": "env",
						}},
					},
				}},
			},
			expectedProblems: []string{
				"apps.0.selector.matchExpressions.0: operator is required",
			},
		},
		{
			name: "app selector matchExpression key is empty string",
			config: promotion.Config{
				"apps": []promotion.Config{{
					"selector": promotion.Config{
						"matchExpressions": []promotion.Config{{
							"key":      "",
							"operator": "In",
						}},
					},
				}},
			},
			expectedProblems: []string{
				"apps.0.selector.matchExpressions.0.key: String length must be greater than or equal to 1",
			},
		},
		{
			name: "app selector matchExpression operator is empty string",
			config: promotion.Config{
				"apps": []promotion.Config{{
					"selector": promotion.Config{
						"matchExpressions": []promotion.Config{{
							"key":      "env",
							"operator": "",
						}},
					},
				}},
			},
			expectedProblems: []string{
				"apps.0.selector.matchExpressions.0.operator must be one of the following",
			},
		},
		{
			name: "app selector matchExpression operator is invalid",
			config: promotion.Config{
				"apps": []promotion.Config{{
					"selector": promotion.Config{
						"matchExpressions": []promotion.Config{{
							"key":      "env",
							"operator": "InvalidOperator",
						}},
					},
				}},
			},
			expectedProblems: []string{
				"apps.0.selector.matchExpressions.0.operator must be one of the following",
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
			name: "valid selector with matchLabels",
			config: promotion.Config{
				"apps": []promotion.Config{{
					"selector": promotion.Config{
						"matchLabels": promotion.Config{
							"env":  "prod",
							"team": "platform",
						},
					},
				}},
			},
		},
		{
			name: "valid selector with matchExpressions",
			config: promotion.Config{
				"apps": []promotion.Config{{
					"selector": promotion.Config{
						"matchExpressions": []promotion.Config{{
							"key":      "env",
							"operator": "In",
							"values":   []string{"prod"},
						}},
					},
				}},
			},
		},
		{
			name: "valid selector with both matchLabels and matchExpressions",
			config: promotion.Config{
				"apps": []promotion.Config{{
					"selector": promotion.Config{
						"matchLabels": promotion.Config{
							"team": "platform",
						},
						"matchExpressions": []promotion.Config{{
							"key":      "env",
							"operator": "In",
							"values":   []string{"prod", "staging"},
						}},
					},
				}},
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

	r := newArgocdUpdater(promotion.StepRunnerCapabilities{})
	runner, ok := r.(*argocdUpdater)
	require.True(t, ok)
	runValidationTests(t, runner.convert, tests)
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
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, res.Status)
				assert.Nil(t, res.RetryAfter)
				require.ErrorContains(
					t, err, "Argo CD integration is disabled on this controller",
				)
			},
		},
		{
			name: "error retrieving authorized application",
			runner: &argocdUpdater{
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationsFn: func(
					context.Context,
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
				) ([]*argocd.Application, error) {
					return nil, errors.New("something went wrong")
				},
			},
			stepCfg: builtin.ArgoCDUpdateConfig{
				Apps: []builtin.ArgoCDAppUpdate{{}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, res.Status)
				assert.Nil(t, res.RetryAfter)
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error determining if update is necessary",
			runner: &argocdUpdater{
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationsFn: func(
					context.Context,
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
				) ([]*argocd.Application, error) {
					return []*argocd.Application{{}}, nil
				},
				buildDesiredSourcesFn: func(
					*builtin.ArgoCDAppUpdate,
					[]string,
					*argocd.Application,
				) (argocd.ApplicationSources, error) {
					return []argocd.ApplicationSource{{}}, nil
				},
				mustPerformUpdateFn: func(
					context.Context,
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
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, res.Status)
				assert.Nil(t, res.RetryAfter)
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "determination error can be solved by applying update",
			runner: &argocdUpdater{
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationsFn: func(
					context.Context,
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
				) ([]*argocd.Application, error) {
					return []*argocd.Application{{}}, nil
				},
				buildDesiredSourcesFn: func(
					*builtin.ArgoCDAppUpdate,
					[]string,
					*argocd.Application,
				) (argocd.ApplicationSources, error) {
					return []argocd.ApplicationSource{{}}, nil
				},
				mustPerformUpdateFn: func(
					context.Context,
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
				assert.Equal(t, kargoapi.PromotionStepStatusRunning, res.Status)
				assert.NotNil(t, res.RetryAfter)
				require.NoError(t, err)
			},
		},
		{
			name: "must wait for update to complete",
			runner: &argocdUpdater{
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationsFn: func(
					context.Context,
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
				) ([]*argocd.Application, error) {
					return []*argocd.Application{{}}, nil
				},
				buildDesiredSourcesFn: func(
					*builtin.ArgoCDAppUpdate,
					[]string,
					*argocd.Application,
				) (argocd.ApplicationSources, error) {
					return []argocd.ApplicationSource{{}}, nil
				},
				mustPerformUpdateFn: func(
					context.Context,
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
				assert.Equal(t, kargoapi.PromotionStepStatusRunning, res.Status)
				assert.NotNil(t, res.RetryAfter)
				require.NoError(t, err)
			},
		},
		{
			name: "must wait for operation from different user to complete",
			runner: &argocdUpdater{
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationsFn: func(
					context.Context,
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
				) ([]*argocd.Application, error) {
					return []*argocd.Application{{}}, nil
				},
				buildDesiredSourcesFn: func(
					*builtin.ArgoCDAppUpdate,
					[]string,
					*argocd.Application,
				) (argocd.ApplicationSources, error) {
					return []argocd.ApplicationSource{{}}, nil
				},
				mustPerformUpdateFn: func(
					context.Context,
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
				assert.Equal(t, kargoapi.PromotionStepStatusRunning, res.Status)
				assert.NotNil(t, res.RetryAfter)
				require.NoError(t, err)
			},
		},
		{
			name: "error building desired sources",
			runner: &argocdUpdater{
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationsFn: func(
					context.Context,
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
				) ([]*argocd.Application, error) {
					return []*argocd.Application{{}}, nil
				},
				mustPerformUpdateFn: func(
					context.Context,
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
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, res.Status)
				assert.Nil(t, res.RetryAfter)
				// Single app - error caught during processing, not validation
				require.ErrorContains(t, err, "error building desired sources for Argo CD Application")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error applying update",
			runner: &argocdUpdater{
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationsFn: func(
					context.Context,
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
				) ([]*argocd.Application, error) {
					return []*argocd.Application{{}}, nil
				},
				mustPerformUpdateFn: func(
					context.Context,
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
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, res.Status)
				assert.Nil(t, res.RetryAfter)
				require.ErrorContains(t, err, "error syncing Argo CD Application")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "failed and pending update",
			runner: &argocdUpdater{
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationsFn: func(
					context.Context,
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
				) ([]*argocd.Application, error) {
					return []*argocd.Application{{}}, nil
				},
				mustPerformUpdateFn: func() func(
					context.Context,
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
					*argocd.Application,
				) (argocd.OperationPhase, bool, error) {
					var count uint
					return func(
						context.Context,
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
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, res.Status)
				assert.Nil(t, res.RetryAfter)
				require.NoError(t, err)
			},
		},
		{
			name: "operation phase aggregation error",
			runner: &argocdUpdater{
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationsFn: func(
					context.Context,
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
				) ([]*argocd.Application, error) {
					return []*argocd.Application{{}}, nil
				},
				buildDesiredSourcesFn: func(
					*builtin.ArgoCDAppUpdate,
					[]string,
					*argocd.Application,
				) (argocd.ApplicationSources, error) {
					return []argocd.ApplicationSource{{}}, nil
				},
				mustPerformUpdateFn: func(
					context.Context,
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
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, res.Status)
				assert.Nil(t, res.RetryAfter)
				require.ErrorContains(t, err, "could not determine promotion step status")
			},
		},
		{
			name: "completed",
			runner: &argocdUpdater{
				getAuthorizedApplicationsFn: func(
					context.Context,
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
				) ([]*argocd.Application, error) {
					return []*argocd.Application{{}}, nil
				},
				buildDesiredSourcesFn: func(
					*builtin.ArgoCDAppUpdate,
					[]string,
					*argocd.Application,
				) (argocd.ApplicationSources, error) {
					return []argocd.ApplicationSource{{}}, nil
				},
				mustPerformUpdateFn: func(
					context.Context,
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
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
				assert.Nil(t, res.RetryAfter)
				require.NoError(t, err)
			},
		},
		{
			name: "selector returns multiple apps - all succeed",
			runner: &argocdUpdater{
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationsFn: func(
					context.Context,
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
				) ([]*argocd.Application, error) {
					return []*argocd.Application{
						{ObjectMeta: metav1.ObjectMeta{Name: "app1", Namespace: "argocd"}},
						{ObjectMeta: metav1.ObjectMeta{Name: "app2", Namespace: "argocd"}},
						{ObjectMeta: metav1.ObjectMeta{Name: "app3", Namespace: "argocd"}},
					}, nil
				},
				buildDesiredSourcesFn: func(
					*builtin.ArgoCDAppUpdate,
					[]string,
					*argocd.Application,
				) (argocd.ApplicationSources, error) {
					return []argocd.ApplicationSource{{}}, nil
				},
				mustPerformUpdateFn: func(
					context.Context,
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
					*argocd.Application,
				) (argocd.OperationPhase, bool, error) {
					return argocd.OperationSucceeded, false, nil
				},
			},
			stepCfg: builtin.ArgoCDUpdateConfig{
				Apps: []builtin.ArgoCDAppUpdate{{}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
				assert.Nil(t, res.RetryAfter)
				require.NoError(t, err)
				// Verify health checks include all 3 apps
				require.NotNil(t, res.HealthCheck)
				apps, ok := res.HealthCheck.Input["apps"]
				assert.True(t, ok)
				assert.Len(t, apps, 3)
			},
		},
		{
			name: "selector returns multiple apps - one fails during sync",
			runner: &argocdUpdater{
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationsFn: func(
					context.Context,
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
				) ([]*argocd.Application, error) {
					return []*argocd.Application{
						{ObjectMeta: metav1.ObjectMeta{Name: "app1", Namespace: "argocd"}},
						{ObjectMeta: metav1.ObjectMeta{Name: "app2", Namespace: "argocd"}},
						{ObjectMeta: metav1.ObjectMeta{Name: "app3", Namespace: "argocd"}},
					}, nil
				},
				mustPerformUpdateFn: func() func(
					context.Context,
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
					*argocd.Application,
				) (argocd.OperationPhase, bool, error) {
					var count int
					return func(
						_ context.Context,
						_ *promotion.StepContext,
						_ *builtin.ArgoCDAppUpdate,
						_ *argocd.Application,
					) (argocd.OperationPhase, bool, error) {
						count++
						if count == 1 {
							// App1: already synced, no update needed
							return argocd.OperationSucceeded, false, nil
						}
						// App2: needs update
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
					_ context.Context,
					_ *promotion.StepContext,
					app *argocd.Application,
					_ argocd.ApplicationSources,
				) error {
					if app.Name == "app2" {
						return errors.New("sync failed for app2")
					}
					return nil
				},
			},
			stepCfg: builtin.ArgoCDUpdateConfig{
				Apps: []builtin.ArgoCDAppUpdate{{}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, res.Status)
				assert.Nil(t, res.RetryAfter)
				require.ErrorContains(t, err, "error syncing Argo CD Application")
				require.ErrorContains(t, err, "app2")
				require.ErrorContains(t, err, "sync failed for app2")
			},
		},
		{
			name: "selector returns multiple apps - mixed phases",
			runner: &argocdUpdater{
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationsFn: func(
					context.Context,
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
				) ([]*argocd.Application, error) {
					return []*argocd.Application{
						{ObjectMeta: metav1.ObjectMeta{Name: "app1", Namespace: "argocd"}},
						{ObjectMeta: metav1.ObjectMeta{Name: "app2", Namespace: "argocd"}},
						{ObjectMeta: metav1.ObjectMeta{Name: "app3", Namespace: "argocd"}},
					}, nil
				},
				mustPerformUpdateFn: func() func(
					context.Context,
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
					*argocd.Application,
				) (argocd.OperationPhase, bool, error) {
					var count int
					return func(
						_ context.Context,
						_ *promotion.StepContext,
						_ *builtin.ArgoCDAppUpdate,
						_ *argocd.Application,
					) (argocd.OperationPhase, bool, error) {
						count++
						switch count {
						case 1:
							// App1: completed
							return argocd.OperationSucceeded, false, nil
						case 2:
							// App2: still running
							return argocd.OperationRunning, false, nil
						case 3:
							// App3: needs update
							return "", true, nil
						default:
							return "", false, nil
						}
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
				Apps: []builtin.ArgoCDAppUpdate{{}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				assert.Equal(t, kargoapi.PromotionStepStatusRunning, res.Status)
				assert.NotNil(t, res.RetryAfter)
				require.NoError(t, err)
				// Verify health checks include all 3 apps
				require.NotNil(t, res.HealthCheck)
				apps, ok := res.HealthCheck.Input["apps"]
				assert.True(t, ok)
				assert.Len(t, apps, 3)
			},
		},
		{
			name: "validation fails - no apps updated",
			runner: &argocdUpdater{
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationsFn: func(
					context.Context,
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
				) ([]*argocd.Application, error) {
					return []*argocd.Application{
						{ObjectMeta: metav1.ObjectMeta{Name: "app1", Namespace: "argocd"}},
						{ObjectMeta: metav1.ObjectMeta{Name: "app2", Namespace: "argocd"}},
					}, nil
				},
				buildDesiredSourcesFn: func() func(
					*builtin.ArgoCDAppUpdate,
					[]string,
					*argocd.Application,
				) (argocd.ApplicationSources, error) {
					count := 0
					return func(
						_ *builtin.ArgoCDAppUpdate,
						_ []string,
						app *argocd.Application,
					) (argocd.ApplicationSources, error) {
						count++
						if app.Name == "app2" {
							return nil, fmt.Errorf("no source matched update for repoURL https://github.com/example/repo")
						}
						return []argocd.ApplicationSource{{}}, nil
					}
				}(),
				syncApplicationFn: func(
					context.Context,
					*promotion.StepContext,
					*argocd.Application,
					argocd.ApplicationSources,
				) error {
					panic("syncApplicationFn should not be called when validation fails")
				},
			},
			stepCfg: builtin.ArgoCDUpdateConfig{
				Apps: []builtin.ArgoCDAppUpdate{{
					Sources: []builtin.ArgoCDAppSourceUpdate{
						{RepoURL: "https://github.com/example/repo"},
					},
				}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, res.Status)
				require.ErrorContains(t, err, "selected Applications must have compatible sources")
				require.ErrorContains(t, err, "app2")
				require.ErrorContains(t, err, "No Applications were updated")
			},
		},
		{
			name: "selector returns multiple apps - one operation failed",
			runner: &argocdUpdater{
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationsFn: func(
					context.Context,
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
				) ([]*argocd.Application, error) {
					return []*argocd.Application{
						{ObjectMeta: metav1.ObjectMeta{Name: "app1", Namespace: "argocd"}},
						{
							ObjectMeta: metav1.ObjectMeta{Name: "app2", Namespace: "argocd"},
							Status: argocd.ApplicationStatus{
								OperationState: &argocd.OperationState{
									Message: "deployment failed: timeout",
								},
							},
						},
					}, nil
				},
				buildDesiredSourcesFn: func(
					*builtin.ArgoCDAppUpdate,
					[]string,
					*argocd.Application,
				) (argocd.ApplicationSources, error) {
					return []argocd.ApplicationSource{{}}, nil
				},
				mustPerformUpdateFn: func() func(
					context.Context,
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
					*argocd.Application,
				) (argocd.OperationPhase, bool, error) {
					var count int
					return func(
						_ context.Context,
						_ *promotion.StepContext,
						_ *builtin.ArgoCDAppUpdate,
						_ *argocd.Application,
					) (argocd.OperationPhase, bool, error) {
						count++
						if count == 1 {
							// App1: succeeded
							return argocd.OperationSucceeded, false, nil
						}
						// App2: failed
						return argocd.OperationFailed, false, nil
					}
				}(),
			},
			stepCfg: builtin.ArgoCDUpdateConfig{
				Apps: []builtin.ArgoCDAppUpdate{{}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, res.Status)
				assert.Nil(t, res.RetryAfter)
				require.ErrorContains(t, err, "app2")
				require.ErrorContains(t, err, "deployment failed: timeout")
			},
		},
		{
			name: "mixed name and selector updates",
			runner: &argocdUpdater{
				argocdClient: fake.NewFakeClient(),
				getAuthorizedApplicationsFn: func() func(
					context.Context,
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
				) ([]*argocd.Application, error) {
					var count int
					return func(
						_ context.Context,
						_ *promotion.StepContext,
						_ *builtin.ArgoCDAppUpdate,
					) ([]*argocd.Application, error) {
						count++
						if count == 1 {
							// First update config: name-based (returns 1 app)
							return []*argocd.Application{
								{ObjectMeta: metav1.ObjectMeta{Name: "app-by-name", Namespace: "argocd"}},
							}, nil
						}
						// Second update config: selector-based (returns 2 apps)
						return []*argocd.Application{
							{ObjectMeta: metav1.ObjectMeta{Name: "app1", Namespace: "argocd"}},
							{ObjectMeta: metav1.ObjectMeta{Name: "app2", Namespace: "argocd"}},
						}, nil
					}
				}(),
				buildDesiredSourcesFn: func(
					*builtin.ArgoCDAppUpdate,
					[]string,
					*argocd.Application,
				) (argocd.ApplicationSources, error) {
					return []argocd.ApplicationSource{{}}, nil
				},
				mustPerformUpdateFn: func(
					context.Context,
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
					*argocd.Application,
				) (argocd.OperationPhase, bool, error) {
					return argocd.OperationSucceeded, false, nil
				},
			},
			stepCfg: builtin.ArgoCDUpdateConfig{
				Apps: []builtin.ArgoCDAppUpdate{
					{Name: "app-by-name"}, // Name-based
					{},                    // Selector-based (empty for simplicity)
				},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
				assert.Nil(t, res.RetryAfter)
				require.NoError(t, err)
				// Verify health checks include all 3 apps (1 from name + 2 from selector)
				require.NotNil(t, res.HealthCheck)
				apps, ok := res.HealthCheck.Input["apps"]
				assert.True(t, ok)
				assert.Len(t, apps, 3)
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
				assert.Equal(t, 2, len(desiredSources))
				assert.Equal(t, "fake-version", desiredSources[0].TargetRevision)
				assert.Equal(t, "fake-commit", desiredSources[1].TargetRevision)
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
				assert.Empty(t, phase)
				assert.True(t, mustUpdate)
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
						Info: nil, // promotionInfoKey is not set
					},
				}
			},
			assertions: func(t *testing.T, phase argocd.OperationPhase, mustUpdate bool, err error) {
				require.ErrorContains(t, err, "current operation was initiated by")
				require.ErrorContains(t, err, "waiting for operation to complete")
				assert.Equal(t, argocd.OperationRunning, phase)
				assert.False(t, mustUpdate)
			},
		},
		{
			name: "completed operation initiated by non-kargo",
			modifyApplication: func(app *argocd.Application) {
				app.Status.OperationState = &argocd.OperationState{
					Phase: argocd.OperationSucceeded,
					Operation: argocd.Operation{
						InitiatedBy: argocd.OperationInitiator{
							Username: "someone-else",
						},
						Info: nil, // promotionInfoKey is not set
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
				assert.Equal(t, argocd.OperationRunning, phase)
				assert.False(t, mustUpdate)
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
				assert.False(t, mustUpdate)
				assert.Equal(t, argocd.OperationRunning, phase)
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
				assert.Equal(t, argocd.OperationSucceeded, phase)
				assert.False(t, mustUpdate)
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
				assert.Empty(t, phase)
				assert.True(t, mustUpdate)
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
				assert.Empty(t, phase)
				assert.True(t, mustUpdate)
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
				assert.Equal(t, argocd.OperationSucceeded, phase)
				assert.False(t, mustUpdate)
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
				t.Context(),
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
		assertions     func(*testing.T, *argocd.Application, error)
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
			assertions: func(t *testing.T, _ *argocd.Application, err error) {
				require.ErrorContains(t, err, "error patching Argo CD Application")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "success (no sources to update)",
			runner: &argocdUpdater{
				argoCDAppPatchFn: func(
					context.Context,
					kubeclient.ObjectWithKind,
					kubeclient.UnstructuredPatchFn,
				) error {
					return nil
				},
				logAppEventFn: func(context.Context,
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
			assertions: func(t *testing.T, _ *argocd.Application, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "updates Sources when present",
			runner: &argocdUpdater{
				argoCDAppPatchFn: func(
					context.Context,
					kubeclient.ObjectWithKind,
					kubeclient.UnstructuredPatchFn,
				) error {
					return nil
				},
				logAppEventFn: func(context.Context,
					*argocd.Application,
					string,
					string,
					string,
				) {
				},
			},
			app: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{Name: "fake", Namespace: "fake"},
				Spec: argocd.ApplicationSpec{
					Sources: argocd.ApplicationSources{{TargetRevision: "old-rev"}},
				},
			},
			desiredSources: argocd.ApplicationSources{{TargetRevision: "new-rev"}},
			assertions: func(t *testing.T, patched *argocd.Application, err error) {
				require.NoError(t, err)
				require.NotNil(t, patched)
				assert.Len(t, patched.Spec.Sources, 1)
				assert.Equal(t, "new-rev", patched.Spec.Sources[0].TargetRevision)
			},
		},
		{
			name: "updates Source when Sources not present",
			runner: &argocdUpdater{
				argoCDAppPatchFn: func(
					context.Context,
					kubeclient.ObjectWithKind,
					kubeclient.UnstructuredPatchFn,
				) error {
					return nil
				},
				logAppEventFn: func(context.Context,
					*argocd.Application,
					string,
					string,
					string,
				) {
				}},
			app: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{Name: "fake", Namespace: "fake"},
				Spec: argocd.ApplicationSpec{
					Source: &argocd.ApplicationSource{TargetRevision: "old-rev"},
				},
			},
			desiredSources: argocd.ApplicationSources{{TargetRevision: "new-rev"}},
			assertions: func(t *testing.T, patched *argocd.Application, err error) {
				require.NoError(t, err)
				require.NotNil(t, patched)
				require.NotNil(t, patched.Spec.Source)
				assert.Equal(t, "new-rev", patched.Spec.Source.TargetRevision)
			},
		},
		{
			name: "updates Sources when both Source and Sources are present",
			runner: &argocdUpdater{
				argoCDAppPatchFn: func(
					context.Context,
					kubeclient.ObjectWithKind,
					kubeclient.UnstructuredPatchFn,
				) error {
					return nil
				},
				logAppEventFn: func(context.Context,
					*argocd.Application,
					string,
					string,
					string,
				) {
				}},
			app: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{Name: "fake", Namespace: "fake"},
				Spec: argocd.ApplicationSpec{
					Source:  &argocd.ApplicationSource{TargetRevision: "old-rev-source"},
					Sources: argocd.ApplicationSources{{TargetRevision: "old-rev-sources"}},
				},
			},
			desiredSources: argocd.ApplicationSources{{TargetRevision: "new-rev"}},
			assertions: func(t *testing.T, patched *argocd.Application, err error) {
				require.NoError(t, err)
				require.NotNil(t, patched)
				// Sources should be updated
				assert.Len(t, patched.Spec.Sources, 1)
				assert.Equal(t, "new-rev", patched.Spec.Sources[0].TargetRevision)
				// Source should remain untouched
				assert.Equal(t, "old-rev-source", patched.Spec.Source.TargetRevision)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			stepCtx := &promotion.StepContext{
				Freight: kargoapi.FreightCollection{
					ID: "fake-freight-collection-id",
				},
			}
			err := testCase.runner.syncApplication(
				context.Background(),
				stepCtx,
				testCase.app,
				testCase.desiredSources,
			)
			testCase.assertions(t, testCase.app, err)
		})
	}
}

func TestSyncMessage(t *testing.T) {
	testCases := []struct {
		name     string
		app      *argocd.Application
		expected string
	}{
		{
			name: "single Source",
			app: &argocd.Application{
				Spec: argocd.ApplicationSpec{
					Source: &argocd.ApplicationSource{
						TargetRevision: "rev-123",
					},
				},
			},
			expected: "initiated sync to rev-123",
		},
		{
			name: "single Sources",
			app: &argocd.Application{
				Spec: argocd.ApplicationSpec{
					Sources: argocd.ApplicationSources{
						{
							TargetRevision: "rev-456",
						},
					},
				},
			},
			expected: "initiated sync to rev-456",
		},
		{
			name: "multiple Sources",
			app: &argocd.Application{
				Spec: argocd.ApplicationSpec{
					Sources: argocd.ApplicationSources{
						{TargetRevision: "rev-a"},
						{TargetRevision: "rev-b"},
					},
				},
			},
			expected: "initiated sync to 2 sources",
		},
		{
			name:     "no Source or Sources",
			app:      &argocd.Application{},
			expected: "initiated sync",
		},
	}

	runner := &argocdUpdater{}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			message := runner.formatSyncMessage(tc.app)
			assert.Equal(t, tc.expected, message)
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
				assert.Len(t, events.Items, 1)

				event := events.Items[0]
				assert.Equal(t, corev1.ObjectReference{
					APIVersion:      argocd.GroupVersion.String(),
					Kind:            app.Kind,
					Name:            app.Name,
					Namespace:       app.Namespace,
					UID:             app.UID,
					ResourceVersion: app.ResourceVersion,
				}, event.InvolvedObject)
				assert.NotNil(t, event.FirstTimestamp)
				assert.NotNil(t, event.LastTimestamp)
				assert.Equal(t, 1, int(event.Count))
				assert.Equal(t, corev1.EventTypeNormal, event.Type)
				assert.Equal(t, "fake-reason", event.Reason)
				assert.Equal(t, "fake-user fake-message", event.Message)
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
				assert.Len(t, events.Items, 1)

				event := events.Items[0]
				assert.Equal(t, "Unknown user fake-message", event.Message)
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
				assert.False(t, updated)
				// Source should be entirely unchanged
				assert.Equal(t, originalSource, updatedSource)
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
				assert.True(t, updated)
				// TargetRevision should be updated
				assert.Equal(t, "fake-commit", updatedSource.TargetRevision)
				// Everything else should be unchanged
				updatedSource.TargetRevision = originalSource.TargetRevision
				assert.Equal(t, originalSource, updatedSource)
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
				assert.True(t, updated)
				// TargetRevision should be updated
				assert.Equal(t, "fake-version", updatedSource.TargetRevision)
				// Everything else should be unchanged
				updatedSource.TargetRevision = originalSource.TargetRevision
				assert.Equal(t, originalSource, updatedSource)
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
				assert.True(t, updated)
				// Kustomize attributes should be updated
				require.NotNil(t, updatedSource.Kustomize)
				assert.Equal(
					t,
					argocd.KustomizeImages{
						"fake-image-url:fake-tag",
					},
					updatedSource.Kustomize.Images,
				)
				// Everything else should be unchanged
				updatedSource.Kustomize = originalSource.Kustomize
				assert.Equal(t, originalSource, updatedSource)
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
				assert.True(t, updated)
				// Helm attributes should be updated
				require.NotNil(t, updatedSource.Helm)
				require.NotNil(t, updatedSource.Helm.Parameters)
				assert.Equal(
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
				assert.Equal(t, originalSource, updatedSource)
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
	assert.Equal(
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
	assert.Equal(
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
			assert.Equal(t, tc.expected, result)
		})
	}
}

func Test_argoCDUpdater_buildLabelSelector(t *testing.T) {
	testCases := []struct {
		name       string
		selector   *builtin.ArgoCDAppSelector
		assertions func(*testing.T, labels.Selector, error)
	}{
		{
			name: "selector with matchLabels only",
			selector: &builtin.ArgoCDAppSelector{
				MatchLabels: map[string]string{
					"env":  "prod",
					"team": "platform",
				},
			},
			assertions: func(t *testing.T, sel labels.Selector, err error) {
				require.NoError(t, err)
				require.NotNil(t, sel)
				assert.True(t, sel.Matches(labels.Set{"env": "prod", "team": "platform"}))
				assert.False(t, sel.Matches(labels.Set{"env": "dev", "team": "platform"}))
			},
		},
		{
			name: "selector with matchExpressions only",
			selector: &builtin.ArgoCDAppSelector{
				MatchExpressions: []builtin.MatchExpression{
					{
						Key:      "env",
						Operator: builtin.In,
						Values:   []string{"prod", "staging"},
					},
				},
			},
			assertions: func(t *testing.T, sel labels.Selector, err error) {
				require.NoError(t, err)
				require.NotNil(t, sel)
				assert.True(t, sel.Matches(labels.Set{"env": "prod"}))
				assert.True(t, sel.Matches(labels.Set{"env": "staging"}))
				assert.False(t, sel.Matches(labels.Set{"env": "dev"}))
			},
		},
		{
			name: "selector with both matchLabels and matchExpressions",
			selector: &builtin.ArgoCDAppSelector{
				MatchLabels: map[string]string{
					"team": "platform",
				},
				MatchExpressions: []builtin.MatchExpression{
					{
						Key:      "env",
						Operator: builtin.In,
						Values:   []string{"prod", "staging"},
					},
				},
			},
			assertions: func(t *testing.T, sel labels.Selector, err error) {
				require.NoError(t, err)
				require.NotNil(t, sel)
				assert.True(t, sel.Matches(labels.Set{"env": "prod", "team": "platform"}))
				assert.False(t, sel.Matches(labels.Set{"env": "prod", "team": "other"}))
				assert.False(t, sel.Matches(labels.Set{"env": "dev", "team": "platform"}))
			},
		},
		{
			name: "selector with NotIn operator",
			selector: &builtin.ArgoCDAppSelector{
				MatchExpressions: []builtin.MatchExpression{
					{
						Key:      "env",
						Operator: builtin.NotIn,
						Values:   []string{"dev", "test"},
					},
				},
			},
			assertions: func(t *testing.T, sel labels.Selector, err error) {
				require.NoError(t, err)
				require.NotNil(t, sel)
				assert.True(t, sel.Matches(labels.Set{"env": "prod"}))
				assert.False(t, sel.Matches(labels.Set{"env": "dev"}))
			},
		},
		{
			name: "selector with Exists operator",
			selector: &builtin.ArgoCDAppSelector{
				MatchExpressions: []builtin.MatchExpression{
					{
						Key:      "environment",
						Operator: builtin.Exists,
					},
				},
			},
			assertions: func(t *testing.T, sel labels.Selector, err error) {
				require.NoError(t, err)
				require.NotNil(t, sel)
				assert.True(t, sel.Matches(labels.Set{"environment": "prod"}))
				assert.False(t, sel.Matches(labels.Set{"env": "prod"}))
			},
		},
		{
			name: "selector with DoesNotExist operator",
			selector: &builtin.ArgoCDAppSelector{
				MatchExpressions: []builtin.MatchExpression{
					{
						Key:      "deprecated",
						Operator: builtin.DoesNotExist,
					},
				},
			},
			assertions: func(t *testing.T, sel labels.Selector, err error) {
				require.NoError(t, err)
				require.NotNil(t, sel)
				assert.True(t, sel.Matches(labels.Set{"env": "prod"}))
				assert.False(t, sel.Matches(labels.Set{"deprecated": "true"}))
			},
		},
		{
			name: "empty selector returns error",
			selector: &builtin.ArgoCDAppSelector{
				MatchLabels:      map[string]string{},
				MatchExpressions: []builtin.MatchExpression{},
			},
			assertions: func(t *testing.T, sel labels.Selector, err error) {
				require.ErrorContains(t, err, "selector must have at least one match criterion")
				require.Nil(t, sel)
			},
		},
	}

	runner := &argocdUpdater{}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			sel, err := runner.buildLabelSelector(testCase.selector)
			testCase.assertions(t, sel, err)
		})
	}
}

func Test_argoCDUpdater_getAuthorizedApplications(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, argocd.AddToScheme(scheme))

	testCases := []struct {
		name        string
		apps        []*argocd.Application
		update      *builtin.ArgoCDAppUpdate
		interceptor interceptor.Funcs
		assertions  func(*testing.T, []*argocd.Application, error)
	}{
		{
			name: "selector returns multiple authorized apps",
			apps: []*argocd.Application{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "app1",
						Namespace: "argocd",
						Labels: map[string]string{
							"env": "prod",
						},
						Annotations: map[string]string{
							kargoapi.AnnotationKeyAuthorizedStage: "fake-project:fake-stage",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "app2",
						Namespace: "argocd",
						Labels: map[string]string{
							"env": "prod",
						},
						Annotations: map[string]string{
							kargoapi.AnnotationKeyAuthorizedStage: "fake-project:fake-stage",
						},
					},
				},
			},
			update: &builtin.ArgoCDAppUpdate{
				Namespace: "argocd",
				Selector: &builtin.ArgoCDAppSelector{
					MatchLabels: map[string]string{
						"env": "prod",
					},
				},
			},
			assertions: func(t *testing.T, apps []*argocd.Application, err error) {
				require.NoError(t, err)
				assert.Len(t, apps, 2)
				assert.Equal(t, "app1", apps[0].Name)
				assert.Equal(t, "app2", apps[1].Name)
			},
		},
		{
			name: "selector filters out unauthorized apps",
			apps: []*argocd.Application{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "app1",
						Namespace: "argocd",
						Labels: map[string]string{
							"env": "prod",
						},
						Annotations: map[string]string{
							kargoapi.AnnotationKeyAuthorizedStage: "fake-project:fake-stage",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "app2",
						Namespace: "argocd",
						Labels: map[string]string{
							"env": "prod",
						},
						// No authorization annotation
					},
				},
			},
			update: &builtin.ArgoCDAppUpdate{
				Namespace: "argocd",
				Selector: &builtin.ArgoCDAppSelector{
					MatchLabels: map[string]string{
						"env": "prod",
					},
				},
			},
			assertions: func(t *testing.T, apps []*argocd.Application, err error) {
				require.NoError(t, err)
				assert.Len(t, apps, 1)
				assert.Equal(t, "app1", apps[0].Name)
			},
		},
		{
			name: "selector returns no apps",
			apps: []*argocd.Application{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "app1",
						Namespace: "argocd",
						Labels: map[string]string{
							"env": "dev",
						},
					},
				},
			},
			update: &builtin.ArgoCDAppUpdate{
				Namespace: "argocd",
				Selector: &builtin.ArgoCDAppSelector{
					MatchLabels: map[string]string{
						"env": "prod",
					},
				},
			},
			assertions: func(t *testing.T, apps []*argocd.Application, err error) {
				require.ErrorContains(t, err, "no Argo CD Applications found matching selector")
				require.Nil(t, apps)
			},
		},
		{
			name: "selector matches apps but none authorized",
			apps: []*argocd.Application{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "app1",
						Namespace: "argocd",
						Labels: map[string]string{
							"env": "prod",
						},
						// No authorization annotation
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "app2",
						Namespace: "argocd",
						Labels: map[string]string{
							"env": "prod",
						},
						// No authorization annotation
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "app3",
						Namespace: "argocd",
						Labels: map[string]string{
							"env": "prod",
						},
						// No authorization annotation
					},
				},
			},
			update: &builtin.ArgoCDAppUpdate{
				Namespace: "argocd",
				Selector: &builtin.ArgoCDAppSelector{
					MatchLabels: map[string]string{
						"env": "prod",
					},
				},
			},
			assertions: func(t *testing.T, apps []*argocd.Application, err error) {
				require.ErrorContains(t, err, "found 3 Application(s) matching selector")
				require.ErrorContains(t, err, "but none are authorized for Stage fake-project:fake-stage")
				require.Nil(t, apps)
			},
		},
		{
			name: "name-based selection returns single app",
			apps: []*argocd.Application{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-app",
						Namespace: "argocd",
						Annotations: map[string]string{
							kargoapi.AnnotationKeyAuthorizedStage: "fake-project:fake-stage",
						},
					},
				},
			},
			update: &builtin.ArgoCDAppUpdate{
				Name:      "my-app",
				Namespace: "argocd",
			},
			assertions: func(t *testing.T, apps []*argocd.Application, err error) {
				require.NoError(t, err)
				require.Len(t, apps, 1)
				require.Equal(t, "my-app", apps[0].Name)
			},
		},
		{
			name: "name-based selection app not found",
			apps: []*argocd.Application{},
			update: &builtin.ArgoCDAppUpdate{
				Name:      "nonexistent-app",
				Namespace: "argocd",
			},
			assertions: func(t *testing.T, apps []*argocd.Application, err error) {
				require.ErrorContains(t, err, "unable to find Argo CD Application")
				require.Nil(t, apps)
			},
		},
		{
			name: "name-based selection app not authorized",
			apps: []*argocd.Application{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-app",
						Namespace: "argocd",
						// No authorization annotation
					},
				},
			},
			update: &builtin.ArgoCDAppUpdate{
				Name:      "my-app",
				Namespace: "argocd",
			},
			assertions: func(t *testing.T, apps []*argocd.Application, err error) {
				require.ErrorContains(t, err, "is not authorized")
				require.Nil(t, apps)
			},
		},
		{
			name: "error listing applications",
			update: &builtin.ArgoCDAppUpdate{
				Namespace: "argocd",
				Selector: &builtin.ArgoCDAppSelector{
					MatchLabels: map[string]string{
						"env": "prod",
					},
				},
			},
			interceptor: interceptor.Funcs{
				List: func(
					context.Context,
					client.WithWatch,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, apps []*argocd.Application, err error) {
				require.ErrorContains(t, err, "error listing Argo CD Applications")
				require.ErrorContains(t, err, "something went wrong")
				require.Nil(t, apps)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithInterceptorFuncs(testCase.interceptor)

			if len(testCase.apps) > 0 {
				objects := make([]client.Object, len(testCase.apps))
				for i, app := range testCase.apps {
					objects[i] = app
				}
				c.WithObjects(objects...)
			}

			runner := &argocdUpdater{
				argocdClient: c.Build(),
			}
			runner.buildLabelSelectorFn = runner.buildLabelSelector

			apps, err := runner.getAuthorizedApplications(
				context.Background(),
				&promotion.StepContext{
					Project: "fake-project",
					Stage:   "fake-stage",
				},
				testCase.update,
			)
			testCase.assertions(t, apps, err)
		})
	}
}

func Test_argoCDUpdater_processApplication(t *testing.T) {
	testCases := []struct {
		name       string
		runner     *argocdUpdater
		update     *builtin.ArgoCDAppUpdate
		app        *argocd.Application
		assertions func(*testing.T, argocd.OperationPhase, error)
	}{
		{
			name: "application requires update and succeeds",
			runner: &argocdUpdater{
				mustPerformUpdateFn: func(
					context.Context,
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
					return nil
				},
			},
			update: &builtin.ArgoCDAppUpdate{},
			app:    &argocd.Application{},
			assertions: func(t *testing.T, phase argocd.OperationPhase, err error) {
				require.NoError(t, err)
				assert.Equal(t, argocd.OperationRunning, phase)
			},
		},
		{
			name: "application does not require update",
			runner: &argocdUpdater{
				mustPerformUpdateFn: func(
					context.Context,
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
					*argocd.Application,
				) (argocd.OperationPhase, bool, error) {
					return argocd.OperationSucceeded, false, nil
				},
			},
			update: &builtin.ArgoCDAppUpdate{},
			app:    &argocd.Application{},
			assertions: func(t *testing.T, phase argocd.OperationPhase, err error) {
				require.NoError(t, err)
				assert.Equal(t, argocd.OperationSucceeded, phase)
			},
		},
		{
			name: "application failed - returns error with message",
			runner: &argocdUpdater{
				mustPerformUpdateFn: func(
					context.Context,
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
					*argocd.Application,
				) (argocd.OperationPhase, bool, error) {
					return argocd.OperationFailed, false, nil
				},
			},
			update: &builtin.ArgoCDAppUpdate{},
			app: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: "argocd",
				},
				Status: argocd.ApplicationStatus{
					OperationState: &argocd.OperationState{
						Message: "sync failed: resource not found",
					},
				},
			},
			assertions: func(t *testing.T, phase argocd.OperationPhase, err error) {
				require.ErrorContains(t, err, "test-app")
				require.ErrorContains(t, err, "argocd")
				require.ErrorContains(t, err, "sync failed: resource not found")
				assert.Equal(t, argocd.OperationPhase(""), phase)
			},
		},
		{
			name: "application failed - returns error without message",
			runner: &argocdUpdater{
				mustPerformUpdateFn: func(
					context.Context,
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
					*argocd.Application,
				) (argocd.OperationPhase, bool, error) {
					return argocd.OperationFailed, false, nil
				},
			},
			update: &builtin.ArgoCDAppUpdate{},
			app: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: "argocd",
				},
				Status: argocd.ApplicationStatus{
					OperationState: nil,
				},
			},
			assertions: func(t *testing.T, phase argocd.OperationPhase, err error) {
				require.NoError(t, err)
				assert.Equal(t, argocd.OperationFailed, phase)
			},
		},
		{
			name: "error building desired sources",
			runner: &argocdUpdater{
				mustPerformUpdateFn: func(
					context.Context,
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
					return nil, errors.New("failed to build sources")
				},
			},
			update: &builtin.ArgoCDAppUpdate{},
			app: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: "argocd",
				},
			},
			assertions: func(t *testing.T, phase argocd.OperationPhase, err error) {
				require.ErrorContains(t, err, "error building desired sources")
				require.ErrorContains(t, err, "test-app")
				require.ErrorContains(t, err, "failed to build sources")
				assert.Equal(t, argocd.OperationPhase(""), phase)
			},
		},
		{
			name: "error syncing application",
			runner: &argocdUpdater{
				mustPerformUpdateFn: func(
					context.Context,
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
					return errors.New("sync operation failed")
				},
			},
			update: &builtin.ArgoCDAppUpdate{},
			app: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: "argocd",
				},
			},
			assertions: func(t *testing.T, phase argocd.OperationPhase, err error) {
				require.ErrorContains(t, err, "error syncing")
				require.ErrorContains(t, err, "test-app")
				require.ErrorContains(t, err, "sync operation failed")
				assert.Equal(t, argocd.OperationPhase(""), phase)
			},
		},
		{
			name: "mustUpdate with phase and error - continues processing",
			runner: &argocdUpdater{
				mustPerformUpdateFn: func(
					context.Context,
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
					*argocd.Application,
				) (argocd.OperationPhase, bool, error) {
					return argocd.OperationRunning, false, errors.New("operation in progress")
				},
			},
			update: &builtin.ArgoCDAppUpdate{},
			app:    &argocd.Application{},
			assertions: func(t *testing.T, phase argocd.OperationPhase, err error) {
				require.NoError(t, err)
				assert.Equal(t, argocd.OperationRunning, phase)
			},
		},
		{
			name: "mustUpdate without phase and error - stops processing",
			runner: &argocdUpdater{
				mustPerformUpdateFn: func(
					context.Context,
					*promotion.StepContext,
					*builtin.ArgoCDAppUpdate,
					*argocd.Application,
				) (argocd.OperationPhase, bool, error) {
					return "", false, errors.New("cannot determine status")
				},
			},
			update: &builtin.ArgoCDAppUpdate{},
			app:    &argocd.Application{},
			assertions: func(t *testing.T, phase argocd.OperationPhase, err error) {
				require.ErrorContains(t, err, "cannot determine status")
				assert.Equal(t, argocd.OperationPhase(""), phase)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			phase, err := testCase.runner.processApplication(
				context.Background(),
				&promotion.StepContext{},
				testCase.update,
				testCase.app,
			)
			testCase.assertions(t, phase, err)
		})
	}
}

func Test_argoCDUpdater_validateSourceUpdatesApplicable(t *testing.T) {
	testCases := []struct {
		name       string
		runner     *argocdUpdater
		update     *builtin.ArgoCDAppUpdate
		apps       []*argocd.Application
		assertions func(t *testing.T, err error)
	}{
		{
			name:   "empty apps list returns no error",
			runner: &argocdUpdater{},
			update: &builtin.ArgoCDAppUpdate{},
			apps:   []*argocd.Application{},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "single app with valid sources",
			runner: &argocdUpdater{
				buildDesiredSourcesFn: func(
					*builtin.ArgoCDAppUpdate,
					[]string,
					*argocd.Application,
				) (argocd.ApplicationSources, error) {
					return []argocd.ApplicationSource{{}}, nil
				},
			},
			update: &builtin.ArgoCDAppUpdate{},
			apps: []*argocd.Application{
				{ObjectMeta: metav1.ObjectMeta{Name: "app1", Namespace: "argocd"}},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "multiple apps all valid",
			runner: &argocdUpdater{
				buildDesiredSourcesFn: func(
					*builtin.ArgoCDAppUpdate,
					[]string,
					*argocd.Application,
				) (argocd.ApplicationSources, error) {
					return []argocd.ApplicationSource{{}}, nil
				},
			},
			update: &builtin.ArgoCDAppUpdate{},
			apps: []*argocd.Application{
				{ObjectMeta: metav1.ObjectMeta{Name: "app1", Namespace: "argocd"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "app2", Namespace: "argocd"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "app3", Namespace: "argocd"}},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "one app fails validation",
			runner: &argocdUpdater{
				buildDesiredSourcesFn: func() func(
					*builtin.ArgoCDAppUpdate,
					[]string,
					*argocd.Application,
				) (argocd.ApplicationSources, error) {
					count := 0
					return func(
						_ *builtin.ArgoCDAppUpdate,
						_ []string,
						_ *argocd.Application,
					) (argocd.ApplicationSources, error) {
						count++
						if count == 2 {
							return nil, fmt.Errorf("no source matched update for repoURL https://github.com/example/repo")
						}
						return []argocd.ApplicationSource{{}}, nil
					}
				}(),
			},
			update: &builtin.ArgoCDAppUpdate{},
			apps: []*argocd.Application{
				{ObjectMeta: metav1.ObjectMeta{Name: "app1", Namespace: "argocd"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "app2", Namespace: "argocd"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "app3", Namespace: "argocd"}},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "selected Applications must have compatible sources")
				require.ErrorContains(t, err, "1 incompatible")
				require.ErrorContains(t, err, "No Applications were updated")
				require.ErrorContains(t, err, "app2")
			},
		},
		{
			name: "multiple apps fail validation - aggregates errors",
			runner: &argocdUpdater{
				buildDesiredSourcesFn: func() func(
					*builtin.ArgoCDAppUpdate,
					[]string,
					*argocd.Application,
				) (argocd.ApplicationSources, error) {
					count := 0
					return func(
						_ *builtin.ArgoCDAppUpdate,
						_ []string,
						_ *argocd.Application,
					) (argocd.ApplicationSources, error) {
						count++
						if count == 2 || count == 4 {
							return nil, fmt.Errorf("no source matched update for repoURL https://github.com/example/repo")
						}
						return []argocd.ApplicationSource{{}}, nil
					}
				}(),
			},
			update: &builtin.ArgoCDAppUpdate{},
			apps: []*argocd.Application{
				{ObjectMeta: metav1.ObjectMeta{Name: "app1", Namespace: "argocd"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "app2", Namespace: "argocd"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "app3", Namespace: "argocd"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "app4", Namespace: "argocd"}},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "2 incompatible")
				require.ErrorContains(t, err, "app2")
				require.ErrorContains(t, err, "app4")
			},
		},
		{
			name: "many apps fail - limits error reporting to first 3",
			runner: &argocdUpdater{
				buildDesiredSourcesFn: func(
					_ *builtin.ArgoCDAppUpdate,
					_ []string,
					app *argocd.Application,
				) (argocd.ApplicationSources, error) {
					return nil, fmt.Errorf("validation error for %s", app.Name)
				},
			},
			update: &builtin.ArgoCDAppUpdate{},
			apps: func() []*argocd.Application {
				apps := make([]*argocd.Application, 5)
				for i := 0; i < 5; i++ {
					apps[i] = &argocd.Application{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("app%d", i+1),
							Namespace: "argocd",
						},
					}
				}
				return apps
			}(),
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "5 incompatible (showing first 3)")
				require.ErrorContains(t, err, "app1")
				require.ErrorContains(t, err, "app2")
				require.ErrorContains(t, err, "app3")
				require.NotContains(t, err.Error(), "app4")
				require.NotContains(t, err.Error(), "app5")
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.runner.validateSourceUpdatesApplicable(
				testCase.update,
				testCase.apps,
			)
			testCase.assertions(t, err)
		})
	}
}
