package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestDefault(t *testing.T) {
	const testNamespace = "fake-namespace"
	e := Environment{
		ObjectMeta: v1.ObjectMeta{
			Name:      "fake-stage-env",
			Namespace: testNamespace,
		},
		Spec: &EnvironmentSpec{
			Subscriptions: &Subscriptions{
				UpstreamEnvs: []EnvironmentSubscription{
					{
						Name: "fake-test-env",
					},
				},
			},
			PromotionMechanisms: &PromotionMechanisms{
				ArgoCDAppUpdates: []ArgoCDAppUpdate{
					{
						AppName: "fake-prod-app",
					},
				},
			},
			HealthChecks: &HealthChecks{
				ArgoCDAppChecks: []ArgoCDAppCheck{
					{
						AppName: "fake-prod-app",
					},
				},
			},
		},
	}
	e.Default()
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
		spec       *EnvironmentSpec
		assertions func(*EnvironmentSpec, field.ErrorList)
	}{
		{
			name: "nil",
			assertions: func(_ *EnvironmentSpec, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},

		{
			name: "invalid",
			spec: &EnvironmentSpec{
				// Has two conflicting types of subs...
				Subscriptions: &Subscriptions{
					Repos: &RepoSubscriptions{},
					UpstreamEnvs: []EnvironmentSubscription{
						{},
					},
				},
				// Doesn't actually define any mechanisms...
				PromotionMechanisms: &PromotionMechanisms{},
			},
			assertions: func(spec *EnvironmentSpec, errs field.ErrorList) {
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
			spec: &EnvironmentSpec{
				// Nil subs and promo mechanisms are caught by declarative validation,
				// so for the purposes of this test, leaving those completely undefined
				// should surface no errors.
			},
			assertions: func(_ *EnvironmentSpec, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			e := &Environment{}
			testCase.assertions(
				testCase.spec,
				e.validateSpec(
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
		subs       *Subscriptions
		assertions func(*Subscriptions, field.ErrorList)
	}{
		{
			name: "nil",
			assertions: func(_ *Subscriptions, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},

		{
			name: "no subscriptions",
			subs: &Subscriptions{},
			assertions: func(subs *Subscriptions, errs field.ErrorList) {
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
			subs: &Subscriptions{
				Repos: &RepoSubscriptions{},
				UpstreamEnvs: []EnvironmentSubscription{
					{},
				},
			},
			assertions: func(subs *Subscriptions, errs field.ErrorList) {
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
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			e := &Environment{}
			testCase.assertions(
				testCase.subs,
				e.validateSubs(
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
		subs       *RepoSubscriptions
		assertions func(*RepoSubscriptions, field.ErrorList)
	}{
		{
			name: "nil",
			assertions: func(_ *RepoSubscriptions, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},

		{
			name: "no subscriptions",
			subs: &RepoSubscriptions{}, // Has no subs
			assertions: func(subs *RepoSubscriptions, errs field.ErrorList) {
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
			subs: &RepoSubscriptions{
				Images: []ImageSubscription{
					{
						SemverConstraint: "bogus",
						Platform:         "bogus",
					},
				},
				Charts: []ChartSubscription{
					{
						SemverConstraint: "bogus",
					},
				},
			},
			assertions: func(subs *RepoSubscriptions, errs field.ErrorList) {
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
			subs: &RepoSubscriptions{
				Images: []ImageSubscription{
					{},
				},
			},
			assertions: func(subs *RepoSubscriptions, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			e := &Environment{}
			testCase.assertions(
				testCase.subs,
				e.validateRepoSubs(field.NewPath("repos"), testCase.subs),
			)
		})
	}
}

func TestValidateImageSubs(t *testing.T) {
	testCases := []struct {
		name       string
		sub        ImageSubscription
		assertions func(field.ErrorList)
	}{
		{
			name: "invalid",
			sub: ImageSubscription{
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
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			e := &Environment{}
			testCase.assertions(
				e.validateImageSubs(
					field.NewPath("images"),
					[]ImageSubscription{
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
		sub        ImageSubscription
		assertions func(field.ErrorList)
	}{
		{
			name: "invalid",
			sub: ImageSubscription{
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
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			e := &Environment{}
			testCase.assertions(
				e.validateImageSub(
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
		sub        ChartSubscription
		assertions func(field.ErrorList)
	}{
		{
			name: "invalid",
			sub: ChartSubscription{
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
			sub:  ChartSubscription{},
			assertions: func(errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			e := &Environment{}
			testCase.assertions(
				e.validateChartSubs(
					field.NewPath("charts"),
					[]ChartSubscription{
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
		sub        ChartSubscription
		assertions func(field.ErrorList)
	}{
		{
			name: "invalid",
			sub: ChartSubscription{
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
			sub:  ChartSubscription{},
			assertions: func(errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			e := &Environment{}
			testCase.assertions(
				e.validateChartSub(
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
		promoMechs *PromotionMechanisms
		assertions func(*PromotionMechanisms, field.ErrorList)
	}{
		{
			name: "nil",
			assertions: func(_ *PromotionMechanisms, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},

		{
			name: "invalid",
			// Does not define any mechanisms
			promoMechs: &PromotionMechanisms{},
			assertions: func(promoMechs *PromotionMechanisms, errs field.ErrorList) {
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
			promoMechs: &PromotionMechanisms{
				GitRepoUpdates: []GitRepoUpdate{
					{
						Kustomize: &KustomizePromotionMechanism{},
					},
				},
			},
			assertions: func(_ *PromotionMechanisms, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			e := &Environment{}
			testCase.assertions(
				testCase.promoMechs,
				e.validatePromotionMechanisms(
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
		update     GitRepoUpdate
		assertions func(GitRepoUpdate, field.ErrorList)
	}{
		{
			name:   "no config management tools specified",
			update: GitRepoUpdate{},
			assertions: func(update GitRepoUpdate, errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "gitRepoUpdates[0]",
							BadValue: update,
							Detail: "exactly one of gitRepoUpdates[0].bookkeeper, or " +
								"gitRepoUpdates[0].kustomize, or gitRepoUpdates[0].helm " +
								"must be defined",
						},
					},
					errs,
				)
			},
		},

		{
			name: "more than one config management tool specified",
			update: GitRepoUpdate{
				Bookkeeper: &BookkeeperPromotionMechanism{},
				Kustomize:  &KustomizePromotionMechanism{},
				Helm:       &HelmPromotionMechanism{},
			},
			assertions: func(update GitRepoUpdate, errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "gitRepoUpdates[0]",
							BadValue: update,
							Detail: "exactly one of gitRepoUpdates[0].bookkeeper, or " +
								"gitRepoUpdates[0].kustomize, or gitRepoUpdates[0].helm " +
								"must be defined",
						},
					},
					errs,
				)
			},
		},

		{
			name: "valid",
			update: GitRepoUpdate{
				Kustomize: &KustomizePromotionMechanism{},
			},
			assertions: func(_ GitRepoUpdate, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			e := &Environment{}
			testCase.assertions(
				testCase.update,
				e.validateGitRepoUpdates(
					field.NewPath("gitRepoUpdates"),
					[]GitRepoUpdate{
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
		update     GitRepoUpdate
		assertions func(GitRepoUpdate, field.ErrorList)
	}{
		{
			name:   "no config management tools specified",
			update: GitRepoUpdate{},
			assertions: func(update GitRepoUpdate, errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "gitRepoUpdate",
							BadValue: update,
							Detail: "exactly one of gitRepoUpdate.bookkeeper, or " +
								"gitRepoUpdate.kustomize, or gitRepoUpdate.helm must be " +
								"defined",
						},
					},
					errs,
				)
			},
		},

		{
			name: "more than one config management tool specified",
			update: GitRepoUpdate{
				Bookkeeper: &BookkeeperPromotionMechanism{},
				Kustomize:  &KustomizePromotionMechanism{},
				Helm:       &HelmPromotionMechanism{},
			},
			assertions: func(update GitRepoUpdate, errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "gitRepoUpdate",
							BadValue: update,
							Detail: "exactly one of gitRepoUpdate.bookkeeper, or " +
								"gitRepoUpdate.kustomize, or gitRepoUpdate.helm must be " +
								"defined",
						},
					},
					errs,
				)
			},
		},

		{
			name: "valid",
			update: GitRepoUpdate{
				Kustomize: &KustomizePromotionMechanism{},
			},
			assertions: func(_ GitRepoUpdate, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			e := &Environment{}
			testCase.assertions(
				testCase.update,
				e.validateGitRepoUpdate(
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
		promoMech  *HelmPromotionMechanism
		assertions func(*HelmPromotionMechanism, field.ErrorList)
	}{
		{
			name: "nil",
			assertions: func(_ *HelmPromotionMechanism, errs field.ErrorList) {
				require.Empty(t, errs)
			},
		},

		{
			name: "invalid",
			// Doesn't define any changes
			promoMech: &HelmPromotionMechanism{},
			assertions: func(
				promoMech *HelmPromotionMechanism,
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
			promoMech: &HelmPromotionMechanism{
				Images: []HelmImageUpdate{
					{},
				},
			},
			assertions: func(_ *HelmPromotionMechanism, errs field.ErrorList) {
				require.Empty(t, errs)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			e := &Environment{}
			testCase.assertions(
				testCase.promoMech,
				e.validateHelmPromotionMechanism(
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
