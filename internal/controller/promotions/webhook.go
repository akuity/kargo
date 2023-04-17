package promotions

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	authenticationv1 "k8s.io/api/authentication/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	api "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/config"
	"github.com/akuity/kargo/internal/logging"
)

type webhook struct {
	client client.Client
	config config.ControllerConfig

	// The following behaviors are overridable for testing purposes:

	authorizeFn func(
		ctx context.Context,
		promo *api.Promotion,
		action string,
	) error

	listPoliciesFn func(
		context.Context,
		client.ObjectList,
		...client.ListOption,
	) error

	admissionRequestFromContextFn func(context.Context) (admission.Request, error)

	isSubjectAuthorizedFn func(
		context.Context,
		*api.PromotionPolicy,
		*api.Promotion,
		authenticationv1.UserInfo,
	) (bool, error)

	getSubjectRolesFn func(
		ctx context.Context,
		subjectInfo authenticationv1.UserInfo,
		namespace string,
	) (map[string]struct{}, error)

	listRoleBindingsFn func(
		context.Context,
		client.ObjectList,
		...client.ListOption,
	) error
}

func SetupWebhookWithManager(
	ctx context.Context,
	mgr ctrl.Manager,
	config config.ControllerConfig,
) error {
	if err := mgr.GetFieldIndexer().IndexField(
		ctx,
		&api.PromotionPolicy{},
		"environment",
		func(obj client.Object) []string {
			policy := obj.(*api.PromotionPolicy) // nolint: forcetypeassert
			return []string{policy.Environment}
		},
	); err != nil {
		return errors.Wrap(err, "error indexing Secrets by repo")
	}
	w := &webhook{
		client: mgr.GetClient(),
		config: config,
	}
	w.authorizeFn = w.authorize
	w.listPoliciesFn = w.client.List
	w.admissionRequestFromContextFn = admission.RequestFromContext
	w.isSubjectAuthorizedFn = w.isSubjectAuthorized
	w.getSubjectRolesFn = w.getSubjectRoles
	w.listRoleBindingsFn = w.client.List
	return ctrl.NewWebhookManagedBy(mgr).
		For(&api.Promotion{}).
		WithValidator(w).
		Complete()
}

func (w *webhook) ValidateCreate(
	ctx context.Context,
	obj runtime.Object,
) error {
	return w.authorizeFn(ctx, obj.(*api.Promotion), "create")
}

func (w *webhook) ValidateUpdate(
	ctx context.Context,
	oldObj runtime.Object,
	newObj runtime.Object,
) error {
	promo := newObj.(*api.Promotion)

	if err := w.authorizeFn(ctx, promo, "update"); err != nil {
		return err
	}

	// PromotionSpecs are meant to be immutable
	if *promo.Spec != *(oldObj.(*api.Promotion).Spec) {
		return apierrors.NewInvalid(
			schema.GroupKind{
				Group: api.GroupVersion.Group,
				Kind:  "Promotion",
			},
			promo.Name,
			field.ErrorList{
				field.Invalid(
					field.NewPath("spec"),
					promo.Spec,
					"spec is immutable",
				),
			},
		)
	}
	return nil
}

func (w *webhook) ValidateDelete(
	ctx context.Context,
	obj runtime.Object,
) error {
	logger := logging.LoggerFromContext(ctx)

	promo := obj.(*api.Promotion)

	// Special logic for delete only. Allow any delete by the Kubernetes namespace
	// controller. This prevents the webhook from stopping a namespace from being
	// cleaned up.
	req, err := w.admissionRequestFromContextFn(ctx)
	if err != nil {
		logger.Error(err)
		return apierrors.NewForbidden(
			schema.GroupResource{
				Group:    api.GroupVersion.Group,
				Resource: "Promotion",
			},
			promo.Name,
			errors.New(
				"error retrieving admission request from context; refusing to "+
					"delete Promotion",
			),
		)
	}
	serviceAccountNamespace, serviceAccountName :=
		getServiceAccountNamespaceAndName(req.UserInfo.Username)
	subjectIsServiceAccount := serviceAccountName != ""
	if subjectIsServiceAccount &&
		serviceAccountNamespace == "kube-system" &&
		serviceAccountName == "namespace-controller" {
		return nil
	}

	return w.authorizeFn(ctx, promo, "delete")
}

func (w *webhook) authorize(
	ctx context.Context,
	promo *api.Promotion,
	action string,
) error {
	logger := logging.LoggerFromContext(ctx)

	groupResource := schema.GroupResource{
		Group:    api.GroupVersion.Group,
		Resource: "Promotion",
	}

	policies := api.PromotionPolicyList{}
	if err := w.listPoliciesFn(
		ctx,
		&policies,
		&client.ListOptions{
			Namespace: promo.Namespace,
			FieldSelector: fields.Set(map[string]string{
				"environment": promo.Spec.Environment,
			}).AsSelector(),
		},
	); err != nil {
		logger.Error(err)
		return apierrors.NewForbidden(
			groupResource,
			promo.Name,
			errors.Errorf(
				"error listing PromotionPolicies associated with "+
					"Environment %q in namespace %q; refusing to %s Promotion",
				promo.Spec.Environment,
				promo.Namespace,
				action,
			),
		)
	}

	// TODO: Make the behavior for this case configurable?
	if len(policies.Items) == 0 {
		return apierrors.NewForbidden(
			groupResource,
			promo.Name,
			errors.Errorf(
				"no PromotionPolicy associated with "+
					"Environment %q in namespace %q; refusing to %s Promotion",
				promo.Spec.Environment,
				promo.Namespace,
				action,
			),
		)
	}

	if len(policies.Items) > 1 {
		return apierrors.NewForbidden(
			groupResource,
			promo.Name,
			errors.Errorf(
				"found multiple PromotionPolicies associated with "+
					"Environment %q in namespace %q; refusing to %s Promotion",
				promo.Spec.Environment,
				promo.Namespace,
				action,
			),
		)
	}

	// If we get to here, there's just one PromotionPolicy...

	policy := &policies.Items[0]

	req, err := w.admissionRequestFromContextFn(ctx)
	if err != nil {
		logger.Error(err)
		return apierrors.NewForbidden(
			groupResource,
			promo.Name,
			errors.Errorf(
				"error retrieving admission request from context; refusing to "+
					"%s Promotion",
				action,
			),
		)
	}

	authorized, err := w.isSubjectAuthorizedFn(ctx, policy, promo, req.UserInfo)
	if err != nil {
		logger.Error(err)
		return apierrors.NewForbidden(
			groupResource,
			promo.Name,
			errors.Errorf(
				"error evaluating subject authorities; refusing to %s Promotion",
				action,
			),
		)
	}
	if !authorized {
		return apierrors.NewForbidden(
			groupResource,
			promo.Name,
			errors.Errorf(
				"PromotionPolicy %q in namespace %q does not permit this subject to "+
					"%s Promotions for Environment %q",
				policy.Name,
				policy.Namespace,
				action,
				promo.Spec.Environment,
			),
		)
	}

	return nil
}

func (w *webhook) isSubjectAuthorized(
	ctx context.Context,
	policy *api.PromotionPolicy,
	promo *api.Promotion,
	subjectInfo authenticationv1.UserInfo,
) (bool, error) {
	serviceAccountNamespace, serviceAccountName :=
		getServiceAccountNamespaceAndName(subjectInfo.Username)
	subjectIsServiceAccount := serviceAccountName != ""

	// Special logic that always permits operations by Kargo itself
	if subjectIsServiceAccount &&
		serviceAccountNamespace == w.config.ServiceAccountNamespace &&
		serviceAccountName == w.config.ServiceAccount {
		return true, nil
	}

	subjectRoles, err := w.getSubjectRolesFn(ctx, subjectInfo, promo.Namespace)
	if err != nil {
		return false, errors.Wrap(err, "error retrieving subject Roles")
	}

	for _, authorizedPromoter := range policy.AuthorizedPromoters {
		switch authorizedPromoter.SubjectType {
		case api.AuthorizedPromoterSubjectTypeUser:
			if !subjectIsServiceAccount &&
				subjectInfo.Username == authorizedPromoter.Name {
				return true, nil
			}
		case api.AuthorizedPromoterSubjectTypeServiceAccount:
			if subjectIsServiceAccount &&
				serviceAccountName == authorizedPromoter.Name {
				return true, nil
			}
		case api.AuthorizedPromoterSubjectTypeGroup:
			if subjectHasGroup(subjectInfo, authorizedPromoter.Name) {
				return true, nil
			}
		case api.AuthorizedPromoterSubjectTypeRole:
			if _, hasRole := subjectRoles[authorizedPromoter.Name]; hasRole {
				return true, nil
			}
		}
	}

	return false, nil
}

func (w *webhook) getSubjectRoles(
	ctx context.Context,
	subjectInfo authenticationv1.UserInfo,
	namespace string,
) (map[string]struct{}, error) {
	serviceAccountNamespace, serviceAccountName :=
		getServiceAccountNamespaceAndName(subjectInfo.Username)
	subjectIsServiceAccount := serviceAccountName != ""

	roleBindings := rbacv1.RoleBindingList{}
	if err := w.listRoleBindingsFn(
		ctx,
		&roleBindings,
		&client.ListOptions{
			Namespace: namespace,
		},
	); err != nil {
		return nil,
			errors.Wrapf(err, "error listing RoleBindings in namespace %q", namespace)
	}

	subjectRoles := map[string]struct{}{}
subjectRolesLoop:
	for _, roleBinding := range roleBindings.Items {
		if roleBinding.RoleRef.Kind != "Role" { // Uninterested in ClusterRoles
			continue
		}
		for _, subject := range roleBinding.Subjects {
			switch subject.Kind {
			case "User":
				if !subjectIsServiceAccount && subjectInfo.Username == subject.Name {
					subjectRoles[roleBinding.RoleRef.Name] = struct{}{}
					continue subjectRolesLoop
				}
			case "ServiceAccount":
				if subjectIsServiceAccount &&
					serviceAccountNamespace == subject.Namespace &&
					serviceAccountName == subject.Name {
					subjectRoles[roleBinding.RoleRef.Name] = struct{}{}
					continue subjectRolesLoop
				}
			case "Group":
				if subjectHasGroup(subjectInfo, subject.Name) {
					subjectRoles[roleBinding.RoleRef.Name] = struct{}{}
					continue subjectRolesLoop
				}
			}
		}
	}

	return subjectRoles, nil
}

func subjectHasGroup(subjectInfo authenticationv1.UserInfo, group string) bool {
	for _, subjectGroup := range subjectInfo.Groups {
		if subjectGroup == group {
			return true
		}
	}
	return false
}

func getServiceAccountNamespaceAndName(username string) (string, string) {
	// Usernames for service accounts conform to:
	//   system:serviceaccount:<namespace>:<name>
	if !strings.HasPrefix(username, "system:serviceaccount") {
		return "", ""
	}
	parts := strings.Split(username, ":")
	if len(parts) != 4 {
		return "", ""
	}
	return parts[2], parts[3]
}
