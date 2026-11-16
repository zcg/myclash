package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/egoist/mygo"

	"myclash/internal/config"
	"myclash/internal/core"
	"myclash/internal/platform"
	"myclash/internal/settings"
)

type agentInfo struct {
	Addr         string `json:"addr"`
	Secret       string `json:"secret"`
	PID          int    `json:"pid"`
	ControlAddr  string `json:"control_addr"`
	ControlToken string `json:"control_token"`
}

func agentPath() string {
	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "agent.json")
}

func readAgentInfo() (agentInfo, error) {
	path := agentPath()
	if path == "" {
		return agentInfo{}, errors.New("agent: no user data path")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return agentInfo{}, err
	}
	var info agentInfo
	if err := json.Unmarshal(b, &info); err != nil {
		return agentInfo{}, err
	}
	if info.Addr == "" || info.Secret == "" {
		return agentInfo{}, errors.New("agent: incomplete endpoint")
	}
	return info, nil
}

func writeAgentInfo(info agentInfo) error {
	path := agentPath()
	if path == "" {
		return errors.New("agent: no user data path")
	}
	b, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

type agentApp struct {
	dataDir  string
	settings *settings.Store
	profiles *config.Store
	host     *core.Embedded
	tray     *mygo.Tray
	control  *http.Server
	token    string
}

func runAgent() error {
	a := &agentApp{}
	mygo.App.WhenReady(a.ready)
	return mygo.App.Run()
}

func (a *agentApp) ready() {
	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return
	}
	a.dataDir = dir
	a.settings, err = settings.NewStore(dir)
	if err != nil {
		return
	}
	a.profiles, err = config.NewStore(dir)
	if err != nil {
		return
	}
	s := a.settings.Get()
	if s.TUN.Enable && !platform.IsElevated() {
		// An Agent with TUN needs the same elevation contract as the GUI.
		_ = platform.RelaunchElevated()
		return
	}
	raw, err := a.profiles.Runtime(config.Options{
		MixedPort:      s.Core.MixedPort,
		ControllerPort: s.Core.ControllerPort,
		Mode:           s.Core.Mode,
		LogLevel:       s.Core.LogLevel,
		AllowLAN:       s.Core.AllowLAN,
		IPv6:           s.Core.IPv6,
		DNS: config.DNSOptions{
			Enable: s.DNS.Enable, Listen: s.DNS.Listen, EnhancedMode: s.DNS.EnhancedMode,
			IPv6: s.DNS.IPv6, Nameserver: s.DNS.Nameserver, Fallback: s.DNS.Fallback, FakeIPRange: s.DNS.FakeIPRange,
		},
		Advanced: config.AdvancedOptions{
			UnifiedDelay: s.Advanced.UnifiedDelay, TCPConcurrent: s.Advanced.TCPConcurrent,
			FindProcessMode: s.Advanced.FindProcessMode, Fingerprint: s.Advanced.Fingerprint,
			InterfaceName: s.Advanced.InterfaceName, RoutingMark: s.Advanced.RoutingMark,
		},
		Sniffer: config.SnifferOptions{
			Enable: s.Sniffer.Enable, ForceDNSMapping: s.Sniffer.ForceDNSMapping,
			OverrideDestination: s.Sniffer.OverrideDestination, ParsePureIP: s.Sniffer.ParsePureIP,
			HTTPPorts: splitList(s.Sniffer.HTTPPorts), TLSPorts: splitList(s.Sniffer.TLSPorts),
			QUICPorts: splitList(s.Sniffer.QUICPorts), SkipDomain: splitList(s.Sniffer.SkipDomain),
			ForceDomain: splitList(s.Sniffer.ForceDomain), SkipSrc: splitList(s.Sniffer.SkipSrc),
			SkipDst: splitList(s.Sniffer.SkipDst),
		},
		TUN: config.TUNOptions{
			Enable: s.TUN.Enable, Stack: s.TUN.Stack, Device: s.TUN.Device,
			AutoRoute: s.TUN.AutoRoute, StrictRoute: s.TUN.StrictRoute,
			DNSHijack: s.TUN.DNSHijack, AutoDetect: s.TUN.AutoDetect,
		},
	})
	if err != nil {
		return
	}
	host, err := core.NewEmbedded(core.Home(dir), s.Core.ControllerPort)
	if err != nil {
		return
	}
	if err := host.Start(context.Background(), raw); err != nil {
		return
	}
	a.host = host
	ep := host.Endpoint()
	controlAddr, token, err := a.startControl()
	if err != nil {
		_ = host.Stop(context.Background())
		return
	}
	if err := writeAgentInfo(agentInfo{Addr: ep.Addr, Secret: ep.Secret, PID: os.Getpid(), ControlAddr: controlAddr, ControlToken: token}); err != nil {
		_ = host.Stop(context.Background())
		return
	}
	a.setupTray()
}

func (a *agentApp) runtimeOptions(s settings.Settings) config.Options {
	return config.Options{
		MixedPort: s.Core.MixedPort, ControllerPort: s.Core.ControllerPort,
		Mode: s.Core.Mode, LogLevel: s.Core.LogLevel, AllowLAN: s.Core.AllowLAN, IPv6: s.Core.IPv6,
		DNS: config.DNSOptions{
			Enable: s.DNS.Enable, Listen: s.DNS.Listen, EnhancedMode: s.DNS.EnhancedMode,
			IPv6: s.DNS.IPv6, Nameserver: s.DNS.Nameserver, Fallback: s.DNS.Fallback, FakeIPRange: s.DNS.FakeIPRange,
		},
		Advanced: config.AdvancedOptions{
			UnifiedDelay: s.Advanced.UnifiedDelay, TCPConcurrent: s.Advanced.TCPConcurrent,
			FindProcessMode: s.Advanced.FindProcessMode, Fingerprint: s.Advanced.Fingerprint,
			InterfaceName: s.Advanced.InterfaceName, RoutingMark: s.Advanced.RoutingMark,
		},
		Sniffer: config.SnifferOptions{
			Enable: s.Sniffer.Enable, ForceDNSMapping: s.Sniffer.ForceDNSMapping,
			OverrideDestination: s.Sniffer.OverrideDestination, ParsePureIP: s.Sniffer.ParsePureIP,
			HTTPPorts: splitList(s.Sniffer.HTTPPorts), TLSPorts: splitList(s.Sniffer.TLSPorts),
			QUICPorts: splitList(s.Sniffer.QUICPorts), SkipDomain: splitList(s.Sniffer.SkipDomain),
			ForceDomain: splitList(s.Sniffer.ForceDomain), SkipSrc: splitList(s.Sniffer.SkipSrc),
			SkipDst: splitList(s.Sniffer.SkipDst),
		},
		TUN: config.TUNOptions{
			Enable: s.TUN.Enable, Stack: s.TUN.Stack, Device: s.TUN.Device,
			AutoRoute: s.TUN.AutoRoute, StrictRoute: s.TUN.StrictRoute,
			DNSHijack: s.TUN.DNSHijack, AutoDetect: s.TUN.AutoDetect,
		},
	}
}

func (a *agentApp) reloadCore() error {
	s := a.settings.Get()
	raw, err := a.profiles.Runtime(a.runtimeOptions(s))
	if err != nil {
		return err
	}
	if a.host == nil {
		return errors.New("agent: core not running")
	}
	return a.host.Reload(context.Background(), raw)
}

func (a *agentApp) setupTray() {
	tray, err := mygo.NewTray(mygo.TrayOptions{
		ID: "agent", Icon: trayIcon, ToolTip: "myclash Agent",
		Menu: mygo.NewMenu([]*mygo.MenuItem{
			{Label: "显示窗口", Click: func(*mygo.MenuItem, *mygo.Window) {
				exe, err := os.Executable()
				if err != nil {
					return
				}
				cmd := exec.Command(exe, "--attach")
				_ = cmd.Start()
			}},
			{Type: mygo.MenuItemSeparator},
			{Label: "重启内核", Click: func(*mygo.MenuItem, *mygo.Window) {
				_ = a.reloadCore()
			}},
			{Label: "退出 Agent", Click: func(*mygo.MenuItem, *mygo.Window) {
				a.quit()
			}},
		}),
	})
	if err != nil {
		return
	}
	a.tray = tray
}

func (a *agentApp) startControl() (string, string, error) {
	tokenBytes := make([]byte, 24)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", "", err
	}
	token := hex.EncodeToString(tokenBytes)
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
	mux.HandleFunc("/sysproxy", func(w http.ResponseWriter, r *http.Request) {
		if !auth(w, r) {
			return
		}
		enable := r.URL.Query().Get("enabled") == "1"
		s := a.settings.Get()
		proxy := platform.NewSysProxy()
		var err error
		if enable {
			err = proxy.Enable("127.0.0.1", s.Core.MixedPort, s.Proxy.Bypass)
		} else {
			err = proxy.Disable()
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = a.settings.Update(func(s *settings.Settings) { s.Proxy.SystemEnabled = enable })
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/tun", func(w http.ResponseWriter, r *http.Request) {
		if !auth(w, r) {
			return
		}
		enable := r.URL.Query().Get("enabled") == "1"
		if enable && !platform.IsElevated() {
			http.Error(w, "Agent 不是管理员权限，无法启用 TUN", http.StatusForbidden)
			return
		}
		old := a.settings.Get().TUN.Enable
		if err := a.settings.Update(func(s *settings.Settings) { s.TUN.Enable = enable }); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := a.reloadCore(); err != nil {
			_ = a.settings.Update(func(s *settings.Settings) { s.TUN.Enable = old })
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/restart", func(w http.ResponseWriter, r *http.Request) {
		if !auth(w, r) {
			return
		}
		if err := a.reloadCore(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/quit", func(w http.ResponseWriter, r *http.Request) {
		if !auth(w, r) {
			return
		}
		w.WriteHeader(http.StatusNoContent)
		a.quit()
	})
	srv := &http.Server{Handler: mux}
	a.control = srv
	a.token = token
	go func() { _ = srv.Serve(ln) }()
	return ln.Addr().String(), token, nil
}

func (a *agentApp) quit() {
	if a.tray != nil {
		a.tray.Destroy()
		a.tray = nil
	}
	if a.host != nil {
		_ = a.host.Stop(context.Background())
	}
	_ = os.Remove(agentPath())
	go func() {
		time.Sleep(100 * time.Millisecond)
		os.Exit(0)
	}()
}

func agentAlive() bool {
	info, err := readAgentInfo()
	if err != nil {
		return false
	}
	if info.PID > 0 {
		if p, err := os.FindProcess(info.PID); err != nil || p == nil {
			return false
		}
	}
	return true
}

func (a *App) attachAgent() error {
	info, err := readAgentInfo()
	if err != nil {
		return err
	}
	client := core.NewClient(core.Endpoint{Addr: info.Addr, Secret: info.Secret})
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	version, err := client.Version(ctx)
	if err != nil {
		return fmt.Errorf("Agent 不可达: %w", err)
	}
	proxies, err := client.Proxies(ctx)
	if err != nil {
		return err
	}
	rules, _ := client.Rules(ctx)
	a.mu.Lock()
	a.client = client
	a.attach = true
	a.agentPID = info.PID
	a.agentControl = info.ControlAddr
	a.agentToken = info.ControlToken
	a.mu.Unlock()
	a.state.SetStarted(true)
	a.state.SetVersion(version, a.settings.Get().Core.Mode)
	a.state.SetProxies(proxies)
	a.state.SetRules(rules)
	a.startStreams(client)
	return nil
}

// switchToAgent stops the in-process core and lets the headless Agent take
// over. The GUI exits; reopening goes through attachAgent and never keeps a
// second UI resident in the background.
func (a *App) switchToAgent() {
	a.forceQuit = true
	if a.host != nil {
		a.stopCore()
	}
	exe, err := os.Executable()
	if err != nil {
		a.setError(err)
		return
	}
	args := "--core"
	_ = os.Remove(agentPath())
	if a.agentTray {
		args = "--agent"
	}
	cmd := exec.Command(exe, args)
	if err := cmd.Start(); err != nil {
		a.setError(err)
		return
	}
	_ = cmd.Process.Release()
	for i := 0; i < 50; i++ {
		if agentAlive() {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !agentAlive() {
		a.forceQuit = false
		a.startCore()
		a.setError(errors.New("轻量内核启动失败，已恢复本地内核"))
		return
	}
	os.Exit(0)
}

func (a *App) agentRequest(path string) error {
	if a.agentControl == "" || a.agentToken == "" {
		return errors.New("Agent 控制端点不可用")
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "http://"+a.agentControl+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+a.agentToken)
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("Agent: %s: %s", resp.Status, string(b))
	}
	return nil
}
