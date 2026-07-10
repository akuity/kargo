package types

import "time"

func NewDenyDecision() *Decision {
	return &Decision{Allow: false}
}

func NewAllowDecision() *Decision {
	return &Decision{Allow: true}
}

// Decision is the uniform result of every gate contributor.
// Contributors are predicates: they never promote, only allow, deny, or skip.
type Decision struct {
	Allow bool
	// Message is a human-readable explanation of the decision.
	Message string
	// RequeueAfter makes a gate a "when", not just a "whether".
	RequeueAfter *time.Duration
}

func (d *Decision) WithAllow(allow bool) *Decision {
	d.Allow = allow
	return d
}

func (d *Decision) WithRequeueAfter(requeueAfter *time.Duration) *Decision {
	d.RequeueAfter = requeueAfter
	return d
}

func (d *Decision) WithMessage(message string) *Decision {
	d.Message = message
	return d
}
