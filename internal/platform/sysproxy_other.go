//go:build !windows

package platform

import "os"

// IsElevated is false where the app does not use an elevation flow yet.
func IsElevated() bool { return false }

// RelaunchElevated is only implemented on Windows.
func RelaunchElevated() error { return ErrUnsupported }

// ForceQuit terminates this process immediately.
func ForceQuit() { os.Exit(0) }

// ArmKillWatchdog is a no-op where the native tray does not need it.
func ArmKillWatchdog() {}
