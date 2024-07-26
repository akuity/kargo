package v1alpha1

import (
	"reflect"
	"strings"
	"time"

	"github.com/gobwas/glob"
	"github.com/robfig/cron/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ApplicationSpec   `json:"spec"`
	Status            ApplicationStatus `json:"status,"`
	Operation         *Operation        `json:"operation,omitempty"`
}

type ApplicationSpec struct {
	Source      *ApplicationSource     `json:"source,omitempty"`
	SyncPolicy  *SyncPolicy            `json:"syncPolicy,omitempty"`
	Sources     ApplicationSources     `json:"sources,omitempty"`
	Project     string                 `json:"project,omitempty"`
	Destination ApplicationDestination `json:"destination,omitempty"`
}

type ApplicationSource struct {
	RepoURL        string                      `json:"repoURL"`
	TargetRevision string                      `json:"targetRevision,omitempty"`
	Helm           *ApplicationSourceHelm      `json:"helm,omitempty"`
	Kustomize      *ApplicationSourceKustomize `json:"kustomize,omitempty"`
	Chart          string                      `json:"chart,omitempty"`
}

type ApplicationDestination struct {
	Server    string `json:"server"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type AppProject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              AppProjectSpec   `json:"spec"`
	Status            AppProjectStatus `json:"status,omitempty"`
}

type AppProjectSpec struct {
	Roles       []ProjectRole `json:"roles,omitempty"`
	SyncWindows SyncWindows   `json:"syncWindows,omitempty"`
}

type AppProjectStatus struct {
	JWTTokensByRole map[string]JWTTokens
}

type JWTTokens struct {
	Items []JWTToken `json:"items,omitempty"`
}
type JWTToken struct {
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp,omitempty"`
	ID        string `json:"id,omitempty"`
}

type ProjectRole struct {
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Policies    []string   `json:"policies,omitempty"`
	JWTTokens   []JWTToken `json:"jwtTokens,omitempty"`
	Groups      []string   `json:"groups,omitempty"`
}

type SyncWindows []*SyncWindow
type SyncWindow struct {
	Kind         string   `json:"kind"`
	Schedule     string   `json:"schedule"`
	Duration     string   `json:"duration"`
	Applications []string `json:"applications"`
	Namespaces   []string `json:"namespaces"`
	Clusters     []string `json:"clusters"`
	ManualSync   bool     `json:"manualSync"`
	TimeZone     string   `json:"timeZone"`
}

// Equals compares two instances of ApplicationSource and returns true if
// they are equal.
func (source *ApplicationSource) Equals(other *ApplicationSource) bool {
	if source == nil && other == nil {
		return true
	}
	if source == nil || other == nil {
		return false
	}
	return reflect.DeepEqual(source.DeepCopy(), other.DeepCopy())
}

type ApplicationSources []ApplicationSource

// Equals compares two instances of ApplicationSources and returns true if
// they are equal.
func (s ApplicationSources) Equals(other ApplicationSources) bool {
	if len(s) != len(other) {
		return false
	}
	for i := range s {
		if !s[i].Equals(&other[i]) {
			return false
		}
	}
	return true
}

type RefreshType string

const (
	RefreshTypeHard RefreshType = "hard"
)

type ApplicationSourceHelm struct {
	Parameters []HelmParameter `json:"parameters,omitempty"`
}

type HelmParameter struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

type KustomizeImage string

type KustomizeImages []KustomizeImage

type ApplicationSourceKustomize struct {
	Images KustomizeImages `json:"images,omitempty"`
}

type ApplicationStatus struct {
	Health         HealthStatus           `json:"health,omitempty"`
	Sync           SyncStatus             `json:"sync,omitempty"`
	Conditions     []ApplicationCondition `json:"conditions,omitempty"`
	OperationState *OperationState        `json:"operationState,omitempty"`
}

type OperationInitiator struct {
	Username  string `json:"username,omitempty"`
	Automated bool   `json:"automated,omitempty"`
}

type Operation struct {
	Sync        *SyncOperation     `json:"sync,omitempty"`
	InitiatedBy OperationInitiator `json:"initiatedBy,omitempty"`
	Info        []*Info            `json:"info,omitempty"`
	Retry       RetryStrategy      `json:"retry,omitempty"`
}

type SyncOperation struct {
	SyncOptions SyncOptions `json:"syncOptions,omitempty"`
	Revisions   []string    `json:"revisions,omitempty"`
}

type Info struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type SyncOptions []string

type SyncPolicy struct {
	SyncOptions SyncOptions    `json:"syncOptions,omitempty"`
	Retry       *RetryStrategy `json:"retry,omitempty"`
}

type RetryStrategy struct {
	Limit   int64    `json:"limit,omitempty"`
	Backoff *Backoff `json:"backoff,omitempty"`
}

type Backoff struct {
	Duration    string `json:"duration,omitempty"`
	Factor      *int64 `json:"factor,omitempty"`
	MaxDuration string `json:"maxDuration,omitempty"`
}

//+kubebuilder:object:root=true

type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Application `json:"items"`
}

type SyncStatusCode string

const (
	SyncStatusCodeSynced SyncStatusCode = "Synced"
)

type SyncStatus struct {
	Status    SyncStatusCode `json:"status"`
	Revision  string         `json:"revision,omitempty"`
	Revisions []string       `json:"revisions,omitempty"`
}

type HealthStatus struct {
	Status  HealthStatusCode `json:"status,omitempty"`
	Message string           `json:"message,omitempty"`
}

type ApplicationConditionType string

var (
	ApplicationConditionInvalidSpecError ApplicationConditionType = "InvalidSpecError"
	ApplicationConditionComparisonError  ApplicationConditionType = "ComparisonError"
)

type ApplicationCondition struct {
	Type               ApplicationConditionType `json:"type"`
	Message            string                   `json:"message"`
	LastTransitionTime *metav1.Time             `json:"lastTransitionTime,omitempty"`
}

type OperationState struct {
	Operation  Operation            `json:"operation,omitempty"`
	Phase      OperationPhase       `json:"phase,omitempty"`
	Message    string               `json:"message,omitempty"`
	SyncResult *SyncOperationResult `json:"syncResult,omitempty"`
	FinishedAt *metav1.Time         `json:"finishedAt,omitempty"`
}

type SyncOperationResult struct {
	Revision string             `json:"revision,omitempty"`
	Source   ApplicationSource  `json:"source,omitempty"`
	Sources  ApplicationSources `json:"sources,omitempty"`
}

func Match(pattern, text string, separators ...rune) bool {
	compiledGlob, err := glob.Compile(pattern, separators...)
	if err != nil {
		return false
	}
	return compiledGlob.Match(text)
}

func isDenyPattern(pattern string) bool {
	return strings.HasPrefix(pattern, "!")
}

func globMatch(pattern string, val string, allowNegation bool, separators ...rune) bool {
	if allowNegation && isDenyPattern(pattern) {
		return !Match(pattern[1:], val, separators...)
	}

	if pattern == "*" {
		return true
	}
	return Match(pattern, val, separators...)
}

// HasWindows returns true if SyncWindows has one or more SyncWindow
func (w *SyncWindows) HasWindows() bool {
	return w != nil && len(*w) > 0
}

// Active returns a list of sync windows that are currently active
func (w *SyncWindows) Active() *SyncWindows {
	return w.active(time.Now())
}

func (w *SyncWindows) active(currentTime time.Time) *SyncWindows {
	// If SyncWindows.Active() is called outside of a UTC locale, it should be
	// first converted to UTC before we scan through the SyncWindows.
	currentTime = currentTime.In(time.UTC)

	if w.HasWindows() {
		var active SyncWindows
		specParser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		for _, w := range *w {
			schedule, _ := specParser.Parse(w.Schedule)
			duration, _ := time.ParseDuration(w.Duration)

			// Offset the nextWindow time to consider the timeZone of the sync window
			timeZoneOffsetDuration := w.scheduleOffsetByTimeZone()
			nextWindow := schedule.Next(currentTime.Add(timeZoneOffsetDuration - duration))
			if nextWindow.Before(currentTime.Add(timeZoneOffsetDuration)) {
				active = append(active, w)
			}
		}
		if len(active) > 0 {
			return &active
		}
	}
	return nil
}

// hasDeny will iterate over the SyncWindows and return if a deny window is found and if
// manual sync is enabled. It returns true in the first return boolean value if it finds
// any deny window. Will return true in the second return boolean value if all deny windows
// have manual sync enabled. If one deny window has manual sync disabled it returns false in
// the second return value.
func (w *SyncWindows) hasDeny() (bool, bool) {
	if !w.HasWindows() {
		return false, false
	}
	var denyFound, manualEnabled bool
	for _, a := range *w {
		if a.Kind == "deny" {
			if !denyFound {
				manualEnabled = a.ManualSync
			} else if manualEnabled {
				manualEnabled = a.ManualSync
			}
			denyFound = true
		}
	}
	return denyFound, manualEnabled
}

// hasAllow will iterate over the SyncWindows and returns true if it find any allow window.
func (w *SyncWindows) hasAllow() bool {
	if !w.HasWindows() {
		return false
	}
	for _, a := range *w {
		if a.Kind == "allow" {
			return true
		}
	}
	return false
}

// manualEnabled will iterate over the SyncWindows and return true if all windows have
// ManualSync set to true. Returns false if it finds at least one entry with ManualSync
// set to false
func (w *SyncWindows) manualEnabled() bool {
	if !w.HasWindows() {
		return false
	}
	for _, s := range *w {
		if !s.ManualSync {
			return false
		}
	}
	return true
}

// InactiveAllows will iterate over the SyncWindows and return all inactive allow windows
// for the current time. If the current time is in an inactive allow window, syncs will
// be denied.
func (w *SyncWindows) InactiveAllows() *SyncWindows {
	return w.inactiveAllows(time.Now())
}

func (w *SyncWindows) inactiveAllows(currentTime time.Time) *SyncWindows {
	// If SyncWindows.InactiveAllows() is called outside of a UTC locale, it should be
	// first converted to UTC before we scan through the SyncWindows.
	currentTime = currentTime.In(time.UTC)

	if w.HasWindows() {
		var inactive SyncWindows
		specParser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		for _, w := range *w {
			if w.Kind == "allow" {
				schedule, sErr := specParser.Parse(w.Schedule)
				duration, dErr := time.ParseDuration(w.Duration)
				// Offset the nextWindow time to consider the timeZone of the sync window
				timeZoneOffsetDuration := w.scheduleOffsetByTimeZone()
				nextWindow := schedule.Next(currentTime.Add(timeZoneOffsetDuration - duration))

				if !nextWindow.Before(currentTime.Add(timeZoneOffsetDuration)) && sErr == nil && dErr == nil {
					inactive = append(inactive, w)
				}
			}
		}
		if len(inactive) > 0 {
			return &inactive
		}
	}
	return nil
}

func (w *SyncWindow) scheduleOffsetByTimeZone() time.Duration {
	loc, err := time.LoadLocation(w.TimeZone)
	if err != nil {
		loc = time.Now().UTC().Location()
	}
	_, tzOffset := time.Now().In(loc).Zone()
	return time.Duration(tzOffset) * time.Second
}

// Matches returns a list of sync windows that are defined for a given application
func (w *SyncWindows) Matches(app *Application) *SyncWindows {
	if w.HasWindows() {
		var matchingWindows SyncWindows
		for _, w := range *w {
			if len(w.Applications) > 0 {
				for _, a := range w.Applications {
					if globMatch(a, app.Name, false) {
						matchingWindows = append(matchingWindows, w)
						break
					}
				}
			}
			if len(w.Clusters) > 0 {
				for _, c := range w.Clusters {
					dst := app.Spec.Destination
					dstNameMatched := dst.Name != "" && globMatch(c, dst.Name, false)
					dstServerMatched := dst.Server != "" && globMatch(c, dst.Server, false)
					if dstNameMatched || dstServerMatched {
						matchingWindows = append(matchingWindows, w)
						break
					}
				}
			}
			if len(w.Namespaces) > 0 {
				for _, n := range w.Namespaces {
					if globMatch(n, app.Spec.Destination.Namespace, false) {
						matchingWindows = append(matchingWindows, w)
						break
					}
				}
			}
		}
		if len(matchingWindows) > 0 {
			return &matchingWindows
		}
	}
	return nil
}

// CanSync returns true if a sync window currently allows a sync
func (w *SyncWindows) CanSync(isManual bool) bool {
	if !w.HasWindows() {
		return true
	}

	active := w.Active()
	hasActiveDeny, manualEnabled := active.hasDeny()

	if hasActiveDeny {
		if isManual && manualEnabled {
			return true
		}
		return false
	}

	if active.hasAllow() {
		return true
	}

	inactiveAllows := w.InactiveAllows()
	if inactiveAllows.HasWindows() {
		if isManual && inactiveAllows.manualEnabled() {
			return true
		}
		return false
	}

	return true
}
