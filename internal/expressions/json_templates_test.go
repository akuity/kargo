package expressions

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

func TestEvaluateJSONTemplate(t *testing.T) {
	// Context used for all test cases:
	type testStruct struct {
		AString    string
		AnInt      int
		AFloat     float64
		ABool      bool
		AStringMap map[string]string
		AnIntMap   map[string]int
		AStringArr []string
		AnIntArr   []int
		AStruct    *testStruct
	}

	testStringMap := map[string]string{"aString": "hello", "anotherString": "world"}
	testIntMap := map[string]int{"anInt": 42, "anotherInt": 43}
	testStringArr := []string{"one", "two", "three"}
	testIntArr := []int{1, 2, 3}
	testEnv := map[string]any{
		"aString":    "hello",
		"anInt":      42,
		"aFloat":     3.14,
		"aBool":      true,
		"aStringMap": testStringMap,
		"anIntMap":   testIntMap,
		"aStringArr": testStringArr,
		"anIntArr":   testIntArr,
		"aStruct": testStruct{
			AString: "hello",
			AnInt:   42,
		},
	}

	testCases := []struct {
		name         string
		jsonTemplate string
		yamlTemplate string
		assertions   func(t *testing.T, jsonOutput []byte, err error)
	}{
		{
			name: "template is not valid JSON",
			// This is invalid because the expression itself is not enclosed in
			// quotes. This would never be able to move over the wire.
			jsonTemplate: `{ "key": ${{ true }} }`,
			assertions: func(t *testing.T, _ []byte, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "input is not valid JSON")
			},
		},
		{
			name: "scalar values",
			jsonTemplate: `{
				"AString": "${{ aString }}",
				"AnInt": "${{ anInt }}",
				"AFloat": "${{ aFloat }}",
				"ABool": "${{ aBool }}"
			}`,
			assertions: func(t *testing.T, jsonOutput []byte, err error) {
				require.NoError(t, err)
				parsed := testStruct{}
				require.NoError(t, json.Unmarshal(jsonOutput, &parsed))
				require.Equal(t, testEnv["aString"], parsed.AString)
				require.Equal(t, testEnv["anInt"], parsed.AnInt)
				require.Equal(t, testEnv["aFloat"], parsed.AFloat)
				require.Equal(t, testEnv["aBool"], parsed.ABool)
			},
		},
		{
			name:         "mixing an expression with string literals",
			jsonTemplate: `{ "AString": "${{ aString }}, world!" }`,
			assertions: func(t *testing.T, jsonOutput []byte, err error) {
				require.NoError(t, err)
				parsed := testStruct{}
				require.NoError(t, json.Unmarshal(jsonOutput, &parsed))
				require.Equal(
					t,
					testEnv["aString"].(string)+", world!", // nolint: forcetypeassert
					parsed.AString,
				)
			},
		},
		{
			name:         "multiple expressions in one value",
			jsonTemplate: `{ "AString": "${{ aString }}, ${{ anInt }}!" }`,
			assertions: func(t *testing.T, jsonOutput []byte, err error) {
				require.NoError(t, err)
				parsed := testStruct{}
				require.NoError(t, json.Unmarshal(jsonOutput, &parsed))
				require.Equal(
					t,
					fmt.Sprintf("%s, %d!", testEnv["aString"], testEnv["anInt"]), // nolint: forcetypeassert
					parsed.AString,
				)
			},
		},
		{
			name: "maps",
			jsonTemplate: `{
				"AStringMap": "${{ aStringMap }}",
				"AnIntMap": "${{ anIntMap }}"
			}`,
			assertions: func(t *testing.T, jsonOutput []byte, err error) {
				require.NoError(t, err)
				parsed := testStruct{}
				require.NoError(t, json.Unmarshal(jsonOutput, &parsed))
				require.Equal(t, testStringMap, parsed.AStringMap)
				require.Equal(t, testIntMap, parsed.AnIntMap)
			},
		},
		{
			name: "arrays",
			jsonTemplate: `{
				"AStringArr": "${{ aStringArr }}",
				"AnIntArr": "${{ anIntArr }}"
			}`,
			assertions: func(t *testing.T, jsonOutput []byte, err error) {
				require.NoError(t, err)
				parsed := testStruct{}
				require.NoError(t, json.Unmarshal(jsonOutput, &parsed))
				require.Equal(t, testStringArr, parsed.AStringArr)
				require.Equal(t, testIntArr, parsed.AnIntArr)
			},
		},
		{
			name:         "structs",
			jsonTemplate: `{ "AStruct": "${{ aStruct }}" }`,
			assertions: func(t *testing.T, jsonOutput []byte, err error) {
				require.NoError(t, err)
				parsed := testStruct{}
				require.NoError(t, json.Unmarshal(jsonOutput, &parsed))
				require.NotNil(t, parsed.AStruct)
				require.Equal(t, testEnv["aStruct"], *parsed.AStruct)
			},
		},
		{
			name:         "null",
			jsonTemplate: `{ "AStruct": "${{ \"null\" }}" }`,
			assertions: func(t *testing.T, jsonOutput []byte, err error) {
				require.NoError(t, err)
				parsed := testStruct{
					AStruct: &testStruct{}, // This should get nilled out
				}
				require.NoError(t, json.Unmarshal(jsonOutput, &parsed))
				require.Nil(t, parsed.AStruct)
			},
		},
		{
			name: "test recursion",
			jsonTemplate: `{
				"AStringMap": { "key": "${{ aString }}" },
				"AnIntMap": { "key": "${{ anInt }}" },
				"AStringArr": [ "${{ aString }}"],
				"AnIntArr": [ "${{ anInt }}"]
			}`,
			assertions: func(t *testing.T, jsonOutput []byte, err error) {
				require.NoError(t, err)
				parsed := testStruct{}
				require.NoError(t, json.Unmarshal(jsonOutput, &parsed))
				require.Equal(
					t,
					// nolint: forcetypeassert
					testStruct{
						AStringMap: map[string]string{"key": testEnv["aString"].(string)},
						AnIntMap:   map[string]int{"key": testEnv["anInt"].(int)},
						AStringArr: []string{testEnv["aString"].(string)},
						AnIntArr:   []int{testEnv["anInt"].(int)},
					},
					parsed,
				)
			},
		},
		{
			name:         "quote function forces null to string",
			jsonTemplate: `{ "AString": "${{ quote(null) }}" }`,
			assertions: func(t *testing.T, jsonOutput []byte, err error) {
				require.NoError(t, err)
				parsed := testStruct{}
				require.NoError(t, json.Unmarshal(jsonOutput, &parsed))
				require.Equal(t, "null", parsed.AString)
			},
		},
		{
			name:         "quote function forces bool to string",
			jsonTemplate: `{ "AString": "${{ quote(true) }}" }`,
			assertions: func(t *testing.T, jsonOutput []byte, err error) {
				require.NoError(t, err)
				parsed := testStruct{}
				require.NoError(t, json.Unmarshal(jsonOutput, &parsed))
				require.Equal(t, "true", parsed.AString)
			},
		},
		{
			name:         "quote function forces number to string",
			jsonTemplate: `{ "AString": "${{ quote(42) }}" }`,
			assertions: func(t *testing.T, jsonOutput []byte, err error) {
				require.NoError(t, err)
				parsed := testStruct{}
				require.NoError(t, json.Unmarshal(jsonOutput, &parsed))
				require.Equal(t, "42", parsed.AString)
			},
		},
		{
			name:         "quote function forces object to string",
			jsonTemplate: `{ "AString": "${{ quote({ 'foo': 'bar' }) }}" }`,
			assertions: func(t *testing.T, jsonOutput []byte, err error) {
				require.NoError(t, err)
				parsed := testStruct{}
				require.NoError(t, json.Unmarshal(jsonOutput, &parsed))
				require.Equal(t, `{"foo":"bar"}`, parsed.AString)
			},
		},
		{
			name:         "quote function doesn't double quote string input",
			jsonTemplate: `{ "AString": "${{ quote('foo') }}" }`,
			assertions: func(t *testing.T, jsonOutput []byte, err error) {
				require.NoError(t, err)
				parsed := testStruct{}
				require.NoError(t, json.Unmarshal(jsonOutput, &parsed))
				require.Equal(t, "foo", parsed.AString)
			},
		},
		{
			name:         "unsafeQuote function does double quote string input",
			jsonTemplate: `{ "AString": "${{ unsafeQuote('foo') }}" }`,
			assertions: func(t *testing.T, jsonOutput []byte, err error) {
				require.NoError(t, err)
				parsed := testStruct{}
				require.NoError(t, json.Unmarshal(jsonOutput, &parsed))
				require.Equal(t, "\"foo\"", parsed.AString)
			},
		},
		{
			name: "a variety of tricky cases dealing with YAML and quotes",
			yamlTemplate: `
value1: {"foo": "bar"} # This is a JSON object
value2: '${{ quote({"foo": "bar"}) }}' # This is a string
value3: | # This is a string
  {"foo": "bar"}
value4: | # This is a JSON object
  ${{ {"foo": "bar"} }}
value5: | # This is a string
  ${{ quote({"foo": "bar"}) }}
value6: | # Make sure we're not tripped up by multiple newlines
  ${{ quote({"foo": "bar"}) }}
  
value7: 42 # This is a number
value8: "42" # This is a string
value9: '42'	# This is a string
value10: '${{ quote(42) }}' # This is a string
value11: | # This is a string
  42
value12: | # This is a number
  ${{ 42 }}
value13: | # This is a string
  ${{ quote(42) }}
`,
			assertions: func(t *testing.T, jsonOutput []byte, err error) {
				require.NoError(t, err)
				parsed := map[string]any{}
				require.NoError(t, json.Unmarshal(jsonOutput, &parsed))
				require.Equal(t, map[string]any{"foo": "bar"}, parsed["value1"])
				require.Equal(t, `{"foo":"bar"}`, parsed["value2"])
				require.Equal(t, "{\"foo\": \"bar\"}\n", parsed["value3"])
				require.Equal(t, map[string]any{"foo": "bar"}, parsed["value4"])
				require.Equal(t, "{\"foo\":\"bar\"}\n", parsed["value5"])
				require.Equal(t, "{\"foo\":\"bar\"}\n", parsed["value6"])
				require.Equal(t, float64(42), parsed["value7"])
				require.Equal(t, "42", parsed["value8"])
				require.Equal(t, "42", parsed["value9"])
				require.Equal(t, "42", parsed["value10"])
				require.Equal(t, "42\n", parsed["value11"])
				require.Equal(t, float64(42), parsed["value12"])
				require.Equal(t, "42\n", parsed["value13"])
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.yamlTemplate != "" {
				jsonBytes, err := yaml.YAMLToJSON([]byte(testCase.yamlTemplate))
				require.NoError(t, err)
				testCase.jsonTemplate = string(jsonBytes)
			}
			jsonOutput, err := EvaluateJSONTemplate([]byte(testCase.jsonTemplate), testEnv)
			testCase.assertions(t, jsonOutput, err)
		})
	}

	t.Run("quote function is forbidden", func(t *testing.T) {
		_, err := EvaluateJSONTemplate([]byte(`{}`), map[string]any{"quote": nil})
		require.ErrorContains(t, err, `"quote" is a forbidden key`)
	})
}
