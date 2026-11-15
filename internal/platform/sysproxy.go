package platform

import "errors"

// ErrUnsupported marks a platform where a system feature is not implemented.
var ErrUnsupported = errors.New("platform: unsupported")

// ProxySnapshot is the previous system proxy configuration, kept so Disable and
// Restore can put the user's settings back.
type ProxySnapshot struct {
	Enable     uint32
	Server     string
	Override   string
	AutoConfig string
}

// SysProxy is the platform's system proxy. On Windows it writes the current
// user's WinINet settings; other platforms report ErrUnsupported for now.
type SysProxy interface {
	Snapshot() (ProxySnapshot, error)
	Enable(host string, port int, bypass []string) error
	Disable() error
	Enabled() (bool, error)
	Restore(ProxySnapshot) error
}
