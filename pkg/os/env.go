package os

import (
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

// GetEnvInt retrieves the value of an environment variable having the specified
// key. If the value is empty string, or cannot parse as an int, a specified
// default is returned instead.
func GetEnvInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseInt(valueStr, 10, 0)
	if err != nil {
		return defaultValue
	}
	return int(value)
}
