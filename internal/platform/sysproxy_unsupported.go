//go:build !windows && !darwin && !linux

package platform

type unsupportedProxy struct{}

func NewSysProxy() SysProxy { return unsupportedProxy{} }

func (unsupportedProxy) Snapshot() (ProxySnapshot, error)   { return ProxySnapshot{}, ErrUnsupported }
func (unsupportedProxy) Enable(string, int, []string) error { return ErrUnsupported }
func (unsupportedProxy) Disable() error                     { return ErrUnsupported }
func (unsupportedProxy) Enabled() (bool, error)             { return false, ErrUnsupported }
func (unsupportedProxy) Restore(ProxySnapshot) error        { return ErrUnsupported }
