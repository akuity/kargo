package server

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/akuity/kargo/internal/api"
	apiconfig "github.com/akuity/kargo/internal/api/config"
	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/internal/api/rbac"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/internal/cli/templates"
	fakeevent "github.com/akuity/kargo/internal/kubernetes/event/fake"
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
	// TODO: This is at present incomplete, and is a placeholder for future work.
	//
	// - The server should be started with a Kubernetes client which does NOT
	//   make use of an authorization wrapper.
	// - It should allow the user to visit the UI in their browser.
	// - It should allow the user to interact with the API through `kargo`
	//   commands, but _without_ needing to authenticate, as the server is
	//   running locally using the user's kubeconfig.
	// - It should properly handle signals and clean up after itself.
	//
	// xref: https://github.com/akuity/kargo/issues/1569

	restCfg, err := config.GetConfig()
	if err != nil {
		return fmt.Errorf("get REST config: %w", err)
	}

	client, err := kubernetes.NewClient(ctx, restCfg, kubernetes.ClientOptions{})
	if err != nil {
		return fmt.Errorf("error creating Kubernetes client: %w", err)
	}

	l, err := net.Listen("tcp", o.address)
	if err != nil {
		return fmt.Errorf("start local server: %w", err)
	}
	defer l.Close() // nolint: errcheck

	srv := api.NewServer(
		apiconfig.ServerConfig{
			LocalMode: true,
		},
		client,
		client,
		rbac.NewKubernetesRolesDatabase(client),
		&fakeevent.EventRecorder{},
	)
	if err := srv.Serve(ctx, l); err != nil {
		return fmt.Errorf("serve error: %w", err)
	}
	return nil
}
