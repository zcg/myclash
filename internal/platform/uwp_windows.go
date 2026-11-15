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
	DisplayName     string `json:"DisplayName"`
	Publisher       string `json:"Publisher"`
	Loopback        bool   `json:"-"`
}

func ListUWP() ([]UWPApp, error) {
	script := `$start = @{}; Get-StartApps -ErrorAction SilentlyContinue | ForEach-Object { $start[$_.AppID] = $_.Name }
$apps = Get-AppxPackage | Where-Object { -not $_.IsFramework -and $_.PackageFullName } | ForEach-Object {
  $dn = $_.Name
  try { $dn = (Get-AppxPackageManifest $_ -ErrorAction Stop).Package.Properties.DisplayName } catch {}
  if (-not $dn -or $dn -like 'ms-resource:*') {
    $fam = ($_.PackageFullName -split '_')[0]
    foreach ($k in $start.Keys) { if ($k -like "$fam!*") { $dn = $start[$k]; break } }
  }
  [pscustomobject]@{ Name = $_.Name; PackageFullName = $_.PackageFullName; DisplayName = $dn; Publisher = $_.PublisherDisplayName }
}
$apps | ConvertTo-Json -Compress`
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
	return SetUWPLoopbackMany([]string{packageFullName}, enable)
}

// SetUWPLoopbackMany asks for UAC once and applies the operation to every
// package in one elevated process, instead of one prompt per app.
func SetUWPLoopbackMany(packages []string, enable bool) error {
	flag := "-d"
	if enable {
		flag = "-a"
	}
	quoted := make([]string, 0, len(packages))
	for _, p := range packages {
		if strings.TrimSpace(p) != "" {
			quoted = append(quoted, "'"+strings.ReplaceAll(p, "'", "''")+"'")
		}
	}
	if len(quoted) == 0 {
		return nil
	}
	if IsElevated() {
		for _, p := range packages {
			if err := exec.Command("CheckNetIsolation.exe", "LoopbackExempt", flag, "-n="+p).Run(); err != nil {
				return err
			}
		}
		return nil
	}
	body := fmt.Sprintf("$ids=@(%s); foreach($id in $ids){ & \"$env:SystemRoot\\System32\\CheckNetIsolation.exe\" LoopbackExempt %s \"-n=$id\" }", strings.Join(quoted, ","), flag)
	script := fmt.Sprintf("Start-Process powershell -ArgumentList '-NoProfile','-WindowStyle','Hidden','-Command',@'%s'@ -Verb RunAs -Wait", body)
	return exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", script).Run()
}
