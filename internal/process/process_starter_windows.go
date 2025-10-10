//go:build windows

package process

import (
	"os/exec"
	"syscall"
)

// setProcessCredentials is a no-op on Windows
func (ps *ProcessStarter) setProcessCredentials(cmd *exec.Cmd, cred *syscall.Credential) {
	// Windows doesn't use syscall.Credential
	// Would require different APIs (CreateProcessAsUser, etc.)
}
