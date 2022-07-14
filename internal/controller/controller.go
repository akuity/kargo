package controller

import (
	"context"
	"sync"
	"time"

	appclientset "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"

	"github.com/akuityio/k8sta/internal/scratch"
)

type Controller interface {
	Run(context.Context) error
}

type controller struct {
	config       scratch.Config
	kubeClient   kubernetes.Interface
	argocdClient *appclientset.Clientset
	logger       *log.Logger
	// All of the controller's goroutines will send fatal errors only to here
	errCh chan error
	// All of these internal functions are overridable for testing purposes
	syncApplicationsFn func(context.Context)
	syncApplicationFn  func(obj any)
	syncLinesFn        func(context.Context)
	syncLineFn         func(obj any)
	syncTicketsFn      func(context.Context)
	syncTicketFn       func(obj any)
}

func NewController(
	config scratch.Config,
	kubeClient kubernetes.Interface,
	argocdClient *appclientset.Clientset,
) Controller {
	c := &controller{
		config:       config,
		kubeClient:   kubeClient,
		argocdClient: argocdClient,
		logger:       log.New(),
		errCh:        make(chan error),
	}
	c.logger.SetLevel(log.DebugLevel) // TODO: Make this configurable
	c.syncApplicationsFn = c.syncApplications
	c.syncApplicationFn = c.syncApplication
	c.syncLinesFn = c.syncLines
	c.syncLineFn = c.syncLine
	c.syncTicketsFn = c.syncTickets
	c.syncTicketFn = c.syncTicket
	return c
}

func (c *controller) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wg := sync.WaitGroup{}

	// TODO: We don't need this yet, because currently we're temporarily ingesting
	// configuration from a file at startup, but in the future, it will be CRDs.
	//
	// // Continuously sync K8sTA Lines (configuration)
	// wg.Add(1)
	// go func() {
	// 	defer wg.Done()
	// 	syncLineFn(ctx)
	// }()

	// Continuously sync Argo CD Applications
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.syncApplicationsFn(ctx)
	}()

	// Continuously sync K8sTA Tickets
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.syncTicketsFn(ctx)
	}()

	// Wait for an error or a completed context
	var err error
	select {
	case err = <-c.errCh:
		cancel() // Shut it all down
	case <-ctx.Done():
		err = ctx.Err()
	}

	// Adapt wg to a channel that can be used in a select
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		wg.Wait()
	}()

	select {
	case <-doneCh:
	case <-time.After(3 * time.Second):
		// Probably doesn't matter that this is hardcoded. Relatively speaking, 3
		// seconds is a lot of time for things to wrap up.
	}

	return err
}
