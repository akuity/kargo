package manifest

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestParser(t *testing.T) {
	testCases := map[string]struct {
		data            []byte
		expectedObjects []runtime.Object
		expectedErr     bool
	}{
		"single yaml document": {
			data: []byte(`apiVersion: v1
kind: Namespace
metadata:
  name: kargo-demo
  labels:
    kargo.akuity.io/project: "true"
`),
			expectedObjects: []runtime.Object{
				&corev1.Namespace{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Namespace",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "kargo-demo",
						Labels: map[string]string{
							"kargo.akuity.io/project": "true",
						},
					},
				},
			},
		},
		"multiple yaml documents": {
			data: []byte(`---
apiVersion: v1
kind: Namespace
metadata:
  name: kargo-demo
  labels:
    kargo.akuity.io/project: "true"
---
apiVersion: kargo.akuity.io/v1alpha1
kind: PromotionPolicy
metadata:
  name: test
  namespace: kargo-demo
stage: test
enableAutoPromotion: true
`),
			expectedObjects: []runtime.Object{
				&corev1.Namespace{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Namespace",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "kargo-demo",
						Labels: map[string]string{
							"kargo.akuity.io/project": "true",
						},
					},
				},
				&kargoapi.PromotionPolicy{
					TypeMeta: metav1.TypeMeta{
						APIVersion: kargoapi.GroupVersion.String(),
						Kind:       "PromotionPolicy",
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "kargo-demo",
						Name:      "test",
					},
					Stage:               "test",
					EnableAutoPromotion: true,
				},
			},
		},
		"cluster resource should be present first": {
			data: []byte(`---
apiVersion: kargo.akuity.io/v1alpha1
kind: PromotionPolicy
metadata:
  name: test
  namespace: kargo-demo
stage: test
enableAutoPromotion: true
---
apiVersion: v1
kind: Namespace
metadata:
  name: kargo-demo
  labels:
    kargo.akuity.io/project: "true"
`),
			expectedObjects: []runtime.Object{
				&corev1.Namespace{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Namespace",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "kargo-demo",
						Labels: map[string]string{
							"kargo.akuity.io/project": "true",
						},
					},
				},
				&kargoapi.PromotionPolicy{
					TypeMeta: metav1.TypeMeta{
						APIVersion: kargoapi.GroupVersion.String(),
						Kind:       "PromotionPolicy",
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "kargo-demo",
						Name:      "test",
					},
					Stage:               "test",
					EnableAutoPromotion: true,
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, kargoapi.AddToScheme(scheme))
	parseKubernetesManifest := NewParser(scheme)

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			actual, err := parseKubernetesManifest(tc.data)
			if tc.expectedErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			expected := make([]*unstructured.Unstructured, len(tc.expectedObjects))
			for idx, obj := range tc.expectedObjects {
				u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
				require.NoError(t, err)
				expected[idx] = &unstructured.Unstructured{Object: u}
			}
			require.EqualValues(t, expected, actual)
		})
	}
}
