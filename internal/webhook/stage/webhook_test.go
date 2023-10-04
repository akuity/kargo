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
					Warehouse: "test-warehouse",
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
							Detail: "exactly one of spec.subscriptions.warehouse or " +
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
							Detail: "exactly one of subscriptions.warehouse or " +
								"subscriptions.upstreamStages must be defined",
						},
					},
					errs,
				)
			},
		},

		{
			name: "has warehouse sub and Stage subs", // Should be "one of"
			subs: &kargoapi.Subscriptions{
				Warehouse: "test-warehouse",
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
							Detail: "exactly one of subscriptions.warehouse or " +
								"subscriptions.upstreamStages must be defined",
						},
					},
					errs,
				)
			},
		},

		{
			name: "success",
			subs: &kargoapi.Subscriptions{
				Warehouse: "test-warehouse",
			},
			assertions: func(_ *kargoapi.Subscriptions, errs field.ErrorList) {
				require.Nil(t, errs)
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
