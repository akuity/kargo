package rollouts

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	rolloutsapi "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
)

func TestAnalysisRunBuilder_Build(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, rolloutsapi.AddToScheme(scheme))

	tests := []struct {
		name         string
		cfg          Config
		namespace    string
		verification *kargoapi.Verification
		objects      []client.Object
		options      []AnalysisRunOption
		assertions   func(*testing.T, *rolloutsapi.AnalysisRun, error)
	}{
		{
			name:      "nil verification config returns error",
			namespace: "default",
			assertions: func(t *testing.T, ar *rolloutsapi.AnalysisRun, err error) {
				assert.ErrorContains(t, err, "missing verification configuration")
				assert.Nil(t, ar)
			},
		},
		{
			name:      "basic AnalysisRun creation",
			namespace: "default",
			cfg: Config{
				ControllerInstanceID: "test-controller",
			},
			verification: &kargoapi.Verification{
				AnalysisTemplates: []kargoapi.AnalysisTemplateReference{
					{Name: "template1"},
				},
				Args: []kargoapi.AnalysisRunArgument{
					{Name: "arg1", Value: "val1"},
				},
				AnalysisRunMetadata: &kargoapi.AnalysisRunMetadata{
					Labels: map[string]string{
						"custom-label": "value",
					},
					Annotations: map[string]string{
						"custom-annotation": "value",
					},
				},
			},
			objects: []client.Object{
				&rolloutsapi.AnalysisTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "template1",
						Namespace: "default",
					},
					Spec: rolloutsapi.AnalysisTemplateSpec{
						Metrics: []rolloutsapi.Metric{
							{Name: "metric1"},
						},
						Args: []rolloutsapi.Argument{
							{Name: "arg1"},
						},
					},
				},
			},
			options: []AnalysisRunOption{
				WithNamePrefix("prefix"),
				WithNameSuffix("suffix"),
				WithExtraLabels(map[string]string{"extra": "label"}),
			},
			assertions: func(t *testing.T, ar *rolloutsapi.AnalysisRun, err error) {
				require.NoError(t, err)
				require.NotNil(t, ar)

				assert.True(t, strings.HasPrefix(ar.Name, "prefix."))
				assert.True(t, strings.HasSuffix(ar.Name, ".suffix"))

				assert.Equal(t, "test-controller", ar.Labels[controllerInstanceIDLabelKey])
				assert.Equal(t, "value", ar.Labels["custom-label"])
				assert.Equal(t, "label", ar.Labels["extra"])
				assert.Equal(t, "value", ar.Annotations["custom-annotation"])

				assert.Len(t, ar.Spec.Metrics, 1)
				assert.Equal(t, "metric1", ar.Spec.Metrics[0].Name)
				assert.Len(t, ar.Spec.Args, 1)
				assert.Equal(t, "arg1", ar.Spec.Args[0].Name)
				assert.Equal(t, "val1", *ar.Spec.Args[0].Value)
			},
		},
		{
			name:      "owner references",
			namespace: "default",
			verification: &kargoapi.Verification{
				AnalysisTemplates: []kargoapi.AnalysisTemplateReference{
					{Name: "template1"},
				},
			},
			objects: []client.Object{
				&rolloutsapi.AnalysisTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "template1",
						Namespace: "default",
					},
				},
				&unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "apps/v1",
						"kind":       "Deployment",
						"metadata": map[string]any{
							"name":      "owner-deploy",
							"namespace": "default",
							"uid":       "test-uid",
						},
					},
				},
			},
			options: []AnalysisRunOption{
				WithOwner(Owner{
					APIVersion:    "apps/v1",
					Kind:          "Deployment",
					Reference:     types.NamespacedName{Name: "owner-deploy", Namespace: "default"},
					BlockDeletion: true,
				}),
			},
			assertions: func(t *testing.T, ar *rolloutsapi.AnalysisRun, err error) {
				require.NoError(t, err)
				require.NotNil(t, ar)
				require.Len(t, ar.OwnerReferences, 1)

				owner := ar.OwnerReferences[0]
				assert.Equal(t, "apps/v1", owner.APIVersion)
				assert.Equal(t, "Deployment", owner.Kind)
				assert.Equal(t, "owner-deploy", owner.Name)
				assert.Equal(t, "test-uid", string(owner.UID))
				assert.True(t, *owner.BlockOwnerDeletion)
			},
		},
		{
			name:      "multiple templates",
			namespace: "default",
			verification: &kargoapi.Verification{
				AnalysisTemplates: []kargoapi.AnalysisTemplateReference{
					{Name: "template1"},
					{Name: "template2"},
				},
			},
			objects: []client.Object{
				&rolloutsapi.AnalysisTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "template1",
						Namespace: "default",
					},
					Spec: rolloutsapi.AnalysisTemplateSpec{
						Metrics: []rolloutsapi.Metric{{Name: "metric1"}},
						Args:    []rolloutsapi.Argument{{Name: "arg1", Value: ptr.To("val1")}},
					},
				},
				&rolloutsapi.AnalysisTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "template2",
						Namespace: "default",
					},
					Spec: rolloutsapi.AnalysisTemplateSpec{
						Metrics: []rolloutsapi.Metric{{Name: "metric2"}},
						Args:    []rolloutsapi.Argument{{Name: "arg2", Value: ptr.To("val2")}},
					},
				},
			},
			assertions: func(t *testing.T, ar *rolloutsapi.AnalysisRun, err error) {
				require.NoError(t, err)
				require.NotNil(t, ar)
				assert.Len(t, ar.Spec.Metrics, 2)
				assert.Equal(t, "metric1", ar.Spec.Metrics[0].Name)
				assert.Equal(t, "metric2", ar.Spec.Metrics[1].Name)
			},
		},
		{
			name:      "template not found",
			namespace: "default",
			verification: &kargoapi.Verification{
				AnalysisTemplates: []kargoapi.AnalysisTemplateReference{
					{Name: "nonexistent"},
				},
			},
			assertions: func(t *testing.T, ar *rolloutsapi.AnalysisRun, err error) {
				assert.ErrorContains(t, err, "get AnalysisRun")
				assert.Nil(t, ar)
			},
		},
		{
			name:      "owner not found",
			namespace: "default",
			verification: &kargoapi.Verification{
				AnalysisTemplates: []kargoapi.AnalysisTemplateReference{
					{Name: "template1"},
				},
			},
			objects: []client.Object{
				&rolloutsapi.AnalysisTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "template1",
						Namespace: "default",
					},
				},
			},
			options: []AnalysisRunOption{
				WithOwner(Owner{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Reference:  types.NamespacedName{Name: "nonexistent", Namespace: "default"},
				}),
			},
			assertions: func(t *testing.T, ar *rolloutsapi.AnalysisRun, err error) {
				assert.ErrorContains(t, err, "get Deployment")
				assert.Nil(t, ar)
			},
		},
		{
			name:      "spec building error",
			namespace: "default",
			verification: &kargoapi.Verification{
				AnalysisTemplates: []kargoapi.AnalysisTemplateReference{
					{Name: "template1"},
					{Name: "template2"},
				},
			},
			objects: []client.Object{
				&rolloutsapi.AnalysisTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "template1",
						Namespace: "default",
					},
					Spec: rolloutsapi.AnalysisTemplateSpec{
						Metrics: []rolloutsapi.Metric{
							{Name: "duplicate-metric"},
						},
					},
				},
				&rolloutsapi.AnalysisTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "template2",
						Namespace: "default",
					},
					Spec: rolloutsapi.AnalysisTemplateSpec{
						Metrics: []rolloutsapi.Metric{
							{Name: "duplicate-metric"},
						},
					},
				},
			},
			assertions: func(t *testing.T, ar *rolloutsapi.AnalysisRun, err error) {
				assert.ErrorContains(t, err, "build spec")
				assert.Nil(t, ar)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				Build()

			builder := NewAnalysisRunBuilder(c, tt.cfg)
			ar, err := builder.Build(context.Background(), tt.namespace, tt.verification, tt.options...)
			tt.assertions(t, ar, err)
		})
	}
}

func TestAnalysisRunBuilder_generateName(t *testing.T) {
	builder := &AnalysisRunBuilder{}

	tests := []struct {
		name       string
		prefix     string
		suffix     string
		assertions func(*testing.T, string)
	}{
		{
			name:   "no prefix or suffix",
			prefix: "",
			suffix: "",
			assertions: func(t *testing.T, result string) {
				assert.Len(t, result, 26) // ULID length
			},
		},
		{
			name:   "with prefix",
			prefix: "test",
			suffix: "",
			assertions: func(t *testing.T, result string) {
				assert.True(t, strings.HasPrefix(result, "test."))
				assert.Len(t, strings.Split(result, "."), 2)
			},
		},
		{
			name:   "with suffix",
			prefix: "",
			suffix: "suffix",
			assertions: func(t *testing.T, result string) {
				assert.True(t, strings.HasSuffix(result, ".suffix"))
				assert.Len(t, strings.Split(result, "."), 2)
			},
		},
		{
			name:   "with prefix and suffix",
			prefix: "test",
			suffix: "suffix",
			assertions: func(t *testing.T, result string) {
				assert.True(t, strings.HasPrefix(result, "test."))
				assert.True(t, strings.HasSuffix(result, ".suffix"))
				assert.Len(t, strings.Split(result, "."), 3)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builder.generateName(tt.prefix, tt.suffix)
			tt.assertions(t, result)
		})
	}
}

func TestAnalysisRunBuilder_buildMetadata(t *testing.T) {
	builder := &AnalysisRunBuilder{
		cfg: Config{
			ControllerInstanceID: "test-controller",
		},
	}

	tests := []struct {
		name        string
		namespace   string
		objName     string
		metadata    *kargoapi.AnalysisRunMetadata
		extraLabels map[string]string
		assertions  func(*testing.T, metav1.ObjectMeta)
	}{
		{
			name:      "basic metadata",
			namespace: "test-ns",
			objName:   "test-name",
			assertions: func(t *testing.T, meta metav1.ObjectMeta) {
				assert.Equal(t, "test-ns", meta.Namespace)
				assert.Equal(t, "test-name", meta.Name)
				assert.Equal(t, "test-controller", meta.Labels[controllerInstanceIDLabelKey])
			},
		},
		{
			name:      "with metadata and extra labels",
			namespace: "test-ns",
			objName:   "test-name",
			metadata: &kargoapi.AnalysisRunMetadata{
				Labels: map[string]string{
					"label1": "value1",
				},
				Annotations: map[string]string{
					"anno1": "value1",
				},
			},
			extraLabels: map[string]string{
				"extra": "value",
			},
			assertions: func(t *testing.T, meta metav1.ObjectMeta) {
				assert.Equal(t, "value1", meta.Labels["label1"])
				assert.Equal(t, "value", meta.Labels["extra"])
				assert.Equal(t, "value1", meta.Annotations["anno1"])
				assert.Equal(t, "test-controller", meta.Labels[controllerInstanceIDLabelKey])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builder.buildMetadata(tt.namespace, tt.objName, tt.metadata, tt.extraLabels)
			tt.assertions(t, result)
		})
	}
}

func TestAnalysisRunBuilder_buildSpec(t *testing.T) {
	tests := []struct {
		name       string
		templates  []*rolloutsapi.AnalysisTemplate
		args       []kargoapi.AnalysisRunArgument
		assertions func(*testing.T, rolloutsapi.AnalysisRunSpec, error)
	}{
		{
			name:      "empty templates and args",
			templates: []*rolloutsapi.AnalysisTemplate{},
			args:      []kargoapi.AnalysisRunArgument{},
			assertions: func(t *testing.T, spec rolloutsapi.AnalysisRunSpec, err error) {
				require.NoError(t, err)
				assert.Empty(t, spec.Metrics)
				assert.Empty(t, spec.Args)
				assert.Empty(t, spec.DryRun)
				assert.Empty(t, spec.MeasurementRetention)
			},
		},
		{
			name: "single template with metrics and args",
			templates: []*rolloutsapi.AnalysisTemplate{
				{
					Spec: rolloutsapi.AnalysisTemplateSpec{
						Metrics: []rolloutsapi.Metric{
							{Name: "metric1", Provider: rolloutsapi.MetricProvider{Prometheus: &rolloutsapi.PrometheusMetric{}}},
						},
						Args: []rolloutsapi.Argument{
							{Name: "param1"},
						},
						DryRun: []rolloutsapi.DryRun{
							{MetricName: "metric1"},
						},
						MeasurementRetention: []rolloutsapi.MeasurementRetention{
							{MetricName: "metric1", Limit: int32(5)},
						},
					},
				},
			},
			args: []kargoapi.AnalysisRunArgument{
				{Name: "param1", Value: "value1"},
			},
			assertions: func(t *testing.T, spec rolloutsapi.AnalysisRunSpec, err error) {
				require.NoError(t, err)
				assert.Len(t, spec.Metrics, 1)

				assert.Equal(t, "metric1", spec.Metrics[0].Name)
				assert.NotNil(t, spec.Metrics[0].Provider.Prometheus)

				assert.Len(t, spec.Args, 1)
				assert.Equal(t, "param1", spec.Args[0].Name)
				assert.Equal(t, "value1", *spec.Args[0].Value)

				assert.Len(t, spec.DryRun, 1)
				assert.Equal(t, "metric1", spec.DryRun[0].MetricName)

				assert.Len(t, spec.MeasurementRetention, 1)
				assert.Equal(t, "metric1", spec.MeasurementRetention[0].MetricName)
				assert.Equal(t, int32(5), spec.MeasurementRetention[0].Limit)
			},
		},
		{
			name: "template flattening error",
			templates: []*rolloutsapi.AnalysisTemplate{
				{
					Spec: rolloutsapi.AnalysisTemplateSpec{
						Metrics: []rolloutsapi.Metric{{Name: "metric1"}},
					},
				},
				{
					Spec: rolloutsapi.AnalysisTemplateSpec{
						Metrics: []rolloutsapi.Metric{{Name: "metric1"}},
					},
				},
			},
			assertions: func(t *testing.T, _ rolloutsapi.AnalysisRunSpec, err error) {
				assert.ErrorContains(t, err, "flatten templates")
			},
		},
		{
			name: "argument error",
			templates: []*rolloutsapi.AnalysisTemplate{
				{
					Spec: rolloutsapi.AnalysisTemplateSpec{
						Args: []rolloutsapi.Argument{
							{Name: "param1"},
						},
					},
				},
			},
			args: []kargoapi.AnalysisRunArgument{
				{Name: "param1"},
			},
			assertions: func(t *testing.T, _ rolloutsapi.AnalysisRunSpec, err error) {
				assert.ErrorContains(t, err, "build arguments")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := &AnalysisRunBuilder{}
			spec, err := builder.buildSpec(tt.templates, tt.args)
			tt.assertions(t, spec, err)
		})
	}
}

func TestAnalysisRunBuilder_buildArgs(t *testing.T) {
	tests := []struct {
		name       string
		template   *rolloutsapi.AnalysisTemplate
		args       []kargoapi.AnalysisRunArgument
		assertions func(*testing.T, []rolloutsapi.Argument, error)
	}{
		{
			name: "nil args",
			template: &rolloutsapi.AnalysisTemplate{
				Spec: rolloutsapi.AnalysisTemplateSpec{
					Args: []rolloutsapi.Argument{
						{Name: "param1", Value: ptr.To("value1")},
					},
				},
			},
			args: nil,
			assertions: func(t *testing.T, args []rolloutsapi.Argument, err error) {
				require.NoError(t, err)
				assert.Len(t, args, 1)
				assert.Equal(t, "param1", args[0].Name)
				assert.Equal(t, "value1", *args[0].Value)
			},
		},
		{
			name: "merge template and run args",
			template: &rolloutsapi.AnalysisTemplate{
				Spec: rolloutsapi.AnalysisTemplateSpec{
					Args: []rolloutsapi.Argument{
						{Name: "param1"},
						{Name: "param2", Value: ptr.To("default2")},
					},
				},
			},
			args: []kargoapi.AnalysisRunArgument{
				{Name: "param1", Value: "value1"},
			},
			assertions: func(t *testing.T, args []rolloutsapi.Argument, err error) {
				require.NoError(t, err)
				assert.Len(t, args, 2)
				assert.Equal(t, "value1", *args[0].Value)
				assert.Equal(t, "default2", *args[1].Value)
			},
		},
		{
			name: "argument conflict",
			template: &rolloutsapi.AnalysisTemplate{
				Spec: rolloutsapi.AnalysisTemplateSpec{
					Args: []rolloutsapi.Argument{
						{Name: "param1"},
					},
				},
			},
			args: []kargoapi.AnalysisRunArgument{
				{Name: "param1"},
			},
			assertions: func(t *testing.T, _ []rolloutsapi.Argument, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "merge arguments")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := &AnalysisRunBuilder{}
			args, err := builder.buildArgs(tt.template, tt.args)
			tt.assertions(t, args, err)
		})
	}
}

func TestAnalysisRunBuilder_buildOwnerReferences(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, rolloutsapi.AddToScheme(scheme))

	tests := []struct {
		name       string
		owners     []Owner
		objects    []client.Object
		assertions func(*testing.T, []metav1.OwnerReference, error)
	}{
		{
			name:   "empty owners list",
			owners: []Owner{},
			assertions: func(t *testing.T, refs []metav1.OwnerReference, err error) {
				require.NoError(t, err)
				assert.Empty(t, refs)
			},
		},
		{
			name: "single deployment owner",
			owners: []Owner{
				{
					APIVersion:    "apps/v1",
					Kind:          "Deployment",
					Reference:     types.NamespacedName{Name: "test-deploy", Namespace: "default"},
					BlockDeletion: true,
				},
			},
			objects: []client.Object{
				&unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "apps/v1",
						"kind":       "Deployment",
						"metadata": map[string]any{
							"name":      "test-deploy",
							"namespace": "default",
							"uid":       "test-uid",
						},
					},
				},
			},
			assertions: func(t *testing.T, refs []metav1.OwnerReference, err error) {
				require.NoError(t, err)
				require.Len(t, refs, 1)
				assert.Equal(t, metav1.OwnerReference{
					APIVersion:         "apps/v1",
					Kind:               "Deployment",
					Name:               "test-deploy",
					UID:                "test-uid",
					BlockOwnerDeletion: ptr.To(true),
				}, refs[0])
			},
		},
		{
			name: "multiple owners of different kinds",
			owners: []Owner{
				{
					APIVersion:    "apps/v1",
					Kind:          "Deployment",
					Reference:     types.NamespacedName{Name: "test-deploy", Namespace: "default"},
					BlockDeletion: true,
				},
				{
					APIVersion:    "v1",
					Kind:          "Service",
					Reference:     types.NamespacedName{Name: "test-svc", Namespace: "default"},
					BlockDeletion: false,
				},
			},
			objects: []client.Object{
				&unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "apps/v1",
						"kind":       "Deployment",
						"metadata": map[string]any{
							"name":      "test-deploy",
							"namespace": "default",
							"uid":       "deploy-uid",
						},
					},
				},
				&unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "v1",
						"kind":       "Service",
						"metadata": map[string]any{
							"name":      "test-svc",
							"namespace": "default",
							"uid":       "svc-uid",
						},
					},
				},
			},
			assertions: func(t *testing.T, refs []metav1.OwnerReference, err error) {
				require.NoError(t, err)
				require.Len(t, refs, 2)
				assert.Equal(t, []metav1.OwnerReference{
					{
						APIVersion:         "apps/v1",
						Kind:               "Deployment",
						Name:               "test-deploy",
						UID:                "deploy-uid",
						BlockOwnerDeletion: ptr.To(true),
					},
					{
						APIVersion:         "v1",
						Kind:               "Service",
						Name:               "test-svc",
						UID:                "svc-uid",
						BlockOwnerDeletion: ptr.To(false),
					},
				}, refs)
			},
		},
		{
			name: "owner not found",
			owners: []Owner{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Reference:  types.NamespacedName{Name: "nonexistent", Namespace: "default"},
				},
			},
			assertions: func(t *testing.T, refs []metav1.OwnerReference, err error) {
				assert.ErrorContains(t, err, "get Deployment")
				assert.Nil(t, refs)
			},
		},
		{
			name: "multiple owners: one not found",
			owners: []Owner{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Reference:  types.NamespacedName{Name: "test-deploy", Namespace: "default"},
				},
				{
					APIVersion: "v1",
					Kind:       "Service",
					Reference:  types.NamespacedName{Name: "nonexistent", Namespace: "default"},
				},
			},
			objects: []client.Object{
				&unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "apps/v1",
						"kind":       "Deployment",
						"metadata": map[string]any{
							"name":      "test-deploy",
							"namespace": "default",
							"uid":       "deploy-uid",
						},
					},
				},
			},
			assertions: func(t *testing.T, refs []metav1.OwnerReference, err error) {
				assert.ErrorContains(t, err, "get Service")
				assert.Nil(t, refs)
			},
		},
		{
			name: "custom resource owner",
			owners: []Owner{
				{
					APIVersion:    "custom.io/v1",
					Kind:          "CustomKind",
					Reference:     types.NamespacedName{Name: "custom-res", Namespace: "default"},
					BlockDeletion: true,
				},
			},
			objects: []client.Object{
				&unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "custom.io/v1",
						"kind":       "CustomKind",
						"metadata": map[string]any{
							"name":      "custom-res",
							"namespace": "default",
							"uid":       "custom-uid",
						},
					},
				},
			},
			assertions: func(t *testing.T, refs []metav1.OwnerReference, err error) {
				require.NoError(t, err)
				require.Len(t, refs, 1)
				assert.Equal(t, metav1.OwnerReference{
					APIVersion:         "custom.io/v1",
					Kind:               "CustomKind",
					Name:               "custom-res",
					UID:                "custom-uid",
					BlockOwnerDeletion: ptr.To(true),
				}, refs[0])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				Build()

			builder := &AnalysisRunBuilder{client: c}
			refs, err := builder.buildOwnerReferences(context.Background(), tt.owners)
			tt.assertions(t, refs, err)
		})
	}
}

func TestAnalysisRunBuilder_getAnalysisTemplates(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, rolloutsapi.AddToScheme(scheme))

	tests := []struct {
		name       string
		namespace  string
		references []kargoapi.AnalysisTemplateReference
		objects    []client.Object
		assertions func(*testing.T, []*rolloutsapi.AnalysisTemplate, error)
	}{
		{
			name:       "empty references",
			namespace:  "default",
			references: []kargoapi.AnalysisTemplateReference{},
			assertions: func(t *testing.T, templates []*rolloutsapi.AnalysisTemplate, err error) {
				require.NoError(t, err)
				assert.Empty(t, templates)
			},
		},
		{
			name:      "single template",
			namespace: "default",
			references: []kargoapi.AnalysisTemplateReference{
				{Name: "template1"},
			},
			objects: []client.Object{
				&rolloutsapi.AnalysisTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "template1",
						Namespace: "default",
					},
					Spec: rolloutsapi.AnalysisTemplateSpec{
						Metrics: []rolloutsapi.Metric{{Name: "metric1"}},
					},
				},
			},
			assertions: func(t *testing.T, templates []*rolloutsapi.AnalysisTemplate, err error) {
				require.NoError(t, err)
				assert.Len(t, templates, 1)
				assert.Equal(t, "template1", templates[0].Name)
				assert.Len(t, templates[0].Spec.Metrics, 1)
			},
		},
		{
			name:      "template not found",
			namespace: "default",
			references: []kargoapi.AnalysisTemplateReference{
				{Name: "nonexistent"},
			},
			assertions: func(t *testing.T, templates []*rolloutsapi.AnalysisTemplate, err error) {
				assert.ErrorContains(t, err, "get AnalysisRun")
				assert.Nil(t, templates)
			},
		},
		{
			name:      "multiple templates",
			namespace: "default",
			references: []kargoapi.AnalysisTemplateReference{
				{Name: "template1"},
				{Name: "template2"},
			},
			objects: []client.Object{
				&rolloutsapi.AnalysisTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "template1",
						Namespace: "default",
					},
					Spec: rolloutsapi.AnalysisTemplateSpec{
						Metrics: []rolloutsapi.Metric{{Name: "metric1"}},
					},
				},
				&rolloutsapi.AnalysisTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "template2",
						Namespace: "default",
					},
					Spec: rolloutsapi.AnalysisTemplateSpec{
						Metrics: []rolloutsapi.Metric{{Name: "metric2"}},
					},
				},
			},
			assertions: func(t *testing.T, templates []*rolloutsapi.AnalysisTemplate, err error) {
				require.NoError(t, err)
				assert.Len(t, templates, 2)
				assert.Equal(t, "template1", templates[0].Name)
				assert.Equal(t, "template2", templates[1].Name)
			},
		},
		{
			name:      "namespace mismatch",
			namespace: "default",
			references: []kargoapi.AnalysisTemplateReference{
				{Name: "template1"},
			},
			objects: []client.Object{
				&rolloutsapi.AnalysisTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "template1",
						Namespace: "different",
					},
				},
			},
			assertions: func(t *testing.T, templates []*rolloutsapi.AnalysisTemplate, err error) {
				assert.ErrorContains(t, err, "get AnalysisRun")
				assert.Nil(t, templates)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				Build()

			builder := &AnalysisRunBuilder{client: c}
			templates, err := builder.getAnalysisTemplates(context.Background(), tt.namespace, tt.references)
			tt.assertions(t, templates, err)
		})
	}
}
