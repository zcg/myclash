//go:build darwin

package platform

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func IsElevated() bool { return os.Geteuid() == 0 }

// RelaunchElevated asks macOS for administrator rights and starts this
// executable as root after the current process exits. If the user cancels the
// authorization sheet, the shell command fails and the current process stays
// alive; the caller can then leave TUN off instead of pretending it is on.
func RelaunchElevated() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	script := fmt.Sprintf("sleep 0.5; kill -9 %d; nohup %s >/dev/null 2>&1 &", os.Getpid(), shellQuote(exe))
	cmd := exec.Command("osascript", "-e", fmt.Sprintf("do shell script %s with administrator privileges", appleString(script)))
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'" }

func appleString(s string) string {
	return strconv.Quote(s)
}

func ForceQuit()       { os.Exit(0) }
func ArmKillWatchdog() {}
