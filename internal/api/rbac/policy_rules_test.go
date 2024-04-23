package rbac

import (
	"testing"

	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestNormalizePolicyRules(t *testing.T) {

	t.Run("wildcard group not allowed", func(t *testing.T) {
		_, err := NormalizePolicyRules([]rbacv1.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"pods"},
				Verbs:     []string{"get"},
			},
		})
		require.ErrorContains(t, err, "wildcard APIGroup is not allowed")
	})

	t.Run("wildcard resource not allowed", func(t *testing.T) {
		_, err := NormalizePolicyRules([]rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"*"},
				Verbs:     []string{"get"},
			},
		})
		require.ErrorContains(t, err, "wildcard Resource is not allowed")
	})

	t.Run("multiple groups expand", func(t *testing.T) {
		rules, err := NormalizePolicyRules([]rbacv1.PolicyRule{
			{ // Never mind that this doesn't make sense
				APIGroups: []string{"", rbacv1.GroupName},
				Resources: []string{"pods"},
				Verbs:     []string{"get"},
			},
		})
		require.NoError(t, err)
		require.Equal(
			t,
			[]rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"pods"},
					Verbs:     []string{"get"},
				},
				{
					APIGroups: []string{rbacv1.GroupName},
					Resources: []string{"pods"},
					Verbs:     []string{"get"},
				},
			},
			rules,
		)
	})

	t.Run("multiple resources expand", func(t *testing.T) {
		rules, err := NormalizePolicyRules([]rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods", "services"},
				Verbs:     []string{"get"},
			},
		})
		require.NoError(t, err)
		require.Equal(
			t,
			[]rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"pods"},
					Verbs:     []string{"get"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"services"},
					Verbs:     []string{"get"},
				},
			},
			rules,
		)
	})

	t.Run("multiple resources names expand", func(t *testing.T) {
		rules, err := NormalizePolicyRules([]rbacv1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{"pods"},
				ResourceNames: []string{"foo", "bar"},
				Verbs:         []string{"get"},
			},
		})
		require.NoError(t, err)
		require.Equal(
			t,
			[]rbacv1.PolicyRule{
				{
					APIGroups:     []string{""},
					Resources:     []string{"pods"},
					ResourceNames: []string{"bar"},
					Verbs:         []string{"get"},
				},
				{
					APIGroups:     []string{""},
					Resources:     []string{"pods"},
					ResourceNames: []string{"foo"},
					Verbs:         []string{"get"},
				},
			},
			rules,
		)
	})

	t.Run("verbs get sorted", func(t *testing.T) {
		rules, err := NormalizePolicyRules([]rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"list", "get"},
			},
		})
		require.NoError(t, err)
		require.Equal(
			t,
			[]rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"pods"},
					Verbs:     []string{"get", "list"},
				},
			},
			rules,
		)
	})

	t.Run("verbs get de-duped", func(t *testing.T) {
		rules, err := NormalizePolicyRules([]rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "get"},
			},
		})
		require.NoError(t, err)
		require.Equal(
			t,
			[]rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"pods"},
					Verbs:     []string{"get"},
				},
			},
			rules,
		)
	})

	t.Run("wildcard verbs expand", func(t *testing.T) {
		rules, err := NormalizePolicyRules([]rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"*"},
			},
		})
		require.NoError(t, err)
		require.Equal(
			t,
			[]rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"pods"},
					Verbs:     allVerbs,
				},
			},
			rules,
		)
	})

	t.Run("kitchen sink", func(t *testing.T) {
		rules, err := NormalizePolicyRules([]rbacv1.PolicyRule{
			{ // Never mind that this doesn't make sense
				APIGroups: []string{"", rbacv1.GroupName},
				Resources: []string{"pods", "services"},
				Verbs:     []string{"*"},
			},
			{ // These should get de-duped
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups:     []string{kargoapi.GroupVersion.Group},
				Resources:     []string{"stages"},
				ResourceNames: []string{"foo", "bar"},
				Verbs:         []string{"get", "list"},
			},
		})
		require.NoError(t, err)
		require.Equal(
			t,
			[]rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"pods"},
					Verbs:     allVerbs,
				},
				{
					APIGroups: []string{""},
					Resources: []string{"services"},
					Verbs:     allVerbs,
				},
				{
					APIGroups:     []string{kargoapi.GroupVersion.Group},
					Resources:     []string{"stages"},
					ResourceNames: []string{"bar"},
					Verbs:         []string{"get", "list"},
				},
				{
					APIGroups:     []string{kargoapi.GroupVersion.Group},
					Resources:     []string{"stages"},
					ResourceNames: []string{"foo"},
					Verbs:         []string{"get", "list"},
				},
				{
					APIGroups: []string{rbacv1.GroupName},
					Resources: []string{"pods"},
					Verbs:     allVerbs,
				},
				{
					APIGroups: []string{rbacv1.GroupName},
					Resources: []string{"services"},
					Verbs:     allVerbs,
				},
			},
			rules,
		)
	})

}
