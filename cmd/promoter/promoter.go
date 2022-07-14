package promoter

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/akuityio/k8sta/internal/common/version"
)

// RunPromoter configures and runs the component that carries out a logic
// promotion from one environment to the next.
func RunPromoter(ctx context.Context) error {
	log.WithFields(log.Fields{
		"version": version.Version(),
		"commit":  version.Commit(),
	}).Info("Starting K8sTA Promoter")

	// TODO: Finish implementing this. It should load a Ticket to understand the
	// change it is supposed to be making, clone the applicable repo, make the
	// applicable change, push to the applicable repo, and update the Ticket
	// status.
	//
	// TODO: This will need to support a few different plugable strategies based
	// on whatever GitOps patterns are in use.

	return nil
}
