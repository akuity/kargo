package main

import (
	"github.com/akuityio/k8sta/internal/bookkeeper"
	"github.com/spf13/cobra"
)

func getClient(cmd *cobra.Command) (bookkeeper.Client, error) {
	serverAddress, err := cmd.Flags().GetString(flagServer)
	if err != nil {
		return nil, err
	}
	allowInsecure, err := cmd.Flags().GetBool(flagInsecure)
	if err != nil {
		return nil, err
	}
	return bookkeeper.NewClient(
		serverAddress,
		&bookkeeper.ClientOptions{
			AllowInsecureConnections: allowInsecure,
		},
	), nil
}
