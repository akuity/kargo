package yaml

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestParseKubernetesManifest(t *testing.T) {
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
	}

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, kargoapi.AddToScheme(scheme))
	parseKubernetesManifest := NewKubernetesManifestParser(scheme)

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

func TestSetStringsInBytes(t *testing.T) {
	testCases := []struct {
		name       string
		inBytes    []byte
		changes    map[string]string
		assertions func([]byte, error)
	}{
		{
			name: "invalid YAML",
			// Note: This YAML is invalid because one line is indented with a tab
			inBytes: []byte(`
characters:
- name: Anakin
	affiliation: Light side
`),
			assertions: func(bytes []byte, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error unmarshaling input")
				require.Nil(t, bytes)
			},
		},
		{
			name: "success",
			inBytes: []byte(`
characters:
- name: Anakin
  affiliation: Light side
`),
			changes: map[string]string{
				"characters.0.affiliation": "Dark side",
			},
			assertions: func(bytes []byte, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					[]byte(`
characters:
- name: Anakin
  affiliation: Dark side
`),
					bytes,
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				SetStringsInBytes(testCase.inBytes, testCase.changes),
			)
		})
	}
}

func TestFindScalarNode(t *testing.T) {
	yamlBytes := []byte(`
characters:
  rebels:
  - name: Skywalker
`)
	testCases := []struct {
		name       string
		keyPath    string
		assertions func(found bool, line, col int)
	}{
		{
			name:    "node not found",
			keyPath: "characters.imperials",
			assertions: func(found bool, line, col int) {
				require.False(t, found)
				require.Zero(t, line)
				require.Zero(t, col)
			},
		},
		{
			name: "node not found due to error parsing int",
			// Really, this is a special case of a key that doesn't address a node,
			// because there is alpha input where numeric input would be expected.
			keyPath: "characters.rebels.first.name",
			assertions: func(found bool, line, col int) {
				require.False(t, found)
				require.Zero(t, line)
				require.Zero(t, col)
			},
		},
		{
			name:    "node found, but isn't a scalar node",
			keyPath: "characters.rebels",
			assertions: func(found bool, line, col int) {
				require.False(t, found)
				require.Zero(t, line)
				require.Zero(t, col)
			},
		},
		{
			name:    "success",
			keyPath: "characters.rebels.0.name",
			assertions: func(found bool, line, col int) {
				require.True(t, found)
				require.Equal(t, 3, line)
				require.Equal(t, 10, col)
			},
		},
	}
	doc := &yaml.Node{}
	err := yaml.Unmarshal(yamlBytes, doc)
	require.NoError(t, err)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				findScalarNode(doc, strings.Split(testCase.keyPath, ".")),
			)
		})
	}
}
