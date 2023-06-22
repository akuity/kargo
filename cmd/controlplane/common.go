package main

import (
	"strings"

	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func getRestConfig(
	cfgCtx string,
	preferInClusterCfg bool,
) (*rest.Config, error) {
	var cfg *rest.Config
	var err error
	if preferInClusterCfg {
		if cfg, err = rest.InClusterConfig(); err != nil {
			return nil, errors.Wrapf(err, "error loading in-cluster rest config")
		}
		return cfg, nil
	}
	if cfg, err = config.GetConfigWithContext(cfgCtx); err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			if cfg, err = config.GetConfig(); err != nil {
				return nil, errors.Wrapf(err, "error loading default rest config")
			}
			return cfg, nil
		}
		return nil,
			errors.Wrapf(err, "error loading rest config for context %q", cfgCtx)
	}
	return cfg, nil
}
