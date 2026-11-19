package config

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRuntimeFallback(t *testing.T) {
	st, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := st.Runtime(Options{MixedPort: 7892, Mode: "rule", LogLevel: "info"})
	if err != nil {
		t.Fatal(err)
	}
	out := string(raw)
	for _, want := range []string{"mixed-port: 7892", "mode: rule", "MATCH,DIRECT"} {
		if !strings.Contains(out, want) {
			t.Fatalf("runtime missing %q:\n%s", want, out)
		}
	}
}

func TestRuntimeInjectsProfile(t *testing.T) {
	st, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	p, err := st.AddLocal("demo", []byte("proxies: []\nrules:\n  - MATCH,DIRECT\n"))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Select(p.UID); err != nil {
		t.Fatal(err)
	}
	raw, err := st.Runtime(Options{MixedPort: 7893, Mode: "global", LogLevel: "debug"})
	if err != nil {
		t.Fatal(err)
	}
	out := string(raw)
	for _, want := range []string{"mixed-port: 7893", "mode: global", "log-level: debug", "store-selected: true"} {
		if !strings.Contains(out, want) {
			t.Fatalf("runtime missing %q:\n%s", want, out)
		}
	}
}

func TestMergeOverride(t *testing.T) {
	base := []byte("dns:\n  enable: true\n  fallback: [1.1.1.1]\nrules:\n  - MATCH,DIRECT\n")
	over := []byte("dns:\n  enhanced-mode: fake-ip\ndelete:\n  - dns.fallback\nprepend-rules:\n  - DOMAIN-SUFFIX,corp.example,DIRECT\n")
	out, err := Merge(base, over)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{"enhanced-mode: fake-ip", "DOMAIN-SUFFIX,corp.example,DIRECT", "MATCH,DIRECT"} {
		if !strings.Contains(s, want) {
			t.Fatalf("merged missing %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "fallback") {
		t.Fatalf("delete did not remove fallback:\n%s", s)
	}
}

func TestAddSubscription(t *testing.T) {
	const body = "proxies: []\nrules:\n  - MATCH,DIRECT\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("missing user agent")
		}
		w.Header().Set("subscription-userinfo", "upload=1; download=2; total=100; expire=1700000000")
		fmt.Fprint(w, body)
	}))
	defer srv.Close()

	st, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	p, err := st.AddSubscription(context.Background(), "demo", srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if p.Source != "subscription" || p.URL != srv.URL {
		t.Fatalf("profile = %+v", p)
	}
	if p.Userinfo == nil || p.Userinfo.Total != 100 {
		t.Fatalf("userinfo = %+v", p.Userinfo)
	}
	if st.Selected() != p.UID {
		t.Fatal("first profile not selected")
	}
	raw, err := st.Runtime(Options{MixedPort: 7899, Mode: "rule", LogLevel: "info"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "mixed-port: 7899") {
		t.Fatalf("runtime missing injected port:\n%s", raw)
	}
}

func TestRuntimeDNSInjection(t *testing.T) {
	st, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.AddLocal("demo", []byte("proxies: []\nrules:\n  - MATCH,DIRECT\n")); err != nil {
		t.Fatal(err)
	}
	raw, err := st.Runtime(Options{
		MixedPort: 7890, Mode: "rule", LogLevel: "info",
		DNS: DNSOptions{
			Enable: true, EnhancedMode: "fake-ip", FakeIPRange: "198.18.0.1/16",
			Nameserver: []string{"https://223.5.5.5/dns-query"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"enhanced-mode: fake-ip", "fake-ip-range: 198.18.0.1/16", "223.5.5.5", "cache-max-size: 1024"} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("runtime missing %q:\n%s", want, raw)
		}
	}
}

func TestRuntimeKeepsExplicitDNSCacheSize(t *testing.T) {
	st, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.AddLocal("demo", []byte("dns:\n  cache-max-size: 8192\nproxies: []\nrules:\n  - MATCH,DIRECT\n")); err != nil {
		t.Fatal(err)
	}
	raw, err := st.Runtime(Options{
		MixedPort: 7890, Mode: "rule", LogLevel: "info",
		DNS: DNSOptions{Enable: true, EnhancedMode: "fake-ip", FakeIPRange: "198.18.0.1/16"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "cache-max-size: 8192") {
		t.Fatalf("explicit cache size was overwritten:\n%s", raw)
	}
}

func TestRuntimeInjectsTUN(t *testing.T) {
	st, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	p, err := st.AddLocal("tun", []byte("proxies: []\nrules:\n  - MATCH,DIRECT\n"))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Select(p.UID); err != nil {
		t.Fatal(err)
	}
	raw, err := st.Runtime(Options{
		MixedPort: 7894, Mode: "rule", LogLevel: "info",
		TUN: TUNOptions{Enable: true, Stack: "mixed", Device: "myclash", AutoRoute: true, DNSHijack: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	out := string(raw)
	for _, want := range []string{"tun:", "stack: mixed", "device: myclash", "auto-route: true", "dns-hijack:", "any:53", "enable: true"} {
		if !strings.Contains(out, want) {
			t.Fatalf("TUN runtime missing %q:\n%s", want, out)
		}
	}
}
