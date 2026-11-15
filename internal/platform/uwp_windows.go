//go:build windows

package platform

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type UWPApp struct {
	Name            string `json:"Name"`
	PackageFullName string `json:"PackageFullName"`
	Loopback        bool   `json:"-"`
}

func ListUWP() ([]UWPApp, error) {
	script := `$apps = Get-AppxPackage | Where-Object { -not $_.IsFramework -and $_.PackageFullName } | Select-Object Name,PackageFullName; $apps | ConvertTo-Json -Compress`
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).Output()
	if err != nil {
		return nil, fmt.Errorf("list UWP: %w", err)
	}
	text := strings.TrimSpace(string(out))
	if text == "" || text == "null" {
		return nil, nil
	}
	var apps []UWPApp
	if err := json.Unmarshal([]byte(text), &apps); err != nil {
		var one UWPApp
		if err2 := json.Unmarshal([]byte(text), &one); err2 != nil {
			return nil, fmt.Errorf("decode UWP list: %w", err)
		}
		apps = []UWPApp{one}
	}
	exempt := loopbackExempt()
	for i := range apps {
		apps[i].Loopback = exempt[strings.ToLower(apps[i].PackageFullName)]
	}
	return apps, nil
}

func loopbackExempt() map[string]bool {
	out, err := exec.Command("CheckNetIsolation.exe", "LoopbackExempt", "-s").Output()
	if err != nil {
		return nil
	}
	outMap := map[string]bool{}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if i := strings.Index(strings.ToLower(line), "-n="); i >= 0 {
			name := strings.TrimSpace(line[i+3:])
			name = strings.Trim(name, "\"")
			outMap[strings.ToLower(name)] = true
		}
	}
	return outMap
}

func SetUWPLoopback(packageFullName string, enable bool) error {
	flag := "-d"
	if enable {
		flag = "-a"
	}
	arg := fmt.Sprintf("LoopbackExempt %s -n=%q", flag, packageFullName)
	if IsElevated() {
		return exec.Command("CheckNetIsolation.exe", "LoopbackExempt", flag, "-n="+packageFullName).Run()
	}
	// The app is normally unelevated; ask Windows for a single elevated
	// CheckNetIsolation call rather than restarting the whole UI.
	script := fmt.Sprintf(`Start-Process -FilePath "$env:SystemRoot\System32\CheckNetIsolation.exe" -ArgumentList '%s' -Verb RunAs -Wait`, arg)
	return exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", script).Run()
}
