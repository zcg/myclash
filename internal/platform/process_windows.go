//go:build windows

package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"

	"golang.org/x/sys/windows"
)

// processExePath returns the executable image path of pid, resolved from the
// OS process handle rather than a command line that can be forged.
func processExePath(pid int) (string, bool) {
	if pid <= 0 {
		return "", false
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return "", false
	}
	defer windows.CloseHandle(h)
	buf := make([]uint16, windows.MAX_PATH+1)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil || size == 0 {
		return "", false
	}
	return string(utf16.Decode(buf[:size])), true
}

// sameExecutable reports whether pid runs this exact executable file. Path
// comparison is case-insensitive on Windows; the executable path is resolved
// by the kernel, so a different program with the same name never matches.
func sameExecutable(pid int, selfExe string) bool {
	path, ok := processExePath(pid)
	if !ok || path == "" {
		return false
	}
	p, err := filepath.EvalSymlinks(path)
	if err == nil {
		path = p
	}
	return strings.EqualFold(filepath.Clean(path), filepath.Clean(selfExe))
}

// KillOtherInstances force-terminates every OTHER process running this exact
// executable. Processes with the same name but a different image path are
// left alone.
func KillOtherInstances() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	if p, err := filepath.EvalSymlinks(exe); err == nil {
		exe = p
	}
	self := os.Getpid()
	// Enumerate PIDs from the OS, then verify each image path with a kernel
	// handle. Never trust Taskkill by name.
	out, _ := exec.Command("tasklist", "/FO", "CSV", "/NH").Output()
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Split(line, `","`)
		if len(fields) < 2 {
			continue
		}
		pidText := strings.Trim(fields[1], `" `)
		pid, err := strconv.Atoi(pidText)
		if err != nil || pid == self {
			continue
		}
		if sameExecutable(pid, exe) {
			KillPID(pid)
		}
	}
}

// IsSelfExecutable reports whether pid runs the same executable as this
// process, verified by the kernel-reported image path.
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

// IsProcessAlive reports whether pid names a running process.
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	var code uint32
	if err := windows.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	return code == 259 // STILL_ACTIVE
}

// KillPID force-terminates a process by PID. It never touches our own PID.
func KillPID(pid int) {
	if pid <= 0 || pid == os.Getpid() {
		return
	}
	_ = exec.Command("taskkill", "/F", "/PID", strconv.Itoa(pid)).Run()
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
	script := "Start-Sleep -Milliseconds " + strconv.Itoa(int(delay.Milliseconds())) +
		"; while (Get-Process -Id " + strconv.Itoa(os.Getpid()) + " -ErrorAction SilentlyContinue) { Start-Sleep -Milliseconds 100 }; " +
		"Start-Process -FilePath " + quotePS(exe) + " -WorkingDirectory " + quotePS(wd) +
		" -ArgumentList @(" + psArgs(os.Args[1:]) + ")"
	cmd := exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000008 | 0x00000200, HideWindow: true} // DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP
	return cmd.Start()
}

func quotePS(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

func psArgs(args []string) string {
	q := make([]string, 0, len(args))
	for _, a := range args {
		q = append(q, "'"+strings.ReplaceAll(a, "'", "''")+"'")
	}
	return strings.Join(q, ",")
}
