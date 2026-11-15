//go:build windows

package platform

import (
	"strconv"

	"golang.org/x/sys/windows/registry"
)

// AcrylicSupported reports whether the DWM system-backdrop material is
// available. Windows 11 22H2 (build 22621) is the first release with
// DWMWA_SYSTEMBACKDROP_TYPE.
func AcrylicSupported() bool {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	v, _, err := k.GetStringValue("CurrentBuildNumber")
	if err != nil {
		return false
	}
	build, err := strconv.Atoi(v)
	return err == nil && build >= 22621
}
