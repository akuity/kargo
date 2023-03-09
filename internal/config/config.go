package config

import (
	"github.com/sirupsen/logrus"
)

type BaseConfig struct {
	LogLevel logrus.Level
}

func newBaseConfig() BaseConfig {
	return BaseConfig{
		LogLevel: MustParseLogLevel(MustGetEnv("LOG_LEVEL", "INFO")),
	}
}

type APIConfig struct {
	BaseConfig
}

func NewAPIConfig() APIConfig {
	return APIConfig{
		BaseConfig: newBaseConfig(),
	}
}

type ControllerConfig struct {
	BaseConfig
}

func NewControllerConfig() ControllerConfig {
	return ControllerConfig{
		BaseConfig: newBaseConfig(),
	}
}
