//go:build !windows

package platform

// Acrylic is a Windows DWM material. Other platforms have no equivalent
// system-backed acrylic; the Liquid Glass plugin is used instead.
func SetAcrylic(uintptr, float64) error { return nil }
func ClearAcrylic(uintptr) error        { return nil }
