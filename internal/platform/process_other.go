//go:build !windows && !darwin && !linux

package platform

import "time"

func KillOtherInstances()                       {}
func IsSelfExecutable(int) bool                 { return false }
func IsProcessAlive(int) bool                   { return false }
func KillPID(int)                               {}
func WaitForExit(int, time.Duration)            {}
func StartDetachedRelaunch(time.Duration) error { return ErrUnsupported }
