package webhook

import (
	"regexp"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	// RawControlplaneUserRegex is a regular expression to match the username in
	// admission request to distinguish if the request is coming from controlplane.
	RawControlplaneUserRegex string         `envconfig:"CONTROLPLANE_USER_REGEX"`
	ControlplaneUserRegex    *regexp.Regexp `ignored:"true"`
}

func ConfigFromEnv() Config {
	var cfg Config
	envconfig.MustProcess("", &cfg)

	if cfg.RawControlplaneUserRegex != "" {
		cfg.ControlplaneUserRegex = regexp.MustCompile(cfg.RawControlplaneUserRegex)
	}
	return cfg
}
