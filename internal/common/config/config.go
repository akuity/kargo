package config

import log "github.com/sirupsen/logrus"

type Config struct {
	LogLevel        log.Level
	ArgoCDNamespace string
	K8sTANamespace  string
}
