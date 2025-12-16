package webhook

import (
	"regexp"

	"github.com/kelseyhightower/envconfig"

	"github.com/akuity/kargo/pkg/controller/warehouses"
)

type Config struct {
	// RawControlplaneUserRegex is a regular expression to match the username in
	// admission request to distinguish if the request is coming from controlplane.
	RawControlplaneUserRegex string                      `envconfig:"CONTROLPLANE_USER_REGEX"`
	ControlplaneUserRegex    *regexp.Regexp              `ignored:"true"`
	CacheByTagPolicy         warehouses.CacheByTagPolicy `envconfig:"CACHE_BY_TAG_POLICY" default:"Allow"`
}

func ConfigFromEnv() Config {
	var cfg Config
	envconfig.MustProcess("", &cfg)

	if cfg.RawControlplaneUserRegex != "" {
		cfg.ControlplaneUserRegex = regexp.MustCompile(cfg.RawControlplaneUserRegex)
	}
	return cfg
}
