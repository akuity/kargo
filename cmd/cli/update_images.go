package cli

import (
	"fmt"

	"github.com/akuityio/k8sta/api/v1alpha1"
	"github.com/akuityio/k8sta/internal/bookkeeper"
	"github.com/akuityio/k8sta/internal/strings"
	"github.com/spf13/cobra"
)

func newUpdateImagesCommand() (*cobra.Command, error) {
	const desc = "Update images(s) used while rendering environment-specific " +
		"configuration from a remote gitops repo to an environment-specific " +
		"branch"
	command := &cobra.Command{
		Use:     "update-images",
		Aliases: []string{"update-image"},
		Short:   desc,
		Long:    desc,
		RunE:    runUpdateImagesCommand,
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
	command.Flags().StringArrayP(
		flagImage,
		"i",
		nil,
		"specify a new image to apply to the final result (this flag may be "+
			"used more than once)",
	)
	if err := command.MarkFlagRequired(flagImage); err != nil {
		return nil, err
	}
	command.Flags().AddFlagSet(flagSetServer)
	if err := command.MarkFlagRequired(flagServer); err != nil {
		return nil, err
	}
	return command, nil
}

func runUpdateImagesCommand(cmd *cobra.Command, _ []string) error {
	renderReq, err := buildRenderRequest(cmd)
	if err != nil {
		return err
	}
	imageStrs, err := cmd.Flags().GetStringArray(flagImage)
	if err != nil {
		return err
	}
	req := bookkeeper.ImageUpdateRequest{
		RenderRequest: renderReq,
		Images:        make([]v1alpha1.Image, len(imageStrs)),
	}
	for i, imageStr := range imageStrs {
		image := v1alpha1.Image{}
		if image.Repo, image.Tag, err =
			strings.SplitLast(imageStr, ":"); err != nil {
			return err
		}
		req.Images[i] = image
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	res, err := client.UpdateImage(cmd.Context(), req)
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
