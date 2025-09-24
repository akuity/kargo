package rollouts

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	rolloutsapi "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
)

func TestGetAnalysisTemplate(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, rolloutsapi.SchemeBuilder.AddToScheme(scheme))

	testCases := []struct {
		name       string
		client     client.Client
		assertions func(*testing.T, *rolloutsapi.AnalysisTemplate, error)
	}{
		{
			name:   "not found",
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			assertions: func(t *testing.T, template *rolloutsapi.AnalysisTemplate, err error) {
				require.NoError(t, err)
				require.Nil(t, template)
			},
		},

		{
			name: "found",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&rolloutsapi.AnalysisTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-template",
						Namespace: "fake-namespace",
					},
				},
			).Build(),
			assertions: func(t *testing.T, template *rolloutsapi.AnalysisTemplate, err error) {
				require.NoError(t, err)
				require.Equal(t, "fake-template", template.Name)
				require.Equal(t, "fake-namespace", template.Namespace)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			template, err := GetAnalysisTemplate(
				context.Background(),
				testCase.client,
				types.NamespacedName{
					Namespace: "fake-namespace",
					Name:      "fake-template",
				},
			)
			testCase.assertions(t, template, err)
		})
	}
}

func TestGetClusterAnalysisTemplate(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, rolloutsapi.SchemeBuilder.AddToScheme(scheme))

	testCases := []struct {
		name       string
		client     client.Client
		assertions func(*testing.T, *rolloutsapi.ClusterAnalysisTemplate, error)
	}{
		{
			name:   "not found",
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			assertions: func(t *testing.T, template *rolloutsapi.ClusterAnalysisTemplate, err error) {
				require.NoError(t, err)
				require.Nil(t, template)
			},
		},

		{
			name: "found",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&rolloutsapi.ClusterAnalysisTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name: "fake-template",
					},
				},
			).Build(),
			assertions: func(t *testing.T, template *rolloutsapi.ClusterAnalysisTemplate, err error) {
				require.NoError(t, err)
				require.Equal(t, "fake-template", template.Name)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			template, err := GetClusterAnalysisTemplate(
				context.Background(),
				testCase.client,
				"fake-template",
			)
			testCase.assertions(t, template, err)
		})
	}
}

func TestGetAnalysisRun(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, rolloutsapi.SchemeBuilder.AddToScheme(scheme))

	testCases := []struct {
		name       string
		client     client.Client
		assertions func(*testing.T, *rolloutsapi.AnalysisRun, error)
	}{
		{
			name:   "not found",
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			assertions: func(t *testing.T, run *rolloutsapi.AnalysisRun, err error) {
				require.NoError(t, err)
				require.Nil(t, run)
			},
		},

		{
			name: "found",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-run",
						Namespace: "fake-namespace",
					},
				},
			).Build(),
			assertions: func(t *testing.T, run *rolloutsapi.AnalysisRun, err error) {
				require.NoError(t, err)
				require.Equal(t, "fake-run", run.Name)
				require.Equal(t, "fake-namespace", run.Namespace)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			run, err := GetAnalysisRun(
				context.Background(),
				testCase.client,
				types.NamespacedName{
					Namespace: "fake-namespace",
					Name:      "fake-run",
				},
			)
			testCase.assertions(t, run, err)
		})
	}
}
