//go:build darwin || linux

package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// processExePath returns the real executable path of pid. /proc/pid/exe is a
// kernel-provided symlink on Linux; on macOS lsof reports the same path.
func processExePath(pid int) (string, bool) {
	if pid <= 0 {
		return "", false
	}
	if runtime.GOOS == "linux" {
		path, err := os.Readlink("/proc/" + strconv.Itoa(pid) + "/exe")
		if err == nil {
			return path, true
		}
		return "", false
	}
	out, err := exec.Command("lsof", "-p", strconv.Itoa(pid), "-Fn").Output()
	if err != nil {
		return "", false
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "n/") {
			return strings.TrimPrefix(line, "n"), true
		}
	}
	return "", false
}

// sameExecutable reports whether pid runs this exact executable file. A
// different program with the same basename never matches.
func sameExecutable(pid int, selfExe string) bool {
	path, ok := processExePath(pid)
	if !ok || path == "" {
		return false
	}
	if p, err := filepath.EvalSymlinks(path); err == nil {
		path = p
	}
	return filepath.Clean(path) == filepath.Clean(selfExe)
}

// KillOtherInstances force-terminates every OTHER process running this exact
// executable, verified by the kernel-reported executable path.
func KillOtherInstances() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	if p, err := filepath.EvalSymlinks(exe); err == nil {
		exe = p
	}
	self := os.Getpid()
	out, _ := exec.Command("ps", "-eo", "pid=").Output()
	for _, line := range strings.Split(string(out), "\n") {
		pid, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil || pid == self {
			continue
		}
		if sameExecutable(pid, exe) {
			KillPID(pid)
		}
	}
}

// IsSelfExecutable reports whether pid runs the same executable as this
// process, verified by the kernel-reported executable path.
func IsSelfExecutable(pid int) bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	if p, err := filepath.EvalSymlinks(exe); err == nil {
		exe = p
	}
	return sameExecutable(pid, exe)
}

// IsProcessAlive reports whether pid is a running process we may signal.
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

// KillPID force-terminates a process by PID. It never touches our own PID.
func KillPID(pid int) {
	if pid <= 0 || pid == os.Getpid() {
		return
	}
	if proc, err := os.FindProcess(pid); err == nil {
		_ = proc.Signal(syscall.SIGKILL)
	}
}

// WaitForExit waits until pid is gone, or the timeout elapses.
func WaitForExit(pid int, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !IsProcessAlive(pid) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// StartDetachedRelaunch starts a helper that waits for this PID to exit, then
// starts a fresh copy of the app with the same arguments and working dir.
func StartDetachedRelaunch(delay time.Duration) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	wd, err := os.Getwd()
	if err != nil {
		wd = ""
	}
	script := "sleep " + strconv.FormatFloat(delay.Seconds(), 'f', 3, 64) +
		"; while kill -0 " + strconv.Itoa(os.Getpid()) + " 2>/dev/null; do sleep 0.1; done; cd " + quoteShellArg(wd) +
		" && exec " + quoteShellArg(exe)
	for _, a := range os.Args[1:] {
		script += " " + quoteShellArg(a)
	}
	cmd := exec.Command("/bin/sh", "-c", script)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, nil, nil
	return cmd.Start()
}

func quoteShellArg(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
