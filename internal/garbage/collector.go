package garbage

import (
	"context"
	"math"
	"sort"
	"sync"

	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/akuity/kargo/api/v1alpha1"
	logging "github.com/akuity/kargo/internal/logging"
)

// CollectorConfig is configuration for the garbage collector.
type CollectorConfig struct {
	// NumWorkers specifies the number of concurrent workers working on garbage
	// collection. Tuning this too low will result in slow garbage collection.
	// Tuning this too high will result in too many API calls and may result in
	// throttling.
	NumWorkers int `envconfig:"NUM_WORKERS" default:"3"`
	// MaxRetainedPromotions specifies the maximum number of Promotions in
	// terminal phases per Project that may be spared by the garbage collector.
	MaxRetainedPromotions int `envconfig:"MAX_RETAINED_PROMOTIONS" default:"20"`
}

// CollectorConfigFromEnv returns a CollectorConfig populated from environment
// variables.
func CollectorConfigFromEnv() CollectorConfig {
	cfg := CollectorConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

// Collector is an interface for the garbage collector.
type Collector interface {
	// Run runs the garbage collector until all eligible Promotion resources have
	// been deleted -- or until an unrecoverable error occurs.
	Run(context.Context) error
}

// collector is an implementation of the Collector interface.
type collector struct {
	cfg CollectorConfig

	// The following behaviors are overridable for testing purposes:
	cleanProjectsFn func(
		ctx context.Context,
		projectCh <-chan string,
		errCh chan<- struct{},
	)

	cleanProjectFn func(
		ctx context.Context,
		project string,
	) error

	listProjectsFn func(
		context.Context,
		client.ObjectList,
		...client.ListOption,
	) error

	listPromotionsFn func(
		context.Context,
		client.ObjectList,
		...client.ListOption,
	) error

	deletePromotionFn func(
		context.Context,
		client.Object,
		...client.DeleteOption,
	) error
}

// NewCollector initializes and returns an implementation of the Collector
// interface.
func NewCollector(client client.Client, cfg CollectorConfig) Collector {
	c := &collector{
		cfg: cfg,
	}
	c.cleanProjectsFn = c.cleanProjects
	c.cleanProjectFn = c.cleanProject
	c.listProjectsFn = client.List
	c.listPromotionsFn = client.List
	c.deletePromotionFn = client.Delete
	return c
}

func (c *collector) Run(ctx context.Context) error {
	projects := corev1.NamespaceList{}
	if err := c.listProjectsFn(
		ctx,
		&projects,
		&client.ListOptions{
			LabelSelector: labels.Set(
				map[string]string{
					api.LabelProjectKey: "true",
				},
			).AsSelector(),
		},
	); err != nil {
		return errors.Wrap(
			err,
			"error listing projects; no garbage collection performed",
		)
	}

	projectCh := make(chan string)
	errCh := make(chan struct{})

	// Fan out -- start workers
	numWorkers :=
		int(math.Min(float64(c.cfg.NumWorkers), float64(len(projects.Items))))
	workersWG := sync.WaitGroup{}
	workersWG.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			defer workersWG.Done()
			c.cleanProjectsFn(ctx, projectCh, errCh)
		}()
	}

	// This is a very simple mechanism for workers to communicate that they have
	// encountered an error. We don't do anything other than count them, and when
	// the process completes, exit non-zero if the count is greater than zero.
	var errCount int
	errsWG := sync.WaitGroup{}
	errsWG.Add(1)
	go func() {
		defer errsWG.Done()
		for {
			select {
			case _, ok := <-errCh:
				if !ok {
					return // Channel was closed
				}
				errCount++
			case <-ctx.Done():
				return
			}
		}
	}()

	// Distribute work across workers
	for _, project := range projects.Items {
		select {
		case projectCh <- project.Name:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Workers idly waiting for a project will return when the channel is closed
	close(projectCh)
	// Wait for remaining workers to finish
	workersWG.Wait()
	// Close error channel to signal that no more errors will be received
	close(errCh)
	// Wait for error counter to finish
	errsWG.Wait()

	if errCount > 0 {
		return errors.New(
			"one or more errors were encountered during garbage collection; " +
				"see logs for details",
		)
	}

	return nil
}

// cleanProjects is a worker function that receives Project names over a channel
// until that channel is closed. It will execute garbage collection for each
// Project name received.
func (c *collector) cleanProjects(
	ctx context.Context,
	projectCh <-chan string,
	errCh chan<- struct{},
) {
	for {
		select {
		case project, ok := <-projectCh:
			if !ok {
				return // Channel was closed
			}
			if err := c.cleanProjectFn(ctx, project); err != nil {
				select {
				case errCh <- struct{}{}:
				case <-ctx.Done():
					return
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// cleanProject executes garbage collection for a single Project.
func (c *collector) cleanProject(ctx context.Context, project string) error {
	logger := logging.LoggerFromContext(ctx).WithField("project", project)

	promos := api.PromotionList{}
	if err := c.listPromotionsFn(
		ctx,
		&promos,
		client.InNamespace(project),
	); err != nil {
		return errors.Wrapf(err, "error listing Promotions for Project %q", project)
	}

	if len(promos.Items) <= c.cfg.MaxRetainedPromotions {
		return nil // Done
	}

	// Sort Promotions by creation time
	sort.Sort(byCreation(promos.Items))

	// Delete oldest Promotions (in terminal phases only) that are in excess of
	// MaxRetainedPromotions
	var deleteErrCount int
	for i := c.cfg.MaxRetainedPromotions; i < len(promos.Items); i++ {
		promo := promos.Items[i]
		switch promo.Status.Phase {
		case api.PromotionPhaseComplete, api.PromotionPhaseFailed:
			promoLogger := logger.WithField("promotion", promo.Name)
			if err := c.deletePromotionFn(ctx, &promo); err != nil {
				promoLogger.Errorf("error deleting Promotion: %s", err)
				deleteErrCount++
			} else {
				promoLogger.Debug("deleted Promotion")
			}
		}
	}

	if deleteErrCount > 0 {
		return errors.Errorf(
			"error deleting one or more Promotions from Project %q",
			project,
		)
	}

	return nil
}

// byCreation implements sort.Interface for []api.Promotion.
type byCreation []api.Promotion

func (b byCreation) Len() int {
	return len(b)
}

func (b byCreation) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b byCreation) Less(i, j int) bool {
	return b[i].ObjectMeta.CreationTimestamp.Time.After(
		b[j].ObjectMeta.CreationTimestamp.Time,
	)
}
