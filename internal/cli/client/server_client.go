package client

import (
	"context"
	"net"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/akuity/kargo/internal/api"
	apiconfig "github.com/akuity/kargo/internal/api/config"
	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/pkg/api/service/v1alpha1/svcv1alpha1connect"
)

// CloseableClient is a svcv1alpha1connect.KargoServiceClient that can be
// closed.
type CloseableClient interface {
	svcv1alpha1connect.KargoServiceClient

	// Close closes the client.
	Close() error
}

// CloseIfPossible closes the client if it is a CloseableClient.
func CloseIfPossible(c svcv1alpha1connect.KargoServiceClient) error {
	if c, ok := c.(CloseableClient); ok {
		return c.Close()
	}
	return nil
}

// LocalServerClient is a client that starts a local server and connects to it.
type LocalServerClient struct {
	svcv1alpha1connect.KargoServiceClient

	listener net.Listener
}

// GetLocalServerClient starts a local server and returns a client which connects
// to it. The client should be closed when it is no longer needed.
func GetLocalServerClient(ctx context.Context, opts Options) (*LocalServerClient, error) {
	restCfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "get REST config")
	}
	client, err := kubernetes.NewClient(ctx, restCfg, kubernetes.ClientOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "error creating Kubernetes client")
	}

	serverClient := &LocalServerClient{}

	if serverClient.listener, err = net.Listen("tcp", "127.0.0.1:0"); err != nil {
		return nil, errors.Wrap(err, "start local server")
	}

	srv := api.NewServer(
		apiconfig.ServerConfig{
			LocalMode: true,
		},
		client,
		client,
	)
	go srv.Serve(ctx, serverClient.listener) // nolint: errcheck

	serverClient.KargoServiceClient = GetClient(serverClient.Addr(), "", opts.InsecureTLS)

	return serverClient, nil
}

// Addr returns the address of the local server.
func (l *LocalServerClient) Addr() string {
	if l.listener != nil {
		return "http://" + l.listener.Addr().String()
	}
	return ""
}

// Close closes the local server.
func (l *LocalServerClient) Close() error {
	if l.listener != nil {
		return l.listener.Close()
	}
	return nil
}
