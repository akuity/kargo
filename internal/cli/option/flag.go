package option

import "github.com/spf13/pflag"

type FlagFn func(*pflag.FlagSet)

func ServerURL(v *string) FlagFn {
	return func(fs *pflag.FlagSet) {
		fs.StringVar(v, "server", "", "Server URL")
	}
}

func LocalServer(v *bool) FlagFn {
	return func(fs *pflag.FlagSet) {
		fs.BoolVar(v, "local-server", false, "Use local server")
	}
}

func State(v *string) FlagFn {
	return func(fs *pflag.FlagSet) {
		fs.StringVar(v, "state", "", "State ID")
	}
}
