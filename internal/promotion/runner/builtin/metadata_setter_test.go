package builtin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
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

	r := newMetadataSetter(nil)
	runner, ok := r.(*metadataSetter)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

func Test_metadataSetter_run(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	_ = metav1.AddMetaToScheme(scheme)

	tests := []struct {
		name    string
		cfg     builtin.SetMetadataConfig
		setup   func(t *testing.T, client client.Client)
		verify  func(t *testing.T, client client.Client)
		wantErr bool
	}{
		{
			name: "unsupported kind",
			cfg: builtin.SetMetadataConfig{
				Updates: []builtin.Update{
					{
						Kind:   "UnsupportedKind",
						Name:   "test-resource",
						Values: map[string]any{"key": "value"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "resource not found",
			cfg: builtin.SetMetadataConfig{
				Updates: []builtin.Update{
					{
						Kind:   "Stage",
						Name:   "nonexistent-stage",
						Values: map[string]any{"key": "value"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "successful update of stage metadata",
			cfg: builtin.SetMetadataConfig{
				Updates: []builtin.Update{
					{
						Kind:   "Stage",
						Name:   "test-stage",
						Values: map[string]any{"key1": "value1", "key2": 42},
					},
				},
			},
			setup: func(t *testing.T, client client.Client) {
				stage := &kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-stage",
						Namespace: "test-project",
					},
				}
				require.NoError(t, client.Create(context.Background(), stage))
				s := &kargoapi.Stage{}
				err := client.Get(context.Background(), types.NamespacedName{
					Name:      "test-stage",
					Namespace: "test-project",
				}, s)
				require.NoError(t, err, "Stage not found after Create")
			},
			verify: func(t *testing.T, client client.Client) {
				stage := &kargoapi.Stage{}
				require.NoError(t, client.Get(
					context.Background(),
					types.NamespacedName{Name: "test-stage", Namespace: "test-project"},
					stage,
				))

				var value1 string
				exists, err := stage.Status.GetMetadata("key1", &value1)
				require.NoError(t, err)
				require.True(t, exists)
				require.Equal(t, "value1", value1)

				var value2 int
				exists, err = stage.Status.GetMetadata("key2", &value2)
				require.NoError(t, err)
				require.True(t, exists)
				require.Equal(t, 42, value2)
			},
		},
		{
			name: "successful update of freight metadata",
			cfg: builtin.SetMetadataConfig{
				Updates: []builtin.Update{
					{
						Kind:   "Freight",
						Name:   "test-freight",
						Values: map[string]any{"version": "1.0.0", "deployed": true},
					},
				},
			},
			setup: func(t *testing.T, client client.Client) {
				freight := &kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-freight",
						Namespace: "test-project",
					},
				}
				require.NoError(t, client.Create(context.Background(), freight))
				f := &kargoapi.Freight{}
				err := client.Get(context.Background(), types.NamespacedName{
					Name:      "test-freight",
					Namespace: "test-project",
				}, f)
				require.NoError(t, err, "Freight not found after Create")
			},
			verify: func(t *testing.T, client client.Client) {
				freight := &kargoapi.Freight{}
				require.NoError(t, client.Get(
					context.Background(),
					types.NamespacedName{Name: "test-freight", Namespace: "test-project"},
					freight,
				))

				var version string
				exists, err := freight.Status.GetMetadata("version", &version)
				require.NoError(t, err)
				require.True(t, exists)
				require.Equal(t, "1.0.0", version)

				var deployed bool
				exists, err = freight.Status.GetMetadata("deployed", &deployed)
				require.NoError(t, err)
				require.True(t, exists)
				require.True(t, deployed)
			},
		},
		{
			name: "multiple updates to same resource",
			cfg: builtin.SetMetadataConfig{
				Updates: []builtin.Update{
					{
						Kind:   "Stage",
						Name:   "test-stage",
						Values: map[string]any{"key1": "value1"},
					},
					{
						Kind:   "Stage",
						Name:   "test-stage",
						Values: map[string]any{"key2": "value2"},
					},
				},
			},
			setup: func(t *testing.T, client client.Client) {
				stage := &kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-stage",
						Namespace: "test-project",
					},
				}
				require.NoError(t, client.Create(context.Background(), stage))
				s := &kargoapi.Stage{}
				err := client.Get(context.Background(), types.NamespacedName{
					Name:      "test-stage",
					Namespace: "test-project",
				}, s)
				require.NoError(t, err, "Stage not found after Create")
			},
			verify: func(t *testing.T, client client.Client) {
				stage := &kargoapi.Stage{}
				require.NoError(t, client.Get(
					context.Background(),
					types.NamespacedName{Name: "test-stage", Namespace: "test-project"},
					stage,
				))

				var value1, value2 string
				exists, err := stage.Status.GetMetadata("key1", &value1)
				require.NoError(t, err)
				require.True(t, exists)
				require.Equal(t, "value1", value1)

				exists, err = stage.Status.GetMetadata("key2", &value2)
				require.NoError(t, err)
				require.True(t, exists)
				require.Equal(t, "value2", value2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&kargoapi.Stage{}, &kargoapi.Freight{}).
				Build()

			if tt.setup != nil {
				tt.setup(t, client)
			}

			setter := &metadataSetter{kargoClient: client}
			stepCtx := &promotion.StepContext{Project: "test-project"}
			result, err := setter.run(context.Background(), stepCtx, tt.cfg)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

			if tt.verify != nil {
				tt.verify(t, client)
			}
		})
	}
}
