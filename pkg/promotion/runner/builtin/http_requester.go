package builtin

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/expr-lang/expr"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/xeipuuv/gojsonschema"
	"sigs.k8s.io/yaml"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	kargoio "github.com/akuity/kargo/pkg/io"
	"github.com/akuity/kargo/pkg/io/fs"
	"github.com/akuity/kargo/pkg/logging"
	kargonet "github.com/akuity/kargo/pkg/net"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const (
	stepKindHTTP = "http"

	maxResponseBytes      = 2 << 20
	requestTimeoutDefault = 10 * time.Second

	// downloadModeTimeoutDefault is the default request timeout when outPath
	// is set. Downloads need longer than the 10s default for plain requests,
	// but not the 5m the http-download step used.
	downloadModeTimeoutDefault = 1 * time.Minute

	// httpPollIntervalDefault is the suggested interval at which the http step
	// re-polls its URL while waiting for its success or failure criteria to be
	// met, absent an explicitly configured pollInterval.
	httpPollIntervalDefault = 30 * time.Second

	contentTypeHeader = "Content-Type"

	contentTypeJSON      = "application/json"
	contentTypeYAML      = "application/yaml"
	contentTypeYAMLAlt   = "text/yaml"
	contentTypeYAMLX     = "application/x-yaml"
	contentTypeTextPlain = "text/plain"
)

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name:  stepKindHTTP,
			Value: newHTTPRequester,
		},
	)
}

// httpRequester is an implementation of the promotion.StepRunner interface that
// sends an HTTP request and processes the response.
type httpRequester struct {
	schemaLoader gojsonschema.JSONLoader
}

// newHTTPRequester returns an implementation of the promotion.StepRunner
// interface that sends an HTTP request and processes the response.
func newHTTPRequester(promotion.StepRunnerCapabilities) promotion.StepRunner {
	return &httpRequester{schemaLoader: getConfigSchemaLoader(stepKindHTTP)}

}

// Run implements the promotion.StepRunner interface.
func (h *httpRequester) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	cfg, err := h.convert(stepCtx.Config)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &promotion.TerminalError{Err: err}
	}
	return h.run(ctx, stepCtx, cfg)
}

// convert validates httpRequester configuration against a JSON schema and
// converts it into a builtin.HTTPConfig struct.
func (h *httpRequester) convert(cfg promotion.Config) (builtin.HTTPConfig, error) {
	return validateAndConvert[builtin.HTTPConfig](h.schemaLoader, cfg, stepKindHTTP)
}

func (h *httpRequester) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.HTTPConfig,
) (promotion.StepResult, error) {
	dl, err := h.prepareDownload(stepCtx, cfg)
	if err != nil {
		var termErr *promotion.TerminalError
		if errors.As(err, &termErr) {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed}, err
		}
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}
	req, err := h.buildRequest(ctx, stepCtx, cfg)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			&promotion.TerminalError{Err: fmt.Errorf("error building HTTP request: %w", err)}
	}
	client, err := h.getClient(cfg)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			&promotion.TerminalError{Err: fmt.Errorf("error creating HTTP client: %w", err)}
	}
	// #nosec G704 -- The client is using a custom dialer that mitigates the worst
	// practical risks of SSRF by refusing to dial link-local addresses.
	resp, err := client.Do(req)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error sending HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if dl != nil {
		return h.runDownload(ctx, cfg, resp, dl)
	}

	env, err := h.buildExprEnv(ctx, resp, cfg.ResponseContentType)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error building expression context from HTTP response: %w", err)
	}

	return h.evaluateOutcome(ctx, cfg, resp, env, nil)
}

// runDownload implements the http step's download behavior when outPath is
// set: the response body is streamed to a temporary file (capped at
// maxDownloadSize) and moved to outPath only if the step succeeds.
func (h *httpRequester) runDownload(
	ctx context.Context,
	cfg builtin.HTTPConfig,
	resp *http.Response,
	dl *httpDownload,
) (promotion.StepResult, error) {
	defer dl.discard()
	env, err := h.buildDownloadEnv(ctx, resp, cfg.ResponseContentType, dl)
	if err != nil {
		var termErr *promotion.TerminalError
		if errors.As(err, &termErr) {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed}, err
		}
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error building expression context from HTTP response: %w", err)
	}
	return h.evaluateOutcome(ctx, cfg, resp, env, dl)
}

// evaluateOutcome evaluates the success and failure criteria against the
// response env and maps the result to a step result, exactly as the http step
// always has. When a download is in progress, its temporary file is moved to
// outPath only on success and discarded otherwise.
func (h *httpRequester) evaluateOutcome(
	ctx context.Context,
	cfg builtin.HTTPConfig,
	resp *http.Response,
	env map[string]any,
	dl *httpDownload,
) (promotion.StepResult, error) {
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
		// Failure criteria met: terminal failure. Optionally enrich the error
		// with a message extracted from the response.
		errorMessage, err := h.extractErrorMessageFromResponse(ctx, cfg, env)
		if err != nil {
			// Only a misconfigured (uncompilable) expression is surfaced here,
			// consistent with the success and failure expressions.
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
		}
		if errorMessage == "" {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed},
				&promotion.TerminalError{Err: fmt.Errorf(
					"HTTP (%d) response met failure criteria",
					resp.StatusCode,
				)}
		}

		return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed},
			&promotion.TerminalError{Err: fmt.Errorf(
				"HTTP (%d) response met failure criteria: %q",
				resp.StatusCode,
				errorMessage,
			)}
	case successResult != nil && *successResult:
		// Success criteria met: success
		outputs, err := h.buildOutputs(cfg.Outputs, env)
		if err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf("error extracting outputs from HTTP response: %w", err)
		}
		if err := dl.complete(); err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
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
			if err := dl.complete(); err != nil {
				return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
			}
			return promotion.StepResult{
				Status: kargoapi.PromotionStepStatusSucceeded,
				Output: outputs,
			}, nil
		}
		// Non-2xx: retried failure (not terminal)
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed}, nil
	default:
		// All other cases: running (polled)
		// This includes:
		// - Success unmet, failure undefined
		// - Success undefined, failure unmet
		// - Success unmet, failure unmet
		pollInterval, err := resolvePollInterval(cfg.PollInterval, httpPollIntervalDefault)
		if err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
		}
		return promotion.StepResult{
			Status:     kargoapi.PromotionStepStatusRunning,
			RetryAfter: &pollInterval,
		}, nil
	}
}

// httpDownload tracks the temporary file a response body is streamed to when
// the http step's outPath is set. The file is moved to its final destination
// only if the step succeeds; it is discarded on every other outcome.
type httpDownload struct {
	tempPath   string
	absOutPath string
	moved      bool
}

// discard removes the temporary file unless it was already moved to its
// destination. It is safe to call on a nil receiver.
func (d *httpDownload) discard() {
	if d == nil || d.moved {
		return
	}
	_ = os.Remove(d.tempPath)
}

// complete atomically moves the temporary file to its final destination. It
// is a no-op on a nil receiver.
func (d *httpDownload) complete() error {
	if d == nil {
		return nil
	}
	if err := fs.SimpleAtomicMove(d.tempPath, d.absOutPath); err != nil {
		return fmt.Errorf("failed to move downloaded file to destination: %w", err)
	}
	d.moved = true
	return nil
}

// prepareDownload validates the download destination when outPath is set. A
// file that already exists while allowOverwrite is false fails terminally
// before any request is sent. It returns nil when outPath is unset.
func (h *httpRequester) prepareDownload(
	stepCtx *promotion.StepContext,
	cfg builtin.HTTPConfig,
) (*httpDownload, error) {
	if cfg.OutPath == "" {
		return nil, nil
	}
	if stepCtx == nil {
		return nil, fmt.Errorf("cannot download to outPath %q without a step context", cfg.OutPath)
	}
	absOutPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.OutPath)
	if err != nil {
		return nil, fmt.Errorf("failed to join path %q: %w", cfg.OutPath, err)
	}
	if !cfg.AllowOverwrite {
		if _, err := os.Stat(absOutPath); err == nil || !os.IsNotExist(err) {
			if err != nil {
				return nil, fmt.Errorf("error checking destination file: %w", err)
			}
			return nil, &promotion.TerminalError{Err: fmt.Errorf(
				"file already exists at %s and overwrite is not allowed", cfg.OutPath,
			)}
		}
	}
	if err := os.MkdirAll(filepath.Dir(absOutPath), 0o700); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}
	return &httpDownload{absOutPath: absOutPath}, nil
}

// buildDownloadEnv streams the response body to a temporary file capped at
// maxDownloadSize and builds the expression env from it. Exceeding the cap is
// a terminal failure: retrying would download the same oversized body again.
// Bodies larger than maxResponseBytes are not parsed into response.body;
// status and headers are always available, so criteria for large downloads
// should use those.
func (h *httpRequester) buildDownloadEnv(
	ctx context.Context,
	resp *http.Response,
	contentType string,
	dl *httpDownload,
) (map[string]any, error) {
	// Fail fast on a declared size over the cap, before downloading anything.
	if resp.ContentLength > maxDownloadSize {
		return nil, &promotion.TerminalError{Err: fmt.Errorf(
			"response exceeds download limit of %d bytes", maxDownloadSize,
		)}
	}

	tempPath, size, err := streamResponseToTempFile(
		ctx,
		resp.Body,
		filepath.Dir(dl.absOutPath),
		filepath.Base(dl.absOutPath),
	)
	if err != nil {
		return nil, err
	}
	dl.tempPath = tempPath

	var bodyBytes []byte
	if size > maxResponseBytes {
		logging.LoggerFromContext(ctx).Debug(
			"response body exceeds 2 MiB; leaving response.body empty",
			"size", size,
		)
	} else {
		// #nosec G304 -- tempPath is a temp file this step just created.
		if bodyBytes, err = os.ReadFile(tempPath); err != nil {
			return nil, fmt.Errorf("reading downloaded response body: %w", err)
		}
	}

	return h.buildExprEnvFromBytes(
		ctx,
		resp.StatusCode,
		resp.Header,
		bodyBytes,
		contentType,
	)
}

// streamResponseToTempFile streams r to a temporary file in dir, capped at
// maxDownloadSize. Hitting the cap is a terminal failure: retrying would
// download the same oversized body again. The temp file is removed on any
// error; on success the caller owns it.
func streamResponseToTempFile(
	ctx context.Context,
	r io.Reader,
	dir string,
	prefix string,
) (string, int64, error) {
	tempFile, err := os.CreateTemp(dir, prefix+".tmp")
	if err != nil {
		return "", 0, fmt.Errorf("failed to create temporary file: %w", err)
	}
	tempPath := tempFile.Name()
	keep := false
	defer func() {
		_ = tempFile.Close()
		if !keep {
			_ = os.Remove(tempPath)
		}
	}()
	if err = tempFile.Chmod(0o600); err != nil {
		return "", 0, fmt.Errorf("failed to set permissions on temporary file: %w", err)
	}

	limitedReader := io.LimitReader(r, maxDownloadSize)
	buf := downloadBufferPool.Get().([]byte) // nolint:forcetypeassert
	defer func() {
		clear(buf)
		downloadBufferPool.Put(buf) // nolint:staticcheck
	}()

	var size int64
	for {
		select {
		case <-ctx.Done():
			return "", 0, fmt.Errorf("download canceled: %w", ctx.Err())
		default:
		}
		n, readErr := limitedReader.Read(buf)
		if n > 0 {
			if _, writeErr := tempFile.Write(buf[:n]); writeErr != nil {
				return "", 0, fmt.Errorf("failed to write to file: %w", writeErr)
			}
			size += int64(n)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", 0, fmt.Errorf("failed to read response body: %w", readErr)
		}
	}

	if err = tempFile.Close(); err != nil {
		return "", 0, fmt.Errorf("failed to close temporary file: %w", err)
	}

	if size == maxDownloadSize {
		// The body might be larger than the cap; probe for one more byte.
		var probe [1]byte
		n, probeErr := r.Read(probe[:])
		if probeErr != nil && probeErr != io.EOF {
			return "", 0, fmt.Errorf("failed to check for additional content: %w", probeErr)
		}
		if n > 0 {
			return "", 0, &promotion.TerminalError{Err: fmt.Errorf(
				"response exceeds download limit of %d bytes", maxDownloadSize,
			)}
		}
	}

	keep = true
	return tempPath, size, nil
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

// extractErrorMessageFromResponse evaluates the error expression, if defined,
// returning the message it extracts from the response. The cases are
// distinguished as follows:
//
//   - No expression defined: returns an empty string and no error.
//   - Compilation error: returns a TerminalError, consistent with the success
//     and failure expressions, since an uncompilable expression is a
//     misconfiguration the user must fix.
//   - Evaluation error: best-effort, since the response may not be shaped as the
//     expression expects. Logs at debug level and returns an empty string so the
//     caller falls back to the default error message.
//   - Nil or non-string result: expected (e.g. nil coalescing that finds no
//     matching field). Returns an empty string so the caller falls back.
func (h *httpRequester) extractErrorMessageFromResponse(
	ctx context.Context,
	cfg builtin.HTTPConfig,
	env map[string]any,
) (string, error) {
	if cfg.ErrorExpression == "" {
		return "", nil
	}

	program, err := expr.Compile(cfg.ErrorExpression)
	if err != nil {
		return "", &promotion.TerminalError{
			Err: fmt.Errorf("error compiling error expression %q: %w", cfg.ErrorExpression, err),
		}
	}
	errorAny, err := expr.Run(program, env)
	if err != nil {
		logging.LoggerFromContext(ctx).Debug(
			"error evaluating error expression for HTTP response",
			"expression", cfg.ErrorExpression,
			"error", err,
		)
		return "", nil
	}
	errorMessage, _ := errorAny.(string)
	return errorMessage, nil
}

func (h *httpRequester) buildRequest(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.HTTPConfig,
) (*http.Request, error) {
	method := cfg.Method
	if method == "" {
		method = http.MethodGet
	}
	var bodyReader io.Reader = strings.NewReader(cfg.Body)
	if cfg.BodyFromFile != "" {
		if stepCtx == nil {
			return nil, fmt.Errorf("cannot read bodyFromFile %q without a step context", cfg.BodyFromFile)
		}
		bodyPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.BodyFromFile)
		if err != nil {
			return nil, fmt.Errorf("could not secure join bodyFromFile %q: %w", cfg.BodyFromFile, err)
		}
		bodyFile, err := os.Open(bodyPath)
		if err != nil {
			return nil, fmt.Errorf("could not read bodyFromFile %q: %w", cfg.BodyFromFile, err)
		}
		bodyBytes, err := kargoio.LimitRead(bodyFile, maxResponseBytes)
		if err != nil {
			return nil, fmt.Errorf("could not read bodyFromFile %q: %w", cfg.BodyFromFile, err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}
	req, err := http.NewRequestWithContext(
		ctx,
		method,
		cfg.URL,
		bodyReader,
	)
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
	kargonet.HardenTransport(httpTransport, nil)
	if cfg.InsecureSkipTLSVerify {
		httpTransport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, // nolint: gosec
		}
	}
	timeout := requestTimeoutDefault
	if cfg.OutPath != "" {
		timeout = downloadModeTimeoutDefault
	}
	if cfg.Timeout != "" {
		var err error
		if timeout, err = time.ParseDuration(cfg.Timeout); err != nil {
			// Input is validated, so this really should not happen
			return nil, fmt.Errorf("error parsing timeout: %w", err)
		}
	}
	if cfg.Proxy != "" {
		proxyURL, err := url.Parse(cfg.Proxy)
		if err != nil {
			return nil, fmt.Errorf("error parsing proxy URL: %w", err)
		}
		httpTransport.Proxy = http.ProxyURL(proxyURL)
	}
	return &http.Client{
		Transport: httpTransport,
		Timeout:   timeout,
	}, nil
}

func (h *httpRequester) buildExprEnv(
	ctx context.Context,
	resp *http.Response,
	contentType string,
) (map[string]any, error) {
	// Early check of Content-Length if available
	if contentLength := resp.ContentLength; contentLength > maxResponseBytes {
		return nil, fmt.Errorf("response body size %d exceeds limit of %d bytes", contentLength, maxResponseBytes)
	}

	// Read the response body up to the maximum allowed size
	bodyBytes, err := kargoio.LimitRead(resp.Body, maxResponseBytes)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	return h.buildExprEnvFromBytes(
		ctx,
		resp.StatusCode,
		resp.Header,
		bodyBytes,
		contentType,
	)
}

// buildExprEnvFromBytes builds the expression env from an already-read
// response body. It is shared by plain requests and downloads so the parsing
// rules stay identical. A nil or empty bodyBytes leaves response.body as an
// empty map.
func (h *httpRequester) buildExprEnvFromBytes(
	ctx context.Context,
	statusCode int,
	header http.Header,
	bodyBytes []byte,
	contentType string,
) (map[string]any, error) {
	// TODO(hidde): It has proven to be difficult to figure out why a HTTP step
	// fails or is not working as expected. To remediate this, we log the
	// response body and headers at trace level. This is a temporary solution
	// until we have a better way to present this information to the user, e.g.
	// as part of the step output or error message.
	logging.LoggerFromContext(ctx).Trace(
		"HTTP request response",
		"status", statusCode,
		"header", header,
		"body", string(bodyBytes),
	)

	response := map[string]any{
		// TODO(krancour): Casting as an int64 is a short-term fix here because
		// deep copy of the output map will panic if any value is an int. This is
		// a near-term fix and a better solution will be PR'ed soon.
		"status":  int64(statusCode),
		"header":  header.Get,
		"headers": header,
		"body":    map[string]any{},
	}

	if contentType == "" {
		contentType, _, _ = mime.ParseMediaType(header.Get(contentTypeHeader))
	}

	if len(bodyBytes) > 0 {
		parseMode := h.determineResponseParseMode(contentType)

		switch parseMode {
		case httpParseModeJSON:
			if contentType != contentTypeJSON {
				if !json.Valid(bodyBytes) {
					logging.LoggerFromContext(ctx).Debug(
						"unrecognized content type is not valid JSON, ignoring response body",
						"contentType", contentType,
					)
					break
				}
			}
			var parsedBody any
			if err := json.Unmarshal(bodyBytes, &parsedBody); err != nil {
				return nil, fmt.Errorf("failed to parse JSON response: %w", err)
			}
			response["body"] = parsedBody
		case httpParseModeYAML:
			var parsedBody any
			if err := yaml.Unmarshal(bodyBytes, &parsedBody); err != nil {
				return nil, fmt.Errorf("failed to parse YAML response: %w", err)
			}
			response["body"] = parsedBody
		case httpParseModeText:
			response["body"] = string(bodyBytes)
		}
	}

	return map[string]any{
		"response": response,
	}, nil
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

// httpResponseParseMode identifies how an HTTP response should be parsed.
type httpResponseParseMode string

const (
	httpParseModeJSON httpResponseParseMode = "JSON"
	httpParseModeYAML httpResponseParseMode = "YAML"
	httpParseModeText httpResponseParseMode = "text"
)

// determineResponseParseMode determines how to parse the response body based on
// the provided MIME media type.
func (h *httpRequester) determineResponseParseMode(contentType string) httpResponseParseMode {
	switch {
	case strings.EqualFold(contentType, contentTypeJSON):
		return httpParseModeJSON
	case strings.EqualFold(contentType, contentTypeYAML),
		strings.EqualFold(contentType, contentTypeYAMLAlt),
		strings.EqualFold(contentType, contentTypeYAMLX):
		return httpParseModeYAML
	case strings.EqualFold(contentType, contentTypeTextPlain):
		return httpParseModeText
	default:
		// Fallback: try to parse as JSON
		return httpParseModeJSON
	}
}
