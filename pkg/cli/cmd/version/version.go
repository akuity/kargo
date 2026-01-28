package version

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/io"
	"github.com/akuity/kargo/pkg/cli/kubernetes"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
	"github.com/akuity/kargo/pkg/client/generated/models"
	"github.com/akuity/kargo/pkg/client/generated/system"
	"github.com/akuity/kargo/pkg/x/version"
)

type versionOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	Config        config.CLIConfig
	ClientOptions client.Options

	ClientOnly bool
}

func NewCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &versionOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use:   "version [--client]",
		Short: "Show the client and server version information",
		Args:  option.NoArgs,
		Example: templates.Example(`
# Print the client and server version information
kargo version

# Print the client version information only
kargo version --client
`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmdOpts.run(cmd.Context())
		},
	}

	// Register the option flags on the command.
	cmdOpts.addFlags(cmd)

	// Set the input/output streams for the command.
	io.SetIOStreams(cmd, cmdOpts.IOStreams)

	return cmd
}

// addFlags adds the flags for the version options to the provided command.
func (o *versionOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.AddFlags(cmd)

	cmd.Flags().BoolVar(&o.ClientOnly, "client", o.ClientOnly, "If true, shows client version only (no server required)")
}

// run prints the client and server version information.
func (o *versionOptions) run(ctx context.Context) error {
	printToStdout := o.OutputFlagSpecified == nil || !o.OutputFlagSpecified()

	if printToStdout {
		_, _ = fmt.Fprintln(o.Out, "Client Version:", version.GetVersion().Version)
	}

	var serverVersion *models.VersionInfo
	var serverErr error
	if !o.ClientOnly {
		serverVersion, serverErr = getServerVersion(ctx, o.Config, o.ClientOptions)
	}

	if printToStdout {
		if serverVersion != nil {
			_, _ = fmt.Fprintln(o.Out, "Server Version:", serverVersion.Version)
		}
		return serverErr
	}

	printer, err := o.ToPrinter()
	if err != nil {
		return fmt.Errorf("new printer: %w", err)
	}
	// Convert to something compatible with the printer
	obj, err := componentVersionsToRuntimeObject(
		version.GetVersion(),
		serverVersion,
	)
	if err != nil {
		return fmt.Errorf("map component versions to runtime object: %w", err)
	}

	if err := printer.PrintObj(obj, o.Out); err != nil {
		return fmt.Errorf("printing object: %w", err)
	}
	return serverErr
}

func getServerVersion(
	ctx context.Context,
	cfg config.CLIConfig,
	opts client.Options,
) (*models.VersionInfo, error) {
	// Don't bother if definitely not authenticated to a specified server
	if cfg.APIAddress == "" || cfg.BearerToken == "" {
		return nil, nil
	}

	apiClient, err := client.GetClientFromConfig(ctx, cfg, opts)
	if err != nil {
		return nil, fmt.Errorf("get client from config: %w", err)
	}

	res, err := apiClient.System.GetVersionInfo(
		system.NewGetVersionInfoParams(),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("get version info from server: %w", err)
	}

	return res.Payload, nil
}

func componentVersionsToRuntimeObject(
	cliVersion version.Version,
	serverVersion *models.VersionInfo,
) (runtime.Object, error) {
	content := map[string]any{
		"apiVersion": "kargo.akuity.io/v1alpha1",
		"kind":       "ComponentVersions",
	}

	// Add client version
	clientData, err := json.Marshal(cliVersion)
	if err != nil {
		return nil, fmt.Errorf("marshal client version: %w", err)
	}
	var clientContent map[string]any
	if err := json.Unmarshal(clientData, &clientContent); err != nil {
		return nil, fmt.Errorf("unmarshal client version: %w", err)
	}
	content["client"] = clientContent

	// Add server version
	if serverVersion != nil {
		serverData, err := json.Marshal(serverVersion)
		if err != nil {
			return nil, fmt.Errorf("marshal server version: %w", err)
		}
		var serverContent map[string]any
		if err := json.Unmarshal(serverData, &serverContent); err != nil {
			return nil, fmt.Errorf("unmarshal server version: %w", err)
		}
		content["server"] = serverContent
	}

	u := &unstructured.Unstructured{}
	u.SetUnstructuredContent(content)
	return u, nil
}
