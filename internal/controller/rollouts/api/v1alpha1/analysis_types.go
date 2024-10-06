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
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              AnalysisTemplateSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

//+kubebuilder:object:root=true

type AnalysisTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Items           []AnalysisTemplate `json:"items" protobuf:"bytes,2,rep,name=items"`
}

//+kubebuilder:object:root=true

type ClusterAnalysisTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              AnalysisTemplateSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

//+kubebuilder:object:root=true

type ClusterAnalysisTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Items           []ClusterAnalysisTemplate `json:"items" protobuf:"bytes,2,rep,name=items"`
}

type AnalysisTemplateSpec struct {
	Metrics              []Metric               `json:"metrics" protobuf:"bytes,1,rep,name=metrics"`
	Args                 []Argument             `json:"args,omitempty" protobuf:"bytes,2,rep,name=args"`
	DryRun               []DryRun               `json:"dryRun,omitempty" protobuf:"bytes,3,rep,name=dryRun"`
	MeasurementRetention []MeasurementRetention `json:"measurementRetention,omitempty" protobuf:"bytes,4,rep,name=measurementRetention"`
}

type DurationString string

func (d DurationString) Duration() (time.Duration, error) {
	return time.ParseDuration(string(d))
}

type Metric struct {
	Name                  string                  `json:"name" protobuf:"bytes,1,opt,name=name"`
	Interval              DurationString          `json:"interval,omitempty" protobuf:"bytes,2,opt,name=interval,casttype=DurationString"`
	InitialDelay          DurationString          `json:"initialDelay,omitempty" protobuf:"bytes,3,opt,name=initialDelay,casttype=DurationString"`
	Count                 *intstrutil.IntOrString `json:"count,omitempty" protobuf:"bytes,4,opt,name=count"`
	SuccessCondition      string                  `json:"successCondition,omitempty" protobuf:"bytes,5,opt,name=successCondition"`
	FailureCondition      string                  `json:"failureCondition,omitempty" protobuf:"bytes,6,opt,name=failureCondition"`
	FailureLimit          *intstrutil.IntOrString `json:"failureLimit,omitempty" protobuf:"bytes,7,opt,name=failureLimit"`
	InconclusiveLimit     *intstrutil.IntOrString `json:"inconclusiveLimit,omitempty" protobuf:"bytes,8,opt,name=inconclusiveLimit"`
	ConsecutiveErrorLimit *intstrutil.IntOrString `json:"consecutiveErrorLimit,omitempty" protobuf:"bytes,9,opt,name=consecutiveErrorLimit"`
	Provider              MetricProvider          `json:"provider" protobuf:"bytes,10,opt,name=provider"`
}

type DryRun struct {
	MetricName string `json:"metricName" protobuf:"bytes,1,opt,name=metricName"`
}

type MeasurementRetention struct {
	MetricName string `json:"metricName" protobuf:"bytes,1,opt,name=metricName"`
	Limit      int32  `json:"limit" protobuf:"varint,2,opt,name=limit"`
}

type MetricProvider struct {
	Prometheus *PrometheusMetric          `json:"prometheus,omitempty" protobuf:"bytes,1,opt,name=prometheus"`
	Kayenta    *KayentaMetric             `json:"kayenta,omitempty" protobuf:"bytes,2,opt,name=kayenta"`
	Web        *WebMetric                 `json:"web,omitempty" protobuf:"bytes,3,opt,name=web"`
	Datadog    *DatadogMetric             `json:"datadog,omitempty" protobuf:"bytes,4,opt,name=datadog"`
	Wavefront  *WavefrontMetric           `json:"wavefront,omitempty" protobuf:"bytes,5,opt,name=wavefront"`
	NewRelic   *NewRelicMetric            `json:"newRelic,omitempty" protobuf:"bytes,6,opt,name=newRelic"`
	Job        *JobMetric                 `json:"job,omitempty" protobuf:"bytes,7,opt,name=job"`
	CloudWatch *CloudWatchMetric          `json:"cloudWatch,omitempty" protobuf:"bytes,8,opt,name=cloudWatch"`
	Graphite   *GraphiteMetric            `json:"graphite,omitempty" protobuf:"bytes,9,opt,name=graphite"`
	Influxdb   *InfluxdbMetric            `json:"influxdb,omitempty" protobuf:"bytes,10,opt,name=influxdb"`
	SkyWalking *SkyWalkingMetric          `json:"skywalking,omitempty" protobuf:"bytes,11,opt,name=skywalking"`
	Plugin     map[string]json.RawMessage `json:"plugin,omitempty" protobuf:"bytes,12,rep,name=plugin,castvalue=encoding/json.RawMessage"`
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
	Address        string            `json:"address,omitempty" protobuf:"bytes,1,opt,name=address"`
	Query          string            `json:"query,omitempty" protobuf:"bytes,2,opt,name=query"`
	Authentication Authentication    `json:"authentication,omitempty" protobuf:"bytes,3,opt,name=authentication"`
	Timeout        *int64            `json:"timeout,omitempty" protobuf:"varint,4,opt,name=timeout"`
	Insecure       bool              `json:"insecure,omitempty" protobuf:"varint,5,opt,name=insecure"`
	Headers        []WebMetricHeader `json:"headers,omitempty" protobuf:"bytes,6,rep,name=headers"`
}

type Authentication struct {
	Sigv4  Sigv4Config  `json:"sigv4,omitempty" protobuf:"bytes,1,opt,name=sigv4"`
	OAuth2 OAuth2Config `json:"oauth2,omitempty" protobuf:"bytes,2,opt,name=oauth2"`
}

type OAuth2Config struct {
	TokenURL     string   `json:"tokenUrl,omitempty" protobuf:"bytes,1,opt,name=tokenUrl"`
	ClientID     string   `json:"clientId,omitempty" protobuf:"bytes,2,opt,name=clientId"`
	ClientSecret string   `json:"clientSecret,omitempty" protobuf:"bytes,3,opt,name=clientSecret"`
	Scopes       []string `json:"scopes,omitempty" protobuf:"bytes,4,rep,name=scopes"`
}

type Sigv4Config struct {
	Region  string `json:"region,omitempty" protobuf:"bytes,1,opt,name=region"`
	Profile string `json:"profile,omitempty" protobuf:"bytes,2,opt,name=profile"`
	RoleARN string `json:"roleArn,omitempty" protobuf:"bytes,3,opt,name=roleArn"`
}

type WavefrontMetric struct {
	Address string `json:"address,omitempty" protobuf:"bytes,1,opt,name=address"`
	Query   string `json:"query,omitempty" protobuf:"bytes,2,opt,name=query"`
}

type NewRelicMetric struct {
	Profile string `json:"profile,omitempty" protobuf:"bytes,1,opt,name=profile"`
	Query   string `json:"query" protobuf:"bytes,2,opt,name=query"`
}

type JobMetric struct {
	Metadata metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec     batchv1.JobSpec   `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

type GraphiteMetric struct {
	Address string `json:"address,omitempty" protobuf:"bytes,1,opt,name=address"`
	Query   string `json:"query,omitempty" protobuf:"bytes,2,opt,name=query"`
}

type InfluxdbMetric struct {
	Profile string `json:"profile,omitempty" protobuf:"bytes,1,opt,name=profile"`
	Query   string `json:"query,omitempty" protobuf:"bytes,2,opt,name=query"`
}

type CloudWatchMetric struct {
	Interval          DurationString              `json:"interval,omitempty" protobuf:"bytes,1,opt,name=interval,casttype=DurationString"`
	MetricDataQueries []CloudWatchMetricDataQuery `json:"metricDataQueries" protobuf:"bytes,2,rep,name=metricDataQueries"`
}

type CloudWatchMetricDataQuery struct {
	Id         string                  `json:"id,omitempty" protobuf:"bytes,1,opt,name=id"`
	Expression *string                 `json:"expression,omitempty" protobuf:"bytes,2,opt,name=expression"`
	Label      *string                 `json:"label,omitempty" protobuf:"bytes,3,opt,name=label"`
	MetricStat *CloudWatchMetricStat   `json:"metricStat,omitempty" protobuf:"bytes,4,opt,name=metricStat"`
	Period     *intstrutil.IntOrString `json:"period,omitempty" protobuf:"bytes,5,opt,name=period"`
	ReturnData *bool                   `json:"returnData,omitempty" protobuf:"varint,6,opt,name=returnData"`
}

type CloudWatchMetricStat struct {
	Metric CloudWatchMetricStatMetric `json:"metric,omitempty" protobuf:"bytes,1,opt,name=metric"`
	Period intstrutil.IntOrString     `json:"period,omitempty" protobuf:"bytes,2,opt,name=period"`
	Stat   string                     `json:"stat,omitempty" protobuf:"bytes,3,opt,name=stat"`
	Unit   string                     `json:"unit,omitempty" protobuf:"bytes,4,opt,name=unit"`
}

type CloudWatchMetricStatMetric struct {
	Dimensions []CloudWatchMetricStatMetricDimension `json:"dimensions,omitempty" protobuf:"bytes,1,rep,name=dimensions"`
	MetricName string                                `json:"metricName,omitempty" protobuf:"bytes,2,opt,name=metricName"`
	Namespace  *string                               `json:"namespace,omitempty" protobuf:"bytes,3,opt,name=namespace"`
}

type CloudWatchMetricStatMetricDimension struct {
	Name  string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	Value string `json:"value,omitempty" protobuf:"bytes,2,opt,name=value"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

type AnalysisRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              AnalysisRunSpec   `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	Status            AnalysisRunStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

//+kubebuilder:object:root=true

type AnalysisRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Items           []AnalysisRun `json:"items" protobuf:"bytes,2,rep,name=items"`
}

type SkyWalkingMetric struct {
	Address  string         `json:"address,omitempty" protobuf:"bytes,1,opt,name=address"`
	Query    string         `json:"query,omitempty" protobuf:"bytes,2,opt,name=query"`
	Interval DurationString `json:"interval,omitempty" protobuf:"bytes,3,opt,name=interval,casttype=DurationString"`
}

type AnalysisRunSpec struct {
	Metrics              []Metric               `json:"metrics" protobuf:"bytes,1,rep,name=metrics"`
	Args                 []Argument             `json:"args,omitempty" protobuf:"bytes,2,rep,name=args"`
	Terminate            bool                   `json:"terminate,omitempty" protobuf:"varint,3,opt,name=terminate"`
	DryRun               []DryRun               `json:"dryRun,omitempty" protobuf:"bytes,4,rep,name=dryRun"`
	MeasurementRetention []MeasurementRetention `json:"measurementRetention,omitempty" protobuf:"bytes,5,rep,name=measurementRetention"`
}

type Argument struct {
	Name      string     `json:"name" protobuf:"bytes,1,opt,name=name"`
	Value     *string    `json:"value,omitempty" protobuf:"bytes,2,opt,name=value"`
	ValueFrom *ValueFrom `json:"valueFrom,omitempty" protobuf:"bytes,3,opt,name=valueFrom"`
}

type ValueFrom struct {
	SecretKeyRef *SecretKeyRef `json:"secretKeyRef,omitempty" protobuf:"bytes,1,opt,name=secretKeyRef"`
	FieldRef     *FieldRef     `json:"fieldRef,omitempty" protobuf:"bytes,2,opt,name=fieldRef"`
}

type SecretKeyRef struct {
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	Key  string `json:"key" protobuf:"bytes,2,opt,name=key"`
}

type AnalysisRunStatus struct {
	Phase         AnalysisPhase  `json:"phase" protobuf:"bytes,1,opt,name=phase,casttype=AnalysisPhase"`
	Message       string         `json:"message,omitempty" protobuf:"bytes,2,opt,name=message"`
	MetricResults []MetricResult `json:"metricResults,omitempty" protobuf:"bytes,3,rep,name=metricResults"`
	StartedAt     *metav1.Time   `json:"startedAt,omitempty" protobuf:"bytes,4,opt,name=startedAt"`
	RunSummary    RunSummary     `json:"runSummary,omitempty" protobuf:"bytes,5,opt,name=runSummary"`
	DryRunSummary *RunSummary    `json:"dryRunSummary,omitempty" protobuf:"bytes,6,opt,name=dryRunSummary"`
}

func (s *AnalysisRunStatus) CompletedAt() *metav1.Time {
	if !s.Phase.Completed() {
		return nil
	}

	// FIXME: Use `CompletedAt` (which will be introduced in rollouts v1.7.0) as a default value
	var completedAt *metav1.Time

	// TODO: Remove after we bump up minimum rollouts version to v1.7.0
	for _, mr := range s.MetricResults {
		for _, m := range mr.Measurements {
			if m.FinishedAt == nil {
				continue
			}
			if completedAt == nil || m.FinishedAt.After(completedAt.Time) {
				completedAt = m.FinishedAt.DeepCopy()
			}
		}
	}
	return completedAt
}

type RunSummary struct {
	Count        int32 `json:"count,omitempty" protobuf:"varint,1,opt,name=count"`
	Successful   int32 `json:"successful,omitempty" protobuf:"varint,2,opt,name=successful"`
	Failed       int32 `json:"failed,omitempty" protobuf:"varint,3,opt,name=failed"`
	Inconclusive int32 `json:"inconclusive,omitempty" protobuf:"varint,4,opt,name=inconclusive"`
	Error        int32 `json:"error,omitempty" protobuf:"varint,5,opt,name=error"`
}

type MetricResult struct {
	Name             string            `json:"name" protobuf:"bytes,1,opt,name=name"`
	Phase            AnalysisPhase     `json:"phase" protobuf:"bytes,2,opt,name=phase,casttype=AnalysisPhase"`
	Measurements     []Measurement     `json:"measurements,omitempty" protobuf:"bytes,3,rep,name=measurements"`
	Message          string            `json:"message,omitempty" protobuf:"bytes,4,opt,name=message"`
	Count            int32             `json:"count,omitempty" protobuf:"varint,5,opt,name=count"`
	Successful       int32             `json:"successful,omitempty" protobuf:"varint,6,opt,name=successful"`
	Failed           int32             `json:"failed,omitempty" protobuf:"varint,7,opt,name=failed"`
	Inconclusive     int32             `json:"inconclusive,omitempty" protobuf:"varint,8,opt,name=inconclusive"`
	Error            int32             `json:"error,omitempty" protobuf:"varint,9,opt,name=error"`
	ConsecutiveError int32             `json:"consecutiveError,omitempty" protobuf:"varint,10,opt,name=consecutiveError"`
	DryRun           bool              `json:"dryRun,omitempty" protobuf:"varint,11,opt,name=dryRun"`
	Metadata         map[string]string `json:"metadata,omitempty" protobuf:"bytes,12,rep,name=metadata"`
}

type Measurement struct {
	Phase      AnalysisPhase     `json:"phase" protobuf:"bytes,1,opt,name=phase,casttype=AnalysisPhase"`
	Message    string            `json:"message,omitempty" protobuf:"bytes,2,opt,name=message"`
	StartedAt  *metav1.Time      `json:"startedAt,omitempty" protobuf:"bytes,3,opt,name=startedAt"`
	FinishedAt *metav1.Time      `json:"finishedAt,omitempty" protobuf:"bytes,4,opt,name=finishedAt"`
	Value      string            `json:"value,omitempty" protobuf:"bytes,5,opt,name=value"`
	Metadata   map[string]string `json:"metadata,omitempty" protobuf:"bytes,6,rep,name=metadata"`
	ResumeAt   *metav1.Time      `json:"resumeAt,omitempty" protobuf:"bytes,7,opt,name=resumeAt"`
}

type KayentaMetric struct {
	Address                  string           `json:"address" protobuf:"bytes,1,opt,name=address"`
	Application              string           `json:"application" protobuf:"bytes,2,opt,name=application"`
	CanaryConfigName         string           `json:"canaryConfigName" protobuf:"bytes,3,opt,name=canaryConfigName"`
	MetricsAccountName       string           `json:"metricsAccountName" protobuf:"bytes,4,opt,name=metricsAccountName"`
	ConfigurationAccountName string           `json:"configurationAccountName" protobuf:"bytes,5,opt,name=configurationAccountName"`
	StorageAccountName       string           `json:"storageAccountName" protobuf:"bytes,6,opt,name=storageAccountName"`
	Threshold                KayentaThreshold `json:"threshold" protobuf:"bytes,7,opt,name=threshold"`
	Scopes                   []KayentaScope   `json:"scopes" protobuf:"bytes,8,rep,name=scopes"`
}

type KayentaThreshold struct {
	Pass     int64 `json:"pass" protobuf:"varint,1,opt,name=pass"`
	Marginal int64 `json:"marginal" protobuf:"varint,2,opt,name=marginal"`
}

type KayentaScope struct {
	Name            string      `json:"name" protobuf:"bytes,1,opt,name=name"`
	ControlScope    ScopeDetail `json:"controlScope" protobuf:"bytes,2,opt,name=controlScope"`
	ExperimentScope ScopeDetail `json:"experimentScope" protobuf:"bytes,3,opt,name=experimentScope"`
}

type ScopeDetail struct {
	Scope  string `json:"scope" protobuf:"bytes,1,opt,name=scope"`
	Region string `json:"region" protobuf:"bytes,2,opt,name=region"`
	Step   int64  `json:"step" protobuf:"varint,3,opt,name=step"`
	Start  string `json:"start" protobuf:"bytes,4,opt,name=start"`
	End    string `json:"end" protobuf:"bytes,5,opt,name=end"`
}

type WebMetric struct {
	Method WebMetricMethod `json:"method,omitempty" protobuf:"bytes,1,opt,name=method,casttype=WebMetricMethod"`
	// URL is the address of the web metric
	URL            string            `json:"url" protobuf:"bytes,2,opt,name=url"`
	Headers        []WebMetricHeader `json:"headers,omitempty" protobuf:"bytes,3,rep,name=headers"`
	Body           string            `json:"body,omitempty" protobuf:"bytes,4,opt,name=body"`
	TimeoutSeconds int64             `json:"timeoutSeconds,omitempty" protobuf:"varint,5,opt,name=timeoutSeconds"`
	JSONPath       string            `json:"jsonPath,omitempty" protobuf:"bytes,6,opt,name=jsonPath"`
	Insecure       bool              `json:"insecure,omitempty" protobuf:"varint,7,opt,name=insecure"`
	JSONBody       json.RawMessage   `json:"jsonBody,omitempty" protobuf:"bytes,8,opt,name=jsonBody,casttype=encoding/json.RawMessage"`
	Authentication Authentication    `json:"authentication,omitempty" protobuf:"bytes,9,opt,name=authentication"`
}

type WebMetricMethod string

type WebMetricHeader struct {
	Key   string `json:"key" protobuf:"bytes,1,opt,name=key"`
	Value string `json:"value" protobuf:"bytes,2,opt,name=value"`
}

type DatadogMetric struct {
	Interval   DurationString    `json:"interval,omitempty" protobuf:"bytes,1,opt,name=interval,casttype=DurationString"`
	Query      string            `json:"query,omitempty" protobuf:"bytes,2,opt,name=query"`
	Queries    map[string]string `json:"queries,omitempty" protobuf:"bytes,3,rep,name=queries"`
	Formula    string            `json:"formula,omitempty" protobuf:"bytes,4,opt,name=formula"`
	ApiVersion string            `json:"apiVersion,omitempty" protobuf:"bytes,5,opt,name=apiVersion"`
}
