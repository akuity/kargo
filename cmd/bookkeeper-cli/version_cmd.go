package main

import (
	"flag"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/akuityio/k8sta/internal/common/version"
)

var versionCmdFlagSet = pflag.NewFlagSet(
	"version",
	pflag.ErrorHandling(flag.ExitOnError),
)

func init() {
	versionCmdFlagSet.AddFlagSet(flagSetOutput)
}

func newVersionCommand() *cobra.Command {
	const desc = "Print version information"
	cmd := &cobra.Command{
		Use:   "version",
		Short: desc,
		Long:  desc,
		RunE:  runVersionCommand,
	}
	cmd.Flags().AddFlagSet(versionCmdFlagSet)
	return cmd
}

func runVersionCommand(cmd *cobra.Command, args []string) error {
	clientVersion := version.GetVersion()

	outputFormat, err := cmd.Flags().GetString(flagOutput)
	if err != nil {
		return err
	}
	if outputFormat == "" {
		outputFormat = flagOutputJSON
	}

	if cmd.Flags().Lookup(flagServer) == nil { // Thick CLI...
		return output(clientVersion, cmd.OutOrStdout(), outputFormat)
	}

	// Thin CLI...

	versions := struct {
		Client *version.Version `json:"client,omitempty"`
		Server *version.Version `json:"server,omitempty"`
	}{
		Client: &clientVersion,
	}
	serverAddress, err := cmd.Flags().GetString(flagServer)
	if err != nil {
		return err
	}
	if serverAddress != "" {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		var serverVersion version.Version
		if serverVersion, err = client.ServerVersion(cmd.Context()); err != nil {
			return err
		}
		versions.Server = &serverVersion
	}
	return output(versions, cmd.OutOrStdout(), outputFormat)
}
