//go:build !windows

package platform

type unsupportedProxy struct{}

func NewSysProxy() SysProxy { return unsupportedProxy{} }

func (unsupportedProxy) Snapshot() (ProxySnapshot, error)   { return ProxySnapshot{}, ErrUnsupported }
func (unsupportedProxy) Enable(string, int, []string) error { return ErrUnsupported }
func (unsupportedProxy) Disable() error                     { return ErrUnsupported }
func (unsupportedProxy) Enabled() (bool, error)             { return false, ErrUnsupported }
func (unsupportedProxy) Restore(ProxySnapshot) error        { return ErrUnsupported }

// IsElevated is false where the app does not use an elevation flow yet.
func IsElevated() bool { return false }

// RelaunchElevated is only implemented on Windows.
func RelaunchElevated() error { return ErrUnsupported }

// ForceQuit terminates this process immediately.
func ForceQuit() { os.Exit(0) }

// ArmKillWatchdog is a no-op where the native tray does not need it.
func ArmKillWatchdog() {}
