//go:build windows

package process

import "syscall"

// SysCredential is a Windows-specific placeholder for syscall.Credential (which doesn't exist on Windows)
type SysCredential struct{}

// GetUserCredentials is not supported on Windows
// Returns nil (no credentials to set)
func GetUserCredentials(username, groupname string) (*SysCredential, error) {
	// Windows doesn't use syscall.Credential for process creation
	// User/group switching would require different APIs
	return nil, nil
}

// Windows doesn't have syscall.Credential, but we need the type to exist
// Create a type alias to our placeholder
type _ = syscall.SysProcAttr // Use a Windows syscall type to avoid unused import
