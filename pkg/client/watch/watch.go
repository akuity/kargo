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

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/hashicorp/go-cleanhttp"
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

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			errCh <- fmt.Errorf("creating request: %w", err)
			return
		}

		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Cache-Control", "no-cache")
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			errCh <- fmt.Errorf("executing request: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			errCh <- fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
			return
		}

		if err := readSSEStream(ctx, resp.Body, eventCh); err != nil {
			errCh <- err
		}
	}()

	return eventCh, errCh
}

// readSSEStream reads SSE events from the response body and sends them to the
// channel.
func readSSEStream[T any](
	ctx context.Context,
	body io.Reader,
	eventCh chan<- Event[T],
) error {
	scanner := bufio.NewScanner(body)
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

// PromoteToStageRequest represents the request body for promoting to a stage.
type PromoteToStageRequest struct {
	Freight      string `json:"freight,omitempty"`
	FreightAlias string `json:"freightAlias,omitempty"`
}

// PromoteToStageResponse represents the response from promoting to a stage.
type PromoteToStageResponse struct {
	Promotion *kargoapi.Promotion `json:"promotion"`
}

// PromoteDownstreamResponse represents the response from promoting downstream.
type PromoteDownstreamResponse struct {
	Promotions []*kargoapi.Promotion `json:"promotions"`
	Errors     string                `json:"errors,omitempty"`
}

// PromoteToStage creates a promotion to transition a stage to the specified freight.
func (c *Client) PromoteToStage(
	ctx context.Context,
	project string,
	stage string,
	freight string,
	freightAlias string,
) (*kargoapi.Promotion, error) {
	url := fmt.Sprintf("%s/v2/projects/%s/stages/%s/promotions", c.baseURL, project, stage)

	reqBody := PromoteToStageRequest{
		Freight:      freight,
		FreightAlias: freightAlias,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	var response PromoteToStageResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	return response.Promotion, nil
}

// PromoteDownstream creates promotions for all stages downstream from the specified stage.
func (c *Client) PromoteDownstream(
	ctx context.Context,
	project string,
	stage string,
	freight string,
	freightAlias string,
) ([]*kargoapi.Promotion, error) {
	url := fmt.Sprintf("%s/v2/projects/%s/stages/%s/promotions/downstream", c.baseURL, project, stage)

	reqBody := PromoteToStageRequest{
		Freight:      freight,
		FreightAlias: freightAlias,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusCreated &&
		resp.StatusCode != http.StatusMultiStatus {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	var response PromoteDownstreamResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	if response.Errors != "" {
		return response.Promotions, fmt.Errorf("partial failure: %s", response.Errors)
	}

	return response.Promotions, nil
}

// AbortPromotion aborts a running promotion.
func (c *Client) AbortPromotion(
	ctx context.Context,
	project string,
	promotion string,
) error {
	url := fmt.Sprintf("%s/v2/projects/%s/promotions/%s/abort", c.baseURL, project, promotion)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
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

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			errCh <- fmt.Errorf("creating request: %w", err)
			return
		}

		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Cache-Control", "no-cache")
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			errCh <- fmt.Errorf("executing request: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			errCh <- fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
			return
		}

		if err := readLogStream(ctx, resp.Body, logCh); err != nil {
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
