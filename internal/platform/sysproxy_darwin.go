//go:build darwin

package platform

import (
	"fmt"
	"os/exec"
	"strings"
)

type darwinProxy struct{}

func NewSysProxy() SysProxy { return darwinProxy{} }

func (darwinProxy) service() string {
	out, err := exec.Command("networksetup", "-listallnetworkservices").Output()
	if err != nil {
		return "Wi-Fi"
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "*"))
		if line != "" && !strings.Contains(line, "denotes") {
			return line
		}
	}
	return "Wi-Fi"
}

func (darwinProxy) Snapshot() (ProxySnapshot, error) {
	return ProxySnapshot{}, nil
}

func (darwinProxy) Enable(host string, port int, bypass []string) error {
	svc := darwinProxy{}.service()
	for _, cmd := range [][]string{
		{"-setwebproxy", svc, host, fmt.Sprint(port)},
		{"-setsecurewebproxy", svc, host, fmt.Sprint(port)},
		{"-setwebproxystate", svc, "on"},
		{"-setsecurewebproxystate", svc, "on"},
	} {
		if out, err := exec.Command("networksetup", cmd...).CombinedOutput(); err != nil {
			return fmt.Errorf("networksetup %v: %w: %s", cmd, err, strings.TrimSpace(string(out)))
		}
	}
	if len(bypass) > 0 {
		args := append([]string{"-setproxybypassdomains", svc}, bypass...)
		_ = exec.Command("networksetup", args...).Run()
	}
	return nil
}

func (darwinProxy) Disable() error {
	svc := darwinProxy{}.service()
	_ = exec.Command("networksetup", "-setwebproxystate", svc, "off").Run()
	_ = exec.Command("networksetup", "-setsecurewebproxystate", svc, "off").Run()
	return nil
}

func (darwinProxy) Enabled() (bool, error) {
	svc := darwinProxy{}.service()
	out, err := exec.Command("networksetup", "-getwebproxy", svc).Output()
	if err != nil {
		return false, err
	}
	return strings.Contains(strings.ToLower(string(out)), "enabled: yes"), nil
}

func (darwinProxy) Restore(ProxySnapshot) error { return darwinProxy{}.Disable() }
