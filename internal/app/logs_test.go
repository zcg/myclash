package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPruneLogFileDropsExpiredLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "core.log")
	old := time.Now().Add(-10 * 24 * time.Hour).Format(time.RFC3339)
	fresh := time.Now().Format(time.RFC3339)
	body := old + "\tinfo\told\n" + fresh + "\tinfo\tfresh\nnot-a-timestamp\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := pruneLogFile(path, time.Now().Add(-7*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "old") {
		t.Fatalf("expired line was kept: %q", got)
	}
	if !strings.Contains(string(got), "fresh") || !strings.Contains(string(got), "not-a-timestamp") {
		t.Fatalf("expected fresh and unparseable lines: %q", got)
	}
}

func TestEnforceLogSizeTruncates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "core.log")
	if err := os.WriteFile(path, []byte("123456"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := enforceLogSize(path, 4); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Size() != 0 {
		t.Fatalf("size = %d, want 0", st.Size())
	}
}
