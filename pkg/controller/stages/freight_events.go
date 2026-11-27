package stages

import (
	"context"
	"fmt"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/event"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/server/user"
)

// emitFreightCurrentlyInStage sends a FreightCurrentlyInStage event for the
// Freight. It is a no-op when no event sender is configured.
func emitFreightCurrentlyInStage(
	ctx context.Context,
	eventSender event.Sender,
	freight *kargoapi.Freight,
	stageName string,
	verifiedInStage bool,
) {
	if eventSender == nil {
		return
	}

	var actor string
	msg := fmt.Sprintf("Freight currently in Stage %q", stageName)
	if u, ok := user.IdentityFromContext(ctx); ok {
		actor = u.Actor()
		msg += fmt.Sprintf(" by %q", actor)
	}

	evt, err := event.NewFreightCurrentlyInStage(msg, actor, stageName, verifiedInStage, freight)
	if err == nil {
		err = eventSender.Send(ctx, evt)
	}
	if err != nil {
		logging.LoggerFromContext(ctx).Error(err,
			"error sending Freight currently in Stage event")
	}
}

// emitFreightVerifiedInStage sends a FreightVerifiedInStage event for the
// Freight. It is a no-op when no event sender is configured.
func emitFreightVerifiedInStage(
	ctx context.Context,
	eventSender event.Sender,
	freight *kargoapi.Freight,
	stageName string,
) {
	if eventSender == nil {
		return
	}

	var actor string
	msg := fmt.Sprintf("Freight verified in Stage %q", stageName)
	if u, ok := user.IdentityFromContext(ctx); ok {
		actor = u.Actor()
		msg += fmt.Sprintf(" by %q", actor)
	}

	evt, err := event.NewFreightVerifiedInStage(msg, actor, stageName, freight)
	if err == nil {
		err = eventSender.Send(ctx, evt)
	}
	if err != nil {
		logging.LoggerFromContext(ctx).Error(err,
			"error sending Freight verified in Stage event")
	}
}
