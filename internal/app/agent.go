package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	Addr   string `json:"addr"`
	Secret string `json:"secret"`
	PID    int    `json:"pid"`
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
	if err := writeAgentInfo(agentInfo{Addr: ep.Addr, Secret: ep.Secret, PID: os.Getpid()}); err != nil {
		_ = host.Stop(context.Background())
		return
	}
	a.setupTray()
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
	cmd := exec.Command(exe, "--agent")
	if err := cmd.Start(); err != nil {
		a.setError(err)
		return
	}
	_ = cmd.Process.Release()
	go func() {
		time.Sleep(200 * time.Millisecond)
		os.Exit(0)
	}()
}
