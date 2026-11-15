package config

import (
	"strings"
	"testing"
)

func TestApplyScript(t *testing.T) {
	out, err := ApplyScript([]byte("mixed-port: 7890\nmode: rule\n"), []byte(`
function main(config) {
  config.mode = "global";
  config["allow-lan"] = true;
  return config;
}
`))
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if !strings.Contains(text, "mode: global") || !strings.Contains(text, "allow-lan: true") {
		t.Fatalf("unexpected script output:\n%s", text)
	}
}

func TestApplyScriptRequiresMain(t *testing.T) {
	if _, err := ApplyScript([]byte("mode: rule\n"), []byte("var x = 1")); err == nil {
		t.Fatal("expected missing main error")
	}
}
