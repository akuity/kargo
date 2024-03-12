package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	versionpkg "github.com/akuity/kargo/internal/version"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:               "version",
		DisableAutoGenTag: true,
		SilenceErrors:     true,
		SilenceUsage:      true,
		RunE: func(cmd *cobra.Command, args []string) error {
			version, err := json.MarshalIndent(versionpkg.GetVersion(), "", "  ")
			if err != nil {
				return fmt.Errorf("marshal version: %w", err)
			}
			fmt.Println(string(version))
			return nil
		},
	}
}
