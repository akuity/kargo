package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
)

// MustGetEnv returns environment variable named `key`.
// If the value is an empty string, it returns `defaultValue`.
func MustGetEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		if defaultValue != "" {
			value = defaultValue
		} else {
			panic(fmt.Sprintf("missing required environment variable: %s", key))
		}
	}
	return value
}

func MustAtoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return i
}

func MustParseLogLevel(s string) logrus.Level {
	lvl, err := logrus.ParseLevel(s)
	if err != nil {
		panic(err)
	}
	return lvl
}
