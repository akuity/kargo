package server

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/internal/cli/templates"
	fakeevent "github.com/akuity/kargo/internal/kubernetes/event/fake"
	"github.com/akuity/kargo/internal/server"
	apiconfig "github.com/akuity/kargo/internal/server/config"
	"github.com/akuity/kargo/internal/server/kubernetes"
	"github.com/akuity/kargo/internal/server/rbac"
)

type serverOptions struct {
	address string
}

func NewCommand() *cobra.Command {
	cmdOpts := &serverOptions{}

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start a local Kargo API server",
		Args:  option.NoArgs,
		Example: templates.Example(`
# Start a local Kargo API server on a random port
kargo server

# Start a local Kargo API server on a specific address
kargo server --address=127.0.0.1:3000
`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := cmdOpts.validate(); err != nil {
				return err
			}
			return cmdOpts.run(cmd.Context())
		},
	}

	// Register the option flags on the command.
	cmdOpts.addFlags(cmd)

	return cmd
}

// addFlags adds the flags for the server options to the provided command.
func (o *serverOptions) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&o.address, "address", "127.0.0.1:0",
		"Address to bind the server to. Defaults to binding to a random port on localhost.")
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *serverOptions) validate() error {
	if o.address == "" {
		return errors.New("address cannot be empty")
	}
	return nil
}

// run starts a local server on the provided address.
func (o *serverOptions) run(ctx context.Context) error {
	restCfg, err := config.GetConfig()
	if err != nil {
		return fmt.Errorf("get REST config: %w", err)
	}

	client, err := kubernetes.NewClient(
		ctx,
		restCfg,
		kubernetes.ClientOptions{
			SkipAuthorization: true,
		},
	)
	if err != nil {
		return fmt.Errorf("error creating Kubernetes client: %w", err)
	}

	l, err := net.Listen("tcp", o.address)
	if err != nil {
		return fmt.Errorf("start local server: %w", err)
	}
	defer l.Close() // nolint: errcheck

	srv := server.NewServer(
		apiconfig.ServerConfig{
			LocalMode: true,
		},
		client,
		rbac.NewKubernetesRolesDatabase(client),
		&fakeevent.EventRecorder{},
	)
	if err := srv.Serve(ctx, l); err != nil {
		return fmt.Errorf("serve error: %w", err)
	}
	return nil
}
