//go:build !windows

package platform

import "errors"

type UWPApp struct {
	Name            string
	PackageFullName string
	Loopback        bool
}

func ListUWP() ([]UWPApp, error) { return nil, errors.New("UWP loopback is Windows only") }

func SetUWPLoopback(string, bool) error { return errors.New("UWP loopback is Windows only") }
