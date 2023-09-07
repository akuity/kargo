package stage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestDefault(t *testing.T) {
	const testNamespace = "fake-namespace"
	e := &kargoapi.Stage{
		ObjectMeta: v1.ObjectMeta{
			Name:      "fake-uat-stage",
			Namespace: testNamespace,
		},
		Spec: &kargoapi.StageSpec{
			Subscriptions: &kargoapi.Subscriptions{
				UpstreamStages: []kargoapi.StageSubscription{
					{
						Name: "fake-test-stage",
					},
				},
			},
			PromotionMechanisms: &kargoapi.PromotionMechanisms{
				ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{
					{
						AppName: "fake-prod-app",
					},
				},
			},
		},
	}
	err := (&webhook{}).Default(context.Background(), e)
	require.NoError(t, err)
	require.Len(t, e.Spec.Subscriptions.UpstreamStages, 1)
	require.Len(t, e.Spec.PromotionMechanisms.ArgoCDAppUpdates, 1)
}

func TestValidateSpec(t *testing.T) {
	testCases := []struct {
		name       string
		spec       *kargoapi.StageSpec
		assertions func(*kargoapi.StageSpec, field.ErrorList)
	}{
		{
			name: "nil",
			assertions: func(_ *kargoapi.StageSpec, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},

		{
			name: "invalid",
			spec: &kargoapi.StageSpec{
				// Has two conflicting types of subs...
				Subscriptions: &kargoapi.Subscriptions{
					Repos: &kargoapi.RepoSubscriptions{},
					UpstreamStages: []kargoapi.StageSubscription{
						{},
					},
				},
				// Doesn't actually define any mechanisms...
				PromotionMechanisms: &kargoapi.PromotionMechanisms{},
			},
			assertions: func(spec *kargoapi.StageSpec, errs field.ErrorList) {
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
								"spec.subscriptions.upstreamStages must be defined",
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
			spec: &kargoapi.StageSpec{
				// Nil subs and promo mechanisms are caught by declarative validation,
				// so for the purposes of this test, leaving those completely undefined
				// should surface no errors.
			},
			assertions: func(_ *kargoapi.StageSpec, errs field.ErrorList) {
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
		subs       *kargoapi.Subscriptions
		assertions func(*kargoapi.Subscriptions, field.ErrorList)
	}{
		{
			name: "nil",
			assertions: func(_ *kargoapi.Subscriptions, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},

		{
			name: "no subscriptions",
			subs: &kargoapi.Subscriptions{},
			assertions: func(subs *kargoapi.Subscriptions, errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "subscriptions",
							BadValue: subs,
							Detail: "exactly one of subscriptions.repos or " +
								"subscriptions.upstreamStages must be defined",
						},
					},
					errs,
				)
			},
		},

		{
			name: "has repo subs and Stage subs", // Should be "one of"
			subs: &kargoapi.Subscriptions{
				Repos: &kargoapi.RepoSubscriptions{},
				UpstreamStages: []kargoapi.StageSubscription{
					{},
				},
			},
			assertions: func(subs *kargoapi.Subscriptions, errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "subscriptions",
							BadValue: subs,
							Detail: "exactly one of subscriptions.repos or " +
								"subscriptions.upstreamStages must be defined",
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
		subs       *kargoapi.RepoSubscriptions
		assertions func(*kargoapi.RepoSubscriptions, field.ErrorList)
	}{
		{
			name: "nil",
			assertions: func(_ *kargoapi.RepoSubscriptions, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},

		{
			name: "no subscriptions",
			subs: &kargoapi.RepoSubscriptions{}, // Has no subs
			assertions: func(subs *kargoapi.RepoSubscriptions, errs field.ErrorList) {
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
			subs: &kargoapi.RepoSubscriptions{
				Images: []kargoapi.ImageSubscription{
					{
						SemverConstraint: "bogus",
						Platform:         "bogus",
					},
				},
				Charts: []kargoapi.ChartSubscription{
					{
						SemverConstraint: "bogus",
					},
				},
			},
			assertions: func(subs *kargoapi.RepoSubscriptions, errs field.ErrorList) {
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
			subs: &kargoapi.RepoSubscriptions{
				Images: []kargoapi.ImageSubscription{
					{},
				},
			},
			assertions: func(subs *kargoapi.RepoSubscriptions, errs field.ErrorList) {
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
		sub        kargoapi.ImageSubscription
		assertions func(field.ErrorList)
	}{
		{
			name: "invalid",
			sub: kargoapi.ImageSubscription{
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
					[]kargoapi.ImageSubscription{
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
		sub        kargoapi.ImageSubscription
		assertions func(field.ErrorList)
	}{
		{
			name: "invalid",
			sub: kargoapi.ImageSubscription{
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
		sub        kargoapi.ChartSubscription
		assertions func(field.ErrorList)
	}{
		{
			name: "invalid",
			sub: kargoapi.ChartSubscription{
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
			sub:  kargoapi.ChartSubscription{},
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
					[]kargoapi.ChartSubscription{
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
		sub        kargoapi.ChartSubscription
		assertions func(field.ErrorList)
	}{
		{
			name: "invalid",
			sub: kargoapi.ChartSubscription{
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
			sub:  kargoapi.ChartSubscription{},
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
		promoMechs *kargoapi.PromotionMechanisms
		assertions func(*kargoapi.PromotionMechanisms, field.ErrorList)
	}{
		{
			name: "nil",
			assertions: func(_ *kargoapi.PromotionMechanisms, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},

		{
			name: "invalid",
			// Does not define any mechanisms
			promoMechs: &kargoapi.PromotionMechanisms{},
			assertions: func(
				promoMechs *kargoapi.PromotionMechanisms,
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
			promoMechs: &kargoapi.PromotionMechanisms{
				GitRepoUpdates: []kargoapi.GitRepoUpdate{
					{
						Kustomize: &kargoapi.KustomizePromotionMechanism{},
					},
				},
			},
			assertions: func(_ *kargoapi.PromotionMechanisms, errs field.ErrorList) {
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
		update     kargoapi.GitRepoUpdate
		assertions func(kargoapi.GitRepoUpdate, field.ErrorList)
	}{
		{
			name: "more than one config management tool specified",
			update: kargoapi.GitRepoUpdate{
				Bookkeeper: &kargoapi.BookkeeperPromotionMechanism{},
				Kustomize:  &kargoapi.KustomizePromotionMechanism{},
				Helm:       &kargoapi.HelmPromotionMechanism{},
			},
			assertions: func(update kargoapi.GitRepoUpdate, errs field.ErrorList) {
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
			update: kargoapi.GitRepoUpdate{
				Kustomize: &kargoapi.KustomizePromotionMechanism{},
			},
			assertions: func(_ kargoapi.GitRepoUpdate, errs field.ErrorList) {
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
					[]kargoapi.GitRepoUpdate{
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
		update     kargoapi.GitRepoUpdate
		assertions func(kargoapi.GitRepoUpdate, field.ErrorList)
	}{
		{
			name: "more than one config management tool specified",
			update: kargoapi.GitRepoUpdate{
				Bookkeeper: &kargoapi.BookkeeperPromotionMechanism{},
				Kustomize:  &kargoapi.KustomizePromotionMechanism{},
				Helm:       &kargoapi.HelmPromotionMechanism{},
			},
			assertions: func(update kargoapi.GitRepoUpdate, errs field.ErrorList) {
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
			update: kargoapi.GitRepoUpdate{
				Kustomize: &kargoapi.KustomizePromotionMechanism{},
			},
			assertions: func(_ kargoapi.GitRepoUpdate, errs field.ErrorList) {
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
		promoMech  *kargoapi.HelmPromotionMechanism
		assertions func(*kargoapi.HelmPromotionMechanism, field.ErrorList)
	}{
		{
			name: "nil",
			assertions: func(_ *kargoapi.HelmPromotionMechanism, errs field.ErrorList) {
				require.Empty(t, errs)
			},
		},

		{
			name: "invalid",
			// Doesn't define any changes
			promoMech: &kargoapi.HelmPromotionMechanism{},
			assertions: func(
				promoMech *kargoapi.HelmPromotionMechanism,
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
			promoMech: &kargoapi.HelmPromotionMechanism{
				Images: []kargoapi.HelmImageUpdate{
					{},
				},
			},
			assertions: func(_ *kargoapi.HelmPromotionMechanism, errs field.ErrorList) {
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
