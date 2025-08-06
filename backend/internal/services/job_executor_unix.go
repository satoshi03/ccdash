//go:build darwin || linux

package services

import (
	"os/exec"
	"syscall"
)

// configurePlatformSpecificAttrs sets platform-specific process attributes for Unix-like systems
func configurePlatformSpecificAttrs(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// Inherit the same user/group permissions as the backend process
		Credential: nil, // nil means inherit current process credentials
		// Prevent the process from being stopped by TTY signals
		Setsid: true, // Create a new session to detach from controlling terminal
	}
}