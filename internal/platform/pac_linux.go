//go:build linux

package platform

func (gnomeProxy) EnablePAC(pacURL string) error {
	if _, err := gsettings("set", "org.gnome.system.proxy", "mode", "auto"); err != nil {
		return err
	}
	_, err := gsettings("set", "org.gnome.system.proxy", "autoconfig-url", pacURL)
	return err
}

func (gnomeProxy) DisablePAC() error {
	_, err := gsettings("set", "org.gnome.system.proxy", "mode", "none")
	return err
}

var _ PACProxy = gnomeProxy{}
