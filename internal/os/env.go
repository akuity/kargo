package os

import (
	"fmt"
	"os"
	"strconv"
)

// GetEnv retrieves the value of an environment variable having the specified
// key. If the value is empty string, a specified default is returned instead.
func GetEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// MustGetEnv retrieves the value of an environment variable having the
// specified key. If the value is empty string, a specified default is returned
// instead. It will panic if the defaultValue is empty too.
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

// MustGetEnvAsBool attempts to parse a bool from a string value retrieved from
// the specified environment variable. If the environment variable is undefined,
// the specified default value is returned instead. If the environment variable
// is defined and its value cannot successfully be parsed as a bool, the
// function panics.
func MustGetEnvAsBool(name string, defaultValue bool) bool {
	valStr := os.Getenv(name)
	if valStr == "" {
		return defaultValue
	}
	if val, err := strconv.ParseBool(valStr); err == nil {
		return val
	}
	panic(
		fmt.Sprintf(
			"value %q for environment variable %s was not parsable as a bool",
			valStr,
			name,
		),
	)
}
