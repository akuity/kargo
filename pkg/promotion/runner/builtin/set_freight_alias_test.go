package builtin

import (
	"context"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func Test_setFreightAlias_convert(t *testing.T) {
	tests := []validationTestCase{
		{
			name:   "both freightID and newAlias are not specified",
			config: promotion.Config{},
			expectedProblems: []string{
				"(root): freightID is required",
				"(root): newAlias is required",
			},
		},
		{
			name: "freightID is not specified",
			config: promotion.Config{
				"newAlias": "stable",
			},
			expectedProblems: []string{
				"(root): freightID is required",
			},
		},
		{
			name: "freightID is empty",
			config: promotion.Config{
				"freightID": "",
				"newAlias":  "new-alias",
			},
			expectedProblems: []string{
				"freightID: String length must be greater than or equal to 1",
			},
		},
		{
			name: "newAlias is not specified",
			config: promotion.Config{
				"freightID": "fake-freight-id",
			},
			expectedProblems: []string{
				"(root): newAlias is required",
			},
		},
		{
			name: "newAlias is empty",
			config: promotion.Config{
				"freightID": "fake-freight-id",
				"newAlias":  "",
			},
			expectedProblems: []string{
				"newAlias: String length must be greater than or equal to 1",
			},
		},
		{
			name: "unknown field is not allowed",
			config: promotion.Config{
				"freightID":  "fake-freight-id",
				"newAlias":   "new-alias",
				"unexpected": "nope",
			},
			expectedProblems: []string{
				"(root): Additional property unexpected is not allowed",
			},
		},
		{
			name: "valid minimal config",
			config: promotion.Config{
				"freightID": "fake-freight-id",
				"newAlias":  "new-alias",
			},
		},
	}

	r := newSetFreightAlias(promotion.StepRunnerCapabilities{})
	runner, ok := r.(*setFreightAlias)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

func Test_setFreightAlias_run(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	const (
		testFreight  = "freight-id-1"
		otherFreight = "freight-id-2"
		testProject  = "test-project"
		oldAlias     = "old-alias"
		newAlias     = "new-alias"
	)

	tests := []struct {
		name       string
		client     client.Client
		cfg        builtin.SetFreightAliasConfig
		assertions func(t2 *testing.T, result promotion.StepResult, client2 client.Client, err error)
	}{
		{
			name:   "freight not found",
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			cfg: builtin.SetFreightAliasConfig{
				FreightID: testFreight,
				NewAlias:  newAlias,
			},
			assertions: func(t *testing.T, res promotion.StepResult, _ client.Client, err error) {
				require.ErrorContains(t, err, "not found")
				require.ErrorContains(t, err, testFreight)
				require.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)
			},
		},
		{
			name: "alias already used by another freight in the project",
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(
					&kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Name:      testFreight,
							Namespace: testProject,
						},
					},
					&kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Name:      otherFreight,
							Namespace: testProject,
							Labels: map[string]string{
								kargoapi.LabelKeyAlias: newAlias,
							},
						},
						Alias: newAlias,
					},
				).
				Build(),
			cfg: builtin.SetFreightAliasConfig{
				FreightID: testFreight,
				NewAlias:  newAlias,
			},
			assertions: func(t *testing.T, res promotion.StepResult, _ client.Client, err error) {
				require.ErrorContains(t, err, "already in use")
				require.ErrorContains(t, err, newAlias)
				require.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)
			},
		},
		{
			name: "patch error",
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(
					&kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Name:            testFreight,
							Namespace:       testProject,
							ResourceVersion: "invalid", // forces patch failure
						},
						Alias: oldAlias,
					},
				).
				Build(),
			cfg: builtin.SetFreightAliasConfig{
				FreightID: testFreight,
				NewAlias:  newAlias,
			},
			assertions: func(t *testing.T, res promotion.StepResult, _ client.Client, err error) {
				require.ErrorContains(t, err, "failed to patch alias")
				require.ErrorContains(t, err, testFreight)
				require.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)
			},
		},
		{
			name: "successful alias update",
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(
					&kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Name:      testFreight,
							Namespace: testProject,
							Labels: map[string]string{
								kargoapi.LabelKeyAlias: oldAlias,
							},
						},
						Alias: oldAlias,
					},
				).
				Build(),
			cfg: builtin.SetFreightAliasConfig{
				FreightID: testFreight,
				NewAlias:  newAlias,
			},
			assertions: func(t *testing.T, res promotion.StepResult, c client.Client, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)

				freight := &kargoapi.Freight{}
				err = c.Get(
					context.Background(),
					types.NamespacedName{
						Name:      testFreight,
						Namespace: testProject,
					},
					freight,
				)
				require.NoError(t, err)

				require.Equal(t, newAlias, freight.Alias)
				require.Equal(t, newAlias, freight.Labels[kargoapi.LabelKeyAlias])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &setFreightAlias{
				kargoClient: tt.client,
			}

			stepCtx := &promotion.StepContext{
				Project: testProject,
			}

			res, err := step.run(context.Background(), stepCtx, tt.cfg)
			tt.assertions(t, res, tt.client, err)
		})
	}
}
