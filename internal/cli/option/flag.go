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

func EnableAutoPromotion(v *bool) FlagFn {
	return func(fs *pflag.FlagSet) {
		fs.BoolVar(v, "enable-auto-promotion", false, "Enable auto promotion")
	}
}

func OptionalEnableAutoPromotion(v Optional[bool]) FlagFn {
	return func(fs *pflag.FlagSet) {
		fs.Var(v, "enable-auto-promotion", "Enable auto promotion")
	}
}

func Stage(v *string) FlagFn {
	return func(fs *pflag.FlagSet) {
		fs.StringVar(v, "stage", "", "Stage")
	}
}

func OptionalStage(v Optional[string]) FlagFn {
	return func(fs *pflag.FlagSet) {
		fs.Var(v, "stage", "Stage")
	}
}

func State(v *string) FlagFn {
	return func(fs *pflag.FlagSet) {
		fs.StringVar(v, "state", "", "State ID")
	}
}
