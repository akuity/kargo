package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	versionpkg "github.com/akuity/kargo/pkg/x/version"
)

type versionOptions struct{}

func newVersionCommand() *cobra.Command {
	cmdOpts := &versionOptions{}

	cmd := &cobra.Command{
		Use:               "version",
		DisableAutoGenTag: true,
		SilenceErrors:     true,
		SilenceUsage:      true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmdOpts.run(cmd.OutOrStdout())
		},
	}

	return cmd
}

func (o *versionOptions) run(out io.Writer) error {
	version, err := json.MarshalIndent(versionpkg.GetVersion(), "", "  ")
	if err != nil {
		return fmt.Errorf("marshal version: %w", err)
	}
	_, _ = fmt.Fprintln(out, string(version))
	return nil
}
