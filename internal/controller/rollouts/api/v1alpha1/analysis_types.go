package v1alpha1

import (
	"encoding/json"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	intstrutil "k8s.io/apimachinery/pkg/util/intstr"
)

//+kubebuilder:object:root=true

type AnalysisTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              AnalysisTemplateSpec `json:"spec"`
}

//+kubebuilder:object:root=true

type AnalysisTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []AnalysisTemplate `json:"items"`
}

type AnalysisTemplateSpec struct {
	Metrics              []Metric               `json:"metrics"`
	Args                 []Argument             `json:"args,omitempty"`
	DryRun               []DryRun               `json:"dryRun,omitempty"`
	MeasurementRetention []MeasurementRetention `json:"measurementRetention,omitempty"`
}

type DurationString string

func (d DurationString) Duration() (time.Duration, error) {
	return time.ParseDuration(string(d))
}

type Metric struct {
	Name                  string                  `json:"name"`
	Interval              DurationString          `json:"interval,omitempty"`
	InitialDelay          DurationString          `json:"initialDelay,omitempty"`
	Count                 *intstrutil.IntOrString `json:"count,omitempty"`
	SuccessCondition      string                  `json:"successCondition,omitempty"`
	FailureCondition      string                  `json:"failureCondition,omitempty"`
	FailureLimit          *intstrutil.IntOrString `json:"failureLimit,omitempty"`
	InconclusiveLimit     *intstrutil.IntOrString `json:"inconclusiveLimit,omitempty"`
	ConsecutiveErrorLimit *intstrutil.IntOrString `json:"consecutiveErrorLimit,omitempty"`
	Provider              MetricProvider          `json:"provider"`
}

type DryRun struct {
	MetricName string `json:"metricName"`
}

type MeasurementRetention struct {
	MetricName string `json:"metricName"`
	Limit      int32  `json:"limit"`
}

type MetricProvider struct {
	Prometheus *PrometheusMetric          `json:"prometheus,omitempty"`
	Kayenta    *KayentaMetric             `json:"kayenta,omitempty"`
	Web        *WebMetric                 `json:"web,omitempty"`
	Datadog    *DatadogMetric             `json:"datadog,omitempty"`
	Wavefront  *WavefrontMetric           `json:"wavefront,omitempty"`
	NewRelic   *NewRelicMetric            `json:"newRelic,omitempty"`
	Job        *JobMetric                 `json:"job,omitempty"`
	CloudWatch *CloudWatchMetric          `json:"cloudWatch,omitempty"`
	Graphite   *GraphiteMetric            `json:"graphite,omitempty"`
	Influxdb   *InfluxdbMetric            `json:"influxdb,omitempty"`
	SkyWalking *SkyWalkingMetric          `json:"skywalking,omitempty"`
	Plugin     map[string]json.RawMessage `json:"plugin,omitempty"`
}

type AnalysisPhase string

const (
	AnalysisPhasePending      AnalysisPhase = "Pending"
	AnalysisPhaseRunning      AnalysisPhase = "Running"
	AnalysisPhaseSuccessful   AnalysisPhase = "Successful"
	AnalysisPhaseFailed       AnalysisPhase = "Failed"
	AnalysisPhaseError        AnalysisPhase = "Error"
	AnalysisPhaseInconclusive AnalysisPhase = "Inconclusive"
)

// Completed returns whether or not the analysis status is considered completed
func (as AnalysisPhase) Completed() bool {
	switch as {
	case AnalysisPhaseSuccessful, AnalysisPhaseFailed, AnalysisPhaseError, AnalysisPhaseInconclusive:
		return true
	}
	return false
}

type PrometheusMetric struct {
	Address        string            `json:"address,omitempty"`
	Query          string            `json:"query,omitempty"`
	Authentication Authentication    `json:"authentication,omitempty"`
	Timeout        *int64            `json:"timeout,omitempty"`
	Insecure       bool              `json:"insecure,omitempty"`
	Headers        []WebMetricHeader `json:"headers,omitempty"`
}

type Authentication struct {
	Sigv4  Sigv4Config  `json:"sigv4,omitempty"`
	OAuth2 OAuth2Config `json:"oauth2,omitempty"`
}

type OAuth2Config struct {
	TokenURL     string   `json:"tokenUrl,omitempty"`
	ClientID     string   `json:"clientId,omitempty"`
	ClientSecret string   `json:"clientSecret,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
}

type Sigv4Config struct {
	Region  string `json:"region,omitempty"`
	Profile string `json:"profile,omitempty"`
	RoleARN string `json:"roleArn,omitempty"`
}

type WavefrontMetric struct {
	Address string `json:"address,omitempty"`
	Query   string `json:"query,omitempty"`
}

type NewRelicMetric struct {
	Profile string `json:"profile,omitempty"`
	Query   string `json:"query"`
}

type JobMetric struct {
	Metadata metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec     batchv1.JobSpec   `json:"spec"`
}

type GraphiteMetric struct {
	Address string `json:"address,omitempty"`
	Query   string `json:"query,omitempty"`
}

type InfluxdbMetric struct {
	Profile string `json:"profile,omitempty"`
	Query   string `json:"query,omitempty"`
}

type CloudWatchMetric struct {
	Interval          DurationString              `json:"interval,omitempty"`
	MetricDataQueries []CloudWatchMetricDataQuery `json:"metricDataQueries"`
}

type CloudWatchMetricDataQuery struct {
	Id         string                  `json:"id,omitempty"`
	Expression *string                 `json:"expression,omitempty"`
	Label      *string                 `json:"label,omitempty"`
	MetricStat *CloudWatchMetricStat   `json:"metricStat,omitempty"`
	Period     *intstrutil.IntOrString `json:"period,omitempty"`
	ReturnData *bool                   `json:"returnData,omitempty"`
}

type CloudWatchMetricStat struct {
	Metric CloudWatchMetricStatMetric `json:"metric,omitempty"`
	Period intstrutil.IntOrString     `json:"period,omitempty"`
	Stat   string                     `json:"stat,omitempty"`
	Unit   string                     `json:"unit,omitempty"`
}

type CloudWatchMetricStatMetric struct {
	Dimensions []CloudWatchMetricStatMetricDimension `json:"dimensions,omitempty"`
	MetricName string                                `json:"metricName,omitempty"`
	Namespace  *string                               `json:"namespace,omitempty"`
}

type CloudWatchMetricStatMetricDimension struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

type AnalysisRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              AnalysisRunSpec   `json:"spec"`
	Status            AnalysisRunStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

type AnalysisRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []AnalysisRun `json:"items"`
}

type SkyWalkingMetric struct {
	Address  string         `json:"address,omitempty"`
	Query    string         `json:"query,omitempty"`
	Interval DurationString `json:"interval,omitempty"`
}

type AnalysisRunSpec struct {
	Metrics              []Metric               `json:"metrics"`
	Args                 []Argument             `json:"args,omitempty"`
	Terminate            bool                   `json:"terminate,omitempty"`
	DryRun               []DryRun               `json:"dryRun,omitempty"`
	MeasurementRetention []MeasurementRetention `json:"measurementRetention,omitempty"`
}

type Argument struct {
	Name      string     `json:"name"`
	Value     *string    `json:"value,omitempty"`
	ValueFrom *ValueFrom `json:"valueFrom,omitempty"`
}

type ValueFrom struct {
	SecretKeyRef *SecretKeyRef `json:"secretKeyRef,omitempty"`
	FieldRef     *FieldRef     `json:"fieldRef,omitempty"`
}

type SecretKeyRef struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type AnalysisRunStatus struct {
	Phase         AnalysisPhase  `json:"phase"`
	Message       string         `json:"message,omitempty"`
	MetricResults []MetricResult `json:"metricResults,omitempty"`
	StartedAt     *metav1.Time   `json:"startedAt,omitempty"`
	RunSummary    RunSummary     `json:"runSummary,omitempty"`
	DryRunSummary *RunSummary    `json:"dryRunSummary,omitempty"`
}

type RunSummary struct {
	Count        int32 `json:"count,omitempty"`
	Successful   int32 `json:"successful,omitempty"`
	Failed       int32 `json:"failed,omitempty"`
	Inconclusive int32 `json:"inconclusive,omitempty"`
	Error        int32 `json:"error,omitempty"`
}

type MetricResult struct {
	Name             string            `json:"name"`
	Phase            AnalysisPhase     `json:"phase"`
	Measurements     []Measurement     `json:"measurements,omitempty"`
	Message          string            `json:"message,omitempty"`
	Count            int32             `json:"count,omitempty"`
	Successful       int32             `json:"successful,omitempty"`
	Failed           int32             `json:"failed,omitempty"`
	Inconclusive     int32             `json:"inconclusive,omitempty"`
	Error            int32             `json:"error,omitempty"`
	ConsecutiveError int32             `json:"consecutiveError,omitempty"`
	DryRun           bool              `json:"dryRun,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

type Measurement struct {
	Phase      AnalysisPhase     `json:"phase"`
	Message    string            `json:"message,omitempty"`
	StartedAt  *metav1.Time      `json:"startedAt,omitempty"`
	FinishedAt *metav1.Time      `json:"finishedAt,omitempty"`
	Value      string            `json:"value,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	ResumeAt   *metav1.Time      `json:"resumeAt,omitempty"`
}

type KayentaMetric struct {
	Address                  string           `json:"address"`
	Application              string           `json:"application"`
	CanaryConfigName         string           `json:"canaryConfigName"`
	MetricsAccountName       string           `json:"metricsAccountName"`
	ConfigurationAccountName string           `json:"configurationAccountName"`
	StorageAccountName       string           `json:"storageAccountName"`
	Threshold                KayentaThreshold `json:"threshold"`
	Scopes                   []KayentaScope   `json:"scopes"`
}

type KayentaThreshold struct {
	Pass     int64 `json:"pass"`
	Marginal int64 `json:"marginal"`
}

type KayentaScope struct {
	Name            string      `json:"name"`
	ControlScope    ScopeDetail `json:"controlScope"`
	ExperimentScope ScopeDetail `json:"experimentScope"`
}

type ScopeDetail struct {
	Scope  string `json:"scope"`
	Region string `json:"region"`
	Step   int64  `json:"step"`
	Start  string `json:"start"`
	End    string `json:"end"`
}

type WebMetric struct {
	Method WebMetricMethod `json:"method,omitempty"`
	// URL is the address of the web metric
	URL            string            `json:"url"`
	Headers        []WebMetricHeader `json:"headers,omitempty"`
	Body           string            `json:"body,omitempty"`
	TimeoutSeconds int64             `json:"timeoutSeconds,omitempty"`
	JSONPath       string            `json:"jsonPath,omitempty"`
	Insecure       bool              `json:"insecure,omitempty"`
	JSONBody       json.RawMessage   `json:"jsonBody,omitempty"`
	Authentication Authentication    `json:"authentication,omitempty"`
}

type WebMetricMethod string

type WebMetricHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type DatadogMetric struct {
	Interval   DurationString    `json:"interval,omitempty"`
	Query      string            `json:"query,omitempty"`
	Queries    map[string]string `json:"queries,omitempty"`
	Formula    string            `json:"formula,omitempty"`
	ApiVersion string            `json:"apiVersion,omitempty"`
}
