//go:build !windows

package platform

import (
	"os/exec"
	"syscall"
)

func Detach(cmd *exec.Cmd) {
	if cmd != nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	}
}
