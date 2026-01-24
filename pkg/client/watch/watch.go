package watch

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/go-cleanhttp"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// EventType represents the type of watch event.
type EventType string

const (
	Added    EventType = "ADDED"
	Modified EventType = "MODIFIED"
	Deleted  EventType = "DELETED"
)

// Event represents a watch event for a specific resource type.
type Event[T any] struct {
	Type   EventType
	Object T
}

// Client is an SSE watch client for Kargo resources.
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

// NewClient creates a new watch client.
func NewClient(baseURL string, httpClient *http.Client, token string) *Client {
	if httpClient == nil {
		httpClient = cleanhttp.DefaultClient()
	}
	return &Client{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		httpClient: httpClient,
		token:      token,
	}
}

// WatchStage watches a specific Stage for changes. Cancel the provided context
// to stop watching.
func (c *Client) WatchStage(
	ctx context.Context,
	project string,
	stage string,
) (<-chan Event[*kargoapi.Stage], <-chan error) {
	url := fmt.Sprintf("%s/v2/projects/%s/stages/%s?watch=true", c.baseURL, project, stage)
	return watchResource[*kargoapi.Stage](ctx, c, url)
}

// WatchWarehouses watches all Warehouses in a project for changes. Cancel the
// provided context to stop watching.
func (c *Client) WatchWarehouses(
	ctx context.Context,
	project string,
) (<-chan Event[*kargoapi.Warehouse], <-chan error) {
	url := fmt.Sprintf("%s/v2/projects/%s/warehouses?watch=true", c.baseURL, project)
	return watchResource[*kargoapi.Warehouse](ctx, c, url)
}

// WatchStages watches all Stages in a project for changes. Cancel the provided
// context to stop watching.
func (c *Client) WatchStages(
	ctx context.Context,
	project string,
) (<-chan Event[*kargoapi.Stage], <-chan error) {
	url := fmt.Sprintf("%s/v2/projects/%s/stages?watch=true", c.baseURL, project)
	return watchResource[*kargoapi.Stage](ctx, c, url)
}

// WatchPromotions watches all Promotions in a project for changes. Cancel the
// provided context to stop watching.
func (c *Client) WatchPromotions(
	ctx context.Context,
	project string,
) (<-chan Event[*kargoapi.Promotion], <-chan error) {
	url := fmt.Sprintf("%s/v2/projects/%s/promotions?watch=true", c.baseURL, project)
	return watchResource[*kargoapi.Promotion](ctx, c, url)
}

// WatchWarehouse watches a specific Warehouse for changes. Cancel the provided
// context to stop watching.
func (c *Client) WatchWarehouse(
	ctx context.Context,
	project string,
	warehouse string,
) (<-chan Event[*kargoapi.Warehouse], <-chan error) {
	url := fmt.Sprintf("%s/v2/projects/%s/warehouses/%s?watch=true", c.baseURL, project, warehouse)
	return watchResource[*kargoapi.Warehouse](ctx, c, url)
}

// WatchPromotion watches a specific Promotion for changes. Cancel the provided
// context to stop watching.
func (c *Client) WatchPromotion(
	ctx context.Context,
	project string,
	promotion string,
) (<-chan Event[*kargoapi.Promotion], <-chan error) {
	url := fmt.Sprintf("%s/v2/projects/%s/promotions/%s?watch=true", c.baseURL, project, promotion)
	return watchResource[*kargoapi.Promotion](ctx, c, url)
}

// WatchProjectConfig watches the ProjectConfig for a specific project. Cancel
// the provided context to stop watching.
func (c *Client) WatchProjectConfig(
	ctx context.Context,
	project string,
) (<-chan Event[*kargoapi.ProjectConfig], <-chan error) {
	url := fmt.Sprintf("%s/v2/projects/%s/config?watch=true", c.baseURL, project)
	return watchResource[*kargoapi.ProjectConfig](ctx, c, url)
}

// WatchClusterConfig watches the ClusterConfig. Cancel the provided context to
// stop watching.
func (c *Client) WatchClusterConfig(
	ctx context.Context,
) (<-chan Event[*kargoapi.ClusterConfig], <-chan error) {
	url := fmt.Sprintf("%s/v2/cluster-config?watch=true", c.baseURL)
	return watchResource[*kargoapi.ClusterConfig](ctx, c, url)
}

// watchEvent is the generic JSON structure for all watch events.
type watchEvent[T any] struct {
	Type   string `json:"type"`
	Object T      `json:"object"`
}

// doSSERequest executes an SSE request and calls the provided handler with the
// response body. It handles common setup like headers, authentication, and
// error checking.
func (c *Client) doSSERequest(
	ctx context.Context,
	url string,
	handleBody func(ctx context.Context, body io.Reader) error,
) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return handleBody(ctx, resp.Body)
}

// watchResource is the generic implementation for watching any resource type.
func watchResource[T any](
	ctx context.Context,
	c *Client,
	url string,
) (<-chan Event[T], <-chan error) {
	eventCh := make(chan Event[T])
	errCh := make(chan error, 1)

	go func() {
		defer close(eventCh)
		defer close(errCh)

		if err := c.doSSERequest(ctx, url, func(ctx context.Context, body io.Reader) error {
			return readSSEStream(ctx, body, eventCh)
		}); err != nil {
			errCh <- err
		}
	}()

	return eventCh, errCh
}

// maxEventSize is the maximum size of a single SSE event block.
// This is set to 1MB to accommodate large Kargo resources.
const maxEventSize = 1024 * 1024

// readSSEStream reads SSE events from the response body and sends them to the
// channel.
func readSSEStream[T any](
	ctx context.Context,
	body io.Reader,
	eventCh chan<- Event[T],
) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, maxEventSize), maxEventSize)
	scanner.Split(scanSSEEvents)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		eventBlock := scanner.Text()

		// Parse lines within the event block
		var dataLines []string
		for _, line := range strings.Split(eventBlock, "\n") {
			// Skip comments (keepalives)
			if strings.HasPrefix(line, ":") {
				continue
			}
			// Collect data lines
			if strings.HasPrefix(line, "data: ") {
				dataLines = append(dataLines, strings.TrimPrefix(line, "data: "))
			}
		}

		if len(dataLines) == 0 {
			continue
		}

		// Join multiple data lines per SSE spec
		data := strings.Join(dataLines, "\n")

		var event watchEvent[T]
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			return fmt.Errorf("unmarshaling event: %w", err)
		}

		select {
		case eventCh <- Event[T]{
			Type:   EventType(event.Type),
			Object: event.Object,
		}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading stream: %w", err)
	}

	return nil
}

// scanSSEEvents is a bufio.SplitFunc that splits on double newlines,
// which is the event delimiter per the SSE specification.
func scanSSEEvents(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	// Look for double newline (event delimiter)
	if i := bytes.Index(data, []byte("\n\n")); i >= 0 {
		return i + 2, data[:i], nil
	}

	// Also handle \r\n\r\n for Windows-style line endings
	if i := bytes.Index(data, []byte("\r\n\r\n")); i >= 0 {
		return i + 4, data[:i], nil
	}

	// If at EOF and we have data, return it as the final event
	if atEOF {
		return len(data), data, nil
	}

	// Request more data
	return 0, nil, nil
}

// LogEntry represents a single log line from an analysis run.
type LogEntry struct {
	Chunk string `json:"chunk"`
}

// StreamAnalysisRunLogs streams logs from an analysis run.
func (c *Client) StreamAnalysisRunLogs(
	ctx context.Context,
	project string,
	analysisRun string,
	metricName string,
	containerName string,
) (<-chan string, <-chan error) {
	url := fmt.Sprintf("%s/v2/projects/%s/analysis-runs/%s/logs", c.baseURL, project, analysisRun)

	// Add query parameters if provided
	params := make([]string, 0)
	if metricName != "" {
		params = append(params, "metricName="+metricName)
	}
	if containerName != "" {
		params = append(params, "containerName="+containerName)
	}
	if len(params) > 0 {
		url += "?" + strings.Join(params, "&")
	}

	return streamLogs(ctx, c, url)
}

// streamLogs streams log data from an SSE endpoint.
func streamLogs(
	ctx context.Context,
	c *Client,
	url string,
) (<-chan string, <-chan error) {
	logCh := make(chan string)
	errCh := make(chan error, 1)

	go func() {
		defer close(logCh)
		defer close(errCh)

		if err := c.doSSERequest(ctx, url, func(ctx context.Context, body io.Reader) error {
			return readLogStream(ctx, body, logCh)
		}); err != nil {
			errCh <- err
		}
	}()

	return logCh, errCh
}

// readLogStream reads log entries from an SSE stream. The server sends each
// line prefixed with "data: " and terminates events with a double newline.
// Multi-line log chunks are sent as multiple "data:" lines within a single
// event.
func readLogStream(
	ctx context.Context,
	body io.Reader,
	logCh chan<- string,
) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, maxEventSize), maxEventSize)
	scanner.Split(scanSSEEvents)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		eventBlock := scanner.Text()

		// Parse lines within the event block
		var dataLines []string
		for _, line := range strings.Split(eventBlock, "\n") {
			// Skip comments (keepalives)
			if strings.HasPrefix(line, ":") {
				continue
			}
			// Collect data lines
			if strings.HasPrefix(line, "data: ") {
				dataLines = append(dataLines, strings.TrimPrefix(line, "data: "))
			}
		}

		if len(dataLines) == 0 {
			continue
		}

		// Join multiple data lines per SSE spec
		data := strings.Join(dataLines, "\n")

		select {
		case logCh <- data:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading stream: %w", err)
	}

	return nil
}
