package option

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type Option struct {
	InsecureTLS        bool
	LocalServerAddress string
	UseLocalServer     bool

	Project Optional[string]

	IOStreams  *genericclioptions.IOStreams
	PrintFlags *genericclioptions.PrintFlags
}

func NewOption() *Option {
	return &Option{
		Project: OptionalString(),
	}
}
