//go:build windows

package platform

import (
	"errors"
	"syscall"
	"unsafe"
)

const (
	accDisabled          = 0
	accEnableAcrylicBlur = 4
	accFlagDrawAll       = 2
)

type accentPolicy struct {
	State         uint32
	Flags         uint32
	GradientColor uint32
	AnimationID   uint32
}

type winCompositionAttributeData struct {
	Attribute  uint32
	Data       uintptr
	SizeOfData uintptr
}

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	procSetWindowCompositionAttr = user32.NewProc("SetWindowCompositionAttribute")
)

// SetAcrylic applies Windows' legacy ACCENT_ENABLE_ACRYLICBLURBEHIND to the
// HWND. The alpha byte in GradientColor is the one intensity knob Windows
// exposes for this material; opacity 0 means fully transparent, 1 fully
// opaque. It is intentionally separate from MyGo's DWM Mica/Acrylic backdrop.
func SetAcrylic(hwnd uintptr, opacity float64) error {
	if hwnd == 0 {
		return errors.New("acrylic: empty window handle")
	}
	if opacity < 0 {
		opacity = 0
	}
	if opacity > 1 {
		opacity = 1
	}
	alpha := uint32(opacity*255) << 24
	accent := accentPolicy{
		State:         accEnableAcrylicBlur,
		Flags:         accFlagDrawAll,
		GradientColor: alpha | 0x000000,
	}
	data := winCompositionAttributeData{
		Attribute:  19, // WCA_ACCENT_POLICY
		Data:       uintptr(unsafe.Pointer(&accent)),
		SizeOfData: unsafe.Sizeof(accent),
	}
	r, _, err := procSetWindowCompositionAttr.Call(hwnd, uintptr(unsafe.Pointer(&data)))
	if r == 0 {
		return err
	}
	return nil
}

func ClearAcrylic(hwnd uintptr) error {
	if hwnd == 0 {
		return nil
	}
	accent := accentPolicy{State: accDisabled}
	data := winCompositionAttributeData{
		Attribute:  19,
		Data:       uintptr(unsafe.Pointer(&accent)),
		SizeOfData: unsafe.Sizeof(accent),
	}
	r, _, err := procSetWindowCompositionAttr.Call(hwnd, uintptr(unsafe.Pointer(&data)))
	if r == 0 {
		return err
	}
	return nil
}
