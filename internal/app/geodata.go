package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// installGeoData seeds the core home with the GeoIP and GeoSite databases
// bundled next to the executable. Without them mihomo downloads the files on
// first start; that download has no deadline and can leave "starting the
// core" hanging on a blocked or slow network.
func installGeoData(userData string) error {
	coreDir := filepath.Join(userData, "core")
	if err := os.MkdirAll(coreDir, 0o700); err != nil {
		return err
	}
	exe, _ := os.Executable()
	dirs := []string{filepath.Dir(exe), filepath.Join(userData, "geo"), "resources", "."}
	for _, name := range []string{"geoip.metadb", "GeoSite.dat"} {
		dest := filepath.Join(coreDir, name)
		if st, err := os.Stat(dest); err == nil && st.Size() > 0 {
			continue
		}
		var src string
		for _, dir := range dirs {
			candidate := filepath.Join(dir, name)
			if st, err := os.Stat(candidate); err == nil && st.Size() > 0 {
				src = candidate
				break
			}
		}
		if src == "" {
			continue
		}
		if err := copyFile(src, dest); err != nil {
			return fmt.Errorf("install %s: %w", name, err)
		}
	}
	return nil
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp, err := os.CreateTemp(filepath.Dir(dest), ".geo-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := io.Copy(tmp, in); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, dest)
}
