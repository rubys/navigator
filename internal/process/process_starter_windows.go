//go:build windows

package process

import (
	"os/exec"
)

// setProcessCredentials is a no-op on Windows
func (ps *ProcessStarter) setProcessCredentials(cmd *exec.Cmd, cred *SysCredential) {
	// Windows doesn't use syscall.Credential
	// Would require different APIs (CreateProcessAsUser, etc.)
}
