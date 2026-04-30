package rbac

import (
	"testing"

	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestNormalizePolicyRules(t *testing.T) {
	t.Run("invalid resource type", func(t *testing.T) {
		_, err := NormalizePolicyRules(
			[]rbacv1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{"fake-resource"},
				Verbs:     []string{"get"},
			}},
			nil,
		)
		require.ErrorContains(t, err, "unrecognized resource type")
	})

	t.Run("singular resource type", func(t *testing.T) {
		_, err := NormalizePolicyRules(
			[]rbacv1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{"stage"},
				Verbs:     []string{"get"},
			}},
			nil,
		)
		require.ErrorContains(t, err, `unrecognized resource type "stage"`)
		require.ErrorContains(t, err, `did you mean "stages"`)
	})

	t.Run("multiple resources expand", func(t *testing.T) {
		rules, err := NormalizePolicyRules(
			[]rbacv1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{"secrets", "serviceaccounts"},
				Verbs:     []string{"get"},
			}},
			nil,
		)
		require.NoError(t, err)
		require.Equal(
			t,
			[]rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"secrets"},
					Verbs:     []string{"get"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"serviceaccounts"},
					Verbs:     []string{"get"},
				},
			},
			rules,
		)
	})

	t.Run("multiple resources names expand", func(t *testing.T) {
		rules, err := NormalizePolicyRules(
			[]rbacv1.PolicyRule{{
				APIGroups:     []string{""},
				Resources:     []string{"serviceaccounts"},
				ResourceNames: []string{"foo", "bar"},
				Verbs:         []string{"get"},
			}},
			nil,
		)
		require.NoError(t, err)
		require.Equal(
			t,
			[]rbacv1.PolicyRule{
				{
					APIGroups:     []string{""},
					Resources:     []string{"serviceaccounts"},
					ResourceNames: []string{"bar"},
					Verbs:         []string{"get"},
				},
				{
					APIGroups:     []string{""},
					Resources:     []string{"serviceaccounts"},
					ResourceNames: []string{"foo"},
					Verbs:         []string{"get"},
				},
			},
			rules,
		)
	})

	t.Run("verbs get sorted", func(t *testing.T) {
		rules, err := NormalizePolicyRules(
			[]rbacv1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{"serviceaccounts"},
				Verbs:     []string{"list", "get"},
			}},
			nil,
		)
		require.NoError(t, err)
		require.Equal(
			t,
			[]rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"serviceaccounts"},
					Verbs:     []string{"get", "list"},
				},
			},
			rules,
		)
	})

	t.Run("verbs get de-duped", func(t *testing.T) {
		rules, err := NormalizePolicyRules(
			[]rbacv1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{"serviceaccounts"},
				Verbs:     []string{"get", "get"},
			}},
			nil,
		)
		require.NoError(t, err)
		require.Equal(
			t,
			[]rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"serviceaccounts"},
					Verbs:     []string{"get"},
				},
			},
			rules,
		)
	})

	t.Run("wildcard verbs expand", func(t *testing.T) {
		rules, err := NormalizePolicyRules(
			[]rbacv1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{"serviceaccounts"},
				Verbs:     []string{"*"},
			}},
			nil,
		)
		require.NoError(t, err)
		require.Equal(
			t,
			[]rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"serviceaccounts"},
					Verbs:     allVerbs,
				},
			},
			rules,
		)
	})

	t.Run("correct groups are determined automatically", func(t *testing.T) {
		rules, err := NormalizePolicyRules(
			[]rbacv1.PolicyRule{{
				APIGroups: []string{"", "foo", "bar"},
				Resources: []string{"stages"},
				Verbs:     []string{"get"},
			}},
			nil,
		)
		require.NoError(t, err)
		require.Equal(
			t,
			[]rbacv1.PolicyRule{
				{
					APIGroups: []string{kargoapi.GroupVersion.Group},
					Resources: []string{"stages"},
					Verbs:     []string{"get"},
				},
			},
			rules,
		)
	})

	t.Run("kitchen sink", func(t *testing.T) {
		rules, err := NormalizePolicyRules(
			[]rbacv1.PolicyRule{
				{ // Never mind that this doesn't make sense. It should all get fixed
					APIGroups: []string{""},
					Resources: []string{"serviceaccounts", "stages"},
					Verbs:     []string{"*"},
				},
				{ // These should get de-duped
					APIGroups: []string{kargoapi.GroupVersion.Group},
					Resources: []string{"stages"},
					Verbs:     []string{"*"},
				},
				{
					APIGroups:     []string{kargoapi.GroupVersion.Group},
					Resources:     []string{"warehouses"},
					ResourceNames: []string{"foo", "bar"},
					Verbs:         []string{"get", "list"},
				},
			},
			&PolicyRuleNormalizationOptions{IncludeCustomVerbsInExpansion: true},
		)
		require.NoError(t, err)
		require.Equal(
			t,
			[]rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"serviceaccounts"},
					Verbs:     allVerbs,
				},
				{
					APIGroups: []string{kargoapi.GroupVersion.Group},
					Resources: []string{"stages"},
					Verbs:     allStagesVerbs,
				},
				{
					APIGroups:     []string{kargoapi.GroupVersion.Group},
					Resources:     []string{"warehouses"},
					ResourceNames: []string{"bar"},
					Verbs:         []string{"get", "list"},
				},
				{
					APIGroups:     []string{kargoapi.GroupVersion.Group},
					Resources:     []string{"warehouses"},
					ResourceNames: []string{"foo"},
					Verbs:         []string{"get", "list"},
				},
			},
			rules,
		)
	})

}
