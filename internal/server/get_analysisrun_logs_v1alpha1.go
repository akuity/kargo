package server

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"connectrpc.com/connect"
	"github.com/hashicorp/go-cleanhttp"
	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"
	"k8s.io/apimachinery/pkg/types"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	rolloutsapi "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
	"github.com/akuity/kargo/internal/api/stubs/rollouts"
	libEncoding "github.com/akuity/kargo/internal/encoding"
	"github.com/akuity/kargo/internal/expressions"
	"github.com/akuity/kargo/internal/server/user"
)

func (s *server) GetAnalysisRunLogs(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetAnalysisRunLogsRequest],
	stream *connect.ServerStream[svcv1alpha1.GetAnalysisRunLogsResponse],
) error {
	if !s.cfg.RolloutsIntegrationEnabled {
		return connect.NewError(
			connect.CodeUnimplemented,
			errors.New("Argo Rollouts integration is not enabled"),
		)
	}

	if s.cfg.AnalysisRunLogURLTemplate == "" {
		return connect.NewError(
			connect.CodeUnimplemented,
			errors.New("AnalysisRun log streaming is not configured"),
		)
	}

	namespace := req.Msg.GetNamespace()
	if err := validateFieldNotEmpty("namespace", namespace); err != nil {
		return err
	}

	name := req.Msg.GetName()
	if err := validateFieldNotEmpty("name", name); err != nil {
		return err
	}

	analysisRun, err := rollouts.GetAnalysisRun(
		ctx,
		s.client,
		types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
	)
	if err != nil {
		return fmt.Errorf(
			"error getting AnalysisRun %q in namespace %q: %w",
			name, namespace, err,
		)
	}
	if analysisRun == nil {
		return connect.NewError(
			connect.CodeNotFound,
			fmt.Errorf("AnalysisRun %q in namespace %q not found", name, namespace),
		)
	}

	// Don't stream logs for an AnalysisRun that is not complete.
	if !analysisRun.Status.Phase.Completed() {
		return connect.NewError(
			connect.CodeFailedPrecondition,
			fmt.Errorf(
				"AnalysisRun %q in namespace %q is not complete; cannot retrieve logs",
				name, namespace,
			),
		)
	}

	jobMetricName, jobMetric, err := s.getJobMetric(analysisRun, req.Msg.MetricName)
	if err != nil {
		return err
	}

	containerName, err := s.getContainerName(
		analysisRun,
		jobMetricName,
		jobMetric,
		req.Msg.ContainerName,
	)
	if err != nil {
		return err
	}

	jobNamespace, jobName, err := s.getJobNamespaceAndName(analysisRun, jobMetricName)
	if err != nil {
		return err
	}

	stage, err := s.getStageFromAnalysisRun(ctx, analysisRun)
	if err != nil {
		return err
	}

	httpReq, err := s.buildRequest(
		ctx,
		stage,
		analysisRun,
		jobMetricName,
		jobNamespace,
		jobName,
		containerName,
	)
	if err != nil {
		return err
	}

	httpResp, err := cleanhttp.DefaultClient().Do(httpReq)
	if err != nil {
		return fmt.Errorf(
			"error performing GET request for log url %s: %w",
			httpReq.URL.String(), err,
		)
	}
	defer httpResp.Body.Close()

	// Logs can be large, so we read them using a buffered reader.
	reader := bufio.NewReader(httpResp.Body)

	const bufferSize = 4096 // 4 KB

	peekedBytes, err := reader.Peek(bufferSize)
	if err != nil && err != io.EOF {
		return fmt.Errorf("error peeking at log stream: %w", err)
	}

	// Log data has a higher than average probability of being encoded with
	// something other than UTF-8.
	enc := libEncoding.DetectEncoding(httpResp.Header.Get("Content-Type"), peekedBytes)

	logCh, err := streamLogs(ctx, reader, enc.NewDecoder(), bufferSize)
	if err != nil {
		return fmt.Errorf("error streaming logs: %w", err)
	}

	for {
		select {
		case chunk, ok := <-logCh:
			if !ok {
				// Channel closed
				return nil
			}

			if chunk.Error != nil {
				// Error reading log data
				return fmt.Errorf("error streaming logs: %w", chunk.Error)
			}

			if err = stream.Send(&svcv1alpha1.GetAnalysisRunLogsResponse{
				Chunk: chunk.Data,
			}); err != nil {
				return fmt.Errorf("error sending log chunk: %w", err)
			}
		case <-ctx.Done():
			// Context canceled or timed out
			return ctx.Err()
		}
	}
}

// getJobMetric confirms the existence of a JobMetric with the provided name or,
// when the provided name is empty, attempts to infer one, which can only
// succeed when the AnalysisRun has EXACTLY one JobMetric. If a JobMetric is
// found or inferred, its name and the JobMetric itself are returned. If a
// JobMetric is not found or inferred, for any reason, an error is returned.
func (s *server) getJobMetric(
	run *rolloutsapi.AnalysisRun,
	jobMetricName string,
) (string, *rolloutsapi.JobMetric, error) {
	jobMetrics := make(map[string]*rolloutsapi.JobMetric)
	for _, metric := range run.Spec.Metrics {
		if metric.Provider.Job != nil {
			if jobMetricName != "" && metric.Name == jobMetricName {
				// If we know the name of the metric we want, we can return it as soon
				// as we find it.
				return jobMetricName, metric.Provider.Job, nil
			}
			jobMetrics[metric.Name] = metric.Provider.Job
		}
	}
	if jobMetricName != "" {
		return "", nil, connect.NewError(
			connect.CodeNotFound,
			fmt.Errorf(
				"AnalysisRun %q in namespace %q has no job metric named %q",
				run.Name, run.Namespace, jobMetricName,
			),
		)
	}
	if len(jobMetrics) == 0 {
		return "", nil, connect.NewError(
			connect.CodeNotFound,
			fmt.Errorf(
				"AnalysisRun %q in namespace %q has no job metrics",
				run.Name, run.Namespace,
			),
		)
	}
	if len(jobMetrics) > 1 {
		return "", nil, connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf(
				"AnalysisRun %q in namespace %q has multiple job metrics; please specify a metric name",
				run.Name, run.Namespace,
			),
		)
	}
	// If we get to here, there is exactly one job metric.
	var jobMetric *rolloutsapi.JobMetric
	for jobMetricName, jobMetric = range jobMetrics {
		break
	}
	return jobMetricName, jobMetric, nil
}

// getContainerName confirms the existence of a container in the provided
// JobMetric's pod template having the provided name or, when the provided name
// is empty, attempts to infer one, which can only succeed when the JobMetric's
// pod template has EXACTLY one container. If a container name is confirmed or
// inferred, it is returned. If a container name is not confirmed or inferred,
// for any reason, an error is returned.
func (s *server) getContainerName(
	run *rolloutsapi.AnalysisRun,
	jobMetricName string,
	jobMetric *rolloutsapi.JobMetric,
	containerName string,
) (string, error) {
	if len(jobMetric.Spec.Template.Spec.Containers) == 0 {
		// This probably isn't possible, but we'll check...
		return "", connect.NewError(
			connect.CodeNotFound,
			fmt.Errorf(
				"AnalysisRun %q in namespace %q has no containers in Jobs for metric %q",
				run.Name, run.Namespace, jobMetricName,
			),
		)
	}
	containerNames := make(map[string]struct{}, len(jobMetric.Spec.Template.Spec.Containers))
	for _, container := range jobMetric.Spec.Template.Spec.Containers {
		if containerName != "" && container.Name == containerName {
			// If we know the name of the container we want, we can return it as soon
			// as we confirm it exists.
			return containerName, nil
		}
		containerNames[container.Name] = struct{}{}
	}
	if containerName != "" {
		return "", connect.NewError(
			connect.CodeNotFound,
			fmt.Errorf(
				"AnalysisRun %q in namespace %q has no container named %q in Jobs for metric %q",
				run.Name, run.Namespace, containerName, jobMetricName,
			),
		)
	}
	if len(containerNames) > 1 {
		return "", connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf(
				"AnalysisRun %q in namespace %q has multiple containers in Jobs for metric %q; please specify a container name",
				run.Name, run.Namespace, jobMetricName,
			),
		)
	}
	// If we get to here, there is exactly one container.
	for containerName = range containerNames {
		break
	}
	return containerName, nil
}

// getJobNamespaceAndName extracts the namespace and name of the Job instance
// associated with the provided AnalysisRun and JobMetric name. If these cannot
// be determined, for any reason, an error is returned.
func (s *server) getJobNamespaceAndName(
	run *rolloutsapi.AnalysisRun,
	jobMetricName string,
) (string, string, error) {
	var metricResult *rolloutsapi.MetricResult
	for _, mr := range run.Status.MetricResults {
		if mr.Name == jobMetricName {
			metricResult = &mr
			break
		}
	}
	if metricResult == nil {
		return "", "", connect.NewError(
			connect.CodeNotFound,
			fmt.Errorf(
				"AnalysisRun %q in namespace %q has no result for metric  %q",
				run.Name, run.Namespace, jobMetricName,
			),
		)
	}
	if len(metricResult.Measurements) == 0 {
		return "", "", connect.NewError(
			connect.CodeNotFound,
			fmt.Errorf("result for metric %q has no measurements", jobMetricName),
		)
	}
	// TODO(krancour): Under what circumstances would there be more than one
	// measurement? Ask jessesuen.
	jobNamespace := metricResult.Measurements[0].Metadata["job-namespace"]
	if jobNamespace == "" {
		return "", "", connect.NewError(
			connect.CodeNotFound,
			fmt.Errorf("result for metric %q has no Job namespace metadata", jobMetricName),
		)
	}
	jobName := metricResult.Measurements[0].Metadata["job-name"]
	if jobName == "" {
		return "", "", connect.NewError(
			connect.CodeNotFound,
			fmt.Errorf("result for metric %q has no Job name metadata", jobMetricName),
		)
	}
	return jobNamespace, jobName, nil
}

// getStageFromAnalysisRun determines the Stage associated with the provided
// AnalysisRun. If that can be determined, the Stage itself is returned. If the
// Stage cannot be determined, for any reason, or cannot be retrieved, an error
// is returned.
func (s *server) getStageFromAnalysisRun(
	ctx context.Context,
	run *rolloutsapi.AnalysisRun,
) (*kargoapi.Stage, error) {
	stageName, ok := run.Labels[kargoapi.StageLabelKey]
	if !ok {
		return nil, connect.NewError(
			connect.CodeNotFound,
			fmt.Errorf(
				"AnalysisRun %q in namespace %q has no stage label",
				run.Name, run.Namespace,
			),
		)
	}
	stage, err := api.GetStage(
		ctx,
		s.client,
		types.NamespacedName{
			Namespace: run.Namespace,
			Name:      stageName,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"error getting Stage %q in namespace %q: %w",
			stageName, run.Namespace, err,
		)
	}
	if stage == nil {
		return nil, connect.NewError(
			connect.CodeNotFound,
			fmt.Errorf("Stage %q in namespace %q not found", stageName, run.Namespace),
		)
	}
	return stage, nil
}

// buildRequest constructs an HTTP GET request using the provided details, which
// are themselves, used in evaluation of a URL template. The request is
// returned. If it is not successfully constructed, an error is returned.
func (s *server) buildRequest(
	ctx context.Context,
	stage *kargoapi.Stage,
	run *rolloutsapi.AnalysisRun,
	jobMetricName, jobNamespace, jobName, containerName string,
) (*http.Request, error) {
	env := map[string]any{
		"project":      stage.Namespace,
		"namespace":    stage.Namespace,
		"shard":        stage.Spec.Shard,
		"stage":        stage.Name,
		"analysisRun":  run.Name,
		"metricName":   jobMetricName,
		"jobNamespace": jobNamespace,
		"jobName":      jobName,
		"container":    containerName,
	}
	urlAny, err := expressions.EvaluateTemplate(s.cfg.AnalysisRunLogURLTemplate, env)
	if err != nil {
		return nil, fmt.Errorf("error constructing log url: %w", err)
	}
	url, ok := urlAny.(string)
	if !ok {
		// There is a very small, but non-zero chance of this happening. Expression
		// evaluation will return a boolean, number, list, or object if the result
		// is marshalable as any of those things, and returns a string otherwise.
		// With an egregiously malformed template, the result could be a non-string
		// type.
		return nil, fmt.Errorf("constructed log url %v is not a string", urlAny)
	}
	httpReq, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating GET request for log url %s: %w", url, err)
	}
	if s.cfg.AnalysisRunLogToken != "" {
		env["token"] = s.cfg.AnalysisRunLogToken
	} else if userInfo, ok := user.InfoFromContext(ctx); ok {
		env["token"] = userInfo.BearerToken
	}
	for key, valTemplate := range s.cfg.AnalysisRunLogHTTPHeaders {
		valTemplateAny, err := expressions.EvaluateTemplate(valTemplate, env)
		if err != nil {
			return nil, fmt.Errorf("error constructing value for header %s: %w", key, err)
		}
		val, ok := valTemplateAny.(string)
		if !ok {
			return nil, fmt.Errorf("constructed value for header %s is not a string", key)
		}
		httpReq.Header.Set(key, val)
	}
	return httpReq, nil
}

// logChunk represents a chunk of log data or an error.
type logChunk struct {
	Data  string
	Error error
}

// streamLogs reads log data from the provided reader, decodes it using the
// specified decoder, and returns a channel that receives chunks of log data.
// The channel is closed when all data has been read or an error occurs.
func streamLogs(
	ctx context.Context,
	reader *bufio.Reader,
	decoder *encoding.Decoder,
	bufferSize int,
) (<-chan logChunk, error) {
	// Special case: We only use UTF-16 decoders that ignore the BOM, but that
	// only means they do not REQUIRE there to be a BOM at the beginning of the
	// stream. If it's there, it still gets decoded. A UTF-16 BOM is an invisible
	// character in UTF-8, but it's still there. We don't want it.
	//
	// A UTF-16 BOM is two bytes. Peek at the first two bytes of the stream.
	peekedBytes, err := reader.Peek(2)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("error peeking at log stream: %w", err)
	}

	if libEncoding.HasUTF16BOM(peekedBytes) {
		if _, err = reader.Discard(2); err != nil {
			return nil, fmt.Errorf("error discarding BOM: %w", err)
		}
	}

	transformReader := transform.NewReader(reader, decoder)
	resultCh := make(chan logChunk)

	go func() {
		defer close(resultCh)

		buf := make([]byte, bufferSize)

		for {
			n, err := transformReader.Read(buf)
			if n > 0 {
				select {
				case resultCh <- logChunk{Data: string(buf[:n])}:
				case <-ctx.Done():
					return
				}
			}
			if err != nil {
				if err == io.EOF {
					return
				}
				select {
				case resultCh <- logChunk{Error: fmt.Errorf("error reading data: %w", err)}:
				case <-ctx.Done():
				}
				return
			}
		}
	}()

	return resultCh, nil
}
