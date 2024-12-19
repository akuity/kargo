package directives

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func Test_httpRequester_validate(t *testing.T) {
	testCases := []struct {
		name             string
		config           Config
		expectedProblems []string
	}{
		{
			name:   "url not specified",
			config: Config{},
			expectedProblems: []string{
				"(root): url is required",
			},
		},
		{
			name: "url is empty string",
			config: Config{
				"url": "",
			},
			expectedProblems: []string{
				"url: String length must be greater than or equal to 1",
			},
		},
		{
			name: "invalid method",
			config: Config{
				"method": "invalid",
			},
			expectedProblems: []string{
				"method: Does not match pattern",
			},
		},
		{
			name: "header name not specified",
			config: Config{
				"headers": []Config{{}},
			},
			expectedProblems: []string{
				"headers.0: name is required",
			},
		},
		{
			name: "header name is empty string",
			config: Config{
				"headers": []Config{{
					"name": "",
				}},
			},
			expectedProblems: []string{
				"headers.0.name: String length must be greater than or equal to 1",
			},
		},
		{
			name: "header value not specified",
			config: Config{
				"headers": []Config{{}},
			},
			expectedProblems: []string{
				"headers.0: value is required",
			},
		},
		{
			name: "header value is empty string",
			config: Config{
				"headers": []Config{{
					"value": "",
				}},
			},
			expectedProblems: []string{
				"headers.0.value: String length must be greater than or equal to 1",
			},
		},
		{
			name: "query param name not specified",
			config: Config{
				"queryParams": []Config{{}},
			},
			expectedProblems: []string{
				"queryParams.0: name is required",
			},
		},
		{
			name: "query param name is empty string",
			config: Config{
				"queryParams": []Config{{
					"name": "",
				}},
			},
			expectedProblems: []string{
				"queryParams.0.name: String length must be greater than or equal to 1",
			},
		},
		{
			name: "query param value not specified",
			config: Config{
				"queryParams": []Config{{}},
			},
			expectedProblems: []string{
				"queryParams.0: value is required",
			},
		},
		{
			name: "query param value is empty string",
			config: Config{
				"queryParams": []Config{{
					"value": "",
				}},
			},
			expectedProblems: []string{
				"queryParams.0.value: String length must be greater than or equal to 1",
			},
		},
		{
			name: "invalid timeout",
			config: Config{
				"timeout": "invalid",
			},
			expectedProblems: []string{
				"timeout: Does not match pattern",
			},
		},
		{
			name: "output name not specified",
			config: Config{
				"outputs": []Config{{}},
			},
			expectedProblems: []string{
				"outputs.0: name is required",
			},
		},
		{
			name: "output name is empty string",
			config: Config{
				"outputs": []Config{{
					"name": "",
				}},
			},
			expectedProblems: []string{
				"outputs.0.name: String length must be greater than or equal to 1",
			},
		},
		{
			name: "output fromExpression not specified",
			config: Config{
				"outputs": []Config{{}},
			},
			expectedProblems: []string{
				"outputs.0: fromExpression is required",
			},
		},
		{
			name: "output fromExpression is empty string",
			config: Config{
				"outputs": []Config{{
					"fromExpression": "",
				}},
			},
			expectedProblems: []string{
				"outputs.0.fromExpression: String length must be greater than or equal to 1",
			},
		},
		{
			name: "valid kitchen sink",
			config: Config{
				"method": "GET",
				"url":    "https://example.com",
				"headers": []Config{{
					"name":  "Accept",
					"value": "application/json",
				}},
				"queryParams": []Config{{
					"name":  "foo",
					"value": "bar",
				}},
				"insecureSkipTLSVerify": true,
				"timeout":               "30s",
				"successExpression":     "response.status == 200",
				"failureExpression":     "response.status == 404",
				"outputs": []Config{
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

	r := newHTTPRequester()
	runner, ok := r.(*httpRequester)
	require.True(t, ok)

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := runner.validate(testCase.config)
			if len(testCase.expectedProblems) == 0 {
				require.NoError(t, err)
			} else {
				for _, problem := range testCase.expectedProblems {
					require.ErrorContains(t, err, problem)
				}
			}
		})
	}
}

func Test_httpRequester_runPromotionStep(t *testing.T) {
	testCases := []struct {
		name       string
		cfg        HTTPConfig
		handler    http.HandlerFunc
		assertions func(*testing.T, PromotionStepResult, error)
	}{
		{
			name:    "success and not failed; no body",
			handler: func(_ http.ResponseWriter, _ *http.Request) {},
			cfg: HTTPConfig{
				SuccessExpression: "true",
				Outputs: []HTTPOutput{
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
			assertions: func(t *testing.T, res PromotionStepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionPhaseSucceeded, res.Status)
				require.Equal(
					t,
					map[string]any{
						"status":           http.StatusOK,
						"theMeaningOfLife": nil,
					},
					res.Output,
				)
			},
		},
		{
			name: "success and not failed; non-json body",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				// This is JSON, but the content type is not set to application/json
				_, err := w.Write([]byte(`{"theMeaningOfLife": 42}`))
				require.NoError(t, err)
			},
			cfg: HTTPConfig{
				SuccessExpression: "true",
				Outputs: []HTTPOutput{
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
			assertions: func(t *testing.T, res PromotionStepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionPhaseSucceeded, res.Status)
				require.Equal(
					t,
					map[string]any{
						"status":           http.StatusOK,
						"theMeaningOfLife": nil,
					},
					res.Output,
				)
			},
		},
		{
			name: "success and not failed with json body",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set(contentTypeHeader, contentTypeJSON)
				_, err := w.Write([]byte(`{"theMeaningOfLife": 42}`))
				require.NoError(t, err)
			},
			cfg: HTTPConfig{
				SuccessExpression: "true",
				Outputs: []HTTPOutput{
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
			assertions: func(t *testing.T, res PromotionStepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionPhaseSucceeded, res.Status)
				require.Equal(
					t,
					map[string]any{
						"status":           http.StatusOK,
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
			cfg: HTTPConfig{
				FailureExpression: "response.status == 404",
			},
			assertions: func(t *testing.T, res PromotionStepResult, err error) {
				require.ErrorContains(t, err, "HTTP response met failure criteria")
				require.True(t, isTerminal(err))
				require.Equal(t, kargoapi.PromotionPhaseFailed, res.Status)
			},
		},
		{
			name:    "success AND failed", // Treated like a failure
			handler: func(_ http.ResponseWriter, _ *http.Request) {},
			cfg: HTTPConfig{
				SuccessExpression: "response.status == 200",
				FailureExpression: "response.status == 200",
			},
			assertions: func(t *testing.T, res PromotionStepResult, err error) {
				require.ErrorContains(t, err, "HTTP response met failure criteria")
				require.True(t, isTerminal(err))
				require.Equal(t, kargoapi.PromotionPhaseFailed, res.Status)
			},
		},
		{
			name: "neither success nor failed",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusBadGateway)
			},
			cfg: HTTPConfig{
				SuccessExpression: "response.status == 200",
				FailureExpression: "response.status == 404",
			},
			assertions: func(t *testing.T, res PromotionStepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionPhaseRunning, res.Status)
			},
		},
	}

	h := &httpRequester{}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			srv := httptest.NewServer(testCase.handler)
			t.Cleanup(srv.Close)
			testCase.cfg.URL = srv.URL
			res, err := h.runPromotionStep(context.Background(), nil, testCase.cfg)
			testCase.assertions(t, res, err)
		})
	}
}

func Test_httpRequester_buildRequest(t *testing.T) {
	req, err := (&httpRequester{}).buildRequest(HTTPConfig{
		Method: "GET",
		URL:    "http://example.com",
		Headers: []HTTPHeader{{
			Name:  "Content-Type",
			Value: "application/json",
		}},
		QueryParams: []HTTPQueryParam{{
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
		cfg        HTTPConfig
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
			cfg: HTTPConfig{
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
			cfg: HTTPConfig{
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
		name       string
		resp       *http.Response
		assertions func(*testing.T, map[string]any, error)
	}{
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
				status, ok := statusAny.(int)
				require.True(t, ok)
				require.Equal(t, http.StatusOK, status)
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
				StatusCode: 200,
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
	}
	h := &httpRequester{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			env, err := h.buildExprEnv(testCase.resp)
			testCase.assertions(t, env, err)
		})
	}
}

func Test_httpRequester_wasRequestSuccessful(t *testing.T) {
	testCases := []struct {
		name       string
		cfg        HTTPConfig
		statusCode int
		assertions func(t *testing.T, success bool, err error)
	}{
		{
			name: "error compiling success expression",
			cfg:  HTTPConfig{SuccessExpression: "(1 + 2"},
			assertions: func(t *testing.T, _ bool, err error) {
				require.ErrorContains(t, err, "error compiling success expression")
			},
		},
		{
			name: "error evaluating success expression",
			cfg:  HTTPConfig{SuccessExpression: "invalid()"},
			assertions: func(t *testing.T, _ bool, err error) {
				require.ErrorContains(t, err, "error evaluating success expression")
			},
		},
		{
			name: "success expression evaluates to non-boolean",
			cfg:  HTTPConfig{SuccessExpression: `"foo"`},
			assertions: func(t *testing.T, _ bool, err error) {
				require.ErrorContains(t, err, "success expression did not evaluate to a boolean")
			},
		},
		{
			name: "success expression evaluates to true",
			cfg:  HTTPConfig{SuccessExpression: "true"},
			assertions: func(t *testing.T, success bool, err error) {
				require.NoError(t, err)
				require.True(t, success)
			},
		},
		{
			name: "success expression evaluates to false",
			cfg:  HTTPConfig{SuccessExpression: "false"},
			assertions: func(t *testing.T, success bool, err error) {
				require.NoError(t, err)
				require.False(t, success)
			},
		},
		{
			name: "no success expression, but failure expression",
			cfg:  HTTPConfig{FailureExpression: "true"},
			assertions: func(t *testing.T, success bool, err error) {
				require.NoError(t, err)
				require.False(t, success)
			},
		},
		{
			name:       "no success or failure expression; good status code",
			statusCode: 200,
			assertions: func(t *testing.T, success bool, err error) {
				require.NoError(t, err)
				require.True(t, success)
			},
		},
		{
			name:       "no success or failure expression; bad status code",
			statusCode: 404,
			assertions: func(t *testing.T, success bool, err error) {
				require.NoError(t, err)
				require.False(t, success)
			},
		},
	}
	h := &httpRequester{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			success, err := h.wasRequestSuccessful(testCase.cfg, testCase.statusCode, nil)
			testCase.assertions(t, success, err)
		})
	}
}

func Test_httpRequester_didRequestFail(t *testing.T) {
	testCases := []struct {
		name       string
		cfg        HTTPConfig
		statusCode int
		assertions func(t *testing.T, failed bool, err error)
	}{
		{
			name: "error compiling failure expression",
			cfg:  HTTPConfig{FailureExpression: "(1 + 2"},
			assertions: func(t *testing.T, _ bool, err error) {
				require.ErrorContains(t, err, "error compiling failure expression")
			},
		},
		{
			name: "error evaluating failure expression",
			cfg:  HTTPConfig{FailureExpression: "invalid()"},
			assertions: func(t *testing.T, _ bool, err error) {
				require.ErrorContains(t, err, "error evaluating failure expression")
			},
		},
		{
			name: "failure expression evaluates to non-boolean",
			cfg:  HTTPConfig{FailureExpression: `"foo"`},
			assertions: func(t *testing.T, _ bool, err error) {
				require.ErrorContains(t, err, "failure expression did not evaluate to a boolean")
			},
		},
		{
			name: "failure expression evaluates to true",
			cfg:  HTTPConfig{FailureExpression: "true"},
			assertions: func(t *testing.T, failed bool, err error) {
				require.NoError(t, err)
				require.True(t, failed)
			},
		},
		{
			name: "failure expression evaluates to false",
			cfg:  HTTPConfig{FailureExpression: "false"},
			assertions: func(t *testing.T, failed bool, err error) {
				require.NoError(t, err)
				require.False(t, failed)
			},
		},
		{
			name: "no failure expression, but success expression",
			cfg:  HTTPConfig{SuccessExpression: "true"},
			assertions: func(t *testing.T, failed bool, err error) {
				require.NoError(t, err)
				require.False(t, failed)
			},
		},
		{
			name:       "no success or failure expression; good status code",
			statusCode: 200,
			assertions: func(t *testing.T, failed bool, err error) {
				require.NoError(t, err)
				require.False(t, failed)
			},
		},
		{
			name:       "no success or failure expression; bad status code",
			statusCode: 404,
			assertions: func(t *testing.T, failed bool, err error) {
				require.NoError(t, err)
				require.True(t, failed)
			},
		},
	}
	h := &httpRequester{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			failed, err := h.didRequestFail(testCase.cfg, testCase.statusCode, nil)
			testCase.assertions(t, failed, err)
		})
	}
}

func Test_httpRequester_buildOutputs(t *testing.T) {
	testCases := []struct {
		name        string
		outputExprs []HTTPOutput
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
			outputExprs: []HTTPOutput{{
				Name:           "fake-output",
				FromExpression: "(1 + 2",
			}},
			assertions: func(t *testing.T, _ map[string]any, err error) {
				require.ErrorContains(t, err, "error compiling expression")
			},
		},
		{
			name: "error evaluating output expression",
			outputExprs: []HTTPOutput{{
				Name:           "fake-output",
				FromExpression: "invalid()",
			}},
			assertions: func(t *testing.T, _ map[string]any, err error) {
				require.ErrorContains(t, err, "error evaluating expression")
			},
		},
		{
			name: "success",
			outputExprs: []HTTPOutput{
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
