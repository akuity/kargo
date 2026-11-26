package projectconfigs

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestValidate(t *testing.T) {
	const project = "kargo-demo"

	newCfg := func(spec kargoapi.ProjectConfigSpec) *kargoapi.ProjectConfig {
		return &kargoapi.ProjectConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      project,
				Namespace: project,
			},
			Spec: spec,
		}
	}
	github := &kargoapi.GitHubWebhookReceiverConfig{
		SecretRef: corev1.LocalObjectReference{Name: "s"},
	}
	windowErr := field.Invalid(
		field.NewPath("spec", "promotionWindows").Index(0).Child("rrule"),
		"bad",
		"invalid rrule",
	)

	testCases := []struct {
		name       string
		cfg        *kargoapi.ProjectConfig
		create     bool
		validators []SpecValidatorRegistration
		assert     func(*testing.T, field.ErrorList)
	}{
		{
			name:   "valid",
			cfg:    newCfg(kargoapi.ProjectConfigSpec{}),
			create: true,
			assert: func(t *testing.T, errs field.ErrorList) {
				require.Empty(t, errs)
			},
		},
		{
			name: "name does not match Project on create",
			cfg: &kargoapi.ProjectConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "other"},
			},
			create: true,
			assert: func(t *testing.T, errs field.ErrorList) {
				requirePaths(t, errs, "metadata.name")
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
			},
		},
		{
			name: "namespace does not match Project on create",
			cfg: &kargoapi.ProjectConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      project,
					Namespace: "other",
				},
			},
			create: true,
			assert: func(t *testing.T, errs field.ErrorList) {
				requirePaths(t, errs, "metadata.namespace")
			},
		},
		{
			name: "empty namespace is accepted on create",
			cfg: &kargoapi.ProjectConfig{
				ObjectMeta: metav1.ObjectMeta{Name: project},
			},
			create: true,
			assert: func(t *testing.T, errs field.ErrorList) {
				require.Empty(t, errs)
			},
		},
		{
			name: "metadata rules skipped on update",
			cfg: &kargoapi.ProjectConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "other",
					Namespace: "other",
				},
			},
			create: false,
			assert: func(t *testing.T, errs field.ErrorList) {
				require.Empty(t, errs)
			},
		},
		{
			name: "invalid label key on create",
			cfg: &kargoapi.ProjectConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:   project,
					Labels: map[string]string{"bad key!": "v"},
				},
			},
			create: true,
			assert: func(t *testing.T, errs field.ErrorList) {
				requirePaths(t, errs, "metadata.labels")
			},
		},
		{
			name: "invalid label value on update",
			cfg: &kargoapi.ProjectConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:   project,
					Labels: map[string]string{"k": "bad value!"},
				},
			},
			create: false,
			assert: func(t *testing.T, errs field.ErrorList) {
				requirePaths(t, errs, "metadata.labels")
			},
		},
		{
			name: "invalid annotation key on update",
			cfg: &kargoapi.ProjectConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:        project,
					Annotations: map[string]string{"a/b/c": "v"},
				},
			},
			create: false,
			assert: func(t *testing.T, errs field.ErrorList) {
				requirePaths(t, errs, "metadata.annotations")
			},
		},
		{
			name: "annotations too large",
			cfg: &kargoapi.ProjectConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: project,
					Annotations: map[string]string{
						"k": strings.Repeat("x", 256*1024),
					},
				},
			},
			create: true,
			assert: func(t *testing.T, errs field.ErrorList) {
				requirePaths(t, errs, "metadata.annotations")
				require.Equal(t, field.ErrorTypeTooLong, errs[0].Type)
			},
		},
		{
			name: "valid labels and annotations",
			cfg: &kargoapi.ProjectConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:        project,
					Labels:      map[string]string{"example.com/team": "a-1"},
					Annotations: map[string]string{"example.com/note": "any text!"},
				},
			},
			create: true,
			assert: func(t *testing.T, errs field.ErrorList) {
				require.Empty(t, errs)
			},
		},
		{
			name: "duplicate Stage",
			cfg: newCfg(kargoapi.ProjectConfigSpec{
				PromotionPolicies: []kargoapi.PromotionPolicy{
					{Stage: "test"},
					{StageSelector: &kargoapi.PromotionPolicySelector{Name: "test"}},
				},
			}),
			assert: func(t *testing.T, errs field.ErrorList) {
				requirePaths(t, errs, "spec.promotionPolicies[1]")
				require.Contains(t, errs[0].Detail, "spec.promotionPolicies[0]")
			},
		},
		{
			name: "invalid stage selector pattern",
			cfg: newCfg(kargoapi.ProjectConfigSpec{
				PromotionPolicies: []kargoapi.PromotionPolicy{
					{StageSelector: &kargoapi.PromotionPolicySelector{Name: "regex:["}},
					{StageSelector: &kargoapi.PromotionPolicySelector{Name: "foo:bar"}},
				},
			}),
			assert: func(t *testing.T, errs field.ErrorList) {
				requirePaths(
					t,
					errs,
					"spec.promotionPolicies[0].stageSelector.name",
					"spec.promotionPolicies[1].stageSelector.name",
				)
			},
		},
		{
			name: "PromotionPolicy one-of rule",
			cfg: newCfg(kargoapi.ProjectConfigSpec{
				PromotionPolicies: []kargoapi.PromotionPolicy{
					{
						Stage:         "a",
						StageSelector: &kargoapi.PromotionPolicySelector{Name: "b"},
					},
					{AutoPromotionEnabled: true},
				},
			}),
			assert: func(t *testing.T, errs field.ErrorList) {
				requirePaths(
					t,
					errs,
					"spec.promotionPolicies[0]",
					"spec.promotionPolicies[1]",
				)
				require.Equal(t, policyOneOfMessage, errs[0].Detail)
			},
		},
		{
			name: "stage name pattern",
			cfg: newCfg(kargoapi.ProjectConfigSpec{
				PromotionPolicies: []kargoapi.PromotionPolicy{{Stage: "Bad_Name"}},
			}),
			assert: func(t *testing.T, errs field.ErrorList) {
				requirePaths(t, errs, "spec.promotionPolicies[0].stage")
			},
		},
		{
			name: "autoRollback values, size, and uniqueness",
			cfg: newCfg(kargoapi.ProjectConfigSpec{
				PromotionPolicies: []kargoapi.PromotionPolicy{
					{
						Stage: "a",
						AutoRollback: &kargoapi.AutoRollbackConfig{
							OnPromotion: []kargoapi.PromotionPhase{
								kargoapi.PromotionPhaseSucceeded,
							},
							OnVerification: []kargoapi.VerificationPhase{
								kargoapi.VerificationPhaseError,
								kargoapi.VerificationPhaseError,
							},
						},
					},
					{
						Stage: "b",
						AutoRollback: &kargoapi.AutoRollbackConfig{
							OnPromotion: []kargoapi.PromotionPhase{
								kargoapi.PromotionPhaseFailed,
								kargoapi.PromotionPhaseErrored,
								kargoapi.PromotionPhaseFailed,
							},
						},
					},
				},
			}),
			assert: func(t *testing.T, errs field.ErrorList) {
				requirePaths(
					t,
					errs,
					"spec.promotionPolicies[0].autoRollback.onPromotion",
					"spec.promotionPolicies[0].autoRollback.onVerification[1]",
					"spec.promotionPolicies[1].autoRollback.onPromotion",
					"spec.promotionPolicies[1].autoRollback.onPromotion[2]",
				)
				require.Equal(
					t,
					"onPromotion[0] must be Failed or Errored",
					errs[0].Detail,
				)
				require.Equal(t, field.ErrorTypeDuplicate, errs[1].Type)
				require.Equal(t, field.ErrorTypeTooMany, errs[2].Type)
				require.Equal(t, field.ErrorTypeDuplicate, errs[3].Type)
			},
		},
		{
			name: "receiver errors",
			cfg: newCfg(kargoapi.ProjectConfigSpec{
				WebhookReceivers: []kargoapi.WebhookReceiverConfig{
					{Name: "dup", GitHub: github},
					{Name: "dup", GitHub: github},
					{Name: strings.Repeat("a", 254), GitHub: github},
					{Name: "Bad_Name", GitHub: github},
					{GitHub: github},
				},
			}),
			assert: func(t *testing.T, errs field.ErrorList) {
				requirePaths(
					t,
					errs,
					"spec.webhookReceivers[2].name",
					"spec.webhookReceivers[3].name",
					"spec.webhookReceivers[4].name",
					"spec.webhookReceivers[1].name",
				)
				require.Equal(t, field.ErrorTypeTooLong, errs[0].Type)
				require.Equal(t, field.ErrorTypeRequired, errs[2].Type)
				require.Contains(t, errs[3].Detail, "already defined")
			},
		},
		{
			name: "generic receiver schema rules",
			cfg: newCfg(kargoapi.ProjectConfigSpec{
				WebhookReceivers: []kargoapi.WebhookReceiverConfig{
					{
						Name: "a",
						Generic: &kargoapi.GenericWebhookReceiverConfig{
							Actions: []kargoapi.GenericWebhookAction{},
						},
					},
					{
						Name: "b",
						Generic: &kargoapi.GenericWebhookReceiverConfig{
							Actions: []kargoapi.GenericWebhookAction{
								{
									ActionType: "Delete",
									TargetSelectionCriteria: []kargoapi.GenericWebhookTargetSelectionCriteria{{
										Kind: kargoapi.GenericWebhookTargetKindPromotion,
										Name: "p",
										IndexSelector: kargoapi.IndexSelector{
											MatchIndices: []kargoapi.IndexSelectorRequirement{{
												Key:      "bogus",
												Operator: kargoapi.IndexSelectorOperatorEqual,
												Value:    "x",
											}},
										},
									}},
								},
								{
									ActionType:              kargoapi.GenericWebhookActionTypeRefresh,
									TargetSelectionCriteria: []kargoapi.GenericWebhookTargetSelectionCriteria{},
								},
							},
						},
					},
				},
			}),
			assert: func(t *testing.T, errs field.ErrorList) {
				requirePaths(
					t,
					errs,
					"spec.webhookReceivers[0].generic.actions",
					"spec.webhookReceivers[1].generic.actions[0].action",
					"spec.webhookReceivers[1].generic.actions[0].targetSelectionCriteria[0].kind",
					"spec.webhookReceivers[1].generic.actions[0].targetSelectionCriteria[0]"+
						".indexSelector.matchIndices[0].key",
					"spec.webhookReceivers[1].generic.actions[1].targetSelectionCriteria",
				)
				require.Equal(t, field.ErrorTypeNotSupported, errs[1].Type)
			},
		},
		{
			name: "deep links",
			cfg: newCfg(kargoapi.ProjectConfigSpec{
				FreightLinks: []kargoapi.DeepLink{{URL: "https://example.com"}},
				StageLinks:   []kargoapi.DeepLink{{Title: "t"}},
			}),
			assert: func(t *testing.T, errs field.ErrorList) {
				requirePaths(
					t,
					errs,
					"spec.freightLinks[0].title",
					"spec.stageLinks[0].url",
				)
			},
		},
		{
			name: "promotion windows",
			cfg: newCfg(kargoapi.ProjectConfigSpec{
				PromotionWindows: []kargoapi.PromotionWindow{
					{Name: "w", Kind: kargoapi.PromotionWindowKindAllow},
					{
						Name:        "w",
						Kind:        "Maybe",
						Description: strings.Repeat("d", 1025),
					},
					{Name: "Bad_Name", Kind: kargoapi.PromotionWindowKindDeny},
				},
			}),
			assert: func(t *testing.T, errs field.ErrorList) {
				requirePaths(
					t,
					errs,
					"spec.promotionWindows[1].description",
					"spec.promotionWindows[1].kind",
					"spec.promotionWindows[2].name",
					"spec.promotionWindows[1]",
				)
				require.Equal(t, field.ErrorTypeDuplicate, errs[3].Type)
			},
		},
		{
			name: "registered validator runs after built-ins",
			cfg:  newCfg(kargoapi.ProjectConfigSpec{}),
			validators: []SpecValidatorRegistration{
				{
					Name: "first",
					Value: func(
						_ context.Context,
						p string,
						spec *kargoapi.ProjectConfigSpec,
					) field.ErrorList {
						if p != project || spec == nil {
							return field.ErrorList{field.InternalError(nil, nil)}
						}
						return field.ErrorList{windowErr}
					},
				},
				{
					Name: "second",
					Value: func(
						context.Context,
						string,
						*kargoapi.ProjectConfigSpec,
					) field.ErrorList {
						return field.ErrorList{
							field.Forbidden(field.NewPath("spec", "x"), "no"),
						}
					},
				},
			},
			assert: func(t *testing.T, errs field.ErrorList) {
				requirePaths(
					t,
					errs,
					"spec.promotionWindows[0].rrule",
					"spec.x",
				)
			},
		},
		{
			name: "registered validator skipped when built-ins fail",
			cfg: newCfg(kargoapi.ProjectConfigSpec{
				PromotionPolicies: []kargoapi.PromotionPolicy{{}},
			}),
			validators: []SpecValidatorRegistration{{
				Name: "never",
				Value: func(
					context.Context,
					string,
					*kargoapi.ProjectConfigSpec,
				) field.ErrorList {
					return field.ErrorList{windowErr}
				},
			}},
			assert: func(t *testing.T, errs field.ErrorList) {
				requirePaths(t, errs, "spec.promotionPolicies[0]")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			registry := MustNewSpecValidatorRegistry(testCase.validators...)
			testCase.assert(
				t,
				validate(
					t.Context(),
					project,
					testCase.cfg,
					testCase.create,
					registry,
				),
			)
		})
	}
}

func TestValidateUsesDefaultRegistry(t *testing.T) {
	// Validate must consult DefaultSpecValidatorRegistry. An empty default
	// registry contributes no errors to a valid ProjectConfig.
	cfg := &kargoapi.ProjectConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "kargo-demo"},
	}
	require.Empty(t, Validate(t.Context(), "kargo-demo", cfg, true))
	require.Empty(t, DefaultSpecValidatorRegistry.List())
}

func TestSpecValidatorRegistry(t *testing.T) {
	noop := func(
		context.Context,
		string,
		*kargoapi.ProjectConfigSpec,
	) field.ErrorList {
		return nil
	}

	t.Run("lists registrations in registration order", func(t *testing.T) {
		t.Parallel()
		r := MustNewSpecValidatorRegistry()
		for _, name := range []string{"c", "a", "b"} {
			r.MustRegister(SpecValidatorRegistration{Name: name, Value: noop})
		}
		var names []string
		for _, reg := range r.List() {
			names = append(names, reg.Name)
		}
		require.Equal(t, []string{"c", "a", "b"}, names)

		reg, err := r.Get("a")
		require.NoError(t, err)
		require.Equal(t, "a", reg.Name)
	})

	t.Run("rejects duplicate names", func(t *testing.T) {
		t.Parallel()
		r := MustNewSpecValidatorRegistry(
			SpecValidatorRegistration{Name: "a", Value: noop},
		)
		err := r.Register(SpecValidatorRegistration{Name: "a", Value: noop})
		require.Error(t, err)
		require.Len(t, r.List(), 1)
	})

	t.Run("rejects empty names", func(t *testing.T) {
		t.Parallel()
		r := MustNewSpecValidatorRegistry()
		require.Error(t, r.Register(SpecValidatorRegistration{Value: noop}))
		require.Empty(t, r.List())
	})
}

func requirePaths(t *testing.T, errs field.ErrorList, paths ...string) {
	t.Helper()
	got := make([]string, len(errs))
	for i, e := range errs {
		got[i] = e.Field
	}
	require.Equal(t, paths, got)
}
