package external

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/server/projectconfigs/wire"
)

// webhookSpanName is the name of the server span recorded for each inbound
// webhook request. Every receiver's path is distinct, so naming spans after the
// path would make them impossible to aggregate; the path is recorded as an
// attribute instead.
const webhookSpanName = "Receive webhook"

// receiverLookup finds the ProjectConfig that owns a webhook receiver's URL
// path, by asking the Kargo API. It returns nil, nil when no ProjectConfig does.
type receiverLookup interface {
	LookupWebhookReceiver(
		ctx context.Context,
		path string,
	) (*wire.WebhookReceiverLookup, error)
}

type server struct {
	cfg    ServerConfig
	client client.Client
	lookup receiverLookup
	// project scoped secrets are not cached so we need to query the api-server directly
	apiReader client.Reader
}

type Server interface {
	Serve(ctx context.Context, l net.Listener) error
}

func NewServer(
	cfg ServerConfig,
	cl client.Client,
	r client.Reader,
	lookup receiverLookup,
) Server {
	registerMetrics()
	return &server{
		apiReader: r,
		cfg:       cfg,
		client:    cl,
		lookup:    lookup,
	}
}

func (s *server) Serve(ctx context.Context, l net.Listener) error {
	logger := logging.LoggerFromContext(ctx)

	mux := http.NewServeMux()
	// Health check endpoint. Keep health handling separate from the route handler.
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	// All other requests are delegated to the route handler.
	mux.HandleFunc("/", s.route)

	var handler http.Handler = mux
	if s.cfg.TracingEnabled {
		// Health checks are frequent and uninteresting, so they are not traced.
		handler = otelhttp.NewHandler(
			mux,
			webhookSpanName,
			otelhttp.WithSpanNameFormatter(func(string, *http.Request) string {
				return webhookSpanName
			}),
			otelhttp.WithFilter(func(r *http.Request) bool {
				return r.URL.Path != "/healthz"
			}),
		)
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

	logger.Info(
		"Server is listening",
		"tls", s.cfg.TLSConfig != nil,
		"address", l.Addr().String(),
	)

	select {
	case <-ctx.Done():
		logger.Info("Gracefully stopping server...")
		time.Sleep(s.cfg.GracefulShutdownTimeout)
		return srv.Shutdown(context.Background())
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
