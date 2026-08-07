package rbac

import (
	"errors"
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

	t.Run("groups are preserved", func(t *testing.T) {
		rules, err := NormalizePolicyRules(
			[]rbacv1.PolicyRule{{
				APIGroups: []string{kargoapi.GroupVersion.Group},
				Resources: []string{"stages"},
				Verbs:     []string{"get"},
			}},
			nil,
		)
		require.NoError(t, err)
		require.Equal(
			t,
			[]rbacv1.PolicyRule{{
				APIGroups: []string{kargoapi.GroupVersion.Group},
				Resources: []string{"stages"},
				Verbs:     []string{"get"},
			}},
			rules,
		)
	})

	t.Run("multiple groups expand", func(t *testing.T) {
		rules, err := NormalizePolicyRules(
			[]rbacv1.PolicyRule{{
				APIGroups: []string{"foo", "bar"},
				Resources: []string{"widgets"},
				Verbs:     []string{"get"},
			}},
			nil,
		)
		require.NoError(t, err)
		require.Equal(
			t,
			[]rbacv1.PolicyRule{
				{
					APIGroups: []string{"bar"},
					Resources: []string{"widgets"},
					Verbs:     []string{"get"},
				},
				{
					APIGroups: []string{"foo"},
					Resources: []string{"widgets"},
					Verbs:     []string{"get"},
				},
			},
			rules,
		)
	})

	t.Run("kitchen sink", func(t *testing.T) {
		rules, err := NormalizePolicyRules(
			[]rbacv1.PolicyRule{
				{ // group "" is preserved as core (not inferred); splits per resource
					APIGroups: []string{""},
					Resources: []string{"serviceaccounts", "stages"},
					Verbs:     []string{"*"},
				},
				{ // merges with the get-only kargo stages rule below
					APIGroups: []string{kargoapi.GroupVersion.Group},
					Resources: []string{"stages"},
					Verbs:     []string{"*"},
				},
				{ // merged with the stages rule above (same group + resource)
					APIGroups: []string{kargoapi.GroupVersion.Group},
					Resources: []string{"stages"},
					Verbs:     []string{"get"},
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
					// The group "" stages rule is preserved as core -- it is NOT
					// inferred to kargo.akuity.io and so does NOT merge with the
					// kargo.akuity.io stages rule below. (Group inference for
					// group-less input happens at the Create/Update layer via
					// resolveRuleGroups, not in NormalizePolicyRules.)
					APIGroups: []string{""},
					Resources: []string{"stages"},
					Verbs:     allStagesVerbs,
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

func TestResolveRuleGroups(t *testing.T) {
	resolve := func(resource string) (string, error) {
		switch resource {
		case "stages":
			return kargoapi.GroupVersion.Group, nil
		case "secrets":
			return "", nil
		default:
			return "", errors.New("unrecognized resource type")
		}
	}

	t.Run("nil resolver leaves rules unchanged", func(t *testing.T) {
		in := []rbacv1.PolicyRule{{Resources: []string{"stages"}, Verbs: []string{"get"}}}
		out, err := resolveRuleGroups(in, nil)
		require.NoError(t, err)
		require.Equal(t, in, out)
	})

	t.Run("explicit group is preserved", func(t *testing.T) {
		in := []rbacv1.PolicyRule{{
			APIGroups: []string{"already.set"},
			Resources: []string{"stages"},
			Verbs:     []string{"get"},
		}}
		out, err := resolveRuleGroups(in, resolve)
		require.NoError(t, err)
		require.Equal(t, in, out)
	})

	t.Run("empty group is resolved from the resource", func(t *testing.T) {
		out, err := resolveRuleGroups([]rbacv1.PolicyRule{{
			Resources: []string{"stages"},
			Verbs:     []string{"get"},
		}}, resolve)
		require.NoError(t, err)
		require.Equal(t, []rbacv1.PolicyRule{{
			APIGroups: []string{kargoapi.GroupVersion.Group},
			Resources: []string{"stages"},
			Verbs:     []string{"get"},
		}}, out)
	})

	t.Run("group-less multi-resource rule splits per resource", func(t *testing.T) {
		out, err := resolveRuleGroups([]rbacv1.PolicyRule{{
			Resources: []string{"stages", "secrets"},
			Verbs:     []string{"get"},
		}}, resolve)
		require.NoError(t, err)
		require.Equal(
			t,
			[]rbacv1.PolicyRule{
				{
					APIGroups: []string{kargoapi.GroupVersion.Group},
					Resources: []string{"stages"},
					Verbs:     []string{"get"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"secrets"},
					Verbs:     []string{"get"},
				},
			},
			out,
		)
	})

	t.Run("resolver error is propagated", func(t *testing.T) {
		_, err := resolveRuleGroups([]rbacv1.PolicyRule{{
			Resources: []string{"unknown"},
			Verbs:     []string{"get"},
		}}, resolve)
		require.Error(t, err)
	})
}
