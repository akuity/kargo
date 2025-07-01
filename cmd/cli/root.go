package main

import (
	"os"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	"github.com/akuity/kargo/internal/cli/cmd/apply"
	"github.com/akuity/kargo/internal/cli/cmd/approve"
	cliconfigcmd "github.com/akuity/kargo/internal/cli/cmd/config"
	"github.com/akuity/kargo/internal/cli/cmd/create"
	"github.com/akuity/kargo/internal/cli/cmd/dashboard"
	"github.com/akuity/kargo/internal/cli/cmd/delete"
	"github.com/akuity/kargo/internal/cli/cmd/get"
	"github.com/akuity/kargo/internal/cli/cmd/grant"
	"github.com/akuity/kargo/internal/cli/cmd/login"
	"github.com/akuity/kargo/internal/cli/cmd/logout"
	"github.com/akuity/kargo/internal/cli/cmd/logs"
	"github.com/akuity/kargo/internal/cli/cmd/promote"
	"github.com/akuity/kargo/internal/cli/cmd/refresh"
	"github.com/akuity/kargo/internal/cli/cmd/revoke"
	"github.com/akuity/kargo/internal/cli/cmd/server"
	"github.com/akuity/kargo/internal/cli/cmd/update"
	"github.com/akuity/kargo/internal/cli/cmd/verify"
	"github.com/akuity/kargo/internal/cli/cmd/version"
	clicfg "github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/io"
)

func NewRootCommand(cfg clicfg.CLIConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "kargo",
		DisableAutoGenTag: true,
		SilenceUsage:      true,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}

	// Set up the IOStreams for the commands to use.
	streams := genericiooptions.IOStreams{Out: os.Stdout, ErrOut: os.Stderr, In: os.Stdin}
	io.SetIOStreams(cmd, streams)

	// Register the subcommands.
	cmd.AddCommand(apply.NewCommand(cfg, streams))
	cmd.AddCommand(approve.NewCommand(cfg))
	cmd.AddCommand(cliconfigcmd.NewCommand(cfg, streams))
	cmd.AddCommand(create.NewCommand(cfg, streams))
	cmd.AddCommand(delete.NewCommand(cfg, streams))
	cmd.AddCommand(get.NewCommand(cfg, streams))
	cmd.AddCommand(grant.NewCommand(cfg, streams))
	cmd.AddCommand(login.NewCommand(cfg))
	cmd.AddCommand(logs.NewCommand(cfg, streams))
	cmd.AddCommand(logout.NewCommand())
	cmd.AddCommand(refresh.NewCommand(cfg))
	cmd.AddCommand(revoke.NewCommand(cfg, streams))
	cmd.AddCommand(update.NewCommand(cfg, streams))
	cmd.AddCommand(dashboard.NewCommand(cfg))
	cmd.AddCommand(promote.NewCommand(cfg, streams))
	cmd.AddCommand(verify.NewCommand(cfg))
	cmd.AddCommand(version.NewCommand(cfg, streams))
	cmd.AddCommand(server.NewCommand())

	return cmd
}
