package rbac

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/kelseyhightower/envconfig"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

type RolesDatabaseConfig struct {
	KargoNamespace string `envconfig:"KARGO_NAMESPACE" default:"kargo"`
}

func RolesDatabaseConfigFromEnv() RolesDatabaseConfig {
	cfg := RolesDatabaseConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

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
	Get(
		ctx context.Context,
		systemLevel bool,
		project string,
		name string,
	) (*rbacapi.Role, error)
	// GetAsResources returns the ServiceAccount and any Roles and RoleBindings
	// underlying a Kargo Role. It will return an error if no underlying
	// ServiceAccount exists. It is valid for the Roles and/or RoleBindings to be
	// missing, in which case they will be returned as nil.
	GetAsResources(
		ctx context.Context,
		systemLevel bool,
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
		claims []rbacapi.Claim,
	) (*rbacapi.Role, error)
	// List returns Kargo Role representations of underlying ServiceAccounts and
	// andy Roles and RoleBindings associated with them.
	List(
		ctx context.Context,
		systemLevel bool,
		project string,
	) ([]*rbacapi.Role, error)
	// ListNames returns names of Kargo Roles..
	ListNames(
		ctx context.Context,
		systemLevel bool,
		project string,
	) ([]string, error)
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
		claims []rbacapi.Claim,
	) (*rbacapi.Role, error)
	// Update updates the underlying ServiceAccount and Role resources underlying
	// a Kargo Role. It will return an error if no underlying ServiceAccount
	// exists or any underlying resources are not Kargo-manageable. It will create
	// underlying Role and RoleBinding resources if they do not exist.
	Update(context.Context, *rbacapi.Role) (*rbacapi.Role, error)
	// CreateAPIToken generates and returns a new bearer token associated with a
	// Kargo Role in the form of a Kubernetes Secret.
	CreateAPIToken(
		ctx context.Context,
		systemLevel bool,
		project string,
		roleName string,
		tokenName string,
	) (*corev1.Secret, error)
	// DeleteAPIToken deletes a bearer token associated with a Kargo Role.
	DeleteAPIToken(
		ctx context.Context,
		systemLevel bool,
		project string,
		name string,
	) error
	// GetAPIToken returns a bearer token associated with a Kargo Role.
	GetAPIToken(
		ctx context.Context,
		systemLevel bool,
		project string,
		name string,
	) (*corev1.Secret, error)
	// ListAPITokens lists all bearer tokens associated with a specified Kargo
	// Role.
	ListAPITokens(
		ctx context.Context,
		systemLevel bool,
		project string,
		roleName string,
	) ([]corev1.Secret, error)
}

// rolesDatabase is an implementation of the RolesDatabase interface
// that utilizes a Kubernetes controller runtime client to store and retrieve
// Kargo Roles stored Kubernetes in the form of ServiceAccount/Role/RoleBinding
// trios.
type rolesDatabase struct {
	client client.Client
	cfg    RolesDatabaseConfig
}

// NewKubernetesRolesDatabase returns an implementation of the RolesDatabase
// interface that utilizes a Kubernetes controller runtime client to store and
// retrieve Kargo Roles stored Kubernetes in the form of
// ServiceAccount/Role/RoleBinding trios.
func NewKubernetesRolesDatabase(
	c client.Client,
	cfg RolesDatabaseConfig,
) RolesDatabase {
	return &rolesDatabase{
		client: c,
		cfg:    cfg,
	}
}

// CreateRole implements the RolesDatabase interface.
func (c *rolesDatabase) Create(
	ctx context.Context,
	kargoRole *rbacapi.Role,
) (*rbacapi.Role, error) {
	objKey := client.ObjectKey{
		Namespace: kargoRole.Namespace,
		Name:      kargoRole.Name,
	}

	// Check if the ServiceAccount we would create already exists
	sa := &corev1.ServiceAccount{}
	if err := c.client.Get(ctx, objKey, sa); client.IgnoreNotFound(err) != nil {
		return nil, fmt.Errorf(
			"error getting ServiceAccount %q in namespace %q: %w", kargoRole.Name, kargoRole.Namespace, err,
		)
	} else if err == nil {
		return nil, apierrors.NewAlreadyExists(
			schema.GroupResource{
				Group:    sa.GetObjectKind().GroupVersionKind().Group,
				Resource: strings.ToLower(sa.GetObjectKind().GroupVersionKind().Kind),
			},
			sa.Name,
		)
	}

	// Check if the Role we would create already exists
	role := &rbacv1.Role{}
	if err := c.client.Get(ctx, objKey, role); client.IgnoreNotFound(err) != nil {
		return nil, fmt.Errorf(
			"error getting Role %q in namespace %q: %w", kargoRole.Name, kargoRole.Namespace, err,
		)
	} else if err == nil {
		return nil, apierrors.NewAlreadyExists(
			schema.GroupResource{
				Group:    role.GetObjectKind().GroupVersionKind().Group,
				Resource: strings.ToLower(role.GetObjectKind().GroupVersionKind().Kind),
			},
			role.Name,
		)
	}

	// Check if the RoleBinding we would create already exists
	rb := &rbacv1.RoleBinding{}
	if err := c.client.Get(ctx, objKey, rb); client.IgnoreNotFound(err) != nil {
		return nil, fmt.Errorf(
			"error getting RoleBinding %q in namespace %q: %w", kargoRole.Name, kargoRole.Namespace, err,
		)
	} else if err == nil {
		return nil, apierrors.NewAlreadyExists(
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

	if err = c.client.Create(ctx, sa); err != nil {
		return nil, fmt.Errorf(
			"error creating ServiceAccount %q in namespace %q: %w", kargoRole.Name, kargoRole.Namespace, err,
		)
	}

	if err := c.client.Create(ctx, rb); err != nil {
		return nil, fmt.Errorf(
			"error creating RoleBinding %q in namespace %q: %w", kargoRole.Name, kargoRole.Namespace, err,
		)
	}

	// Note: The Role's rules are already normalized
	if err := c.client.Create(ctx, role); err != nil {
		return nil, fmt.Errorf(
			"error creating Role %q in namespace %q: %w", kargoRole.Name, kargoRole.Namespace, err,
		)
	}

	return kargoRole, nil
}

// DeleteRole implements the RolesDatabase interface.
func (c *rolesDatabase) Delete(
	ctx context.Context,
	project string,
	name string,
) error {
	sa, roles, rbs, err := c.GetAsResources(ctx, false, project, name)
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
		if err := c.client.Delete(ctx, role); err != nil {
			return fmt.Errorf(
				"error deleting Role %q in namespace %q: %w", role.Name, role.Namespace, err,
			)
		}
	}

	if rb != nil {
		if err := c.client.Delete(ctx, rb); err != nil {
			return fmt.Errorf(
				"error deleting RoleBinding %q in namespace %q: %w", rb.Name, rb.Namespace, err,
			)
		}
	}

	// If we got to here, sa cannot have been nil
	if err := c.client.Delete(ctx, sa); err != nil {
		return fmt.Errorf(
			"error deleting ServiceAccount %q in namespace %q: %w", sa.Name, sa.Namespace, err,
		)
	}

	return nil
}

// Get implements the RolesDatabase interface.
func (c *rolesDatabase) Get(
	ctx context.Context,
	systemLevel bool,
	project string,
	name string,
) (*rbacapi.Role, error) {
	sa, roles, rbs, err := c.GetAsResources(ctx, systemLevel, project, name)
	if err != nil {
		return nil, err
	}

	// Note: The underlying resources we found may not be manageable, but we
	// can still return a Kargo Role that summarizes them.

	// Note: The Kargo Role will come back with normalized rules
	return ResourcesToRole(sa, roles, rbs)
}

// GetAsResources implements the RolesDatabase interface.
func (c *rolesDatabase) GetAsResources(
	ctx context.Context,
	systemLevel bool,
	project string,
	name string,
) (*corev1.ServiceAccount, []rbacv1.Role, []rbacv1.RoleBinding, error) {
	namespace := project
	if systemLevel {
		namespace = c.cfg.KargoNamespace
	}
	objKey := client.ObjectKey{Namespace: namespace, Name: name}

	sa := &corev1.ServiceAccount{}
	if err := c.client.Get(ctx, objKey, sa); err != nil {
		return nil, nil, nil, fmt.Errorf(
			"error getting ServiceAccount %q in namespace %q: %w", name, namespace, err,
		)
	}
	// System level roles must be labeled as such
	if systemLevel {
		if sa.Labels[rbacapi.LabelKeySystemRole] != rbacapi.LabelValueTrue {
			return nil, nil, nil, apierrors.NewNotFound(
				rbacapi.GroupVersion.WithResource("Role").GroupResource(), name,
			)
		}
	}

	// Find all RoleBindings in the namespace
	rbList := &rbacv1.RoleBindingList{}
	if err := c.client.List(ctx, rbList, client.InNamespace(namespace)); err != nil {
		return nil, nil, nil, fmt.Errorf(
			"error listing RoleBindings in namespace %q: %w", namespace, err,
		)
	}
	// Narrow the list to just the RoleBindings that reference the ServiceAccount
	rbs := make([]rbacv1.RoleBinding, 0, len(rbList.Items))
	for i := range rbList.Items {
		rb := &rbList.Items[i]
		for _, subject := range rb.Subjects {
			if subject.Kind == rbacv1.ServiceAccountKind &&
				subject.Namespace == namespace &&
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
		if err := c.client.Get(
			ctx, client.ObjectKey{Namespace: namespace, Name: rb.RoleRef.Name},
			role,
		); err != nil {
			return nil, nil, nil, fmt.Errorf(
				"error getting Role %q in namespace %q: %w",
				rb.RoleRef.Name, namespace, err,
			)
		}
		roles = append(roles, *role)
	}

	return sa, roles, rbs, nil
}

// GrantPermissionsToRole implements the RolesDatabase interface.
func (c *rolesDatabase) GrantPermissionsToRole(
	ctx context.Context,
	project string,
	name string,
	resourceDetails *rbacapi.ResourceDetails,
) (*rbacapi.Role, error) {
	sa, roles, rbs, err := c.GetAsResources(ctx, false, project, name)
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

	// Deal with wildcard verb
	for _, verb := range resourceDetails.Verbs {
		if strings.TrimSpace(verb) == "*" {
			resourceDetails.Verbs = append(
				resourceDetails.Verbs,
				allVerbsFor(resourceDetails.ResourceType, true)...,
			)
			break
		}
	}

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
	if newRole.Rules, err = NormalizePolicyRules(append(newRole.Rules, newRule), nil); err != nil {
		return nil, fmt.Errorf("error normalizing RBAC policy rules: %w", err)
	}

	if role == nil {
		if err = c.client.Create(ctx, newRole); err != nil {
			return nil, fmt.Errorf("error creating Role %q in namespace %q: %w", name, project, err)
		}
	} else if err = c.client.Update(ctx, newRole); err != nil {
		return nil, fmt.Errorf("error updating Role %q in namespace %q: %w", name, project, err)
	}

	if rb == nil {
		rb = buildNewRoleBinding(project, name)
		if err = c.client.Create(ctx, rb); err != nil {
			return nil, fmt.Errorf("error creating RoleBinding %q in namespace %q: %w", name, project, err)
		}
	}

	return ResourcesToRole(sa, []rbacv1.Role{*newRole}, []rbacv1.RoleBinding{*rb})
}

// GrantRoleToUsers implements the RolesDatabase interface.
func (c *rolesDatabase) GrantRoleToUsers(
	ctx context.Context,
	project string,
	name string,
	claims []rbacapi.Claim,
) (*rbacapi.Role, error) {
	sa, roles, rbs, err := c.GetAsResources(ctx, false, project, name)
	if err != nil {
		return nil, err
	}
	// This will return an error if these resources are not manageable for any
	// reason.
	role, _, err := manageableResources(*sa, roles, rbs)
	if err != nil {
		return nil, err
	}
	if err = amendClaimAnnotations(sa, claimListToMap(claims)); err != nil {
		return nil, fmt.Errorf("error amending claim annotations: %w", err)
	}
	if err = c.client.Update(ctx, sa); err != nil {
		return nil, fmt.Errorf("error updating ServiceAccount %q in namespace %q: %w", name, project, err)
	}

	if role == nil {
		return ResourcesToRole(sa, nil, rbs)
	}
	return ResourcesToRole(sa, []rbacv1.Role{*role}, rbs)
}

// List implements the RolesDatabase interface.
func (c *rolesDatabase) List(
	ctx context.Context,
	systemLevel bool,
	project string,
) ([]*rbacapi.Role, error) {
	namespace := project
	if systemLevel {
		namespace = c.cfg.KargoNamespace
	}
	listOpts := []client.ListOption{client.InNamespace(namespace)}
	if systemLevel {
		listOpts = append(
			listOpts,
			client.MatchingLabels{rbacapi.LabelKeySystemRole: rbacapi.LabelValueTrue},
		)
	}

	saList := &corev1.ServiceAccountList{}
	if err := c.client.List(ctx, saList, listOpts...); err != nil {
		return nil, fmt.Errorf(
			"error listing ServiceAccounts in namespace %q: %w", namespace, err,
		)
	}

	kargoRoles := make([]*rbacapi.Role, 0, len(saList.Items))
	for i := range saList.Items {
		sa, roles, rbs, err := c.GetAsResources(
			ctx,
			systemLevel,
			namespace,
			saList.Items[i].Name,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"error getting underlying resources for Kargo Role %q from namespace %q: %w",
				saList.Items[i].Name, namespace, err,
			)
		}
		// Note: The underlying resources we found may not be manageable, but we
		// can still return a Kargo Role that summarizes them.
		kargoRole, err := ResourcesToRole(sa, roles, rbs)
		if err != nil {
			return nil, fmt.Errorf(
				"error converting underlying resources to Kargo Role %q: %w",
				sa.Name, err,
			)
		}
		kargoRoles = append(kargoRoles, kargoRole)
	}

	// Sort ascending by name
	slices.SortFunc(kargoRoles, func(lhs, rhs *rbacapi.Role) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	return kargoRoles, nil
}

func (c *rolesDatabase) ListNames(
	ctx context.Context,
	systemLevel bool,
	project string,
) ([]string, error) {
	namespace := project
	if systemLevel {
		namespace = c.cfg.KargoNamespace
	}
	listOpts := []client.ListOption{client.InNamespace(namespace)}
	if systemLevel {
		listOpts = append(
			listOpts,
			client.MatchingLabels{rbacapi.LabelKeySystemRole: rbacapi.LabelValueTrue},
		)
	}

	saList := &corev1.ServiceAccountList{}
	if err := c.client.List(ctx, saList, listOpts...); err != nil {
		return nil, fmt.Errorf(
			"error listing ServiceAccounts in namespace %q: %w", namespace, err,
		)
	}
	names := make([]string, 0, len(saList.Items))
	for _, sa := range saList.Items {
		names = append(names, sa.Name)
	}
	slices.Sort(names)
	return names, nil
}

// RevokePermissionFromRole implements the RolesDatabase interface.
func (c *rolesDatabase) RevokePermissionsFromRole(
	ctx context.Context,
	project string,
	name string,
	resourceDetails *rbacapi.ResourceDetails,
) (*rbacapi.Role, error) {
	sa, roles, rbs, err := c.GetAsResources(ctx, false, project, name)
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
	if role.Rules, err = NormalizePolicyRules(role.Rules, nil); err != nil {
		return nil, fmt.Errorf("error normalizing RBAC policy rules: %w", err)
	}

	// Deal with wildcard verb
	for _, verb := range resourceDetails.Verbs {
		if strings.TrimSpace(verb) == "*" {
			resourceDetails.Verbs = append(
				resourceDetails.Verbs,
				allVerbsFor(resourceDetails.ResourceType, true)...,
			)
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
		ruleResourceName := ""
		if len(rule.ResourceNames) > 0 {
			ruleResourceName = rule.ResourceNames[0]
		}
		if rule.APIGroups[0] != group || rule.Resources[0] != resourceDetails.ResourceType ||
			(resourceDetails.ResourceName != "" && ruleResourceName != resourceDetails.ResourceName) {
			filteredRules = append(filteredRules, rule)
			continue
		}
		rule.Verbs = slices.DeleteFunc(rule.Verbs, func(s string) bool {
			return slices.Contains(resourceDetails.Verbs, s)
		})
		if len(rule.Verbs) > 0 {
			filteredRules = append(filteredRules, rule)
		}
	}
	role.Rules = filteredRules

	if err = c.client.Update(ctx, role); err != nil {
		return nil, fmt.Errorf("error updating Role %q in namespace %q: %w", name, project, err)
	}

	return ResourcesToRole(sa, []rbacv1.Role{*role}, rbs)
}

// RevokeRoleFromUsers implements the RolesDatabase interface.
func (c *rolesDatabase) RevokeRoleFromUsers(
	ctx context.Context,
	project string,
	name string,
	claims []rbacapi.Claim,
) (*rbacapi.Role, error) {
	// Make sure at least part of the ServiceAccount/Role/RoleBinding trio exists
	sa, roles, rbs, err := c.GetAsResources(ctx, false, project, name)
	if err != nil {
		return nil, err
	}
	// This will return an error if these resources are not manageable for any
	// reason.
	role, _, err := manageableResources(*sa, roles, rbs)
	if err != nil {
		return nil, err
	}

	if err = dropFromClaimAnnotations(sa, claimListToMap(claims)); err != nil {
		return nil, fmt.Errorf("error dropping from claim annotations: %w", err)
	}

	if err = c.client.Update(ctx, sa); err != nil {
		return nil, fmt.Errorf("error updating ServiceAccount %q in namespace %q: %w", name, project, err)
	}

	if role == nil {
		return ResourcesToRole(sa, nil, rbs)
	}
	return ResourcesToRole(sa, []rbacv1.Role{*role}, rbs)
}

// Update implements the RolesDatabase interface.
func (c *rolesDatabase) Update(
	ctx context.Context,
	kargoRole *rbacapi.Role,
) (*rbacapi.Role, error) {
	sa, roles, rbs, err := c.GetAsResources(
		ctx,
		false,
		kargoRole.Namespace,
		kargoRole.Name,
	)
	if err != nil {
		return nil, err
	}
	// Narrow down to manageable resources. This will return an error if these
	// resources are not manageable for any reason.
	role, rb, err := manageableResources(*sa, roles, rbs)
	if err != nil {
		return nil, err
	}

	if err = rbacapi.SetOIDCClaimsAnnotation(sa, claimListToMap(kargoRole.Claims)); err != nil {
		return nil, fmt.Errorf("error replacing claim annotations: %w", err)
	}

	if description, ok := kargoRole.Annotations[kargoapi.AnnotationKeyDescription]; ok {
		if sa.Annotations == nil {
			sa.Annotations = map[string]string{}
		}
		sa.Annotations[kargoapi.AnnotationKeyDescription] = description
	} else {
		delete(sa.Annotations, kargoapi.AnnotationKeyDescription)
	}

	if err = c.client.Update(ctx, sa); err != nil {
		return nil, fmt.Errorf(
			"error updating ServiceAccount %q in namespace %q: %w", kargoRole.Name, kargoRole.Namespace, err,
		)
	}

	newRole := role
	if newRole == nil {
		newRole = buildNewRole(kargoRole.Namespace, kargoRole.Name)
	}
	if newRole.Rules, err = NormalizePolicyRules(
		kargoRole.Rules,
		&PolicyRuleNormalizationOptions{IncludeCustomVerbsInExpansion: true},
	); err != nil {
		return nil, fmt.Errorf("error normalizing RBAC policy rules: %w", err)
	}
	if role == nil {
		if err := c.client.Create(ctx, newRole); err != nil {
			return nil, fmt.Errorf("error creating Role %q in namespace %q: %w", kargoRole.Name, kargoRole.Namespace, err)
		}
	} else if err := c.client.Update(ctx, newRole); err != nil {
		return nil, fmt.Errorf("error updating Role %q in namespace %q: %w", kargoRole.Name, kargoRole.Namespace, err)
	}

	if rb == nil {
		rb = buildNewRoleBinding(kargoRole.Namespace, kargoRole.Name)
		if err := c.client.Create(ctx, rb); err != nil {
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

	claims, err := rbacapi.OIDCClaimsFromAnnotationValues(sa.Annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OIDC claims from annotation values: %w", err)
	}

	for name, values := range claims {
		kargoRole.Claims = append(kargoRole.Claims,
			rbacapi.Claim{
				Name:   name,
				Values: values,
			},
		)
	}
	slices.SortFunc(kargoRole.Claims, func(lhs, rhs rbacapi.Claim) int {
		return strings.Compare(lhs.Name, rhs.Name)
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
		if kargoRole.Rules, err = NormalizePolicyRules(kargoRole.Rules, nil); err != nil {
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
	if err := amendClaimAnnotations(sa, claimListToMap(kargoRole.Claims)); err != nil {
		return nil, nil, nil, fmt.Errorf("error amending claim annotations: %w", err)
	}

	role := buildNewRole(kargoRole.Namespace, kargoRole.Name)
	var err error
	if role.Rules, err = NormalizePolicyRules(
		kargoRole.Rules,
		&PolicyRuleNormalizationOptions{IncludeCustomVerbsInExpansion: true},
	); err != nil {
		return nil, nil, nil, fmt.Errorf("error normalizing RBAC policy rules: %w", err)
	}

	rb := buildNewRoleBinding(kargoRole.Namespace, kargoRole.Name)

	return sa, role, rb, nil
}

func claimListToMap(claims []rbacapi.Claim) map[string][]string {
	claimMap := map[string][]string{}
	for _, claim := range claims {
		if _, ok := claimMap[claim.Name]; !ok {
			claimMap[claim.Name] = claim.Values
		} else {
			claimMap[claim.Name] = append(claimMap[claim.Name], claim.Values...)
		}
		slices.Sort(claimMap[claim.Name])
		claimMap[claim.Name] = slices.Compact(claimMap[claim.Name])
	}
	return claimMap
}

func amendClaimAnnotations(sa *corev1.ServiceAccount, claims map[string][]string) error {
	existingClaims, err := rbacapi.OIDCClaimsFromAnnotationValues(sa.Annotations)
	if err != nil {
		return fmt.Errorf("failed to parse OIDC claims from annotation values: %w", err)
	}
	for name, values := range claims {
		existingClaims[name] = append(existingClaims[name], values...)
		slices.Sort(existingClaims[name])
		existingClaims[name] = slices.Compact(existingClaims[name])
	}
	return rbacapi.SetOIDCClaimsAnnotation(sa, existingClaims)
}

func dropFromClaimAnnotations(sa *corev1.ServiceAccount, claims map[string][]string) error {
	existingClaims, err := rbacapi.OIDCClaimsFromAnnotationValues(sa.Annotations)
	if err != nil {
		return fmt.Errorf("failed to parse OIDC claims from annotation values: %w", err)
	}
	for name := range claims {
		if existingValues, ok := existingClaims[name]; ok {
			values := slices.DeleteFunc(existingValues, func(s string) bool {
				return slices.Contains(claims[name], s)
			})
			if len(values) == 0 {
				delete(existingClaims, name)
				continue
			}
			slices.Sort(values)
			existingClaims[name] = slices.Compact(values)
		}
	}
	return rbacapi.SetOIDCClaimsAnnotation(sa, existingClaims)
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

func buildNewRoleBinding(namespace string, roleName string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      roleName,
			Annotations: map[string]string{
				rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     roleName,
		},
		Subjects: []rbacv1.Subject{{
			Kind:      rbacv1.ServiceAccountKind,
			Namespace: namespace,
			Name:      roleName,
		}},
	}
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
		return nil, nil, apierrors.NewBadRequest(
			fmt.Sprintf(
				"ServiceAccount %q in namespace %q is not annotated as Kargo-managed",
				sa.Name, sa.Namespace,
			),
		)
	}
	if len(roles) > 1 {
		return nil, nil, apierrors.NewBadRequest(
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
			return nil, nil, apierrors.NewBadRequest(
				fmt.Sprintf(
					"Role %q in namespace %q is not annotated as Kargo-managed",
					role.Name, role.Namespace,
				),
			)
		}
	}
	if len(rbs) > 1 {
		return nil, nil, apierrors.NewBadRequest(
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
			return nil, nil, apierrors.NewBadRequest(
				fmt.Sprintf(
					"RoleBinding %q in namespace %q is not annotated as Kargo-managed",
					rb.Name, rb.Namespace,
				),
			)
		}
	}
	return role, rb, nil
}

// CreateAPIToken implements RolesDatabase.
func (c *rolesDatabase) CreateAPIToken(
	ctx context.Context,
	systemLevel bool,
	project string,
	roleName string,
	tokenName string,
) (*corev1.Secret, error) {
	namespace := project
	if systemLevel {
		namespace = c.cfg.KargoNamespace
	}
	sa, _, _, err := c.GetAsResources(ctx, systemLevel, project, roleName)
	if err != nil {
		return nil, err
	}
	if systemLevel && sa.Labels[rbacapi.LabelKeySystemRole] != rbacapi.LabelValueTrue {
		return nil, apierrors.NewBadRequest(
			fmt.Sprintf(
				"ServiceAccount %q in namespace %q is not labeled as a system-level Kargo role",
				roleName, namespace,
			),
		)
	}
	tokenSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      tokenName,
			Labels: map[string]string{
				rbacapi.LabelKeyAPIToken: rbacapi.LabelValueTrue,
			},
			Annotations: map[string]string{
				"kubernetes.io/service-account.name": roleName,
				rbacapi.AnnotationKeyManaged:         rbacapi.AnnotationValueTrue,
			},
			// Make sure deleting the ServiceAccount cascades to associated tokens.
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "v1",
				Kind:       "ServiceAccount",
				Name:       roleName,
				UID:        sa.UID,
			}},
		},
		Type: corev1.SecretTypeServiceAccountToken,
	}
	fmt.Println(tokenSecret.OwnerReferences)
	if err = c.client.Create(ctx, tokenSecret); err != nil {
		return nil, fmt.Errorf(
			"error creating token Secret %q for ServiceAccount %q in namespace %q: %w",
			tokenName, roleName, namespace, err,
		)
	}

	// Retrieve Secret -- this is necessary to actually get the token. We wrap
	// this in a retry because token data is created asynchronously and we don't
	// want to prematurely return the Secret without its data.
	tokenSecret, err = c.waitForTokenData(
		ctx,
		namespace,
		tokenName,
		5, // Up to five attempts
	)
	if err != nil {
		return nil, err
	}

	return tokenSecret, nil
}

// waitForTokenData retrieves a token Secret with retry logic. It retries when:
//
//  1. The Secret exists but token data hasn't been populated yet
//  2. Transient errors occur (timeouts, rate limits, server errors, conflicts)
//
// It does NOT retry on permanent errors like NotFound, BadRequest, Forbidden,
// etc.
func (c *rolesDatabase) waitForTokenData(
	ctx context.Context,
	namespace string,
	tokenName string,
	maxAttempts int,
) (*corev1.Secret, error) {
	var tokenSecret *corev1.Secret
	backoff := retry.DefaultBackoff
	backoff.Steps = maxAttempts

	if err := retry.OnError(
		backoff,
		func(innerErr error) bool {
			if innerErr == nil {
				return false // Stop retrying if no error
			}
			// Retry on transient errors
			_, isTokenNotPopulatedErr := innerErr.(*errTokenNotPopulated)
			return isTokenNotPopulatedErr ||
				apierrors.IsServerTimeout(innerErr) ||
				apierrors.IsTimeout(innerErr) ||
				apierrors.IsTooManyRequests(innerErr) ||
				apierrors.IsServiceUnavailable(innerErr) ||
				apierrors.IsInternalError(innerErr) ||
				apierrors.IsConflict(innerErr)
		},
		func() error {
			tokenSecret = &corev1.Secret{}
			if innerErr := c.client.Get(
				ctx,
				client.ObjectKey{
					Namespace: namespace,
					Name:      tokenName,
				},
				tokenSecret,
			); innerErr != nil {
				return innerErr
			}
			if _, gotToken := tokenSecret.Data["token"]; !gotToken {
				return &errTokenNotPopulated{}
			}
			return nil
		},
	); err != nil {
		return nil, fmt.Errorf(
			"error while waiting for token Secret %q in namespace %q to be "+
				"populated: %w",
			tokenName, namespace, err,
		)
	}

	return tokenSecret, nil
}

// DeleteAPIToken implements RolesDatabase.
func (c *rolesDatabase) DeleteAPIToken(
	ctx context.Context,
	systemLevel bool,
	project string,
	name string,
) error {
	namespace := project
	if systemLevel {
		namespace = c.cfg.KargoNamespace
	}
	tokenSecret := &corev1.Secret{}
	if err := c.client.Get(
		ctx,
		client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		},
		tokenSecret,
	); err != nil {
		return fmt.Errorf(
			"error getting token Secret %q in namespace %q: %w", name, namespace, err,
		)
	}
	if tokenSecret.Type != corev1.SecretTypeServiceAccountToken {
		return apierrors.NewConflict(
			corev1.SchemeGroupVersion.WithResource("secrets").GroupResource(),
			name,
			fmt.Errorf( // nolint: staticcheck
				"Kubernetes Secret %q in namespace %q is not a service account "+
					"token",
				tokenSecret.Name, tokenSecret.Namespace,
			),
		)
	}
	if _, ok := tokenSecret.Annotations["kubernetes.io/service-account.name"]; !ok {
		return apierrors.NewConflict(
			corev1.SchemeGroupVersion.WithResource("secrets").GroupResource(),
			name,
			fmt.Errorf( // nolint: staticcheck
				"Kubernetes Secret %q in namespace %q is missing the service account "+
					"annotation",
				tokenSecret.Name, tokenSecret.Namespace,
			),
		)
	}
	if !isKargoAPIToken(tokenSecret) {
		return apierrors.NewConflict(
			corev1.SchemeGroupVersion.WithResource("secrets").GroupResource(),
			name,
			fmt.Errorf( // nolint: staticcheck
				"Kubernetes Secret %q in namespace %q is not labeled as a Kargo "+
					"API token",
				tokenSecret.Name, tokenSecret.Namespace,
			),
		)
	}
	if !isKargoManaged(tokenSecret) {
		return apierrors.NewConflict(
			corev1.SchemeGroupVersion.WithResource("secrets").GroupResource(),
			name,
			fmt.Errorf( // nolint: staticcheck
				"Kubernetes Secret %q in namespace %q is not annotated as Kargo-managed",
				tokenSecret.Name, tokenSecret.Namespace,
			),
		)
	}
	if err := c.client.Delete(ctx, tokenSecret); err != nil {
		return fmt.Errorf(
			"error deleting token Secret %q in namespace %q: %w",
			tokenSecret.Name, tokenSecret.Namespace, err,
		)
	}
	return nil
}

// GetAPIToken implements RolesDatabase.
func (c *rolesDatabase) GetAPIToken(
	ctx context.Context,
	systemLevel bool,
	project string,
	name string,
) (*corev1.Secret, error) {
	namespace := project
	if systemLevel {
		namespace = c.cfg.KargoNamespace
	}
	tokenSecret := &corev1.Secret{}
	if err := c.client.Get(
		ctx,
		client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		},
		tokenSecret,
	); err != nil {
		return nil, fmt.Errorf(
			"error getting token Secret %q in namespace %q: %w", name, namespace, err,
		)
	}
	if !isKargoAPIToken(tokenSecret) {
		return nil, apierrors.NewBadRequest(
			fmt.Sprintf(
				"Kubernetes Secret %q in namespace %q is not labeled as a Kargo "+
					"API token",
				tokenSecret.Name, tokenSecret.Namespace,
			),
		)
	}
	redactTokenData(tokenSecret)
	return tokenSecret, nil
}

// ListAPITokens implements RolesDatabase.
func (c *rolesDatabase) ListAPITokens(
	ctx context.Context,
	systemLevel bool,
	project string,
	roleName string,
) ([]corev1.Secret, error) {
	namespace := project
	if systemLevel {
		namespace = c.cfg.KargoNamespace
	}
	if roleName != "" {
		if _, err := c.Get(ctx, systemLevel, project, roleName); err != nil {
			return nil, err
		}
	}
	tokenSecretList := &corev1.SecretList{}
	if err := c.client.List(
		ctx,
		tokenSecretList,
		client.InNamespace(namespace),
		client.MatchingLabels{rbacapi.LabelKeyAPIToken: rbacapi.LabelValueTrue},
	); err != nil {
		return nil, fmt.Errorf(
			"error listing token Secrets for ServiceAccount %q in namespace %q: %w",
			roleName, namespace, err,
		)
	}
	var tokenSecrets []corev1.Secret
	for _, tokenSecret := range tokenSecretList.Items {
		if isKargoAPIToken(&tokenSecret) &&
			(roleName == "" || tokenSecret.Annotations["kubernetes.io/service-account.name"] == roleName) {
			redactTokenData(&tokenSecret)
			tokenSecrets = append(tokenSecrets, tokenSecret)
		}
	}
	return tokenSecrets, nil
}

func isKargoAPIToken(secret *corev1.Secret) bool {
	return secret.Type == corev1.SecretTypeServiceAccountToken &&
		secret.Labels[rbacapi.LabelKeyAPIToken] == rbacapi.LabelValueTrue
}

func redactTokenData(tokenSecret *corev1.Secret) {
	if _, ok := tokenSecret.Data["token"]; ok {
		tokenSecret.Data["token"] = []byte("*** REDACTED ***")
	}
}

type errTokenNotPopulated struct{}

func (e *errTokenNotPopulated) Error() string {
	return "did not find token data"
}
