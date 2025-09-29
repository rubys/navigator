package utils

import (
	"time"
)

// ParseDurationWithDefault parses a duration string and returns defaultDuration if parsing fails or input is empty
func ParseDurationWithDefault(input string, defaultDuration time.Duration) time.Duration {
	if input == "" {
		return defaultDuration
	}
	duration, err := time.ParseDuration(input)
	if err != nil {
		return defaultDuration
	}
	return duration
}

// MustParseDuration parses a duration string and panics if it fails (for compile-time constants)
func MustParseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic(err)
	}
	return d
}
