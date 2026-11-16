package settings

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Settings is the user's intent, not the live state. Runtime facts (whether the
// proxy is actually on, which profile is running) live in state.yaml instead.
type Settings struct {
	Schema   int      `yaml:"schema"`
	App      App      `yaml:"app"`
	Core     Core     `yaml:"core"`
	UI       UI       `yaml:"ui"`
	Proxy    Proxy    `yaml:"proxy"`
	DNS      DNS      `yaml:"dns"`
	Advanced Advanced `yaml:"advanced"`
	Sniffer  Sniffer  `yaml:"sniffer"`
	Hotkeys  Hotkeys  `yaml:"hotkeys"`
	Logs     Logs     `yaml:"logs"`
	TUN      TUN      `yaml:"tun"`
	Test     Test     `yaml:"test"`
	WebDAV   WebDAV   `yaml:"webdav"`
}

type WebDAV struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Logs struct {
	Retention string `yaml:"retention"` // forever | day | week | month
	MaxSizeMB int    `yaml:"max_size_mb"`
}

type Test struct {
	DelayURL string `yaml:"delay_url"`
	SpeedURL string `yaml:"speed_url"`
}

type Hotkeys struct {
	Show        string `yaml:"show"`
	SystemProxy string `yaml:"system_proxy"`
	TUN         string `yaml:"tun"`
	Mode        string `yaml:"mode"`
	Quit        string `yaml:"quit"`
}

type App struct {
	AutoStartProfile bool   `yaml:"auto_start_profile"` // start the last profile on launch
	CloseToTray      bool   `yaml:"close_to_tray"`
	OpenAtLogin      bool   `yaml:"open_at_login"`
	Language         string `yaml:"language"` // auto | zh-CN | en
	Notifications    bool   `yaml:"notifications"`
}

type Core struct {
	MixedPort      int    `yaml:"mixed_port"`
	ControllerPort int    `yaml:"controller_port"`
	Mode           string `yaml:"mode"` // rule | global | direct
	LogLevel       string `yaml:"log_level"`
	AllowLAN       bool   `yaml:"allow_lan"`
	IPv6           bool   `yaml:"ipv6"`
}

type UI struct {
	Preset          string  `yaml:"preset"` // glass | clear | solid
	Blur            string  `yaml:"blur"`   // derived: glass=deep, others=none
	CardOpacity     float64 `yaml:"card_opacity"`
	Theme           string  `yaml:"theme"` // system | light | dark
	Density         string  `yaml:"density"`
	MonochromeIcons bool    `yaml:"monochrome_icons"`
	Scale           float64 `yaml:"scale"`
}

type Proxy struct {
	SystemEnabled bool     `yaml:"system_enabled"`
	Mode          string   `yaml:"mode"` // manual | pac
	Bypass        []string `yaml:"bypass"`
	LastGroup     string   `yaml:"last_group"`
	LastNode      string   `yaml:"last_node"`
}

type Advanced struct {
	UnifiedDelay    bool   `yaml:"unified_delay"`
	TCPConcurrent   bool   `yaml:"tcp_concurrent"`
	FindProcessMode string `yaml:"find_process_mode"` // off | strict | always
	Fingerprint     string `yaml:"fingerprint"`
	InterfaceName   string `yaml:"interface_name"`
	RoutingMark     int    `yaml:"routing_mark"`
}

const DefaultTUNStack = "mips"

type TUN struct {
	Enable      bool   `yaml:"enable"`
	Stack       string `yaml:"stack"` // system | gvisor | mixed | mips
	Device      string `yaml:"device"`
	AutoRoute   bool   `yaml:"auto_route"`
	StrictRoute bool   `yaml:"strict_route"`
	DNSHijack   bool   `yaml:"dns_hijack"`
	AutoDetect  bool   `yaml:"auto_detect_interface"`
}

type Sniffer struct {
	Enable              bool   `yaml:"enable"`
	ForceDNSMapping     bool   `yaml:"force_dns_mapping"`
	OverrideDestination bool   `yaml:"override_destination"`
	ParsePureIP         bool   `yaml:"parse_pure_ip"`
	HTTPPorts           string `yaml:"http_ports"`
	TLSPorts            string `yaml:"tls_ports"`
	QUICPorts           string `yaml:"quic_ports"`
	SkipDomain          string `yaml:"skip_domain"`
	ForceDomain         string `yaml:"force_domain"`
	SkipSrc             string `yaml:"skip_src"`
	SkipDst             string `yaml:"skip_dst"`
}

type DNS struct {
	Enable       bool     `yaml:"enable"`
	Listen       string   `yaml:"listen"`
	EnhancedMode string   `yaml:"enhanced_mode"` // normal | fake-ip
	IPv6         bool     `yaml:"ipv6"`
	Nameserver   []string `yaml:"nameserver"`
	Fallback     []string `yaml:"fallback"`
	FakeIPRange  string   `yaml:"fake_ip_range"`
}

const Schema = 1

func Default() Settings {
	return Settings{
		Schema: Schema,
		App: App{
			AutoStartProfile: true,
			CloseToTray:      true,
			Language:         "auto",
			Notifications:    true,
		},
		Core: Core{
			MixedPort:      7890,
			ControllerPort: 9090,
			Mode:           "rule",
			LogLevel:       "info",
		},
		UI: UI{
			Preset:      "glass",
			Blur:        "deep",
			CardOpacity: 0.3,
			Theme:       "system",
			Density:     "comfortable",
			Scale:       1.25,
		},
		Proxy: Proxy{
			Mode:   "manual",
			Bypass: []string{"<local>", "127.*", "10.*", "172.16.*", "192.168.*"},
		},
		DNS: DNS{
			EnhancedMode: "fake-ip",
			FakeIPRange:  "198.18.0.1/16",
			Nameserver:   []string{"https://223.5.5.5/dns-query"},
		},
		Advanced: Advanced{
			UnifiedDelay:    true,
			TCPConcurrent:   true,
			FindProcessMode: "always",
			Fingerprint:     "chrome",
		},
		Sniffer: Sniffer{
			HTTPPorts: "80, 8080-8880",
			TLSPorts:  "443, 8443",
			QUICPorts: "443, 8443",
		},
		Hotkeys: Hotkeys{
			Show:        "CmdOrCtrl+Shift+M",
			SystemProxy: "CmdOrCtrl+Shift+P",
			TUN:         "CmdOrCtrl+Shift+T",
			Mode:        "CmdOrCtrl+Shift+R",
			Quit:        "CmdOrCtrl+Shift+Q",
		},
		Logs: Logs{
			Retention: "week",
			MaxSizeMB: 8,
		},
		Test: Test{
			DelayURL: "https://www.gstatic.com/generate_204",
			SpeedURL: "https://speed.cloudflare.com/__down?bytes=4000000",
		},
		TUN: TUN{
			Stack:      DefaultTUNStack,
			Device:     "myclash",
			AutoRoute:  true,
			DNSHijack:  true,
			AutoDetect: true,
		},
	}
}

// Store owns settings.yaml. Reads are cheap; writes are atomic and serialized.
type Store struct {
	mu   sync.RWMutex
	path string
	s    Settings
}

func NewStore(dir string) (*Store, error) {
	if dir == "" {
		return nil, errors.New("settings: empty user data directory")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("settings: create %s: %w", dir, err)
	}
	st := &Store{path: filepath.Join(dir, "settings.yaml"), s: Default()}
	if err := st.load(); err != nil {
		return nil, err
	}
	return st, nil
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return s.saveLocked()
	}
	if err != nil {
		return fmt.Errorf("settings: read %s: %w", s.path, err)
	}
	var v Settings
	if err := yaml.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("settings: parse %s: %w", s.path, err)
	}
	v = normalize(v)
	s.mu.Lock()
	s.s = v
	s.mu.Unlock()

	return nil
}

func (s *Store) Get() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.s
}

// Update applies fn and writes the result. A failed write leaves the in-memory
// value unchanged.
func (s *Store) Update(fn func(*Settings)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := normalize(s.s)
	fn(&next)
	next = normalize(next)
	if err := s.saveLockedValue(next); err != nil {
		return err
	}
	s.s = next
	return nil
}

func (s *Store) saveLocked() error {
	return s.saveLockedValue(s.s)
}

func (s *Store) saveLockedValue(v Settings) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Errorf("settings: encode: %w", err)
	}
	return writeAtomic(s.path, data, 0o600)
}

func writeAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp in %s: %w", dir, err)
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return fmt.Errorf("chmod %s: %w", name, err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write %s: %w", name, err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync %s: %w", name, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close %s: %w", name, err)
	}
	var renameErr error
	for i := 0; i < 5; i++ {
		if renameErr = os.Rename(name, path); renameErr == nil {
			return nil
		}
		time.Sleep(time.Duration(20*(i+1)) * time.Millisecond)
	}
	return fmt.Errorf("rename %s -> %s: %w", name, path, renameErr)
}

func normalize(v Settings) Settings {
	if v.Schema == 0 {
		v.Schema = Schema
	}
	if v.App.Language == "" {
		v.App.Language = "auto"
	}
	if v.Core.MixedPort <= 0 || v.Core.MixedPort > 65535 {
		v.Core.MixedPort = 7890
	}
	if v.Core.ControllerPort <= 0 || v.Core.ControllerPort > 65535 {
		v.Core.ControllerPort = 9090
	}
	if v.Core.Mode != "global" && v.Core.Mode != "direct" {
		v.Core.Mode = "rule"
	}
	switch v.Core.LogLevel {
	case "debug", "info", "warning", "error", "silent":
	default:
		v.Core.LogLevel = "info"
	}
	switch v.TUN.Stack {
	case "system", "gvisor", "mixed", "mips":
	default:
		v.TUN.Stack = DefaultTUNStack
	}
	if v.UI.Preset != "clear" && v.UI.Preset != "solid" {
		v.UI.Preset = "glass"
	}
	if v.UI.Preset == "glass" {
		v.UI.Blur = "deep"
	} else {
		v.UI.Blur = "none"
	}
	if v.UI.CardOpacity < 0 || v.UI.CardOpacity > 1 {
		v.UI.CardOpacity = 0.8
	}
	if v.UI.Theme != "light" && v.UI.Theme != "dark" {
		v.UI.Theme = "system"
	}
	if v.UI.Density != "compact" {
		v.UI.Density = "comfortable"
	}
	if v.UI.Scale < 1 || v.UI.Scale > 1.25 {
		v.UI.Scale = 1
	}
	if v.DNS.EnhancedMode != "normal" {
		v.DNS.EnhancedMode = "fake-ip"
	}
	if v.DNS.FakeIPRange == "" {
		v.DNS.FakeIPRange = "198.18.0.1/16"
	}
	if v.Advanced.FindProcessMode != "off" && v.Advanced.FindProcessMode != "always" {
		v.Advanced.FindProcessMode = "strict"
	}
	if v.Advanced.Fingerprint == "" {
		v.Advanced.Fingerprint = "chrome"
	}
	switch v.Logs.Retention {
	case "forever", "day", "week", "month":
	default:
		v.Logs.Retention = "week"
	}
	if v.Logs.MaxSizeMB <= 0 || v.Logs.MaxSizeMB > 1024 {
		v.Logs.MaxSizeMB = 8
	}
	if v.Test.DelayURL == "" {
		v.Test.DelayURL = "https://www.gstatic.com/generate_204"
	}
	if v.Test.SpeedURL == "" {
		v.Test.SpeedURL = "https://speed.cloudflare.com/__down?bytes=4000000"
	}
	if v.Proxy.Mode != "pac" {
		v.Proxy.Mode = "manual"
	}
	return v
}
