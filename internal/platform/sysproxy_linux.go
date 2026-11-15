//go:build linux

package platform

import (
	"fmt"
	"os/exec"
	"strings"
)

type gnomeProxy struct{}

func NewSysProxy() SysProxy { return gnomeProxy{} }

func gsettings(args ...string) (string, error) {
	out, err := exec.Command("gsettings", args...).Output()
	return strings.TrimSpace(string(out)), err
}

func (gnomeProxy) Snapshot() (ProxySnapshot, error) { return ProxySnapshot{}, nil }

func (gnomeProxy) Enable(host string, port int, bypass []string) error {
	cmds := [][]string{
		{"set", "org.gnome.system.proxy", "mode", "manual"},
		{"set", "org.gnome.system.proxy.http", "host", host},
		{"set", "org.gnome.system.proxy.http", "port", fmt.Sprint(port)},
		{"set", "org.gnome.system.proxy.https", "host", host},
		{"set", "org.gnome.system.proxy.https", "port", fmt.Sprint(port)},
	}
	for _, cmd := range cmds {
		if out, err := exec.Command("gsettings", cmd...).CombinedOutput(); err != nil {
			return fmt.Errorf("gsettings %v: %w: %s", cmd, err, strings.TrimSpace(string(out)))
		}
	}
	if len(bypass) > 0 {
		quoted := make([]string, len(bypass))
		for i, v := range bypass {
			quoted[i] = "'" + v + "'"
		}
		_ = exec.Command("gsettings", "set", "org.gnome.system.proxy", "ignore-hosts",
			"["+strings.Join(quoted, ", ")+"]").Run()
	}
	return nil
}

func (gnomeProxy) Disable() error {
	_, err := gsettings("set", "org.gnome.system.proxy", "mode", "none")
	return err
}

func (gnomeProxy) Enabled() (bool, error) {
	out, err := gsettings("get", "org.gnome.system.proxy", "mode")
	if err != nil {
		return false, err
	}
	return strings.Trim(out, "'") == "manual", nil
}

func (gnomeProxy) Restore(ProxySnapshot) error { return gnomeProxy{}.Disable() }
