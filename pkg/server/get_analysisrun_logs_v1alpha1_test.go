package server

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	rolloutsapi "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
)

func TestServer_getJobMetric(t *testing.T) {
	const testMetricName = "fake-metric"
	testCases := []struct {
		name               string
		providedMetricName string
		run                *rolloutsapi.AnalysisRun
		assertions         func(
			t *testing.T,
			metricName string,
			metric *rolloutsapi.JobMetric,
			err error,
		)
	}{
		{
			name:               "job metric with specified name found",
			providedMetricName: testMetricName,
			run: &rolloutsapi.AnalysisRun{
				Spec: rolloutsapi.AnalysisRunSpec{
					Metrics: []rolloutsapi.Metric{{
						Name: testMetricName,
						Provider: rolloutsapi.MetricProvider{
							Job: &rolloutsapi.JobMetric{},
						},
					}},
				},
			},
			assertions: func(
				t *testing.T,
				metricName string,
				metric *rolloutsapi.JobMetric,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, testMetricName, metricName)
				require.NotNil(t, metric)
			},
		},
		{
			name:               "job metric with specified name not found",
			providedMetricName: testMetricName,
			run: &rolloutsapi.AnalysisRun{
				Spec: rolloutsapi.AnalysisRunSpec{
					Metrics: []rolloutsapi.Metric{{
						Name: "wrong-metric",
						Provider: rolloutsapi.MetricProvider{
							Job: &rolloutsapi.JobMetric{},
						},
					}},
				},
			},
			assertions: func(t *testing.T, _ string, _ *rolloutsapi.JobMetric, err error) {
				require.ErrorContains(t, err, "has no job metric named")
			},
		},
		{
			name: "no job metrics found",
			run: &rolloutsapi.AnalysisRun{
				Spec: rolloutsapi.AnalysisRunSpec{
					Metrics: []rolloutsapi.Metric{{
						Name: testMetricName,
						Provider: rolloutsapi.MetricProvider{
							Prometheus: &rolloutsapi.PrometheusMetric{}, // Wrong kind of metric
						},
					}},
				},
			},
			assertions: func(t *testing.T, _ string, _ *rolloutsapi.JobMetric, err error) {
				require.ErrorContains(t, err, "has no job metrics")
			},
		},
		{
			name: "multiple job metrics found",
			run: &rolloutsapi.AnalysisRun{
				Spec: rolloutsapi.AnalysisRunSpec{
					Metrics: []rolloutsapi.Metric{
						{
							Name: "foo",
							Provider: rolloutsapi.MetricProvider{
								Job: &rolloutsapi.JobMetric{},
							},
						},
						{
							Name: "bar",
							Provider: rolloutsapi.MetricProvider{
								Job: &rolloutsapi.JobMetric{},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, _ string, _ *rolloutsapi.JobMetric, err error) {
				require.ErrorContains(t, err, "has multiple job metrics; please specify a metric name")
			},
		},
		{
			name: "one job metric found",
			run: &rolloutsapi.AnalysisRun{
				Spec: rolloutsapi.AnalysisRunSpec{
					Metrics: []rolloutsapi.Metric{
						{
							Name: testMetricName,
							Provider: rolloutsapi.MetricProvider{
								Job: &rolloutsapi.JobMetric{},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, name string, metric *rolloutsapi.JobMetric, err error) {
				require.NoError(t, err)
				require.Equal(t, testMetricName, name)
				require.NotNil(t, metric)
			},
		},
	}
	s := &server{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			metricName, metric, err := s.getJobMetric(testCase.run, testCase.providedMetricName)
			testCase.assertions(t, metricName, metric, err)
		})
	}
}

func TestServer_getContainerName(t *testing.T) {
	const testContainerName = "fake-container"
	testCases := []struct {
		name                  string
		metric                *rolloutsapi.JobMetric
		providedContainerName string
		assertions            func(t *testing.T, name string, err error)
	}{
		{
			name:   "no containers in pod template",
			metric: &rolloutsapi.JobMetric{},
			assertions: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, "has no containers in Jobs for metric")
			},
		},
		{
			name: "container with specified name found",
			metric: &rolloutsapi.JobMetric{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{Name: testContainerName}},
						},
					},
				},
			},
			providedContainerName: testContainerName,
			assertions: func(t *testing.T, name string, err error) {
				require.NoError(t, err)
				require.Equal(t, testContainerName, name)
			},
		},
		{
			name: "container with specified name not found",
			metric: &rolloutsapi.JobMetric{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{Name: "wrong-container"}},
						},
					},
				},
			},
			providedContainerName: testContainerName,
			assertions: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, "has no container named")
			},
		},
		{
			name: "multiple containers found",
			metric: &rolloutsapi.JobMetric{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "foo"},
								{Name: "bar"},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, "has multiple containers in Jobs for metric")
			},
		},
		{
			name: "one container found",
			metric: &rolloutsapi.JobMetric{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{Name: testContainerName}},
						},
					},
				},
			},
			assertions: func(t *testing.T, name string, err error) {
				require.NoError(t, err)
				require.Equal(t, testContainerName, name)
			},
		},
	}
	s := &server{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			name, err := s.getContainerName(
				&rolloutsapi.AnalysisRun{},
				"fake-metric",
				testCase.metric,
				testCase.providedContainerName,
			)
			testCase.assertions(t, name, err)
		})
	}
}

func TestServer_getJobNamespaceAndName(t *testing.T) {
	const testMetricName = "fake-metric"
	testCases := []struct {
		name       string
		run        *rolloutsapi.AnalysisRun
		assertions func(t *testing.T, namespace, name string, err error)
	}{
		{
			name: "metric result not found",
			run: &rolloutsapi.AnalysisRun{
				Status: rolloutsapi.AnalysisRunStatus{
					MetricResults: []rolloutsapi.MetricResult{
						{Name: "wrong-metric"},
						{Name: "another-wrong-metric"},
					},
				},
			},
			assertions: func(t *testing.T, _, _ string, err error) {
				require.ErrorContains(t, err, "has no result for metric")
			},
		},
		{
			name: "result has no measurements",
			run: &rolloutsapi.AnalysisRun{
				Status: rolloutsapi.AnalysisRunStatus{
					MetricResults: []rolloutsapi.MetricResult{{
						Name: testMetricName,
					}},
				},
			},
			assertions: func(t *testing.T, _, _ string, err error) {
				require.ErrorContains(t, err, "has no measurements")
			},
		},
		{
			name: "result is missing namespace metadata",
			run: &rolloutsapi.AnalysisRun{
				Status: rolloutsapi.AnalysisRunStatus{
					MetricResults: []rolloutsapi.MetricResult{{
						Name:         testMetricName,
						Measurements: []rolloutsapi.Measurement{{}},
					}},
				},
			},
			assertions: func(t *testing.T, _, _ string, err error) {
				require.ErrorContains(t, err, "has no Job namespace metadata")
			},
		},
		{
			name: "result is missing name metadata",
			run: &rolloutsapi.AnalysisRun{
				Status: rolloutsapi.AnalysisRunStatus{
					MetricResults: []rolloutsapi.MetricResult{{
						Name: testMetricName,
						Measurements: []rolloutsapi.Measurement{{
							Metadata: map[string]string{"job-namespace": "fake-namespace"},
						}},
					}},
				},
			},
			assertions: func(t *testing.T, _, _ string, err error) {
				require.ErrorContains(t, err, "has no Job name metadata")
			},
		},
		{
			name: "success",
			run: &rolloutsapi.AnalysisRun{
				Status: rolloutsapi.AnalysisRunStatus{
					MetricResults: []rolloutsapi.MetricResult{{
						Name: testMetricName,
						Measurements: []rolloutsapi.Measurement{{
							Metadata: map[string]string{
								"job-namespace": "fake-namespace",
								"job-name":      "fake-name",
							},
						}},
					}},
				},
			},
			assertions: func(t *testing.T, namespace, name string, err error) {
				require.NoError(t, err)
				require.Equal(t, "fake-namespace", namespace)
				require.Equal(t, "fake-name", name)
			},
		},
	}
	s := &server{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			namespace, name, err := s.getJobNamespaceAndName(testCase.run, testMetricName)
			testCase.assertions(t, namespace, name, err)
		})
	}
}

func TestServer_getStageFromAnalysisRun(t *testing.T) {
	const testNamespace = "fake-namespace"
	const testStageName = "fake-stage"

	testScheme := runtime.NewScheme()
	err := kargoapi.SchemeBuilder.AddToScheme(testScheme)
	require.NoError(t, err)

	testCases := []struct {
		name       string
		run        *rolloutsapi.AnalysisRun
		client     client.WithWatch
		assertions func(t *testing.T, stage *kargoapi.Stage, err error)
	}{
		{
			name: "analysis run is missing stage annotation and label",
			run:  &rolloutsapi.AnalysisRun{},
			assertions: func(t *testing.T, _ *kargoapi.Stage, err error) {
				require.ErrorContains(t, err, "has no stage label")
			},
		},
		{
			name: "error getting stage from annotation",
			run: &rolloutsapi.AnalysisRun{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyStage: testStageName,
					},
				},
			},
			client: fake.NewClientBuilder().WithScheme(testScheme).WithInterceptorFuncs(
				interceptor.Funcs{
					Get: func(
						context.Context,
						client.WithWatch,
						client.ObjectKey,
						client.Object,
						...client.GetOption,
					) error {
						return fmt.Errorf("something went wrong")
					},
				},
			).Build(),
			assertions: func(t *testing.T, _ *kargoapi.Stage, err error) {
				require.ErrorContains(t, err, "error getting Stage")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error getting stage from label",
			run: &rolloutsapi.AnalysisRun{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						kargoapi.LabelKeyStage: testStageName,
					},
				},
			},
			client: fake.NewClientBuilder().WithScheme(testScheme).WithInterceptorFuncs(
				interceptor.Funcs{
					Get: func(
						context.Context,
						client.WithWatch,
						client.ObjectKey,
						client.Object,
						...client.GetOption,
					) error {
						return fmt.Errorf("something went wrong")
					},
				},
			).Build(),
			assertions: func(t *testing.T, _ *kargoapi.Stage, err error) {
				require.ErrorContains(t, err, "error getting Stage")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "stage not found from annotation",
			run: &rolloutsapi.AnalysisRun{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyStage: testStageName,
					},
				},
			},
			client: fake.NewClientBuilder().WithScheme(testScheme).WithInterceptorFuncs(
				interceptor.Funcs{
					Get: func(
						context.Context,
						client.WithWatch,
						client.ObjectKey,
						client.Object,
						...client.GetOption,
					) error {
						return apierrors.NewNotFound(schema.GroupResource{}, "")
					},
				},
			).Build(),
			assertions: func(t *testing.T, _ *kargoapi.Stage, err error) {
				require.ErrorContains(t, err, "not found")
			},
		},
		{
			name: "stage not found from label",
			run: &rolloutsapi.AnalysisRun{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						kargoapi.LabelKeyStage: testStageName,
					},
				},
			},
			client: fake.NewClientBuilder().WithScheme(testScheme).WithInterceptorFuncs(
				interceptor.Funcs{
					Get: func(
						context.Context,
						client.WithWatch,
						client.ObjectKey,
						client.Object,
						...client.GetOption,
					) error {
						return apierrors.NewNotFound(schema.GroupResource{}, "")
					},
				},
			).Build(),
			assertions: func(t *testing.T, _ *kargoapi.Stage, err error) {
				require.ErrorContains(t, err, "not found")
			},
		},
		{
			name: "success with annotation",
			run: &rolloutsapi.AnalysisRun{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Annotations: map[string]string{
						kargoapi.AnnotationKeyStage: testStageName,
					},
				},
			},
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testStageName,
						Namespace: testNamespace,
					},
				},
			).Build(),
			assertions: func(t *testing.T, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				require.NotNil(t, stage)
				require.Equal(t, testNamespace, stage.Namespace)
				require.Equal(t, testStageName, stage.Name)
			},
		},
		{
			name: "success with label",
			run: &rolloutsapi.AnalysisRun{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Labels: map[string]string{
						kargoapi.LabelKeyStage: testStageName,
					},
				},
			},
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testStageName,
						Namespace: testNamespace,
					},
				},
			).Build(),
			assertions: func(t *testing.T, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				require.NotNil(t, stage)
				require.Equal(t, testNamespace, stage.Namespace)
				require.Equal(t, testStageName, stage.Name)
			},
		},
		{
			name: "annotation takes precedence over label",
			run: &rolloutsapi.AnalysisRun{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Annotations: map[string]string{
						kargoapi.AnnotationKeyStage: testStageName,
					},
					Labels: map[string]string{
						kargoapi.LabelKeyStage: "different-stage-name",
					},
				},
			},
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testStageName,
						Namespace: testNamespace,
					},
				},
			).Build(),
			assertions: func(t *testing.T, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				require.NotNil(t, stage)
				require.Equal(t, testNamespace, stage.Namespace)
				require.Equal(t, testStageName, stage.Name) // Should use annotation value, not label value
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			cl, err := kubernetes.NewClient(
				t.Context(),
				nil,
				kubernetes.ClientOptions{
					SkipAuthorization: true,
					NewInternalClient: func(
						context.Context,
						*rest.Config,
						*runtime.Scheme,
					) (client.WithWatch, error) {
						return testCase.client, nil
					},
				},
			)
			require.NoError(t, err)
			s := &server{client: cl}
			stage, err := s.getStageFromAnalysisRun(t.Context(), testCase.run)
			testCase.assertions(t, stage, err)
		})
	}
}

func TestServer_buildRequest(t *testing.T) {
	const (
		testNamespace   = "fake-namespace"
		testAnalysisRun = "fake-analysis-run"
		testURL         = "https://logs.example.com"
	)
	testCases := []struct {
		name           string
		urlTemplate    string
		requestHeaders map[string]string
		assertions     func(t *testing.T, req *http.Request, err error)
	}{
		{
			name:        "evaluated url template is not a string",
			urlTemplate: "${{42}}",
			assertions: func(t *testing.T, _ *http.Request, err error) {
				require.ErrorContains(t, err, "constructed log url")
				require.ErrorContains(t, err, "is not a string")
			},
		},
		{
			name:        "evaluated header template is not a string",
			urlTemplate: testURL,
			requestHeaders: map[string]string{
				"foo": "${{42}}",
			},
			assertions: func(t *testing.T, _ *http.Request, err error) {
				require.ErrorContains(t, err, "constructed value for header")
				require.ErrorContains(t, err, "is not a string")
			},
		},
		{
			name:        "success",
			urlTemplate: "https://logs.example.com/${{ namespace }}/${{ analysisRun }}",
			requestHeaders: map[string]string{
				"ns":       "${{ namespace }}",
				"analysis": "${{ analysisRun }}",
			},
			assertions: func(t *testing.T, req *http.Request, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					fmt.Sprintf("https://logs.example.com/%s/%s",
						testNamespace, testAnalysisRun,
					),
					req.URL.String(),
				)
				require.Equal(t, testNamespace, req.Header.Get("ns"))
				require.Equal(t, testAnalysisRun, req.Header.Get("analysis"))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			s := &server{
				cfg: config.ServerConfig{
					AnalysisRunLogURLTemplate: testCase.urlTemplate,
					AnalysisRunLogHTTPHeaders: testCase.requestHeaders,
				},
			}
			req, err := s.buildRequest(
				t.Context(),
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace},
				},
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNamespace,
						Name:      testAnalysisRun,
					},
				},
				"", "", "", "",
			)
			testCase.assertions(t, req, err)
		})
	}
}

func Test_streamLogs(t *testing.T) {
	// Strings in Go are UTF-8 encoded byte slices.
	// testBytes is also a UTF-8 encoded byte slice.
	testBytes := []byte("😊😊😊") // Emojis use four bytes in both UTF-8 and UTF-16
	testCases := []struct {
		name    string
		encoder *encoding.Encoder
		decoder *encoding.Decoder
	}{
		{
			name:    "no transformation",
			encoder: unicode.UTF8.NewEncoder(),
			decoder: unicode.UTF8.NewDecoder(),
		},
		{
			name:    "transform utf-16 bytes without BOM to utf-8 string",
			encoder: unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewEncoder(), // Don't include BOM
			decoder: unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewDecoder(), // Don't expect BOM
		},
		{
			name:    "transform utf-16 bytes with BOM to utf-8 string",
			encoder: unicode.UTF16(unicode.BigEndian, unicode.UseBOM).NewEncoder(),    // Include BOM
			decoder: unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewDecoder(), // Ignore BOM if present
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Time out in case something doesn't work as expected
			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
			defer cancel()

			// Encode input bytes
			input := make([]byte, len(testBytes)*2) // UTF-16 could use up to 2x the bytes of UTF-8
			nInput, nTestBytes, err := testCase.encoder.Transform(input, testBytes, true)
			require.NoError(t, err)
			require.Equal(t, len(testBytes), nTestBytes)

			// Create a buffered reader with the encoded input
			bufReader := bufio.NewReader(bytes.NewReader(input[:nInput]))

			// Stream logs using the smallest buffer possible to make sure we test
			// that multi-byte encoding sequences spanning buffer boundaries are
			// handled correctly.
			chunkCh, err := streamLogs(ctx, bufReader, testCase.decoder, 256)
			require.NoError(t, err)
			var reassembled string
		loop:
			for {
				select {
				case chunk, ok := <-chunkCh:
					if !ok {
						break loop
					}

					require.NoError(t, chunk.Error, "received unexpected error while streaming logs")

					reassembled += chunk.Data
				case <-ctx.Done():
					require.Fail(t, "timed out")
				}
			}

			// Did we get the original text back?
			require.Equal(t, string(testBytes), reassembled)
		})
	}
}

func Test_server_getAnalysisRunLogs(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	const runName = "fake-analysisrun"
	testRESTEndpoint(
		t, &config.ServerConfig{RolloutsIntegrationEnabled: true},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/analysis-runs/"+runName+"/logs",
		[]restTestCase{
			{
				name:          "Rollouts integration disabled",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				serverConfig:  &config.ServerConfig{RolloutsIntegrationEnabled: false},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotImplemented, w.Code)
				},
			},
			{
				name:          "log streaming not configured",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				serverConfig: &config.ServerConfig{
					RolloutsIntegrationEnabled: true,
					AnalysisRunLogURLTemplate:  "",
					AnalysisRunLogHTTPHeaders:  nil,
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotImplemented, w.Code)
				},
			},
			{
				name: "Project does not exist",
				serverConfig: &config.ServerConfig{
					RolloutsIntegrationEnabled: true,
					AnalysisRunLogURLTemplate:  "http://example.com",
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "AnalysisRun does not exist",
				serverConfig: &config.ServerConfig{
					RolloutsIntegrationEnabled: true,
					AnalysisRunLogURLTemplate:  "http://example.com",
				},
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					// The error from rollouts.GetAnalysisRun gets wrapped, so we get 500
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
		},
	)
}

func Test_server_getAnalysisRunLogs_success(t *testing.T) {
	// Create a test HTTP server that will serve the log content
	testLogContent := "log line 1\nlog line 2\nlog line 3\n"
	logServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(testLogContent))
	}))
	defer logServer.Close()

	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testStage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-stage",
			Namespace: testProject.Name,
		},
	}
	testRun := &rolloutsapi.AnalysisRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      "fake-analysisrun",
			Annotations: map[string]string{
				kargoapi.AnnotationKeyStage: testStage.Name,
			},
		},
		Spec: rolloutsapi.AnalysisRunSpec{
			Metrics: []rolloutsapi.Metric{{
				Name: "test-metric",
				Provider: rolloutsapi.MetricProvider{
					Job: &rolloutsapi.JobMetric{
						Spec: batchv1.JobSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{{Name: "test-container"}},
								},
							},
						},
					},
				},
			}},
		},
		Status: rolloutsapi.AnalysisRunStatus{
			MetricResults: []rolloutsapi.MetricResult{{
				Name: "test-metric",
				Measurements: []rolloutsapi.Measurement{{
					Metadata: map[string]string{
						"job-namespace": testProject.Name,
						"job-name":      "test-job",
					},
				}},
			}},
		},
	}

	testRESTEndpoint(
		t, &config.ServerConfig{RolloutsIntegrationEnabled: true},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/analysis-runs/"+testRun.Name+"/logs",
		[]restTestCase{
			{
				name: "streams logs successfully",
				serverConfig: &config.ServerConfig{
					RolloutsIntegrationEnabled: true,
					// Use a template that just returns the log server URL
					// The template needs to be valid expression syntax
					AnalysisRunLogURLTemplate: "${{ \"" + logServer.URL + "\" }}",
				},
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testStage, testRun),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					if w.Code != http.StatusOK {
						t.Logf("Response body: %s", w.Body.String())
					}
					require.Equal(t, http.StatusOK, w.Code)

					// Verify SSE headers
					require.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
					require.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
					require.Equal(t, "keep-alive", w.Header().Get("Connection"))

					// The response body should contain the log content wrapped in SSE format
					body := w.Body.String()
					require.Contains(t, body, "data:")
					// Each line should be in an SSE data event
					require.Contains(t, body, "log line 1")
					require.Contains(t, body, "log line 2")
					require.Contains(t, body, "log line 3")
				},
			},
		},
	)
}
