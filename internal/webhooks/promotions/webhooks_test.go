package promotions

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	api "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/config"
)

func TestValidateCreate(t *testing.T) {
	w := &webhook{
		authorizeFn: func(context.Context, *api.Promotion, string) error {
			return nil // Always authorize
		},
	}
	require.NoError(t, w.ValidateCreate(context.Background(), &api.Promotion{}))
}

func TestValidateUpdate(t *testing.T) {
	testCases := []struct {
		name        string
		setup       func() (*api.Promotion, *api.Promotion)
		authorizeFn func(
			ctx context.Context,
			promo *api.Promotion,
			action string,
		) error
		assertions func(error)
	}{
		{
			name: "authorization error",
			setup: func() (*api.Promotion, *api.Promotion) {
				return &api.Promotion{}, &api.Promotion{}
			},
			authorizeFn: func(context.Context, *api.Promotion, string) error {
				return errors.New("something went wrong")
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "attempt to mutate",
			setup: func() (*api.Promotion, *api.Promotion) {
				oldPromo := &api.Promotion{
					ObjectMeta: v1.ObjectMeta{
						Name:      "fake-name",
						Namespace: "fake-namespace",
					},
					Spec: &api.PromotionSpec{
						Environment: "fake-environment",
						State:       "fake-state",
					},
				}
				newPromo := oldPromo.DeepCopy()
				newPromo.Spec.State = "another-fake-state"
				return oldPromo, newPromo
			},
			authorizeFn: func(context.Context, *api.Promotion, string) error {
				return nil
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "\"fake-name\" is invalid")
				require.Contains(t, err.Error(), "spec is immutable")
			},
		},

		{
			name: "update without mutation",
			setup: func() (*api.Promotion, *api.Promotion) {
				oldPromo := &api.Promotion{
					ObjectMeta: v1.ObjectMeta{
						Name:      "fake-name",
						Namespace: "fake-namespace",
					},
					Spec: &api.PromotionSpec{
						Environment: "fake-environment",
						State:       "fake-state",
					},
				}
				newPromo := oldPromo.DeepCopy()
				return oldPromo, newPromo
			},
			authorizeFn: func(context.Context, *api.Promotion, string) error {
				return nil
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			w := &webhook{
				authorizeFn: testCase.authorizeFn,
			}
			oldPromo, newPromo := testCase.setup()
			testCase.assertions(
				w.ValidateUpdate(context.Background(), oldPromo, newPromo),
			)
		})
	}
}

func TestValidateDelete(t *testing.T) {
	testCases := []struct {
		name                          string
		admissionRequestFromContextFn func(
			context.Context,
		) (admission.Request, error)
		authorizeFn func(
			context.Context,
			*api.Promotion,
			string,
		) error
		assertions func(error)
	}{
		{
			name: "error getting admission request bound to context",
			admissionRequestFromContextFn: func(
				context.Context,
			) (admission.Request, error) {
				return admission.Request{}, errors.New("something went wrong")
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.True(t, apierrors.IsForbidden(err))
				require.Contains(
					t,
					err.Error(),
					"error retrieving admission request from context",
				)
			},
		},
		{
			name: "user is namespace controller service account",
			admissionRequestFromContextFn: func(
				context.Context,
			) (admission.Request, error) {
				return admission.Request{
					AdmissionRequest: admissionv1.AdmissionRequest{
						UserInfo: authenticationv1.UserInfo{
							Username: "system:serviceaccount:kube-system:namespace-controller", // nolint: lll
						},
					},
				}, nil
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "user is not authorized",
			admissionRequestFromContextFn: func(
				context.Context,
			) (admission.Request, error) {
				return admission.Request{}, nil
			},
			authorizeFn: func(context.Context, *api.Promotion, string) error {
				return errors.Errorf("not authorized")
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Equal(t, "not authorized", err.Error())
			},
		},
		{
			name: "user is authorized",
			admissionRequestFromContextFn: func(
				context.Context,
			) (admission.Request, error) {
				return admission.Request{}, nil
			},
			authorizeFn: func(context.Context, *api.Promotion, string) error {
				return nil
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			w := &webhook{
				admissionRequestFromContextFn: testCase.admissionRequestFromContextFn,
				authorizeFn:                   testCase.authorizeFn,
			}
			testCase.assertions(
				w.ValidateDelete(context.Background(), &api.Promotion{}),
			)
		})
	}
}

func TestAuthorize(t *testing.T) {
	testCases := []struct {
		name           string
		listPoliciesFn func(
			context.Context,
			client.ObjectList,
			...client.ListOption,
		) error
		admissionRequestFromContextFn func(context.Context) (
			admission.Request,
			error,
		)
		isSubjectAuthorizedFn func(
			context.Context,
			*api.PromotionPolicy,
			*api.Promotion,
			authenticationv1.UserInfo,
		) (bool, error)
		assertions func(err error)
	}{
		{
			name: "error listing promotion policies",
			listPoliciesFn: func(
				context.Context,
				client.ObjectList,
				...client.ListOption,
			) error {
				return errors.New("something went wrong")
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error listing PromotionPolicies associated with",
				)
				require.Contains(t, err.Error(), "refusing to")
			},
		},
		{
			name: "no promotion policy",
			listPoliciesFn: func(
				context.Context,
				client.ObjectList,
				...client.ListOption,
			) error {
				return nil
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"no PromotionPolicy associated with Environment",
				)
				require.Contains(t, err.Error(), "refusing to")
			},
		},
		{
			name: "multiple promotion policies",
			listPoliciesFn: func(
				_ context.Context,
				objs client.ObjectList,
				_ ...client.ListOption,
			) error {
				policies := objs.(*api.PromotionPolicyList) // nolint: forcetypeassert
				policies.Items = []api.PromotionPolicy{{}, {}}
				return nil
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"found multiple PromotionPolicies associated with Environment",
				)
				require.Contains(t, err.Error(), "refusing to")
			},
		},
		{
			name: "error getting admission request bound to context",
			listPoliciesFn: func(
				_ context.Context,
				objs client.ObjectList,
				_ ...client.ListOption,
			) error {
				policies := objs.(*api.PromotionPolicyList) // nolint: forcetypeassert
				policies.Items = []api.PromotionPolicy{{}}
				return nil
			},
			admissionRequestFromContextFn: func(
				context.Context,
			) (admission.Request, error) {
				return admission.Request{}, errors.New("something went wrong")
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error retrieving admission request from context; refusing to",
				)
			},
		},
		{
			name: "error determining if subject is authorized",
			listPoliciesFn: func(
				_ context.Context,
				objs client.ObjectList,
				_ ...client.ListOption,
			) error {
				policies := objs.(*api.PromotionPolicyList) // nolint: forcetypeassert
				policies.Items = []api.PromotionPolicy{{}}
				return nil
			},
			admissionRequestFromContextFn: func(
				context.Context,
			) (admission.Request, error) {
				return admission.Request{}, nil
			},
			isSubjectAuthorizedFn: func(
				context.Context,
				*api.PromotionPolicy,
				*api.Promotion,
				authenticationv1.UserInfo,
			) (bool, error) {
				return false, errors.New("something went wrong")
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error evaluating subject authorities; refusing to",
				)
			},
		},
		{
			name: "subject is not authorized",
			listPoliciesFn: func(
				_ context.Context,
				objs client.ObjectList,
				_ ...client.ListOption,
			) error {
				policies := objs.(*api.PromotionPolicyList) // nolint: forcetypeassert
				policies.Items = []api.PromotionPolicy{{}}
				return nil
			},
			admissionRequestFromContextFn: func(
				context.Context,
			) (admission.Request, error) {
				return admission.Request{}, nil
			},
			isSubjectAuthorizedFn: func(
				context.Context,
				*api.PromotionPolicy,
				*api.Promotion,
				authenticationv1.UserInfo,
			) (bool, error) {
				return false, nil
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "does not permit subject")
			},
		},
		{
			name: "subject is authorized",
			listPoliciesFn: func(
				_ context.Context,
				objs client.ObjectList,
				_ ...client.ListOption,
			) error {
				policies := objs.(*api.PromotionPolicyList) // nolint: forcetypeassert
				policies.Items = []api.PromotionPolicy{{}}
				return nil
			},
			admissionRequestFromContextFn: func(
				context.Context,
			) (admission.Request, error) {
				return admission.Request{}, nil
			},
			isSubjectAuthorizedFn: func(
				context.Context,
				*api.PromotionPolicy,
				*api.Promotion,
				authenticationv1.UserInfo,
			) (bool, error) {
				return true, nil
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			w := &webhook{
				listPoliciesFn:                testCase.listPoliciesFn,
				admissionRequestFromContextFn: testCase.admissionRequestFromContextFn,
				isSubjectAuthorizedFn:         testCase.isSubjectAuthorizedFn,
			}
			testCase.assertions(
				w.authorize(
					context.Background(),
					&api.Promotion{
						ObjectMeta: v1.ObjectMeta{
							Name:      "fake-promotion",
							Namespace: "fake-namespace",
						},
						Spec: &api.PromotionSpec{
							Environment: "fake-environment",
						},
					},
					"create",
				),
			)
		})
	}
}

func TestIsSubjectAuthorized(t *testing.T) {
	testCases := []struct {
		name              string
		policy            *api.PromotionPolicy
		promotion         *api.Promotion
		subjectInfo       authenticationv1.UserInfo
		getSubjectRolesFn func(
			ctx context.Context,
			subjectInfo authenticationv1.UserInfo,
			namespace string,
		) (map[string]struct{}, error)
		assertions func(authorized bool, err error)
	}{
		{
			name: "subject is kargo controller's own service account",
			subjectInfo: authenticationv1.UserInfo{
				Username: "system:serviceaccount:kargo:kargo-controller",
			},
			assertions: func(authorized bool, err error) {
				require.NoError(t, err)
				require.True(t, authorized)
			},
		},
		{
			name: "error getting subject's roles",
			getSubjectRolesFn: func(
				context.Context,
				authenticationv1.UserInfo,
				string,
			) (map[string]struct{}, error) {
				return nil, errors.New("something went wrong")
			},
			promotion: &api.Promotion{
				ObjectMeta: v1.ObjectMeta{
					Name:      "fake-promotion",
					Namespace: "fake-namespace",
				},
			},
			assertions: func(authorized bool, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(t, err.Error(), "error retrieving subject Roles")
				require.False(t, authorized)
			},
		},
		{
			name: "subject is user named directly as promoter",
			policy: &api.PromotionPolicy{
				AuthorizedPromoters: []api.AuthorizedPromoter{
					{
						SubjectType: api.AuthorizedPromoterSubjectTypeUser,
						Name:        "fake-user",
					},
				},
			},
			promotion: &api.Promotion{},
			subjectInfo: authenticationv1.UserInfo{
				Username: "fake-user",
			},
			getSubjectRolesFn: func(
				context.Context,
				authenticationv1.UserInfo,
				string,
			) (map[string]struct{}, error) {
				return nil, nil
			},
			assertions: func(authorized bool, err error) {
				require.NoError(t, err)
				require.True(t, authorized)
			},
		},
		{
			name: "subject is service account named directly as promoter",
			policy: &api.PromotionPolicy{
				AuthorizedPromoters: []api.AuthorizedPromoter{
					{
						SubjectType: api.AuthorizedPromoterSubjectTypeServiceAccount,
						Name:        "fake-service-account",
					},
				},
			},
			promotion: &api.Promotion{},
			subjectInfo: authenticationv1.UserInfo{
				Username: "system:serviceaccount:fake-namespace:fake-service-account",
			},
			getSubjectRolesFn: func(
				context.Context,
				authenticationv1.UserInfo,
				string,
			) (map[string]struct{}, error) {
				return nil, nil
			},
			assertions: func(authorized bool, err error) {
				require.NoError(t, err)
				require.True(t, authorized)
			},
		},
		{
			name: "subject has group that is a promoter",
			policy: &api.PromotionPolicy{
				AuthorizedPromoters: []api.AuthorizedPromoter{
					{
						SubjectType: api.AuthorizedPromoterSubjectTypeGroup,
						Name:        "fake-group",
					},
				},
			},
			promotion: &api.Promotion{},
			subjectInfo: authenticationv1.UserInfo{
				Username: "fake-user",
				Groups:   []string{"fake-group"},
			},
			getSubjectRolesFn: func(
				context.Context,
				authenticationv1.UserInfo,
				string,
			) (map[string]struct{}, error) {
				return nil, nil
			},
			assertions: func(authorized bool, err error) {
				require.NoError(t, err)
				require.True(t, authorized)
			},
		},
		{
			name: "subject has role that is a promoter",
			policy: &api.PromotionPolicy{
				AuthorizedPromoters: []api.AuthorizedPromoter{
					{
						SubjectType: api.AuthorizedPromoterSubjectTypeRole,
						Name:        "fake-role",
					},
				},
			},
			promotion: &api.Promotion{},
			subjectInfo: authenticationv1.UserInfo{
				Username: "fake-user",
			},
			getSubjectRolesFn: func(
				context.Context,
				authenticationv1.UserInfo,
				string,
			) (map[string]struct{}, error) {
				return map[string]struct{}{
					"fake-role": {},
				}, nil
			},
			assertions: func(authorized bool, err error) {
				require.NoError(t, err)
				require.True(t, authorized)
			},
		},
		{
			name:      "subject is not authorized",
			policy:    &api.PromotionPolicy{},
			promotion: &api.Promotion{},
			subjectInfo: authenticationv1.UserInfo{
				Username: "fake-user",
			},
			getSubjectRolesFn: func(
				context.Context,
				authenticationv1.UserInfo,
				string,
			) (map[string]struct{}, error) {
				return nil, nil
			},
			assertions: func(authorized bool, err error) {
				require.NoError(t, err)
				require.False(t, authorized)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			w := &webhook{
				config: config.WebhooksConfig{
					ServiceAccountNamespace: "kargo",
					ServiceAccount:          "kargo-controller",
				},
				getSubjectRolesFn: testCase.getSubjectRolesFn,
			}
			testCase.assertions(
				w.isSubjectAuthorized(
					context.Background(),
					testCase.policy,
					testCase.promotion,
					testCase.subjectInfo,
				),
			)
		})
	}
}

func TestGetSubjectRoles(t *testing.T) {
	testCases := []struct {
		name               string
		subjectInfo        authenticationv1.UserInfo
		listRoleBindingsFn func(
			context.Context,
			client.ObjectList,
			...client.ListOption,
		) error
		assertions func(map[string]struct{}, error)
	}{
		{
			name: "error listing role bindings",
			listRoleBindingsFn: func(
				context.Context,
				client.ObjectList,
				...client.ListOption,
			) error {
				return errors.New("something went wrong")
			},
			assertions: func(_ map[string]struct{}, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(
					t,
					err.Error(),
					"error listing RoleBindings in namespace",
				)
			},
		},
		{
			name: "only binding is to a cluster role",
			listRoleBindingsFn: func(
				_ context.Context,
				objs client.ObjectList,
				_ ...client.ListOption,
			) error {
				// nolint: forcetypeassert
				roleBindings := objs.(*rbacv1.RoleBindingList)
				roleBindings.Items = []rbacv1.RoleBinding{
					{
						RoleRef: rbacv1.RoleRef{
							Kind: "ClusterRole",
						},
					},
				}
				return nil
			},
			assertions: func(roles map[string]struct{}, err error) {
				require.NoError(t, err)
				require.Empty(t, roles)
			},
		},
		{
			name: "user has role directly",
			subjectInfo: authenticationv1.UserInfo{
				Username: "fake-user",
			},
			listRoleBindingsFn: func(
				_ context.Context,
				objs client.ObjectList,
				_ ...client.ListOption,
			) error {
				// nolint: forcetypeassert
				roleBindings := objs.(*rbacv1.RoleBindingList)
				roleBindings.Items = []rbacv1.RoleBinding{
					{
						RoleRef: rbacv1.RoleRef{
							Kind: "Role",
							Name: "fake-role",
						},
						Subjects: []rbacv1.Subject{
							{
								Kind: "User",
								Name: "fake-user",
							},
						},
					},
				}
				return nil
			},
			assertions: func(roles map[string]struct{}, err error) {
				require.NoError(t, err)
				require.Len(t, roles, 1)
				require.Contains(t, roles, "fake-role")
			},
		},
		{
			name: "user has role via group",
			subjectInfo: authenticationv1.UserInfo{
				Username: "fake-user",
				Groups:   []string{"fake-group"},
			},
			listRoleBindingsFn: func(
				_ context.Context,
				objs client.ObjectList,
				_ ...client.ListOption,
			) error {
				// nolint: forcetypeassert
				roleBindings := objs.(*rbacv1.RoleBindingList)
				roleBindings.Items = []rbacv1.RoleBinding{
					{
						RoleRef: rbacv1.RoleRef{
							Kind: "Role",
							Name: "fake-role",
						},
						Subjects: []rbacv1.Subject{
							{
								Kind: "Group",
								Name: "fake-group",
							},
						},
					},
				}
				return nil
			},
			assertions: func(roles map[string]struct{}, err error) {
				require.NoError(t, err)
				require.Len(t, roles, 1)
				require.Contains(t, roles, "fake-role")
			},
		},
		{
			name: "service account has role directly",
			subjectInfo: authenticationv1.UserInfo{
				Username: "system:serviceaccount:fake-namespace:fake-service-account",
			},
			listRoleBindingsFn: func(
				_ context.Context,
				objs client.ObjectList,
				_ ...client.ListOption,
			) error {
				// nolint: forcetypeassert
				roleBindings := objs.(*rbacv1.RoleBindingList)
				roleBindings.Items = []rbacv1.RoleBinding{
					{
						RoleRef: rbacv1.RoleRef{
							Kind: "Role",
							Name: "fake-role",
						},
						Subjects: []rbacv1.Subject{
							{
								Kind:      "ServiceAccount",
								Namespace: "fake-namespace",
								Name:      "fake-service-account",
							},
						},
					},
				}
				return nil
			},
			assertions: func(roles map[string]struct{}, err error) {
				require.NoError(t, err)
				require.Len(t, roles, 1)
				require.Contains(t, roles, "fake-role")
			},
		},
		{
			name: "service account has role via group",
			subjectInfo: authenticationv1.UserInfo{
				Username: "system:serviceaccount:fake-namespace:fake-service-account",
				Groups:   []string{"fake-group"},
			},
			listRoleBindingsFn: func(
				_ context.Context,
				objs client.ObjectList,
				_ ...client.ListOption,
			) error {
				// nolint: forcetypeassert
				roleBindings := objs.(*rbacv1.RoleBindingList)
				roleBindings.Items = []rbacv1.RoleBinding{
					{
						RoleRef: rbacv1.RoleRef{
							Kind: "Role",
							Name: "fake-role",
						},
						Subjects: []rbacv1.Subject{
							{
								Kind: "Group",
								Name: "fake-group",
							},
						},
					},
				}
				return nil
			},
			assertions: func(roles map[string]struct{}, err error) {
				require.NoError(t, err)
				require.Len(t, roles, 1)
				require.Contains(t, roles, "fake-role")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			w := &webhook{
				listRoleBindingsFn: testCase.listRoleBindingsFn,
			}
			testCase.assertions(
				w.getSubjectRoles(
					context.Background(),
					testCase.subjectInfo,
					testCase.name,
				),
			)
		})
	}
}

func TestSubjectHasGroup(t *testing.T) {
	testCases := []struct {
		name        string
		subjectInfo authenticationv1.UserInfo
		group       string
		hasGroup    bool
	}{
		{
			name: "subject has group",
			subjectInfo: authenticationv1.UserInfo{
				Groups: []string{"fake-group"},
			},
			group:    "fake-group",
			hasGroup: true,
		},
		{
			name: "subject does not have group",
			subjectInfo: authenticationv1.UserInfo{
				Groups: []string{"different-fake-group"},
			},
			group:    "fake-group",
			hasGroup: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.hasGroup,
				subjectHasGroup(testCase.subjectInfo, testCase.group),
			)
		})
	}
}

func TestGetServiceAccountNamespaceAndName(t *testing.T) {
	testCases := []struct {
		name              string
		username          string
		expectedNamespace string
		expectedName      string
	}{
		{
			name:     "subject name does not have service account prefix",
			username: "fake-user",
		},
		{
			name:     "subject name does not have four parts",
			username: "system:serviceaccount:fake-service-account",
		},
		{
			name:              "subject name represents a service account",
			username:          "system:serviceaccount:fake-namespace:fake-service-account", // nolint: lll
			expectedNamespace: "fake-namespace",
			expectedName:      "fake-service-account",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			namespace, name := getServiceAccountNamespaceAndName(testCase.username)
			require.Equal(t, testCase.expectedNamespace, namespace)
			require.Equal(t, testCase.expectedName, name)
		})
	}
}
