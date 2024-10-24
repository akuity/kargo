package expressions

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
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
			name:         "quote function forces string result",
			jsonTemplate: `{ "AString": "${{ quote(anInt) }}" }`,
			assertions: func(t *testing.T, jsonOutput []byte, err error) {
				require.NoError(t, err)
				parsed := testStruct{}
				require.NoError(t, json.Unmarshal(jsonOutput, &parsed))
				require.Equal(t, fmt.Sprintf("%d", testEnv["anInt"]), parsed.AString)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			jsonOutput, err := EvaluateJSONTemplate([]byte(testCase.jsonTemplate), testEnv)
			testCase.assertions(t, jsonOutput, err)
		})
	}

	t.Run("quote function is forbidden", func(t *testing.T) {
		_, err := EvaluateJSONTemplate([]byte(`{}`), map[string]any{"quote": nil})
		require.Error(t, err)
		require.Contains(t, err.Error(), `"quote" is a forbidden key`)
	})
}
