# V1JobSpec

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ActiveDeadlineSeconds** | Pointer to **int32** | Specifies the duration in seconds relative to the startTime that the job may be continuously active before the system tries to terminate it; value must be positive integer. If a Job is suspended (at creation or through an update), this timer will effectively be stopped and reset when the Job is resumed again. +optional | [optional] 
**BackoffLimit** | Pointer to **int32** | Specifies the number of retries before marking this job failed. Defaults to 6, unless backoffLimitPerIndex (only Indexed Job) is specified. When backoffLimitPerIndex is specified, backoffLimit defaults to 2147483647. +optional | [optional] 
**BackoffLimitPerIndex** | Pointer to **int32** | Specifies the limit for the number of retries within an index before marking this index as failed. When enabled the number of failures per index is kept in the pod&#39;s batch.kubernetes.io/job-index-failure-count annotation. It can only be set when Job&#39;s completionMode&#x3D;Indexed, and the Pod&#39;s restart policy is Never. The field is immutable. +optional | [optional] 
**CompletionMode** | Pointer to [**V1CompletionMode**](V1CompletionMode.md) | completionMode specifies how Pod completions are tracked. It can be &#x60;NonIndexed&#x60; (default) or &#x60;Indexed&#x60;.  &#x60;NonIndexed&#x60; means that the Job is considered complete when there have been .spec.completions successfully completed Pods. Each Pod completion is homologous to each other.  &#x60;Indexed&#x60; means that the Pods of a Job get an associated completion index from 0 to (.spec.completions - 1), available in the annotation batch.kubernetes.io/job-completion-index. The Job is considered complete when there is one successfully completed Pod for each index. When value is &#x60;Indexed&#x60;, .spec.completions must be specified and &#x60;.spec.parallelism&#x60; must be less than or equal to 10^5. In addition, The Pod name takes the form &#x60;$(job-name)-$(index)-$(random-string)&#x60;, the Pod hostname takes the form &#x60;$(job-name)-$(index)&#x60;.  More completion modes can be added in the future. If the Job controller observes a mode that it doesn&#39;t recognize, which is possible during upgrades due to version skew, the controller skips updates for the Job. +optional | [optional] 
**Completions** | Pointer to **int32** | Specifies the desired number of successfully finished pods the job should be run with.  Setting to null means that the success of any pod signals the success of all pods, and allows parallelism to have any positive value.  Setting to 1 means that parallelism is limited to 1 and the success of that pod signals the success of the job. More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/ +optional | [optional] 
**ManagedBy** | Pointer to **string** | ManagedBy field indicates the controller that manages a Job. The k8s Job controller reconciles jobs which don&#39;t have this field at all or the field value is the reserved string &#x60;kubernetes.io/job-controller&#x60;, but skips reconciling Jobs with a custom value for this field. The value must be a valid domain-prefixed path (e.g. acme.io/foo) - all characters before the first \&quot;/\&quot; must be a valid subdomain as defined by RFC 1123. All characters trailing the first \&quot;/\&quot; must be valid HTTP Path characters as defined by RFC 3986. The value cannot exceed 63 characters. This field is immutable.  This field is beta-level. The job controller accepts setting the field when the feature gate JobManagedBy is enabled (enabled by default). +optional | [optional] 
**ManualSelector** | Pointer to **bool** | manualSelector controls generation of pod labels and pod selectors. Leave &#x60;manualSelector&#x60; unset unless you are certain what you are doing. When false or unset, the system pick labels unique to this job and appends those labels to the pod template.  When true, the user is responsible for picking unique labels and specifying the selector.  Failure to pick a unique label may cause this and other jobs to not function correctly.  However, You may see &#x60;manualSelector&#x3D;true&#x60; in jobs that were created with the old &#x60;extensions/v1beta1&#x60; API. More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/#specifying-your-own-pod-selector +optional | [optional] 
**MaxFailedIndexes** | Pointer to **int32** | Specifies the maximal number of failed indexes before marking the Job as failed, when backoffLimitPerIndex is set. Once the number of failed indexes exceeds this number the entire Job is marked as Failed and its execution is terminated. When left as null the job continues execution of all of its indexes and is marked with the &#x60;Complete&#x60; Job condition. It can only be specified when backoffLimitPerIndex is set. It can be null or up to completions. It is required and must be less than or equal to 10^4 when is completions greater than 10^5. +optional | [optional] 
**Parallelism** | Pointer to **int32** | Specifies the maximum desired number of pods the job should run at any given time. The actual number of pods running in steady state will be less than this number when ((.spec.completions - .status.successful) &lt; .spec.parallelism), i.e. when the work left to do is less than max parallelism. More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/ +optional | [optional] 
**PodFailurePolicy** | Pointer to [**V1PodFailurePolicy**](V1PodFailurePolicy.md) | Specifies the policy of handling failed pods. In particular, it allows to specify the set of actions and conditions which need to be satisfied to take the associated action. If empty, the default behaviour applies - the counter of failed pods, represented by the jobs&#39;s .status.failed field, is incremented and it is checked against the backoffLimit. This field cannot be used in combination with restartPolicy&#x3D;OnFailure.  +optional | [optional] 
**PodReplacementPolicy** | Pointer to [**V1PodReplacementPolicy**](V1PodReplacementPolicy.md) | podReplacementPolicy specifies when to create replacement Pods. Possible values are: - TerminatingOrFailed means that we recreate pods   when they are terminating (has a metadata.deletionTimestamp) or failed. - Failed means to wait until a previously created Pod is fully terminated (has phase   Failed or Succeeded) before creating a replacement Pod.  When using podFailurePolicy, Failed is the the only allowed value. TerminatingOrFailed and Failed are allowed values when podFailurePolicy is not in use. +optional | [optional] 
**Selector** | Pointer to [**V1LabelSelector**](V1LabelSelector.md) | A label query over pods that should match the pod count. Normally, the system sets this field for you. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors +optional | [optional] 
**SuccessPolicy** | Pointer to [**V1SuccessPolicy**](V1SuccessPolicy.md) | successPolicy specifies the policy when the Job can be declared as succeeded. If empty, the default behavior applies - the Job is declared as succeeded only when the number of succeeded pods equals to the completions. When the field is specified, it must be immutable and works only for the Indexed Jobs. Once the Job meets the SuccessPolicy, the lingering pods are terminated.  +optional | [optional] 
**Suspend** | Pointer to **bool** | suspend specifies whether the Job controller should create Pods or not. If a Job is created with suspend set to true, no Pods are created by the Job controller. If a Job is suspended after creation (i.e. the flag goes from false to true), the Job controller will delete all active Pods associated with this Job. Users must design their workload to gracefully handle this. Suspending a Job will reset the StartTime field of the Job, effectively resetting the ActiveDeadlineSeconds timer too. Defaults to false.  +optional | [optional] 
**Template** | Pointer to [**V1PodTemplateSpec**](V1PodTemplateSpec.md) | Describes the pod that will be created when executing a job. The only allowed template.spec.restartPolicy values are \&quot;Never\&quot; or \&quot;OnFailure\&quot;. More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/ | [optional] 
**TtlSecondsAfterFinished** | Pointer to **int32** | ttlSecondsAfterFinished limits the lifetime of a Job that has finished execution (either Complete or Failed). If this field is set, ttlSecondsAfterFinished after the Job finishes, it is eligible to be automatically deleted. When the Job is being deleted, its lifecycle guarantees (e.g. finalizers) will be honored. If this field is unset, the Job won&#39;t be automatically deleted. If this field is set to zero, the Job becomes eligible to be deleted immediately after it finishes. +optional | [optional] 

## Methods

### NewV1JobSpec

`func NewV1JobSpec() *V1JobSpec`

NewV1JobSpec instantiates a new V1JobSpec object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1JobSpecWithDefaults

`func NewV1JobSpecWithDefaults() *V1JobSpec`

NewV1JobSpecWithDefaults instantiates a new V1JobSpec object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetActiveDeadlineSeconds

`func (o *V1JobSpec) GetActiveDeadlineSeconds() int32`

GetActiveDeadlineSeconds returns the ActiveDeadlineSeconds field if non-nil, zero value otherwise.

### GetActiveDeadlineSecondsOk

`func (o *V1JobSpec) GetActiveDeadlineSecondsOk() (*int32, bool)`

GetActiveDeadlineSecondsOk returns a tuple with the ActiveDeadlineSeconds field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetActiveDeadlineSeconds

`func (o *V1JobSpec) SetActiveDeadlineSeconds(v int32)`

SetActiveDeadlineSeconds sets ActiveDeadlineSeconds field to given value.

### HasActiveDeadlineSeconds

`func (o *V1JobSpec) HasActiveDeadlineSeconds() bool`

HasActiveDeadlineSeconds returns a boolean if a field has been set.

### GetBackoffLimit

`func (o *V1JobSpec) GetBackoffLimit() int32`

GetBackoffLimit returns the BackoffLimit field if non-nil, zero value otherwise.

### GetBackoffLimitOk

`func (o *V1JobSpec) GetBackoffLimitOk() (*int32, bool)`

GetBackoffLimitOk returns a tuple with the BackoffLimit field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBackoffLimit

`func (o *V1JobSpec) SetBackoffLimit(v int32)`

SetBackoffLimit sets BackoffLimit field to given value.

### HasBackoffLimit

`func (o *V1JobSpec) HasBackoffLimit() bool`

HasBackoffLimit returns a boolean if a field has been set.

### GetBackoffLimitPerIndex

`func (o *V1JobSpec) GetBackoffLimitPerIndex() int32`

GetBackoffLimitPerIndex returns the BackoffLimitPerIndex field if non-nil, zero value otherwise.

### GetBackoffLimitPerIndexOk

`func (o *V1JobSpec) GetBackoffLimitPerIndexOk() (*int32, bool)`

GetBackoffLimitPerIndexOk returns a tuple with the BackoffLimitPerIndex field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBackoffLimitPerIndex

`func (o *V1JobSpec) SetBackoffLimitPerIndex(v int32)`

SetBackoffLimitPerIndex sets BackoffLimitPerIndex field to given value.

### HasBackoffLimitPerIndex

`func (o *V1JobSpec) HasBackoffLimitPerIndex() bool`

HasBackoffLimitPerIndex returns a boolean if a field has been set.

### GetCompletionMode

`func (o *V1JobSpec) GetCompletionMode() V1CompletionMode`

GetCompletionMode returns the CompletionMode field if non-nil, zero value otherwise.

### GetCompletionModeOk

`func (o *V1JobSpec) GetCompletionModeOk() (*V1CompletionMode, bool)`

GetCompletionModeOk returns a tuple with the CompletionMode field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCompletionMode

`func (o *V1JobSpec) SetCompletionMode(v V1CompletionMode)`

SetCompletionMode sets CompletionMode field to given value.

### HasCompletionMode

`func (o *V1JobSpec) HasCompletionMode() bool`

HasCompletionMode returns a boolean if a field has been set.

### GetCompletions

`func (o *V1JobSpec) GetCompletions() int32`

GetCompletions returns the Completions field if non-nil, zero value otherwise.

### GetCompletionsOk

`func (o *V1JobSpec) GetCompletionsOk() (*int32, bool)`

GetCompletionsOk returns a tuple with the Completions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCompletions

`func (o *V1JobSpec) SetCompletions(v int32)`

SetCompletions sets Completions field to given value.

### HasCompletions

`func (o *V1JobSpec) HasCompletions() bool`

HasCompletions returns a boolean if a field has been set.

### GetManagedBy

`func (o *V1JobSpec) GetManagedBy() string`

GetManagedBy returns the ManagedBy field if non-nil, zero value otherwise.

### GetManagedByOk

`func (o *V1JobSpec) GetManagedByOk() (*string, bool)`

GetManagedByOk returns a tuple with the ManagedBy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetManagedBy

`func (o *V1JobSpec) SetManagedBy(v string)`

SetManagedBy sets ManagedBy field to given value.

### HasManagedBy

`func (o *V1JobSpec) HasManagedBy() bool`

HasManagedBy returns a boolean if a field has been set.

### GetManualSelector

`func (o *V1JobSpec) GetManualSelector() bool`

GetManualSelector returns the ManualSelector field if non-nil, zero value otherwise.

### GetManualSelectorOk

`func (o *V1JobSpec) GetManualSelectorOk() (*bool, bool)`

GetManualSelectorOk returns a tuple with the ManualSelector field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetManualSelector

`func (o *V1JobSpec) SetManualSelector(v bool)`

SetManualSelector sets ManualSelector field to given value.

### HasManualSelector

`func (o *V1JobSpec) HasManualSelector() bool`

HasManualSelector returns a boolean if a field has been set.

### GetMaxFailedIndexes

`func (o *V1JobSpec) GetMaxFailedIndexes() int32`

GetMaxFailedIndexes returns the MaxFailedIndexes field if non-nil, zero value otherwise.

### GetMaxFailedIndexesOk

`func (o *V1JobSpec) GetMaxFailedIndexesOk() (*int32, bool)`

GetMaxFailedIndexesOk returns a tuple with the MaxFailedIndexes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMaxFailedIndexes

`func (o *V1JobSpec) SetMaxFailedIndexes(v int32)`

SetMaxFailedIndexes sets MaxFailedIndexes field to given value.

### HasMaxFailedIndexes

`func (o *V1JobSpec) HasMaxFailedIndexes() bool`

HasMaxFailedIndexes returns a boolean if a field has been set.

### GetParallelism

`func (o *V1JobSpec) GetParallelism() int32`

GetParallelism returns the Parallelism field if non-nil, zero value otherwise.

### GetParallelismOk

`func (o *V1JobSpec) GetParallelismOk() (*int32, bool)`

GetParallelismOk returns a tuple with the Parallelism field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetParallelism

`func (o *V1JobSpec) SetParallelism(v int32)`

SetParallelism sets Parallelism field to given value.

### HasParallelism

`func (o *V1JobSpec) HasParallelism() bool`

HasParallelism returns a boolean if a field has been set.

### GetPodFailurePolicy

`func (o *V1JobSpec) GetPodFailurePolicy() V1PodFailurePolicy`

GetPodFailurePolicy returns the PodFailurePolicy field if non-nil, zero value otherwise.

### GetPodFailurePolicyOk

`func (o *V1JobSpec) GetPodFailurePolicyOk() (*V1PodFailurePolicy, bool)`

GetPodFailurePolicyOk returns a tuple with the PodFailurePolicy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPodFailurePolicy

`func (o *V1JobSpec) SetPodFailurePolicy(v V1PodFailurePolicy)`

SetPodFailurePolicy sets PodFailurePolicy field to given value.

### HasPodFailurePolicy

`func (o *V1JobSpec) HasPodFailurePolicy() bool`

HasPodFailurePolicy returns a boolean if a field has been set.

### GetPodReplacementPolicy

`func (o *V1JobSpec) GetPodReplacementPolicy() V1PodReplacementPolicy`

GetPodReplacementPolicy returns the PodReplacementPolicy field if non-nil, zero value otherwise.

### GetPodReplacementPolicyOk

`func (o *V1JobSpec) GetPodReplacementPolicyOk() (*V1PodReplacementPolicy, bool)`

GetPodReplacementPolicyOk returns a tuple with the PodReplacementPolicy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPodReplacementPolicy

`func (o *V1JobSpec) SetPodReplacementPolicy(v V1PodReplacementPolicy)`

SetPodReplacementPolicy sets PodReplacementPolicy field to given value.

### HasPodReplacementPolicy

`func (o *V1JobSpec) HasPodReplacementPolicy() bool`

HasPodReplacementPolicy returns a boolean if a field has been set.

### GetSelector

`func (o *V1JobSpec) GetSelector() V1LabelSelector`

GetSelector returns the Selector field if non-nil, zero value otherwise.

### GetSelectorOk

`func (o *V1JobSpec) GetSelectorOk() (*V1LabelSelector, bool)`

GetSelectorOk returns a tuple with the Selector field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSelector

`func (o *V1JobSpec) SetSelector(v V1LabelSelector)`

SetSelector sets Selector field to given value.

### HasSelector

`func (o *V1JobSpec) HasSelector() bool`

HasSelector returns a boolean if a field has been set.

### GetSuccessPolicy

`func (o *V1JobSpec) GetSuccessPolicy() V1SuccessPolicy`

GetSuccessPolicy returns the SuccessPolicy field if non-nil, zero value otherwise.

### GetSuccessPolicyOk

`func (o *V1JobSpec) GetSuccessPolicyOk() (*V1SuccessPolicy, bool)`

GetSuccessPolicyOk returns a tuple with the SuccessPolicy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSuccessPolicy

`func (o *V1JobSpec) SetSuccessPolicy(v V1SuccessPolicy)`

SetSuccessPolicy sets SuccessPolicy field to given value.

### HasSuccessPolicy

`func (o *V1JobSpec) HasSuccessPolicy() bool`

HasSuccessPolicy returns a boolean if a field has been set.

### GetSuspend

`func (o *V1JobSpec) GetSuspend() bool`

GetSuspend returns the Suspend field if non-nil, zero value otherwise.

### GetSuspendOk

`func (o *V1JobSpec) GetSuspendOk() (*bool, bool)`

GetSuspendOk returns a tuple with the Suspend field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSuspend

`func (o *V1JobSpec) SetSuspend(v bool)`

SetSuspend sets Suspend field to given value.

### HasSuspend

`func (o *V1JobSpec) HasSuspend() bool`

HasSuspend returns a boolean if a field has been set.

### GetTemplate

`func (o *V1JobSpec) GetTemplate() V1PodTemplateSpec`

GetTemplate returns the Template field if non-nil, zero value otherwise.

### GetTemplateOk

`func (o *V1JobSpec) GetTemplateOk() (*V1PodTemplateSpec, bool)`

GetTemplateOk returns a tuple with the Template field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTemplate

`func (o *V1JobSpec) SetTemplate(v V1PodTemplateSpec)`

SetTemplate sets Template field to given value.

### HasTemplate

`func (o *V1JobSpec) HasTemplate() bool`

HasTemplate returns a boolean if a field has been set.

### GetTtlSecondsAfterFinished

`func (o *V1JobSpec) GetTtlSecondsAfterFinished() int32`

GetTtlSecondsAfterFinished returns the TtlSecondsAfterFinished field if non-nil, zero value otherwise.

### GetTtlSecondsAfterFinishedOk

`func (o *V1JobSpec) GetTtlSecondsAfterFinishedOk() (*int32, bool)`

GetTtlSecondsAfterFinishedOk returns a tuple with the TtlSecondsAfterFinished field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTtlSecondsAfterFinished

`func (o *V1JobSpec) SetTtlSecondsAfterFinished(v int32)`

SetTtlSecondsAfterFinished sets TtlSecondsAfterFinished field to given value.

### HasTtlSecondsAfterFinished

`func (o *V1JobSpec) HasTtlSecondsAfterFinished() bool`

HasTtlSecondsAfterFinished returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


