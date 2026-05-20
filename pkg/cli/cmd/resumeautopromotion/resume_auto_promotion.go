package resumeautopromotion

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-openapi/swag/conv"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/io"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
	"github.com/akuity/kargo/pkg/client/generated/core"
	"github.com/akuity/kargo/pkg/client/generated/models"
)

type options struct {
	genericiooptions.IOStreams

	Config        config.CLIConfig
	ClientOptions client.Options

	Project string
	Stage   string
	Origin  string
}

func NewCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &options{
		Config:    cfg,
		IOStreams: streams,
	}

	cmd := &cobra.Command{
		Use:   "resume-auto-promotion [--project=project] --stage=stage --origin=Warehouse/name",
		Short: "Resume auto-promotion for a held Stage origin",
		Args:  option.NoArgs,
		Example: templates.Example(`
# Resume auto-promotion for Warehouse/main on the QA stage
kargo resume-auto-promotion --project=my-project --stage=qa --origin=Warehouse/main

# Resume auto-promotion in the default project
kargo config set-project my-project
kargo resume-auto-promotion --stage=qa --origin=Warehouse/main
`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := cmdOpts.validate(); err != nil {
				return err
			}
			return cmdOpts.run(cmd.Context())
		},
	}

	cmdOpts.addFlags(cmd)
	io.SetIOStreams(cmd, cmdOpts.IOStreams)
	return cmd
}

func (o *options) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	option.Project(
		cmd.Flags(),
		&o.Project,
		o.Config.Project,
		"The project the stage belongs to. If not set, the default project will be used.",
	)
	option.Stage(cmd.Flags(), &o.Stage, "The stage with a held auto-promotion origin.")
	cmd.Flags().StringVar(
		&o.Origin,
		option.OriginFlag,
		"",
		`The held origin to resume, formatted as "Warehouse/name".`,
	)
	_ = cmd.MarkFlagRequired(option.StageFlag)
	_ = cmd.MarkFlagRequired(option.OriginFlag)
}

func (o *options) validate() error {
	var errs []error
	o.Project = strings.TrimSpace(o.Project)
	o.Stage = strings.TrimSpace(o.Stage)
	o.Origin = strings.TrimSpace(o.Origin)
	if o.Project == "" {
		errs = append(errs, fmt.Errorf("%s is required", option.ProjectFlag))
	}
	if o.Stage == "" {
		errs = append(errs, fmt.Errorf("%s is required", option.StageFlag))
	}
	if o.Origin == "" {
		errs = append(errs, fmt.Errorf("%s is required", option.OriginFlag))
	} else if _, err := kargoapi.ParseFreightOriginKey(o.Origin); err != nil {
		errs = append(errs, fmt.Errorf("invalid %s %q: %w", option.OriginFlag, o.Origin, err))
	}
	return errors.Join(errs...)
}

func (o *options) run(ctx context.Context) error {
	apiClient, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	origin, err := kargoapi.ParseFreightOriginKey(o.Origin)
	if err != nil {
		return fmt.Errorf("parse origin: %w", err)
	}
	req := &models.ResumeStageAutoPromotionRequest{}
	req.Origin.Kind = conv.Pointer(string(origin.Kind))
	req.Origin.Name = conv.Pointer(origin.Name)

	if _, err = apiClient.Core.ResumeStageAutoPromotion(
		core.NewResumeStageAutoPromotionParams().
			WithProject(o.Project).
			WithStage(o.Stage).
			WithBody(req),
		nil,
	); err != nil {
		return fmt.Errorf("resume auto-promotion: %w", err)
	}
	_, _ = fmt.Fprintln(o.Out, "Auto-promotion resumed.")
	return nil
}
