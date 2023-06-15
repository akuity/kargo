package credentials

import (
	"context"
	"testing"

	"github.com/argoproj/argo-cd/v2/applicationset/utils"
	"github.com/argoproj/argo-cd/v2/common"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestIndex(t *testing.T) {
	testCases := []struct {
		name           string
		secret         *corev1.Secret
		expectedResult []string
	}{
		{
			name: "no labels",
			secret: &corev1.Secret{
				Data: make(map[string][]byte),
			},
		},

		{
			name: "no data",
			secret: &corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{},
				},
			},
		},

		{
			name: "secret in Argo CD namespace not labeled",
			secret: &corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Namespace: "argo-cd",
					Labels:    map[string]string{},
				},
				Data: map[string][]byte{},
			},
		},

		{
			name: "secret in Argo CD namespace not labeled as a repo",
			secret: &corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Namespace: "argo-cd",
					Labels: map[string]string{
						utils.ArgoCDSecretTypeLabel: "bogus",
					},
				},
				Data: map[string][]byte{},
			},
		},

		{
			name: "secret in other namespace not labeled",
			secret: &corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Namespace: "fake-namespace",
					Labels:    map[string]string{},
				},
				Data: map[string][]byte{},
			},
		},

		{
			name: "secret in other namespace not labeled as a repo",
			secret: &corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Namespace: "fake-namespace",
					Labels: map[string]string{
						kargoSecretTypeLabel: "bogus",
					},
				},
				Data: map[string][]byte{},
			},
		},

		{
			name: "credentials type is invalid",
			secret: &corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Namespace: "fake-namespace",
					Labels: map[string]string{
						kargoSecretTypeLabel: common.LabelValueSecretTypeRepository,
					},
				},
				Data: map[string][]byte{
					"type": []byte("bogus"),
				},
			},
		},

		{
			name: "URL is missing",
			secret: &corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Namespace: "fake-namespace",
					Labels: map[string]string{
						kargoSecretTypeLabel: common.LabelValueSecretTypeRepository,
					},
				},
				Data: map[string][]byte{},
			},
		},

		{
			name: "URL is missing",
			secret: &corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Namespace: "fake-namespace",
					Labels: map[string]string{
						kargoSecretTypeLabel: common.LabelValueSecretTypeRepository,
					},
				},
				Data: map[string][]byte{},
			},
			expectedResult: nil,
		},

		{
			name: "success",
			secret: &corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Namespace: "fake-namespace",
					Labels: map[string]string{
						kargoSecretTypeLabel: common.LabelValueSecretTypeRepository,
					},
				},
				Data: map[string][]byte{
					"url": []byte("fake-url"),
				},
			},
			expectedResult: []string{"git:fake-url"},
		},
	}
	credsDB := &kubernetesDatabase{
		argoCDNamespace: "argo-cd",
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.expectedResult, credsDB.index(testCase.secret))
		})
	}
}

func TestGetCredentialsSecret(t *testing.T) {
	testCases := []struct {
		name          string
		clientBuilder *fake.ClientBuilder
		assertions    func(*corev1.Secret, error)
	}{
		{
			name:          "no secrets found",
			clientBuilder: fake.NewClientBuilder(),
			assertions: func(secret *corev1.Secret, err error) {
				require.NoError(t, err)
				require.Nil(t, secret)
			},
		},

		{
			name: "success",
			clientBuilder: fake.NewClientBuilder().WithObjects(
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{
						Name:      "fake-creds",
						Namespace: "fake-namespace",
					},
				},
			),
			assertions: func(secret *corev1.Secret, err error) {
				require.NoError(t, err)
				require.NotNil(t, secret)
				require.Equal(t, "fake-creds", secret.Name)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				getCredentialsSecret(
					context.Background(),
					testCase.clientBuilder.Build(),
					"fake-namespace",
					labels.Everything(),
					fields.Everything(),
				),
			)
		})
	}
}

func TestGetCredentialsTemplateSecret(t *testing.T) {
	testCases := []struct {
		name          string
		clientBuilder *fake.ClientBuilder
		assertions    func(*corev1.Secret, error)
	}{
		{
			name:          "no secrets found",
			clientBuilder: fake.NewClientBuilder(),
			assertions: func(secret *corev1.Secret, err error) {
				require.NoError(t, err)
				require.Nil(t, secret)
			},
		},

		{
			name: "success",
			clientBuilder: fake.NewClientBuilder().WithObjects(
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{
						Name:      "wrong-creds-template",
						Namespace: "fake-namespace",
					},
					Data: nil, // No data
				},
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{
						Name:      "another-wrong-creds-template",
						Namespace: "fake-namespace",
					},
					Data: map[string][]byte{
						"url": []byte("not-a-match"),
					},
				},
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{
						Name:      "fake-creds-template",
						Namespace: "fake-namespace",
					},
					Data: map[string][]byte{
						"url": []byte("fake"),
					},
				},
			),
			assertions: func(secret *corev1.Secret, err error) {
				require.NoError(t, err)
				require.NotNil(t, secret)
				require.Equal(t, "fake-creds-template", secret.Name)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				getCredentialsTemplateSecret(
					context.Background(),
					testCase.clientBuilder.Build(),
					"fake-namespace",
					labels.Everything(),
					"fake-url",
				),
			)
		})
	}
}
