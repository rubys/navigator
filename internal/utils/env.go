package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// SetCommandEnvironment sets environment variables for an exec.Cmd
func SetCommandEnvironment(cmd *exec.Cmd, envMap map[string]string) {
	cmd.Env = os.Environ()
	for key, value := range envMap {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}
}

// MergeEnvironment merges environment variables, with later values overwriting earlier ones
func MergeEnvironment(base []string, overrides map[string]string) []string {
	// Build a map from base environment
	envMap := make(map[string]string)
	for _, env := range base {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Apply overrides
	for key, value := range overrides {
		envMap[key] = value
	}

	// Convert back to slice
	result := make([]string, 0, len(envMap))
	for key, value := range envMap {
		result = append(result, fmt.Sprintf("%s=%s", key, value))
	}
	return result
}

// ExpandVariables expands ${var} placeholders in the given map using provided variables
func ExpandVariables(env map[string]string, vars map[string]string) map[string]string {
	result := make(map[string]string)
	for key, value := range env {
		expanded := value
		for varName, varValue := range vars {
			expanded = strings.ReplaceAll(expanded, "${"+varName+"}", varValue)
		}
		result[key] = expanded
	}
	return result
}