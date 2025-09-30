package utils

import (
	"os/exec"
	"reflect"
	"testing"
)

func TestSetCommandEnvironment(t *testing.T) {
	tests := []struct {
		name   string
		envMap map[string]string
	}{
		{
			name:   "empty environment map",
			envMap: map[string]string{},
		},
		{
			name: "single environment variable",
			envMap: map[string]string{
				"TEST_VAR": "test_value",
			},
		},
		{
			name: "multiple environment variables",
			envMap: map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
				"VAR3": "value3",
			},
		},
		{
			name: "environment with special characters",
			envMap: map[string]string{
				"PATH_VAR":  "/usr/local/bin:/usr/bin",
				"QUOTE_VAR": "value with spaces",
				"EQUAL_VAR": "key=value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &exec.Cmd{}
			SetCommandEnvironment(cmd, tt.envMap)

			// Check that all expected variables are in the environment
			for key, value := range tt.envMap {
				expected := key + "=" + value
				found := false
				for _, env := range cmd.Env {
					if env == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected environment variable %q not found", expected)
				}
			}

			// Ensure we have at least the number of vars we added
			// (plus system environment variables)
			if len(cmd.Env) < len(tt.envMap) {
				t.Errorf("Environment has %d variables, expected at least %d",
					len(cmd.Env), len(tt.envMap))
			}
		})
	}
}

func TestMergeEnvironment(t *testing.T) {
	tests := []struct {
		name      string
		base      []string
		overrides map[string]string
		want      map[string]string // Expected key-value pairs
	}{
		{
			name: "merge with empty base",
			base: []string{},
			overrides: map[string]string{
				"NEW_VAR": "new_value",
			},
			want: map[string]string{
				"NEW_VAR": "new_value",
			},
		},
		{
			name:      "merge with empty overrides",
			base:      []string{"EXISTING=value"},
			overrides: map[string]string{},
			want: map[string]string{
				"EXISTING": "value",
			},
		},
		{
			name: "override existing variable",
			base: []string{"VAR1=old_value", "VAR2=keep_this"},
			overrides: map[string]string{
				"VAR1": "new_value",
			},
			want: map[string]string{
				"VAR1": "new_value",
				"VAR2": "keep_this",
			},
		},
		{
			name: "add new variables",
			base: []string{"EXISTING=value"},
			overrides: map[string]string{
				"NEW1": "value1",
				"NEW2": "value2",
			},
			want: map[string]string{
				"EXISTING": "value",
				"NEW1":     "value1",
				"NEW2":     "value2",
			},
		},
		{
			name: "handle malformed base entries",
			base: []string{"VALID=value", "INVALID", "ALSO_VALID=ok"},
			overrides: map[string]string{
				"NEW": "new_value",
			},
			want: map[string]string{
				"VALID":      "value",
				"ALSO_VALID": "ok",
				"NEW":        "new_value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeEnvironment(tt.base, tt.overrides)

			// Convert result to map for easier comparison
			resultMap := make(map[string]string)
			for _, env := range result {
				parts := splitEnvVar(env)
				if len(parts) == 2 {
					resultMap[parts[0]] = parts[1]
				}
			}

			// Check all expected values are present
			for key, expectedValue := range tt.want {
				if actualValue, exists := resultMap[key]; !exists {
					t.Errorf("Missing key %q in result", key)
				} else if actualValue != expectedValue {
					t.Errorf("For key %q: got value %q, want %q",
						key, actualValue, expectedValue)
				}
			}

			// Check no unexpected keys
			for key := range resultMap {
				if _, expected := tt.want[key]; !expected {
					t.Errorf("Unexpected key %q in result", key)
				}
			}
		})
	}
}

func TestExpandVariables(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		vars map[string]string
		want map[string]string
	}{
		{
			name: "no variables to expand",
			env: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
			},
			vars: map[string]string{
				"unused": "value",
			},
			want: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
			},
		},
		{
			name: "single variable expansion",
			env: map[string]string{
				"DATABASE": "app_${env}",
			},
			vars: map[string]string{
				"env": "production",
			},
			want: map[string]string{
				"DATABASE": "app_production",
			},
		},
		{
			name: "multiple variable expansion",
			env: map[string]string{
				"DB_NAME":  "${app}_${env}",
				"DB_USER":  "${app}_user",
				"LOG_PATH": "/var/log/${app}/${env}.log",
			},
			vars: map[string]string{
				"app": "myapp",
				"env": "staging",
			},
			want: map[string]string{
				"DB_NAME":  "myapp_staging",
				"DB_USER":  "myapp_user",
				"LOG_PATH": "/var/log/myapp/staging.log",
			},
		},
		{
			name: "variable not found keeps placeholder",
			env: map[string]string{
				"PATH": "/app/${undefined}/bin",
			},
			vars: map[string]string{
				"other": "value",
			},
			want: map[string]string{
				"PATH": "/app/${undefined}/bin",
			},
		},
		{
			name: "same variable used multiple times",
			env: map[string]string{
				"CONNECTION": "${host}:${port}",
				"URL":        "http://${host}:${port}/api",
			},
			vars: map[string]string{
				"host": "localhost",
				"port": "8080",
			},
			want: map[string]string{
				"CONNECTION": "localhost:8080",
				"URL":        "http://localhost:8080/api",
			},
		},
		{
			name: "empty variables map",
			env: map[string]string{
				"KEY": "${var}",
			},
			vars: map[string]string{},
			want: map[string]string{
				"KEY": "${var}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandVariables(tt.env, tt.vars)

			if !reflect.DeepEqual(result, tt.want) {
				t.Errorf("ExpandVariables() = %v, want %v", result, tt.want)
			}
		})
	}
}

// Helper function to split environment variable string
func splitEnvVar(env string) []string {
	for i := 0; i < len(env); i++ {
		if env[i] == '=' {
			return []string{env[:i], env[i+1:]}
		}
	}
	return []string{env}
}
