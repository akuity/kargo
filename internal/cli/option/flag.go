package option

import "github.com/spf13/pflag"

type FlagFn func(*pflag.FlagSet)

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

func Freight(v *string) FlagFn {
	return func(fs *pflag.FlagSet) {
		fs.StringVar(v, "freight", "", "Freight ID")
	}
}
