//go:build !windows

package platform

import "errors"

func AgentAtLogin() bool { return false }

func SetAgentAtLogin(bool) error { return errors.New("elevated Agent login task is Windows only") }
