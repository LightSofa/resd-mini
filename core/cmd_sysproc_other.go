//go:build !windows

package core

import "os/exec"

func applyCmdSysProcAttr(cmd *exec.Cmd) {}
