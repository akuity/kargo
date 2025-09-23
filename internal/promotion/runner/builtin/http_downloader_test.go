package builtin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func Test_httpDownloader_validate(t *testing.T) {
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
			name: "outPath not specified",
			config: promotion.Config{
				"url": "https://example.com/file.txt",
			},
			expectedProblems: []string{
				"(root): outPath is required",
			},
		},
		{
			name: "outPath is empty string",
			config: promotion.Config{
				"url":     "https://example.com/file.txt",
				"outPath": "",
			},
			expectedProblems: []string{
				"outPath: String length must be greater than or equal to 1",
			},
		},
		{
			name: "header name not specified",
			config: promotion.Config{
				"url":     "https://example.com/file.txt",
				"outPath": "file.txt",
				"headers": []promotion.Config{{}},
			},
			expectedProblems: []string{
				"headers.0: name is required",
			},
		},
		{
			name: "header name is empty string",
			config: promotion.Config{
				"url":     "https://example.com/file.txt",
				"outPath": "file.txt",
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
				"url":     "https://example.com/file.txt",
				"outPath": "file.txt",
				"headers": []promotion.Config{{}},
			},
			expectedProblems: []string{
				"headers.0: value is required",
			},
		},
		{
			name: "header value is empty string",
			config: promotion.Config{
				"url":     "https://example.com/file.txt",
				"outPath": "file.txt",
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
				"url":         "https://example.com/file.txt",
				"outPath":     "file.txt",
				"queryParams": []promotion.Config{{}},
			},
			expectedProblems: []string{
				"queryParams.0: name is required",
			},
		},
		{
			name: "query param name is empty string",
			config: promotion.Config{
				"url":     "https://example.com/file.txt",
				"outPath": "file.txt",
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
				"url":         "https://example.com/file.txt",
				"outPath":     "file.txt",
				"queryParams": []promotion.Config{{}},
			},
			expectedProblems: []string{
				"queryParams.0: value is required",
			},
		},
		{
			name: "query param value is empty string",
			config: promotion.Config{
				"url":     "https://example.com/file.txt",
				"outPath": "file.txt",
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
				"url":     "https://example.com/file.txt",
				"outPath": "file.txt",
				"timeout": "invalid",
			},
			expectedProblems: []string{
				"timeout: Does not match pattern",
			},
		},
		{
			name: "valid kitchen sink",
			config: promotion.Config{
				"url":     "https://example.com/file.txt",
				"outPath": "downloads/file.txt",
				"headers": []promotion.Config{{
					"name":  "Authorization",
					"value": "Bearer token123",
				}},
				"queryParams": []promotion.Config{{
					"name":  "version",
					"value": "latest",
				}},
				"insecureSkipTLSVerify": true,
				"timeout":               "30s",
				"allowOverwrite":        true,
			},
		},
	}

	r := newHTTPDownloader(promotion.StepRunnerCapabilities{})
	runner, ok := r.(*httpDownloader)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

func Test_httpDownloader_run(t *testing.T) {
	tests := []struct {
		name       string
		cfg        builtin.HTTPDownloadConfig
		handler    http.HandlerFunc
		setupFile  func(t *testing.T, workDir, outPath string)           // Setup existing file if needed
		assertions func(*testing.T, promotion.StepResult, error, string) // workDir passed for file checks
	}{
		{
			name: "successful download",
			cfg: builtin.HTTPDownloadConfig{
				OutPath: "test-file.txt",
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, err := w.Write([]byte("Hello, World!"))
				require.NoError(t, err)
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error, workDir string) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)

				// Check that file was created with correct content
				content, err := os.ReadFile(filepath.Join(workDir, "test-file.txt"))
				require.NoError(t, err)
				require.Equal(t, "Hello, World!", string(content))
			},
		},
		{
			name: "successful download with nested path",
			cfg: builtin.HTTPDownloadConfig{
				OutPath: "nested/dir/test-file.txt",
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, err := w.Write([]byte("nested content"))
				require.NoError(t, err)
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error, workDir string) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)

				// Check that nested directories were created
				content, err := os.ReadFile(filepath.Join(workDir, "nested/dir/test-file.txt"))
				require.NoError(t, err)
				require.Equal(t, "nested content", string(content))
			},
		},
		{
			name: "file already exists and overwrite not allowed",
			cfg: builtin.HTTPDownloadConfig{
				OutPath:        "existing-file.txt",
				AllowOverwrite: false,
			},
			setupFile: func(t *testing.T, workDir, outPath string) {
				err := os.WriteFile(filepath.Join(workDir, outPath), []byte("existing"), 0o600)
				require.NoError(t, err)
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, err := w.Write([]byte("new content"))
				require.NoError(t, err)
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error, workDir string) {
				require.ErrorContains(t, err, "file already exists")
				require.True(t, promotion.IsTerminal(err))
				require.Equal(t, kargoapi.PromotionStepStatusErrored, res.Status)

				// Original file should remain unchanged
				content, err := os.ReadFile(filepath.Join(workDir, "existing-file.txt"))
				require.NoError(t, err)
				require.Equal(t, "existing", string(content))
			},
		},
		{
			name: "file already exists and overwrite allowed",
			cfg: builtin.HTTPDownloadConfig{
				OutPath:        "existing-file.txt",
				AllowOverwrite: true,
			},
			setupFile: func(t *testing.T, workDir, outPath string) {
				err := os.WriteFile(filepath.Join(workDir, outPath), []byte("existing"), 0o600)
				require.NoError(t, err)
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, err := w.Write([]byte("new content"))
				require.NoError(t, err)
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error, workDir string) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)

				// File should be overwritten
				content, err := os.ReadFile(filepath.Join(workDir, "existing-file.txt"))
				require.NoError(t, err)
				require.Equal(t, "new content", string(content))
			},
		},
		{
			name: "HTTP error response",
			cfg: builtin.HTTPDownloadConfig{
				OutPath: "test-file.txt",
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error, workDir string) {
				require.ErrorContains(t, err, "HTTP request failed with status 404")
				require.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)

				// File should not exist
				_, err = os.Stat(filepath.Join(workDir, "test-file.txt"))
				require.True(t, os.IsNotExist(err))
			},
		},
		{
			name: "download exceeds size limit via Content-Length",
			cfg: builtin.HTTPDownloadConfig{
				OutPath: "large-file.txt",
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Length", fmt.Sprintf("%d", maxDownloadSize+1))
				_, err := w.Write([]byte("content"))
				require.NoError(t, err)
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error, _ string) {
				require.ErrorContains(t, err, "download exceeds limit")
				require.True(t, promotion.IsTerminal(err))
				require.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)
			},
		},
		{
			name: "download with custom headers and query params",
			cfg: builtin.HTTPDownloadConfig{
				OutPath: "test-file.txt",
				Headers: []builtin.HTTPDownloadConfigHeader{{
					Name:  "Authorization",
					Value: "Bearer token123",
				}},
				QueryParams: []builtin.HTTPDownloadConfigQueryParam{{
					Name:  "version",
					Value: "latest",
				}},
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				// Verify headers and query params were sent
				require.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
				require.Equal(t, "latest", r.URL.Query().Get("version"))

				_, err := w.Write([]byte("authenticated content"))
				require.NoError(t, err)
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error, workDir string) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)

				content, err := os.ReadFile(filepath.Join(workDir, "test-file.txt"))
				require.NoError(t, err)
				require.Equal(t, "authenticated content", string(content))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary work directory
			workDir := t.TempDir()

			// Setup existing file if needed
			if tt.setupFile != nil {
				tt.setupFile(t, workDir, tt.cfg.OutPath)
			}

			// Create test server
			srv := httptest.NewServer(tt.handler)
			t.Cleanup(srv.Close)
			tt.cfg.URL = srv.URL

			// Create step context
			stepCtx := &promotion.StepContext{
				WorkDir: workDir,
			}

			// Run the downloader
			d := &httpDownloader{}
			res, err := d.run(context.Background(), stepCtx, tt.cfg)
			tt.assertions(t, res, err, workDir)
		})
	}
}

func Test_httpDownloader_buildRequest(t *testing.T) {
	d := &httpDownloader{}
	req, err := d.buildRequest(builtin.HTTPDownloadConfig{
		URL: "http://example.com/file.txt",
		Headers: []builtin.HTTPDownloadConfigHeader{{
			Name:  "Authorization",
			Value: "Bearer token123",
		}},
		QueryParams: []builtin.HTTPDownloadConfigQueryParam{{
			Name:  "version",
			Value: "some value", // We want to be sure this gets url-encoded
		}},
	})
	require.NoError(t, err)
	require.Equal(t, "GET", req.Method)
	require.Equal(t, "http://example.com/file.txt?version=some+value", req.URL.String())
	require.Equal(t, "Bearer token123", req.Header.Get("Authorization"))
}

func Test_httpDownloader_buildHTTPClient(t *testing.T) {
	tests := []struct {
		name       string
		cfg        builtin.HTTPDownloadConfig
		assertions func(*testing.T, *http.Client, error)
	}{
		{
			name: "default configuration",
			assertions: func(t *testing.T, client *http.Client, err error) {
				require.NoError(t, err)
				require.NotNil(t, client)
				require.Equal(t, downloadTimeoutDefault, client.Timeout)
				transport, ok := client.Transport.(*http.Transport)
				require.True(t, ok)
				require.Nil(t, transport.TLSClientConfig)
			},
		},
		{
			name: "with insecureSkipTLSVerify",
			cfg: builtin.HTTPDownloadConfig{
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
			name: "with custom timeout",
			cfg: builtin.HTTPDownloadConfig{
				Timeout: "30s",
			},
			assertions: func(t *testing.T, client *http.Client, err error) {
				require.NoError(t, err)
				require.NotNil(t, client)
				require.Equal(t, 30*time.Second, client.Timeout)
			},
		},
		{
			name: "with invalid timeout",
			cfg: builtin.HTTPDownloadConfig{
				Timeout: "invalid",
			},
			assertions: func(t *testing.T, _ *http.Client, err error) {
				require.ErrorContains(t, err, "error parsing timeout")
			},
		},
	}

	d := &httpDownloader{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := d.buildHTTPClient(tt.cfg)
			tt.assertions(t, client, err)
		})
	}
}

func Test_httpDownloader_downloadToFile(t *testing.T) {
	tests := []struct {
		name        string
		contentSize int64
		content     string
		assertions  func(*testing.T, error, string)
	}{
		{
			name:        "small file download",
			contentSize: 100,
			content:     strings.Repeat("a", 100),
			assertions: func(t *testing.T, err error, outPath string) {
				require.NoError(t, err)
				content, err := os.ReadFile(outPath)
				require.NoError(t, err)
				require.Equal(t, strings.Repeat("a", 100), string(content))
			},
		},
		{
			name:        "empty file download",
			contentSize: 0,
			content:     "",
			assertions: func(t *testing.T, err error, outPath string) {
				require.NoError(t, err)
				content, err := os.ReadFile(outPath)
				require.NoError(t, err)
				require.Equal(t, "", string(content))
			},
		},
		{
			name:        "file download at size limit",
			contentSize: maxDownloadSize,
			content:     strings.Repeat("x", int(maxDownloadSize)),
			assertions: func(t *testing.T, err error, outPath string) {
				require.NoError(t, err)
				content, err := os.ReadFile(outPath)
				require.NoError(t, err)
				require.Equal(t, int64(maxDownloadSize), int64(len(content)))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock response
			resp := &http.Response{
				Request: &http.Request{
					URL: &url.URL{
						Scheme: "https",
						Host:   "example.com",
						Path:   "downloaded-file.txt",
					},
				},
				StatusCode:    200,
				ContentLength: tt.contentSize,
				Body:          io.NopCloser(strings.NewReader(tt.content)),
			}

			// Create temporary output file path
			tempDir := t.TempDir()
			outPath := filepath.Join(tempDir, "downloaded-file.txt")

			// Download the file
			d := &httpDownloader{}
			err := d.downloadToFile(context.Background(), resp, outPath)
			tt.assertions(t, err, outPath)
		})
	}
}

func Test_httpDownloader_downloadToFile_contextCancellation(t *testing.T) {
	// Create a slow reader that will be interrupted by context cancellation
	sr := &slowReader{
		content: strings.Repeat("x", 1000),
		delay:   100 * time.Millisecond,
	}

	resp := &http.Response{
		Request: &http.Request{
			URL: &url.URL{
				Scheme: "https",
				Host:   "example.com",
				Path:   "downloaded-file.txt",
			},
		},
		StatusCode: 200,
		Body:       io.NopCloser(sr),
	}

	tempDir := t.TempDir()
	outPath := filepath.Join(tempDir, "interrupted-file.txt")

	// Create context that will be canceled
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	d := &httpDownloader{}
	err := d.downloadToFile(ctx, resp, outPath)

	require.ErrorContains(t, err, "download canceled")

	// File should not exist or be incomplete
	_, err = os.Stat(outPath)
	require.True(t, os.IsNotExist(err))
}

func Test_httpDownloader_downloadToFile_sizeExceeded(t *testing.T) {
	// Create content that exceeds the size limit
	largeContent := strings.Repeat("x", int(maxDownloadSize)+1000)

	resp := &http.Response{
		Request: &http.Request{
			URL: &url.URL{
				Scheme: "https",
				Host:   "example.com",
				Path:   "downloaded-file.txt",
			},
		},
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(largeContent)),
	}

	tempDir := t.TempDir()
	outPath := filepath.Join(tempDir, "large-file.txt")

	d := &httpDownloader{}
	err := d.downloadToFile(context.Background(), resp, outPath)

	require.ErrorContains(t, err, "download exceeds limit")

	// File should not exist
	_, err = os.Stat(outPath)
	require.True(t, os.IsNotExist(err))
}

type slowReader struct {
	content string
	pos     int
	delay   time.Duration
}

func (sr *slowReader) Read(p []byte) (n int, err error) {
	if sr.pos >= len(sr.content) {
		return 0, io.EOF
	}

	// Simulate slow reading
	time.Sleep(sr.delay)

	// Read one byte at a time
	if len(p) > 0 {
		p[0] = sr.content[sr.pos]
		sr.pos++
		return 1, nil
	}

	return 0, nil
}
