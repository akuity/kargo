package config

import (
	"errors"
	"fmt"
)

type ErrConfigNotFound struct {
	Path string
}

func IsConfigNotFoundErr(target error) bool {
	var err *ErrConfigNotFound
	return errors.As(target, &err)
}

func NewConfigNotFoundErr(path string) error {
	return &ErrConfigNotFound{Path: path}
}

func (e *ErrConfigNotFound) Error() string {
	return fmt.Sprintf("no configuration file was found at %q", e.Path)
}
