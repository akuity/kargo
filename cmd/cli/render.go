package cli

import (
	"fmt"

	"github.com/akuityio/k8sta/api/v1alpha1"
	"github.com/akuityio/k8sta/internal/bookkeeper"
	"github.com/spf13/cobra"
)

func newRenderCommand() (*cobra.Command, error) {
	const desc = "Render environment-specific configuration from a remote " +
		"gitops repo to an environment-specific branch"
	command := &cobra.Command{
		Use:   "render",
		Short: desc,
		Long:  desc,
		RunE:  runRenderCmd,
	}
	command.Flags().StringP(
		flagCommit,
		"c",
		"",
		"specify a precise commit to render from; if this is not provided, "+
			"Bookkeeper renders from the head of the default branch",
	)
	command.Flags().StringArrayP(
		flagImage,
		"i",
		nil,
		"specify a new image to apply to the final result (this flag may be "+
			"used more than once)",
	)
	command.Flags().BoolP(
		flagInsecure,
		"k",
		false,
		"tolerate certificate errors for HTTPS connections",
	)
	command.Flags().AddFlagSet(flagSetOutput)
	command.Flags().Bool(
		flagPR,
		false,
		"open a pull request against the target branch instead of committing "+
			"rendered configuration directly",
	)
	command.Flags().StringP(
		flagRepo,
		"r",
		"",
		"the URL of a remote gitops repo (required)",
	)
	command.Flags().StringP(
		flagRepoPassword,
		"p",
		"",
		"password or token for reading from and writing to the remote gitops "+
			"repo (required; can also be set using the BOOKKEEPER_REPO_PASSWORD "+
			"environment variable)",
	)
	command.Flags().StringP(
		flagRepoUsername,
		"u",
		"",
		"username for reading from and writing to the remote gitops repo "+
			"(required can also be set using the BOOKKEEPER_REPO_USERNAME "+
			"environment variable)",
	)
	command.Flags().StringP(
		flagServer,
		"s",
		"",
		"specify the address of the Bookkeeper server (required; can also be "+
			"set using the BOOKKEEPER_SERVER environment variable)",
	)
	command.Flags().StringP(
		flagTargetBranch,
		"t",
		"",
		"the environment-specific branch to write fully-rendered configuration "+
			"to (required)",
	)
	if err := command.MarkFlagRequired(flagRepo); err != nil {
		return nil, err
	}
	if err := command.MarkFlagRequired(flagRepoUsername); err != nil {
		return nil, err
	}
	if err := command.MarkFlagRequired(flagRepoPassword); err != nil {
		return nil, err
	}
	if err := command.MarkFlagRequired(flagServer); err != nil {
		return nil, err
	}
	if err := command.MarkFlagRequired(flagTargetBranch); err != nil {
		return nil, err
	}
	return command, nil
}

func runRenderCmd(cmd *cobra.Command, args []string) error {
	req := bookkeeper.RenderRequest{
		ConfigManagement: v1alpha1.ConfigManagementConfig{
			// TODO: Don't hard code this
			Kustomize: &v1alpha1.KustomizeConfig{},
		},
	}
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

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	res, err := client.RenderConfig(cmd.Context(), req)
	if err != nil {
		return err
	}

	outputFormat, err := cmd.Flags().GetString(flagOutput)
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	if outputFormat == "" {
		if res.CommitID != "" {
			fmt.Fprintf(
				out,
				"Committed %s to branch %s\n",
				res.CommitID,
				req.TargetBranch,
			)
		} else {
			fmt.Fprintf(
				out,
				"Opened PR %s\n",
				res.PullRequestURL,
			)
		}
	} else {
		if err := output(res, out, outputFormat); err != nil {
			return err
		}
	}

	return nil
}
