package config

import (
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	sigyaml "sigs.k8s.io/yaml"

	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/io"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
)

type viewOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	Config      config.CLIConfig
	RawByteData bool
}

func newViewCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &viewOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("").WithDefaultOutput("yaml"),
	}

	cmd := &cobra.Command{
		Use:   "view",
		Short: "Display the CLI config",
		Args:  option.NoArgs,
		Example: templates.Example(`
# Display the CLI config as YAML output
kargo config view

# Display the CLI config as JSON output
kargo config view -o json

# Display the CLI config including sensitive data
kargo config view --raw

# Display the unmasked bearer token in the CLI config
kargo config view --raw --output=jsonpath='{.bearerToken}'
`),
		RunE: func(*cobra.Command, []string) error {
			return cmdOpts.run()
		},
	}

	// Register the option flags on the command.
	cmdOpts.addFlags(cmd)

	// Set the input/output streams for the command.
	io.SetIOStreams(cmd, cmdOpts.IOStreams)

	return cmd
}

// addFlags adds the flags for the get freight options to the provided command.
func (o *viewOptions) addFlags(cmd *cobra.Command) {
	o.AddFlags(cmd)

	cmd.Flags().BoolVar(&o.RawByteData, "raw", o.RawByteData, "Display raw byte data and sensitive data")
}

// run displays the CLI config using the provided output format.
func (o *viewOptions) run() error {
	cfg := o.Config
	if !o.RawByteData {
		cfg = config.MaskedConfig(cfg)
	}

	b, err := sigyaml.Marshal(cfg)
	if err != nil {
		return err
	}

	var rawData map[string]any
	if err = sigyaml.Unmarshal(b, &rawData); err != nil {
		return err
	}

	u := unstructured.Unstructured{Object: rawData}
	// NOTE: This is a workaround to be able to print the object using the
	//       printer, which requires the object to have a kind set.
	//       We may want to consider making the CLIConfig a proper "Kubernetes
	//       API object" in the future.
	u.SetKind("CLIConfig")

	printer, err := o.ToPrinter()
	if err != nil {
		return err
	}
	return printer.PrintObj(&u, o.Out)
}
