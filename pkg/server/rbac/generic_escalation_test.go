package rbac

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	"github.com/akuity/kargo/pkg/server/kubernetes"
)

func TestVerifyResourceNotEscalating(t *testing.T) {
	const ns = "fake-project"
	const otherNS = "other-project"
	const globalNS = "kargo-global"

	role := &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{Name: "powerful", Namespace: ns},
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{""},
			Resources: []string{"secrets"},
			Verbs:     []string{"get"},
		}},
	}
	roleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{Name: "binding", Namespace: ns},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "powerful",
		},
		Subjects: []rbacv1.Subject{{Kind: "ServiceAccount", Name: "attacker", Namespace: ns}},
	}

	clusterRole := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-powerful"},
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{""},
			Resources: []string{"secrets"},
			Verbs:     []string{"get"},
		}},
	}
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-binding"},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cluster-powerful",
		},
		Subjects: []rbacv1.Subject{{Kind: "ServiceAccount", Name: "attacker", Namespace: ns}},
	}

	// A RoleBinding that binds "bound-sa" to the powerful Role, so the
	// ServiceAccount-claims vector can be resolved.
	saBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "bound-sa-binding", Namespace: ns},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "powerful",
		},
		Subjects: []rbacv1.Subject{{Kind: "ServiceAccount", Name: "bound-sa", Namespace: ns}},
	}
	// A ClusterRoleBinding that binds "cluster-bound-sa" to the powerful
	// ClusterRole, so the cluster-wide ServiceAccount-claims vector resolves.
	saClusterBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-bound-sa-binding"},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cluster-powerful",
		},
		Subjects: []rbacv1.Subject{{Kind: "ServiceAccount", Name: "cluster-bound-sa", Namespace: ns}},
	}
	// A Role and a RoleBinding in a DIFFERENT namespace that empower a
	// ServiceAccount living in `ns`. This confers permissions within otherNS.
	roleOtherNS := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: "xns-role", Namespace: otherNS},
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{""},
			Resources: []string{"secrets"},
			Verbs:     []string{"get"},
		}},
	}
	saCrossNSBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "xns-binding", Namespace: otherNS},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "xns-role",
		},
		Subjects: []rbacv1.Subject{{Kind: "ServiceAccount", Name: "xns-bound-sa", Namespace: ns}},
	}
	// A cross-namespace RoleBinding empowering a ServiceAccount that lives in a
	// GLOBAL namespace. Because a global-namespace SA is consulted for ops in
	// every namespace, such a binding is relevant and must be checked.
	globalCrossNSBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "global-xns-binding", Namespace: otherNS},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "xns-role",
		},
		Subjects: []rbacv1.Subject{{Kind: "ServiceAccount", Name: "global-sa", Namespace: globalNS}},
	}
	saWithClaimsInNS := func(name, namespace string) *unstructured.Unstructured {
		return toUnstructured(t, &corev1.ServiceAccount{
			TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "ServiceAccount"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					rbacapi.AnnotationKeyOIDCClaim("sub"): "attacker@example.com",
				},
			},
		})
	}

	saWithClaims := func(name string) *unstructured.Unstructured {
		return toUnstructured(t, &corev1.ServiceAccount{
			TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "ServiceAccount"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
				Annotations: map[string]string{
					rbacapi.AnnotationKeyOIDCClaim("sub"): "attacker@example.com",
				},
			},
		})
	}

	// Resolver client holds the referenced Roles and the SA bindings so the
	// binding and ServiceAccount checks can resolve rules.
	testScheme := runtime.NewScheme()
	require.NoError(t, rbacv1.AddToScheme(testScheme))
	require.NoError(t, corev1.AddToScheme(testScheme))
	resolver := fake.NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(
			role, clusterRole, roleOtherNS,
			saBinding, saClusterBinding, saCrossNSBinding, globalCrossNSBinding,
		).
		Build()

	holdsSecretsGet := func(verb string, gvr schema.GroupVersionResource, _ string) bool {
		return verb == "get" && gvr.Group == "" && gvr.Resource == "secrets"
	}
	holdsNothing := func(string, schema.GroupVersionResource, string) bool { return false }

	testCases := []struct {
		name             string
		authz            kubernetes.Authorizer
		globalNamespaces []string
		obj              *unstructured.Unstructured
		wantErr          bool
	}{
		{
			name:  "nil authorizer disables the check",
			authz: nil,
			obj:   toUnstructured(t, role),
		},
		{
			name:  "non-RBAC resource is ignored",
			authz: &fakeAuthorizer{allow: holdsNothing},
			obj: toUnstructured(t, &corev1.ConfigMap{
				TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
				ObjectMeta: metav1.ObjectMeta{Name: "cm", Namespace: ns},
			}),
		},
		{
			name:    "Role granting rules the requester lacks is rejected",
			authz:   &fakeAuthorizer{allow: holdsNothing},
			obj:     toUnstructured(t, role),
			wantErr: true,
		},
		{
			name:  "Role granting rules the requester holds is allowed",
			authz: &fakeAuthorizer{allow: holdsSecretsGet},
			obj:   toUnstructured(t, role),
		},
		{
			name:    "RoleBinding to a Role the requester lacks is rejected",
			authz:   &fakeAuthorizer{allow: holdsNothing},
			obj:     toUnstructured(t, roleBinding),
			wantErr: true,
		},
		{
			name:  "RoleBinding to a Role the requester holds is allowed",
			authz: &fakeAuthorizer{allow: holdsSecretsGet},
			obj:   toUnstructured(t, roleBinding),
		},
		{
			name:    "ServiceAccount with claims bound to a Role the requester lacks is rejected",
			authz:   &fakeAuthorizer{allow: holdsNothing},
			obj:     saWithClaims("bound-sa"),
			wantErr: true,
		},
		{
			name:  "ServiceAccount with claims bound to a Role the requester holds is allowed",
			authz: &fakeAuthorizer{allow: holdsSecretsGet},
			obj:   saWithClaims("bound-sa"),
		},
		{
			name:  "ServiceAccount without claims is ignored",
			authz: &fakeAuthorizer{allow: holdsNothing},
			obj: toUnstructured(t, &corev1.ServiceAccount{
				TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "ServiceAccount"},
				ObjectMeta: metav1.ObjectMeta{Name: "bound-sa", Namespace: ns},
			}),
		},
		{
			name:    "ClusterRole granting rules the requester lacks is rejected",
			authz:   &fakeAuthorizer{allow: holdsNothing},
			obj:     toUnstructured(t, clusterRole),
			wantErr: true,
		},
		{
			name:  "ClusterRole granting rules the requester holds is allowed",
			authz: &fakeAuthorizer{allow: holdsSecretsGet},
			obj:   toUnstructured(t, clusterRole),
		},
		{
			name:    "ClusterRoleBinding to a ClusterRole the requester lacks is rejected",
			authz:   &fakeAuthorizer{allow: holdsNothing},
			obj:     toUnstructured(t, clusterRoleBinding),
			wantErr: true,
		},
		{
			name:  "ClusterRoleBinding to a ClusterRole the requester holds is allowed",
			authz: &fakeAuthorizer{allow: holdsSecretsGet},
			obj:   toUnstructured(t, clusterRoleBinding),
		},
		{
			name:    "ServiceAccount with claims bound cluster-wide the requester lacks is rejected",
			authz:   &fakeAuthorizer{allow: holdsNothing},
			obj:     saWithClaims("cluster-bound-sa"),
			wantErr: true,
		},
		{
			name:  "ServiceAccount with claims bound cluster-wide the requester holds is allowed",
			authz: &fakeAuthorizer{allow: holdsSecretsGet},
			obj:   saWithClaims("cluster-bound-sa"),
		},
		{
			// A binding in another namespace grants a project-namespace SA
			// authority the mapped identity could never exercise through Kargo
			// (that SA is never consulted for ops in the other namespace), so it
			// must NOT be treated as escalation -- even though the requester
			// holds nothing.
			name:  "cross-namespace binding to a project-namespace SA is not considered",
			authz: &fakeAuthorizer{allow: holdsNothing},
			obj:   saWithClaims("xns-bound-sa"),
		},
		{
			// A SA living in a global namespace IS consulted for ops in every
			// namespace, so a cross-namespace binding empowering it is relevant
			// and must be checked.
			name:             "cross-namespace binding to a global-namespace SA is enforced",
			authz:            &fakeAuthorizer{allow: holdsNothing},
			globalNamespaces: []string{globalNS},
			obj:              saWithClaimsInNS("global-sa", globalNS),
			wantErr:          true,
		},
		{
			name:             "cross-namespace binding to a global-namespace SA the requester holds is allowed",
			authz:            &fakeAuthorizer{allow: holdsSecretsGet},
			globalNamespaces: []string{globalNS},
			obj:              saWithClaimsInNS("global-sa", globalNS),
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := VerifyResourceNotEscalating(
				context.Background(),
				testCase.authz,
				resolver,
				testCase.globalNamespaces,
				testCase.obj,
			)
			if testCase.wantErr {
				require.Error(t, err)
				require.True(t, apierrors.IsForbidden(err))
				return
			}
			require.NoError(t, err)
		})
	}
}

func toUnstructured(t *testing.T, obj any) *unstructured.Unstructured {
	t.Helper()
	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	require.NoError(t, err)
	return &unstructured.Unstructured{Object: m}
}
