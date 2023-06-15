package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	versionpkg "github.com/akuity/kargo/internal/version"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use: "version",
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := json.Marshal(versionpkg.GetVersion())
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		},
	}
}
