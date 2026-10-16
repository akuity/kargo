package builtin

import (
	"bytes"
	"context"
	"errors"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_httpRequester_convert(t *testing.T) {
	tests := []validationTestCase{
		{
			name:   "url not specified",
			config: promotion.Config{},
			expectedProblems: []string{
				"(root): url is required",
			},
		},
		{
			name: "url is empty string",
			config: promotion.Config{
				"url": "",
			},
			expectedProblems: []string{
				"url: String length must be greater than or equal to 1",
			},
		},
		{
			name: "invalid method",
			config: promotion.Config{
				"method": "invalid",
			},
			expectedProblems: []string{
				"method: Does not match pattern",
			},
		},
		{
			name: "header name not specified",
			config: promotion.Config{
				"headers": []promotion.Config{{}},
			},
			expectedProblems: []string{
				"headers.0: name is required",
			},
		},
		{
			name: "header name is empty string",
			config: promotion.Config{
				"headers": []promotion.Config{{
					"name": "",
				}},
			},
			expectedProblems: []string{
				"headers.0.name: String length must be greater than or equal to 1",
			},
		},
		{
			name: "header value not specified",
			config: promotion.Config{
				"headers": []promotion.Config{{}},
			},
			expectedProblems: []string{
				"headers.0: value is required",
			},
		},
		{
			name: "header value is empty string",
			config: promotion.Config{
				"headers": []promotion.Config{{
					"value": "",
				}},
			},
			expectedProblems: []string{
				"headers.0.value: String length must be greater than or equal to 1",
			},
		},
		{
			name: "query param name not specified",
			config: promotion.Config{
				"queryParams": []promotion.Config{{}},
			},
			expectedProblems: []string{
				"queryParams.0: name is required",
			},
		},
		{
			name: "query param name is empty string",
			config: promotion.Config{
				"queryParams": []promotion.Config{{
					"name": "",
				}},
			},
			expectedProblems: []string{
				"queryParams.0.name: String length must be greater than or equal to 1",
			},
		},
		{
			name: "query param value not specified",
			config: promotion.Config{
				"queryParams": []promotion.Config{{}},
			},
			expectedProblems: []string{
				"queryParams.0: value is required",
			},
		},
		{
			name: "query param value is empty string",
			config: promotion.Config{
				"queryParams": []promotion.Config{{
					"value": "",
				}},
			},
			expectedProblems: []string{
				"queryParams.0.value: String length must be greater than or equal to 1",
			},
		},
		{
			name: "invalid timeout",
			config: promotion.Config{
				"timeout": "invalid",
			},
			expectedProblems: []string{
				"timeout: Does not match pattern",
			},
		},
		{
			name: "outPath is empty string",
			config: promotion.Config{
				"outPath": "",
			},
			expectedProblems: []string{
				"outPath: String length must be greater than or equal to 1",
			},
		},
		{
			name: "invalid pollInterval",
			config: promotion.Config{
				"pollInterval": "invalid",
			},
			expectedProblems: []string{
				"pollInterval: Does not match pattern",
			},
		},
		{
			name: "invalid response content type",
			config: promotion.Config{
				"responseContentType": "invalid",
			},
			expectedProblems: []string{
				"responseContentType: Does not match pattern",
			},
		},
		{
			name: "output name not specified",
			config: promotion.Config{
				"outputs": []promotion.Config{{}},
			},
			expectedProblems: []string{
				"outputs.0: name is required",
			},
		},
		{
			name: "output name is empty string",
			config: promotion.Config{
				"outputs": []promotion.Config{{
					"name": "",
				}},
			},
			expectedProblems: []string{
				"outputs.0.name: String length must be greater than or equal to 1",
			},
		},
		{
			name: "output fromExpression not specified",
			config: promotion.Config{
				"outputs": []promotion.Config{{}},
			},
			expectedProblems: []string{
				"outputs.0: fromExpression is required",
			},
		},
		{
			name: "output fromExpression is empty string",
			config: promotion.Config{
				"outputs": []promotion.Config{{
					"fromExpression": "",
				}},
			},
			expectedProblems: []string{
				"outputs.0.fromExpression: String length must be greater than or equal to 1",
			},
		},
		{
			name: "proxy url has no scheme",
			config: promotion.Config{
				"proxy": "proxy.example.com:3000",
			},
			expectedProblems: []string{
				"proxy: Does not match pattern",
			},
		},
		{
			name: "body and bodyFromFile are mutually exclusive",
			config: promotion.Config{
				"body":         "{}",
				"bodyFromFile": "payload.json",
			},
			expectedProblems: []string{
				"Must validate one and only one schema (oneOf)",
			},
		},
		{
			name: "valid kitchen sink",
			config: promotion.Config{
				"method": "GET",
				"url":    "https://example.com",
				"headers": []promotion.Config{{
					"name":  "Accept",
					"value": "application/json",
				}},
				"queryParams": []promotion.Config{{
					"name":  "foo",
					"value": "bar",
				}},
				"insecureSkipTLSVerify": true,
				"timeout":               "30s",
				"pollInterval":          "20s",
				"outPath":               "downloads/report.json",
				"allowOverwrite":        true,
				"successExpression":     "response.status == 200",
				"failureExpression":     "response.status == 404",
				"proxy":                 "https://proxy.example.com:3000",
				"errorExpression":       "response.body?.error ?? response.body?.errorMessage",
				"outputs": []promotion.Config{
					{
						"name":           "fact1",
						"fromExpression": "response.body.facts[0]",
					},
					{
						"name":           "fact2",
						"fromExpression": "response.body.facts[1]",
					},
				},
			},
		},
	}

	r := newHTTPRequester(promotion.StepRunnerCapabilities{})
	runner, ok := r.(*httpRequester)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

func Test_httpRequester_run(t *testing.T) {
	testCases := []struct {
		name       string
		cfg        builtin.HTTPConfig
		handler    http.HandlerFunc
		assertions func(*testing.T, promotion.StepResult, error)
	}{
		{
			name:    "success and not failed; no body",
			handler: func(_ http.ResponseWriter, _ *http.Request) {},
			cfg: builtin.HTTPConfig{
				SuccessExpression: "true",
				Outputs: []builtin.HTTPOutput{
					{
						Name:           "status",
						FromExpression: "response.status",
					},
					{
						Name:           "theMeaningOfLife",
						FromExpression: "response.body.theMeaningOfLife",
					},
				},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
				require.Equal(
					t,
					map[string]any{
						"status":           int64(http.StatusOK),
						"theMeaningOfLife": nil,
					},
					res.Output,
				)
			},
		},
		{
			name: "unknown content-type with invalid JSON leaves body empty",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				// Unknown content-type with invalid JSON should leave body as empty map
				w.Header().Set("Content-Type", "text/html")
				_, err := w.Write([]byte(`<html>hello</html>`))
				require.NoError(t, err)
			},
			cfg: builtin.HTTPConfig{
				SuccessExpression: "true",
				Outputs: []builtin.HTTPOutput{
					{
						Name:           "status",
						FromExpression: "response.status",
					},
				},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
				require.Equal(t, map[string]any{"status": int64(http.StatusOK)}, res.Output)
			},
		},
		{
			name: "success and not failed with json body",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set(contentTypeHeader, contentTypeJSON)
				_, err := w.Write([]byte(`{"theMeaningOfLife": 42}`))
				require.NoError(t, err)
			},
			cfg: builtin.HTTPConfig{
				SuccessExpression: "true",
				Outputs: []builtin.HTTPOutput{
					{
						Name:           "status",
						FromExpression: "response.status",
					},
					{
						Name:           "theMeaningOfLife",
						FromExpression: "response.body.theMeaningOfLife",
					},
				},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
				require.Equal(
					t,
					map[string]any{
						"status":           int64(http.StatusOK),
						"theMeaningOfLife": float64(42),
					},
					res.Output,
				)
			},
		},
		{
			name: "success and not failed with json body and response is array",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set(contentTypeHeader, contentTypeJSON)
				_, err := w.Write([]byte(`[{"theMeaningOfLife": 42}]`))
				require.NoError(t, err)
			},
			cfg: builtin.HTTPConfig{
				SuccessExpression: "true",
				Outputs: []builtin.HTTPOutput{
					{
						Name:           "status",
						FromExpression: "response.status",
					},
					{
						Name:           "theMeaningOfLife",
						FromExpression: "response.body[0].theMeaningOfLife",
					},
				},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
				require.Equal(
					t,
					map[string]any{
						"status":           int64(http.StatusOK),
						"theMeaningOfLife": float64(42),
					},
					res.Output,
				)
			},
		},
		{
			name: "failed and not success",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			cfg: builtin.HTTPConfig{
				FailureExpression: "response.status == 404",
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.ErrorContains(t, err, "HTTP (404) response met failure criteria")
				require.True(t, promotion.IsTerminal(err))
				require.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)
			},
		},
		{
			name: "failed with errorExpression",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set(contentTypeHeader, contentTypeJSON)
				w.WriteHeader(http.StatusNotFound)
				_, err := w.Write([]byte(`{"error": "resource not found"}`))
				require.NoError(t, err)
			},
			cfg: builtin.HTTPConfig{
				FailureExpression: "response.status == 404",
				ErrorExpression:   "response.body.error",
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.ErrorContains(
					t,
					err,
					`HTTP (404) response met failure criteria: "resource not found"`,
				)
				require.True(t, promotion.IsTerminal(err))
				require.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)
			},
		},
		{
			name: "failed with errorExpression null coalescing",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set(contentTypeHeader, contentTypeJSON)
				w.WriteHeader(http.StatusNotFound)
				_, err := w.Write([]byte(`{"errorMessage": "something went wrong"}`))
				require.NoError(t, err)
			},
			cfg: builtin.HTTPConfig{
				FailureExpression: "response.status == 404",
				ErrorExpression:   "response.body?.error ?? response.body?.errorMessage",
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.ErrorContains(
					t,
					err,
					`HTTP (404) response met failure criteria: "something went wrong"`,
				)
				require.True(t, promotion.IsTerminal(err))
				require.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)
			},
		},
		{
			name: "failed with errorExpression evaluating to nil",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set(contentTypeHeader, contentTypeJSON)
				w.WriteHeader(http.StatusNotFound)
				_, err := w.Write([]byte(`{}`))
				require.NoError(t, err)
			},
			cfg: builtin.HTTPConfig{
				FailureExpression: "response.status == 404",
				ErrorExpression:   "response.body?.error ?? response.body?.errorMessage",
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.ErrorContains(t, err, "HTTP (404) response met failure criteria")
				require.NotContains(t, err.Error(), "HTTP (404) response met failure criteria:")
				require.True(t, promotion.IsTerminal(err))
				require.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)
			},
		},
		{
			name:    "failed with errorExpression compile error",
			handler: func(_ http.ResponseWriter, _ *http.Request) {},
			cfg: builtin.HTTPConfig{
				FailureExpression: "true",
				ErrorExpression:   "(1 + 2",
			},
			// A compile error is a misconfiguration, surfaced terminally and as
			// an errored step, consistent with the success/failure expressions.
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.ErrorContains(t, err, `error compiling error expression "(1 + 2"`)
				require.True(t, promotion.IsTerminal(err))
				require.Equal(t, kargoapi.PromotionStepStatusErrored, res.Status)
			},
		},
		{
			name: "failed with errorExpression non-string result",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set(contentTypeHeader, contentTypeJSON)
				w.WriteHeader(http.StatusNotFound)
				_, err := w.Write([]byte(`{"error": 42}`))
				require.NoError(t, err)
			},
			cfg: builtin.HTTPConfig{
				FailureExpression: "response.status == 404",
				ErrorExpression:   "response.body.error",
			},
			// A non-string result falls back to the default message.
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.ErrorContains(t, err, "HTTP (404) response met failure criteria")
				require.NotContains(t, err.Error(), "HTTP (404) response met failure criteria:")
				require.True(t, promotion.IsTerminal(err))
				require.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)
			},
		},
		{
			name:    "failed with errorExpression eval error",
			handler: func(_ http.ResponseWriter, _ *http.Request) {},
			cfg: builtin.HTTPConfig{
				FailureExpression: "true",
				ErrorExpression:   "invalid()",
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.ErrorContains(t, err, "HTTP (200) response met failure criteria")
				require.NotContains(t, err.Error(), "HTTP (200) response met failure criteria:")
				require.True(t, promotion.IsTerminal(err))
				require.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)
			},
		},
		{
			name:    "success AND failed", // Treated like a failure
			handler: func(_ http.ResponseWriter, _ *http.Request) {},
			cfg: builtin.HTTPConfig{
				SuccessExpression: "response.status == 200",
				FailureExpression: "response.status == 200",
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.ErrorContains(t, err, "HTTP (200) response met failure criteria")
				require.True(t, promotion.IsTerminal(err))
				require.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)
			},
		},
		{
			name: "neither success nor failed",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusBadGateway)
			},
			cfg: builtin.HTTPConfig{
				SuccessExpression: "response.status == 200",
				FailureExpression: "response.status == 404",
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusRunning, res.Status)
				// Suggests the default poll interval while waiting.
				require.NotNil(t, res.RetryAfter)
				require.Equal(t, httpPollIntervalDefault, *res.RetryAfter)
			},
		},
		{
			name: "neither success nor failed with explicit pollInterval",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusBadGateway)
			},
			cfg: builtin.HTTPConfig{
				SuccessExpression: "response.status == 200",
				FailureExpression: "response.status == 404",
				PollInterval:      "15s",
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusRunning, res.Status)
				require.NotNil(t, res.RetryAfter)
				require.Equal(t, 15*time.Second, *res.RetryAfter)
			},
		},
		{
			name: "undefined criteria with 2xx response",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			cfg: builtin.HTTPConfig{
				// No success or failure expressions
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
			},
		},
		{
			name: "undefined criteria with non-2xx response",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			cfg: builtin.HTTPConfig{
				// No success or failure expressions
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err) // Not terminal, should be retried
				require.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)
			},
		},
		{
			name: "text/plain response with numeric content",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				_, err := w.Write([]byte(`1`))
				require.NoError(t, err)
			},
			cfg: builtin.HTTPConfig{
				SuccessExpression: `response.body == "1"`,
				Outputs: []builtin.HTTPOutput{
					{
						Name:           "numeric_response",
						FromExpression: "response.body",
					},
				},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
				require.Equal(
					t,
					map[string]any{
						"numeric_response": "1",
					},
					res.Output,
				)
			},
		},
		{
			name: "text/plain response with word content",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				_, err := w.Write([]byte(`one`))
				require.NoError(t, err)
			},
			cfg: builtin.HTTPConfig{
				SuccessExpression: `response.body == "one"`,
				Outputs: []builtin.HTTPOutput{
					{
						Name:           "word_response",
						FromExpression: "response.body",
					},
				},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
				require.Equal(
					t,
					map[string]any{
						"word_response": "one",
					},
					res.Output,
				)
			},
		},
		{
			name: "YAML response with application/yaml content-type",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/yaml")
				_, err := w.Write([]byte("status: ok\ncount: 42\n"))
				require.NoError(t, err)
			},
			cfg: builtin.HTTPConfig{
				SuccessExpression: `response.body.status == "ok"`,
				Outputs: []builtin.HTTPOutput{
					{
						Name:           "yaml_status",
						FromExpression: "response.body.status",
					},
					{
						Name:           "yaml_count",
						FromExpression: "response.body.count",
					},
				},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
				require.Equal(
					t,
					map[string]any{
						"yaml_status": "ok",
						// sigs.k8s.io/yaml converts YAML to JSON first, so integers become float64
						"yaml_count": float64(42),
					},
					res.Output,
				)
			},
		},
		{
			name: "responseContentType config override forces YAML parsing",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				_, err := w.Write([]byte("key: value\n"))
				require.NoError(t, err)
			},
			cfg: builtin.HTTPConfig{
				ResponseContentType: "application/yaml",
				SuccessExpression:   `response.body.key == "value"`,
				Outputs: []builtin.HTTPOutput{
					{
						Name:           "yaml_key",
						FromExpression: "response.body.key",
					},
				},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
				require.Equal(
					t,
					map[string]any{
						"yaml_key": "value",
					},
					res.Output,
				)
			},
		},
		{
			name: "responseContentType config override forces text/plain parsing",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, err := w.Write([]byte(`{"foo": "bar"}`))
				require.NoError(t, err)
			},
			cfg: builtin.HTTPConfig{
				ResponseContentType: "text/plain",
				SuccessExpression:   `response.body == "{\"foo\": \"bar\"}"`,
				Outputs: []builtin.HTTPOutput{
					{
						Name:           "raw_body",
						FromExpression: "response.body",
					},
				},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
				require.Equal(
					t,
					map[string]any{
						"raw_body": `{"foo": "bar"}`,
					},
					res.Output,
				)
			},
		},
		{
			name: "ignored errorExpression",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set(contentTypeHeader, contentTypeJSON)
				w.WriteHeader(http.StatusBadGateway)
				_, err := w.Write([]byte(`{"error": "should not appear"}`))
				require.NoError(t, err)
			},
			cfg: builtin.HTTPConfig{
				SuccessExpression: "response.status == 200",
				FailureExpression: "response.status == 404",
				ErrorExpression:   "response.body.error",
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusRunning, res.Status)
			},
		},
	}

	h := &httpRequester{}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			srv := httptest.NewServer(testCase.handler)
			t.Cleanup(srv.Close)
			testCase.cfg.URL = srv.URL
			res, err := h.run(t.Context(), nil, testCase.cfg)
			testCase.assertions(t, res, err)
		})
	}
}

func Test_httpRequester_buildRequest(t *testing.T) {
	testCases := []struct {
		name       string
		setup      func(*testing.T) (*promotion.StepContext, builtin.HTTPConfig)
		assertions func(*testing.T, context.Context, *http.Request, error)
	}{
		{
			name: "request options",
			setup: func(_ *testing.T) (*promotion.StepContext, builtin.HTTPConfig) {
				return nil, builtin.HTTPConfig{
					Method: "GET",
					URL:    "http://example.com",
					Headers: []builtin.HTTPConfigHeader{{
						Name:  "Content-Type",
						Value: "application/json",
					}},
					QueryParams: []builtin.HTTPConfigQueryParam{{
						Name:  "param",
						Value: "some value", // We want to be sure this gets url-encoded
					}},
				}
			},
			assertions: func(t *testing.T, ctx context.Context, req *http.Request, err error) {
				require.NoError(t, err)
				require.Equal(t, ctx, req.Context())
				require.Equal(t, "GET", req.Method)
				require.Equal(t, "http://example.com?param=some+value", req.URL.String())
				require.Equal(t, "application/json", req.Header.Get("Content-Type"))
			},
		},
		{
			name: "body from file",
			setup: func(t *testing.T) (*promotion.StepContext, builtin.HTTPConfig) {
				workDir := t.TempDir()
				require.NoError(t, os.WriteFile(
					filepath.Join(workDir, "payload.json"),
					[]byte{0x00, 0xff, 0x01, 0x7f},
					0600,
				))
				return &promotion.StepContext{WorkDir: workDir}, builtin.HTTPConfig{
					Method:       "POST",
					URL:          "http://example.com",
					BodyFromFile: "payload.json",
				}
			},
			assertions: func(t *testing.T, _ context.Context, req *http.Request, err error) {
				require.NoError(t, err)
				body, readErr := io.ReadAll(req.Body)
				require.NoError(t, readErr)
				require.Equal(t, []byte{0x00, 0xff, 0x01, 0x7f}, body)
			},
		},
		{
			name: "body from file rejects oversize",
			setup: func(t *testing.T) (*promotion.StepContext, builtin.HTTPConfig) {
				workDir := t.TempDir()
				require.NoError(t, os.WriteFile(
					filepath.Join(workDir, "payload.bin"),
					bytes.Repeat([]byte{0x01}, maxResponseBytes+1),
					0600,
				))
				return &promotion.StepContext{WorkDir: workDir}, builtin.HTTPConfig{
					Method:       "POST",
					URL:          "http://example.com",
					BodyFromFile: "payload.bin",
				}
			},
			assertions: func(t *testing.T, _ context.Context, _ *http.Request, err error) {
				require.ErrorContains(t, err, "content exceeds limit")
			},
		},
		{
			name: "body from file stays inside work directory",
			setup: func(t *testing.T) (*promotion.StepContext, builtin.HTTPConfig) {
				parentDir := t.TempDir()
				workDir := filepath.Join(parentDir, "work")
				require.NoError(t, os.Mkdir(workDir, 0700))
				require.NoError(t, os.WriteFile(
					filepath.Join(parentDir, "payload.json"),
					[]byte("outside work directory"),
					0600,
				))
				return &promotion.StepContext{WorkDir: workDir}, builtin.HTTPConfig{
					URL:          "http://example.com",
					BodyFromFile: "../payload.json",
				}
			},
			assertions: func(t *testing.T, _ context.Context, _ *http.Request, err error) {
				require.ErrorContains(t, err, "could not read bodyFromFile")
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			stepCtx, cfg := testCase.setup(t)
			ctx := t.Context()
			req, err := (&httpRequester{}).buildRequest(ctx, stepCtx, cfg)
			testCase.assertions(t, ctx, req, err)
		})
	}
}

func Test_httpRequester_getClient(t *testing.T) {
	testCases := []struct {
		name       string
		cfg        builtin.HTTPConfig
		assertions func(*testing.T, *http.Client, error)
	}{
		{
			name: "without insecureSkipTLSVerify",
			assertions: func(t *testing.T, client *http.Client, err error) {
				require.NoError(t, err)
				require.NotNil(t, client)
				transport, ok := client.Transport.(*http.Transport)
				require.True(t, ok)
				require.Nil(t, transport.TLSClientConfig)
			},
		},
		{
			name: "with insecureSkipTLSVerify",
			cfg: builtin.HTTPConfig{
				InsecureSkipTLSVerify: true,
			},
			assertions: func(t *testing.T, client *http.Client, err error) {
				require.NoError(t, err)
				require.NotNil(t, client)
				transport, ok := client.Transport.(*http.Transport)
				require.True(t, ok)
				require.NotNil(t, transport.TLSClientConfig)
				require.True(t, transport.TLSClientConfig.InsecureSkipVerify)
			},
		},
		{
			name: "with invalid timeout",
			cfg: builtin.HTTPConfig{
				Timeout: "invalid",
			},
			assertions: func(t *testing.T, _ *http.Client, err error) {
				require.ErrorContains(t, err, "error parsing timeout")
			},
		},
	}
	h := &httpRequester{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			client, err := h.getClient(testCase.cfg)
			testCase.assertions(t, client, err)
		})
	}
}

func Test_httpRequester_buildExprEnv(t *testing.T) {
	testCases := []struct {
		name                string
		resp                *http.Response
		responseContentType string
		assertions          func(*testing.T, map[string]any, error)
	}{
		{
			name: "response body Content-Length exceeds limit",
			resp: &http.Response{
				StatusCode:    http.StatusOK,
				ContentLength: (2 << 20) + 1,
				Header:        http.Header{"Content-Type": []string{"application/json"}},
				Body:          io.NopCloser(strings.NewReader(`{"foo": "bar"}`)),
			},
			assertions: func(t *testing.T, env map[string]any, err error) {
				require.Error(t, err)
				require.ErrorContains(t, err, "response body size")
				require.Nil(t, env)
			},
		},
		{
			name: "without body",
			resp: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader("")),
			},
			assertions: func(t *testing.T, env map[string]any, err error) {
				require.NoError(t, err)
				statusAny, ok := env["response"].(map[string]any)["status"]
				require.True(t, ok)
				status, ok := statusAny.(int64)
				require.True(t, ok)
				require.Equal(t, int64(http.StatusOK), status)
				headerFnAny, ok := env["response"].(map[string]any)["header"]
				require.True(t, ok)
				headerFn, ok := headerFnAny.(func(string) string)
				require.True(t, ok)
				require.Equal(t, "application/json", headerFn("Content-Type"))
				headersAny, ok := env["response"].(map[string]any)["headers"]
				require.True(t, ok)
				headers, ok := headersAny.(http.Header)
				require.True(t, ok)
				require.Equal(t, http.Header{"Content-Type": []string{"application/json"}}, headers)
				bodyAny, ok := env["response"].(map[string]any)["body"]
				require.True(t, ok)
				body, ok := bodyAny.(map[string]any)
				require.True(t, ok)
				require.Empty(t, body)
			},
		},
		{
			name: "with body",
			resp: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"foo": "bar"}`)),
			},
			assertions: func(t *testing.T, env map[string]any, err error) {
				require.NoError(t, err)
				bodyAny, ok := env["response"].(map[string]any)["body"]
				require.True(t, ok)
				body, ok := bodyAny.(map[string]any)
				require.True(t, ok)
				require.Equal(t, map[string]any{"foo": "bar"}, body)
			},
		},
		{
			name: "with body as an array",
			resp: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`[{"foo1": "bar1"}, {"foo2": "bar2"}]`)),
			},
			assertions: func(t *testing.T, env map[string]any, err error) {
				require.NoError(t, err)
				bodyAny, ok := env["response"].(map[string]any)["body"]
				require.True(t, ok)

				// Check if interface is of type []any
				body, ok := bodyAny.([]any)
				require.True(t, ok)
				require.Len(t, body, 2)

				firstItem, ok := body[0].(map[string]any)
				require.True(t, ok)
				require.Equal(t, map[string]any{"foo1": "bar1"}, firstItem)
			},
		},
		{
			name: "invalid JSON body",
			resp: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"foo":`)),
			},
			assertions: func(t *testing.T, _ map[string]any, err error) {
				require.Error(t, err)
				require.ErrorContains(t, err, "failed to parse JSON response")
			},
		},
		{
			name: "JSON string response succeeds",
			resp: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`"foo"`)),
			},
			assertions: func(t *testing.T, env map[string]any, err error) {
				require.NoError(t, err)
				body := env["response"].(map[string]any)["body"] // nolint:forcetypeassert
				require.Equal(t, "foo", body)
			},
		},
		{
			name: "JSON number response succeeds",
			resp: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`42`)),
			},
			assertions: func(t *testing.T, env map[string]any, err error) {
				require.NoError(t, err)
				body := env["response"].(map[string]any)["body"] // nolint:forcetypeassert
				require.Equal(t, float64(42), body)
			},
		},
		{
			name: "JSON boolean response succeeds",
			resp: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`true`)),
			},
			assertions: func(t *testing.T, env map[string]any, err error) {
				require.NoError(t, err)
				body := env["response"].(map[string]any)["body"] // nolint:forcetypeassert
				require.Equal(t, true, body)
			},
		},
		{
			name: "JSON null response succeeds",
			resp: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`null`)),
			},
			assertions: func(t *testing.T, env map[string]any, err error) {
				require.NoError(t, err)
				body := env["response"].(map[string]any)["body"] // nolint:forcetypeassert
				require.Nil(t, body)
			},
		},
		{
			name: "case-insensitive JSON content-type",
			resp: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"APPLICATION/JSON"}},
				Body:       io.NopCloser(strings.NewReader(`{"foo": "bar"}`)),
			},
			assertions: func(t *testing.T, env map[string]any, err error) {
				require.NoError(t, err)
				body := env["response"].(map[string]any)["body"].(map[string]any) // nolint:forcetypeassert
				require.Equal(t, "bar", body["foo"])
			},
		},
		{
			name: "unknown content-type with invalid JSON leaves body empty",
			resp: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/html"}},
				Body:       io.NopCloser(strings.NewReader(`<html>hello</html>`)),
			},
			assertions: func(t *testing.T, env map[string]any, err error) {
				require.NoError(t, err)
				body := env["response"].(map[string]any)["body"] // nolint:forcetypeassert
				require.Equal(t, map[string]any{}, body)
			},
		},
		{
			name: "text/plain with JSON-like content stays as string",
			resp: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/plain"}},
				Body:       io.NopCloser(strings.NewReader(`{"key": "value"}`)),
			},
			assertions: func(t *testing.T, env map[string]any, err error) {
				require.NoError(t, err)
				body := env["response"].(map[string]any)["body"] // nolint:forcetypeassert
				// Should be a string, not parsed as JSON
				require.Equal(t, `{"key": "value"}`, body)
			},
		},
		{
			name: "empty content-type with non-JSON body leaves body empty",
			resp: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{},
				Body:       io.NopCloser(strings.NewReader(`hello world`)),
			},
			assertions: func(t *testing.T, env map[string]any, err error) {
				require.NoError(t, err)
				body := env["response"].(map[string]any)["body"] // nolint:forcetypeassert
				require.Equal(t, map[string]any{}, body)
			},
		},
		{
			name: "missing content-type but valid JSON body",
			resp: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{}, // No Content-Type header
				Body:       io.NopCloser(strings.NewReader(`{"foo": "bar"}`)),
			},
			assertions: func(t *testing.T, env map[string]any, err error) {
				require.NoError(t, err)
				bodyAny, ok := env["response"].(map[string]any)["body"]
				require.True(t, ok)

				body, ok := bodyAny.(map[string]any)
				require.True(t, ok)
				require.Equal(t, map[string]any{"foo": "bar"}, body)
			},
		},
		{
			name: "text/plain with numeric content",
			resp: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}},
				Body:       io.NopCloser(strings.NewReader(`1`)),
			},
			assertions: func(t *testing.T, env map[string]any, err error) {
				require.NoError(t, err)
				bodyAny, ok := env["response"].(map[string]any)["body"]
				require.True(t, ok)
				body, ok := bodyAny.(string)
				require.True(t, ok)
				require.Equal(t, "1", body)
			},
		},
		{
			name: "text/plain with float content",
			resp: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}},
				Body:       io.NopCloser(strings.NewReader(`3.14`)),
			},
			assertions: func(t *testing.T, env map[string]any, err error) {
				require.NoError(t, err)
				bodyAny, ok := env["response"].(map[string]any)["body"]
				require.True(t, ok)
				body, ok := bodyAny.(string)
				require.True(t, ok)
				require.Equal(t, "3.14", body)
			},
		},
		{
			name: "text/plain with word content",
			resp: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}},
				Body:       io.NopCloser(strings.NewReader(`one`)),
			},
			assertions: func(t *testing.T, env map[string]any, err error) {
				require.NoError(t, err)
				bodyAny, ok := env["response"].(map[string]any)["body"]
				require.True(t, ok)
				body, ok := bodyAny.(string)
				require.True(t, ok)
				require.Equal(t, "one", body)
			},
		},
		{
			name: "text/plain with sentence content",
			resp: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}},
				Body:       io.NopCloser(strings.NewReader(`this is not json`)),
			},
			assertions: func(t *testing.T, env map[string]any, err error) {
				require.NoError(t, err)
				bodyAny, ok := env["response"].(map[string]any)["body"]
				require.True(t, ok)
				body, ok := bodyAny.(string)
				require.True(t, ok)
				require.Equal(t, "this is not json", body)
			},
		},
		{
			name: "application/yaml content-type parses as YAML",
			resp: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/yaml"}},
				Body:       io.NopCloser(strings.NewReader("foo: bar\nbaz: 42\n")),
			},
			assertions: func(t *testing.T, env map[string]any, err error) {
				require.NoError(t, err)
				body := env["response"].(map[string]any)["body"].(map[string]any) // nolint:forcetypeassert
				require.Equal(t, "bar", body["foo"])
				// sigs.k8s.io/yaml converts YAML to JSON first, so integers become float64
				require.Equal(t, float64(42), body["baz"])
			},
		},
		{
			name: "text/yaml content-type parses as YAML",
			resp: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/yaml"}},
				Body:       io.NopCloser(strings.NewReader("items:\n  - one\n  - two\n")),
			},
			assertions: func(t *testing.T, env map[string]any, err error) {
				require.NoError(t, err)
				body := env["response"].(map[string]any)["body"].(map[string]any) // nolint:forcetypeassert
				items := body["items"].([]any)                                    // nolint:forcetypeassert
				require.Len(t, items, 2)
				require.Equal(t, "one", items[0])
				require.Equal(t, "two", items[1])
			},
		},
		{
			name: "application/x-yaml content-type parses as YAML",
			resp: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/x-yaml"}},
				Body:       io.NopCloser(strings.NewReader("key: value\n")),
			},
			assertions: func(t *testing.T, env map[string]any, err error) {
				require.NoError(t, err)
				body := env["response"].(map[string]any)["body"].(map[string]any) // nolint:forcetypeassert
				require.Equal(t, "value", body["key"])
			},
		},
		{
			name: "invalid YAML body returns error",
			resp: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/yaml"}},
				Body:       io.NopCloser(strings.NewReader("foo: [bar\n")),
			},
			assertions: func(t *testing.T, _ map[string]any, err error) {
				require.Error(t, err)
				require.ErrorContains(t, err, "failed to parse YAML response")
			},
		},
		// responseContentType config override tests
		{
			name: "responseContentType override forces JSON parsing",
			resp: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/plain"}},
				Body:       io.NopCloser(strings.NewReader(`{"foo": "bar"}`)),
			},
			responseContentType: "application/json",
			assertions: func(t *testing.T, env map[string]any, err error) {
				require.NoError(t, err)
				body := env["response"].(map[string]any)["body"].(map[string]any) // nolint:forcetypeassert
				require.Equal(t, "bar", body["foo"])
			},
		},
		{
			name: "responseContentType override forces YAML parsing",
			resp: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/plain"}},
				Body:       io.NopCloser(strings.NewReader("foo: bar\n")),
			},
			responseContentType: "application/yaml",
			assertions: func(t *testing.T, env map[string]any, err error) {
				require.NoError(t, err)
				body := env["response"].(map[string]any)["body"].(map[string]any) // nolint:forcetypeassert
				require.Equal(t, "bar", body["foo"])
			},
		},
		{
			name: "responseContentType override forces text/plain parsing",
			resp: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"foo": "bar"}`)),
			},
			responseContentType: "text/plain",
			assertions: func(t *testing.T, env map[string]any, err error) {
				require.NoError(t, err)
				body := env["response"].(map[string]any)["body"] // nolint:forcetypeassert
				// Should be a string, not parsed as JSON
				require.Equal(t, `{"foo": "bar"}`, body)
			},
		},
		{
			name: "responseContentType override text/yaml works",
			resp: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/html"}},
				Body:       io.NopCloser(strings.NewReader("key: value\n")),
			},
			responseContentType: "text/yaml",
			assertions: func(t *testing.T, env map[string]any, err error) {
				require.NoError(t, err)
				body := env["response"].(map[string]any)["body"].(map[string]any) // nolint:forcetypeassert
				require.Equal(t, "value", body["key"])
			},
		},
		{
			name: "unknown content-type with valid JSON falls back to JSON",
			resp: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/octet-stream"}},
				Body:       io.NopCloser(strings.NewReader(`{"success": true}`)),
			},
			assertions: func(t *testing.T, env map[string]any, err error) {
				require.NoError(t, err)
				body := env["response"].(map[string]any)["body"].(map[string]any) // nolint:forcetypeassert
				require.Equal(t, true, body["success"])
			},
		},
	}
	h := &httpRequester{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			env, err := h.buildExprEnv(t.Context(), testCase.resp, testCase.responseContentType)
			testCase.assertions(t, env, err)
		})
	}
}

func Test_httpRequester_evaluateSuccessCriteria(t *testing.T) {
	testCases := []struct {
		name       string
		cfg        builtin.HTTPConfig
		assertions func(t *testing.T, result *bool, err error)
	}{
		{
			name: "no success expression",
			cfg:  builtin.HTTPConfig{},
			assertions: func(t *testing.T, result *bool, err error) {
				require.NoError(t, err)
				require.Nil(t, result)
			},
		},
		{
			name: "error compiling success expression",
			cfg:  builtin.HTTPConfig{SuccessExpression: "(1 + 2"},
			assertions: func(t *testing.T, result *bool, err error) {
				require.ErrorContains(t, err, "error compiling success expression")
				require.True(t, promotion.IsTerminal(err))
				require.Nil(t, result)
			},
		},
		{
			name: "error evaluating success expression",
			cfg:  builtin.HTTPConfig{SuccessExpression: "invalid()"},
			assertions: func(t *testing.T, result *bool, err error) {
				require.ErrorContains(t, err, "error evaluating success expression")
				require.False(t, promotion.IsTerminal(err))
				require.Nil(t, result)
			},
		},
		{
			name: "success expression evaluates to non-boolean",
			cfg:  builtin.HTTPConfig{SuccessExpression: `"foo"`},
			assertions: func(t *testing.T, result *bool, err error) {
				require.ErrorContains(t, err, "success expression")
				require.ErrorContains(t, err, "did not evaluate to a boolean")
				require.False(t, promotion.IsTerminal(err))
				require.Nil(t, result)
			},
		},
		{
			name: "success expression evaluates to true",
			cfg:  builtin.HTTPConfig{SuccessExpression: "true"},
			assertions: func(t *testing.T, result *bool, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.True(t, *result)
			},
		},
		{
			name: "success expression evaluates to false",
			cfg:  builtin.HTTPConfig{SuccessExpression: "false"},
			assertions: func(t *testing.T, result *bool, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.False(t, *result)
			},
		},
	}
	h := &httpRequester{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := h.evaluateSuccessCriteria(testCase.cfg, nil)
			testCase.assertions(t, result, err)
		})
	}
}

func Test_httpRequester_evaluateFailureCriteria(t *testing.T) {
	testCases := []struct {
		name       string
		cfg        builtin.HTTPConfig
		assertions func(t *testing.T, result *bool, err error)
	}{
		{
			name: "no failure expression",
			cfg:  builtin.HTTPConfig{},
			assertions: func(t *testing.T, result *bool, err error) {
				require.NoError(t, err)
				require.Nil(t, result)
			},
		},
		{
			name: "error compiling failure expression",
			cfg:  builtin.HTTPConfig{FailureExpression: "(1 + 2"},
			assertions: func(t *testing.T, result *bool, err error) {
				require.ErrorContains(t, err, "error compiling failure expression")
				require.True(t, promotion.IsTerminal(err))
				require.Nil(t, result)
			},
		},
		{
			name: "error evaluating failure expression",
			cfg:  builtin.HTTPConfig{FailureExpression: "invalid()"},
			assertions: func(t *testing.T, result *bool, err error) {
				require.ErrorContains(t, err, "error evaluating failure expression")
				require.False(t, promotion.IsTerminal(err))
				require.Nil(t, result)
			},
		},
		{
			name: "failure expression evaluates to non-boolean",
			cfg:  builtin.HTTPConfig{FailureExpression: `"foo"`},
			assertions: func(t *testing.T, result *bool, err error) {
				require.ErrorContains(t, err, "did not evaluate to a boolean")
				require.False(t, promotion.IsTerminal(err))
				require.Nil(t, result)
			},
		},
		{
			name: "failure expression evaluates to true",
			cfg:  builtin.HTTPConfig{FailureExpression: "true"},
			assertions: func(t *testing.T, result *bool, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.True(t, *result)
			},
		},
		{
			name: "failure expression evaluates to false",
			cfg:  builtin.HTTPConfig{FailureExpression: "false"},
			assertions: func(t *testing.T, result *bool, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.False(t, *result)
			},
		},
	}
	h := &httpRequester{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := h.evaluateFailureCriteria(testCase.cfg, nil)
			testCase.assertions(t, result, err)
		})
	}
}

func Test_httpRequester_buildOutputs(t *testing.T) {
	testCases := []struct {
		name        string
		outputExprs []builtin.HTTPOutput
		assertions  func(t *testing.T, outputs map[string]any, err error)
	}{
		{
			name: "no outputs specified",
			assertions: func(t *testing.T, outputs map[string]any, err error) {
				require.NoError(t, err)
				require.Empty(t, outputs)
			},
		},
		{
			name: "error compiling output expression",
			outputExprs: []builtin.HTTPOutput{{
				Name:           "fake-output",
				FromExpression: "(1 + 2",
			}},
			assertions: func(t *testing.T, _ map[string]any, err error) {
				require.ErrorContains(t, err, "error compiling output expression")
				require.True(t, promotion.IsTerminal(err))
			},
		},
		{
			name: "error evaluating output expression",
			outputExprs: []builtin.HTTPOutput{{
				Name:           "fake-output",
				FromExpression: "invalid()",
			}},
			assertions: func(t *testing.T, _ map[string]any, err error) {
				require.ErrorContains(t, err, "error evaluating output expression")
				require.False(t, promotion.IsTerminal(err))
			},
		},
		{
			name: "success",
			outputExprs: []builtin.HTTPOutput{
				{
					Name:           "string-output",
					FromExpression: `"foo"`,
				},
				{
					Name:           "int-output",
					FromExpression: "42",
				},
			},
			assertions: func(t *testing.T, outputs map[string]any, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					map[string]any{
						"string-output": "foo",
						"int-output":    42,
					},
					outputs,
				)
			},
		},
	}
	h := &httpRequester{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			outputs, err := h.buildOutputs(testCase.outputExprs, nil)
			testCase.assertions(t, outputs, err)
		})
	}
}

func Test_httpRequester_determineResponseParseMode(t *testing.T) {
	testCases := []struct {
		name        string
		contentType string
		expected    httpResponseParseMode
	}{
		{
			name:        "application/json returns JSON mode",
			contentType: "application/json",
			expected:    httpParseModeJSON,
		},
		{
			name:        "APPLICATION/JSON (uppercase) returns JSON mode",
			contentType: "APPLICATION/JSON",
			expected:    httpParseModeJSON,
		},
		{
			name:        "Application/Json (mixed case) returns JSON mode",
			contentType: "Application/Json",
			expected:    httpParseModeJSON,
		},
		{
			name:        "application/yaml returns YAML mode",
			contentType: "application/yaml",
			expected:    httpParseModeYAML,
		},
		{
			name:        "text/yaml returns YAML mode",
			contentType: "text/yaml",
			expected:    httpParseModeYAML,
		},
		{
			name:        "application/x-yaml returns YAML mode",
			contentType: "application/x-yaml",
			expected:    httpParseModeYAML,
		},
		{
			name:        "APPLICATION/YAML (uppercase) returns YAML mode",
			contentType: "APPLICATION/YAML",
			expected:    httpParseModeYAML,
		},
		{
			name:        "TEXT/YAML (uppercase) returns YAML mode",
			contentType: "TEXT/YAML",
			expected:    httpParseModeYAML,
		},
		{
			name:        "text/plain returns text mode",
			contentType: "text/plain",
			expected:    httpParseModeText,
		},
		{
			name:        "TEXT/PLAIN (uppercase) returns text mode",
			contentType: "TEXT/PLAIN",
			expected:    httpParseModeText,
		},
		{
			name:        "empty string falls back to JSON mode",
			contentType: "",
			expected:    httpParseModeJSON,
		},
		{
			name:        "text/html falls back to JSON mode",
			contentType: "text/html",
			expected:    httpParseModeJSON,
		},
		{
			name:        "application/octet-stream falls back to JSON mode",
			contentType: "application/octet-stream",
			expected:    httpParseModeJSON,
		},
		{
			name:        "unknown/type falls back to JSON mode",
			contentType: "unknown/type",
			expected:    httpParseModeJSON,
		},
	}

	h := &httpRequester{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := h.determineResponseParseMode(tc.contentType)
			require.Equal(t, tc.expected, result)
		})
	}
}

func Test_httpRequester_proxy(t *testing.T) {
	// Target server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that the request passed through the proxy
		require.Equal(t, "true", r.Header.Get("X-Test-HttpRequester-Proxy-Injected"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"reachedBackend": true}`))
		require.NoError(t, err)
	}))
	defer backend.Close()

	// Forward proxy
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		outReq, err := http.NewRequestWithContext(r.Context(), r.Method, r.URL.String(), r.Body)
		require.NoError(t, err)

		// Add an extra header, to prove that the request passed through this proxy
		maps.Copy(outReq.Header, r.Header)
		outReq.Header.Set("X-Test-HttpRequester-Proxy-Injected", "true")

		// Forward the request to the target URL
		resp, err := http.DefaultClient.Do(outReq)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Relay response to the client
		maps.Copy(w.Header(), resp.Header)
		w.WriteHeader(resp.StatusCode)
		_, err = io.Copy(w, resp.Body)
		require.NoError(t, err)
	}))
	defer proxy.Close()

	testCases := []struct {
		name       string
		cfg        builtin.HTTPConfig
		assertions func(t *testing.T, res promotion.StepResult, err error)
	}{
		{
			name: "proxy URL with invalid format returns error",
			cfg: builtin.HTTPConfig{
				URL:   backend.URL,
				Proxy: "http://[invalid-url",
			},
			assertions: func(t *testing.T, _ promotion.StepResult, err error) {
				require.ErrorContains(t, err, "error parsing proxy URL")
			},
		},
		{
			name: "proxy is used when configured",
			cfg: builtin.HTTPConfig{
				URL:               backend.URL,
				Proxy:             proxy.URL,
				SuccessExpression: "response.status == 200",
				Outputs: []builtin.HTTPOutput{{
					Name:           "reachedBackend",
					FromExpression: "response.body.reachedBackend",
				}},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
				require.Equal(t, true, res.Output["reachedBackend"])
			},
		},
	}

	for _, tc := range testCases {
		h := &httpRequester{}
		result, err := h.run(t.Context(), nil, tc.cfg)
		tc.assertions(t, result, err)
	}
}

func Test_httpRequester_run_download(t *testing.T) {
	testCases := []struct {
		name    string
		cfg     builtin.HTTPConfig
		setup   func(*testing.T, string)
		handler http.HandlerFunc
		assert  func(*testing.T, string, promotion.StepResult, error)
	}{
		{
			name: "small download succeeds, file moved, body parsed",
			cfg: builtin.HTTPConfig{
				OutPath:           "downloads/answer.json",
				SuccessExpression: "true",
				Outputs: []builtin.HTTPOutput{{
					Name:           "meaning",
					FromExpression: "response.body.theMeaningOfLife",
				}},
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set(contentTypeHeader, contentTypeJSON)
				_, err := w.Write([]byte(`{"theMeaningOfLife": 42}`))
				require.NoError(t, err)
			},
			assert: func(t *testing.T, workDir string, res promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
				content, err := os.ReadFile(filepath.Join(workDir, "downloads", "answer.json"))
				require.NoError(t, err)
				require.JSONEq(t, `{"theMeaningOfLife": 42}`, string(content))
				require.Equal(t, float64(42), res.Output["meaning"])
				// No temp files left behind.
				requireNoTempFiles(t, filepath.Join(workDir, "downloads"))
			},
		},
		{
			name: "existing file without allowOverwrite fails terminally before request",
			cfg: builtin.HTTPConfig{
				OutPath: "existing.txt",
			},
			setup: func(t *testing.T, workDir string) {
				require.NoError(t, os.WriteFile(
					filepath.Join(workDir, "existing.txt"), []byte("old"), 0o600,
				))
			},
			handler: func(_ http.ResponseWriter, _ *http.Request) {
				t.Error("request must not be sent when the file exists and overwrite is disallowed")
			},
			assert: func(t *testing.T, workDir string, res promotion.StepResult, err error) {
				require.Error(t, err)
				var termErr *promotion.TerminalError
				require.ErrorAs(t, err, &termErr)
				require.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)
				requireNoTempFiles(t, workDir)
			},
		},
		{
			name: "existing file with allowOverwrite is replaced",
			cfg: builtin.HTTPConfig{
				OutPath:           "f.txt",
				AllowOverwrite:    true,
				SuccessExpression: "true",
			},
			setup: func(t *testing.T, workDir string) {
				require.NoError(t, os.WriteFile(
					filepath.Join(workDir, "f.txt"), []byte("old"), 0o600,
				))
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, err := w.Write([]byte("new"))
				require.NoError(t, err)
			},
			assert: func(t *testing.T, workDir string, res promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
				content, err := os.ReadFile(filepath.Join(workDir, "f.txt"))
				require.NoError(t, err)
				require.Equal(t, "new", string(content))
			},
		},
		{
			name: "large body is downloaded but not parsed into response.body",
			cfg: builtin.HTTPConfig{
				OutPath:           "big.bin",
				SuccessExpression: `response.body == {}`,
				Outputs: []builtin.HTTPOutput{{
					Name:           "body",
					FromExpression: "response.body",
				}},
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set(contentTypeHeader, contentTypeTextPlain)
				_, err := w.Write(bytes.Repeat([]byte("x"), 3<<20))
				require.NoError(t, err)
			},
			assert: func(t *testing.T, workDir string, res promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
				info, err := os.Stat(filepath.Join(workDir, "big.bin"))
				require.NoError(t, err)
				require.Equal(t, int64(3<<20), info.Size())
				require.Equal(t, map[string]any{}, res.Output["body"])
			},
		},
		{
			name: "oversized response fails terminally",
			cfg: builtin.HTTPConfig{
				OutPath: "huge.bin",
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Length", "104857601")
				w.WriteHeader(http.StatusOK)
			},
			assert: func(t *testing.T, workDir string, res promotion.StepResult, err error) {
				require.Error(t, err)
				var termErr *promotion.TerminalError
				require.ErrorAs(t, err, &termErr)
				require.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)
				_, statErr := os.Stat(filepath.Join(workDir, "huge.bin"))
				require.True(t, os.IsNotExist(statErr))
				requireNoTempFiles(t, workDir)
			},
		},
		{
			name: "failure criteria discards the file",
			cfg: builtin.HTTPConfig{
				OutPath:           "f.json",
				FailureExpression: "true",
				ErrorExpression:   `"boom"`,
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set(contentTypeHeader, contentTypeJSON)
				_, err := w.Write([]byte(`{}`))
				require.NoError(t, err)
			},
			assert: func(t *testing.T, workDir string, res promotion.StepResult, err error) {
				require.Error(t, err)
				var termErr *promotion.TerminalError
				require.ErrorAs(t, err, &termErr)
				require.ErrorContains(t, err, "boom")
				require.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)
				_, statErr := os.Stat(filepath.Join(workDir, "f.json"))
				require.True(t, os.IsNotExist(statErr))
				requireNoTempFiles(t, workDir)
			},
		},
		{
			name: "non-2xx without criteria fails retried and discards the file",
			cfg: builtin.HTTPConfig{
				OutPath: "f.json",
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			assert: func(t *testing.T, workDir string, res promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)
				_, statErr := os.Stat(filepath.Join(workDir, "f.json"))
				require.True(t, os.IsNotExist(statErr))
				requireNoTempFiles(t, workDir)
			},
		},
		{
			name: "outputs evaluation failure errors and discards the file",
			cfg: builtin.HTTPConfig{
				OutPath:           "f.json",
				SuccessExpression: "true",
				Outputs: []builtin.HTTPOutput{{
					Name:           "bad",
					FromExpression: "this is not valid expr [",
				}},
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, err := w.Write([]byte("ok"))
				require.NoError(t, err)
			},
			assert: func(t *testing.T, workDir string, res promotion.StepResult, err error) {
				require.Error(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusErrored, res.Status)
				_, statErr := os.Stat(filepath.Join(workDir, "f.json"))
				require.True(t, os.IsNotExist(statErr))
				requireNoTempFiles(t, workDir)
			},
		},
		{
			name: "unparseable body errors and removes the temp file",
			cfg:  builtin.HTTPConfig{OutPath: "f.json"},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set(contentTypeHeader, contentTypeJSON)
				_, _ = w.Write([]byte("{not json"))
			},
			assert: func(t *testing.T, workDir string, res promotion.StepResult, err error) {
				require.Error(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusErrored, res.Status)
				_, statErr := os.Stat(filepath.Join(workDir, "f.json"))
				require.True(t, os.IsNotExist(statErr))
				requireNoTempFiles(t, workDir)
			},
		},
		{
			name: "oversized body without declared length fails terminally and removes the temp file",
			cfg: builtin.HTTPConfig{
				OutPath: "huge2.bin",
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				// No Content-Length is declared, so the cap can only be hit
				// mid-stream. Write errors are expected once the client hits
				// the cap and hangs up, so they are ignored.
				chunk := bytes.Repeat([]byte("x"), 1<<20)
				for written := int64(0); written <= maxDownloadSize; written += int64(len(chunk)) {
					_, _ = w.Write(chunk)
				}
			},
			assert: func(t *testing.T, workDir string, res promotion.StepResult, err error) {
				require.Error(t, err)
				var termErr *promotion.TerminalError
				require.ErrorAs(t, err, &termErr)
				require.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)
				_, statErr := os.Stat(filepath.Join(workDir, "huge2.bin"))
				require.True(t, os.IsNotExist(statErr))
				requireNoTempFiles(t, workDir)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			workDir := t.TempDir()
			if testCase.setup != nil {
				testCase.setup(t, workDir)
			}
			srv := httptest.NewServer(testCase.handler)
			t.Cleanup(srv.Close)
			testCase.cfg.URL = srv.URL
			stepCtx := &promotion.StepContext{WorkDir: workDir}
			h := &httpRequester{}
			res, err := h.run(t.Context(), stepCtx, testCase.cfg)
			testCase.assert(t, workDir, res, err)
		})
	}
}

// requireNoTempFiles asserts that dir holds no leftover download temp files.
func requireNoTempFiles(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, e := range entries {
		require.NotContains(t, e.Name(), ".tmp")
	}
}

func Test_streamResponseToTempFile(t *testing.T) {
	testCases := []struct {
		name string
		// setup builds the context and reader for the case. Most cases just
		// pair t.Context() with a fixed reader, but the canceled-context
		// case needs the two built together.
		setup func(t *testing.T) (context.Context, io.Reader)
		// subDir, when set, streams into this subdirectory of the test's
		// temp dir instead of the temp dir itself. It is deliberately not
		// created first, so a case can use a path that does not exist.
		subDir string
		assert func(t *testing.T, dir string, tempPath string, size int64, err error)
	}{
		{
			name: "success leaves the temp file for the caller",
			setup: func(t *testing.T) (context.Context, io.Reader) {
				return t.Context(), strings.NewReader("hello")
			},
			assert: func(t *testing.T, _ string, tempPath string, size int64, err error) {
				require.NoError(t, err)
				require.Equal(t, int64(5), size)
				content, err := os.ReadFile(tempPath)
				require.NoError(t, err)
				require.Equal(t, "hello", string(content))
			},
		},
		{
			name: "oversized body removes the temp file",
			setup: func(t *testing.T) (context.Context, io.Reader) {
				return t.Context(), io.LimitReader(zeroReader{}, maxDownloadSize+1)
			},
			assert: func(t *testing.T, dir string, _ string, _ int64, err error) {
				require.Error(t, err)
				var termErr *promotion.TerminalError
				require.ErrorAs(t, err, &termErr)
				requireNoTempFiles(t, dir)
			},
		},
		{
			name: "read error removes the temp file",
			setup: func(t *testing.T) (context.Context, io.Reader) {
				return t.Context(), io.MultiReader(
					strings.NewReader("partial"), errReader{},
				)
			},
			assert: func(t *testing.T, dir string, _ string, _ int64, err error) {
				require.ErrorContains(t, err, "failed to read response body")
				requireNoTempFiles(t, dir)
			},
		},
		{
			name: "canceled context removes the temp file",
			setup: func(t *testing.T) (context.Context, io.Reader) {
				ctx, cancel := context.WithCancel(t.Context())
				return ctx, &cancelingReader{cancel: cancel}
			},
			assert: func(t *testing.T, dir string, _ string, _ int64, err error) {
				require.ErrorContains(t, err, "download canceled")
				requireNoTempFiles(t, dir)
			},
		},
		{
			name:   "unwritable directory fails before creating anything",
			subDir: "does-not-exist",
			setup: func(t *testing.T) (context.Context, io.Reader) {
				return t.Context(), strings.NewReader("hello")
			},
			assert: func(t *testing.T, dir string, _ string, _ int64, err error) {
				require.ErrorContains(t, err, "failed to create temporary file")
				requireNoTempFiles(t, dir)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			streamDir := dir
			if testCase.subDir != "" {
				streamDir = filepath.Join(dir, testCase.subDir)
			}
			ctx, reader := testCase.setup(t)
			tempPath, size, err := streamResponseToTempFile(
				ctx, reader, streamDir, "out",
			)
			testCase.assert(t, dir, tempPath, size, err)
		})
	}
}

// zeroReader is an io.Reader that yields an endless stream of zero bytes.
type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

// errReader is an io.Reader whose reads always fail.
type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, errors.New("read failure")
}

// cancelingReader cancels its context on the first Read and then yields one
// byte per Read without ever reaching EOF, so the only way out of the read
// loop is the canceled context.
type cancelingReader struct {
	cancel context.CancelFunc
	done   bool
}

func (r *cancelingReader) Read(p []byte) (int, error) {
	if !r.done {
		r.done = true
		r.cancel()
	}
	p[0] = 'x'
	return 1, nil
}

func Test_httpRequester_getClient_downloadTimeout(t *testing.T) {
	testCases := []struct {
		name        string
		cfg         builtin.HTTPConfig
		wantTimeout time.Duration
	}{
		{
			name:        "default timeout without outPath",
			wantTimeout: 10 * time.Second,
		},
		{
			name: "default timeout with outPath",
			cfg: builtin.HTTPConfig{
				OutPath: "f.bin",
			},
			wantTimeout: time.Minute,
		},
		{
			name: "explicit timeout wins over download default",
			cfg: builtin.HTTPConfig{
				OutPath: "f.bin",
				Timeout: "5s",
			},
			wantTimeout: 5 * time.Second,
		},
	}
	h := &httpRequester{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			client, err := h.getClient(testCase.cfg)
			require.NoError(t, err)
			require.Equal(t, testCase.wantTimeout, client.Timeout)
		})
	}
}
