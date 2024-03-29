package stages

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	rollouts "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
)

func TestStartVerification(t *testing.T) {
	testCases := []struct {
		name       string
		stage      *kargoapi.Stage
		reconciler *reconciler
		assertions func(*testing.T, *kargoapi.VerificationInfo)
	}{
		{
			name:       "rollouts integration not enabled",
			reconciler: &reconciler{},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo) {
				require.Contains(
					t,
					vi.Message,
					"Rollouts integration is disabled on this controller",
				)
			},
		},
		{
			name: "error listing AnalysisRuns",
			stage: &kargoapi.Stage{
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.FreightReference{
						Name: "fake-id",
					},
				},
			},
			reconciler: &reconciler{
				cfg: ReconcilerConfig{
					RolloutsIntegrationEnabled: true,
				},
				kargoClient: fake.NewClientBuilder().Build(),
				listAnalysisRunsFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo) {
				require.Contains(t, vi.Message, "something went wrong")
				require.Contains(t, vi.Message, "error listing AnalysisRuns for Stage")
			},
		},
		{
			name: "AnalysisRun already exists",
			stage: &kargoapi.Stage{
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.FreightReference{
						Name: "fake-id",
					},
				},
			},
			reconciler: &reconciler{
				cfg: ReconcilerConfig{
					RolloutsIntegrationEnabled: true,
				},
				kargoClient: fake.NewClientBuilder().Build(),
				listAnalysisRunsFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					analysisRuns, ok := objList.(*rollouts.AnalysisRunList)
					require.True(t, ok)
					analysisRuns.Items = []rollouts.AnalysisRun{{}}
					return nil
				},
			},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo) {
				require.NotNil(t, vi)
				require.Empty(t, vi.Message)
			},
		},
		{
			name: "AnalysisRun already exists but reverification is requested",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyReverify: "fake-id",
					},
				},
				Spec: &kargoapi.StageSpec{
					Verification: &kargoapi.Verification{
						AnalysisTemplates: []kargoapi.AnalysisTemplateReference{{}},
					},
				},
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.FreightReference{
						Name: "fake-id",
					},
				},
			},
			reconciler: &reconciler{
				cfg: ReconcilerConfig{
					RolloutsIntegrationEnabled: true,
				},
				kargoClient: fake.NewClientBuilder().Build(),
				listAnalysisRunsFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					analysisRuns, ok := objList.(*rollouts.AnalysisRunList)
					require.True(t, ok)
					analysisRuns.Items = []rollouts.AnalysisRun{{
						ObjectMeta: metav1.ObjectMeta{
							Name: "fake-run",
						},
					}}
					return nil
				},
				getAnalysisTemplateFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*rollouts.AnalysisTemplate, error) {
					return &rollouts.AnalysisTemplate{}, nil
				},
				getFreightFn: func(context.Context, client.Client, types.NamespacedName) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				buildAnalysisRunFn: func(
					*kargoapi.Stage,
					*kargoapi.Freight,
					[]*rollouts.AnalysisTemplate,
				) (*rollouts.AnalysisRun, error) {
					return &rollouts.AnalysisRun{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "new-fake-run",
							Namespace: "fake-namespace",
						},
					}, nil
				},
				createAnalysisRunFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return nil
				},
			},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo) {
				require.NotNilf(t, vi, "expected non-nil VerificationInfo")
				require.NotEmptyf(t, vi.ID, "expected non-empty VerificationInfo.ID")
				require.Equal(t, kargoapi.VerificationPhasePending, vi.Phase)
				require.Equal(t, &kargoapi.AnalysisRunReference{
					Name:      "new-fake-run",
					Namespace: "fake-namespace",
				}, vi.AnalysisRun)
			},
		},
		{
			name: "error getting AnalysisTemplate",
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					Verification: &kargoapi.Verification{
						AnalysisTemplates: []kargoapi.AnalysisTemplateReference{{}},
					},
				},
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.FreightReference{
						Name: "fake-id",
					},
				},
			},
			reconciler: &reconciler{
				cfg: ReconcilerConfig{
					RolloutsIntegrationEnabled: true,
				},
				kargoClient: fake.NewClientBuilder().Build(),
				listAnalysisRunsFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
				getAnalysisTemplateFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*rollouts.AnalysisTemplate, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo) {
				require.NotNil(t, vi)
				require.Contains(t, vi.Message, "something went wrong")
				require.Contains(t, vi.Message, "error getting AnalysisTemplate")
			},
		},
		{
			name: "AnalysisTemplate not found",
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					Verification: &kargoapi.Verification{
						AnalysisTemplates: []kargoapi.AnalysisTemplateReference{{}},
					},
				},
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.FreightReference{
						Name: "fake-id",
					},
				},
			},
			reconciler: &reconciler{
				cfg: ReconcilerConfig{
					RolloutsIntegrationEnabled: true,
				},
				kargoClient: fake.NewClientBuilder().Build(),
				listAnalysisRunsFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
				getAnalysisTemplateFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*rollouts.AnalysisTemplate, error) {
					return nil, nil
				},
			},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo) {
				require.NotNil(t, vi)
				require.Contains(t, vi.Message, "AnalysisTemplate")
				require.Contains(t, vi.Message, "not found")
			},
		},
		{
			name: "error getting Freight",
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					Verification: &kargoapi.Verification{
						AnalysisTemplates: []kargoapi.AnalysisTemplateReference{{}},
					},
				},
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.FreightReference{
						Name: "fake-id",
					},
				},
			},
			reconciler: &reconciler{
				cfg: ReconcilerConfig{
					RolloutsIntegrationEnabled: true,
				},
				kargoClient: fake.NewClientBuilder().Build(),
				listAnalysisRunsFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
				getAnalysisTemplateFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*rollouts.AnalysisTemplate, error) {
					return &rollouts.AnalysisTemplate{}, nil
				},
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return nil, fmt.Errorf("something went wrong")
				},
			},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo) {
				require.NotNil(t, vi)
				require.Contains(t, vi.Message, "something went wrong")
				require.Contains(t, vi.Message, "error getting Freight")
			},
		},
		{
			name: "Freight not found",
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					Verification: &kargoapi.Verification{
						AnalysisTemplates: []kargoapi.AnalysisTemplateReference{{}},
					},
				},
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.FreightReference{
						Name: "fake-id",
					},
				},
			},
			reconciler: &reconciler{
				cfg: ReconcilerConfig{
					RolloutsIntegrationEnabled: true,
				},
				kargoClient: fake.NewClientBuilder().Build(),
				listAnalysisRunsFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
				getAnalysisTemplateFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*rollouts.AnalysisTemplate, error) {
					return &rollouts.AnalysisTemplate{}, nil
				},
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return nil, nil
				},
			},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo) {
				require.NotNil(t, vi)
				require.Contains(t, vi.Message, "Freight")
				require.Contains(t, vi.Message, "not found")
			},
		},
		{
			name: "error building AnalysisRun",
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					Verification: &kargoapi.Verification{
						AnalysisTemplates: []kargoapi.AnalysisTemplateReference{{}},
					},
				},
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.FreightReference{
						Name: "fake-id",
					},
				},
			},
			reconciler: &reconciler{
				cfg: ReconcilerConfig{
					RolloutsIntegrationEnabled: true,
				},
				kargoClient: fake.NewClientBuilder().Build(),
				listAnalysisRunsFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
				getAnalysisTemplateFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*rollouts.AnalysisTemplate, error) {
					return &rollouts.AnalysisTemplate{}, nil
				},
				getFreightFn: func(
					context.Context,
					client.Client, types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				buildAnalysisRunFn: func(
					*kargoapi.Stage,
					*kargoapi.Freight,
					[]*rollouts.AnalysisTemplate,
				) (*rollouts.AnalysisRun, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo) {
				require.NotNil(t, vi)
				require.Contains(t, vi.Message, "something went wrong")
				require.Contains(t, vi.Message, "error building AnalysisRun for Stage")
			},
		},
		{
			name: "error creating AnalysisRun",
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					Verification: &kargoapi.Verification{
						AnalysisTemplates: []kargoapi.AnalysisTemplateReference{{}},
					},
				},
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.FreightReference{
						Name: "fake-id",
					},
				},
			},
			reconciler: &reconciler{
				cfg: ReconcilerConfig{
					RolloutsIntegrationEnabled: true,
				},
				kargoClient: fake.NewClientBuilder().Build(),
				listAnalysisRunsFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
				getAnalysisTemplateFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*rollouts.AnalysisTemplate, error) {
					return &rollouts.AnalysisTemplate{}, nil
				},
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				buildAnalysisRunFn: func(
					*kargoapi.Stage,
					*kargoapi.Freight,
					[]*rollouts.AnalysisTemplate,
				) (*rollouts.AnalysisRun, error) {
					return &rollouts.AnalysisRun{}, nil
				},
				createAnalysisRunFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo) {
				require.NotNil(t, vi)
				require.Contains(t, vi.Message, "something went wrong")
				require.Contains(t, vi.Message, "error creating AnalysisRun")
			},
		},
		{
			name: "success",
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					Verification: &kargoapi.Verification{
						AnalysisTemplates: []kargoapi.AnalysisTemplateReference{{}},
					},
				},
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.FreightReference{
						Name: "fake-id",
					},
				},
			},
			reconciler: &reconciler{
				cfg: ReconcilerConfig{
					RolloutsIntegrationEnabled: true,
				},
				kargoClient: fake.NewClientBuilder().Build(),
				listAnalysisRunsFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
				getAnalysisTemplateFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*rollouts.AnalysisTemplate, error) {
					return &rollouts.AnalysisTemplate{}, nil
				},
				getFreightFn: func(
					_ context.Context,
					_ client.Client,
					_ types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				buildAnalysisRunFn: func(
					*kargoapi.Stage,
					*kargoapi.Freight,
					[]*rollouts.AnalysisTemplate,
				) (*rollouts.AnalysisRun, error) {
					return &rollouts.AnalysisRun{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fake-run",
							Namespace: "fake-namespace",
						},
					}, nil
				},
				createAnalysisRunFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return nil
				},
			},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo) {
				require.NotNilf(t, vi, "expected non-nil VerificationInfo")
				require.NotEmptyf(t, vi.ID, "expected non-empty VerificationInfo.ID")
				require.Equal(t, kargoapi.VerificationPhasePending, vi.Phase)
				require.Equal(t, &kargoapi.AnalysisRunReference{
					Name:      "fake-run",
					Namespace: "fake-namespace",
				}, vi.AnalysisRun)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				testCase.reconciler.startVerification(
					context.Background(),
					testCase.stage,
				),
			)
		})
	}
}

func TestGetVerificationInfo(t *testing.T) {
	testCases := []struct {
		name       string
		stage      *kargoapi.Stage
		reconciler *reconciler
		assertions func(*testing.T, *kargoapi.VerificationInfo)
	}{
		{
			name:       "rollouts integration not enabled",
			reconciler: &reconciler{},
			stage: &kargoapi.Stage{
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.FreightReference{
						VerificationInfo: &kargoapi.VerificationInfo{},
					},
				},
			},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo) {
				require.NotNil(t, vi)
				require.Contains(
					t,
					vi.Message,
					"Rollouts integration is disabled on this controller",
				)
			},
		},
		{
			name: "error getting AnalysisRun",
			stage: &kargoapi.Stage{
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.FreightReference{
						VerificationInfo: &kargoapi.VerificationInfo{
							AnalysisRun: &kargoapi.AnalysisRunReference{
								Name:      "fake-run",
								Namespace: "fake-namespace",
							},
						},
					},
				},
			},
			reconciler: &reconciler{
				cfg: ReconcilerConfig{
					RolloutsIntegrationEnabled: true,
				},
				kargoClient: fake.NewClientBuilder().Build(),
				getAnalysisRunFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*rollouts.AnalysisRun, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo) {
				require.NotNil(t, vi)
				require.Contains(t, vi.Message, "something went wrong")
				require.Contains(t, vi.Message, "error getting AnalysisRun")
			},
		},
		{
			name: "AnalysisRun not found",
			stage: &kargoapi.Stage{
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.FreightReference{
						VerificationInfo: &kargoapi.VerificationInfo{
							AnalysisRun: &kargoapi.AnalysisRunReference{
								Name:      "fake-run",
								Namespace: "fake-namespace",
							},
						},
					},
				},
			},
			reconciler: &reconciler{
				cfg: ReconcilerConfig{
					RolloutsIntegrationEnabled: true,
				},
				kargoClient: fake.NewClientBuilder().Build(),
				getAnalysisRunFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*rollouts.AnalysisRun, error) {
					return nil, nil
				},
			},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo) {
				require.NotNil(t, vi)
				require.Contains(t, vi.Message, "AnalysisRun")
				require.Contains(t, vi.Message, "not found")
			},
		},
		{
			name: "success",
			stage: &kargoapi.Stage{
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.FreightReference{
						VerificationInfo: &kargoapi.VerificationInfo{
							AnalysisRun: &kargoapi.AnalysisRunReference{
								Name:      "fake-run",
								Namespace: "fake-namespace",
							},
						},
					},
				},
			},
			reconciler: &reconciler{
				cfg: ReconcilerConfig{
					RolloutsIntegrationEnabled: true,
				},
				kargoClient: fake.NewClientBuilder().Build(),
				getAnalysisRunFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*rollouts.AnalysisRun, error) {
					return &rollouts.AnalysisRun{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fake-run",
							Namespace: "fake-namespace",
						},
						Status: rollouts.AnalysisRunStatus{
							Phase: rollouts.AnalysisPhaseSuccessful,
						},
					}, nil
				},
			},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo) {
				require.NotNil(t, vi.StartTime)
				require.Equal(
					t,
					&kargoapi.VerificationInfo{
						StartTime: vi.StartTime,
						Phase:     kargoapi.VerificationPhaseSuccessful,
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "fake-run",
							Namespace: "fake-namespace",
							Phase:     string(rollouts.AnalysisPhaseSuccessful),
						},
					},
					vi,
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				testCase.reconciler.getVerificationInfo(
					context.Background(),
					testCase.stage,
				),
			)
		})
	}
}

func TestAbortVerification(t *testing.T) {
	testCases := []struct {
		name       string
		stage      *kargoapi.Stage
		reconciler *reconciler
		assertions func(*testing.T, *kargoapi.VerificationInfo)
	}{
		{
			name:       "rollouts integration not enabled",
			reconciler: &reconciler{},
			stage: &kargoapi.Stage{
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.FreightReference{
						VerificationInfo: &kargoapi.VerificationInfo{
							ID: "fake-id",
						},
					},
				},
			},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo) {
				require.NotNil(t, vi)
				require.Equal(t, vi.ID, "fake-id")
				require.Contains(
					t,
					vi.Message,
					"Rollouts integration is disabled on this controller",
				)
			},
		},
		{
			name: "error patching AnalysisRun",
			stage: &kargoapi.Stage{
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.FreightReference{
						VerificationInfo: &kargoapi.VerificationInfo{
							ID: "fake-id",
							AnalysisRun: &kargoapi.AnalysisRunReference{
								Name:      "fake-run",
								Namespace: "fake-namespace",
							},
						},
					},
				},
			},
			reconciler: &reconciler{
				cfg: ReconcilerConfig{
					RolloutsIntegrationEnabled: true,
				},
				kargoClient: fake.NewClientBuilder().Build(),
				patchAnalysisRunFn: func(
					context.Context,
					client.Object,
					client.Patch,
					...client.PatchOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo) {
				require.NotNil(t, vi)
				require.Equal(t, "fake-id", vi.ID)
				require.Contains(t, vi.Message, "AnalysisRun")
				require.Contains(t, vi.Message, "something went wrong")
			},
		},
		{
			name: "success",
			stage: &kargoapi.Stage{
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.FreightReference{
						VerificationInfo: &kargoapi.VerificationInfo{
							ID: "fake-id",
							AnalysisRun: &kargoapi.AnalysisRunReference{
								Name:      "fake-run",
								Namespace: "fake-namespace",
							},
						},
					},
				},
			},
			reconciler: &reconciler{
				cfg: ReconcilerConfig{
					RolloutsIntegrationEnabled: true,
				},
				kargoClient: fake.NewClientBuilder().Build(),
				patchAnalysisRunFn: func(
					context.Context,
					client.Object,
					client.Patch,
					...client.PatchOption,
				) error {
					return nil
				},
			},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo) {
				require.NotNil(t, vi)
				require.Equal(t, "fake-id", vi.ID)
				require.Equal(t, kargoapi.VerificationPhaseAborted, vi.Phase)
				require.Equal(t, "Verification aborted by user", vi.Message)
				require.Equal(t, &kargoapi.AnalysisRunReference{
					Name:      "fake-run",
					Namespace: "fake-namespace",
				}, vi.AnalysisRun)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				testCase.reconciler.abortVerification(
					context.Background(),
					testCase.stage,
				),
			)
		})
	}
}

func TestBuildAnalysisRun(t *testing.T) {
	freight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			UID:       "fake-uid",
			Name:      "fake-freight",
			Namespace: "fake-namespace",
		},
	}

	testCases := []struct {
		name       string
		reconciler *reconciler
		stage      *kargoapi.Stage
		freight    *kargoapi.Freight
		templates  []*rollouts.AnalysisTemplate
		assertions func(*testing.T, *kargoapi.Stage, []*rollouts.AnalysisTemplate, *rollouts.AnalysisRun, error)
	}{
		{
			name:       "Builds AnalysisRun successfully",
			reconciler: &reconciler{},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
				},
				Spec: &kargoapi.StageSpec{
					Verification: &kargoapi.Verification{
						AnalysisRunMetadata: &kargoapi.AnalysisRunMetadata{
							Labels: map[string]string{
								"custom":  "label",
								"another": "label",
							},
							Annotations: map[string]string{
								"custom":  "annotation",
								"another": "annotation",
							},
						},
					},
				},
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.FreightReference{
						Name: "fake-id",
					},
				},
			},
			freight: freight,
			templates: []*rollouts.AnalysisTemplate{
				{
					Spec: rollouts.AnalysisTemplateSpec{
						Metrics: []rollouts.Metric{
							{
								Name:             "foo",
								SuccessCondition: "true",
							},
						},
						DryRun: []rollouts.DryRun{
							{
								MetricName: "foo",
							},
						},
						MeasurementRetention: []rollouts.MeasurementRetention{
							{
								MetricName: "foo",
								Limit:      10,
							},
						},
						Args: []rollouts.Argument{
							{
								Name:  "test",
								Value: ptr.To("true"),
							},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				stage *kargoapi.Stage,
				templates []*rollouts.AnalysisTemplate,
				ar *rollouts.AnalysisRun,
				err error,
			) {
				require.NoError(t, err)
				require.NotNil(t, ar)

				require.Contains(t, ar.Name, stage.Name)
				require.Equal(t, ar.Namespace, stage.Namespace)

				require.Equal(t, map[string]string{
					kargoapi.StageLabelKey:   stage.Name,
					kargoapi.FreightLabelKey: stage.Status.CurrentFreight.Name,
					"custom":                 "label",
					"another":                "label",
				}, ar.Labels)
				require.Equal(t, stage.Spec.Verification.AnalysisRunMetadata.Annotations, ar.Annotations)

				require.Equal(t, templates[0].Spec.Metrics, ar.Spec.Metrics)
				require.Equal(t, templates[0].Spec.DryRun, ar.Spec.DryRun)
				require.Equal(t, templates[0].Spec.MeasurementRetention, ar.Spec.MeasurementRetention)
				require.Equal(t, templates[0].Spec.Args, ar.Spec.Args)
			},
		},
		{
			name: "Sets rollout controller instance ID",
			reconciler: &reconciler{
				cfg: ReconcilerConfig{
					RolloutsControllerInstanceID: "fake-instance-id",
				},
			},
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.FreightReference{Name: "fake-id"},
				},
			},
			freight: freight,
			assertions: func(
				t *testing.T,
				_ *kargoapi.Stage,
				_ []*rollouts.AnalysisTemplate,
				ar *rollouts.AnalysisRun,
				err error,
			) {
				require.NoError(t, err)
				require.NotNil(t, ar)

				require.Equal(t, "fake-instance-id", ar.Labels["argo-rollouts.argoproj.io/controller-instance-id"])
			},
		},
		{
			name:       "Flattens multiple templates",
			reconciler: &reconciler{},
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.FreightReference{Name: "fake-id"},
				},
			},
			freight: freight,
			templates: []*rollouts.AnalysisTemplate{
				{
					Spec: rollouts.AnalysisTemplateSpec{
						Metrics: []rollouts.Metric{
							{
								Name:             "foo",
								SuccessCondition: "true",
							},
						},
						Args: []rollouts.Argument{
							{
								Name:  "test",
								Value: ptr.To("true"),
							},
						},
					},
				},
				{
					Spec: rollouts.AnalysisTemplateSpec{
						Metrics: []rollouts.Metric{
							{
								Name:             "bar",
								SuccessCondition: "false",
							},
						},
						Args: []rollouts.Argument{
							{
								Name:  "test",
								Value: ptr.To("true"),
							},
							{
								Name:  "another",
								Value: ptr.To("true"),
							},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.Stage,
				_ []*rollouts.AnalysisTemplate,
				ar *rollouts.AnalysisRun,
				err error,
			) {
				require.NoError(t, err)
				require.NotNil(t, ar)

				require.Len(t, ar.Spec.Metrics, 2)
				require.Len(t, ar.Spec.Args, 2)
			},
		},
		{
			name:       "Merges flattened template args with stage args",
			reconciler: &reconciler{},
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					Verification: &kargoapi.Verification{
						Args: []kargoapi.AnalysisRunArgument{
							{
								Name:  "test",
								Value: "overwrite",
							},
						},
					},
				},
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.FreightReference{Name: "fake-id"},
				},
			},
			freight: freight,
			templates: []*rollouts.AnalysisTemplate{
				{
					Spec: rollouts.AnalysisTemplateSpec{
						Args: []rollouts.Argument{
							{
								Name:  "test",
								Value: ptr.To("true"),
							},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.Stage,
				_ []*rollouts.AnalysisTemplate,
				ar *rollouts.AnalysisRun,
				err error,
			) {
				require.NoError(t, err)
				require.NotNil(t, ar)

				require.Equal(t, []rollouts.Argument{
					{
						Name:  "test",
						Value: ptr.To("overwrite"),
					},
				}, ar.Spec.Args)
			},
		},
		{
			name:       "Sets owner reference to Freight",
			reconciler: &reconciler{},
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.FreightReference{Name: "fake-id"},
				},
			},
			freight: freight,
			assertions: func(
				t *testing.T,
				_ *kargoapi.Stage,
				_ []*rollouts.AnalysisTemplate,
				ar *rollouts.AnalysisRun,
				err error,
			) {
				require.NoError(t, err)
				require.NotNil(t, ar)

				require.Len(t, ar.OwnerReferences, 1)
				require.Equal(t, metav1.OwnerReference{
					APIVersion:         kargoapi.GroupVersion.String(),
					Kind:               "Freight",
					Name:               freight.Name,
					UID:                freight.UID,
					Controller:         ptr.To(true),
					BlockOwnerDeletion: ptr.To(true),
				}, ar.OwnerReferences[0])
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ar, err := testCase.reconciler.buildAnalysisRun(testCase.stage, testCase.freight, testCase.templates)
			testCase.assertions(t, testCase.stage, testCase.templates, ar, err)
		})
	}
}

func TestFlattenTemplates(t *testing.T) {
	metric := func(name, successCondition string) rollouts.Metric {
		return rollouts.Metric{
			Name:             name,
			SuccessCondition: successCondition,
		}
	}
	arg := func(name string, value *string) rollouts.Argument {
		return rollouts.Argument{
			Name:  name,
			Value: value,
		}
	}
	t.Run("Handle empty list", func(t *testing.T) {
		template, err := flattenTemplates([]*rollouts.AnalysisTemplate{})
		require.Nil(t, err)
		require.Len(t, template.Spec.Metrics, 0)
		require.Len(t, template.Spec.Args, 0)

	})
	t.Run("No changes on single template", func(t *testing.T) {
		orig := &rollouts.AnalysisTemplate{
			Spec: rollouts.AnalysisTemplateSpec{
				Metrics: []rollouts.Metric{metric("foo", "{{args.test}}")},
				Args:    []rollouts.Argument{arg("test", ptr.To("true"))},
			},
		}
		template, err := flattenTemplates([]*rollouts.AnalysisTemplate{orig})
		require.Nil(t, err)
		require.Equal(t, orig.Spec, template.Spec)
	})
	t.Run("Merge multiple metrics successfully", func(t *testing.T) {
		fooMetric := metric("foo", "true")
		barMetric := metric("bar", "true")
		template, err := flattenTemplates([]*rollouts.AnalysisTemplate{
			{
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: []rollouts.Metric{fooMetric},
					DryRun: []rollouts.DryRun{{
						MetricName: "foo",
					}},
					MeasurementRetention: []rollouts.MeasurementRetention{{
						MetricName: "foo",
					}},
					Args: nil,
				},
			}, {
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: []rollouts.Metric{barMetric},
					DryRun: []rollouts.DryRun{{
						MetricName: "bar",
					}},
					MeasurementRetention: []rollouts.MeasurementRetention{{
						MetricName: "bar",
					}},
					Args: nil,
				},
			},
		})
		require.Nil(t, err)
		require.Nil(t, template.Spec.Args)
		require.Len(t, template.Spec.Metrics, 2)
		require.Equal(t, fooMetric, template.Spec.Metrics[0])
		require.Equal(t, barMetric, template.Spec.Metrics[1])
	})
	t.Run("Merge analysis templates successfully", func(t *testing.T) {
		fooMetric := metric("foo", "true")
		barMetric := metric("bar", "true")
		template, err := flattenTemplates([]*rollouts.AnalysisTemplate{
			{
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: []rollouts.Metric{fooMetric},
					DryRun: []rollouts.DryRun{
						{
							MetricName: "foo",
						},
					},
					MeasurementRetention: []rollouts.MeasurementRetention{
						{
							MetricName: "foo",
						},
					},
					Args: nil,
				},
			},
			{
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: []rollouts.Metric{barMetric},
					DryRun: []rollouts.DryRun{
						{
							MetricName: "bar",
						},
					},
					MeasurementRetention: []rollouts.MeasurementRetention{
						{
							MetricName: "bar",
						},
					},
					Args: nil,
				},
			},
		})
		require.Nil(t, err)
		require.Nil(t, template.Spec.Args)
		require.Len(t, template.Spec.Metrics, 2)
		require.Equal(t, fooMetric, template.Spec.Metrics[0])
		require.Equal(t, barMetric, template.Spec.Metrics[1])
	})
	t.Run("Merge fail with name collision", func(t *testing.T) {
		fooMetric := metric("foo", "true")
		template, err := flattenTemplates([]*rollouts.AnalysisTemplate{
			{
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: []rollouts.Metric{fooMetric},
					Args:    nil,
				},
			}, {
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: []rollouts.Metric{fooMetric},
					Args:    nil,
				},
			},
		})
		require.Nil(t, template)
		require.Equal(t, err, fmt.Errorf("two metrics have the same name 'foo'"))
	})
	t.Run("Merge fail with dry-run name collision", func(t *testing.T) {
		fooMetric := metric("foo", "true")
		barMetric := metric("bar", "true")
		template, err := flattenTemplates([]*rollouts.AnalysisTemplate{
			{
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: []rollouts.Metric{fooMetric},
					DryRun: []rollouts.DryRun{
						{
							MetricName: "foo",
						},
					},
					Args: nil,
				},
			}, {
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: []rollouts.Metric{barMetric},
					DryRun: []rollouts.DryRun{
						{
							MetricName: "foo",
						},
					},
					Args: nil,
				},
			},
		})
		require.Nil(t, template)
		require.Equal(t, err, fmt.Errorf("two Dry-Run metric rules have the same name 'foo'"))
	})
	t.Run("Merge fail with measurement retention metrics name collision", func(t *testing.T) {
		fooMetric := metric("foo", "true")
		barMetric := metric("bar", "true")
		template, err := flattenTemplates([]*rollouts.AnalysisTemplate{
			{
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: []rollouts.Metric{fooMetric},
					MeasurementRetention: []rollouts.MeasurementRetention{
						{
							MetricName: "foo",
						},
					},
					Args: nil,
				},
			}, {
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: []rollouts.Metric{barMetric},
					MeasurementRetention: []rollouts.MeasurementRetention{
						{
							MetricName: "foo",
						},
					},
					Args: nil,
				},
			},
		})
		require.Nil(t, template)
		require.Equal(t, err, fmt.Errorf("two Measurement Retention metric rules have the same name 'foo'"))
	})
	t.Run("Merge multiple args successfully", func(t *testing.T) {
		fooArgs := arg("foo", ptr.To("true"))
		barArgs := arg("bar", ptr.To("true"))
		template, err := flattenTemplates([]*rollouts.AnalysisTemplate{
			{
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: nil,
					Args:    []rollouts.Argument{fooArgs},
				},
			}, {
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: nil,
					Args:    []rollouts.Argument{barArgs},
				},
			},
		})
		require.Nil(t, err)
		require.Len(t, template.Spec.Args, 2)
		require.Equal(t, fooArgs, template.Spec.Args[0])
		require.Equal(t, barArgs, template.Spec.Args[1])
	})
	t.Run(" Merge args with same name but only one has value", func(t *testing.T) {
		fooArgsValue := arg("foo", ptr.To("true"))
		fooArgsNoValue := arg("foo", nil)
		template, err := flattenTemplates([]*rollouts.AnalysisTemplate{
			{
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: nil,
					Args:    []rollouts.Argument{fooArgsValue},
				},
			}, {
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: nil,
					Args:    []rollouts.Argument{fooArgsNoValue},
				},
			},
		})
		require.Nil(t, err)
		require.Len(t, template.Spec.Args, 1)
		require.Contains(t, template.Spec.Args, fooArgsValue)
	})
	t.Run("Error: merge args with same name and both have values", func(t *testing.T) {
		fooArgs := arg("foo", ptr.To("true"))
		fooArgsWithDiffValue := arg("foo", ptr.To("false"))
		template, err := flattenTemplates([]*rollouts.AnalysisTemplate{
			{
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: nil,
					Args:    []rollouts.Argument{fooArgs},
				},
			}, {
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: nil,
					Args:    []rollouts.Argument{fooArgsWithDiffValue},
				},
			},
		})
		require.Equal(t, fmt.Errorf("Argument `foo` specified multiple times with different values: 'true', 'false'"), err)
		require.Nil(t, template)
	})
}

func TestMergeArgs(t *testing.T) {
	{
		// nil list
		args, err := mergeArgs(nil, nil)
		require.NoError(t, err)
		require.Nil(t, args)
	}
	{
		// empty list
		args, err := mergeArgs(nil, []rollouts.Argument{})
		require.NoError(t, err)
		require.Equal(t, []rollouts.Argument{}, args)
	}
	{
		// use defaults
		args, err := mergeArgs(
			nil, []rollouts.Argument{
				{
					Name:  "foo",
					Value: ptr.To("bar"),
				},
				{
					Name: "my-secret",
					ValueFrom: &rollouts.ValueFrom{
						SecretKeyRef: &rollouts.SecretKeyRef{
							Name: "name",
							Key:  "key",
						},
					},
				},
			})
		require.NoError(t, err)
		require.Len(t, args, 2)
		require.Equal(t, "foo", args[0].Name)
		require.Equal(t, "bar", *args[0].Value)
		require.Equal(t, "my-secret", args[1].Name)
		require.NotNil(t, args[1].ValueFrom)
	}
	{
		// overwrite defaults
		args, err := mergeArgs(
			[]rollouts.Argument{
				{
					Name:  "foo",
					Value: ptr.To("overwrite"),
				},
			}, []rollouts.Argument{
				{
					Name:  "foo",
					Value: ptr.To("bar"),
				},
			})
		require.NoError(t, err)
		require.Len(t, args, 1)
		require.Equal(t, "foo", args[0].Name)
		require.Equal(t, "overwrite", *args[0].Value)
	}
	{
		// not resolved
		args, err := mergeArgs(
			[]rollouts.Argument{
				{
					Name: "foo",
				},
			}, []rollouts.Argument{
				{
					Name: "foo",
				},
			})
		require.EqualError(t, err, "args.foo was not resolved")
		require.Nil(t, args)
	}
	{
		// extra arg
		args, err := mergeArgs(
			[]rollouts.Argument{
				{
					Name:  "foo",
					Value: ptr.To("my-value"),
				},
				{
					Name:  "extra-arg",
					Value: ptr.To("extra-value"),
				},
			}, []rollouts.Argument{
				{
					Name: "foo",
				},
			})
		require.NoError(t, err)
		require.Len(t, args, 1)
		require.Equal(t, "foo", args[0].Name)
		require.Equal(t, "my-value", *args[0].Value)
	}
}
