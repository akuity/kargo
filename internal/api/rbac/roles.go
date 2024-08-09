package rbac

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// RolesDatabase is an interface for the Kargo Roles store.
type RolesDatabase interface {
	// Create creates the ServiceAccount, Role, and RoleBinding underlying a new
	// Kargo Role. It will return an error if any of those resources already
	// exist.
	Create(context.Context, *rbacapi.Role) (*rbacapi.Role, error)
	// Delete deletes a Kargo Role's underlying ServiceAccount, Role, and
	// RoleBinding. It will return an error if no underlying resources exist or if
	// any underlying resources are not Kargo-manageable.
	Delete(ctx context.Context, project, name string) error
	// Get returns a Kargo Role representation of an underlying ServiceAccount
	// and any Roles it is associated with. It will return an error if no
	// underlying ServiceAccount exists.
	Get(ctx context.Context, project, name string) (*rbacapi.Role, error)
	// GetAsResources returns the ServiceAccount and any Roles and RoleBindings
	// underlying a Kargo Role. It will return an error if no underlying
	// ServiceAccount exists. It is valid for the Roles and/or RoleBindings to be
	// missing, in which case they will be returned as nil.
	GetAsResources(
		ctx context.Context,
		project string,
		name string,
	) (*corev1.ServiceAccount, []rbacv1.Role, []rbacv1.RoleBinding, error)
	// GrantPermissionsToRole amends the Role underlying a Kargo Role with new
	// rules. It will return an error if no underlying ServiceAccount exists or
	// any underlying resources are not Kargo-manageable. It will create
	// underlying Role and RoleBinding resources if they do not exist.
	GrantPermissionsToRole(
		ctx context.Context,
		project string,
		name string,
		resourceDetails *rbacapi.ResourceDetails,
	) (*rbacapi.Role, error)
	// GrantRoleToUsers amends claim annotations of the ServiceAccount underlying
	// a Kargo Role. It will return an error if no underlying ServiceAccount
	// exists or any underlying resources are not Kargo-manageable.
	GrantRoleToUsers(
		ctx context.Context,
		project string,
		name string,
		userClaims *rbacapi.UserClaims,
	) (*rbacapi.Role, error)
	// List returns Kargo Role representations of underlying ServiceAccounts and
	// andy Roles and RoleBindings associated with them.
	List(ctx context.Context, project string) ([]*rbacapi.Role, error)
	// ListNames returns names of Kargo Roles..
	ListNames(ctx context.Context, project string) ([]string, error)
	// RevokePermissionFromRole removes select rules from the Role underlying a
	// Kargo Role. It will return an error if no underlying ServiceAccount exists
	// or any underlying resources are not Kargo-manageable.
	RevokePermissionsFromRole(
		ctx context.Context,
		project string,
		name string,
		resourceDetails *rbacapi.ResourceDetails,
	) (*rbacapi.Role, error)
	// RevokeRoleFromUsers removes select claims from claim annotations of the
	// ServiceAccount underlying a Kargo Role. It will return an error if no
	// underlying ServiceAccount exists or any underlying resources are not
	// Kargo-manageable.
	RevokeRoleFromUsers(
		ctx context.Context,
		project string,
		name string,
		userClaims *rbacapi.UserClaims,
	) (*rbacapi.Role, error)
	// Update updates the underlying ServiceAccount and Role resources underlying
	// a Kargo Role. It will return an error if no underlying ServiceAccount
	// exists or any underlying resources are not Kargo-manageable. It will create
	// underlying Role and RoleBinding resources if they do not exist.
	Update(context.Context, *rbacapi.Role) (*rbacapi.Role, error)
}

// rolesDatabase is an implementation of the RolesDatabase interface
// that utilizes a Kubernetes controller runtime client to store and retrieve
// Kargo Roles stored Kubernetes in the form of ServiceAccount/Role/RoleBinding
// trios.
type rolesDatabase struct {
	client client.Client
}

// NewKubernetesRolesDatabase returns an implementation of the RolesDatabase
// interface that utilizes a Kubernetes controller runtime client to store and
// retrieve Kargo Roles stored Kubernetes in the form of
// ServiceAccount/Role/RoleBinding trios.
func NewKubernetesRolesDatabase(c client.Client) RolesDatabase {
	return &rolesDatabase{client: c}
}

// CreateRole implements the RolesDatabase interface.
func (r *rolesDatabase) Create(
	ctx context.Context,
	kargoRole *rbacapi.Role,
) (*rbacapi.Role, error) {
	objKey := client.ObjectKey{
		Namespace: kargoRole.Namespace,
		Name:      kargoRole.Name,
	}

	// Check if the ServiceAccount we would create already exists
	sa := &corev1.ServiceAccount{}
	if err := r.client.Get(ctx, objKey, sa); client.IgnoreNotFound(err) != nil {
		return nil, fmt.Errorf(
			"error getting ServiceAccount %q in namespace %q: %w", kargoRole.Name, kargoRole.Namespace, err,
		)
	} else if err == nil {
		return nil, kubeerr.NewAlreadyExists(
			schema.GroupResource{
				Group:    sa.GetObjectKind().GroupVersionKind().Group,
				Resource: strings.ToLower(sa.GetObjectKind().GroupVersionKind().Kind),
			},
			sa.Name,
		)
	}

	// Check if the Role we would create already exists
	role := &rbacv1.Role{}
	if err := r.client.Get(ctx, objKey, role); client.IgnoreNotFound(err) != nil {
		return nil, fmt.Errorf(
			"error getting Role %q in namespace %q: %w", kargoRole.Name, kargoRole.Namespace, err,
		)
	} else if err == nil {
		return nil, kubeerr.NewAlreadyExists(
			schema.GroupResource{
				Group:    role.GetObjectKind().GroupVersionKind().Group,
				Resource: strings.ToLower(role.GetObjectKind().GroupVersionKind().Kind),
			},
			role.Name,
		)
	}

	// Check if the RoleBinding we would create already exists
	rb := &rbacv1.RoleBinding{}
	if err := r.client.Get(ctx, objKey, rb); client.IgnoreNotFound(err) != nil {
		return nil, fmt.Errorf(
			"error getting RoleBinding %q in namespace %q: %w", kargoRole.Name, kargoRole.Namespace, err,
		)
	} else if err == nil {
		return nil, kubeerr.NewAlreadyExists(
			schema.GroupResource{
				Group:    rb.GetObjectKind().GroupVersionKind().Group,
				Resource: strings.ToLower(rb.GetObjectKind().GroupVersionKind().Kind),
			},
			rb.Name,
		)
	}

	// If we get to here, we may proceed with creating the
	// ServiceAccount/Role/RoleBinding trio

	sa, role, rb, err := RoleToResources(kargoRole)
	if err != nil {
		return nil, fmt.Errorf("error converting Kargo Role to resources: %w", err)
	}

	// Append the description annotation to the Role if it exists
	if description, ok := kargoRole.Annotations[kargoapi.AnnotationKeyDescription]; ok {
		if sa.Annotations == nil {
			sa.Annotations = map[string]string{}
		}
		sa.Annotations[kargoapi.AnnotationKeyDescription] = description
	}

	if err = r.client.Create(ctx, sa); err != nil {
		return nil, fmt.Errorf(
			"error creating ServiceAccount %q in namespace %q: %w", kargoRole.Name, kargoRole.Namespace, err,
		)
	}

	if err := r.client.Create(ctx, rb); err != nil {
		return nil, fmt.Errorf(
			"error creating RoleBinding %q in namespace %q: %w", kargoRole.Name, kargoRole.Namespace, err,
		)
	}

	// Note: The Role's rules are already normalized
	if err := r.client.Create(ctx, role); err != nil {
		return nil, fmt.Errorf(
			"error creating Role %q in namespace %q: %w", kargoRole.Name, kargoRole.Namespace, err,
		)
	}

	return kargoRole, nil
}

// DeleteRole implements the RolesDatabase interface.
func (r *rolesDatabase) Delete(
	ctx context.Context,
	project string,
	name string,
) error {
	sa, roles, rbs, err := r.GetAsResources(ctx, project, name)
	if err != nil {
		return err
	}
	// Narrow down to manageable resources. This will return an error if these
	// resources are not manageable for any reason.
	role, rb, err := manageableResources(*sa, roles, rbs)
	if err != nil {
		return err
	}

	// Proceed with deletion

	if role != nil {
		if err := r.client.Delete(ctx, role); err != nil {
			return fmt.Errorf(
				"error deleting Role %q in namespace %q: %w", role.Name, role.Namespace, err,
			)
		}
	}

	if rb != nil {
		if err := r.client.Delete(ctx, rb); err != nil {
			return fmt.Errorf(
				"error deleting RoleBinding %q in namespace %q: %w", rb.Name, rb.Namespace, err,
			)
		}
	}

	// If we got to here, sa cannot have been nil
	if err := r.client.Delete(ctx, sa); err != nil {
		return fmt.Errorf(
			"error deleting ServiceAccount %q in namespace %q: %w", sa.Name, sa.Namespace, err,
		)
	}

	return nil
}

// Get implements the RolesDatabase interface.
func (r *rolesDatabase) Get(
	ctx context.Context,
	project string,
	name string,
) (*rbacapi.Role, error) {
	sa, roles, rbs, err := r.GetAsResources(ctx, project, name)
	if err != nil {
		return nil, err
	}

	// Note: The underlying resources we found may not be manageable, but we
	// can still return a Kargo Role that summarizes them.

	// Note: The Kargo Role will come back with normalized rules
	return ResourcesToRole(sa, roles, rbs)
}

// GetAsResources implements the RolesDatabase interface.
func (r *rolesDatabase) GetAsResources(
	ctx context.Context,
	project string,
	name string,
) (*corev1.ServiceAccount, []rbacv1.Role, []rbacv1.RoleBinding, error) {
	objKey := client.ObjectKey{
		Namespace: project,
		Name:      name,
	}

	sa := &corev1.ServiceAccount{}
	if err := r.client.Get(ctx, objKey, sa); err != nil {
		return nil, nil, nil, fmt.Errorf(
			"error getting ServiceAccount %q in namespace %q: %w", name, project, err,
		)
	}

	// Find all RoleBindings in the project namespace
	rbList := &rbacv1.RoleBindingList{}
	if err := r.client.List(ctx, rbList, client.InNamespace(project)); err != nil {
		return nil, nil, nil, fmt.Errorf(
			"error listing RoleBindings in namespace %q: %w", project, err,
		)
	}
	// Narrow the list to just the RoleBindings that reference the ServiceAccount
	rbs := make([]rbacv1.RoleBinding, 0, len(rbList.Items))
	for i := range rbList.Items {
		rb := &rbList.Items[i]
		for _, subject := range rb.Subjects {
			if subject.Kind == rbacv1.ServiceAccountKind &&
				subject.Namespace == project &&
				subject.Name == name {
				rbs = append(rbs, *rb)
				break
			}
		}
	}

	if len(rbs) == 0 {
		return sa, nil, nil, nil
	}

	// Find all Roles that are referenced by the RoleBindings
	roles := make([]rbacv1.Role, 0, len(rbs))
	for _, rb := range rbs {
		role := &rbacv1.Role{}
		if err := r.client.Get(
			ctx, client.ObjectKey{
				Namespace: project,
				Name:      rb.RoleRef.Name,
			},
			role,
		); err != nil {
			return nil, nil, nil, fmt.Errorf(
				"error getting Role %q in namespace %q: %w", rb.RoleRef.Name, project, err,
			)
		}
		roles = append(roles, *role)
	}

	return sa, roles, rbs, nil
}

// GrantPermissionsToRole implements the RolesDatabase interface.
func (r *rolesDatabase) GrantPermissionsToRole(
	ctx context.Context,
	project string,
	name string,
	resourceDetails *rbacapi.ResourceDetails,
) (*rbacapi.Role, error) {
	sa, roles, rbs, err := r.GetAsResources(ctx, project, name)
	if err != nil {
		return nil, err
	}
	// Narrow down to manageable resources. This will return an error if these
	// resources are not manageable for any reason.
	role, rb, err := manageableResources(*sa, roles, rbs)
	if err != nil {
		return nil, err
	}

	if err = validateResourceTypeName(resourceDetails.ResourceType); err != nil {
		return nil, err
	}

	group := getGroupName(resourceDetails.ResourceType)

	newRole := role
	if newRole == nil {
		newRole = buildNewRole(project, name)
	}
	newRule := rbacv1.PolicyRule{
		APIGroups: []string{group},
		Resources: []string{resourceDetails.ResourceType},
		Verbs:     resourceDetails.Verbs,
	}
	if resourceDetails.ResourceName != "" {
		newRule.ResourceNames = []string{resourceDetails.ResourceName}
	}
	if newRole.Rules, err = NormalizePolicyRules(append(newRole.Rules, newRule)); err != nil {
		return nil, fmt.Errorf("error normalizing RBAC policy rules: %w", err)
	}

	if role == nil {
		if err = r.client.Create(ctx, newRole); err != nil {
			return nil, fmt.Errorf("error creating Role %q in namespace %q: %w", name, project, err)
		}
	} else if err = r.client.Update(ctx, newRole); err != nil {
		return nil, fmt.Errorf("error updating Role %q in namespace %q: %w", name, project, err)
	}

	if rb == nil {
		rb = buildNewRoleBinding(project, name)
		if err = r.client.Create(ctx, rb); err != nil {
			return nil, fmt.Errorf("error creating RoleBinding %q in namespace %q: %w", name, project, err)
		}
	}

	return ResourcesToRole(sa, []rbacv1.Role{*newRole}, []rbacv1.RoleBinding{*rb})
}

// GrantRoleToUsers implements the RolesDatabase interface.
func (r *rolesDatabase) GrantRoleToUsers(
	ctx context.Context,
	project string,
	name string,
	userClaims *rbacapi.UserClaims,
) (*rbacapi.Role, error) {
	sa, roles, rbs, err := r.GetAsResources(ctx, project, name)
	if err != nil {
		return nil, err
	}
	// This will return an error if these resources are not manageable for any
	// reason.
	role, _, err := manageableResources(*sa, roles, rbs)
	if err != nil {
		return nil, err
	}
	amendClaimAnnotation(sa, rbacapi.AnnotationKeyOIDCClaimNamePrefix, userClaims.Claims)
	if err = r.client.Update(ctx, sa); err != nil {
		return nil, fmt.Errorf("error updating ServiceAccount %q in namespace %q: %w", name, project, err)
	}

	if role == nil {
		return ResourcesToRole(sa, nil, rbs)
	}
	return ResourcesToRole(sa, []rbacv1.Role{*role}, rbs)
}

// List implements the RolesDatabase interface.
func (r *rolesDatabase) List(
	ctx context.Context,
	project string,
) ([]*rbacapi.Role, error) {
	saList := &corev1.ServiceAccountList{}
	if err := r.client.List(
		ctx,
		saList,
		client.InNamespace(project),
	); err != nil {
		return nil, fmt.Errorf("error listing ServiceAccounts in namespace %q: %w", project, err)
	}

	kargoRoles := make([]*rbacapi.Role, 0, len(saList.Items))
	for i := range saList.Items {
		sa, roles, rbs, err := r.GetAsResources(ctx, project, saList.Items[i].Name)
		if err != nil {
			return nil, fmt.Errorf(
				"error getting underlying resources for Kargo Role %q from namespace %q: %w",
				saList.Items[i].Name, project, err,
			)
		}
		// Note: The underlying resources we found may not be manageable, but we
		// can still return a Kargo Role that summarizes them.
		kargoRole, err := ResourcesToRole(sa, roles, rbs)
		if err != nil {
			return nil, fmt.Errorf("error converting underlying resources to Kargo Role %q: %w", sa.Name, err)
		}
		kargoRoles = append(kargoRoles, kargoRole)
	}

	sort.Slice(kargoRoles, func(i, j int) bool {
		return kargoRoles[i].Name < kargoRoles[j].Name
	})

	return kargoRoles, nil
}

func (r *rolesDatabase) ListNames(ctx context.Context, project string) ([]string, error) {
	saList := &corev1.ServiceAccountList{}
	if err := r.client.List(
		ctx,
		saList,
		client.InNamespace(project),
	); err != nil {
		return nil, fmt.Errorf("error listing ServiceAccounts in namespace %q: %w", project, err)
	}
	names := make([]string, 0, len(saList.Items))
	for i := range saList.Items {
		names = append(names, saList.Items[i].Name)
	}
	slices.Sort(names)
	return names, nil
}

// RevokePermissionFromRole implements the RolesDatabase interface.
func (r *rolesDatabase) RevokePermissionsFromRole(
	ctx context.Context,
	project string,
	name string,
	resourceDetails *rbacapi.ResourceDetails,
) (*rbacapi.Role, error) {
	sa, roles, rbs, err := r.GetAsResources(ctx, project, name)
	if err != nil {
		return nil, err
	}
	// Narrow down to manageable resources. This will return an error if these
	// resources are not manageable for any reason.
	role, _, err := manageableResources(*sa, roles, rbs)
	if err != nil {
		return nil, err
	}
	if role == nil { // Nothing to do
		return ResourcesToRole(sa, nil, rbs)
	}

	// Normalize the rules before attempting to modify them
	if role.Rules, err = NormalizePolicyRules(role.Rules); err != nil {
		return nil, fmt.Errorf("error normalizing RBAC policy rules: %w", err)
	}

	// Deal with wildcard verb
	for _, verb := range resourceDetails.Verbs {
		if strings.TrimSpace(verb) == "*" {
			resourceDetails.Verbs = append(resourceDetails.Verbs, allVerbs...)
			break
		}
	}
	// Compact the list of verbs we want to remove
	slices.Sort(resourceDetails.Verbs)
	resourceDetails.Verbs = slices.Compact(resourceDetails.Verbs)

	if err = validateResourceTypeName(resourceDetails.ResourceType); err != nil {
		return nil, err
	}

	group := getGroupName(resourceDetails.ResourceType)

	filteredRules := make([]rbacv1.PolicyRule, 0, len(role.Rules))
	for _, rule := range role.Rules {
		if rule.APIGroups[0] != group || rule.Resources[0] != resourceDetails.ResourceType ||
			(resourceDetails.ResourceName != "" && rule.ResourceNames[0] != resourceDetails.ResourceName) {
			filteredRules = append(filteredRules, rule)
			continue
		}
		rule.Verbs = removeFromStringSlice(rule.Verbs, resourceDetails.Verbs)
		if len(rule.Verbs) > 0 {
			filteredRules = append(filteredRules, rule)
		}
	}
	role.Rules = filteredRules

	if err = r.client.Update(ctx, role); err != nil {
		return nil, fmt.Errorf("error updating Role %q in namespace %q: %w", name, project, err)
	}

	return ResourcesToRole(sa, []rbacv1.Role{*role}, rbs)
}

// RevokeRoleFromUsers implements the RolesDatabase interface.
func (r *rolesDatabase) RevokeRoleFromUsers(
	ctx context.Context,
	project string,
	name string,
	userClaims *rbacapi.UserClaims,
) (*rbacapi.Role, error) {
	// Make sure at least part of the ServiceAccount/Role/RoleBinding trio exists
	sa, roles, rbs, err := r.GetAsResources(ctx, project, name)
	if err != nil {
		return nil, err
	}
	// This will return an error if these resources are not manageable for any
	// reason.
	role, _, err := manageableResources(*sa, roles, rbs)
	if err != nil {
		return nil, err
	}
	dropFromClaimAnnotation(sa, rbacapi.AnnotationKeyOIDCClaimNamePrefix, userClaims.Claims)
	if err = r.client.Update(ctx, sa); err != nil {
		return nil, fmt.Errorf("error updating ServiceAccount %q in namespace %q: %w", name, project, err)
	}

	if role == nil {
		return ResourcesToRole(sa, nil, rbs)
	}
	return ResourcesToRole(sa, []rbacv1.Role{*role}, rbs)
}

// Update implements the RolesDatabase interface.
func (r *rolesDatabase) Update(
	ctx context.Context,
	kargoRole *rbacapi.Role,
) (*rbacapi.Role, error) {
	sa, roles, rbs, err := r.GetAsResources(ctx, kargoRole.Namespace, kargoRole.Name)
	if err != nil {
		return nil, err
	}
	// Narrow down to manageable resources. This will return an error if these
	// resources are not manageable for any reason.
	role, rb, err := manageableResources(*sa, roles, rbs)
	if err != nil {
		return nil, err
	}

	replaceClaimAnnotation(sa, rbacapi.AnnotationKeyOIDCClaimNamePrefix, kargoRole.Claims)
	if description, ok := kargoRole.Annotations[kargoapi.AnnotationKeyDescription]; ok {
		if sa.Annotations == nil {
			sa.Annotations = map[string]string{}
		}
		sa.Annotations[kargoapi.AnnotationKeyDescription] = description
	} else {
		delete(sa.Annotations, kargoapi.AnnotationKeyDescription)
	}

	if err = r.client.Update(ctx, sa); err != nil {
		return nil, fmt.Errorf(
			"error updating ServiceAccount %q in namespace %q: %w", kargoRole.Name, kargoRole.Namespace, err,
		)
	}

	newRole := role
	if newRole == nil {
		newRole = buildNewRole(kargoRole.Namespace, kargoRole.Name)
	}
	if newRole.Rules, err = NormalizePolicyRules(kargoRole.Rules); err != nil {
		return nil, fmt.Errorf("error normalizing RBAC policy rules: %w", err)
	}
	if role == nil {
		if err := r.client.Create(ctx, newRole); err != nil {
			return nil, fmt.Errorf("error creating Role %q in namespace %q: %w", kargoRole.Name, kargoRole.Namespace, err)
		}
	} else if err := r.client.Update(ctx, newRole); err != nil {
		return nil, fmt.Errorf("error updating Role %q in namespace %q: %w", kargoRole.Name, kargoRole.Namespace, err)
	}

	if rb == nil {
		rb = buildNewRoleBinding(kargoRole.Namespace, kargoRole.Name)
		if err := r.client.Create(ctx, rb); err != nil {
			return nil, fmt.Errorf(
				"error creating RoleBinding %q in namespace %q: %w", kargoRole.Name, kargoRole.Namespace, err,
			)
		}
	}

	return ResourcesToRole(sa, []rbacv1.Role{*newRole}, rbs)
}

// ResourcesToRole converts the provided ServiceAccount, Role, and RoleBinding
// into a Kargo Role with normalized policy rules. If the ServiceAccount is nil,
// the Kargo Role will be nil.
func ResourcesToRole(
	sa *corev1.ServiceAccount,
	roles []rbacv1.Role,
	rbs []rbacv1.RoleBinding,
) (*rbacapi.Role, error) {
	if sa == nil {
		return nil, nil
	}

	kargoRole := &rbacapi.Role{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         sa.Namespace,
			Name:              sa.Name,
			CreationTimestamp: sa.CreationTimestamp,
		},
	}

	if description, ok := sa.Annotations[kargoapi.AnnotationKeyDescription]; ok {
		kargoRole.Annotations = map[string]string{kargoapi.AnnotationKeyDescription: description}
	}

	if isKargoManaged(sa) &&
		(len(roles) == 0 || (len(roles) == 1 && isKargoManaged(&roles[0]))) &&
		(len(rbs) == 0 || (len(rbs) == 1 && isKargoManaged(&rbs[0]))) {
		kargoRole.KargoManaged = true
	}

	for annotationKey, annotationValue := range sa.Annotations {
		if strings.HasPrefix(annotationKey, rbacapi.AnnotationKeyOIDCClaimNamePrefix) {
			newClaim := rbacapi.Claim{}
			newClaim.Name = strings.Replace(annotationKey, rbacapi.AnnotationKeyOIDCClaimNamePrefix, "", -1)
			newClaim.Values = strings.Split(annotationValue, ",")
			kargoRole.Claims = append(kargoRole.Claims, newClaim)
		}
	}

	sort.Slice(kargoRole.Claims, func(i, j int) bool {
		return kargoRole.Claims[i].Name < kargoRole.Claims[j].Name
	})

	kargoRole.Rules = []rbacv1.PolicyRule{}
	for _, role := range roles {
		kargoRole.Rules = append(kargoRole.Rules, role.Rules...)
	}

	// Since we cannot make any assumptions that they only contain resource types
	// we recognize, or that they don't do something really unusual like using a
	// wildcard resource type, never attempt to normalize rules if any of the
	// underlying resources are not Kargo-managed.
	if kargoRole.KargoManaged {
		var err error
		if kargoRole.Rules, err = NormalizePolicyRules(kargoRole.Rules); err != nil {
			return nil, fmt.Errorf("error normalizing RBAC policy rules: %w", err)
		}
	}

	return kargoRole, nil
}

// RoleToResources converts the provided Kargo Role into a
// ServiceAccount/Role/RoleBinding trio.
func RoleToResources(
	kargoRole *rbacapi.Role,
) (*corev1.ServiceAccount, *rbacv1.Role, *rbacv1.RoleBinding, error) {
	sa := buildNewServiceAccount(kargoRole.Namespace, kargoRole.Name)
	amendClaimAnnotation(sa, rbacapi.AnnotationKeyOIDCClaimNamePrefix, kargoRole.Claims)

	role := buildNewRole(kargoRole.Namespace, kargoRole.Name)
	var err error
	if role.Rules, err = NormalizePolicyRules(kargoRole.Rules); err != nil {
		return nil, nil, nil, fmt.Errorf("error normalizing RBAC policy rules: %w", err)
	}

	rb := buildNewRoleBinding(kargoRole.Namespace, kargoRole.Name)

	return sa, role, rb, nil
}

func replaceClaimAnnotation(sa *corev1.ServiceAccount, prefix string, values []rbacapi.Claim) {
	for _, claim := range values {
		slices.Sort(claim.Values)
		claim.Values = slices.Compact(claim.Values)
		if sa.Annotations == nil {
			sa.Annotations = map[string]string{}
		}
		sa.Annotations[prefix+claim.Name] = strings.Join(claim.Values, ",")
	}
}

func amendClaimAnnotation(sa *corev1.ServiceAccount, prefix string, values []rbacapi.Claim) {
	for _, claim := range values {
		existing := sa.Annotations[prefix+claim.Name]
		if existing != "" {
			claim.Values = append(strings.Split(existing, ","), claim.Values...)
		}
		slices.Sort(claim.Values)
		claim.Values = slices.Compact(claim.Values)
		if sa.Annotations == nil {
			sa.Annotations = map[string]string{}
		}
		sa.Annotations[prefix+claim.Name] = strings.Join(claim.Values, ",")
	}
}

func dropFromClaimAnnotation(sa *corev1.ServiceAccount, prefix string, values []rbacapi.Claim) {
	for _, claim := range values {
		slices.Sort(claim.Values)
		claim.Values = slices.Compact(claim.Values)
		claim.Values = removeFromStringSlice(strings.Split(sa.Annotations[prefix+claim.Name], ","), claim.Values)
		if len(claim.Values) == 0 {
			delete(sa.Annotations, prefix+claim.Name)
			continue
		}
		slices.Sort(claim.Values)
		if sa.Annotations == nil {
			sa.Annotations = map[string]string{}
		}
		sa.Annotations[prefix+claim.Name] = strings.Join(claim.Values, ",")
	}
}

func buildNewServiceAccount(namespace, name string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Annotations: map[string]string{
				rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
			},
		},
	}
}

func buildNewRole(namespace, name string) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Annotations: map[string]string{
				rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
			},
		},
	}
}

func buildNewRoleBinding(namespace, name string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Annotations: map[string]string{
				rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Namespace: namespace,
				Name:      name,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     name,
		},
	}
}

func removeFromStringSlice(s, items []string) []string {
	if len(items) == 0 {
		return s
	}
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		seen[item] = struct{}{}
	}
	filtered := make([]string, 0, len(s))
	for _, item := range s {
		if _, ok := seen[item]; !ok {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func isKargoManaged(obj metav1.Object) bool {
	return obj.GetAnnotations()[rbacapi.AnnotationKeyManaged] == rbacapi.AnnotationValueTrue
}

func manageableResources(
	sa corev1.ServiceAccount,
	roles []rbacv1.Role,
	rbs []rbacv1.RoleBinding,
) (*rbacv1.Role, *rbacv1.RoleBinding, error) {
	if !isKargoManaged(&sa) {
		return nil, nil, kubeerr.NewBadRequest(
			fmt.Sprintf(
				"ServiceAccount %q in namespace %q is not annotated as Kargo-managed",
				sa.Name, sa.Namespace,
			),
		)
	}
	if len(roles) > 1 {
		return nil, nil, kubeerr.NewBadRequest(
			fmt.Sprintf(
				"multiple Roles associated with ServiceAccount %q in namespace %q",
				sa.Name, sa.Namespace,
			),
		)
	}
	var role *rbacv1.Role
	if len(roles) == 1 {
		role = &roles[0]
		if !isKargoManaged(role) {
			return nil, nil, kubeerr.NewBadRequest(
				fmt.Sprintf(
					"Role %q in namespace %q is not annotated as Kargo-managed",
					role.Name, role.Namespace,
				),
			)
		}
	}
	if len(rbs) > 1 {
		return nil, nil, kubeerr.NewBadRequest(
			fmt.Sprintf(
				"multiple RoleBindings associated with ServiceAccount %q in namespace %q",
				sa.Name, sa.Namespace,
			),
		)
	}
	var rb *rbacv1.RoleBinding
	if len(rbs) == 1 {
		rb = &rbs[0]
		if !isKargoManaged(rb) {
			return nil, nil, kubeerr.NewBadRequest(
				fmt.Sprintf(
					"RoleBinding %q in namespace %q is not annotated as Kargo-managed",
					rb.Name, rb.Namespace,
				),
			)
		}
	}
	return role, rb, nil
}
