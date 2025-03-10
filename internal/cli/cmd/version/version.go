package version

import (
	"context"
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/io"
	"github.com/akuity/kargo/internal/cli/kubernetes"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/internal/cli/templates"
	versionpkg "github.com/akuity/kargo/pkg/x/version"
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
	o.PrintFlags.AddFlags(cmd)

	cmd.Flags().BoolVar(&o.ClientOnly, "client", o.ClientOnly, "If true, shows client version only (no server required)")
}

// run prints the client and server version information.
func (o *versionOptions) run(ctx context.Context) error {
	printToStdout := o.PrintFlags.OutputFlagSpecified == nil || !o.PrintFlags.OutputFlagSpecified()

	cliVersion := svcv1alpha1.ToVersionProto(versionpkg.GetVersion())
	if printToStdout {
		_, _ = fmt.Fprintln(o.IOStreams.Out, "Client Version:", cliVersion.GetVersion())
	}

	var serverVersion *svcv1alpha1.VersionInfo
	var serverErr error
	if !o.ClientOnly {
		serverVersion, serverErr = getServerVersion(ctx, o.Config, o.ClientOptions)
	}

	if printToStdout {
		if serverVersion != nil {
			_, _ = fmt.Fprintln(o.IOStreams.Out, "Server Version:", serverVersion.GetVersion())
		}
		return serverErr
	}

	printer, err := o.PrintFlags.ToPrinter()
	if err != nil {
		return fmt.Errorf("new printer: %w", err)
	}
	obj, err := componentVersionsToRuntimeObject(&svcv1alpha1.ComponentVersions{
		Server: serverVersion,
		Cli:    cliVersion,
	})
	if err != nil {
		return fmt.Errorf("map component versions to runtime object: %w", err)
	}

	if err := printer.PrintObj(obj, o.IOStreams.Out); err != nil {
		return fmt.Errorf("printing object: %w", err)
	}
	return serverErr
}

func getServerVersion(
	ctx context.Context,
	cfg config.CLIConfig,
	opts client.Options,
) (*svcv1alpha1.VersionInfo, error) {
	if cfg.APIAddress == "" || cfg.BearerToken == "" {
		return nil, nil
	}

	kargoSvcCli, err := client.GetClientFromConfig(ctx, cfg, opts)
	if err != nil {
		return nil, fmt.Errorf("get client from config: %w", err)
	}

	resp, err := kargoSvcCli.GetVersionInfo(
		ctx,
		connect.NewRequest(&svcv1alpha1.GetVersionInfoRequest{}),
	)
	if err != nil {
		return nil, fmt.Errorf("get version info from server: %w", err)
	}

	return resp.Msg.GetVersionInfo(), nil
}

func componentVersionsToRuntimeObject(v *svcv1alpha1.ComponentVersions) (runtime.Object, error) {
	data, err := protojson.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal component versions: %w", err)
	}
	var content map[string]any
	if err := json.Unmarshal(data, &content); err != nil {
		return nil, fmt.Errorf("unmarshal component versions: %w", err)
	}
	u := &unstructured.Unstructured{}
	u.SetUnstructuredContent(content)
	u.SetAPIVersion(kargoapi.GroupVersion.String())
	u.SetKind("ComponentVersions")
	return u, nil
}
