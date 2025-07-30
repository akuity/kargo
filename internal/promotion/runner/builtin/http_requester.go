package builtin

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"time"

	"github.com/expr-lang/expr"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/io"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const (
	contentTypeHeader     = "Content-Type"
	contentTypeJSON       = "application/json"
	maxResponseBytes      = 2 << 20
	requestTimeoutDefault = 10 * time.Second
)

// httpRequester is an implementation of the promotion.StepRunner interface that
// sends an HTTP request and processes the response.
type httpRequester struct {
	schemaLoader gojsonschema.JSONLoader
}

// newHTTPRequester returns an implementation of the promotion.StepRunner
// interface that sends an HTTP request and processes the response.
func newHTTPRequester() promotion.StepRunner {
	r := &httpRequester{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the promotion.StepRunner interface.
func (h *httpRequester) Name() string {
	return "http"
}

// Run implements the promotion.StepRunner interface.
func (h *httpRequester) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	if err := h.validate(stepCtx.Config); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			&promotion.TerminalError{Err: err}
	}
	cfg, err := promotion.ConfigToStruct[builtin.HTTPConfig](stepCtx.Config)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			&promotion.TerminalError{Err: fmt.Errorf("could not convert config into http config: %w", err)}
	}
	return h.run(ctx, stepCtx, cfg)
}

// validate validates httpRequester configuration against a JSON schema.
func (h *httpRequester) validate(cfg promotion.Config) error {
	return validate(h.schemaLoader, gojsonschema.NewGoLoader(cfg), h.Name())
}

func (h *httpRequester) run(
	ctx context.Context,
	_ *promotion.StepContext,
	cfg builtin.HTTPConfig,
) (promotion.StepResult, error) {
	req, err := h.buildRequest(cfg)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			&promotion.TerminalError{Err: fmt.Errorf("error building HTTP request: %w", err)}
	}
	client, err := h.getClient(cfg)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			&promotion.TerminalError{Err: fmt.Errorf("error creating HTTP client: %w", err)}
	}
	resp, err := client.Do(req)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error sending HTTP request: %w", err)
	}
	defer resp.Body.Close()
	env, err := h.buildExprEnv(ctx, resp)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error building expression context from HTTP response: %w", err)
	}

	// Evaluate success and failure criteria
	successResult, err := h.evaluateSuccessCriteria(cfg, env)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error evaluating success criteria: %w", err)
	}

	failureResult, err := h.evaluateFailureCriteria(cfg, env)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error evaluating failure criteria: %w", err)
	}

	// Determine outcome based on criteria evaluation results
	switch {
	case failureResult != nil && *failureResult:
		// Failure criteria met: terminal failure
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed},
			&promotion.TerminalError{Err: fmt.Errorf(
				"HTTP (%d) response met failure criteria",
				resp.StatusCode,
			)}
	case successResult != nil && *successResult:
		// Success criteria met: success
		outputs, err := h.buildOutputs(cfg.Outputs, env)
		if err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf("error extracting outputs from HTTP response: %w", err)
		}
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusSucceeded,
			Output: outputs,
		}, nil
	case successResult == nil && failureResult == nil:
		// Both criteria undefined: fall back to response code logic
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			// 2xx: success
			outputs, err := h.buildOutputs(cfg.Outputs, env)
			if err != nil {
				return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
					fmt.Errorf("error extracting outputs from HTTP response: %w", err)
			}
			return promotion.StepResult{
				Status: kargoapi.PromotionStepStatusSucceeded,
				Output: outputs,
			}, nil
		}
		// Non-2xx: retried failure (not terminal)
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed}, nil
	default:
		// All other cases: running (retried)
		// This includes:
		// - Success unmet, failure undefined
		// - Success undefined, failure unmet
		// - Success unmet, failure unmet
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusRunning}, nil
	}
}

// evaluateSuccessCriteria evaluates the success criteria expression if defined.
// If the expression is not defined, it returns nil.
func (h *httpRequester) evaluateSuccessCriteria(
	cfg builtin.HTTPConfig,
	env map[string]any,
) (*bool, error) {
	if cfg.SuccessExpression == "" {
		return nil, nil
	}

	program, err := expr.Compile(cfg.SuccessExpression)
	if err != nil {
		return nil, &promotion.TerminalError{
			Err: fmt.Errorf("error compiling success expression %q: %w", cfg.SuccessExpression, err),
		}
	}
	successAny, err := expr.Run(program, env)
	if err != nil {
		return nil, fmt.Errorf("error evaluating success expression %q: %w", cfg.SuccessExpression, err)
	}
	if success, ok := successAny.(bool); ok {
		return &success, nil
	}
	return nil, fmt.Errorf(
		"success expression %q did not evaluate to a boolean (got %T)",
		cfg.SuccessExpression, successAny,
	)
}

// evaluateFailureCriteria evaluates the failure criteria expression if defined.
// If the expression is not defined, it returns nil as the result.
func (h *httpRequester) evaluateFailureCriteria(
	cfg builtin.HTTPConfig,
	env map[string]any,
) (*bool, error) {
	if cfg.FailureExpression == "" {
		return nil, nil
	}

	program, err := expr.Compile(cfg.FailureExpression)
	if err != nil {
		return nil, &promotion.TerminalError{
			Err: fmt.Errorf("error compiling failure expression %q: %w", cfg.FailureExpression, err),
		}
	}
	failureAny, err := expr.Run(program, env)
	if err != nil {
		return nil, fmt.Errorf("error evaluating failure expression %q: %w", cfg.FailureExpression, err)
	}
	if failure, ok := failureAny.(bool); ok {
		return &failure, nil
	}
	return nil, fmt.Errorf(
		"failure expression %q did not evaluate to a boolean (got %T)",
		cfg.FailureExpression, failureAny,
	)
}

func (h *httpRequester) buildRequest(cfg builtin.HTTPConfig) (*http.Request, error) {
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

func (h *httpRequester) getClient(cfg builtin.HTTPConfig) (*http.Client, error) {
	httpTransport := cleanhttp.DefaultTransport()
	if cfg.InsecureSkipTLSVerify {
		httpTransport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, // nolint: gosec
		}
	}
	timeout := requestTimeoutDefault
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

func (h *httpRequester) buildExprEnv(
	ctx context.Context,
	resp *http.Response,
) (map[string]any, error) {
	// Early check of Content-Length if available
	if contentLength := resp.ContentLength; contentLength > maxResponseBytes {
		return nil, fmt.Errorf("response body size %d exceeds limit of %d bytes", contentLength, maxResponseBytes)
	}

	// Read the response body up to the maximum allowed size
	bodyBytes, err := io.LimitRead(resp.Body, maxResponseBytes)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	// TODO(hidde): It has proven to be difficult to figure out why a HTTP step
	// fails or is not working as expected. To remediate this, we log the
	// response body and headers at trace level. This is a temporary solution
	// until we have a better way to present this information to the user, e.g.
	// as part of the step output or error message.
	logging.LoggerFromContext(ctx).Trace(
		"HTTP request response",
		"status", resp.StatusCode,
		"header", resp.Header,
		"body", string(bodyBytes),
	)

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
	contentType, _, _ := mime.ParseMediaType(resp.Header.Get(contentTypeHeader))
	if len(bodyBytes) > 0 && (contentType == contentTypeJSON || json.Valid(bodyBytes)) {
		var parsedBody any
		if err := json.Unmarshal(bodyBytes, &parsedBody); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		// Unmarshal into map[string]any or []any
		switch parsedBody.(type) {
		case map[string]any, []any:
			env["response"].(map[string]any)["body"] = parsedBody // nolint: forcetypeassert
		default:
			return nil, fmt.Errorf("unexpected type when unmarshaling response: %T", parsedBody)
		}
	}

	return env, nil
}

func (h *httpRequester) buildOutputs(
	outputExprs []builtin.HTTPOutput,
	env map[string]any,
) (map[string]any, error) {
	outputs := make(map[string]any, len(outputExprs))
	for _, output := range outputExprs {
		program, err := expr.Compile(output.FromExpression)
		if err != nil {
			return nil, &promotion.TerminalError{
				Err: fmt.Errorf("error compiling output expression %q: %w", output.Name, err),
			}
		}
		if outputs[output.Name], err = expr.Run(program, env); err != nil {
			return nil, fmt.Errorf("error evaluating output expression %q: %w", output.Name, err)
		}
	}
	return outputs, nil
}
