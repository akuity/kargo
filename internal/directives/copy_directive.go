package directives

import (
	"context"
	"errors"
	"fmt"
)

func init() {
	// Register the copy directive with the builtins registry.
	builtins.RegisterDirective(&copyDirective{}, nil)
}

// copyDirective is a directive that copies a file or directory.
type copyDirective struct{}

// copyConfig is the configuration for the copy directive.
type copyConfig struct {
	// InPath is the path to the file or directory to copy.
	InPath string `json:"inPath"`
	// OutPath is the path to the destination file or directory.
	OutPath string `json:"outPath"`
}

// Validate validates the copy configuration, returning an error if it is invalid.
func (c *copyConfig) Validate() error {
	var err []error
	if c.InPath == "" {
		err = append(err, errors.New("inPath is required"))
	}
	if c.OutPath == "" {
		err = append(err, errors.New("outPath is required"))
	}
	return errors.Join(err...)
}

func (d *copyDirective) Name() string {
	return "copy"
}

func (d *copyDirective) Run(_ context.Context, stepCtx *StepContext) (Result, error) {
	cfg, err := configToStruct[copyConfig](stepCtx.Config)
	if err != nil {
		return ResultFailure, fmt.Errorf("could not convert config into copy config: %w", err)
	}
	if err = cfg.Validate(); err != nil {
		return ResultFailure, fmt.Errorf("invalid copy config: %w", err)
	}

	// TODO: add implementation here

	return ResultSuccess, nil
}
