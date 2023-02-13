package os

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// GetEnvVar retrieves the value of an environment variable having the specified
// name. If that value is the empty string, a specified default is returned
// instead.
func GetEnvVar(name, defaultValue string) string {
	val := os.Getenv(name)
	if val == "" {
		return defaultValue
	}
	return val
}

// GetRequiredEnvVar retrieves the value of an environment variable having the
// specified name. If that value is the empty string, an error is returned.
func GetRequiredEnvVar(name string) (string, error) {
	val := os.Getenv(name)
	if val == "" {
		return "", errors.Errorf(
			"value not found for required environment variable %s",
			name,
		)
	}
	return val, nil
}

// GetStringSliceFromEnvVar retrieves comma-delimited values from an environment
// variable having the specified name and populates a string slice.
func GetStringSliceFromEnvVar(name string, defaultValue []string) []string {
	valStr := os.Getenv(name)
	if valStr == "" {
		return defaultValue
	}
	return strings.Split(valStr, ",")
}

// GetIntFromEnvVar attempts to parse an integer from a string value retrieved
// from the specified environment variable. An error is returned if the string
// value cannot successfully be parsed as an integer.
func GetIntFromEnvVar(name string, defaultValue int) (int, error) {
	valStr := os.Getenv(name)
	if valStr == "" {
		return defaultValue, nil
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return 0, errors.Errorf(
			"value %q for environment variable %s was not parsable as an int",
			valStr,
			name,
		)
	}
	return val, nil
}

// GetBoolFromEnvVar attempts to parse a bool from a string value retrieved from
// the specified environment variable. An error is returned if the string value
// cannot successfully be parsed as a bool.
func GetBoolFromEnvVar(name string, defaultValue bool) (bool, error) {
	valStr := os.Getenv(name)
	if valStr == "" {
		return defaultValue, nil
	}
	val, err := strconv.ParseBool(valStr)
	if err != nil {
		return false, errors.Errorf(
			"value %q for environment variable %s was not parsable as a bool",
			valStr,
			name,
		)
	}
	return val, nil
}

// GetDurationFromEnvVar attempts to parse a time.Duration from a string value
// retrieved from the specified environment variable. An error is returned if
// the string value cannot successfully be parsed as a time.Duration.
func GetDurationFromEnvVar(
	name string,
	defaultValue time.Duration,
) (time.Duration, error) {
	valStr := os.Getenv(name)
	if valStr == "" {
		return defaultValue, nil
	}
	val, err := time.ParseDuration(valStr)
	if err != nil {
		return 0, errors.Errorf(
			"value %q for environment variable %s was not parsable as a duration",
			valStr,
			name,
		)
	}
	return val, nil
}
