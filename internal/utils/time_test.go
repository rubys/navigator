package utils

import (
	"testing"
	"time"
)

func TestParseDurationWithDefault(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		defaultDuration time.Duration
		expected        time.Duration
	}{
		{
			name:            "valid duration string",
			input:           "5m",
			defaultDuration: 10 * time.Minute,
			expected:        5 * time.Minute,
		},
		{
			name:            "empty string returns default",
			input:           "",
			defaultDuration: 10 * time.Minute,
			expected:        10 * time.Minute,
		},
		{
			name:            "invalid duration returns default",
			input:           "invalid",
			defaultDuration: 30 * time.Second,
			expected:        30 * time.Second,
		},
		{
			name:            "complex duration string",
			input:           "1h30m45s",
			defaultDuration: 1 * time.Hour,
			expected:        1*time.Hour + 30*time.Minute + 45*time.Second,
		},
		{
			name:            "zero duration",
			input:           "0s",
			defaultDuration: 5 * time.Minute,
			expected:        0,
		},
		{
			name:            "negative duration",
			input:           "-5m",
			defaultDuration: 10 * time.Minute,
			expected:        -5 * time.Minute,
		},
		{
			name:            "1 year duration",
			input:           "1y",
			defaultDuration: 0,
			expected:        8760 * time.Hour, // 365 days
		},
		{
			name:            "1 week duration",
			input:           "1w",
			defaultDuration: 0,
			expected:        168 * time.Hour, // 7 days
		},
		{
			name:            "7 days duration",
			input:           "7d",
			defaultDuration: 0,
			expected:        168 * time.Hour,
		},
		{
			name:            "1.5 years duration",
			input:           "1.5y",
			defaultDuration: 0,
			expected:        13140 * time.Hour, // 547.5 days
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseDurationWithDefault(tt.input, tt.defaultDuration)
			if result != tt.expected {
				t.Errorf("ParseDurationWithDefault(%q, %v) = %v, want %v",
					tt.input, tt.defaultDuration, result, tt.expected)
			}
		})
	}
}

func TestMustParseDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
		panics   bool
	}{
		{
			name:     "valid duration",
			input:    "30s",
			expected: 30 * time.Second,
			panics:   false,
		},
		{
			name:     "complex valid duration",
			input:    "2h45m",
			expected: 2*time.Hour + 45*time.Minute,
			panics:   false,
		},
		{
			name:   "invalid duration panics",
			input:  "invalid",
			panics: true,
		},
		{
			name:   "empty string panics",
			input:  "",
			panics: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.panics {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("MustParseDuration(%q) should have panicked", tt.input)
					}
				}()
				MustParseDuration(tt.input)
			} else {
				result := MustParseDuration(tt.input)
				if result != tt.expected {
					t.Errorf("MustParseDuration(%q) = %v, want %v",
						tt.input, result, tt.expected)
				}
			}
		})
	}
}

func TestParseDurationWithContext(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		def      time.Duration
		context  map[string]interface{}
		expected time.Duration
	}{
		{
			name:  "Valid duration with context",
			input: "2h",
			def:   1 * time.Hour,
			context: map[string]interface{}{
				"process": "test-process",
			},
			expected: 2 * time.Hour,
		},
		{
			name:  "Invalid duration with context",
			input: "bad-duration",
			def:   5 * time.Minute,
			context: map[string]interface{}{
				"process": "failing-process",
				"field":   "timeout",
			},
			expected: 5 * time.Minute,
		},
		{
			name:     "Empty context",
			input:    "10m",
			def:      1 * time.Minute,
			context:  map[string]interface{}{},
			expected: 10 * time.Minute,
		},
		{
			name:     "Nil context",
			input:    "15s",
			def:      30 * time.Second,
			context:  nil,
			expected: 15 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseDurationWithContext(tt.input, tt.def, tt.context)
			if result != tt.expected {
				t.Errorf("ParseDurationWithContext(%q, %v, %v) = %v, want %v",
					tt.input, tt.def, tt.context, result, tt.expected)
			}
		})
	}
}
