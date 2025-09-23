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