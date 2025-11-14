package builtin

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/io/fs"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const (
	stepKindHTTPDownload = "http-download"

	downloadTimeoutDefault = 5 * time.Minute
	downloadBufferSize     = 64 * 1024
	maxDownloadSize        = 100 << 20
)

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name:  stepKindHTTPDownload,
			Value: newHTTPDownloader,
		},
	)
}

// downloadBufferPool is a sync.Pool that provides byte slices for downloading
// files. It is used to reduce memory allocations during file downloads.
// The size of the byte slices is set to downloadBufferSize.
var downloadBufferPool = sync.Pool{
	New: func() any {
		return make([]byte, downloadBufferSize)
	},
}

// httpDownloader is an implementation of the promotion.StepRunner interface that
// downloads files from HTTP/HTTPS URLs.
type httpDownloader struct {
	schemaLoader gojsonschema.JSONLoader
}

// newHTTPDownloader returns an implementation of the promotion.StepRunner
// interface that downloads files from HTTP/HTTPS URLs.
func newHTTPDownloader(promotion.StepRunnerCapabilities) promotion.StepRunner {
	return &httpDownloader{
		schemaLoader: getConfigSchemaLoader(stepKindHTTPDownload),
	}
}

// Run implements the promotion.StepRunner interface.
func (d *httpDownloader) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	cfg, err := d.convert(stepCtx.Config)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &promotion.TerminalError{Err: err}
	}
	return d.run(ctx, stepCtx, cfg)
}

// convert validates httpDownloader configuration against a JSON schema and
// converts it into a builtin.HTTPDownloadConfig struct.
func (d *httpDownloader) convert(cfg promotion.Config) (builtin.HTTPDownloadConfig, error) {
	return validateAndConvert[builtin.HTTPDownloadConfig](d.schemaLoader, cfg, stepKindHTTPDownload)
}

// run executes the httpDownloader step with the provided configuration.
func (d *httpDownloader) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.HTTPDownloadConfig,
) (promotion.StepResult, error) {
	absOutPath, err := d.prepareOutputPath(stepCtx.WorkDir, cfg.OutPath, cfg.AllowOverwrite)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	resp, err := d.performHTTPRequest(cfg)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}
	defer resp.Body.Close()

	if err = d.validateResponse(resp); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed}, err
	}

	if err = d.downloadToFile(ctx, resp, absOutPath); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	return promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, nil
}

// prepareOutputPath validates and prepares the output path for the download.
func (d *httpDownloader) prepareOutputPath(workDir, outPath string, allowOverwrite bool) (string, error) {
	absOutPath, err := securejoin.SecureJoin(workDir, outPath)
	if err != nil {
		return "", fmt.Errorf("failed to join path %q: %w", outPath, err)
	}

	if err = d.checkFileOverwrite(absOutPath, outPath, allowOverwrite); err != nil {
		return "", err
	}

	destDir := filepath.Dir(absOutPath)
	if err = os.MkdirAll(destDir, 0o700); err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	return absOutPath, nil
}

// checkFileOverwrite validates file overwrite conditions.
func (d *httpDownloader) checkFileOverwrite(absOutPath, outPath string, allowOverwrite bool) error {
	if !allowOverwrite {
		if _, err := os.Stat(absOutPath); err == nil || !os.IsNotExist(err) {
			if err != nil {
				return fmt.Errorf("error checking destination file: %w", err)
			}
			return &promotion.TerminalError{
				Err: fmt.Errorf("file already exists at %s and overwrite is not allowed", outPath),
			}
		}
	}
	return nil
}

// performHTTPRequest executes the HTTP request and returns the response.
func (d *httpDownloader) performHTTPRequest(cfg builtin.HTTPDownloadConfig) (*http.Response, error) {
	req, err := d.buildRequest(cfg)
	if err != nil {
		return nil, &promotion.TerminalError{Err: fmt.Errorf("error building HTTP request: %w", err)}
	}

	client, err := d.buildHTTPClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP client: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending HTTP request: %w", err)
	}

	return resp, nil
}

// buildRequest constructs the HTTP request with headers and query parameters.
func (d *httpDownloader) buildRequest(cfg builtin.HTTPDownloadConfig) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, cfg.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP request: %w", err)
	}

	// Add custom headers
	for _, header := range cfg.Headers {
		req.Header.Add(header.Name, header.Value)
	}

	// Add query parameters
	if len(cfg.QueryParams) > 0 {
		q := req.URL.Query()
		for _, queryParam := range cfg.QueryParams {
			q.Add(queryParam.Name, queryParam.Value)
		}
		req.URL.RawQuery = q.Encode()
	}

	return req, nil
}

// buildHTTPClient creates an HTTP client with the specified configuration.
func (d *httpDownloader) buildHTTPClient(cfg builtin.HTTPDownloadConfig) (*http.Client, error) {
	httpTransport := cleanhttp.DefaultTransport()
	if cfg.InsecureSkipTLSVerify {
		httpTransport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, // nolint: gosec
		}
	}

	timeout := downloadTimeoutDefault
	if cfg.Timeout != "" {
		var err error
		if timeout, err = time.ParseDuration(cfg.Timeout); err != nil {
			return nil, fmt.Errorf("error parsing timeout: %w", err)
		}
	}

	return &http.Client{
		Transport: httpTransport,
		Timeout:   timeout,
	}, nil
}

// validateResponse checks the HTTP response status and content length.
func (d *httpDownloader) validateResponse(resp *http.Response) error {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP request failed with status %d", resp.StatusCode)
	}

	// Check file size limit using Content-Length header if available
	if contentLength := resp.ContentLength; contentLength > maxDownloadSize {
		return &promotion.TerminalError{
			Err: fmt.Errorf("download exceeds limit of %d bytes", maxDownloadSize),
		}
	}

	return nil
}

// downloadToFile downloads the response content to the specified file.
func (d *httpDownloader) downloadToFile(ctx context.Context, resp *http.Response, path string) error {
	logger := logging.LoggerFromContext(ctx)

	tempFile, tempPath, err := d.createTempFile(path)
	if err != nil {
		return err
	}

	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}()

	logger.Debug("starting HTTP download to temporary file", "url", resp.Request.URL.String())
	size, err := d.copyResponseToFile(ctx, resp, tempFile)
	if err != nil {
		return err
	}
	logger.Debug("HTTP download completed", "url", resp.Request.URL.String(), "size", size)

	if err = tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	if err = fs.SimpleAtomicMove(tempPath, path); err != nil {
		return fmt.Errorf("failed to move file to final destination: %w", err)
	}

	return nil
}

// createTempFile creates a temporary file in the same directory as the target.
func (d *httpDownloader) createTempFile(absOutPath string) (*os.File, string, error) {
	destDir := filepath.Dir(absOutPath)
	baseFile := filepath.Base(absOutPath)

	tempFile, err := os.CreateTemp(destDir, baseFile+".tmp")
	if err != nil {
		return nil, "", fmt.Errorf("failed to create temporary file: %w", err)
	}

	if err = tempFile.Chmod(0o600); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
		return nil, "", fmt.Errorf("failed to set permissions on temporary file: %w", err)
	}

	return tempFile, tempFile.Name(), nil
}

// copyResponseToFile copies response content to the file with size limits and
// context cancellation.
func (d *httpDownloader) copyResponseToFile(ctx context.Context, resp *http.Response, f *os.File) (int64, error) {
	limitedReader := io.LimitReader(resp.Body, maxDownloadSize)

	// Obtain a buffer from the pool
	buf := downloadBufferPool.Get().([]byte) // nolint:forcetypeassert
	defer func() {
		clear(buf)
		downloadBufferPool.Put(buf) // nolint:staticcheck
	}()

	// Stream data with context cancellation support
	var totalBytes int64
	for {
		select {
		case <-ctx.Done():
			return totalBytes, fmt.Errorf("download canceled: %w", ctx.Err())
		default:
		}

		var n int
		var err error
		if n, err = limitedReader.Read(buf); n > 0 {
			if _, writeErr := f.Write(buf[:n]); writeErr != nil {
				return totalBytes, fmt.Errorf("failed to write to file: %w", writeErr)
			}
			totalBytes += int64(n)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return totalBytes, fmt.Errorf("failed to read response body: %w", err)
		}
	}

	// If we read exactly the limit, check if there's more data
	if totalBytes == maxDownloadSize {
		if err := d.checkForAdditionalContent(resp); err != nil {
			return totalBytes, err
		}
	}

	return totalBytes, nil
}

// checkForAdditionalContent verifies if the response has more data beyond the
// size limit.
func (d *httpDownloader) checkForAdditionalContent(resp *http.Response) error {
	n, err := resp.Body.Read(make([]byte, 1))
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to check for additional content: %w", err)
	}

	if n > 0 {
		return fmt.Errorf("download exceeds limit of %d bytes", maxDownloadSize)
	}

	return nil
}
