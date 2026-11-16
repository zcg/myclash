//go:build windows

package platform

import (
	"errors"
	"golang.org/x/sys/windows"
)

var coreMutex windows.Handle

// AcquireCoreLock keeps a second headless core from starting. The GUI and a
// core use different command lines; the GUI holds no lock while it embeds its
// own kernel, but only one external core may exist.
func AcquireCoreLock() (func(), error) {
	name, _ := windows.UTF16PtrFromString("Local\\myclash-core")
	h, err := windows.CreateMutex(nil, false, name)
	if err != nil {
		return nil, err
	}
	if err == windows.ERROR_ALREADY_EXISTS {
		windows.CloseHandle(h)
		return nil, errors.New("another myclash core is already running")
	}
	coreMutex = h
	return func() {
		if coreMutex != 0 {
			windows.CloseHandle(coreMutex)
			coreMutex = 0
		}
	}, nil
}
