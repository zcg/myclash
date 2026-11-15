package app

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"myclash/internal/settings"
)

// startPAC serves a PAC document from loopback. The URL stays stable while the
// app runs; the mixed port is read when the document is requested.
func (a *App) startPAC() (string, error) {
	if a.pacServer != nil && a.pacURL != "" {
		return a.pacURL, nil
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/proxy.pac", func(w http.ResponseWriter, r *http.Request) {
		s := a.settings.Get()
		port := s.Core.MixedPort
		bypass := strings.Join(s.Proxy.Bypass, "|")
		w.Header().Set("Content-Type", "application/x-ns-proxy-autoconfig")
		fmt.Fprintf(w, `function FindProxyForURL(url, host) {
  if (isPlainHostName(host) || host == "localhost" || shExpMatch(host, "127.*")) return "DIRECT";
  var bypass = %q.split("|");
  for (var i = 0; i < bypass.length; i++) {
    if (bypass[i] && shExpMatch(host, bypass[i])) return "DIRECT";
  }
  return "PROXY 127.0.0.1:%d; DIRECT";
}
`, bypass, port)
	})
	srv := &http.Server{Handler: mux}
	a.pacServer = srv
	a.pacURL = "http://" + ln.Addr().String() + "/proxy.pac"
	go func() { _ = srv.Serve(ln) }()
	return a.pacURL, nil
}

func (a *App) stopPAC() {
	if a.pacServer != nil {
		_ = a.pacServer.Close()
		a.pacServer = nil
		a.pacURL = ""
	}
}

func (a *App) applyProxyMode() {
	if a.proxyModeIdx != 0 && a.proxyModeIdx != 1 {
		a.proxyModeIdx = 0
	}
	mode := "manual"
	if a.proxyModeIdx == 1 {
		mode = "pac"
	}
	_ = a.settings.Update(func(s *settings.Settings) { s.Proxy.Mode = mode })
	if a.systemProxy {
		a.toggleSystemProxy(false)
		a.toggleSystemProxy(true)
	}
	a.invalidate()
}
