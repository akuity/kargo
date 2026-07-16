package server

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	sigyaml "sigs.k8s.io/yaml"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/user"
)

func Test_splitYAML(t *testing.T) {
	// Test resources
	project1 := kargoapi.Project{
		TypeMeta:   metav1.TypeMeta{APIVersion: kargoapi.GroupVersion.String(), Kind: "Project"},
		ObjectMeta: metav1.ObjectMeta{Name: "project-1"},
	}
	project2 := kargoapi.Project{
		TypeMeta:   metav1.TypeMeta{APIVersion: kargoapi.GroupVersion.String(), Kind: "Project"},
		ObjectMeta: metav1.ObjectMeta{Name: "project-2"},
	}
	stage := kargoapi.Stage{
		TypeMeta:   metav1.TypeMeta{APIVersion: kargoapi.GroupVersion.String(), Kind: "Stage"},
		ObjectMeta: metav1.ObjectMeta{Name: "test-stage", Namespace: "my-project"},
	}
	warehouse := kargoapi.Warehouse{
		TypeMeta:   metav1.TypeMeta{APIVersion: kargoapi.GroupVersion.String(), Kind: "Warehouse"},
		ObjectMeta: metav1.ObjectMeta{Name: "test-warehouse", Namespace: "my-project"},
	}
	secret := corev1.Secret{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
		ObjectMeta: metav1.ObjectMeta{Name: "my-secret", Namespace: "test"},
		Type:       corev1.SecretTypeOpaque,
	}

	tests := []struct {
		name       string
		input      []byte
		assertions func(*testing.T, []unstructured.Unstructured, []unstructured.Unstructured, error)
	}{
		{
			name:  "empty input",
			input: []byte(""),
			assertions: func(t *testing.T, projects, other []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				assert.Empty(t, projects)
				assert.Empty(t, other)
			},
		},
		{
			name:  "whitespace only",
			input: []byte("   \n\t\n   "),
			assertions: func(t *testing.T, projects, other []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				assert.Empty(t, projects)
				assert.Empty(t, other)
			},
		},
		{
			name:  "null document",
			input: []byte("null"),
			assertions: func(t *testing.T, projects, other []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				assert.Empty(t, projects)
				assert.Empty(t, other)
			},
		},
		{
			name:  "single YAML Project",
			input: mustYAML(project1),
			assertions: func(t *testing.T, projects, other []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				require.Len(t, projects, 1)
				assert.Empty(t, other)
				assert.Equal(t, "project-1", projects[0].GetName())
			},
		},
		{
			name:  "single YAML non-Project resource",
			input: mustYAML(stage),
			assertions: func(t *testing.T, projects, other []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				assert.Empty(t, projects)
				require.Len(t, other, 1)
				assert.Equal(t, "Stage", other[0].GetKind())
				assert.Equal(t, "test-stage", other[0].GetName())
			},
		},
		{
			name:  "multiple YAML documents with ---",
			input: mustYAML(project1, project2),
			assertions: func(t *testing.T, projects, other []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				require.Len(t, projects, 2)
				assert.Empty(t, other)
				assert.Equal(t, "project-1", projects[0].GetName())
				assert.Equal(t, "project-2", projects[1].GetName())
			},
		},
		{
			name: "mixed YAML documents - Projects and other resources",
			input: mustYAML(
				kargoapi.Project{
					TypeMeta:   metav1.TypeMeta{APIVersion: kargoapi.GroupVersion.String(), Kind: "Project"},
					ObjectMeta: metav1.ObjectMeta{Name: "my-project"},
				},
				stage,
				warehouse,
			),
			assertions: func(t *testing.T, projects, other []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				require.Len(t, projects, 1)
				require.Len(t, other, 2)
				assert.Equal(t, "my-project", projects[0].GetName())
				assert.Equal(t, "Stage", other[0].GetKind())
				assert.Equal(t, "test-stage", other[0].GetName())
				assert.Equal(t, "Warehouse", other[1].GetKind())
				assert.Equal(t, "test-warehouse", other[1].GetName())
			},
		},
		{
			name:  "single JSON object - Project",
			input: mustJSON(project1),
			assertions: func(t *testing.T, projects, other []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				require.Len(t, projects, 1)
				assert.Empty(t, other)
				assert.Equal(t, "project-1", projects[0].GetName())
			},
		},
		{
			name:  "single JSON object - non-Project",
			input: mustJSON(stage),
			assertions: func(t *testing.T, projects, other []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				assert.Empty(t, projects)
				require.Len(t, other, 1)
				assert.Equal(t, "Stage", other[0].GetKind())
				assert.Equal(t, "test-stage", other[0].GetName())
			},
		},
		{
			name:  "multiple concatenated JSON objects",
			input: mustJSONConcat(project1, project2),
			assertions: func(t *testing.T, projects, other []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				require.Len(t, projects, 2)
				assert.Empty(t, other)
				assert.Equal(t, "project-1", projects[0].GetName())
				assert.Equal(t, "project-2", projects[1].GetName())
			},
		},
		{
			name:  "JSON array of Projects",
			input: mustJSONArray(project1, project2),
			assertions: func(t *testing.T, projects, other []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				require.Len(t, projects, 2)
				assert.Empty(t, other)
				assert.Equal(t, "project-1", projects[0].GetName())
				assert.Equal(t, "project-2", projects[1].GetName())
			},
		},
		{
			name: "JSON array with mixed resource types",
			input: mustJSONArray(
				kargoapi.Project{
					TypeMeta:   metav1.TypeMeta{APIVersion: kargoapi.GroupVersion.String(), Kind: "Project"},
					ObjectMeta: metav1.ObjectMeta{Name: "mixed-project"},
				},
				kargoapi.Stage{
					TypeMeta:   metav1.TypeMeta{APIVersion: kargoapi.GroupVersion.String(), Kind: "Stage"},
					ObjectMeta: metav1.ObjectMeta{Name: "mixed-stage", Namespace: "mixed-project"},
				},
				kargoapi.Warehouse{
					TypeMeta:   metav1.TypeMeta{APIVersion: kargoapi.GroupVersion.String(), Kind: "Warehouse"},
					ObjectMeta: metav1.ObjectMeta{Name: "mixed-warehouse", Namespace: "mixed-project"},
				},
			),
			assertions: func(t *testing.T, projects, other []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				require.Len(t, projects, 1)
				require.Len(t, other, 2)
				assert.Equal(t, "mixed-project", projects[0].GetName())
				assert.Equal(t, "Stage", other[0].GetKind())
				assert.Equal(t, "mixed-stage", other[0].GetName())
				assert.Equal(t, "Warehouse", other[1].GetKind())
				assert.Equal(t, "mixed-warehouse", other[1].GetName())
			},
		},
		{
			name:  "empty JSON array",
			input: []byte("[]"),
			assertions: func(t *testing.T, projects, other []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				assert.Empty(t, projects)
				assert.Empty(t, other)
			},
		},
		{
			name:  "invalid YAML",
			input: []byte("invalid: yaml: content:"),
			assertions: func(t *testing.T, _, _ []unstructured.Unstructured, err error) {
				require.Error(t, err)
			},
		},
		{
			name:  "invalid JSON",
			input: []byte(`{"invalid": json}`),
			assertions: func(t *testing.T, _, _ []unstructured.Unstructured, err error) {
				require.Error(t, err)
			},
		},
		{
			name:  "invalid JSON array",
			input: []byte(`[{"valid": "json"}, invalid]`),
			assertions: func(t *testing.T, _, _ []unstructured.Unstructured, err error) {
				require.Error(t, err)
			},
		},
		{
			name:  "core Kubernetes resource (Secret)",
			input: mustYAML(secret),
			assertions: func(t *testing.T, projects, other []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				assert.Empty(t, projects)
				require.Len(t, other, 1)
				assert.Equal(t, "Secret", other[0].GetKind())
				assert.Equal(t, "my-secret", other[0].GetName())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projects, otherResources, err := splitYAML(tt.input)
			tt.assertions(t, projects, otherResources, err)
		})
	}
}

// mustYAML marshals objects to YAML. Multiple objects are separated by "---".
func mustYAML(objs ...any) []byte {
	var docs []string
	for _, obj := range objs {
		b, err := sigyaml.Marshal(obj)
		if err != nil {
			panic(err)
		}
		docs = append(docs, string(b))
	}
	return []byte(strings.Join(docs, "---\n"))
}

// mustJSON marshals a single object to JSON.
func mustJSON(obj any) []byte {
	b, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}
	return b
}

// mustJSONConcat marshals multiple objects to newline-separated JSON.
func mustJSONConcat(objs ...any) []byte {
	var lines []string
	for _, obj := range objs {
		b, err := json.Marshal(obj)
		if err != nil {
			panic(err)
		}
		lines = append(lines, string(b))
	}
	return []byte(strings.Join(lines, "\n"))
}

// mustJSONArray marshals multiple objects as a JSON array.
func mustJSONArray(objs ...any) []byte {
	b, err := json.Marshal(objs)
	if err != nil {
		panic(err)
	}
	return b
}

func Test_annotateResourceWithCreator(t *testing.T) {
	newObj := func(kind string, annotations map[string]any) *unstructured.Unstructured {
		metadata := map[string]any{"name": "fake-resource"}
		if annotations != nil {
			metadata["annotations"] = annotations
		}
		return &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": kargoapi.GroupVersion.String(),
				"kind":       kind,
				"metadata":   metadata,
			},
		}
	}

	testCases := []struct {
		name     string
		obj      *unstructured.Unstructured
		userInfo *user.Info
		assert   func(*testing.T, *unstructured.Unstructured)
	}{
		{
			name: "nil object does not panic",
		},
		{
			name:     "Project is annotated",
			obj:      newObj("Project", nil),
			userInfo: &user.Info{IsAdmin: true},
			assert: func(t *testing.T, obj *unstructured.Unstructured) {
				require.Equal(
					t,
					kargoapi.EventActorAdmin,
					obj.GetAnnotations()[kargoapi.AnnotationKeyCreateActor],
				)
			},
		},
		{
			name:     "Promotion is annotated",
			obj:      newObj("Promotion", nil),
			userInfo: &user.Info{IsAdmin: true},
			assert: func(t *testing.T, obj *unstructured.Unstructured) {
				require.Equal(
					t,
					kargoapi.EventActorAdmin,
					obj.GetAnnotations()[kargoapi.AnnotationKeyCreateActor],
				)
			},
		},
		{
			name: "caller-supplied actor on a Promotion is overwritten",
			obj: newObj("Promotion", map[string]any{
				kargoapi.AnnotationKeyCreateActor: "controller:forged",
			}),
			userInfo: &user.Info{IsAdmin: true},
			assert: func(t *testing.T, obj *unstructured.Unstructured) {
				require.Equal(
					t,
					kargoapi.EventActorAdmin,
					obj.GetAnnotations()[kargoapi.AnnotationKeyCreateActor],
				)
			},
		},
		{
			name:     "other kinds are not annotated",
			obj:      newObj("Stage", nil),
			userInfo: &user.Info{IsAdmin: true},
			assert: func(t *testing.T, obj *unstructured.Unstructured) {
				require.NotContains(
					t,
					obj.GetAnnotations(),
					kargoapi.AnnotationKeyCreateActor,
				)
			},
		},
		{
			name: "no user info in context leaves object untouched",
			obj:  newObj("Promotion", nil),
			assert: func(t *testing.T, obj *unstructured.Unstructured) {
				require.NotContains(
					t,
					obj.GetAnnotations(),
					kargoapi.AnnotationKeyCreateActor,
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()
			if testCase.userInfo != nil {
				ctx = user.ContextWithInfo(ctx, *testCase.userInfo)
			}
			annotateResourceWithCreator(ctx, testCase.obj)
			if testCase.assert != nil {
				testCase.assert(t, testCase.obj)
			}
		})
	}
}

func Test_server_authorizeResourceCreate(t *testing.T) {
	const project = "fake-project"
	const stage = "fake-stage"

	promotion := func() *unstructured.Unstructured {
		return &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": kargoapi.GroupVersion.String(),
				"kind":       "Promotion",
				"metadata": map[string]any{
					"name":      "fake-promotion",
					"namespace": project,
				},
				"spec": map[string]any{
					"stage":   stage,
					"freight": "fake-freight",
				},
			},
		}
	}

	testCases := []struct {
		name        string
		obj         *unstructured.Unstructured
		authorizeFn func(
			context.Context,
			string,
			schema.GroupVersionResource,
			string,
			types.NamespacedName,
		) error
		assert func(*testing.T, error, bool)
	}{
		{
			name: "non-Promotion resource is not subject to the promote check",
			obj: &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": kargoapi.GroupVersion.String(),
					"kind":       "Stage",
					"metadata":   map[string]any{"name": stage, "namespace": project},
				},
			},
			assert: func(t *testing.T, err error, called bool) {
				require.NoError(t, err)
				require.False(t, called, "authorizeFn must not be called for non-Promotions")
			},
		},
		{
			name: "Promotion without a target Stage is left to validation",
			obj: &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": kargoapi.GroupVersion.String(),
					"kind":       "Promotion",
					"metadata":   map[string]any{"name": "fake-promotion", "namespace": project},
					"spec":       map[string]any{"freight": "fake-freight"},
				},
			},
			assert: func(t *testing.T, err error, called bool) {
				require.NoError(t, err)
				require.False(t, called, "authorizeFn must not be called without a target Stage")
			},
		},
		{
			name: "Promotion checks the promote verb on the target Stage",
			obj:  promotion(),
			authorizeFn: func(
				_ context.Context,
				verb string,
				gvr schema.GroupVersionResource,
				_ string,
				key types.NamespacedName,
			) error {
				require.Equal(t, "promote", verb)
				require.Equal(t, kargoapi.GroupVersion.WithResource("stages"), gvr)
				require.Equal(t, project, key.Namespace)
				require.Equal(t, stage, key.Name)
				return nil
			},
			assert: func(t *testing.T, err error, called bool) {
				require.NoError(t, err)
				require.True(t, called)
			},
		},
		{
			name: "denied promote authorization is surfaced",
			obj:  promotion(),
			authorizeFn: func(
				context.Context,
				string,
				schema.GroupVersionResource,
				string,
				types.NamespacedName,
			) error {
				return errors.New("not permitted to promote")
			},
			assert: func(t *testing.T, err error, called bool) {
				require.ErrorContains(t, err, "not permitted to promote")
				require.True(t, called)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var called bool
			s := &server{
				authorizeFn: func(
					ctx context.Context,
					verb string,
					gvr schema.GroupVersionResource,
					subresource string,
					key types.NamespacedName,
				) error {
					called = true
					if testCase.authorizeFn != nil {
						return testCase.authorizeFn(ctx, verb, gvr, subresource, key)
					}
					return nil
				},
			}
			err := s.authorizeResourceCreate(context.Background(), testCase.obj)
			testCase.assert(t, err, called)
		})
	}
}
