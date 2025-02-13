package directives

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_jsonParser_validate(t *testing.T) {
	tests := []struct {
		name          string
		config        map[string]any
		expectedError string
	}{
		{
			name: "path not specified (missing path field)",
			config: map[string]any{
				"outputs": []map[string]any{
					{"name": "output1", "fromExpression": "$.data"},
				},
			},
			expectedError: "(root): path is required",
		},
		{
			name: "path is empty string",
			config: map[string]any{
				"path": "",
				"outputs": []map[string]any{
					{"name": "output1", "fromExpression": "$.data"},
				},
			},
			expectedError: "path: String length must be greater than or equal to 1",
		},
		{
			name: "outputs field is missing",
			config: map[string]any{
				"path": "valid.json",
			},
			expectedError: "(root): outputs is required",
		},
		{
			name: "outputs field is an empty array",
			config: map[string]any{
				"path":    "valid.json",
				"outputs": []map[string]any{},
			},
			expectedError: "outputs: Array must have at least 1 items",
		},
		{
			name: "name is not specified",
			config: map[string]any{
				"path": "valid.json",
				"outputs": []map[string]any{
					{"fromExpression": "$.data"},
				},
			},
			expectedError: "outputs.0: name is required",
		},
		{
			name: "name is empty",
			config: map[string]any{
				"path": "valid.json",
				"outputs": []map[string]any{
					{"name": "", "fromExpression": "$.data"},
				},
			},
			expectedError: "name: String length must be greater than or equal to 1",
		},
		{
			name: "FromExpression is not specified",
			config: map[string]any{
				"path": "valid.json",
				"outputs": []map[string]any{
					{"name": "output1"},
				},
			},
			expectedError: "outputs.0: fromExpression is required",
		},
		{
			name: "FromExpression is empty",
			config: map[string]any{
				"path": "valid.json",
				"outputs": []map[string]any{
					{"name": "output1", "fromExpression": ""},
				},
			},
			expectedError: "fromExpression: String length must be greater than or equal to 1",
		},
		{
			name: "valid configuration (path + outputs present)",
			config: map[string]any{
				"path": "valid.json",
				"outputs": []map[string]any{
					{"name": "output1", "fromExpression": "$.data"},
				},
			},
			expectedError: "",
		},
	}

	r := newJSONParser()
	runner, ok := r.(*jsonParser)
	require.True(t, ok)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := runner.validate(tc.config)
			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			}
		})
	}
}

func Test_jsonParser_runPromotionStep(t *testing.T) {
	jp := &jsonParser{}

	testCases := []struct {
		name         string
		cfg          JSONParseConfig
		expectedErr  error
		expectedOut  map[string]any
		simulateFile bool
		fileContent  string
	}{
		{
			name: "successful run with outputs",
			cfg: JSONParseConfig{
				Path:    "test.json",
				Outputs: []JSONParse{{Name: "key", FromExpression: "key"}},
			},
			expectedOut:  map[string]any{"key": "value"},
			simulateFile: true,
			fileContent:  `{"key": "value"}`,
		},
		{
			name: "failed to extract outputs",
			cfg: JSONParseConfig{
				Path:    "test.json",
				Outputs: []JSONParse{{Name: "key", FromExpression: "nonexistent"}},
			},
			expectedErr:  errors.New("failed to extract output"),
			simulateFile: true,
			fileContent:  `{"key": "value"}`,
		},
		{
			name: "no outputs provided",
			cfg: JSONParseConfig{
				Path:    "test.json",
				Outputs: []JSONParse{},
			},
			expectedErr:  errors.New("invalid json-parse config: outputs is required"),
			simulateFile: true,
			fileContent:  `{"key": "value"}`,
		},
		{
			name:         "handle empty json file",
			cfg:          JSONParseConfig{Path: "test.json", Outputs: []JSONParse{{Name: "key", FromExpression: "key"}}},
			expectedErr:  errors.New("could not parse JSON"),
			simulateFile: true,
			fileContent:  ``,
		},
		{
			name: "fetch a string, numeric, boolean output from json file",
			cfg: JSONParseConfig{
				Path: "test.json",
				Outputs: []JSONParse{
					{Name: "stringKey", FromExpression: "stringKey"},
					{Name: "numKey", FromExpression: "numKey"},
					{Name: "boolKey", FromExpression: "boolKey"},
				},
			},
			expectedOut:  map[string]any{"stringKey": "hello", "numKey": 123.3, "boolKey": true},
			simulateFile: true,
			fileContent:  `{"stringKey": "hello", "numKey": 123.30, "boolKey": true}`,
		},
		{
			name:        "path is empty",
			cfg:         JSONParseConfig{Path: "", Outputs: []JSONParse{{Name: "key", FromExpression: "key"}}},
			expectedErr: errors.New("JSON file path cannot be empty"),
		},
		{
			name:        "path is a directory instead of a file",
			cfg:         JSONParseConfig{Path: "./testdir", Outputs: []JSONParse{{Name: "key", FromExpression: "key"}}},
			expectedErr: errors.New("could not read file"),
		},
		{
			name: "valid json but expression does not match any field",
			cfg: JSONParseConfig{
				Path:    "test.json",
				Outputs: []JSONParse{{Name: "key", FromExpression: "nonexistent"}},
			},
			expectedErr:  errors.New("failed to extract outputs"),
			simulateFile: true,
			fileContent:  `{"existingKey": "value"}`,
		},
		{
			name: "valid JSON, valid expressions, valid path",
			cfg: JSONParseConfig{
				Path:    "test.json",
				Outputs: []JSONParse{{Name: "key", FromExpression: "key"}}},
			expectedOut:  map[string]any{"key": "value"},
			simulateFile: true,
			fileContent:  `{"key": "value"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.simulateFile {
				_ = os.WriteFile(tc.cfg.Path, []byte(tc.fileContent), 0600)
				defer os.Remove(tc.cfg.Path)
			}

			result, err := jp.runPromotionStep(context.Background(), nil, tc.cfg)
			if tc.expectedErr != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedOut, result.Output)
			}
		})
	}
}

func Test_jsonParser_readAndParseJSON(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name         string
		fileContent  string
		fileExists   bool
		expectedErr  error
		expectedData map[string]any
	}{
		{
			name:        "file doesn't exist",
			fileExists:  false,
			expectedErr: errors.New("could not read file"),
		},
		{
			name:        "file is empty",
			fileExists:  true,
			fileContent: "",
			expectedErr: errors.New("could not parse JSON"),
		},
		{
			name:        "file contains invalid json",
			fileExists:  true,
			fileContent: "{invalid json}",
			expectedErr: errors.New("could not parse JSON"),
		},
		{
			name:         "valid json file with simple structure",
			fileExists:   true,
			fileContent:  `{"key":"value"}`,
			expectedData: map[string]any{"key": "value"},
		},
		{
			name:        "valid json file with deeply nested structure",
			fileExists:  true,
			fileContent: `{"level1": {"level2": {"level3": "deepValue"}}}`,
			expectedData: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{
						"level3": "deepValue",
					},
				},
			},
		},
	}

	jp := &jsonParser{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var filePath string
			if tt.fileExists {
				file, err := os.CreateTemp(tempDir, "test.json")
				if err != nil {
					t.Fatalf("failed to create temp file: %v", err)
				}
				defer os.Remove(file.Name())
				filePath = file.Name()

				if tt.fileContent != "" {
					if _, err := file.WriteString(tt.fileContent); err != nil {
						t.Fatalf("failed to write to temp file: %v", err)
					}
				}
			}

			data, err := jp.readAndParseJSON(filePath)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)

				expectedJSON, _ := json.Marshal(tt.expectedData)
				actualJSON, _ := json.Marshal(data)
				assert.JSONEq(t, string(expectedJSON), string(actualJSON))
			}
		})
	}
}

func Test_jsonParser_extractValues(t *testing.T) {
	jp := &jsonParser{}

	tests := []struct {
		name           string
		data           map[string]any
		outputs        []JSONParse
		expected       map[string]any
		expectedErrMsg string
	}{
		{
			name: "valid json, valid expression",
			data: map[string]any{"key": "value"},
			outputs: []JSONParse{
				{Name: "result", FromExpression: "key"},
			},
			expected: map[string]any{"result": "value"},
		},
		{
			name: "valid json, expression points to missing key",
			data: map[string]any{"key": "value"},
			outputs: []JSONParse{
				{Name: "result", FromExpression: "missingKey"},
			},
			expectedErrMsg: "failed to extract outputs",
		},
		{
			name: "expression evaluates to a nested object",
			data: map[string]any{"nested": map[string]any{"key": "value"}},
			outputs: []JSONParse{
				{Name: "result", FromExpression: "nested"},
			},
			expected: map[string]any{"result": map[string]any{"key": "value"}},
		},
		{
			name: "expression evaluates to an array",
			data: map[string]any{"array": []any{1, 2, 3}},
			outputs: []JSONParse{
				{Name: "result", FromExpression: "array"},
			},
			expected: map[string]any{"result": []any{1, 2, 3}},
		},
		{
			name: "expression evaluates to a string",
			data: map[string]any{"key": "value"},
			outputs: []JSONParse{
				{Name: "result", FromExpression: "key"},
			},
			expected: map[string]any{"result": "value"},
		},
		{
			name: "expression evaluates to an integer",
			data: map[string]any{"number": 42},
			outputs: []JSONParse{
				{Name: "result", FromExpression: "number"},
			},
			expected: map[string]any{"result": 42},
		},
		{
			name: "expression compilation error",
			data: map[string]any{"key": "value"},
			outputs: []JSONParse{
				{Name: "result", FromExpression: "(1 + 2"},
			},
			expectedErrMsg: "error compiling expression",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := jp.extractValues(tt.data, tt.outputs)

			if tt.expectedErrMsg != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
