//go:build !windows && !darwin && !linux

package platform

import "os"

func IsElevated() bool { return false }

func RelaunchElevated() error { return ErrUnsupported }

func ForceQuit() { os.Exit(0) }

func ArmKillWatchdog() {}
