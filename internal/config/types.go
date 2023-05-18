package config

import (
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

func MustAtoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return i
}

func MustParseBool(s string) bool {
	b, err := strconv.ParseBool(s)
	if err != nil {
		panic(err)
	}
	return b
}

func MustParseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic(err)
	}
	return d
}

func MustParseLogLevel(s string) logrus.Level {
	lvl, err := logrus.ParseLevel(s)
	if err != nil {
		panic(err)
	}
	return lvl
}
