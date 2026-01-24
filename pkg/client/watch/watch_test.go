package watch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name       string
		baseURL    string
		httpClient *http.Client
		token      string
		wantURL    string
	}{
		{
			name:       "with all parameters",
			baseURL:    "https://kargo.example.com",
			httpClient: &http.Client{Timeout: 10 * time.Second},
			token:      "test-token",
			wantURL:    "https://kargo.example.com",
		},
		{
			name:       "with trailing slash in base URL",
			baseURL:    "https://kargo.example.com/",
			httpClient: nil,
			token:      "",
			wantURL:    "https://kargo.example.com",
		},
		{
			name:       "with nil http client uses default",
			baseURL:    "https://kargo.example.com",
			httpClient: nil,
			token:      "token",
			wantURL:    "https://kargo.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.baseURL, tt.httpClient, tt.token)
			require.NotNil(t, client)
			assert.Equal(t, tt.wantURL, client.baseURL)
			assert.Equal(t, tt.token, client.token)
			if tt.httpClient != nil {
				assert.Equal(t, tt.httpClient, client.httpClient)
			} else {
				// When nil is passed, NewClient creates a default client via cleanhttp
				assert.NotNil(t, client.httpClient)
			}
		})
	}
}

func TestWatchStage(t *testing.T) {
	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-stage",
			Namespace: "test-project",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/projects/test-project/stages/test-stage", r.URL.Path)
		assert.Equal(t, "true", r.URL.Query().Get("watch"))
		assert.Equal(t, "text/event-stream", r.Header.Get("Accept"))
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		event := watchEvent[*kargoapi.Stage]{
			Type:   string(Added),
			Object: stage,
		}
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", data)
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client(), "test-token")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	eventCh, errCh := client.WatchStage(ctx, "test-project", "test-stage")

	select {
	case event := <-eventCh:
		assert.Equal(t, Added, event.Type)
		assert.Equal(t, "test-stage", event.Object.Name)
	case err := <-errCh:
		t.Fatalf("unexpected error: %v", err)
	case <-ctx.Done():
		t.Fatal("timeout waiting for event")
	}
}

func TestWatchStages(t *testing.T) {
	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-stage",
			Namespace: "test-project",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/projects/test-project/stages", r.URL.Path)
		assert.Equal(t, "true", r.URL.Query().Get("watch"))

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		event := watchEvent[*kargoapi.Stage]{
			Type:   string(Modified),
			Object: stage,
		}
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", data)
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client(), "test-token")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	eventCh, errCh := client.WatchStages(ctx, "test-project")

	select {
	case event := <-eventCh:
		assert.Equal(t, Modified, event.Type)
		assert.Equal(t, "test-stage", event.Object.Name)
	case err := <-errCh:
		t.Fatalf("unexpected error: %v", err)
	case <-ctx.Done():
		t.Fatal("timeout waiting for event")
	}
}

func TestWatchWarehouse(t *testing.T) {
	warehouse := &kargoapi.Warehouse{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-warehouse",
			Namespace: "test-project",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/projects/test-project/warehouses/test-warehouse", r.URL.Path)
		assert.Equal(t, "true", r.URL.Query().Get("watch"))

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		event := watchEvent[*kargoapi.Warehouse]{
			Type:   string(Deleted),
			Object: warehouse,
		}
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", data)
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client(), "test-token")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	eventCh, errCh := client.WatchWarehouse(ctx, "test-project", "test-warehouse")

	select {
	case event := <-eventCh:
		assert.Equal(t, Deleted, event.Type)
		assert.Equal(t, "test-warehouse", event.Object.Name)
	case err := <-errCh:
		t.Fatalf("unexpected error: %v", err)
	case <-ctx.Done():
		t.Fatal("timeout waiting for event")
	}
}

func TestWatchWarehouses(t *testing.T) {
	warehouse := &kargoapi.Warehouse{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-warehouse",
			Namespace: "test-project",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/projects/test-project/warehouses", r.URL.Path)
		assert.Equal(t, "true", r.URL.Query().Get("watch"))

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		event := watchEvent[*kargoapi.Warehouse]{
			Type:   string(Added),
			Object: warehouse,
		}
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", data)
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client(), "test-token")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	eventCh, errCh := client.WatchWarehouses(ctx, "test-project")

	select {
	case event := <-eventCh:
		assert.Equal(t, Added, event.Type)
		assert.Equal(t, "test-warehouse", event.Object.Name)
	case err := <-errCh:
		t.Fatalf("unexpected error: %v", err)
	case <-ctx.Done():
		t.Fatal("timeout waiting for event")
	}
}

func TestWatchPromotion(t *testing.T) {
	promotion := &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-promotion",
			Namespace: "test-project",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/projects/test-project/promotions/test-promotion", r.URL.Path)
		assert.Equal(t, "true", r.URL.Query().Get("watch"))

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		event := watchEvent[*kargoapi.Promotion]{
			Type:   string(Modified),
			Object: promotion,
		}
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", data)
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client(), "test-token")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	eventCh, errCh := client.WatchPromotion(ctx, "test-project", "test-promotion")

	select {
	case event := <-eventCh:
		assert.Equal(t, Modified, event.Type)
		assert.Equal(t, "test-promotion", event.Object.Name)
	case err := <-errCh:
		t.Fatalf("unexpected error: %v", err)
	case <-ctx.Done():
		t.Fatal("timeout waiting for event")
	}
}

func TestWatchPromotions(t *testing.T) {
	promotion := &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-promotion",
			Namespace: "test-project",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/projects/test-project/promotions", r.URL.Path)
		assert.Equal(t, "true", r.URL.Query().Get("watch"))

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		event := watchEvent[*kargoapi.Promotion]{
			Type:   string(Added),
			Object: promotion,
		}
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", data)
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client(), "test-token")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	eventCh, errCh := client.WatchPromotions(ctx, "test-project")

	select {
	case event := <-eventCh:
		assert.Equal(t, Added, event.Type)
		assert.Equal(t, "test-promotion", event.Object.Name)
	case err := <-errCh:
		t.Fatalf("unexpected error: %v", err)
	case <-ctx.Done():
		t.Fatal("timeout waiting for event")
	}
}

func TestWatchProjectConfig(t *testing.T) {
	config := &kargoapi.ProjectConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-project",
			Namespace: "test-project",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/projects/test-project/config", r.URL.Path)
		assert.Equal(t, "true", r.URL.Query().Get("watch"))

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		event := watchEvent[*kargoapi.ProjectConfig]{
			Type:   string(Modified),
			Object: config,
		}
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", data)
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client(), "test-token")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	eventCh, errCh := client.WatchProjectConfig(ctx, "test-project")

	select {
	case event := <-eventCh:
		assert.Equal(t, Modified, event.Type)
		assert.Equal(t, "test-project", event.Object.Name)
	case err := <-errCh:
		t.Fatalf("unexpected error: %v", err)
	case <-ctx.Done():
		t.Fatal("timeout waiting for event")
	}
}

func TestWatchClusterConfig(t *testing.T) {
	config := &kargoapi.ClusterConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster-config",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/cluster-config", r.URL.Path)
		assert.Equal(t, "true", r.URL.Query().Get("watch"))

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		event := watchEvent[*kargoapi.ClusterConfig]{
			Type:   string(Modified),
			Object: config,
		}
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", data)
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client(), "test-token")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	eventCh, errCh := client.WatchClusterConfig(ctx)

	select {
	case event := <-eventCh:
		assert.Equal(t, Modified, event.Type)
		assert.Equal(t, "cluster-config", event.Object.Name)
	case err := <-errCh:
		t.Fatalf("unexpected error: %v", err)
	case <-ctx.Done():
		t.Fatal("timeout waiting for event")
	}
}

func TestWatchResource_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "unauthorized"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client(), "bad-token")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	eventCh, errCh := client.WatchStage(ctx, "test-project", "test-stage")

	select {
	case <-eventCh:
		t.Fatal("expected error, got event")
	case err := <-errCh:
		assert.Contains(t, err.Error(), "unexpected status 401")
	case <-ctx.Done():
		t.Fatal("timeout waiting for error")
	}
}

func TestWatchResource_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Keep connection open until context is canceled
		<-r.Context().Done()
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client(), "test-token")
	ctx, cancel := context.WithCancel(context.Background())

	eventCh, errCh := client.WatchStage(ctx, "test-project", "test-stage")

	// Cancel the context
	cancel()

	// Should receive context error or channel close
	select {
	case _, ok := <-eventCh:
		if ok {
			t.Fatal("expected channel to close")
		}
	case err := <-errCh:
		if err != nil {
			assert.ErrorIs(t, err, context.Canceled)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for channels to close")
	}
}

// testObject is a simple struct for testing readSSEStream
type testObject struct {
	Name string `json:"name"`
}

func TestReadSSEStream(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedType  EventType
		expectedName  string
		expectError   bool
		errorContains string
	}{
		{
			name: "valid event",
			input: `data: {"type":"ADDED","object":{"name":"test"}}

`,
			expectedType: Added,
			expectedName: "test",
		},
		{
			name: "skip empty lines",
			input: `
data: {"type":"MODIFIED","object":{"name":"test"}}

`,
			expectedType: Modified,
			expectedName: "test",
		},
		{
			name: "skip comment lines (keepalives)",
			input: `: keepalive
data: {"type":"DELETED","object":{"name":"test"}}

`,
			expectedType: Deleted,
			expectedName: "test",
		},
		{
			name:          "invalid JSON",
			input:         "data: {invalid json}\n\n",
			expectError:   true,
			errorContains: "unmarshaling event",
		},
		{
			name: "skip non-data lines",
			input: `event: message
data: {"type":"ADDED","object":{"name":"test"}}

`,
			expectedType: Added,
			expectedName: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			eventCh := make(chan Event[*testObject], 1)

			ctx := context.Background()
			err := readSSEStream(ctx, reader, eventCh)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				return
			}

			require.NoError(t, err)

			select {
			case event := <-eventCh:
				assert.Equal(t, tt.expectedType, event.Type)
				assert.Equal(t, tt.expectedName, event.Object.Name)
			default:
				t.Fatal("expected event in channel")
			}
		})
	}
}

func TestReadSSEStream_ContextCancellation(t *testing.T) {
	// Use a pipe to create a blocking reader
	pr, pw := io.Pipe()
	defer pr.Close()
	defer pw.Close()

	eventCh := make(chan Event[*testObject])

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- readSSEStream(ctx, pr, eventCh)
	}()

	// Write a partial event
	_, _ = pw.Write([]byte("data: "))

	// Cancel context
	cancel()

	// Close the pipe to unblock the scanner
	pw.Close()

	select {
	case err := <-errCh:
		// Should get context canceled error or nil (if scanner finished first)
		if err != nil {
			assert.ErrorIs(t, err, context.Canceled)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for readSSEStream to return")
	}
}

func TestStreamAnalysisRunLogs(t *testing.T) {
	tests := []struct {
		name          string
		metricName    string
		containerName string
		expectedPath  string
		expectedQuery string
	}{
		{
			name:          "no query params",
			metricName:    "",
			containerName: "",
			expectedPath:  "/v2/projects/test-project/analysis-runs/test-run/logs",
			expectedQuery: "",
		},
		{
			name:          "with metric name",
			metricName:    "my-metric",
			containerName: "",
			expectedPath:  "/v2/projects/test-project/analysis-runs/test-run/logs",
			expectedQuery: "metricName=my-metric",
		},
		{
			name:          "with container name",
			metricName:    "",
			containerName: "my-container",
			expectedPath:  "/v2/projects/test-project/analysis-runs/test-run/logs",
			expectedQuery: "containerName=my-container",
		},
		{
			name:          "with both params",
			metricName:    "my-metric",
			containerName: "my-container",
			expectedPath:  "/v2/projects/test-project/analysis-runs/test-run/logs",
			expectedQuery: "metricName=my-metric&containerName=my-container",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tt.expectedPath, r.URL.Path)
				assert.Equal(t, tt.expectedQuery, r.URL.RawQuery)
				assert.Equal(t, "text/event-stream", r.Header.Get("Accept"))
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)
				// SSE format: each line prefixed with "data: ", empty line terminates event
				fmt.Fprint(w, "data: log line 1\n\n")
				fmt.Fprint(w, "data: log line 2\n\n")
			}))
			defer server.Close()

			client := NewClient(server.URL, server.Client(), "test-token")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			logCh, errCh := client.StreamAnalysisRunLogs(ctx, "test-project", "test-run", tt.metricName, tt.containerName)

			logs := make([]string, 0)
			for log := range logCh {
				logs = append(logs, log)
			}

			select {
			case err := <-errCh:
				require.NoError(t, err)
			default:
			}

			assert.Equal(t, []string{"log line 1", "log line 2"}, logs)
		})
	}
}

func TestStreamAnalysisRunLogs_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "analysis run not found"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client(), "test-token")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logCh, errCh := client.StreamAnalysisRunLogs(ctx, "test-project", "test-run", "", "")

	// Drain log channel
	for range logCh {
	}

	select {
	case err := <-errCh:
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected status 404")
	case <-ctx.Done():
		t.Fatal("timeout waiting for error")
	}
}

func TestReadLogStream(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedLogs []string
	}{
		{
			name: "valid log entries",
			// SSE format: data lines followed by empty line to terminate event
			input:        "data: line 1\n\ndata: line 2\n\ndata: line 3\n\n",
			expectedLogs: []string{"line 1", "line 2", "line 3"},
		},
		{
			name: "skip comments (keepalives)",
			input: `: keepalive
data: line 1

: another keepalive
data: line 2

`,
			expectedLogs: []string{"line 1", "line 2"},
		},
		{
			name: "multi-line log chunks",
			// Server sends each line of a chunk as a separate data: line
			// Client accumulates and joins with newlines
			input:        "data: first line\ndata: second line\ndata: third line\n\n",
			expectedLogs: []string{"first line\nsecond line\nthird line"},
		},
		{
			name:         "multiple multi-line chunks",
			input:        "data: chunk1 line1\ndata: chunk1 line2\n\ndata: chunk2 line1\ndata: chunk2 line2\n\n",
			expectedLogs: []string{"chunk1 line1\nchunk1 line2", "chunk2 line1\nchunk2 line2"},
		},
		{
			name: "empty data lines preserved",
			// Empty data: lines represent empty lines in the original content
			input:        "data: line 1\ndata: \ndata: line 3\n\n",
			expectedLogs: []string{"line 1\n\nline 3"},
		},
		{
			name:         "handles stream ending without final empty line",
			input:        "data: final chunk\n",
			expectedLogs: []string{"final chunk"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			logCh := make(chan string, 10)

			ctx := context.Background()
			err := readLogStream(ctx, reader, logCh)
			close(logCh)

			require.NoError(t, err)

			logs := make([]string, 0)
			for log := range logCh {
				logs = append(logs, log)
			}
			assert.Equal(t, tt.expectedLogs, logs)
		})
	}
}

func TestMultipleEventsInStream(t *testing.T) {
	stages := []*kargoapi.Stage{
		{ObjectMeta: metav1.ObjectMeta{Name: "stage-1"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "stage-2"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "stage-3"}},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		for i, stage := range stages {
			var eventType EventType
			switch i {
			case 1:
				eventType = Modified
			case 2:
				eventType = Deleted
			default:
				eventType = Added
			}

			event := watchEvent[*kargoapi.Stage]{
				Type:   string(eventType),
				Object: stage,
			}
			data, _ := json.Marshal(event)
			_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client(), "test-token")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	eventCh, errCh := client.WatchStages(ctx, "test-project")

	events := make([]Event[*kargoapi.Stage], 0)
	for event := range eventCh {
		events = append(events, event)
	}

	select {
	case err := <-errCh:
		require.NoError(t, err)
	default:
	}

	require.Len(t, events, 3)
	assert.Equal(t, Added, events[0].Type)
	assert.Equal(t, "stage-1", events[0].Object.Name)
	assert.Equal(t, Modified, events[1].Type)
	assert.Equal(t, "stage-2", events[1].Object.Name)
	assert.Equal(t, Deleted, events[2].Type)
	assert.Equal(t, "stage-3", events[2].Object.Name)
}
