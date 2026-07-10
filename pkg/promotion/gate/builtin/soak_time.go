package builtin

import (
	"context"
	"errors"
	"fmt"
	"time"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion/gate/types"
)

// SoakTimeGateName is the name of the Promotion creation gate that enforces
// Freight soak requirements.
const SoakTimeGateName = "soak-time"

type soakTimeGate struct {
	nowFn func() time.Time
}

// NewSoakTimeGate returns a PromotionGate that enforces the soak requirement in
// a FreightRequest.
func NewSoakTimeGate() types.PromotionGate {
	return &soakTimeGate{nowFn: time.Now}
}

func (s *soakTimeGate) Name() string {
	return SoakTimeGateName
}

func (s *soakTimeGate) Evaluate(
	_ context.Context,
	input types.PromotionInput,
) (*types.Decision, error) {
	if input.Stage == nil {
		return types.NewDenyDecision(), errors.New("stage is nil")
	}
	if input.Freight == nil {
		return types.NewDenyDecision(), errors.New("freight is nil")
	}

	// Approval is a manual override that bypasses the soak requirement, mirroring
	// how the availability gate treats approved Freight.
	if input.Freight.IsApprovedFor(input.Stage.Name) {
		return types.NewAllowDecision(), nil
	}

	request := input.FreightRequest()
	if request == nil {
		// The Stage does not request Freight from this origin, so there is no
		// soak requirement to enforce here. Static eligibility (including the
		// requested-origin check) is enforced by the eligibility gate.
		return types.NewAllowDecision(), nil
	}

	sources := request.Sources
	requiredSoakTime := sources.RequiredSoakTime
	hasRequiredSoakTime := requiredSoakTime != nil && requiredSoakTime.Duration > 0
	if sources.Direct || !hasRequiredSoakTime {
		return types.NewAllowDecision(), nil
	}
	if len(sources.Stages) == 0 {
		message := "FreightRequest has a soak requirement but no upstream Stages"
		return types.NewDenyDecision().WithMessage(message), errors.New(message)
	}

	var (
		allowed      bool
		requeueAfter *time.Duration
	)
	switch sources.AvailabilityStrategy {
	case "", kargoapi.FreightAvailabilityStrategyOneOf:
		allowed, requeueAfter = s.evaluateOneOf(
			input.Freight,
			sources.Stages,
			requiredSoakTime.Duration,
		)
	case kargoapi.FreightAvailabilityStrategyAll:
		allowed, requeueAfter = s.evaluateAll(
			input.Freight,
			sources.Stages,
			requiredSoakTime.Duration,
		)
	default:
		message := fmt.Sprintf(
			"unsupported Freight availability strategy %q",
			sources.AvailabilityStrategy,
		)
		return types.NewDenyDecision().WithMessage(message), errors.New(message)
	}

	if allowed {
		return types.NewAllowDecision(), nil
	}

	message := fmt.Sprintf(
		"Freight %q has not met the %s soak requirement for Stage %q",
		input.Freight.Name,
		requiredSoakTime.Duration,
		input.Stage.Name,
	)

	decision := types.NewDenyDecision().
		WithMessage(message).
		WithRequeueAfter(requeueAfter)

	return decision, nil
}

func (s *soakTimeGate) evaluateOneOf(
	freight *kargoapi.Freight,
	stages []string,
	required time.Duration,
) (bool, *time.Duration) {
	now := s.nowFn()
	var shortestRemaining *time.Duration
	for _, stage := range stages {
		status := getSoakStatus(freight, stage, now)
		if status.longest >= required {
			return true, nil
		}
		if !status.current {
			continue
		}
		remaining := required - status.currentDuration
		if shortestRemaining == nil || remaining < *shortestRemaining {
			shortestRemaining = &remaining
		}
	}
	return false, shortestRemaining
}

func (s *soakTimeGate) evaluateAll(
	freight *kargoapi.Freight,
	stages []string,
	required time.Duration,
) (bool, *time.Duration) {
	now := s.nowFn()
	var (
		longestRemaining *time.Duration
		hasStoppedTimer  bool
	)
	for _, stage := range stages {
		status := getSoakStatus(freight, stage, now)
		if status.longest >= required {
			continue
		}
		if !status.current {
			hasStoppedTimer = true
			continue
		}
		remaining := required - status.currentDuration
		if longestRemaining == nil || remaining > *longestRemaining {
			longestRemaining = &remaining
		}
	}
	if longestRemaining == nil && !hasStoppedTimer {
		return true, nil
	}
	if hasStoppedTimer {
		return false, nil
	}
	return false, longestRemaining
}

type soakStatus struct {
	longest         time.Duration
	current         bool
	currentDuration time.Duration
}

func getSoakStatus(
	freight *kargoapi.Freight,
	stage string,
	now time.Time,
) soakStatus {
	verifiedStage, verified := freight.Status.VerifiedIn[stage]
	if !verified {
		return soakStatus{}
	}

	status := soakStatus{}
	if verifiedStage.LongestCompletedSoak != nil {
		status.longest = verifiedStage.LongestCompletedSoak.Duration
	}
	currentStage, current := freight.Status.CurrentlyIn[stage]
	if !current || currentStage.Since == nil {
		return status
	}
	status.current = true
	status.currentDuration = max(now.Sub(currentStage.Since.Time), 0)
	status.longest = max(status.longest, status.currentDuration)
	return status
}
