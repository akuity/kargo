package projectconfigs

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	structuralschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/schema/cel"
	structuraldefaulting "k8s.io/apiextensions-apiserver/pkg/apiserver/schema/defaulting"
	structurallisttype "k8s.io/apiextensions-apiserver/pkg/apiserver/schema/listtype"
	structuralpruning "k8s.io/apiextensions-apiserver/pkg/apiserver/schema/pruning"
	apiextensionsvalidation "k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	apimachineryvalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utiljson "k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/validation/field"
	celconfig "k8s.io/apiserver/pkg/apis/cel"
	"sigs.k8s.io/yaml"
)

const (
	parityProject  = "kargo-demo"
	parityCRDPath  = "../../../charts/kargo/resources/crds/kargo.akuity.io_projectconfigs.yaml"
	parityFixtures = "testdata/parity"
)

// TestValidateParity runs every fixture under testdata/parity through the new
// validator and through a model of the Kubernetes behavior it replaces: the
// API server's generic metadata validation and the frozen CRD schema
// (pruning, defaulting, OpenAPI, list-type and CEL validation), followed by
// the admission webhook. Both must agree with the
// fixture's "# want:" header.
//
// The admission webhook is gone. Its verdict on each fixture was checked
// against the "# want:" header while it still existed, so the header now
// stands in for it: a fixture the CRD schema accepts is accepted by the
// model exactly when the header says it is valid.
//
// The model covers create only, as the name rule and the Project check are
// create-only. The namespace in the fixture's metadata is compared with the
// request namespace, as the Kubernetes API server does.
func TestValidateParity(t *testing.T) {
	today := newTodayModel(t)

	paths, err := filepath.Glob(filepath.Join(parityFixtures, "*.yaml"))
	require.NoError(t, err)
	require.NotEmpty(t, paths)

	for _, path := range paths {
		t.Run(strings.TrimSuffix(filepath.Base(path), ".yaml"), func(t *testing.T) {
			raw, err := os.ReadFile(path)
			require.NoError(t, err)
			want := readWant(t, raw)

			// Decode the fixture the way the API server will: a
			// case-preserving JSON map, then Decode, then Validate.
			jsonBytes, err := yaml.YAMLToJSON(raw)
			require.NoError(t, err)
			obj := map[string]any{}
			require.NoError(t, json.Unmarshal(jsonBytes, &obj))
			cfg, errs := Decode(obj)
			if len(errs) == 0 {
				errs = Validate(t.Context(), parityProject, cfg, true)
			}
			got := len(errs) == 0

			// The recorded want stands in for the admission webhook.
			todayAccepts := today.crdSchemaAccepts(t, raw) && want

			t.Logf(
				"want valid=%t, new validator valid=%t, today valid=%t",
				want, got, todayAccepts,
			)
			for _, e := range errs {
				t.Logf("new validator: %s", e)
			}
			require.Equal(t, want, got, "new validator disagrees with fixture")
			require.Equal(t, want, todayAccepts, "today's behavior disagrees with fixture")
		})
	}
}

// readWant parses the "# want: valid|invalid" header on a fixture's first
// line.
func readWant(t *testing.T, raw []byte) bool {
	t.Helper()
	line, _, _ := bufio.NewReader(bytes.NewReader(raw)).ReadLine()
	switch strings.TrimSpace(strings.TrimPrefix(string(line), "# want:")) {
	case "valid":
		return true
	case "invalid":
		return false
	}
	t.Fatalf("fixture must start with %q or %q", "# want: valid", "# want: invalid")
	return false
}

// todayModel evaluates a ProjectConfig against the CRD schema as the
// Kubernetes API server applies it.
type todayModel struct {
	structural   *structuralschema.Structural
	schema       apiextensionsvalidation.SchemaValidator
	celValidator *cel.Validator
}

func newTodayModel(t *testing.T) *todayModel {
	t.Helper()

	crdBytes, err := os.ReadFile(parityCRDPath)
	require.NoError(t, err)
	crd := &apiextensionsv1.CustomResourceDefinition{}
	require.NoError(t, yaml.Unmarshal(crdBytes, crd))
	require.Len(t, crd.Spec.Versions, 1)

	internal := &apiextensions.CustomResourceValidation{}
	require.NoError(
		t,
		apiextensionsv1.Convert_v1_CustomResourceValidation_To_apiextensions_CustomResourceValidation(
			crd.Spec.Versions[0].Schema,
			internal,
			nil,
		),
	)
	structural, err := structuralschema.NewStructural(internal.OpenAPIV3Schema)
	require.NoError(t, err)
	schemaValidator, _, err := apiextensionsvalidation.NewSchemaValidator(
		internal.OpenAPIV3Schema,
	)
	require.NoError(t, err)

	return &todayModel{
		structural:   structural,
		schema:       schemaValidator,
		celValidator: cel.NewValidator(structural, true, celconfig.PerCallLimit),
	}
}

// toUnstructured decodes a fixture into the generic form the API server
// works with, then applies the API server's request-namespace handling.
func (m *todayModel) toUnstructured(t *testing.T, raw []byte) (map[string]any, bool) {
	t.Helper()
	jsonBytes, err := yaml.YAMLToJSON(raw)
	require.NoError(t, err)
	obj := map[string]any{}
	require.NoError(t, utiljson.Unmarshal(jsonBytes, &obj))

	meta, _ := obj["metadata"].(map[string]any)
	if meta == nil {
		meta = map[string]any{}
		obj["metadata"] = meta
	}
	// The API server rejects a body whose namespace differs from the request
	// namespace, and fills it in when absent.
	if ns, _ := meta["namespace"].(string); ns != "" && ns != parityProject {
		return nil, false
	}
	meta["namespace"] = parityProject

	structuralpruning.Prune(obj, m.structural, true)
	structuraldefaulting.Default(obj, m.structural)
	return obj, true
}

// crdSchemaAccepts mirrors customResourceStrategy.Validate for a create.
func (m *todayModel) crdSchemaAccepts(t *testing.T, raw []byte) bool {
	t.Helper()
	obj, ok := m.toUnstructured(t, raw)
	if !ok {
		t.Log("today: namespace does not match the request namespace")
		return false
	}
	// The API server validates the generic metadata (label and annotation
	// syntax and size among it) before the CRD schema.
	errs := apimachineryvalidation.ValidateObjectMetaAccessor(
		&unstructured.Unstructured{Object: obj},
		true,
		apimachineryvalidation.NameIsDNSSubdomain,
		field.NewPath("metadata"),
	)
	errs = append(
		errs,
		apiextensionsvalidation.ValidateCustomResource(nil, obj, m.schema)...,
	)
	errs = append(
		errs,
		structurallisttype.ValidateListSetsAndMaps(nil, m.structural, obj)...,
	)
	celErrs, _ := m.celValidator.Validate(
		t.Context(),
		nil,
		m.structural,
		obj,
		nil,
		celconfig.RuntimeCELCostBudget,
	)
	errs = append(errs, celErrs...)
	logErrs(t, "today (metadata and CRD schema)", errs)
	return len(errs) == 0
}

func logErrs(t *testing.T, prefix string, errs field.ErrorList) {
	t.Helper()
	for _, e := range errs {
		t.Logf("%s: %s", prefix, e)
	}
}
