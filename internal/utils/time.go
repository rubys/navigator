package utils

import (
	"log/slog"
	"strconv"
	"strings"
	"time"
)

// ParseDurationWithDefault parses a duration string and returns defaultDuration if parsing fails or input is empty.
// Supports extended duration units: y (years), w (weeks), d (days) in addition to Go's standard units (h, m, s, ms, us, ns).
// Logs a warning if parsing fails.
func ParseDurationWithDefault(input string, defaultDuration time.Duration) time.Duration {
	if input == "" {
		return defaultDuration
	}

	// Try extended duration parsing first (y, w, d)
	if duration, ok := parseExtendedDuration(input); ok {
		return duration
	}

	// Fall back to standard Go duration parsing
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

// parseExtendedDuration handles y (years), w (weeks), d (days) durations
// Returns (duration, true) if successfully parsed, (0, false) otherwise
func parseExtendedDuration(s string) (time.Duration, bool) {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return 0, false
	}

	// Extract number and unit
	var numStr string
	var unit string
	for i, ch := range s {
		if (ch >= '0' && ch <= '9') || ch == '.' {
			numStr += string(ch)
		} else {
			unit = s[i:]
			break
		}
	}

	if numStr == "" || unit == "" {
		return 0, false
	}

	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, false
	}

	// Convert extended units to hours
	var hours float64
	switch unit {
	case "y":
		hours = num * 24 * 365 // 1 year = 365 days
	case "w":
		hours = num * 24 * 7 // 1 week = 7 days
	case "d":
		hours = num * 24 // 1 day = 24 hours
	default:
		return 0, false
	}

	return time.Duration(hours * float64(time.Hour)), true
}

// ParseDurationWithContext parses a duration with additional context for logging.
// If the input string is empty, returns the default duration.
// Supports extended duration units: y (years), w (weeks), d (days).
// Logs a warning with context fields if parsing fails.
func ParseDurationWithContext(s string, defaultDuration time.Duration, context map[string]interface{}) time.Duration {
	if s == "" {
		return defaultDuration
	}

	// Try extended duration parsing first (y, w, d)
	if duration, ok := parseExtendedDuration(s); ok {
		return duration
	}

	// Fall back to standard Go duration parsing
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
