//go:build windows

package platform

import (
	"fmt"
	"os/exec"
	"strings"
)

// StartElevated starts exe with the administrator token without replacing the
// current process. Used when the embedded GUI is normal but the headless core
// must own a TUN adapter.
func StartElevated(exe string, args ...string) error {
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = "'" + strings.ReplaceAll(a, "'", "''") + "'"
	}
	script := fmt.Sprintf("Start-Process -FilePath %q -ArgumentList %s -Verb RunAs", exe, strings.Join(quoted, ","))
	cmd := exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", script)
	return cmd.Start()
}
