//go:build !windows

package platform

// AcquireCoreLock is a no-op on platforms without a named mutex backend.
func AcquireCoreLock() (func(), error) { return func() {}, nil }
