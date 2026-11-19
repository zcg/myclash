package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreDefaultsAndNormalize(t *testing.T) {
	dir := t.TempDir()
	st, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	s := st.Get()
	if s.Core.MixedPort != 7890 || s.UI.CardOpacity != 0.3 {
		t.Fatalf("unexpected defaults: %+v", s)
	}
	if err := st.Update(func(s *Settings) {
		s.Core.MixedPort = 70000
		s.Core.Mode = "nonsense"
		s.UI.CardOpacity = 0.1
		s.UI.Blur = "nonsense"
	}); err != nil {
		t.Fatal(err)
	}
	s = st.Get()
	if s.Core.MixedPort != 7890 || s.Core.Mode != "rule" || s.UI.CardOpacity != 0.1 || s.UI.Blur != "deep" {
		t.Fatalf("not normalized: %+v", s)
	}
}

func TestAutoCheckUpdatesDefaultsAndPersistence(t *testing.T) {
	dir := t.TempDir()
	st, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !st.AutoCheckUpdates() {
		t.Fatal("auto check should be enabled by default")
	}
	if err := st.SetAutoCheckUpdates(false); err != nil {
		t.Fatal(err)
	}
	if st.AutoCheckUpdates() {
		t.Fatal("auto check should be disabled after cancel")
	}
	again, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if again.AutoCheckUpdates() {
		t.Fatal("auto check choice should persist across restarts")
	}
}

func TestAutoCheckUpdatesLegacyConfig(t *testing.T) {
	dir := t.TempDir()
	// A config written before the field existed: no auto_check_updates key.
	legacy := "schema: 1\napp:\n  auto_start_profile: true\ncore:\n  mixed_port: 7890\n"
	if err := os.WriteFile(filepath.Join(dir, "settings.yaml"), []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !st.AutoCheckUpdates() {
		t.Fatal("legacy config should default to auto check enabled")
	}
}

func TestStoreReload(t *testing.T) {
	dir := t.TempDir()
	st, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Update(func(s *Settings) { s.Core.MixedPort = 7891 }); err != nil {
		t.Fatal(err)
	}
	again, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := again.Get().Core.MixedPort; got != 7891 {
		t.Fatalf("mixed port = %d, want 7891", got)
	}
}
