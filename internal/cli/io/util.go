package io

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

// SetIOStreams sets the input/output streams on the provided command.
func SetIOStreams(cmd *cobra.Command, streams genericiooptions.IOStreams) {
	cmd.SetIn(streams.In)
	cmd.SetOut(streams.Out)
	cmd.SetErr(streams.ErrOut)
}
