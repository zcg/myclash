//go:build linux

package platform

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func IsElevated() bool { return os.Geteuid() == 0 }

// RelaunchElevated uses pkexec (the desktop's policy agent) to replace this
// process with a root copy. pkexec prompts for the user's password and the
// shell waits only briefly before the old process is replaced.
func RelaunchElevated() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	script := fmt.Sprintf("sleep 0.5; kill -9 %d; exec %s", os.Getpid(), shellQuote(exe))
	cmd := exec.Command("pkexec", "/bin/sh", "-c", script)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'" }

func ForceQuit()       { os.Exit(0) }
func ArmKillWatchdog() {}
