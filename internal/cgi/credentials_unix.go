//go:build unix

package cgi

import (
	"os/exec"
	"syscall"

	"github.com/rubys/navigator/internal/process"
)

// setProcessCredentials sets the user and group for the CGI process (Unix only)
func (h *Handler) setProcessCredentials(cmd *exec.Cmd, cred *process.SysCredential) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Credential = cred
}
