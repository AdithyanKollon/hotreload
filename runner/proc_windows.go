//go:build windows

package runner

import (
	"syscall"
)

// hiddenWindowAttr returns SysProcAttr that prevents child processes from
// opening a visible console window on Windows.
func hiddenWindowAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}
