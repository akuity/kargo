package types

import (
	"strconv"
	"time"
)

func MustParseBool(s string) bool {
	b, err := strconv.ParseBool(s)
	if err != nil {
		panic(err)
	}
	return b
}

func MustParseDuration(s string) *time.Duration {
	if s == "" {
		return nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		panic(err)
	}
	return &d
}
