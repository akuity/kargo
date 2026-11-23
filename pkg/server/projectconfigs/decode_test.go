package projectconfigs

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/yaml"
)

// TestDecodeEnforcesEveryCRDRequiredKey walks the ProjectConfig CRD schema,
// finds every "required" list under spec, and asserts Decode reports a
// Required error at the path the CRD reports when that key is absent. A new
// required field in the CRD fails this test until Decode covers it.
func TestDecodeEnforcesEveryCRDRequiredKey(t *testing.T) {
	crdBytes, err := os.ReadFile(parityCRDPath)
	require.NoError(t, err)
	crd := map[string]any{}
	require.NoError(t, yaml.Unmarshal(crdBytes, &crd))
	versions := crd["spec"].(map[string]any)["versions"].([]any) // nolint: forcetypeassert
	require.Len(t, versions, 1)
	root := versions[0].(map[string]any)["schema"].(map[string]any)["openAPIV3Schema"] // nolint: forcetypeassert
	specSchema := root.(map[string]any)["properties"].(map[string]any)["spec"]         // nolint: forcetypeassert

	var required []string
	collectRequired(specSchema.(map[string]any), nil, &required) // nolint: forcetypeassert
	// Sanity check that the walk found the schema's required keys.
	require.Greater(t, len(required), 20)

	for _, path := range required {
		t.Run(path, func(t *testing.T) {
			tokens := strings.Split(path, ".")
			parent, key := tokens[:len(tokens)-1], tokens[len(tokens)-1]
			obj := map[string]any{"spec": buildParent(parent)}
			_, errs := Decode(obj)
			want := expectedFieldPath(parent, key)
			var found bool
			for _, e := range errs {
				if e.Field == want && e.Type == field.ErrorTypeRequired {
					found = true
				}
			}
			require.Truef(t, found, "no Required error at %s; got %v", want, errs)
		})
	}
}

// collectRequired appends "a[].b.key" style paths, relative to spec, for
// every required key in the schema.
func collectRequired(schema map[string]any, path []string, out *[]string) {
	if req, ok := schema["required"].([]any); ok {
		for _, k := range req {
			*out = append(*out, strings.Join(append(slices.Clone(path), k.(string)), ".")) // nolint: forcetypeassert
		}
	}
	if props, ok := schema["properties"].(map[string]any); ok {
		for name, child := range props {
			childSchema := child.(map[string]any) // nolint: forcetypeassert
			if items, ok := childSchema["items"].(map[string]any); ok {
				collectRequired(items, append(slices.Clone(path), name+"[]"), out)
				continue
			}
			collectRequired(childSchema, append(slices.Clone(path), name), out)
		}
	}
}

// buildParent builds the minimal spec object that contains the parent of a
// required key, with one element in every list.
func buildParent(tokens []string) map[string]any {
	inner := map[string]any{}
	for i := len(tokens) - 1; i >= 0; i-- {
		name, isList := strings.CutSuffix(tokens[i], "[]")
		if isList {
			inner = map[string]any{name: []any{inner}}
		} else {
			inner = map[string]any{name: inner}
		}
	}
	return inner
}

func expectedFieldPath(parent []string, key string) string {
	p := field.NewPath("spec")
	for _, tok := range parent {
		name, isList := strings.CutSuffix(tok, "[]")
		p = p.Child(name)
		if isList {
			p = p.Index(0)
		}
	}
	return p.Child(key).String()
}

func TestDecode(t *testing.T) {
	testCases := []struct {
		name   string
		obj    string
		assert func(*testing.T, field.ErrorList)
	}{
		{
			name: "valid object decodes",
			obj: `{"metadata":{"name":"p","generation":3},"spec":{
				"promotionPolicies":[{"stageSelector":{"name":"a",
				"matchLabels":{"tier":"x"}}}]}}`,
			assert: func(t *testing.T, errs field.ErrorList) {
				require.Empty(t, errs)
			},
		},
		{
			name: "field names are case-sensitive",
			obj: `{"spec":{"webhookReceivers":[
				{"Name":"foo","github":{"secretRef":{"name":"s"}}}]}}`,
			assert: func(t *testing.T, errs field.ErrorList) {
				requirePaths(t, errs, "spec.webhookReceivers[0].name")
				require.Equal(t, field.ErrorTypeRequired, errs[0].Type)
			},
		},
		{
			name: "null counts as absent",
			obj: `{"spec":{"freightLinks":[
				{"title":null,"url":"https://example.com"}]}}`,
			assert: func(t *testing.T, errs field.ErrorList) {
				requirePaths(t, errs, "spec.freightLinks[0].title")
			},
		},
		{
			name: "empty value is present",
			obj: `{"spec":{"webhookReceivers":[{"name":"g","generic":{
				"secretRef":{},"actions":[{"action":"Refresh",
				"targetSelectionCriteria":[{"kind":"Warehouse",
				"indexSelector":{"matchIndices":[
				{"key":"subscribedURLs","operator":"","value":""}]}}]}]}}]}}`,
			assert: func(t *testing.T, errs field.ErrorList) {
				require.Empty(t, errs)
			},
		},
		{
			name: "explicit empty stage with stageSelector",
			obj: `{"spec":{"promotionPolicies":[
				{"stage":"","stageSelector":{"name":"a"}}]}}`,
			assert: func(t *testing.T, errs field.ErrorList) {
				requirePaths(
					t,
					errs,
					"spec.promotionPolicies[0].stage",
					"spec.promotionPolicies[0]",
				)
				require.Equal(t, policyOneOfMessage, errs[1].Detail)
			},
		},
		{
			name: "explicit empty stage alone",
			obj:  `{"spec":{"promotionPolicies":[{"stage":""}]}}`,
			assert: func(t *testing.T, errs field.ErrorList) {
				requirePaths(t, errs, "spec.promotionPolicies[0].stage")
			},
		},
		{
			name: "wrong type",
			obj:  `{"spec":{"promotionPolicies":"nope"}}`,
			assert: func(t *testing.T, errs field.ErrorList) {
				require.Len(t, errs, 1)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			obj := map[string]any{}
			require.NoError(t, yaml.Unmarshal([]byte(testCase.obj), &obj))
			_, errs := Decode(obj)
			testCase.assert(t, errs)
		})
	}
}

func TestDecodeConversion(t *testing.T) {
	obj := map[string]any{
		"metadata": map[string]any{"name": "p", "generation": float64(3)},
		"spec": map[string]any{
			"bogus": true,
			"promotionPolicies": []any{map[string]any{
				"stageSelector": map[string]any{
					"name":        "a",
					"matchLabels": map[string]any{"tier": "x"},
				},
			}},
			"webhookReceivers": []any{map[string]any{
				"name": "g",
				"generic": map[string]any{
					"secretRef": map[string]any{"name": "s"},
					"actions":   []any{},
				},
			}},
		},
	}
	cfg, errs := Decode(obj)
	require.Empty(t, errs)
	require.Equal(t, "p", cfg.Name)
	require.Equal(t, int64(3), cfg.Generation)
	sel := cfg.Spec.PromotionPolicies[0].StageSelector
	require.Equal(t, "a", sel.Name)
	require.Equal(t, map[string]string{"tier": "x"}, sel.MatchLabels)
	actions := cfg.Spec.WebhookReceivers[0].Generic.Actions
	require.NotNil(t, actions)
	require.Empty(t, actions)
}
