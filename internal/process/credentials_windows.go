//go:build windows

package process

import "syscall"

// GetUserCredentials is not supported on Windows
// Returns nil (no credentials to set)
func GetUserCredentials(username, groupname string) (*syscall.Credential, error) {
	// Windows doesn't use syscall.Credential for process creation
	// User/group switching would require different APIs
	return nil, nil
}
