package main

import (
	"flag"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/akuityio/k8sta/internal/bookkeeper"
	"github.com/akuityio/k8sta/internal/common/config"
)

var renderCmdFlagSet = pflag.NewFlagSet(
	"render",
	pflag.ErrorHandling(flag.ExitOnError),
)

func init() {
	renderCmdFlagSet.AddFlagSet(flagSetOutput)
	renderCmdFlagSet.StringP(
		flagCommit,
		"c",
		"",
		"specify a precise commit to render from; if this is not provided, "+
			"Bookkeeper renders from the head of the default branch",
	)
	renderCmdFlagSet.StringArrayP(
		flagImage,
		"i",
		nil,
		"specify a new image to apply to the final result (this flag may be "+
			"used more than once)",
	)
	renderCmdFlagSet.Bool(
		flagPR,
		false,
		"open a pull request against the target branch instead of committing "+
			"rendered configuration directly",
	)
	renderCmdFlagSet.StringP(
		flagRepo,
		"r",
		"",
		"the URL of a remote gitops repo (required)",
	)
	renderCmdFlagSet.StringP(
		flagRepoPassword,
		"p",
		"",
		"password or token for reading from and writing to the remote gitops "+
			"repo (required; can also be set using the BOOKKEEPER_REPO_PASSWORD "+
			"environment variable)",
	)
	renderCmdFlagSet.StringP(
		flagRepoUsername,
		"u",
		"",
		"username for reading from and writing to the remote gitops repo "+
			"(required can also be set using the BOOKKEEPER_REPO_USERNAME "+
			"environment variable)",
	)
	renderCmdFlagSet.StringP(
		flagTargetBranch,
		"t",
		"",
		"the environment-specific branch to write fully-rendered configuration "+
			"to (required)",
	)
}

func newRenderCommand() (*cobra.Command, error) {
	const desc = "Render environment-specific configuration from a remote " +
		"gitops repo to an environment-specific branch"
	cmd := &cobra.Command{
		Use:   "render",
		Short: desc,
		Long:  desc,
		RunE:  runRenderCmd,
	}
	cmd.Flags().AddFlagSet(renderCmdFlagSet)
	if err := cmd.MarkFlagRequired(flagRepo); err != nil {
		return nil, err
	}
	if err := cmd.MarkFlagRequired(flagRepoUsername); err != nil {
		return nil, err
	}
	if err := cmd.MarkFlagRequired(flagRepoPassword); err != nil {
		return nil, err
	}
	if cmd.Flags().Lookup(flagServer) != nil { // Thin CLI only
		if err := cmd.MarkFlagRequired(flagServer); err != nil {
			return nil, err
		}
	}
	if err := cmd.MarkFlagRequired(flagTargetBranch); err != nil {
		return nil, err
	}
	return cmd, nil
}

func runRenderCmd(cmd *cobra.Command, args []string) error {
	req := bookkeeper.RenderRequest{}
	var err error
	req.Images, err = cmd.Flags().GetStringArray(flagImage)
	if err != nil {
		return err
	}
	req.OpenPR, err = cmd.Flags().GetBool(flagPR)
	if err != nil {
		return err
	}
	req.RepoURL, err = cmd.Flags().GetString(flagRepo)
	if err != nil {
		return err
	}
	req.RepoCreds.Username, err = cmd.Flags().GetString(flagRepoUsername)
	if err != nil {
		return err
	}
	req.RepoCreds.Password, err = cmd.Flags().GetString(flagRepoPassword)
	if err != nil {
		return err
	}
	req.Commit, err = cmd.Flags().GetString(flagCommit)
	if err != nil {
		return err
	}
	req.TargetBranch, err = cmd.Flags().GetString(flagTargetBranch)
	if err != nil {
		return err
	}

	// Choose an appropriate implementation of the bookkeeper.Service interface
	// based on whether this is the thin or thick CLI...
	var svc bookkeeper.Service
	if cmd.Flags().Lookup(flagServer) == nil { // Thick CLI
		svc = bookkeeper.NewService(
			config.Config{
				LogLevel: log.FatalLevel,
			},
		)
	} else { // Thin CLI
		if svc, err = getClient(cmd); err != nil {
			return err
		}
	}

	res, err := svc.RenderConfig(cmd.Context(), req)
	if err != nil {
		return err
	}

	outputFormat, err := cmd.Flags().GetString(flagOutput)
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	if outputFormat == "" {
		switch res.ActionTaken {
		case bookkeeper.ActionTakenPushedDirectly:
			fmt.Fprintf(
				out,
				"\nCommitted %s to branch %s\n",
				res.CommitID,
				req.TargetBranch,
			)
		case bookkeeper.ActionTakenOpenedPR:
			fmt.Fprintf(
				out,
				"\nOpened PR %s\n",
				res.PullRequestURL,
			)
		case bookkeeper.ActionTakenNone:
			fmt.Fprintf(
				out,
				"\nNewly rendered configuration does not differ from the head of "+
					"branch %s. No action was taken.\n",
				req.TargetBranch,
			)
		}
	} else {
		if err := output(res, out, outputFormat); err != nil {
			return err
		}
	}

	return nil
}
