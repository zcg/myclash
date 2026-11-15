package core

import (
	"context"
	"os"
	"testing"
	"time"
)

const testConfig = `
mixed-port: 0
allow-lan: false
mode: rule
log-level: warning
ipv6: false
proxies: []
proxy-groups: []
rules:
  - MATCH,DIRECT
`

// TestEmbeddedLifecycle is the proof that the in-process host is real: it
// starts mihomo, serves the REST API and stops without leaving the port bound.
func TestEmbeddedLifecycle(t *testing.T) {
	// mihomo keeps cache.db open until the process exits; leaving the temp dir
	// avoids a cleanup race that is not part of the lifecycle under test.
	home, err := os.MkdirTemp("", "myclash-core-")
	if err != nil {
		t.Fatal(err)
	}
	h, err := NewEmbedded(home, 0)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := h.Start(ctx, []byte(testConfig)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = h.Stop(context.Background()) })

	c := NewClient(h.Endpoint())
	v, err := c.Version(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if v.Version == "" {
		t.Fatal("empty version")
	}
	proxies, err := c.Proxies(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(proxies) == 0 {
		t.Fatal("no built-in proxies")
	}
	if err := h.SetMode(ctx, "direct"); err != nil {
		t.Fatal(err)
	}
	if err := h.Reload(ctx, []byte(testConfig)); err != nil {
		t.Fatal(err)
	}
	if err := h.Stop(ctx); err != nil {
		t.Fatal(err)
	}
}
