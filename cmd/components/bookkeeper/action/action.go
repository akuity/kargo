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

	if res.CommitID != "" {
		fmt.Printf("Committed %s to branch %s\n", res.CommitID, req.TargetBranch)
	} else {
		fmt.Printf("Opened PR %s\n", res.PullRequestURL)
	}

	return nil
}
