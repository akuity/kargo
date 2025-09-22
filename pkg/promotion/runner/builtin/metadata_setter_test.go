package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_metadataSetter_convert(t *testing.T) {
	tests := []validationTestCase{
		{
			name:   "missing updates field",
			config: promotion.Config{},
			expectedProblems: []string{
				"(root): updates is required",
			},
		},
		{
			name:   "invalid config type",
			config: promotion.Config{"updates": "not-an-array"},
			expectedProblems: []string{
				"updates: Invalid type. Expected: array, given: string",
			},
		},
		{
			name: "missing required fields in updates",
			config: promotion.Config{
				"updates": []map[string]any{
					{
						// missing kind, name, values
					},
				},
			},
			expectedProblems: []string{
				"updates.0: kind is required",
				"updates.0: name is required",
				"updates.0: values is required",
			},
		},
		{
			name: "invalid kind",
			config: promotion.Config{
				"updates": []map[string]any{
					{
						"kind":   "InvalidKind",
						"name":   "test",
						"values": map[string]any{"key": "value"},
					},
				},
			},
			expectedProblems: []string{
				"updates.0.kind: updates.0.kind must be one of the following: \"Stage\", \"Freight\"",
			},
		},
		{
			name: "empty name",
			config: promotion.Config{
				"updates": []map[string]any{
					{
						"kind":   "Stage",
						"name":   "",
						"values": map[string]any{"key": "value"},
					},
				},
			},
			expectedProblems: []string{
				"name: String length must be greater than or equal to 1",
			},
		},
		{
			name: "empty values",
			config: promotion.Config{
				"updates": []map[string]any{
					{
						"kind":   "Stage",
						"name":   "test",
						"values": map[string]any{},
					},
				},
			},
			expectedProblems: []string{
				"values: Must have at least 1 properties",
			},
		},
		{
			name: "valid kitchen sink",
			config: promotion.Config{
				"updates": []map[string]any{
					{
						"kind": "Stage",
						"name": "test-stage",
						"values": map[string]any{
							"string":  "value",
							"number":  42,
							"bool":    true,
							"nullKey": nil,
							"object": map[string]any{
								"nested": "value",
								"array":  []any{"item1", "item2"},
								"deep": map[string]any{
									"foo":  "bar",
									"nums": []any{1, 2, 3},
								},
							},
						},
					},
					{
						"kind": "Freight",
						"name": "test-freight",
						"values": map[string]any{
							"deployed": true,
							"version":  "1.0.0",
						},
					},
				},
			},
			expectedProblems: nil,
		},
	}

	r := newMetadataSetter(promotion.StepRunnerCapabilities{})
	runner, ok := r.(*metadataSetter)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

func Test_metadataSetter_run(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	const (
		testObjName = "test-obj"
		testProject = "test-project"
	)

	testData := map[string]apiextensionsv1.JSON{
		"string":  {Raw: []byte(`"bar"`)},
		"num":     {Raw: []byte(`42`)},
		"complex": {Raw: []byte(`{"foo": "bar", "bat": "baz"}`)},
	}

	testValueUpdates := map[string]any{
		"string":  "bar",
		"num":     43,
		"complex": map[string]any{"updated": true},
		"new":     "success!",
	}

	tests := []struct {
		name       string
		client     client.Client
		cfg        builtin.SetMetadataConfig
		assertions func(*testing.T, promotion.StepResult, client.Client, error)
	}{
		{
			name: "unsupported kind",
			cfg: builtin.SetMetadataConfig{
				Updates: []builtin.Update{{
					Kind: "UnsupportedKind",
					Name: "test-resource",
					// Values: map[string]any{"key": "value"},
				}},
			},
			assertions: func(
				t *testing.T,
				res promotion.StepResult,
				_ client.Client,
				err error,
			) {
				require.ErrorContains(t, err, "unsupported kind")
				require.Equal(t, res.Status, kargoapi.PromotionStepStatusFailed)
			},
		},
		{
			name:   "Stage not found",
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			cfg: builtin.SetMetadataConfig{
				Updates: []builtin.Update{{
					Kind: "Stage",
					Name: "nonexistent",
				}},
			},
			assertions: func(
				t *testing.T,
				res promotion.StepResult,
				_ client.Client,
				err error,
			) {
				require.ErrorContains(t, err, "error getting Stage")
				require.Equal(t, res.Status, kargoapi.PromotionStepStatusErrored)
			},
		},
		{
			name: "error patching Stage status",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:            testObjName,
						Namespace:       testProject,
						ResourceVersion: "invalid", // This will force the patch to fail
					},
				},
			).WithStatusSubresource(&kargoapi.Stage{}).Build(),
			cfg: builtin.SetMetadataConfig{
				Updates: []builtin.Update{{
					Kind:   "Stage",
					Name:   testObjName,
					Values: testValueUpdates,
				}},
			},
			assertions: func(
				t *testing.T,
				res promotion.StepResult,
				_ client.Client,
				err error,
			) {
				require.ErrorContains(t, err, "error patching status of Stage")
				require.ErrorContains(t, err, "can not convert resourceVersion")
				require.Equal(t, res.Status, kargoapi.PromotionStepStatusErrored)
			},
		},
		{
			name: "successful Stage metadata update",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testObjName,
						Namespace: testProject,
					},
					Status: kargoapi.StageStatus{
						Metadata: testData,
					},
				},
			).WithStatusSubresource(&kargoapi.Stage{}).Build(),
			cfg: builtin.SetMetadataConfig{
				Updates: []builtin.Update{{
					Kind:   "Stage",
					Name:   testObjName,
					Values: testValueUpdates,
				}},
			},
			assertions: func(
				t *testing.T,
				res promotion.StepResult,
				c client.Client,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, res.Status, kargoapi.PromotionStepStatusSucceeded)

				stage := &kargoapi.Stage{}
				err = c.Get(
					context.Background(),
					types.NamespacedName{
						Name:      testObjName,
						Namespace: testProject,
					},
					stage,
				)
				require.NoError(t, err)

				dataBytes, ok := stage.Status.Metadata["string"]
				require.True(t, ok)
				var data any
				err = json.Unmarshal(dataBytes.Raw, &data)
				require.NoError(t, err)
				require.Equal(t, "bar", data)

				dataBytes, ok = stage.Status.Metadata["num"]
				require.True(t, ok)
				err = json.Unmarshal(dataBytes.Raw, &data)
				require.NoError(t, err)
				require.Equal(t, float64(43), data)

				dataBytes, ok = stage.Status.Metadata["complex"]
				require.True(t, ok)
				err = json.Unmarshal(dataBytes.Raw, &data)
				require.NoError(t, err)
				require.Equal(
					t,
					map[string]any{"updated": true},
					data,
				)

				dataBytes, ok = stage.Status.Metadata["new"]
				require.True(t, ok)
				err = json.Unmarshal(dataBytes.Raw, &data)
				require.NoError(t, err)
				require.Equal(t, "success!", data)
			},
		},
		{
			name:   "Freight not found",
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			cfg: builtin.SetMetadataConfig{
				Updates: []builtin.Update{{
					Kind: "Freight",
					Name: "nonexistent",
				}},
			},
			assertions: func(
				t *testing.T,
				res promotion.StepResult,
				_ client.Client,
				err error,
			) {
				require.ErrorContains(t, err, "error getting Freight")
				require.Equal(t, res.Status, kargoapi.PromotionStepStatusErrored)
			},
		},
		{
			name: "error patching Freight status",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:            testObjName,
						Namespace:       testProject,
						ResourceVersion: "invalid", // This will force the patch to fail
					},
				},
			).WithStatusSubresource(&kargoapi.Freight{}).Build(),
			cfg: builtin.SetMetadataConfig{
				Updates: []builtin.Update{{
					Kind:   "Freight",
					Name:   testObjName,
					Values: testValueUpdates,
				}},
			},
			assertions: func(
				t *testing.T,
				res promotion.StepResult,
				_ client.Client,
				err error,
			) {
				require.ErrorContains(t, err, "error patching status of Freight")
				require.ErrorContains(t, err, "can not convert resourceVersion")
				require.Equal(t, res.Status, kargoapi.PromotionStepStatusErrored)
			},
		},
		{
			name: "successful Freight metadata update",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testObjName,
						Namespace: testProject,
					},
				},
			).WithStatusSubresource(&kargoapi.Freight{}).Build(),
			cfg: builtin.SetMetadataConfig{
				Updates: []builtin.Update{{
					Kind:   "Freight",
					Name:   testObjName,
					Values: testValueUpdates,
				}},
			},
			assertions: func(
				t *testing.T,
				res promotion.StepResult,
				c client.Client,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, res.Status, kargoapi.PromotionStepStatusSucceeded)

				freight := &kargoapi.Freight{}
				err = c.Get(
					context.Background(),
					types.NamespacedName{
						Name:      testObjName,
						Namespace: testProject,
					},
					freight,
				)
				require.NoError(t, err)

				dataBytes, ok := freight.Status.Metadata["string"]
				require.True(t, ok)
				var data any
				err = json.Unmarshal(dataBytes.Raw, &data)
				require.NoError(t, err)
				require.Equal(t, "bar", data)

				dataBytes, ok = freight.Status.Metadata["num"]
				require.True(t, ok)
				err = json.Unmarshal(dataBytes.Raw, &data)
				require.NoError(t, err)
				require.Equal(t, float64(43), data)

				dataBytes, ok = freight.Status.Metadata["complex"]
				require.True(t, ok)
				err = json.Unmarshal(dataBytes.Raw, &data)
				require.NoError(t, err)
				require.Equal(
					t,
					map[string]any{"updated": true},
					data,
				)

				dataBytes, ok = freight.Status.Metadata["new"]
				require.True(t, ok)
				err = json.Unmarshal(dataBytes.Raw, &data)
				require.NoError(t, err)
				require.Equal(t, "success!", data)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setter := &metadataSetter{kargoClient: tt.client}
			stepCtx := &promotion.StepContext{Project: testProject}
			result, err := setter.run(context.Background(), stepCtx, tt.cfg)
			tt.assertions(t, result, tt.client, err)
		})
	}
}
