package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type DNSOptions struct {
	Enable       bool
	Listen       string
	EnhancedMode string
	IPv6         bool
	Nameserver   []string
	Fallback     []string
	FakeIPRange  string
}

type AdvancedOptions struct {
	UnifiedDelay    bool
	TCPConcurrent   bool
	FindProcessMode string
	Fingerprint     string
	InterfaceName   string
	RoutingMark     int
}

type TUNOptions struct {
	Enable      bool
	Stack       string
	Device      string
	AutoRoute   bool
	StrictRoute bool
	DNSHijack   bool
	AutoDetect  bool
}

type SnifferOptions struct {
	Enable              bool
	ForceDNSMapping     bool
	OverrideDestination bool
	ParsePureIP         bool
	HTTPPorts           []string
	TLSPorts            []string
	QUICPorts           []string
	SkipDomain          []string
	ForceDomain         []string
	SkipSrc             []string
	SkipDst             []string
}

type Options struct {
	MixedPort      int
	ControllerPort int
	Mode           string
	LogLevel       string
	AllowLAN       bool
	IPv6           bool
	DNS            DNSOptions
	Advanced       AdvancedOptions
	Sniffer        SnifferOptions
	TUN            TUNOptions
}

func (o Options) normalized() Options {
	if o.MixedPort <= 0 || o.MixedPort > 65535 {
		o.MixedPort = 7890
	}
	if o.Mode != "global" && o.Mode != "direct" {
		o.Mode = "rule"
	}
	if o.LogLevel == "" {
		o.LogLevel = "info"
	}
	return o
}

// Runtime builds the config the core actually reads. It is intentionally
// minimal for the first slice: base profile + app-injected fields. Merge and
// script overrides are added in the next slice.
func (s *Store) Runtime(opts Options) ([]byte, error) {
	opts = opts.normalized()
	p, ok := s.Current()
	raw := p.Raw
	if ov, err := os.ReadFile(s.overridePath(p.UID)); err == nil && len(ov) > 0 {
		merged, err := Merge(raw, ov)
		if err != nil {
			return nil, err
		}
		raw = merged
	}
	if !ok || len(p.Raw) == 0 {
		return fallback(opts), nil
	}
	var doc map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", p.Name, err)
	}
	if doc == nil {
		return nil, errors.New("config: profile is not a YAML mapping")
	}
	inject(doc, opts)
	out, err := yaml.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("config: encode runtime: %w", err)
	}
	if script, err := s.ScriptOverride(p.UID); err == nil && len(script) > 0 {
		return ApplyScript(out, script)
	}
	return out, nil
}

func inject(doc map[string]any, opts Options) {
	doc["mixed-port"] = opts.MixedPort
	doc["mode"] = opts.Mode
	doc["log-level"] = opts.LogLevel
	doc["allow-lan"] = opts.AllowLAN
	doc["ipv6"] = opts.IPv6
	doc["unified-delay"] = opts.Advanced.UnifiedDelay
	doc["tcp-concurrent"] = opts.Advanced.TCPConcurrent
	doc["find-process-mode"] = opts.Advanced.FindProcessMode
	if opts.Advanced.Fingerprint != "" {
		doc["global-client-fingerprint"] = opts.Advanced.Fingerprint
	}
	if opts.Advanced.InterfaceName != "" {
		doc["interface-name"] = opts.Advanced.InterfaceName
	}
	if opts.Advanced.RoutingMark > 0 {
		doc["routing-mark"] = opts.Advanced.RoutingMark
	}
	doc["profile"] = map[string]any{
		"store-selected": true,
		"store-fake-ip":  true,
	}
	if _, ok := doc["rules"]; !ok {
		doc["rules"] = []any{"MATCH,DIRECT"}
	}
	if opts.TUN.Enable {
		if opts.TUN.Stack == "" {
			opts.TUN.Stack = "mips"
		}
		tun := map[string]any{
			"enable":                true,
			"stack":                 opts.TUN.Stack,
			"device":                opts.TUN.Device,
			"auto-route":            opts.TUN.AutoRoute,
			"strict-route":          opts.TUN.StrictRoute,
			"auto-detect-interface": opts.TUN.AutoDetect,
		}
		if opts.TUN.DNSHijack {
			tun["dns-hijack"] = []string{"any:53"}
		}
		doc["tun"] = tun
		dns, _ := doc["dns"].(map[string]any)
		if dns == nil {
			dns = map[string]any{}
			doc["dns"] = dns
		}
		dns["enable"] = true
		if _, ok := dns["nameserver"]; !ok {
			dns["nameserver"] = []string{"https://223.5.5.5/dns-query"}
		}
	}
	if opts.Sniffer.Enable {
		doc["sniffer"] = map[string]any{
			"enable":               true,
			"force-dns-mapping":    opts.Sniffer.ForceDNSMapping,
			"override-destination": opts.Sniffer.OverrideDestination,
			"parse-pure-ip":        opts.Sniffer.ParsePureIP,
			"sniff": map[string]any{
				"HTTP": map[string]any{"ports": opts.Sniffer.HTTPPorts},
				"TLS":  map[string]any{"ports": opts.Sniffer.TLSPorts},
				"QUIC": map[string]any{"ports": opts.Sniffer.QUICPorts},
			},
			"skip-domain":      opts.Sniffer.SkipDomain,
			"force-domain":     opts.Sniffer.ForceDomain,
			"skip-src-address": opts.Sniffer.SkipSrc,
			"skip-dst-address": opts.Sniffer.SkipDst,
		}
	}
	if opts.DNS.Enable {
		dns, _ := doc["dns"].(map[string]any)
		if dns == nil {
			dns = map[string]any{}
			doc["dns"] = dns
		}
		dns["enable"] = true
		if opts.DNS.Listen != "" {
			dns["listen"] = opts.DNS.Listen
		}
		dns["enhanced-mode"] = opts.DNS.EnhancedMode
		dns["ipv6"] = opts.DNS.IPv6
		if len(opts.DNS.Nameserver) > 0 {
			dns["nameserver"] = opts.DNS.Nameserver
		}
		if len(opts.DNS.Fallback) > 0 {
			dns["fallback"] = opts.DNS.Fallback
		}
		if opts.DNS.FakeIPRange != "" {
			dns["fake-ip-range"] = opts.DNS.FakeIPRange
		}
	}
}

func fallback(opts Options) []byte {
	return []byte(fmt.Sprintf(`mixed-port: %d
allow-lan: false
mode: %s
log-level: %s
ipv6: false
profile:
  store-selected: true
  store-fake-ip: true
rules:
  - MATCH,DIRECT
`, opts.MixedPort, opts.Mode, opts.LogLevel))
}
