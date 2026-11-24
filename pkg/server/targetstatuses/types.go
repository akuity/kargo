package targetstatuses

import (
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database/targetstatusstore"
)

// Entry is where one Target stands with one Stage that governs it. A Target
// governed by several Stages has an entry for each, and each speaks for its
// Stage alone.
type Entry struct {
	// Target is the Target's name.
	Target string `json:"target"`
	// Labels are the Target's labels.
	Labels map[string]string `json:"labels,omitempty"`
	// Stage is the name of the Stage that governs the Target.
	Stage string `json:"stage"`
	// State is the one word for where the Target stands with the Stage.
	State targetstatusstore.State `json:"state"`
	// InFlight is whether a promotion to the Target is on its way or under
	// way: a Promotion to it is pending or running, or a PromotionRollout
	// that has not finished names it and has not reached it yet.
	InFlight bool `json:"inFlight"`
	// Behind is whether the Freight the Target is on is not the Stage's
	// current Freight. It cuts across State: a Target whose last Promotion
	// failed, or that a rollout never attempted, is usually behind as well.
	// It is false while either side is unknown, as for a Target nothing has
	// been promoted to.
	Behind bool `json:"behind"`
	// Freight is what the Target is on for the Stage, one piece per origin,
	// ordered by origin: what the newest Promotion to succeed left there. It
	// is empty until one has succeeded.
	Freight []Freight `json:"freight"`
	// LastPromotion is the Stage's newest Promotion to the Target, whatever
	// its phase. It is absent until there has been one.
	LastPromotion *Promotion `json:"lastPromotion,omitempty"`
	// Rollout is the number, among the Stage's, of the PromotionRollout the
	// State comes from when no Promotion says it: the one a Pending Target is
	// waiting for, or the one that did not attempt a NotAttempted Target. It
	// is absent otherwise.
	Rollout int64 `json:"rollout,omitempty"`
} // @name TargetStatusEntry

// Freight is one origin's Freight on a Target. It is one of a list, one per
// origin, so that what the Freight holds can be described here later without
// the shape changing.
type Freight struct {
	// Origin is where the Freight comes from.
	Origin kargoapi.FreightOrigin `json:"origin"`
	// Name is the Freight's name.
	Name string `json:"name"`
} // @name TargetFreight

// EntryList is one page of entries.
type EntryList struct {
	// Items are the entries, ordered by Target name and then Stage name.
	Items []Entry `json:"items"`
	// Continue is set when there may be more: pass it as continue to read
	// the next page.
	Continue string `json:"continue,omitempty"`
} // @name TargetStatusList

// Summary counts a Project's Targets by where they stand.
type Summary struct {
	// Targets is how many Targets the Project has, including any that no
	// Stage governs.
	Targets int64 `json:"targets"`
	// States counts the Targets some Stage governs, by State. Every State is
	// present. A Target governed by several Stages is counted once, under
	// the worst State any of them reports, in the order Promoting, Pending,
	// Failed, NotAttempted, NeverPromoted, Behind, UpToDate. The counts add
	// up to the number of governed Targets.
	States map[targetstatusstore.State]int64 `json:"states"`
	// Behind is how many of those Targets are behind any Stage that governs
	// them, whatever their State.
	Behind int64 `json:"behind"`
	// Stages holds the same counts for each Stage that governs Targets,
	// ordered by Stage name.
	Stages []StageSummary `json:"stages"`
} // @name TargetStatusSummary

// StageSummary counts the Targets one Stage governs by where they stand
// with it.
type StageSummary struct {
	// Stage is the Stage's name.
	Stage string `json:"stage"`
	// States counts the Targets the Stage governs, by State. Every State is
	// present, and the counts add up to the number of Targets it governs.
	States map[targetstatusstore.State]int64 `json:"states"`
	// Behind is how many of those Targets are behind the Stage, whatever
	// their State.
	Behind int64 `json:"behind"`
} // @name TargetStatusStageSummary
