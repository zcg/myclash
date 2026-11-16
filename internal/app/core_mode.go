package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/egoist/mygo"

	"myclash/internal/config"
	"myclash/internal/core"
	"myclash/internal/platform"
	"myclash/internal/settings"
)

type coreRunner struct {
	mu       sync.Mutex
	settings *settings.Store
	profiles *config.Store
	host     *core.Embedded
	dataDir  string
	control  *http.Server
	token    string
}

func userDataDir() (string, error) {
	return mygo.App.Path(mygo.PathUserData)
}

func runCore() error {
	dir, err := userDataDir()
	if err != nil {
		return err
	}
	a := &coreRunner{dataDir: dir}
	a.settings, err = settings.NewStore(dir)
	if err != nil {
		return err
	}
	a.profiles, err = config.NewStore(dir)
	if err != nil {
		return err
	}
	if err := a.start(); err != nil {
		_ = os.WriteFile(filepath.Join(dir, "core-start.log"), []byte(err.Error()), 0o600)
		return err
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
	a.stop()
	return nil
}

func (a *coreRunner) options(s settings.Settings) config.Options {
	return config.Options{
		MixedPort: s.Core.MixedPort, ControllerPort: s.Core.ControllerPort,
		Mode: s.Core.Mode, LogLevel: s.Core.LogLevel, AllowLAN: s.Core.AllowLAN, IPv6: s.Core.IPv6,
		DNS:      config.DNSOptions{Enable: s.DNS.Enable, Listen: s.DNS.Listen, EnhancedMode: s.DNS.EnhancedMode, IPv6: s.DNS.IPv6, Nameserver: s.DNS.Nameserver, Fallback: s.DNS.Fallback, FakeIPRange: s.DNS.FakeIPRange},
		Advanced: config.AdvancedOptions{UnifiedDelay: s.Advanced.UnifiedDelay, TCPConcurrent: s.Advanced.TCPConcurrent, FindProcessMode: s.Advanced.FindProcessMode, Fingerprint: s.Advanced.Fingerprint, InterfaceName: s.Advanced.InterfaceName, RoutingMark: s.Advanced.RoutingMark},
		Sniffer:  config.SnifferOptions{Enable: s.Sniffer.Enable, ForceDNSMapping: s.Sniffer.ForceDNSMapping, OverrideDestination: s.Sniffer.OverrideDestination, ParsePureIP: s.Sniffer.ParsePureIP, HTTPPorts: splitList(s.Sniffer.HTTPPorts), TLSPorts: splitList(s.Sniffer.TLSPorts), QUICPorts: splitList(s.Sniffer.QUICPorts), SkipDomain: splitList(s.Sniffer.SkipDomain), ForceDomain: splitList(s.Sniffer.ForceDomain), SkipSrc: splitList(s.Sniffer.SkipSrc), SkipDst: splitList(s.Sniffer.SkipDst)},
		TUN:      config.TUNOptions{Enable: s.TUN.Enable, Stack: s.TUN.Stack, Device: s.TUN.Device, AutoRoute: s.TUN.AutoRoute, StrictRoute: s.TUN.StrictRoute, DNSHijack: s.TUN.DNSHijack, AutoDetect: s.TUN.AutoDetect},
	}
}

func (a *coreRunner) start() error {
	s := a.settings.Get()
	if s.TUN.Enable && !platform.IsElevated() {
		return errors.New("core: TUN needs administrator")
	}
	raw, err := a.profiles.Runtime(a.options(s))
	if err != nil {
		return err
	}
	host, err := core.NewEmbedded(core.Home(a.dataDir), s.Core.ControllerPort)
	if err != nil {
		return err
	}
	if err := host.Start(context.Background(), raw); err != nil {
		return err
	}
	a.host = host
	ep := host.Endpoint()
	control, token, err := a.startControl()
	if err != nil {
		_ = host.Stop(context.Background())
		return err
	}
	if err := writeAgentInfo(agentInfo{Addr: ep.Addr, Secret: ep.Secret, PID: os.Getpid(), ControlAddr: control, ControlToken: token}); err != nil {
		_ = host.Stop(context.Background())
		return err
	}
	return nil
}

func (a *coreRunner) reload() error {
	s := a.settings.Get()
	raw, err := a.profiles.Runtime(a.options(s))
	if err != nil {
		return err
	}
	return a.host.Reload(context.Background(), raw)
}

func (a *coreRunner) startControl() (string, string, error) {
	tb := make([]byte, 24)
	if _, err := rand.Read(tb); err != nil {
		return "", "", err
	}
	token := hex.EncodeToString(tb)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", "", err
	}
	mux := http.NewServeMux()
	auth := func(w http.ResponseWriter, r *http.Request) bool {
		if r.Header.Get("Authorization") != "Bearer "+token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return false
		}
		return true
	}
	mux.HandleFunc("/restart", func(w http.ResponseWriter, r *http.Request) {
		if !auth(w, r) {
			return
		}
		if err := a.reload(); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/tun", func(w http.ResponseWriter, r *http.Request) {
		if !auth(w, r) {
			return
		}
		enable := r.URL.Query().Get("enabled") == "1"
		if enable && !platform.IsElevated() {
			http.Error(w, "TUN needs administrator", http.StatusForbidden)
			return
		}
		if err := a.settings.Update(func(st *settings.Settings) { st.TUN.Enable = enable }); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if err := a.reload(); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/sysproxy", func(w http.ResponseWriter, r *http.Request) {
		if !auth(w, r) {
			return
		}
		enable := r.URL.Query().Get("enabled") == "1"
		s := a.settings.Get()
		p := platform.NewSysProxy()
		var err error
		if enable {
			err = p.Enable("127.0.0.1", s.Core.MixedPort, s.Proxy.Bypass)
		} else {
			err = p.Disable()
		}
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/quit", func(w http.ResponseWriter, r *http.Request) {
		if !auth(w, r) {
			return
		}
		w.WriteHeader(http.StatusNoContent)
		go func() { a.stop(); os.Exit(0) }()
	})
	srv := &http.Server{Handler: mux}
	a.control = srv
	a.token = token
	go func() { _ = srv.Serve(ln) }()
	return ln.Addr().String(), token, nil
}

func (a *coreRunner) stop() {
	if a.host != nil {
		_ = a.host.Stop(context.Background())
		a.host = nil
	}
	if a.control != nil {
		_ = a.control.Close()
		a.control = nil
	}
	_ = os.Remove(agentPath())
}
