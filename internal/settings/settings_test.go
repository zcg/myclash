package settings

import "testing"

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
