package garbage

import (
	"context"
	goerrors "errors"

	"github.com/pkg/errors"

	"github.com/akuity/kargo/internal/logging"
)

// cleanProjects is a worker function that receives Project names over a channel
// until that channel is closed. It will execute garbage collection for each
// Project name received.
func (c *collector) cleanProjects(
	ctx context.Context,
	projectCh <-chan string,
	errCh chan<- struct{},
) {
	logger := logging.LoggerFromContext(ctx)
	for {
		select {
		case project, ok := <-projectCh:
			if !ok {
				return // Channel was closed
			}
			projectLogger := logger.WithField("project", project)
			if err := c.cleanProjectFn(ctx, project); err != nil {
				projectLogger.Errorf("error cleaning Project: %s", err)
				select {
				case errCh <- struct{}{}:
				case <-ctx.Done():
					return
				}
			} else {
				projectLogger.Debug("cleaned Project")
			}
		case <-ctx.Done():
			return
		}
	}
}

// cleanProject executes garbage collection for a single Project.
func (c *collector) cleanProject(ctx context.Context, project string) error {
	errs := []error{}

	if err := c.cleanProjectPromotionsFn(ctx, project); err != nil {
		errs = append(
			errs,
			errors.Wrapf(err, "error cleaning Promotions in Project %q", project),
		)
	}

	if err := c.cleanProjectFreightFn(ctx, project); err != nil {
		errs = append(
			errs,
			errors.Wrapf(err, "error cleaning Freight in Project %q", project),
		)
	}

	return goerrors.Join(errs...)
}
