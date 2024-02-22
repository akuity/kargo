package option

import (
	"fmt"

	"github.com/spf13/pflag"
)

func Filenames(fs *pflag.FlagSet, filenames *[]string, verb string) {
	fs.StringSliceVarP(filenames, "filename", "f", nil,
		fmt.Sprintf("Filename, directory, or URL to files to use to %s the resource", verb))
}

func InsecureTLS(fs *pflag.FlagSet, opt *Option) {
	fs.BoolVar(&opt.InsecureTLS, "insecure-skip-tls-verify", false, "Skip TLS certificate verification")
}

func LocalServer(fs *pflag.FlagSet, opt *Option) {
	fs.BoolVar(&opt.UseLocalServer, "local-server", false, "Use local server")
}

func ClientVersion(fs *pflag.FlagSet, opt *Option) {
	fs.BoolVar(&opt.ClientVersionOnly, "client", false, "If true, shows client version only (no server required)")
}

func Project(fs *pflag.FlagSet, opt *Option, defaultProject string) {
	fs.StringVarP(&opt.Project, "project", "p", defaultProject, "Project")
}

func Stage(fs *pflag.FlagSet, stage *string) {
	fs.StringVar(stage, "stage", "", "Stage")
}

func Freight(fs *pflag.FlagSet, freight *string) {
	fs.StringVar(freight, "freight", "", "Freight ID")
}

func Wait(fs *pflag.FlagSet, wait *bool) {
	fs.BoolVar(wait, "wait", false, "Wait until refresh completes")
}
