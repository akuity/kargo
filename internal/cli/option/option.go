package option

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

type Option struct {
	InsecureTLS        bool
	LocalServerAddress string
	UseLocalServer     bool

	Project Optional[string]

	IOStreams  *genericclioptions.IOStreams
	PrintFlags *genericclioptions.PrintFlags
}

func NewOption() *Option {
	return &Option{
		Project: OptionalString(),
	}
}

func NewScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, errors.Wrap(err, "add core v1 scheme")
	}
	if err := kargoapi.AddToScheme(scheme); err != nil {
		return nil, errors.Wrap(err, "add kargo v1alpha1 scheme")
	}
	return scheme, nil
}

// ExactArgs is a wrapper around cobra.ExactArgs to additionally print usage string
func ExactArgs(n int) cobra.PositionalArgs {
	exactArgs := cobra.ExactArgs(n)
	return func(cmd *cobra.Command, args []string) error {
		if err := exactArgs(cmd, args); err != nil {
			fmt.Println(cmd.UsageString())
			return err
		}
		return nil
	}
}

// MinimumNArgs is a wrapper around cobra.MinimumNArgs to additionally print usage string
func MinimumNArgs(n int) cobra.PositionalArgs {
	minNArgs := cobra.MinimumNArgs(n)
	return func(cmd *cobra.Command, args []string) error {
		if err := minNArgs(cmd, args); err != nil {
			fmt.Println(cmd.UsageString())
			return err
		}
		return nil
	}
}
