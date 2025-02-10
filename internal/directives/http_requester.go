package directives

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/expr-lang/expr"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

const (
	contentTypeHeader = "Content-Type"
	contentTypeJSON   = "application/json"
)

func init() {
	builtins.RegisterPromotionStepRunner(newHTTPRequester(), nil)
}

// httpRequester is an implementation of the PromotionStepRunner interface that
// sends an HTTP request and processes the response.
type httpRequester struct {
	schemaLoader gojsonschema.JSONLoader
}

// newHTTPRequester returns an implementation of the PromotionStepRunner
// interface that sends an HTTP request and processes the response.
func newHTTPRequester() PromotionStepRunner {
	r := &httpRequester{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the PromotionStepRunner interface.
func (h *httpRequester) Name() string {
	return "http"
}

func (h *httpRequester) RunPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
) (PromotionStepResult, error) {
	if err := h.validate(stepCtx.Config); err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, err
	}
	cfg, err := ConfigToStruct[HTTPConfig](stepCtx.Config)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("could not convert config into http config: %w", err)
	}
	return h.runPromotionStep(ctx, stepCtx, cfg)
}

// validate validates httpRequester configuration against a JSON schema.
func (h *httpRequester) validate(cfg Config) error {
	return validate(h.schemaLoader, gojsonschema.NewGoLoader(cfg), h.Name())
}

func (h *httpRequester) runPromotionStep(
	_ context.Context,
	_ *PromotionStepContext,
	cfg HTTPConfig,
) (PromotionStepResult, error) {
	req, err := h.buildRequest(cfg)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error building HTTP request: %w", err)
	}
	client, err := h.getClient(cfg)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error creating HTTP client: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error sending HTTP request: %w", err)
	}
	defer resp.Body.Close()
	env, err := h.buildExprEnv(resp)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error building expression context from HTTP response: %w", err)
	}
	success, err := h.wasRequestSuccessful(cfg, resp.StatusCode, env)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error evaluating success criteria: %w", err)
	}
	failure, err := h.didRequestFail(cfg, resp.StatusCode, env)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error evaluating failure criteria: %w", err)
	}
	switch {
	case success && !failure:
		outputs, err := h.buildOutputs(cfg.Outputs, env)
		if err != nil {
			return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
				fmt.Errorf("error extracting outputs from HTTP response: %w", err)
		}
		return PromotionStepResult{
			Status: kargoapi.PromotionPhaseSucceeded,
			Output: outputs,
		}, nil
	case failure:
		return PromotionStepResult{Status: kargoapi.PromotionPhaseFailed},
			&terminalError{err: errors.New("HTTP response met failure criteria")}
	default:
		return PromotionStepResult{Status: kargoapi.PromotionPhaseRunning}, nil
	}
}

func (h *httpRequester) buildRequest(cfg HTTPConfig) (*http.Request, error) {
	method := cfg.Method
	if method == "" {
		method = http.MethodGet
	}
	req, err := http.NewRequest(method, cfg.URL, bytes.NewBufferString(cfg.Body))
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP request: %w", err)
	}
	for _, header := range cfg.Headers {
		req.Header.Add(header.Name, header.Value)
	}
	if len(cfg.QueryParams) > 0 {
		q := req.URL.Query()
		for _, queryParam := range cfg.QueryParams {
			q.Add(queryParam.Name, queryParam.Value)
		}
		req.URL.RawQuery = q.Encode()
	}
	return req, nil
}

func (h *httpRequester) getClient(cfg HTTPConfig) (*http.Client, error) {
	httpTransport := cleanhttp.DefaultTransport()
	if cfg.InsecureSkipTLSVerify {
		httpTransport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, // nolint: gosec
		}
	}
	timeout := 10 * time.Second
	if cfg.Timeout != "" {
		var err error
		if timeout, err = time.ParseDuration(cfg.Timeout); err != nil {
			// Input is validated, so this really should not happen
			return nil, fmt.Errorf("error parsing timeout: %w", err)
		}
	}
	return &http.Client{
		Transport: httpTransport,
		Timeout:   timeout,
	}, nil
}

func (h *httpRequester) buildExprEnv(resp *http.Response) (map[string]any, error) {
	const maxBytes = 2 << 20

	// Early check of Content-Length if available
	if contentLength := resp.ContentLength; contentLength > maxBytes {
		return nil, fmt.Errorf("response body size %d exceeds limit of %d bytes", contentLength, maxBytes)
	}

	// Create a limited reader that will stop after max bytes
	bodyReader := io.LimitReader(resp.Body, maxBytes)

	// Read as far as we are allowed to
	bodyBytes, err := io.ReadAll(bodyReader)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	// If we read exactly the maximum, the body might be larger
	if len(bodyBytes) == maxBytes {
		// Try to read one more byte
		buf := make([]byte, 1)
		var n int
		if n, err = resp.Body.Read(buf); err != nil && err != io.EOF {
			return nil, fmt.Errorf("checking for additional content: %w", err)
		}
		if n > 0 || err != io.EOF {
			return nil, fmt.Errorf("response body exceeds maximum size of %d bytes", maxBytes)
		}
	}
	env := map[string]any{
		"response": map[string]any{
			// TODO(krancour): Casting as an int64 is a short-term fix here because
			// deep copy of the output map will panic if any value is an int. This is
			// a near-term fix and a better solution will be PR'ed soon.
			"status":  int64(resp.StatusCode),
			"header":  resp.Header.Get,
			"headers": resp.Header,
			"body":    map[string]any{},
		},
	}
	contentType := strings.TrimSpace(
		strings.Split(resp.Header.Get(contentTypeHeader), ";")[0],
	)
	if len(bodyBytes) > 0 && contentType == contentTypeJSON {
		body := map[string]any{}
		if err = json.Unmarshal(bodyBytes, &body); err != nil {
			return nil, err
		}
		env["response"].(map[string]any)["body"] = body // nolint: forcetypeassert
	}
	return env, nil
}

func (h *httpRequester) wasRequestSuccessful(
	cfg HTTPConfig,
	statusCode int,
	env map[string]any,
) (bool, error) {
	switch {
	case cfg.SuccessExpression != "":
		program, err := expr.Compile(cfg.SuccessExpression)
		if err != nil {
			return false, fmt.Errorf("error compiling success expression: %w", err)
		}
		successAny, err := expr.Run(program, env)
		if err != nil {
			return false, fmt.Errorf("error evaluating success expression: %w", err)
		}
		if success, ok := successAny.(bool); ok {
			return success, nil
		}
		return false, fmt.Errorf("success expression did not evaluate to a boolean")
	case cfg.FailureExpression != "":
		failure, err := h.didRequestFail(cfg, statusCode, env)
		if err != nil {
			return false, err
		}
		return !failure, nil
	default:
		// The client automatically follows redirects, so we consider only
		// 2xx status codes successful.
		return statusCode >= 200 && statusCode < 300, nil
	}
}

func (h *httpRequester) didRequestFail(
	cfg HTTPConfig,
	statusCode int,
	env map[string]any,
) (bool, error) {
	switch {
	case cfg.FailureExpression != "":
		program, err := expr.Compile(cfg.FailureExpression)
		if err != nil {
			return true, fmt.Errorf("error compiling failure expression: %w", err)
		}
		failureAny, err := expr.Run(program, env)
		if err != nil {
			return true, fmt.Errorf("error evaluating failure expression: %w", err)
		}
		if failure, ok := failureAny.(bool); ok {
			return failure, nil
		}
		return true, fmt.Errorf("failure expression did not evaluate to a boolean")
	case cfg.SuccessExpression != "":
		success, err := h.wasRequestSuccessful(cfg, statusCode, env)
		if err != nil {
			return true, err
		}
		return !success, nil
	default:
		// The client automatically follows redirects, so we consider any
		// non-2xx status code a failure.
		return statusCode < 200 || statusCode >= 300, nil
	}
}

func (h *httpRequester) buildOutputs(
	outputExprs []HTTPOutput,
	env map[string]any,
) (map[string]any, error) {
	outputs := make(map[string]any, len(outputExprs))
	for _, output := range outputExprs {
		program, err := expr.Compile(output.FromExpression)
		if err != nil {
			return nil, fmt.Errorf("error compiling expression: %w", err)
		}
		if outputs[output.Name], err = expr.Run(program, env); err != nil {
			return nil, fmt.Errorf("error evaluating expression: %w", err)
		}
	}
	return outputs, nil
}
