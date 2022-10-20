package action

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/akuityio/k8sta/internal/bookkeeper"
	"github.com/akuityio/k8sta/internal/common/config"
	"github.com/akuityio/k8sta/internal/common/version"
)

func Run(ctx context.Context, config config.Config) error {
	version := version.GetVersion()

	log.WithFields(log.Fields{
		"version": version.Version,
		"commit":  version.GitCommit,
	}).Info("Starting Bookkeeper Action")

	req, err := request()
	if err != nil {
		return err
	}

	res, err := bookkeeper.NewService(config).RenderConfig(ctx, req)
	if err != nil {
		return err
	}

	switch res.ActionTaken {
	case bookkeeper.ActionTakenPushedDirectly:
		fmt.Printf(
			"\nCommitted %s to branch %s\n",
			res.CommitID,
			req.TargetBranch,
		)
	case bookkeeper.ActionTakenOpenedPR:
		fmt.Printf(
			"\nOpened PR %s\n",
			res.PullRequestURL,
		)
	case bookkeeper.ActionTakenNone:
		fmt.Printf(
			"\nNewly rendered configuration does not differ from the head of "+
				"branch %s. No action was taken.\n",
			req.TargetBranch,
		)
	}

	return nil
}
