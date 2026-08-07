package rbac

import (
	"context"
	"fmt"
	"slices"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	"github.com/akuity/kargo/pkg/server/kubernetes"
)

var (
	roleGVK               = rbacv1.SchemeGroupVersion.WithKind("Role")
	roleBindingGVK        = rbacv1.SchemeGroupVersion.WithKind("RoleBinding")
	clusterRoleGVK        = rbacv1.SchemeGroupVersion.WithKind("ClusterRole")
	clusterRoleBindingGVK = rbacv1.SchemeGroupVersion.WithKind("ClusterRoleBinding")
	serviceAccountGVK     = corev1.SchemeGroupVersion.WithKind("ServiceAccount")
)

// VerifyResourceNotEscalating guards the generic resource endpoints against
// RBAC privilege escalation. The authorizing client checks only that the
// requester may perform the verb -- e.g. create Roles in the namespace -- not
// that they already hold the permissions the resource would grant; a
// SubjectAccessReview answers the former, never the latter. Kubernetes itself
// enforces the latter at admission, but against the identity performing the
// write -- and these endpoints write with the API server's own privileged
// credentials, so that check passes vacuously. This supplies what neither does:
// it requires the requester (bound to ctx) to already hold every permission the
// created or updated obj would confer.
//
// Only these kinds confer permissions; any other returns nil:
//
//   - Role, ClusterRole: the rules they declare.
//   - RoleBinding, ClusterRoleBinding: the rules of the Role or ClusterRole
//     they reference.
//   - ServiceAccount annotated for OIDC claim mapping: the rules of every Role
//     and ClusterRole bound to it, because mapping a claim onto it hands those
//     permissions to every user bearing that claim.
//
// Each permission is verified at the scope where it applies: namespaced kinds
// in obj's namespace, cluster-scoped kinds cluster-wide. (A RoleBinding is
// namespaced even when it references a ClusterRole.)
//
// authz runs the "does the requester already hold this?" checks; a nil authz
// disables verification (tests, non-authorizing local mode). resolver reads the
// Roles and bindings needed to decide and must always be able to read them,
// e.g. the server's internal client rather than one scoped to the requester.
// globalNamespaces matters only for the ServiceAccount case; see
// verifyServiceAccountBindingsNotEscalating.
func VerifyResourceNotEscalating(
	ctx context.Context,
	authz kubernetes.Authorizer,
	resolver client.Client,
	globalNamespaces []string,
	obj *unstructured.Unstructured,
) error {
	if obj == nil || authz == nil {
		return nil
	}
	switch obj.GroupVersionKind() {
	case roleGVK:
		role := &rbacv1.Role{}
		if err := fromUnstructured(obj, role); err != nil {
			return err
		}
		_, err := verifyRulesNotEscalating(ctx, authz, obj.GetNamespace(), role.Rules)
		return err
	case clusterRoleGVK:
		clusterRole := &rbacv1.ClusterRole{}
		if err := fromUnstructured(obj, clusterRole); err != nil {
			return err
		}
		_, err := verifyRulesNotEscalating(
			ctx,
			authz,
			"", // Cluster-scoped; no namespace
			clusterRole.Rules,
		)
		return err
	case roleBindingGVK:
		rb := &rbacv1.RoleBinding{}
		if err := fromUnstructured(obj, rb); err != nil {
			return err
		}
		// A namespaced RoleBinding confers within its own namespace, even when it
		// references a ClusterRole.
		rules, err := resolveRoleRefRules(ctx, resolver, obj.GetNamespace(), rb.RoleRef)
		if err != nil {
			return err
		}
		_, err = verifyRulesNotEscalating(ctx, authz, obj.GetNamespace(), rules)
		return err
	case clusterRoleBindingGVK:
		crb := &rbacv1.ClusterRoleBinding{}
		if err := fromUnstructured(obj, crb); err != nil {
			return err
		}
		// A ClusterRoleBinding may only reference a ClusterRole, and confers it
		// cluster-wide.
		rules, err := resolveRoleRefRules(
			ctx,
			resolver,
			"", // Cluster-scoped; no namespace
			crb.RoleRef,
		)
		if err != nil {
			return err
		}
		_, err = verifyRulesNotEscalating(
			ctx,
			authz,
			"", // Cluster-scoped; no namespace
			rules,
		)
		return err
	case serviceAccountGVK:
		// Only ServiceAccounts that map identities via OIDC claim annotations
		// can confer anything; without claims, no identity gains the SA's Roles.
		claims, err := rbacapi.OIDCClaimsFromAnnotationValues(obj.GetAnnotations())
		if err != nil {
			// Malformed claim annotations: fail closed rather than skip the check.
			return fmt.Errorf("error reading ServiceAccount claim annotations: %w", err)
		}
		if len(claims) == 0 {
			return nil
		}
		return verifyServiceAccountBindingsNotEscalating(
			ctx, authz, resolver, globalNamespaces, obj.GetNamespace(), obj.GetName(),
		)
	default:
		return nil
	}
}

// fromUnstructured decodes the raw obj into a typed target (Role, RoleBinding,
// etc.) so the escalation check can inspect it.
func fromUnstructured(obj *unstructured.Unstructured, target any) error {
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(
		obj.Object, target,
	); err != nil {
		return fmt.Errorf(
			"error decoding %s for escalation check: %w", obj.GetKind(), err,
		)
	}
	return nil
}

// verifyServiceAccountBindingsNotEscalating handles the ServiceAccount case: it
// requires the requester to already hold every permission conferred on the
// identities that claim mapping binds to this ServiceAccount.
//
// Those permissions come from the RoleBindings and ClusterRoleBindings naming
// the ServiceAccount as a subject, found by listing (a binding may have any
// name). Only bindings that grant usable authority matter, which turns on where
// the authorizing client consults this ServiceAccount: its own namespace only
// -- or every namespace, if that namespace is one of globalNamespaces. So:
//
//   - RoleBindings: listed in the ServiceAccount's own namespace (a binding
//     elsewhere confers nothing the mapped identity could use), or in every
//     namespace when that namespace is global. Each is verified where it
//     confers -- its own namespace.
//   - ClusterRoleBindings: always listed; verified cluster-wide.
//
// Fails closed on any read error.
func verifyServiceAccountBindingsNotEscalating(
	ctx context.Context,
	authz kubernetes.Authorizer,
	resolver client.Client,
	globalNamespaces []string,
	namespace string,
	saName string,
) error {
	if resolver == nil {
		return fmt.Errorf(
			"no client available to resolve bindings of ServiceAccount %q", saName,
		)
	}

	// Own namespace only, unless it is global -- then every namespace.
	roleBindings := &rbacv1.RoleBindingList{}
	var listOpts []client.ListOption
	if !slices.Contains(globalNamespaces, namespace) {
		listOpts = append(listOpts, client.InNamespace(namespace))
	}
	if err := resolver.List(ctx, roleBindings, listOpts...); err != nil {
		return fmt.Errorf(
			"error listing RoleBindings for escalation check: %w", err,
		)
	}
	for _, rb := range roleBindings.Items {
		if !subjectsReferenceServiceAccount(rb.Subjects, rb.Namespace, namespace, saName) {
			continue
		}
		rules, err := resolveRoleRefRules(ctx, resolver, rb.Namespace, rb.RoleRef)
		if err != nil {
			return err
		}
		if _, err := verifyRulesNotEscalating(ctx, authz, rb.Namespace, rules); err != nil {
			return err
		}
	}

	// ClusterRoleBindings -- conferred cluster-wide.
	clusterRoleBindings := &rbacv1.ClusterRoleBindingList{}
	if err := resolver.List(ctx, clusterRoleBindings); err != nil {
		return fmt.Errorf(
			"error listing ClusterRoleBindings for escalation check: %w", err,
		)
	}
	for _, crb := range clusterRoleBindings.Items {
		// ClusterRoleBinding subjects have no default namespace, so pass "".
		if !subjectsReferenceServiceAccount(crb.Subjects, "", namespace, saName) {
			continue
		}
		rules, err := resolveRoleRefRules(ctx, resolver, "", crb.RoleRef)
		if err != nil {
			return err
		}
		if _, err := verifyRulesNotEscalating(ctx, authz, "", rules); err != nil {
			return err
		}
	}
	return nil
}

// subjectsReferenceServiceAccount reports whether any subject names the
// ServiceAccount identified by saNamespace and saName. A subject that omits its
// namespace defaults to bindingNamespace: the binding's own namespace for a
// RoleBinding, or "" for a ClusterRoleBinding, whose ServiceAccount subjects
// must state a namespace explicitly.
func subjectsReferenceServiceAccount(
	subjects []rbacv1.Subject,
	bindingNamespace string,
	saNamespace string,
	saName string,
) bool {
	for _, s := range subjects {
		if s.Kind != "ServiceAccount" || s.Name != saName {
			continue
		}
		ns := s.Namespace
		if ns == "" {
			ns = bindingNamespace
		}
		if ns == saNamespace {
			return true
		}
	}
	return false
}

// resolveRoleRefRules returns the rules of the Role or ClusterRole named by
// roleRef; namespace locates a Role (ClusterRoles are cluster-scoped). It fails
// closed: any read error, including a missing Role, is returned so the check
// cannot be bypassed by pointing at a Role it cannot resolve.
func resolveRoleRefRules(
	ctx context.Context,
	resolver client.Client,
	namespace string,
	roleRef rbacv1.RoleRef,
) ([]rbacv1.PolicyRule, error) {
	if resolver == nil {
		return nil, fmt.Errorf("no client available to resolve roleRef %q", roleRef.Name)
	}
	switch roleRef.Kind {
	case "Role":
		role := &rbacv1.Role{}
		if err := resolver.Get(
			ctx,
			client.ObjectKey{Namespace: namespace, Name: roleRef.Name},
			role,
		); err != nil {
			return nil, fmt.Errorf(
				"error resolving referenced Role %q in namespace %q for escalation check: %w",
				roleRef.Name, namespace, err,
			)
		}
		return role.Rules, nil
	case "ClusterRole":
		clusterRole := &rbacv1.ClusterRole{}
		if err := resolver.Get(
			ctx,
			client.ObjectKey{Name: roleRef.Name},
			clusterRole,
		); err != nil {
			return nil, fmt.Errorf(
				"error resolving referenced ClusterRole %q for escalation check: %w",
				roleRef.Name, err,
			)
		}
		return clusterRole.Rules, nil
	default:
		return nil, fmt.Errorf(
			"unsupported roleRef kind %q", roleRef.Kind,
		)
	}
}
