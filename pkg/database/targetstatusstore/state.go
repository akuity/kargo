package targetstatusstore

// State is the one word for where a Target stands with a Stage that governs
// it. It is derived in one place, the ListTargetStatuses query, from the
// Promotions and PromotionRollouts on record, and every reader is served
// that word rather than left to work one out. The values below are the whole
// vocabulary, and must match the query's word for word.
//
// A State says how promotion to the Target went, not how the Target is
// doing: its health is not recorded anywhere, so no State speaks of it.
type State string // @name TargetState

const (
	// StatePromoting means a Promotion to the Target is running.
	StatePromoting State = "Promoting"
	// StatePending means a promotion to the Target is on its way but has not
	// started: a Promotion to it has yet to run, or a PromotionRollout that
	// has not finished names the Target and has not reached it yet.
	StatePending State = "Pending"
	// StateFailed means the last Promotion to the Target did not succeed: it
	// failed, errored or was aborted, and no PromotionRollout has named the
	// Target since.
	StateFailed State = "Failed"
	// StateNotAttempted means a PromotionRollout that names the Target ended,
	// without succeeding, before it created a Promotion to it, and there has
	// been no Promotion to the Target since. A later rollout that does not
	// name the Target, such as a retry of other Targets, changes nothing.
	StateNotAttempted State = "NotAttempted"
	// StateNeverPromoted means the Stage governs the Target and there has
	// never been a Promotion to it.
	StateNeverPromoted State = "NeverPromoted"
	// StateBehind means the last Promotion to the Target succeeded, and the
	// Freight it left there is not the Stage's current Freight.
	StateBehind State = "Behind"
	// StateUpToDate means the last Promotion to the Target succeeded and the
	// Target is not known to be behind the Stage.
	StateUpToDate State = "UpToDate"
)

// States returns every State, worst first. A Target governed by several
// Stages is counted once, under the first of these that any of its Stages
// reports, since it is only as far along as the Stage it is furthest behind
// for. This is not the order the query tries them in for one Stage, which
// the query documents: there a rollout that never attempted a Target says
// more than an older Promotion that failed.
func States() []State {
	return []State{
		StatePromoting,
		StatePending,
		StateFailed,
		StateNotAttempted,
		StateNeverPromoted,
		StateBehind,
		StateUpToDate,
	}
}

// Valid reports whether the State is one of the vocabulary.
func (s State) Valid() bool {
	for _, state := range States() {
		if s == state {
			return true
		}
	}
	return false
}
