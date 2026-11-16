//go:build windows

package platform

import "os/exec"

// KillOtherCores removes orphan headless cores left by an interrupted switch.
// It matches the command line, not the process name, so the GUI is untouched.
func KillOtherCores() {
	script := `Get-CimInstance Win32_Process -Filter "Name='myclash.exe'" | Where-Object { $_.CommandLine -match '--core' } | ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }`
	_ = exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", script).Run()
}
