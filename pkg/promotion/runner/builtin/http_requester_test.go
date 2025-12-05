package builtin

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
				"successExpression":     "response.status == 200",
				"failureExpression":     "response.status == 404",
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
	}

	h := &httpRequester{}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			srv := httptest.NewServer(testCase.handler)
			t.Cleanup(srv.Close)
			testCase.cfg.URL = srv.URL
			res, err := h.run(context.Background(), nil, testCase.cfg)
			testCase.assertions(t, res, err)
		})
	}
}

func Test_httpRequester_buildRequest(t *testing.T) {
	req, err := (&httpRequester{}).buildRequest(builtin.HTTPConfig{
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
	})
	require.NoError(t, err)
	require.Equal(t, "GET", req.Method)
	require.Equal(t, "http://example.com?param=some+value", req.URL.String())
	require.Equal(t, "application/json", req.Header.Get("Content-Type"))
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
			env, err := h.buildExprEnv(context.Background(), testCase.resp, testCase.responseContentType)
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
