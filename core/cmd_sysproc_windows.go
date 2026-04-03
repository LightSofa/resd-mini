//go:build windows

package core

import (
	"os/exec"
	"syscall"
)

func applyCmdSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
}
