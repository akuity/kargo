package client

import (
	"errors"

	"github.com/akuity/kargo/internal/cli/config"
)

func IsConfigNotFoundErr(err error) bool {
	var target *config.ErrConfigNotFound
	return errors.As(err, &target)
}
