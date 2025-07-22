package builtin

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/io/fs"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const (
	downloadTimeoutDefault = 5 * time.Minute
	downloadBufferSize     = 64 * 1024
	maxDownloadSize        = 100 << 20
)

// httpDownloader is an implementation of the promotion.StepRunner interface that
// downloads files from HTTP/HTTPS URLs.
type httpDownloader struct {
	schemaLoader gojsonschema.JSONLoader
}

// newHTTPDownloader returns an implementation of the promotion.StepRunner
// interface that downloads files from HTTP/HTTPS URLs.
func newHTTPDownloader() promotion.StepRunner {
	d := &httpDownloader{}
	d.schemaLoader = getConfigSchemaLoader(d.Name())
	return d
}

// Name implements the promotion.StepRunner interface.
func (d *httpDownloader) Name() string {
	return "http-download"
}

// Run implements the promotion.StepRunner interface.
func (d *httpDownloader) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	if err := d.validate(stepCtx.Config); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			&promotion.TerminalError{Err: err}
	}
	cfg, err := promotion.ConfigToStruct[builtin.HTTPDownloadConfig](stepCtx.Config)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			&promotion.TerminalError{Err: fmt.Errorf("could not convert config into download config: %w", err)}
	}
	return d.run(ctx, stepCtx, cfg)
}

// validate validates httpDownloader configuration against a JSON schema.
func (d *httpDownloader) validate(cfg promotion.Config) error {
	return validate(d.schemaLoader, gojsonschema.NewGoLoader(cfg), d.Name())
}

// run executes the HTTP download step, downloading a file from the specified
// URL and saving it to the specified output path. It handles file size limits,
// overwriting existing files, and context cancellation.
func (d *httpDownloader) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.HTTPDownloadConfig,
) (promotion.StepResult, error) {
	absOutPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.OutPath)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to join path %q: %w", cfg.OutPath, err)
	}

	destDir := filepath.Dir(absOutPath)
	if err = os.MkdirAll(destDir, 0o700); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to create destination directory: %w", err)
	}

	if !cfg.AllowOverwrite {
		if _, err = os.Stat(absOutPath); err == nil || !os.IsNotExist(err) {
			if err != nil {
				return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
					fmt.Errorf("error checking destination file: %w", err)
			}

			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				&promotion.TerminalError{Err: fmt.Errorf("file already exists at %s and overwrite is not allowed", cfg.OutPath)}
		}
	}

	req, err := d.buildRequest(cfg)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			&promotion.TerminalError{Err: fmt.Errorf("error building HTTP request: %w", err)}
	}

	client, err := d.getClient(cfg)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error creating HTTP client: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error sending HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed},
			fmt.Errorf("HTTP request failed with status %d", resp.StatusCode)
	}

	// Check file size limit using Content-Length header if available
	if contentLength := resp.ContentLength; contentLength > maxDownloadSize {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			&promotion.TerminalError{
				Err: fmt.Errorf("download exceeds limit of %d bytes", maxDownloadSize),
			}
	}

	// Download the file
	if err = d.downloadFile(ctx, resp, absOutPath); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error downloading file: %w", err)
	}

	return promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, nil
}

func (d *httpDownloader) buildRequest(cfg builtin.HTTPDownloadConfig) (*http.Request, error) {
	req, err := http.NewRequest("GET", cfg.URL, nil)
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

func (d *httpDownloader) getClient(cfg builtin.HTTPDownloadConfig) (*http.Client, error) {
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

func (d *httpDownloader) downloadFile(
	ctx context.Context,
	resp *http.Response,
	outPath string,
) error {
	// Create temporary file in the same directory as the final destination
	tempFile, err := os.CreateTemp(filepath.Dir(outPath), filepath.Base(outPath)+".tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	tempPath := tempFile.Name()

	// Ensure cleanup of temp file regardless of outcome
	defer func() {
		_ = tempFile.Close()
		// Always try to remove temp file. This will be a no-op if the file was
		// successfully renamed to the final destination.
		_ = os.Remove(tempPath)
	}()

	// Set permissions for the temporary file
	if err = tempFile.Chmod(0o600); err != nil {
		return fmt.Errorf("failed to set permissions on temporary file: %w", err)
	}

	limitedReader := io.LimitReader(resp.Body, maxDownloadSize)

	// Stream data with context cancellation support
	var bytesDownloaded int64
	buf := make([]byte, downloadBufferSize)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("download canceled: %w", ctx.Err())
		default:
		}

		var n int
		if n, err = limitedReader.Read(buf); n > 0 {
			if _, writeErr := tempFile.Write(buf[:n]); writeErr != nil {
				return fmt.Errorf("failed to write to file: %w", writeErr)
			}
			bytesDownloaded += int64(n)
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}
	}

	// If we read exactly the limit, check if there's more data
	if bytesDownloaded == maxDownloadSize {
		buf := make([]byte, 1)
		var n int

		if n, err = resp.Body.Read(buf); err != nil && err != io.EOF {
			return fmt.Errorf("failed to check for additional content: %w", err)
		}

		if n > 0 {
			return fmt.Errorf("download exceeds limit of %d bytes", maxDownloadSize)
		}
	}

	// Close temp file before rename
	if err = tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	// Move the temporary file to the final destination
	if err = fs.SimpleAtomicMove(tempPath, outPath); err != nil {
		return fmt.Errorf("failed to move file to final destination: %w", err)
	}

	return nil
}
