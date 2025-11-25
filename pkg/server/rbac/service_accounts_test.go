package rbac

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
)

const testServiceAccountName = "fake-service-account"

func TestNewKubernetesServiceAccountsDatabase(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	cfg := ServiceAccountDatabaseConfig{KargoNamespace: "kargo"}
	d := NewKubernetesServiceAccountsDatabase(c, cfg)
	require.NotNil(t, d)
	db, ok := d.(*serviceAccountsDatabase)
	require.True(t, ok)
	require.Same(t, c, db.client)
	require.Equal(t, cfg, db.cfg)
}

func Test_serviceAccountsDatabase_Create(t *testing.T) {
	t.Run("ServiceAccount already exists", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      testServiceAccountName,
			}},
		).Build()
		_, err := NewKubernetesServiceAccountsDatabase(c, ServiceAccountDatabaseConfig{}).
			Create(t.Context(), &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      testServiceAccountName,
			}})
		require.Error(t, err)
		require.True(t, apierrors.IsAlreadyExists(err))
	})

	t.Run("success", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		sa, err := NewKubernetesServiceAccountsDatabase(c, ServiceAccountDatabaseConfig{}).
			Create(t.Context(), &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      testServiceAccountName,
			}})
		require.NoError(t, err)
		require.NotNil(t, sa)
		require.Equal(t, testProject, sa.Namespace)
		require.Equal(t, testServiceAccountName, sa.Name)
		require.Equal(
			t,
			rbacapi.LabelValueTrue,
			sa.Labels[rbacapi.LabelKeyServiceAccount],
		)
		require.Equal(
			t,
			rbacapi.AnnotationValueTrue,
			sa.Annotations[rbacapi.AnnotationKeyManaged],
		)
		sa = &corev1.ServiceAccount{}
		err = c.Get(
			t.Context(),
			client.ObjectKey{
				Namespace: testProject,
				Name:      testServiceAccountName,
			},
			sa,
		)
		require.NoError(t, err)
		require.NotNil(t, sa)
		require.Equal(t, testProject, sa.Namespace)
		require.Equal(t, testServiceAccountName, sa.Name)
		require.Equal(
			t,
			rbacapi.LabelValueTrue,
			sa.Labels[rbacapi.LabelKeyServiceAccount],
		)
		require.Equal(
			t,
			rbacapi.AnnotationValueTrue,
			sa.Annotations[rbacapi.AnnotationKeyManaged],
		)
	})
}

func Test_serviceAccountsDatabase_Delete(t *testing.T) {
	t.Run("ServiceAccount not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		err := NewKubernetesServiceAccountsDatabase(c, ServiceAccountDatabaseConfig{}).
			Delete(t.Context(), testProject, testServiceAccountName)
		require.Error(t, err)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("ServiceAccount not labeled as Kargo ServiceAccount", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      testServiceAccountName,
			}},
		).Build()
		err := NewKubernetesServiceAccountsDatabase(c, ServiceAccountDatabaseConfig{}).
			Delete(t.Context(), testProject, testServiceAccountName)
		require.Error(t, err)
		require.True(t, apierrors.IsBadRequest(err))
	})

	t.Run("ServiceAccount not annotated as Kargo-managed", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      testServiceAccountName,
				Labels: map[string]string{
					rbacapi.LabelKeyServiceAccount: rbacapi.LabelValueTrue,
				},
			}},
		).Build()
		err := NewKubernetesServiceAccountsDatabase(c, ServiceAccountDatabaseConfig{}).
			Delete(t.Context(), testProject, testServiceAccountName)
		require.Error(t, err)
		require.True(t, apierrors.IsBadRequest(err))
	})

	t.Run("success", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      testServiceAccountName,
				Labels: map[string]string{
					rbacapi.LabelKeyServiceAccount: rbacapi.LabelValueTrue,
				},
				Annotations: map[string]string{
					rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
				},
			}},
		).Build()
		err := NewKubernetesServiceAccountsDatabase(c, ServiceAccountDatabaseConfig{}).
			Delete(t.Context(), testProject, testServiceAccountName)
		require.NoError(t, err)
		sa := &corev1.ServiceAccount{}
		err = c.Get(
			t.Context(),
			client.ObjectKey{
				Namespace: testProject,
				Name:      testServiceAccountName,
			},
			sa,
		)
		require.Error(t, err)
		require.True(t, apierrors.IsNotFound(err))
	})
}

func Test_serviceAccountsDatabase_Get(t *testing.T) {
	t.Run("ServiceAccount not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		_, err := NewKubernetesServiceAccountsDatabase(c, ServiceAccountDatabaseConfig{}).
			Get(t.Context(), false, testProject, testServiceAccountName)
		require.Error(t, err)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("ServiceAccount not labeled as Kargo ServiceAccount", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      testServiceAccountName,
			}},
		).Build()
		_, err := NewKubernetesServiceAccountsDatabase(c, ServiceAccountDatabaseConfig{}).
			Get(t.Context(), false, testProject, testServiceAccountName)
		require.Error(t, err)
		require.True(t, apierrors.IsBadRequest(err))
	})

	t.Run("success", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      testServiceAccountName,
				Labels: map[string]string{
					rbacapi.LabelKeyServiceAccount: rbacapi.LabelValueTrue,
				},
			}},
		).Build()
		sa, err := NewKubernetesServiceAccountsDatabase(c, ServiceAccountDatabaseConfig{}).
			Get(t.Context(), false, testProject, testServiceAccountName)
		require.NoError(t, err)
		require.NotNil(t, sa)
		require.Equal(t, testServiceAccountName, sa.Name)
		require.Equal(t, testProject, sa.Namespace)
	})
}

func Test_serviceAccountsDatabase_CreateToken(t *testing.T) {
	const testTokenName = "fake-token"

	t.Run("ServiceAccount not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		_, err := NewKubernetesServiceAccountsDatabase(c, ServiceAccountDatabaseConfig{}).
			CreateToken(t.Context(), false, testProject, testServiceAccountName, testTokenName)
		require.Error(t, err)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("ServiceAccount not labeled as Kargo ServiceAccount", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).
			WithObjects(&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      testServiceAccountName,
			}}).Build()
		_, err := NewKubernetesServiceAccountsDatabase(c, ServiceAccountDatabaseConfig{}).
			CreateToken(t.Context(), false, testProject, testServiceAccountName, testTokenName)
		require.Error(t, err)
		require.True(t, apierrors.IsBadRequest(err))
	})

	t.Run("Secret with token name already exists", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      testServiceAccountName,
				Labels: map[string]string{
					rbacapi.LabelKeyServiceAccount: rbacapi.LabelValueTrue,
				},
				Annotations: map[string]string{
					rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
				},
			}},
			&corev1.Secret{ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      testTokenName,
			}},
		).Build()
		_, err := NewKubernetesServiceAccountsDatabase(c, ServiceAccountDatabaseConfig{}).
			CreateToken(t.Context(), false, testProject, testServiceAccountName, testTokenName)
		require.Error(t, err)
		require.True(t, apierrors.IsAlreadyExists(err))
	})

	t.Run("success", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).
			WithObjects(&corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      testServiceAccountName,
					Labels: map[string]string{
						rbacapi.LabelKeyServiceAccount: rbacapi.LabelValueTrue,
					},
					Annotations: map[string]string{
						rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
					},
				},
			}).
			WithInterceptorFuncs(interceptor.Funcs{
				// The method under test has a simple retry loop that waits for the
				// new Secret's data to be populated. We need populate the Secret's data
				// ourselves because the fake client doesn't do it.
				Get: func(
					ctx context.Context,
					client client.WithWatch,
					key client.ObjectKey,
					obj client.Object,
					opts ...client.GetOption,
				) error {
					if s, ok := obj.(*corev1.Secret); ok {
						newS := &corev1.Secret{}
						if err := client.Get(ctx, key, newS); err != nil {
							return err
						}
						newS.Data = map[string][]byte{
							"token": []byte("fake-token-value"),
						}
						*s = *newS
						return nil
					}
					return client.Get(ctx, key, obj, opts...)
				},
			}).
			Build()
		tokenSecret, err := NewKubernetesServiceAccountsDatabase(c, ServiceAccountDatabaseConfig{}).
			CreateToken(t.Context(), false, testProject, testServiceAccountName, testTokenName)
		require.NoError(t, err)
		require.NotNil(t, tokenSecret)
		tokenSecret = &corev1.Secret{}
		err = c.Get(
			t.Context(),
			client.ObjectKey{
				Namespace: testProject,
				Name:      testTokenName,
			},
			tokenSecret,
		)
		require.NoError(t, err)
		require.Equal(t, corev1.SecretTypeServiceAccountToken, tokenSecret.Type)
		require.Equal(
			t,
			testServiceAccountName,
			tokenSecret.Annotations["kubernetes.io/service-account.name"],
		)
	})
}

func Test_serviceAccountsDatabase_List(t *testing.T) {
	t.Run("no ServiceAccounts", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		saList, err := NewKubernetesServiceAccountsDatabase(c, ServiceAccountDatabaseConfig{}).
			List(t.Context(), false, testProject)
		require.NoError(t, err)
		require.Empty(t, saList)
	})

	t.Run("with non-Kargo ServiceAccounts", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      testServiceAccountName,
			}},
		).Build()
		saList, err := NewKubernetesServiceAccountsDatabase(c, ServiceAccountDatabaseConfig{}).
			List(t.Context(), false, testProject)
		require.NoError(t, err)
		require.Empty(t, saList)
	})

	t.Run("with Kargo ServiceAccounts", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      "test-sa-1",
					Labels: map[string]string{
						rbacapi.LabelKeyServiceAccount: rbacapi.LabelValueTrue,
					},
				},
			},
			&corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      "test-sa-2",
					Labels: map[string]string{
						rbacapi.LabelKeyServiceAccount: rbacapi.LabelValueTrue,
					},
				},
			},
			&corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      "test-sa-3",
				}, // Not labeled as Kargo ServiceAccount
			},
		).Build()
		saList, err := NewKubernetesServiceAccountsDatabase(c, ServiceAccountDatabaseConfig{}).
			List(t.Context(), false, testProject)
		require.NoError(t, err)
		require.Len(t, saList, 2)

		names := []string{saList[0].Name, saList[1].Name}
		require.Contains(t, names, "test-sa-1")
		require.Contains(t, names, "test-sa-2")
	})
}

func Test_serviceAccountsDatabase_GetToken(t *testing.T) {
	t.Run("token Secret not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		_, err := NewKubernetesServiceAccountsDatabase(c, ServiceAccountDatabaseConfig{}).
			GetToken(t.Context(), false, testProject, "non-existent-token")
		require.Error(t, err)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Secret is not a ServiceAccount token", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      "regular-secret",
				},
				Type: corev1.SecretTypeOpaque,
			},
		).Build()
		_, err := NewKubernetesServiceAccountsDatabase(c, ServiceAccountDatabaseConfig{}).
			GetToken(t.Context(), false, testProject, "regular-secret")
		require.Error(t, err)
		require.True(t, apierrors.IsBadRequest(err))
		require.Contains(t, err.Error(), "not labeled as a Kargo ServiceAccount token")
	})

	t.Run("success", func(t *testing.T) {
		tokenName := "test-token"
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      tokenName,
					Labels: map[string]string{
						rbacapi.LabelKeyServiceAccountToken: rbacapi.LabelValueTrue,
					},
					Annotations: map[string]string{
						"kubernetes.io/service-account.name": testServiceAccountName,
						rbacapi.AnnotationKeyManaged:         rbacapi.AnnotationValueTrue,
					},
				},
				Type: corev1.SecretTypeServiceAccountToken,
				Data: map[string][]byte{
					"token": []byte("test-token-data"),
				},
			},
		).Build()
		secret, err := NewKubernetesServiceAccountsDatabase(c, ServiceAccountDatabaseConfig{}).
			GetToken(t.Context(), false, testProject, tokenName)
		require.NoError(t, err)
		require.NotNil(t, secret)
		require.Equal(t, testProject, secret.Namespace)
		require.Equal(t, tokenName, secret.Name)
		require.Equal(t, corev1.SecretTypeServiceAccountToken, secret.Type)
		require.Equal(
			t,
			testServiceAccountName,
			secret.Annotations["kubernetes.io/service-account.name"],
		)
		// Token data should be redacted
		require.Equal(t, []byte("*** REDACTED ***"), secret.Data["token"])
	})
}

func Test_serviceAccountsDatabase_DeleteToken(t *testing.T) {
	t.Run("token Secret not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      testServiceAccountName,
				Labels: map[string]string{
					rbacapi.LabelKeyServiceAccount: rbacapi.LabelValueTrue,
				},
				Annotations: map[string]string{
					rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
				},
			}},
		).Build()
		err := NewKubernetesServiceAccountsDatabase(c, ServiceAccountDatabaseConfig{}).
			DeleteToken(t.Context(), false, testProject, "non-existent-token")
		require.Error(t, err)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("token Secret not labeled as Kargo ServiceAccount token", func(t *testing.T) {
		tokenName := "test-token"
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      testServiceAccountName,
				Labels: map[string]string{
					rbacapi.LabelKeyServiceAccount: rbacapi.LabelValueTrue,
				},
				Annotations: map[string]string{
					rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
				},
			}},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      tokenName,
					Annotations: map[string]string{
						"kubernetes.io/service-account.name": testServiceAccountName,
						rbacapi.AnnotationKeyManaged:         rbacapi.AnnotationValueTrue,
					},
				},
				Type: corev1.SecretTypeServiceAccountToken,
			},
		).Build()
		err := NewKubernetesServiceAccountsDatabase(c, ServiceAccountDatabaseConfig{}).
			DeleteToken(t.Context(), false, testProject, tokenName)
		require.Error(t, err)
		require.True(t, apierrors.IsBadRequest(err))
		require.Contains(t, err.Error(), "not labeled as a Kargo ServiceAccount token")
	})

	t.Run("token Secret not annotated as Kargo-managed", func(t *testing.T) {
		tokenName := "test-token"
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      testServiceAccountName,
				Labels: map[string]string{
					rbacapi.LabelKeyServiceAccount: rbacapi.LabelValueTrue,
				},
				Annotations: map[string]string{
					rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
				},
			}},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      tokenName,
					Labels: map[string]string{
						rbacapi.LabelKeyServiceAccountToken: rbacapi.LabelValueTrue,
					},
					Annotations: map[string]string{
						"kubernetes.io/service-account.name": testServiceAccountName,
					},
				},
				Type: corev1.SecretTypeServiceAccountToken,
			},
		).Build()
		err := NewKubernetesServiceAccountsDatabase(c, ServiceAccountDatabaseConfig{}).
			DeleteToken(t.Context(), false, testProject, tokenName)
		require.Error(t, err)
		require.True(t, apierrors.IsBadRequest(err))
		require.Contains(t, err.Error(), "not annotated as Kargo-managed")
	})

	t.Run("success", func(t *testing.T) {
		tokenName := "test-token"
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      testServiceAccountName,
				Labels: map[string]string{
					rbacapi.LabelKeyServiceAccount: rbacapi.LabelValueTrue,
				},
				Annotations: map[string]string{
					rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
				},
			}},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      tokenName,
					Labels: map[string]string{
						rbacapi.LabelKeyServiceAccountToken: rbacapi.LabelValueTrue,
					},
					Annotations: map[string]string{
						"kubernetes.io/service-account.name": testServiceAccountName,
						rbacapi.AnnotationKeyManaged:         rbacapi.AnnotationValueTrue,
					},
				},
				Type: corev1.SecretTypeServiceAccountToken,
			},
		).Build()
		err := NewKubernetesServiceAccountsDatabase(c, ServiceAccountDatabaseConfig{}).
			DeleteToken(t.Context(), false, testProject, tokenName)
		require.NoError(t, err)
		// Verify the token Secret was deleted
		secret := &corev1.Secret{}
		err = c.Get(
			t.Context(),
			client.ObjectKey{
				Namespace: testProject,
				Name:      tokenName,
			},
			secret,
		)
		require.Error(t, err)
		require.True(t, apierrors.IsNotFound(err))
	})
}

func Test_serviceAccountsDatabase_ListTokens(t *testing.T) {
	t.Run("ServiceAccount not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		_, err := NewKubernetesServiceAccountsDatabase(c, ServiceAccountDatabaseConfig{}).
			ListTokens(t.Context(), false, testProject, testServiceAccountName)
		require.Error(t, err)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("ServiceAccount not labeled as Kargo ServiceAccount", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      testServiceAccountName,
			}},
		).Build()
		_, err := NewKubernetesServiceAccountsDatabase(c, ServiceAccountDatabaseConfig{}).
			ListTokens(t.Context(), false, testProject, testServiceAccountName)
		require.Error(t, err)
		require.True(t, apierrors.IsBadRequest(err))
	})

	t.Run("no tokens", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      testServiceAccountName,
				Labels: map[string]string{
					rbacapi.LabelKeyServiceAccount: rbacapi.LabelValueTrue,
				},
				Annotations: map[string]string{
					rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
				},
			}},
		).Build()
		tokens, err := NewKubernetesServiceAccountsDatabase(c, ServiceAccountDatabaseConfig{}).
			ListTokens(t.Context(), false, testProject, testServiceAccountName)
		require.NoError(t, err)
		require.Empty(t, tokens)
	})

	t.Run("with tokens for different ServiceAccounts", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      testServiceAccountName,
					Labels: map[string]string{
						rbacapi.LabelKeyServiceAccount: rbacapi.LabelValueTrue,
					},
					Annotations: map[string]string{
						rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
					},
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      "token-1",
					Labels: map[string]string{
						rbacapi.LabelKeyServiceAccountToken: rbacapi.LabelValueTrue,
					},
					Annotations: map[string]string{
						"kubernetes.io/service-account.name": testServiceAccountName,
					},
				},
				Type: corev1.SecretTypeServiceAccountToken,
				Data: map[string][]byte{"token": []byte("token-1-data")},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      "token-2",
					Labels: map[string]string{
						rbacapi.LabelKeyServiceAccountToken: rbacapi.LabelValueTrue,
					},
					Annotations: map[string]string{
						"kubernetes.io/service-account.name": testServiceAccountName,
					},
				},
				Type: corev1.SecretTypeServiceAccountToken,
				Data: map[string][]byte{"token": []byte("token-2-data")},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      "other-token",
					Labels: map[string]string{
						rbacapi.LabelKeyServiceAccountToken: rbacapi.LabelValueTrue,
					},
					Annotations: map[string]string{
						"kubernetes.io/service-account.name": "other-sa",
					},
				},
				Type: corev1.SecretTypeServiceAccountToken,
				Data: map[string][]byte{"token": []byte("other-token-data")},
			},
		).Build()
		tokens, err := NewKubernetesServiceAccountsDatabase(c, ServiceAccountDatabaseConfig{}).
			ListTokens(t.Context(), false, testProject, testServiceAccountName)
		require.NoError(t, err)
		require.Len(t, tokens, 2)

		names := []string{tokens[0].Name, tokens[1].Name}
		require.Contains(t, names, "token-1")
		require.Contains(t, names, "token-2")
		require.NotContains(t, names, "other-token")
	})
}

func Test_isKargoServiceAccount(t *testing.T) {
	t.Run("not a Kargo ServiceAccount", func(t *testing.T) {
		require.False(t, isKargoServiceAccount(
			&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      testServiceAccountName,
			}},
		))
	})

	t.Run("is a Kargo ServiceAccount", func(t *testing.T) {
		require.True(t, isKargoServiceAccount(
			&corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-sa",
					Labels: map[string]string{
						rbacapi.LabelKeyServiceAccount: rbacapi.LabelValueTrue,
					},
				},
			},
		))
	})

	t.Run("has label but wrong value", func(t *testing.T) {
		require.False(t, isKargoServiceAccount(
			&corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-sa",
					Labels: map[string]string{
						rbacapi.LabelKeyServiceAccount: "false",
					},
				},
			},
		))
	})
}

func Test_isKargoServiceAccountToken(t *testing.T) {
	t.Run("not a ServiceAccount token type", func(t *testing.T) {
		require.False(t, isKargoServiceAccountToken(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      "test-secret",
					Labels: map[string]string{
						rbacapi.LabelKeyServiceAccountToken: rbacapi.LabelValueTrue,
					},
				},
				Type: corev1.SecretTypeOpaque,
			},
		))
	})

	t.Run("ServiceAccount token type but not labeled", func(t *testing.T) {
		require.False(t, isKargoServiceAccountToken(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      "test-secret",
				},
				Type: corev1.SecretTypeServiceAccountToken,
			},
		))
	})

	t.Run("has label but wrong value", func(t *testing.T) {
		require.False(t, isKargoServiceAccountToken(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      "test-secret",
					Labels: map[string]string{
						rbacapi.LabelKeyServiceAccountToken: "false",
					},
				},
				Type: corev1.SecretTypeServiceAccountToken,
			},
		))
	})

	t.Run("is a Kargo ServiceAccount token", func(t *testing.T) {
		require.True(t, isKargoServiceAccountToken(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      "test-token",
					Labels: map[string]string{
						rbacapi.LabelKeyServiceAccountToken: rbacapi.LabelValueTrue,
					},
				},
				Type: corev1.SecretTypeServiceAccountToken,
			},
		))
	})
}

func Test_redactTokenData(t *testing.T) {
	t.Run("no token data", func(t *testing.T) {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      "test-token",
			},
			Type: corev1.SecretTypeServiceAccountToken,
			Data: map[string][]byte{
				"ca.crt": []byte("cert-data"),
			},
		}
		redactTokenData(secret)
		require.Equal(t, []byte("cert-data"), secret.Data["ca.crt"])
		require.NotContains(t, secret.Data, "token")
	})

	t.Run("with token data", func(t *testing.T) {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      "test-token",
			},
			Type: corev1.SecretTypeServiceAccountToken,
			Data: map[string][]byte{
				"token":  []byte("sensitive-token-value"),
				"ca.crt": []byte("cert-data"),
			},
		}
		redactTokenData(secret)
		require.Equal(t, []byte("*** REDACTED ***"), secret.Data["token"])
		require.Equal(t, []byte("cert-data"), secret.Data["ca.crt"])
	})

	t.Run("with empty token data", func(t *testing.T) {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      "test-token",
			},
			Type: corev1.SecretTypeServiceAccountToken,
			Data: map[string][]byte{
				"token": []byte(""),
			},
		}
		redactTokenData(secret)
		require.Equal(t, []byte("*** REDACTED ***"), secret.Data["token"])
	})

	t.Run("with nil data map", func(t *testing.T) {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      "test-token",
			},
			Type: corev1.SecretTypeServiceAccountToken,
		}
		// Should not panic
		redactTokenData(secret)
		require.Nil(t, secret.Data)
	})
}

func Test_serviceAccountsDatabase_waitForTokenData(t *testing.T) {
	const testTokenName = "test-token"

	testCases := []struct {
		name       string
		client     client.Client
		assertions func(*testing.T, *corev1.Secret, error)
	}{
		{
			name: "non-retriable error",
			client: fake.NewClientBuilder().WithScheme(scheme).
				WithInterceptorFuncs(interceptor.Funcs{
					Get: func(
						context.Context,
						client.WithWatch,
						client.ObjectKey,
						client.Object,
						...client.GetOption,
					) error {
						// Return a NotFound error - should not retry
						return apierrors.NewNotFound(corev1.Resource("secrets"), "")
					},
				}).
				Build(),
			assertions: func(t *testing.T, secret *corev1.Secret, err error) {
				require.Error(t, err)
				require.Nil(t, secret)
				require.Contains(t, err.Error(), "error while waiting for token Secret")
				require.True(t, apierrors.IsNotFound(err))
			},
		},
		{
			name: "token data not yet populated; all attempts fail",
			client: fake.NewClientBuilder().WithScheme(scheme).
				WithObjects(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      testTokenName,
					},
					Type: corev1.SecretTypeServiceAccountToken,
					// No data
				}).Build(),
			assertions: func(t *testing.T, secret *corev1.Secret, err error) {
				require.Error(t, err)
				require.Nil(t, secret)
				require.Contains(t, err.Error(), "error while waiting for token Secret")
			},
		},
		{
			name: "other retriable error; all attempts fail",
			client: fake.NewClientBuilder().WithScheme(scheme).
				WithInterceptorFuncs(interceptor.Funcs{
					Get: func(
						context.Context,
						client.WithWatch,
						client.ObjectKey,
						client.Object,
						...client.GetOption,
					) error {
						// Always return a retriable error
						return apierrors.NewServiceUnavailable("service unavailable")
					},
				}).
				Build(),
			assertions: func(t *testing.T, secret *corev1.Secret, err error) {
				require.Error(t, err)
				require.Nil(t, secret)
				require.Contains(t, err.Error(), "error while waiting for token Secret")
				require.True(t, apierrors.IsServiceUnavailable(err))
			},
		},
		{
			name: "token data not yet populated; second attempt succeeds",
			client: func() client.Client {
				var attemptCount int
				return fake.NewClientBuilder().WithScheme(scheme).
					WithInterceptorFuncs(interceptor.Funcs{
						Get: func(
							_ context.Context,
							_ client.WithWatch,
							_ client.ObjectKey,
							obj client.Object,
							_ ...client.GetOption,
						) error {
							attemptCount++
							if attemptCount == 1 {
								return nil
							}
							// All subsequent attempts: populate token data
							s, ok := obj.(*corev1.Secret)
							require.True(t, ok)
							s.Data = map[string][]byte{"token": []byte("fake-token-value")}
							return nil
						},
					}).
					Build()
			}(),
			assertions: func(t *testing.T, secret *corev1.Secret, err error) {
				require.NoError(t, err)
				require.NotNil(t, secret)
				require.Equal(t, []byte("fake-token-value"), secret.Data["token"])
			},
		},
		{
			name: "other retriable error; second attempt succeeds",
			client: func() client.Client {
				var attemptCount int
				return fake.NewClientBuilder().WithScheme(scheme).
					WithInterceptorFuncs(interceptor.Funcs{
						Get: func(
							_ context.Context,
							_ client.WithWatch,
							_ client.ObjectKey,
							obj client.Object,
							_ ...client.GetOption,
						) error {
							attemptCount++
							if attemptCount == 1 {
								// First attempt: return retriable error
								return apierrors.NewServerTimeout(
									corev1.Resource("secrets"),
									"get",
									5,
								)
							}
							// All subsequent attempts: populate token data
							s, ok := obj.(*corev1.Secret)
							require.True(t, ok)
							s.Data = map[string][]byte{
								"token": []byte("fake-token-value"),
							}
							return nil
						},
					}).
					Build()
			}(),
			assertions: func(t *testing.T, secret *corev1.Secret, err error) {
				require.NoError(t, err)
				require.NotNil(t, secret)
				require.Equal(t, []byte("fake-token-value"), secret.Data["token"])
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			s := &serviceAccountsDatabase{
				client: testCase.client,
			}
			secret, err := s.waitForTokenData(
				context.Background(),
				testProject,
				testTokenName,
				2, // Only two attempts so that backoffs are minimal during tests
			)
			testCase.assertions(t, secret, err)
		})
	}
}
