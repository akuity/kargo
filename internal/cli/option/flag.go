package option

import (
	"fmt"

	"github.com/spf13/pflag"
)

type FlagFn func(*pflag.FlagSet)

func Filenames(verb string, v *[]string) FlagFn {
	return func(fs *pflag.FlagSet) {
		fs.StringSliceVarP(v, "filename", "f", nil,
			fmt.Sprintf("Filename, directory, or URL to files to use to %s the resource", verb))
	}
}

func InsecureTLS(v *bool) FlagFn {
	return func(fs *pflag.FlagSet) {
		fs.BoolVar(v, "insecure-skip-tls-verify", false, "Skip TLS certificate verification")
	}
}

func LocalServer(v *bool) FlagFn {
	return func(fs *pflag.FlagSet) {
		fs.BoolVar(v, "local-server", false, "Use local server")
	}
}
func OptionalProject(v Optional[string]) FlagFn {
	return func(fs *pflag.FlagSet) {
		fs.VarP(v, "project", "p", "Project")
	}
}

func OptionalStage(v Optional[string]) FlagFn {
	return func(fs *pflag.FlagSet) {
		fs.Var(v, "stage", "Stage")
	}
}

func Freight(v *string) FlagFn {
	return func(fs *pflag.FlagSet) {
		fs.StringVar(v, "freight", "", "Freight ID")
	}
}
