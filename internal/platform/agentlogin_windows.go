//go:build windows

package platform

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const agentTaskName = "myclash-agent"

func AgentAtLogin() bool {
	return exec.Command("schtasks", "/Query", "/TN", agentTaskName).Run() == nil
}

func SetAgentAtLogin(enable bool) error {
	if !enable {
		return runSchtasks("/Delete", "/TN", agentTaskName, "/F")
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return runSchtasks("/Create", "/SC", "ONLOGON", "/RL", "HIGHEST",
		"/F", "/TN", agentTaskName, "/TR", fmt.Sprintf("\"%s\" --agent", exe))
}

func runSchtasks(args ...string) error {
	if IsElevated() {
		out, err := exec.Command("schtasks", args...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("schtasks: %w: %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	}
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = "'" + strings.ReplaceAll(a, "'", "''") + "'"
	}
	script := fmt.Sprintf("Start-Process schtasks -ArgumentList %s -Verb RunAs -Wait", strings.Join(quoted, ","))
	return exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", script).Run()
}
