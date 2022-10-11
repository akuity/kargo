package main

import (
	"github.com/akuityio/k8sta/internal/bookkeeper"
	"github.com/akuityio/k8sta/internal/common/version"
	"github.com/spf13/cobra"
)

func newVersionCommand() *cobra.Command {
	const desc = "Print version information"
	command := &cobra.Command{
		Use:   "version",
		Short: desc,
		Long:  desc,
		RunE:  runVersionCommand,
	}
	command.Flags().AddFlagSet(flagSetOutput)
	command.Flags().BoolP(
		flagInsecure,
		"k",
		false,
		"tolerate certificate errors for HTTPS connections",
	)
	command.Flags().StringP(
		flagServer,
		"s",
		"",
		"specify the address of the Bookkeeper server (can also be set using "+
			"the BOOKKEEPER_SERVER environment variable)",
	)
	return command
}

func runVersionCommand(cmd *cobra.Command, args []string) error {
	serverAddress, err := cmd.Flags().GetString(flagServer)
	if err != nil {
		return err
	}

	clientVersion := version.GetVersion()

	versions := struct {
		Client *version.Version `json:"client,omitempty"`
		Server *version.Version `json:"server,omitempty"`
	}{
		Client: &clientVersion,
	}
	if serverAddress != "" {
		var client bookkeeper.Client
		if client, err = getClient(cmd); err != nil {
			return err
		}
		var serverVersion version.Version
		if serverVersion, err = client.ServerVersion(cmd.Context()); err != nil {
			return err
		}
		versions.Server = &serverVersion
	}
	outputFormat, err := cmd.Flags().GetString(flagOutput)
	if err != nil {
		return err
	}
	if outputFormat == "" {
		outputFormat = "json"
	}
	if err := output(versions, cmd.OutOrStdout(), outputFormat); err != nil {
		return err
	}
	return nil
}
