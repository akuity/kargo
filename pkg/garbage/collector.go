package garbage

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/kelseyhightower/envconfig"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// CollectorConfig is configuration for the garbage collector.
type CollectorConfig struct {
	// NumWorkers specifies the number of concurrent workers working on garbage
	// collection. Tuning this too low will result in slow garbage collection.
	// Tuning this too high will result in too many API calls and may result in
	// throttling.
	NumWorkers int `envconfig:"NUM_WORKERS" default:"3"`
	// MaxRetainedPromotions specifies the ideal maximum number of Promotions
	// OLDER than the oldest in a non-terminal state (associated with each Stage)
	// that may be spared by the garbage collector. The ACTUAL number of
	// Promotions spared may exceed this ideal if some Promotions that would
	// otherwise be deleted do not meet the minimum age criterion.
	MaxRetainedPromotions int `envconfig:"MAX_RETAINED_PROMOTIONS" default:"20"`
	// MinPromotionDeletionAge specifies the minimum age Promotions must be before
	// considered eligible for garbage collection.
	MinPromotionDeletionAge time.Duration `envconfig:"MIN_PROMOTION_DELETION_AGE" default:"336h"` // 2 weeks
	// MaxRetainedFreight specifies the ideal maximum number of Freight OLDER than
	// the oldest still in use (from each Warehouse) that may be spared by the
	// garbage collector. The ACTUAL number of older Freight spared may exceed
	// this ideal if some Freight that would otherwise be deleted do not meet the
	// minimum age criterion.
	MaxRetainedFreight int `envconfig:"MAX_RETAINED_FREIGHT" default:"20"`
	// MinFreightDeletionAge specifies the minimum age Freight must be before
	// considered eligible for garbage collection.
	MinFreightDeletionAge time.Duration `envconfig:"MIN_FREIGHT_DELETION_AGE" default:"336h"` // 2 weeks
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

	cleanProjectPromotionsFn func(context.Context, string) error

	cleanStagePromotionsFn func(
		ctx context.Context,
		project string,
		stage string,
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

	cleanProjectFreightFn func(context.Context, string) error

	listWarehousesFn func(
		context.Context,
		client.ObjectList,
		...client.ListOption,
	) error

	cleanWarehouseFreightFn func(
		ctx context.Context,
		project string,
		warehouse string,
	) error

	listFreightFn func(
		context.Context,
		client.ObjectList,
		...client.ListOption,
	) error

	listStagesFn func(
		context.Context,
		client.ObjectList,
		...client.ListOption,
	) error

	deleteFreightFn func(
		context.Context,
		client.Object,
		...client.DeleteOption,
	) error
}

// NewCollector initializes and returns an implementation of the Collector
// interface.
func NewCollector(kubeClient client.Client, cfg CollectorConfig) Collector {
	c := &collector{
		cfg: cfg,
	}
	c.cleanProjectsFn = c.cleanProjects
	c.cleanProjectFn = c.cleanProject
	c.cleanProjectPromotionsFn = c.cleanProjectPromotions
	c.cleanStagePromotionsFn = c.cleanStagePromotions
	c.listProjectsFn = kubeClient.List
	c.listPromotionsFn = kubeClient.List
	c.deletePromotionFn = kubeClient.Delete
	c.cleanProjectFreightFn = c.cleanProjectFreight
	c.listWarehousesFn = kubeClient.List
	c.cleanWarehouseFreightFn = c.cleanWarehouseFreight
	c.listFreightFn = kubeClient.List
	c.listStagesFn = kubeClient.List
	c.deleteFreightFn = kubeClient.Delete
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
					kargoapi.LabelKeyProject: "true",
				},
			).AsSelector(),
		},
	); err != nil {
		return fmt.Errorf("error listing projects; no garbage collection performed: %w", err)
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
