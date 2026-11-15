//go:build darwin

package platform

import "os/exec"

func (darwinProxy) EnablePAC(pacURL string) error {
	svc := darwinProxy{}.service()
	_ = exec.Command("networksetup", "-setautoproxyurl", svc, pacURL).Run()
	return exec.Command("networksetup", "-setautoproxystate", svc, "on").Run()
}

func (darwinProxy) DisablePAC() error {
	svc := darwinProxy{}.service()
	return exec.Command("networksetup", "-setautoproxystate", svc, "off").Run()
}

var _ PACProxy = darwinProxy{}
