//go:build !windows

package platform

import "errors"

func StartElevated(string, ...string) error {
	return errors.New("elevated core start is not implemented on this platform")
}
