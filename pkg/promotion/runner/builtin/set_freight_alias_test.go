package builtin

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/client/dataplane/freightclient/testutil"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/stepconfig"
)

func Test_setFreightAlias_convert(t *testing.T) {
	tests := []validationTestCase{
		{
			name:   "both name and alias are not specified",
			config: promotion.Config{},
			expectedProblems: []string{
				"(root): name is required",
				"(root): alias is required",
			},
		},
		{
			name: "name is not specified",
			config: promotion.Config{
				"alias": "stable",
			},
			expectedProblems: []string{
				"(root): name is required",
			},
		},
		{
			name: "name is empty",
			config: promotion.Config{
				"name":  "",
				"alias": "new-alias",
			},
			expectedProblems: []string{
				"name: String length must be greater than or equal to 1",
			},
		},
		{
			name: "alias is not specified",
			config: promotion.Config{
				"name": "fake-freight-id",
			},
			expectedProblems: []string{
				"(root): alias is required",
			},
		},
		{
			name: "alias is empty",
			config: promotion.Config{
				"name":  "fake-freight-id",
				"alias": "",
			},
			expectedProblems: []string{
				"alias: String length must be greater than or equal to 1",
			},
		},
		{
			name: "unknown field is not allowed",
			config: promotion.Config{
				"name":       "fake-freight-id",
				"alias":      "new-alias",
				"unexpected": "nope",
			},
			expectedProblems: []string{
				"(root): Additional property unexpected is not allowed",
			},
		},
		{
			name: "valid minimal config",
			config: promotion.Config{
				"name":  "fake-freight-id",
				"alias": "new-alias",
			},
		},
	}

	r := newSetFreightAlias(promotion.StepRunnerCapabilities{})
	runner, ok := r.(*freightAliasSetter)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

func Test_setFreightAlias_run(t *testing.T) {
	const (
		testFreight = "freight-id-1"
		testProject = "test-project"
		oldAlias    = "old-alias"
		alias       = "new-alias"
	)

	// aliasPatch records a call to PatchFreightAlias.
	type aliasPatch struct {
		project     string
		nameOrAlias string
		newAlias    string
	}

	existingFreight := func(context.Context, string, string) (*kargoapi.Freight, error) {
		return &kargoapi.Freight{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testFreight,
				Namespace: testProject,
				Labels:    map[string]string{kargoapi.LabelKeyAlias: oldAlias},
			},
			Alias: oldAlias,
		}, nil
	}

	tests := []struct {
		name         string
		getFreightFn func(ctx context.Context, project, nameOrAlias string) (*kargoapi.Freight, error)
		patchErr     error
		cfg          stepconfig.SetFreightAliasConfig
		assertions   func(*testing.T, promotion.StepResult, []aliasPatch, error)
	}{
		{
			name: "error getting freight",
			getFreightFn: func(context.Context, string, string) (*kargoapi.Freight, error) {
				return nil, errors.New("something went wrong")
			},
			cfg: stepconfig.SetFreightAliasConfig{
				Name:  testFreight,
				Alias: alias,
			},
			assertions: func(t *testing.T, res promotion.StepResult, patches []aliasPatch, err error) {
				require.ErrorContains(t, err, "failed to fetch Freight")
				require.ErrorContains(t, err, "something went wrong")
				require.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)
				require.Empty(t, patches)
			},
		},
		{
			name: "freight not found",
			getFreightFn: func(context.Context, string, string) (*kargoapi.Freight, error) {
				return nil, nil
			},
			cfg: stepconfig.SetFreightAliasConfig{
				Name:  testFreight,
				Alias: alias,
			},
			assertions: func(t *testing.T, res promotion.StepResult, patches []aliasPatch, err error) {
				require.ErrorContains(t, err, "not found")
				require.ErrorContains(t, err, testFreight)
				require.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)
				require.Empty(t, patches)
			},
		},
		{
			name:         "patch error",
			getFreightFn: existingFreight,
			patchErr:     errors.New("something went wrong"),
			cfg: stepconfig.SetFreightAliasConfig{
				Name:  testFreight,
				Alias: alias,
			},
			assertions: func(t *testing.T, res promotion.StepResult, patches []aliasPatch, err error) {
				require.ErrorContains(t, err, "failed to patch alias")
				require.ErrorContains(t, err, testFreight)
				require.ErrorContains(t, err, "something went wrong")
				require.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)
				require.Len(t, patches, 1)
			},
		},
		{
			name:         "successful alias update",
			getFreightFn: existingFreight,
			cfg: stepconfig.SetFreightAliasConfig{
				Name:  testFreight,
				Alias: alias,
			},
			assertions: func(t *testing.T, res promotion.StepResult, patches []aliasPatch, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
				require.Contains(t, res.Message, oldAlias)
				require.Contains(t, res.Message, alias)
				require.Equal(
					t,
					[]aliasPatch{{
						project:     testProject,
						nameOrAlias: testFreight,
						newAlias:    alias,
					}},
					patches,
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var patches []aliasPatch
			step := &freightAliasSetter{
				freightClient: &testutil.MockFreightClient{
					GetFreightFun: func(
						ctx context.Context,
						project string,
						nameOrAlias string,
					) (*kargoapi.Freight, error) {
						require.Equal(t, testProject, project)
						require.Equal(t, tt.cfg.Name, nameOrAlias)
						return tt.getFreightFn(ctx, project, nameOrAlias)
					},
					PatchFreightAliasFun: func(
						_ context.Context,
						project string,
						nameOrAlias string,
						newAlias string,
					) error {
						patches = append(patches, aliasPatch{
							project:     project,
							nameOrAlias: nameOrAlias,
							newAlias:    newAlias,
						})
						return tt.patchErr
					},
				},
			}

			stepCtx := &promotion.StepContext{
				Project: testProject,
			}

			res, err := step.run(t.Context(), stepCtx, tt.cfg)
			tt.assertions(t, res, patches, err)
		})
	}
}
