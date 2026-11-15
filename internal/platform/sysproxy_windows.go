//go:build windows

package platform

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const inetSettings = `Software\Microsoft\Windows\CurrentVersion\Internet Settings`

type windowsProxy struct{}

func NewSysProxy() SysProxy { return windowsProxy{} }

func (windowsProxy) open() (registry.Key, error) {
	return registry.OpenKey(registry.CURRENT_USER, inetSettings, registry.QUERY_VALUE|registry.SET_VALUE)
}

func (windowsProxy) Snapshot() (ProxySnapshot, error) {
	k, err := windowsProxy{}.open()
	if err != nil {
		return ProxySnapshot{}, err
	}
	defer k.Close()
	var s ProxySnapshot
	if v, _, err := k.GetIntegerValue("ProxyEnable"); err == nil {
		s.Enable = uint32(v)
	}
	s.Server, _, _ = k.GetStringValue("ProxyServer")
	s.Override, _, _ = k.GetStringValue("ProxyOverride")
	s.AutoConfig, _, _ = k.GetStringValue("AutoConfigURL")
	return s, nil
}

func (windowsProxy) Enable(host string, port int, bypass []string) error {
	k, err := windowsProxy{}.open()
	if err != nil {
		return err
	}
	defer k.Close()
	if err := k.SetDWordValue("ProxyEnable", 1); err != nil {
		return fmt.Errorf("set ProxyEnable: %w", err)
	}
	if err := k.SetStringValue("ProxyServer", fmt.Sprintf("%s:%d", host, port)); err != nil {
		return fmt.Errorf("set ProxyServer: %w", err)
	}
	if err := k.SetStringValue("ProxyOverride", strings.Join(bypass, ";")); err != nil {
		return fmt.Errorf("set ProxyOverride: %w", err)
	}
	_ = k.DeleteValue("AutoConfigURL")
	refreshWinINet()
	return nil
}

func (windowsProxy) Disable() error {
	k, err := windowsProxy{}.open()
	if err != nil {
		return err
	}
	defer k.Close()
	if err := k.SetDWordValue("ProxyEnable", 0); err != nil {
		return fmt.Errorf("set ProxyEnable: %w", err)
	}
	refreshWinINet()
	return nil
}

func (windowsProxy) Enabled() (bool, error) {
	k, err := windowsProxy{}.open()
	if err != nil {
		return false, err
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("ProxyEnable")
	if err != nil {
		return false, nil
	}
	return v != 0, nil
}

func (windowsProxy) Restore(s ProxySnapshot) error {
	k, err := windowsProxy{}.open()
	if err != nil {
		return err
	}
	defer k.Close()
	if err := k.SetDWordValue("ProxyEnable", s.Enable); err != nil {
		return err
	}
	setOrDelete(k, "ProxyServer", s.Server)
	setOrDelete(k, "ProxyOverride", s.Override)
	setOrDelete(k, "AutoConfigURL", s.AutoConfig)
	refreshWinINet()
	return nil
}

func setOrDelete(k registry.Key, name, value string) {
	if value == "" {
		_ = k.DeleteValue(name)
		return
	}
	_ = k.SetStringValue(name, value)
}

var wininet = windows.NewLazySystemDLL("wininet.dll")
var procInternetSetOption = wininet.NewProc("InternetSetOptionW")

func refreshWinINet() {
	const (
		internetOptionRefresh         = 37
		internetOptionSettingsChanged = 39
	)
	_, _, _ = procInternetSetOption.Call(0, internetOptionSettingsChanged, 0, 0)
	_, _, _ = procInternetSetOption.Call(0, internetOptionRefresh, 0, 0)
}

// IsElevated reports whether this process runs with the administrator token.
func IsElevated() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}

// RelaunchElevated starts this executable again with the administrator token.
func RelaunchElevated() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	// The app holds a single-instance lock, so the elevated copy must start
	// only after this process has exited. A detached helper terminates this
	// PID (bypassing the close-to-tray path and the Go runtime's shutdown
	// callback), then raises the UAC prompt and starts the executable again.
	script := fmt.Sprintf(
		"Start-Sleep -Milliseconds 250; Stop-Process -Id %d -Force -ErrorAction SilentlyContinue; Start-Sleep -Milliseconds 400; try { Start-Process -FilePath %q -Verb RunAs -ErrorAction Stop } catch { Start-Process -FilePath %q -ArgumentList '--tun-off' }",
		os.Getpid(), exe, exe)
	cmd := exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", script)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

// ForceQuit terminates this process immediately.
func ForceQuit() {
	_ = windows.TerminateProcess(windows.CurrentProcess(), 0)
}
