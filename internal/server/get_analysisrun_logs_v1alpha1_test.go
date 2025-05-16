package server

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	rolloutsapi "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/server/config"
	"github.com/akuity/kargo/internal/server/kubernetes"
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
		client     client.Client
		assertions func(t *testing.T, stage *kargoapi.Stage, err error)
	}{
		{
			name: "analysis run is missing stage label",
			run:  &rolloutsapi.AnalysisRun{},
			assertions: func(t *testing.T, _ *kargoapi.Stage, err error) {
				require.ErrorContains(t, err, "has no stage label")
			},
		},
		{
			name: "error getting stage",
			run: &rolloutsapi.AnalysisRun{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						kargoapi.StageLabelKey: testStageName,
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
			name: "stage not found",
			run: &rolloutsapi.AnalysisRun{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						kargoapi.StageLabelKey: testStageName,
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
						return kubeerr.NewNotFound(schema.GroupResource{}, "")
					},
				},
			).Build(),
			assertions: func(t *testing.T, _ *kargoapi.Stage, err error) {
				require.ErrorContains(t, err, "not found")
			},
		},
		{
			name: "success",
			run: &rolloutsapi.AnalysisRun{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Labels: map[string]string{
						kargoapi.StageLabelKey: testStageName,
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
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			cl, err := kubernetes.NewClient(
				context.Background(),
				nil,
				kubernetes.ClientOptions{
					SkipAuthorization: true,
					NewInternalClient: func(context.Context, *rest.Config, *runtime.Scheme) (client.Client, error) {
						return testCase.client, nil
					},
					NewInternalDynamicClient: func(*rest.Config) (dynamic.Interface, error) {
						return nil, nil
					},
				},
			)
			require.NoError(t, err)
			s := &server{client: cl}
			stage, err := s.getStageFromAnalysisRun(context.Background(), testCase.run)
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
				context.Background(),
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
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
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
