package os

import (
	"fmt"
	"os"
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
