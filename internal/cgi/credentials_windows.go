//go:build windows

package cgi

import (
	"os/exec"

	"github.com/rubys/navigator/internal/process"
)

// setProcessCredentials is a no-op on Windows
func (h *Handler) setProcessCredentials(cmd *exec.Cmd, cred *process.SysCredential) {
	// Windows doesn't use syscall.Credential
	// Would require different APIs (CreateProcessAsUser, etc.)
}
