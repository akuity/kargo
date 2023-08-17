package config

import "fmt"

type ErrConfigNotFound struct {
	Path string
}

func NewConfigNotFoundErr(path string) error {
	return &ErrConfigNotFound{Path: path}
}

func (e *ErrConfigNotFound) Error() string {
	return fmt.Sprintf("no configuration file was found at %q", e.Path)
}
