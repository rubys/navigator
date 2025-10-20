//go:build unix

package process

import (
	"os/exec"
	"syscall"
)

// setProcessCredentials sets the user and group for the process (Unix only)
func (ps *ProcessStarter) setProcessCredentials(cmd *exec.Cmd, cred *SysCredential) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Credential = cred
}
