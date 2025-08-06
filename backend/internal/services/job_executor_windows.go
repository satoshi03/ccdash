//go:build windows

package services

import (
	"os/exec"
	"syscall"
)

// configurePlatformSpecificAttrs sets platform-specific process attributes for Windows
func configurePlatformSpecificAttrs(cmd *exec.Cmd) {
	// Windows doesn't support Credential or Setsid fields
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// Windows-specific configuration can be added here if needed
		// For now, we use an empty struct which is valid on Windows
	}
}