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
	command.Flags().AddFlagSet(flagSetOutput)
	command.Flags().AddFlagSet(flagSetRender)
	if err := command.MarkFlagRequired(flagRepo); err != nil {
		return nil, err
	}
	if err := command.MarkFlagRequired(flagRepoUsername); err != nil {
		return nil, err
	}
	if err := command.MarkFlagRequired(flagRepoPassword); err != nil {
		return nil, err
	}
	if err := command.MarkFlagRequired(flagTargetBranch); err != nil {
		return nil, err
	}
	command.Flags().AddFlagSet(flagSetServer)
	if err := command.MarkFlagRequired(flagServer); err != nil {
		return nil, err
	}
	return command, nil
}

func runRenderCmd(cmd *cobra.Command, args []string) error {
	req, err := buildRenderRequest(cmd)
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
		fmt.Fprintf(
			out,
			"Committed %s to branch %s\n",
			res.CommitID,
			req.TargetBranch,
		)
	} else {
		if err := output(res, out, outputFormat); err != nil {
			return err
		}
	}
	return nil
}

func buildRenderRequest(cmd *cobra.Command) (bookkeeper.RenderRequest, error) {
	req := bookkeeper.RenderRequest{
		ConfigManagement: v1alpha1.ConfigManagementConfig{
			// TODO: Don't hard code this
			Kustomize: &v1alpha1.KustomizeConfig{},
		},
	}
	var err error
	req.RepoURL, err = cmd.Flags().GetString(flagRepo)
	if err != nil {
		return req, err
	}
	req.RepoCreds.Username, err = cmd.Flags().GetString(flagRepoUsername)
	if err != nil {
		return req, err
	}
	req.RepoCreds.Password, err = cmd.Flags().GetString(flagRepoPassword)
	if err != nil {
		return req, err
	}
	req.TargetBranch, err = cmd.Flags().GetString(flagTargetBranch)
	if err != nil {
		return req, err
	}
	return req, nil
}
