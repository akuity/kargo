package api

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"time"

	"connectrpc.com/grpchealth"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/config"
	"github.com/akuity/kargo/internal/api/dex"
	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/internal/api/option"
	"github.com/akuity/kargo/internal/api/validation"
	rollouts "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/pkg/api/service/v1alpha1/svcv1alpha1connect"
)

var (
	_ svcv1alpha1connect.KargoServiceHandler = &server{}

	//go:embed all:ui
	ui embed.FS
)

type server struct {
	cfg            config.ServerConfig
	client         kubernetes.Client
	internalClient client.Client
	recorder       record.EventRecorder

	// The following behaviors are overridable for testing purposes:

	// Common validations:
	validateProjectExistsFn func(
		ctx context.Context,
		project string,
	) error

	externalValidateProjectFn func(
		ctx context.Context,
		client client.Client,
		project string,
	) error

	// Common lookups:
	getStageFn func(
		context.Context,
		client.Client,
		types.NamespacedName,
	) (*kargoapi.Stage, error)
	getFreightByNameOrAliasFn func(
		ctx context.Context,
		c client.Client,
		project string,
		name string,
		alias string,
	) (*kargoapi.Freight, error)
	isFreightAvailableFn func(
		freight *kargoapi.Freight,
		stage string,
		upstreamStages []string,
	) bool

	// Common Promotions:
	createPromotionFn func(
		context.Context,
		client.Object,
		...client.CreateOption,
	) error

	// Promote subscribers:
	findStageSubscribersFn func(ctx context.Context, stage *kargoapi.Stage) ([]kargoapi.Stage, error)

	// QueryFreight API:
	listFreightFn func(
		context.Context,
		client.ObjectList,
		...client.ListOption,
	) error
	getAvailableFreightForStageFn func(
		ctx context.Context,
		project string,
		stage string,
		subs kargoapi.Subscriptions,
	) ([]kargoapi.Freight, error)
	getFreightFromWarehouseFn func(
		ctx context.Context,
		project string,
		warehouse string,
	) ([]kargoapi.Freight, error)
	getVerifiedFreightFn func(
		ctx context.Context,
		project string,
		stageSubs []kargoapi.StageSubscription,
	) ([]kargoapi.Freight, error)

	// Freight aliasing:
	patchFreightAliasFn func(
		ctx context.Context,
		freight *kargoapi.Freight,
		alias string,
	) error

	// Freight approval:
	patchFreightStatusFn func(
		ctx context.Context,
		freight *kargoapi.Freight,
		newStatus kargoapi.FreightStatus,
	) error

	// Rollouts integration:
	getAnalysisTemplateFn func(
		context.Context,
		client.Client,
		types.NamespacedName,
	) (*rollouts.AnalysisTemplate, error)

	getAnalysisRunFn func(
		context.Context,
		client.Client,
		types.NamespacedName,
	) (*rollouts.AnalysisRun, error)

	// Special authorizations:
	authorizeFn func(
		ctx context.Context,
		verb string,
		gvr schema.GroupVersionResource,
		subresource string,
		key client.ObjectKey,
	) error
}

type Server interface {
	Serve(ctx context.Context, l net.Listener) error
}

func NewServer(
	cfg config.ServerConfig,
	kubeClient kubernetes.Client,
	internalClient client.Client,
	recorder record.EventRecorder,
) Server {
	s := &server{
		cfg:            cfg,
		client:         kubeClient,
		internalClient: internalClient,
		recorder:       recorder,
	}

	s.validateProjectExistsFn = s.validateProjectExists
	s.externalValidateProjectFn = validation.ValidateProject
	s.getStageFn = kargoapi.GetStage
	s.getFreightByNameOrAliasFn = kargoapi.GetFreightByNameOrAlias
	s.isFreightAvailableFn = kargoapi.IsFreightAvailable
	s.createPromotionFn = kubeClient.Create
	s.findStageSubscribersFn = s.findStageSubscribers
	s.listFreightFn = kubeClient.List
	s.getAvailableFreightForStageFn = s.getAvailableFreightForStage
	s.getFreightFromWarehouseFn = s.getFreightFromWarehouse
	s.getVerifiedFreightFn = s.getVerifiedFreight
	s.patchFreightAliasFn = s.patchFreightAlias
	s.patchFreightStatusFn = s.patchFreightStatus
	s.authorizeFn = kubeClient.Authorize
	s.getAnalysisTemplateFn = rollouts.GetAnalysisTemplate
	s.getAnalysisRunFn = rollouts.GetAnalysisRun

	return s
}

func (s *server) Serve(ctx context.Context, l net.Listener) error {
	log := logging.LoggerFromContext(ctx)
	mux := http.NewServeMux()

	opts, err := option.NewHandlerOption(ctx, s.cfg, s.internalClient)
	if err != nil {
		return fmt.Errorf("error initializing handler options: %w", err)
	}
	mux.Handle(grpchealth.NewHandler(NewHealthChecker(), opts))
	path, svcHandler := svcv1alpha1connect.NewKargoServiceHandler(s, opts)
	mux.Handle(path, svcHandler)
	uiFS := fs.FS(ui)
	if uiFS, err = fs.Sub(uiFS, "ui"); err != nil {
		return fmt.Errorf("error initializing UI file system: %w", err)
	}
	mux.Handle("/", http.FileServer(http.FS(uiFS)))
	if s.cfg.DexProxyConfig != nil {
		dexProxyCfg := dex.ProxyConfigFromEnv()
		dexProxy, err := dex.NewProxy(dexProxyCfg)
		if err != nil {
			return fmt.Errorf("error initializing dex proxy: %w", err)
		}
		mux.Handle("/dex/", dexProxy)
	}

	handler := h2c.NewHandler(mux, &http2.Server{})

	// Sometimes a permissive CORS policy is useful during local development.
	if s.cfg.PermissiveCORSPolicyEnabled {
		handler = cors.New(cors.Options{
			AllowCredentials: true,
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"DELETE", "GET", "POST", "PUT"},
			AllowedHeaders:   []string{"Authorization", "Content-Type"},
		}).Handler(handler)
	}

	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: time.Minute,
	}

	errCh := make(chan error)
	go func() {
		if s.cfg.TLSConfig != nil {
			errCh <- srv.ServeTLS(
				l,
				s.cfg.TLSConfig.CertPath,
				s.cfg.TLSConfig.KeyPath,
			)
		} else {
			errCh <- srv.Serve(l)
		}
	}()

	log.WithFields(logrus.Fields{
		"tls": s.cfg.TLSConfig != nil,
	}).Infof("Server is listening on %q", l.Addr().String())

	select {
	case <-ctx.Done():
		log.Info("Gracefully stopping server...")
		time.Sleep(s.cfg.GracefulShutdownTimeout)
		return srv.Shutdown(context.Background())
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
