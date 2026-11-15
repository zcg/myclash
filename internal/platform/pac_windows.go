//go:build windows

package platform

import "fmt"

func (windowsProxy) EnablePAC(pacURL string) error {
	k, err := windowsProxy{}.open()
	if err != nil {
		return err
	}
	defer k.Close()
	if err := k.SetStringValue("AutoConfigURL", pacURL); err != nil {
		return fmt.Errorf("set AutoConfigURL: %w", err)
	}
	if err := k.SetDWordValue("ProxyEnable", 0); err != nil {
		return fmt.Errorf("disable manual proxy: %w", err)
	}
	return nil
}

func (windowsProxy) DisablePAC() error {
	k, err := windowsProxy{}.open()
	if err != nil {
		return err
	}
	defer k.Close()
	_ = k.DeleteValue("AutoConfigURL")
	return nil
}

var _ PACProxy = windowsProxy{}
