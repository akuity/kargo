package environments

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	api "github.com/akuity/kargo/api/v1alpha1"
)

func TestDefault(t *testing.T) {
	const testNamespace = "fake-namespace"
	e := &api.Environment{
		ObjectMeta: v1.ObjectMeta{
			Name:      "fake-stage-env",
			Namespace: testNamespace,
		},
		Spec: &api.EnvironmentSpec{
			Subscriptions: &api.Subscriptions{
				UpstreamEnvs: []api.EnvironmentSubscription{
					{
						Name: "fake-test-env",
					},
				},
			},
			PromotionMechanisms: &api.PromotionMechanisms{
				ArgoCDAppUpdates: []api.ArgoCDAppUpdate{
					{
						AppName: "fake-prod-app",
					},
				},
			},
			HealthChecks: &api.HealthChecks{
				ArgoCDAppChecks: []api.ArgoCDAppCheck{
					{
						AppName: "fake-prod-app",
					},
				},
			},
		},
	}
	err := (&webhook{}).Default(context.Background(), e)
	require.NoError(t, err)
	require.Len(t, e.Spec.Subscriptions.UpstreamEnvs, 1)
	require.Equal(
		t,
		testNamespace,
		e.Spec.Subscriptions.UpstreamEnvs[0].Namespace,
	)
	require.Len(t, e.Spec.PromotionMechanisms.ArgoCDAppUpdates, 1)
	require.Equal(
		t,
		testNamespace,
		e.Spec.PromotionMechanisms.ArgoCDAppUpdates[0].AppNamespace,
	)
	require.Len(t, e.Spec.HealthChecks.ArgoCDAppChecks, 1)
	require.Equal(
		t,
		testNamespace,
		e.Spec.HealthChecks.ArgoCDAppChecks[0].AppNamespace)

}

func TestValidateSpec(t *testing.T) {
	testCases := []struct {
		name       string
		spec       *api.EnvironmentSpec
		assertions func(*api.EnvironmentSpec, field.ErrorList)
	}{
		{
			name: "nil",
			assertions: func(_ *api.EnvironmentSpec, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},

		{
			name: "invalid",
			spec: &api.EnvironmentSpec{
				// Has two conflicting types of subs...
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{},
					UpstreamEnvs: []api.EnvironmentSubscription{
						{},
					},
				},
				// Doesn't actually define any mechanisms...
				PromotionMechanisms: &api.PromotionMechanisms{},
			},
			assertions: func(spec *api.EnvironmentSpec, errs field.ErrorList) {
				// We really want to see that all underlying errors have been bubbled up
				// to this level and been aggregated.
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "spec.subscriptions",
							BadValue: spec.Subscriptions,
							Detail: "exactly one of spec.subscriptions.repos or " +
								"spec.subscriptions.upstreamEnvs must be defined",
						},
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "spec.promotionMechanisms",
							BadValue: spec.PromotionMechanisms,
							Detail: "at least one of " +
								"spec.promotionMechanisms.gitRepoUpdates or " +
								"spec.promotionMechanisms.argoCDAppUpdates must be non-empty",
						},
					},
					errs,
				)
			},
		},

		{
			name: "valid",
			spec: &api.EnvironmentSpec{
				// Nil subs and promo mechanisms are caught by declarative validation,
				// so for the purposes of this test, leaving those completely undefined
				// should surface no errors.
			},
			assertions: func(_ *api.EnvironmentSpec, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.spec,
				w.validateSpec(
					field.NewPath("spec"),
					testCase.spec,
				),
			)
		})
	}
}

func TestValidateSubs(t *testing.T) {
	testCases := []struct {
		name       string
		subs       *api.Subscriptions
		assertions func(*api.Subscriptions, field.ErrorList)
	}{
		{
			name: "nil",
			assertions: func(_ *api.Subscriptions, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},

		{
			name: "no subscriptions",
			subs: &api.Subscriptions{},
			assertions: func(subs *api.Subscriptions, errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "subscriptions",
							BadValue: subs,
							Detail: "exactly one of subscriptions.repos or " +
								"subscriptions.upstreamEnvs must be defined",
						},
					},
					errs,
				)
			},
		},

		{
			name: "has repo subs and env subs", // Should be "one of"
			subs: &api.Subscriptions{
				Repos: &api.RepoSubscriptions{},
				UpstreamEnvs: []api.EnvironmentSubscription{
					{},
				},
			},
			assertions: func(subs *api.Subscriptions, errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "subscriptions",
							BadValue: subs,
							Detail: "exactly one of subscriptions.repos or " +
								"subscriptions.upstreamEnvs must be defined",
						},
					},
					errs,
				)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.subs,
				w.validateSubs(
					field.NewPath("subscriptions"),
					testCase.subs,
				),
			)
		})
	}
}

func TestValidateRepoSubs(t *testing.T) {
	testCases := []struct {
		name       string
		subs       *api.RepoSubscriptions
		assertions func(*api.RepoSubscriptions, field.ErrorList)
	}{
		{
			name: "nil",
			assertions: func(_ *api.RepoSubscriptions, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},

		{
			name: "no subscriptions",
			subs: &api.RepoSubscriptions{}, // Has no subs
			assertions: func(subs *api.RepoSubscriptions, errs field.ErrorList) {
				require.Len(t, errs, 1)
				require.Equal(
					t,
					&field.Error{
						Type:     field.ErrorTypeInvalid,
						Field:    "repos",
						BadValue: subs,
						Detail: "at least one of repos.git, repos.images, or " +
							"repos.charts must be non-empty",
					},
					errs[0],
				)
			},
		},

		{
			name: "invalid subscriptions",
			subs: &api.RepoSubscriptions{
				Images: []api.ImageSubscription{
					{
						SemverConstraint: "bogus",
						Platform:         "bogus",
					},
				},
				Charts: []api.ChartSubscription{
					{
						SemverConstraint: "bogus",
					},
				},
			},
			assertions: func(subs *api.RepoSubscriptions, errs field.ErrorList) {
				require.Len(t, errs, 3)
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "repos.images[0].semverConstraint",
							BadValue: "bogus",
						},
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "repos.images[0].platform",
							BadValue: "bogus",
						},
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "repos.charts[0].semverConstraint",
							BadValue: "bogus",
						},
					},
					errs,
				)
			},
		},

		{
			name: "valid",
			subs: &api.RepoSubscriptions{
				Images: []api.ImageSubscription{
					{},
				},
			},
			assertions: func(subs *api.RepoSubscriptions, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.subs,
				w.validateRepoSubs(field.NewPath("repos"), testCase.subs),
			)
		})
	}
}

func TestValidateImageSubs(t *testing.T) {
	testCases := []struct {
		name       string
		sub        api.ImageSubscription
		assertions func(field.ErrorList)
	}{
		{
			name: "invalid",
			sub: api.ImageSubscription{
				SemverConstraint: "bogus",
				Platform:         "bogus",
			},
			assertions: func(errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "images[0].semverConstraint",
							BadValue: "bogus",
						},
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "images[0].platform",
							BadValue: "bogus",
						},
					},
					errs,
				)
			},
		},

		{
			name: "valid",
			assertions: func(errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				w.validateImageSubs(
					field.NewPath("images"),
					[]api.ImageSubscription{
						testCase.sub,
					},
				),
			)
		})
	}
}

func TestValidateImageSub(t *testing.T) {
	testCases := []struct {
		name       string
		sub        api.ImageSubscription
		assertions func(field.ErrorList)
	}{
		{
			name: "invalid",
			sub: api.ImageSubscription{
				SemverConstraint: "bogus",
				Platform:         "bogus",
			},
			assertions: func(errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "image.semverConstraint",
							BadValue: "bogus",
						},
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "image.platform",
							BadValue: "bogus",
						},
					},
					errs,
				)
			},
		},

		{
			name: "valid",
			assertions: func(errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				w.validateImageSub(
					field.NewPath("image"),
					testCase.sub,
				),
			)
		})
	}
}

func TestValidateChartSubs(t *testing.T) {
	testCases := []struct {
		name       string
		sub        api.ChartSubscription
		assertions func(field.ErrorList)
	}{
		{
			name: "invalid",
			sub: api.ChartSubscription{
				SemverConstraint: "bogus",
			},
			assertions: func(errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "charts[0].semverConstraint",
							BadValue: "bogus",
						},
					},
					errs,
				)
			},
		},

		{
			name: "valid",
			sub:  api.ChartSubscription{},
			assertions: func(errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				w.validateChartSubs(
					field.NewPath("charts"),
					[]api.ChartSubscription{
						testCase.sub,
					},
				),
			)
		})
	}
}

func TestValidateChartSub(t *testing.T) {
	testCases := []struct {
		name       string
		sub        api.ChartSubscription
		assertions func(field.ErrorList)
	}{
		{
			name: "invalid",
			sub: api.ChartSubscription{
				SemverConstraint: "bogus",
			},
			assertions: func(errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "chart.semverConstraint",
							BadValue: "bogus",
						},
					},
					errs,
				)
			},
		},

		{
			name: "valid",
			sub:  api.ChartSubscription{},
			assertions: func(errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				w.validateChartSub(
					field.NewPath("chart"),
					testCase.sub,
				),
			)
		})
	}
}

func TestValidatePromotionMechanisms(t *testing.T) {
	testCases := []struct {
		name       string
		promoMechs *api.PromotionMechanisms
		assertions func(*api.PromotionMechanisms, field.ErrorList)
	}{
		{
			name: "nil",
			assertions: func(_ *api.PromotionMechanisms, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},

		{
			name: "invalid",
			// Does not define any mechanisms
			promoMechs: &api.PromotionMechanisms{},
			assertions: func(
				promoMechs *api.PromotionMechanisms,
				errs field.ErrorList,
			) {
				require.NotNil(t, errs)
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "promotionMechanisms",
							BadValue: promoMechs,
							Detail: "at least one of promotionMechanisms.gitRepoUpdates or " +
								"promotionMechanisms.argoCDAppUpdates must be non-empty",
						},
					},
					errs,
				)
			},
		},

		{
			name: "valid",
			promoMechs: &api.PromotionMechanisms{
				GitRepoUpdates: []api.GitRepoUpdate{
					{
						Kustomize: &api.KustomizePromotionMechanism{},
					},
				},
			},
			assertions: func(_ *api.PromotionMechanisms, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.promoMechs,
				w.validatePromotionMechanisms(
					field.NewPath("promotionMechanisms"),
					testCase.promoMechs,
				),
			)
		})
	}
}

func TestValidateGitRepoUpdates(t *testing.T) {
	testCases := []struct {
		name       string
		update     api.GitRepoUpdate
		assertions func(api.GitRepoUpdate, field.ErrorList)
	}{
		{
			name: "more than one config management tool specified",
			update: api.GitRepoUpdate{
				Bookkeeper: &api.BookkeeperPromotionMechanism{},
				Kustomize:  &api.KustomizePromotionMechanism{},
				Helm:       &api.HelmPromotionMechanism{},
			},
			assertions: func(update api.GitRepoUpdate, errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "gitRepoUpdates[0]",
							BadValue: update,
							Detail: "no more than one of gitRepoUpdates[0].bookkeeper, or " +
								"gitRepoUpdates[0].kustomize, or gitRepoUpdates[0].helm " +
								"may be defined",
						},
					},
					errs,
				)
			},
		},

		{
			name: "valid",
			update: api.GitRepoUpdate{
				Kustomize: &api.KustomizePromotionMechanism{},
			},
			assertions: func(_ api.GitRepoUpdate, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.update,
				w.validateGitRepoUpdates(
					field.NewPath("gitRepoUpdates"),
					[]api.GitRepoUpdate{
						testCase.update,
					},
				),
			)
		})
	}
}

func TestValidateGitRepoUpdate(t *testing.T) {
	testCases := []struct {
		name       string
		update     api.GitRepoUpdate
		assertions func(api.GitRepoUpdate, field.ErrorList)
	}{
		{
			name: "more than one config management tool specified",
			update: api.GitRepoUpdate{
				Bookkeeper: &api.BookkeeperPromotionMechanism{},
				Kustomize:  &api.KustomizePromotionMechanism{},
				Helm:       &api.HelmPromotionMechanism{},
			},
			assertions: func(update api.GitRepoUpdate, errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "gitRepoUpdate",
							BadValue: update,
							Detail: "no more than one of gitRepoUpdate.bookkeeper, or " +
								"gitRepoUpdate.kustomize, or gitRepoUpdate.helm may be " +
								"defined",
						},
					},
					errs,
				)
			},
		},

		{
			name: "valid",
			update: api.GitRepoUpdate{
				Kustomize: &api.KustomizePromotionMechanism{},
			},
			assertions: func(_ api.GitRepoUpdate, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.update,
				w.validateGitRepoUpdate(
					field.NewPath("gitRepoUpdate"),
					testCase.update,
				),
			)
		})
	}
}

func TestValidateHelmPromotionMechanism(t *testing.T) {
	testCases := []struct {
		name       string
		promoMech  *api.HelmPromotionMechanism
		assertions func(*api.HelmPromotionMechanism, field.ErrorList)
	}{
		{
			name: "nil",
			assertions: func(_ *api.HelmPromotionMechanism, errs field.ErrorList) {
				require.Empty(t, errs)
			},
		},

		{
			name: "invalid",
			// Doesn't define any changes
			promoMech: &api.HelmPromotionMechanism{},
			assertions: func(
				promoMech *api.HelmPromotionMechanism,
				errs field.ErrorList,
			) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "helm",
							BadValue: promoMech,
							Detail: "at least one of helm.images or helm.charts must be " +
								"non-empty",
						},
					},
					errs,
				)
			},
		},

		{
			name: "valid",
			promoMech: &api.HelmPromotionMechanism{
				Images: []api.HelmImageUpdate{
					{},
				},
			},
			assertions: func(_ *api.HelmPromotionMechanism, errs field.ErrorList) {
				require.Empty(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.promoMech,
				w.validateHelmPromotionMechanism(
					field.NewPath("helm"),
					testCase.promoMech,
				),
			)
		})
	}
}

func TestValidateSemverConstraint(t *testing.T) {
	testCases := []struct {
		name             string
		semverConstraint string
		assertions       func(error)
	}{
		{
			name: "empty string",
			assertions: func(err error) {
				require.Nil(t, err)
			},
		},

		{
			name:             "invalid",
			semverConstraint: "bogus",
			assertions: func(err error) {
				require.NotNil(t, err)
				require.Equal(
					t,
					&field.Error{
						Type:     field.ErrorTypeInvalid,
						Field:    "semverConstraint",
						BadValue: "bogus",
					},
					err,
				)
			},
		},

		{
			name:             "valid",
			semverConstraint: "^1.0.0",
			assertions: func(err error) {
				require.Nil(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				validateSemverConstraint(
					field.NewPath("semverConstraint"),
					testCase.semverConstraint,
				),
			)
		})
	}
}
