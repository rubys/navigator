package utils

import (
	"log/slog"
	"time"
)

// ParseDurationWithDefault parses a duration string and returns defaultDuration if parsing fails or input is empty.
// Logs a warning if parsing fails.
func ParseDurationWithDefault(input string, defaultDuration time.Duration) time.Duration {
	if input == "" {
		return defaultDuration
	}
	duration, err := time.ParseDuration(input)
	if err != nil {
		slog.Warn("Invalid duration format, using default",
			"input", input,
			"default", defaultDuration,
			"error", err)
		return defaultDuration
	}
	return duration
}

// ParseDurationWithContext parses a duration with additional context for logging.
// If the input string is empty, returns the default duration.
// Logs a warning with context fields if parsing fails.
func ParseDurationWithContext(s string, defaultDuration time.Duration, context map[string]interface{}) time.Duration {
	if s == "" {
		return defaultDuration
	}

	duration, err := time.ParseDuration(s)
	if err != nil {
		attrs := []interface{}{"input", s, "default", defaultDuration, "error", err}
		for k, v := range context {
			attrs = append(attrs, k, v)
		}
		slog.Warn("Invalid duration format, using default", attrs...)
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
