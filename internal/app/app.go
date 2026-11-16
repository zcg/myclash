package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"myclash/internal/config"
	"myclash/internal/core"
	"myclash/internal/i18n"
	"myclash/internal/platform"
	"myclash/internal/settings"
	"myclash/internal/ui/acrylic"
	"myclash/internal/ui/format"
)

type connRate struct {
	up, down         int64
	at               time.Time
	upRate, downRate float64
}

type siteResult struct {
	Name string
	OK   bool
	MS   int64
	Err  string
}

// App is the composition root: it owns the window, the core host, the profile
// store and every page's shared state. Pages are methods on *App, so they can
// read state without importing this package.
type App struct {
	dataDir string

	settings *settings.Store
	profiles *config.Store
	state    *core.State
	dark     bool

	host   *core.Embedded
	client *core.Client

	sysproxy      platform.SysProxy
	systemProxy   bool
	proxySnapshot platform.ProxySnapshot
	proxyOwned    bool
	proxyModeIdx  int
	pacServer     *http.Server
	pacURL        string

	win    *mygo.Window
	tray   *mygo.Tray
	router *ui.Router
	side   string

	ctx    context.Context
	cancel context.CancelFunc

	mu     sync.Mutex
	busy   bool
	errMsg string
	status string

	subURL           string
	renameUID        string
	renameText       string
	mixed            string
	controller       string
	modeIdx          int
	logLevelIdx      int
	themeIdx         int
	presetIdx        int
	densityIdx       int
	opacity          float64
	scale            float64
	monochromeIcons  bool
	languageIdx      int
	hotkeyShow       string
	hotkeyProxy      string
	hotkeyTUN        string
	hotkeyMode       string
	hotkeyQuit       string
	settingsSection  string
	bypass           string
	updateBusy       bool
	updateStatus     string
	updateVersion    string
	updateNotes      string
	updateProgress   float64
	autoStart        bool
	notifications    bool
	elevatedAgent    bool
	openAtLogin      bool
	attach           bool
	agentPID         int
	agentControl     string
	agentToken       string
	runModeIdx       int
	closeTray        bool
	forceQuit        bool
	logFile          *os.File
	logMu            sync.Mutex
	paintMu          sync.Mutex
	lastTrafficPaint time.Time
	lastConnPaint    time.Time

	connState         ui.ListState
	ruleState         ui.ListState
	logState          ui.ListState
	connSearch        string
	ruleSearch        string
	logSearch         string
	logLevelFilter    int
	logRetentionIdx   int
	logMaxMB          int
	logMaxText        string
	webdavURL         string
	webdavUser        string
	webdavPass        string
	testDelayURL      string
	testSpeedURL      string
	siteResults       []siteResult
	siteTesting       bool
	paletteOpen       bool
	paletteQuery      string
	paletteIndex      int
	floating          *mygo.Window
	trafficPath       string
	trafficHistory    []trafficPoint
	lastTrafficSave   time.Time
	lastTrafficFlush  time.Time
	trafficPending    []trafficPoint
	trafficState      ui.ListState
	uwpApps           []platform.UWPApp
	uwpSearch         string
	uwpLoaded         bool
	uwpBusy           bool
	uwpList           ui.ListState
	proxySearch       string
	proxySortIdx      int
	proxyExpanded     map[string]bool
	proxyColsIdx      int
	proxyLastSearch   string
	proxyGroup        string
	proxyNode         string
	lastProxyNow      map[string]string
	connRates         map[string]connRate
	connSortIdx       int
	scrollProxyGroup  string
	logsPaused        bool
	connPaused        bool
	frozenConns       []core.Connection
	frozenLogs        []core.LogEntry
	override          *editor.State
	overrideUID       string
	overrideMode      int
	overrideScript    *editor.State
	overrideScriptUID string
	baseEdit          *editor.State
	baseUID           string
	dnsName           string
	dnsEnable         bool
	dnsListen         string
	dnsNameservers    string
	dnsFallback       string
	dnsFakeIPRange    string
	dnsModeIdx        int
	unifiedDelay      bool
	tcpConcurrent     bool
	findProcessIdx    int
	fingerprint       string
	iface             string
	routingMark       string
	sniffEnable       bool
	sniffForceDNS     bool
	sniffOverride     bool
	sniffPureIP       bool
	sniffHTTP         string
	sniffTLS          string
	sniffQUIC         string
	sniffSkip         string
	sniffForce        string
	sniffSkipSrc      string
	sniffSkipDst      string
	tunEnable         bool
	tunStackIdx       int
	tunDevice         string
	tunAutoRoute      bool
	tunStrictRoute    bool
	tunDNSHijack      bool
	tunAutoDetect     bool
	tunProxyWasOn     bool
	delays            map[string]int
	delayErrs         map[string]string
	speeds            map[string]float64
	speedErrs         map[string]string
	proxyProviders    []core.Provider
	ruleProviders     []core.Provider
	proxyProvState    ui.ListState
	ruleProvState     ui.ListState
}

// Run starts the application. It returns immediately when another instance is
// already running (that instance gets activated instead).
func Run() error {
	if hasArg("--agent") {
		return runAgent()
	}
	attach := hasArg("--attach") || agentAlive()
	if !attach {
		if !mygo.App.RequestSingleInstanceLock() {
			return nil
		}
	}
	a := &App{
		state:  core.NewState(),
		router: ui.NewRouter("/overview"),
		side:   "overview",
		ctx:    context.Background(),
		attach: attach,
	}
	a.cancel = func() {}
	mygo.App.WhenReady(a.ready)
	return mygo.App.Run()
}

func (a *App) ready() {
	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		a.fatal(err)
		return
	}
	a.dataDir = dir
	a.loadTrafficHistory(dir)
	a.settings, err = settings.NewStore(dir)
	if err != nil {
		a.fatal(err)
		return
	}
	a.profiles, err = config.NewStore(dir)
	if err != nil {
		a.fatal(err)
		return
	}
	a.sysproxy = platform.NewSysProxy()
	if on, err := a.sysproxy.Enabled(); err == nil {
		a.systemProxy = on
	}
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) {
		a.forceQuit = true
	})
	mygo.App.OnQuit(func() {
		a.flushTrafficHistory()
		a.closeLogFile()

		if a.proxyOwned && a.systemProxy && a.sysproxy != nil {
			_ = a.sysproxy.Restore(a.proxySnapshot)
		}
	})
	s := a.settings.Get()
	if hasArg("--tun-off") && s.TUN.Enable {
		_ = a.settings.Update(func(s *settings.Settings) { s.TUN.Enable = false })
		s = a.settings.Get()
	}
	if s.TUN.Enable && !platform.IsElevated() {
		if err := platform.RelaunchElevated(); err == nil {
			return
		}
	}
	a.mixed = strconv.Itoa(s.Core.MixedPort)
	a.controller = strconv.Itoa(s.Core.ControllerPort)
	a.modeIdx = modeIndex(s.Core.Mode)
	a.runModeIdx = 0
	if s.App.RunMode == "lightweight" {
		a.runModeIdx = 1
	}
	a.proxyModeIdx = 0
	if s.Proxy.Mode == "pac" {
		a.proxyModeIdx = 1
	}
	a.logLevelIdx = logLevelIndex(s.Core.LogLevel)
	a.themeIdx = themeIndex(s.UI.Theme)
	a.presetIdx = presetIndex(s.UI.Preset)
	a.densityIdx = densityIndex(s.UI.Density)
	a.opacity = s.UI.CardOpacity
	a.scale = s.UI.Scale
	a.monochromeIcons = s.UI.MonochromeIcons
	a.languageIdx = 0
	if s.App.Language == "zh-CN" {
		a.languageIdx = 1
	} else if s.App.Language == "en" {
		a.languageIdx = 2
	}
	if a.languageIdx == 2 {
		i18n.Set("en")
	} else {
		i18n.Set("zh-CN")
	}
	a.hotkeyShow = s.Hotkeys.Show
	a.hotkeyProxy = s.Hotkeys.SystemProxy
	a.hotkeyTUN = s.Hotkeys.TUN
	a.hotkeyMode = s.Hotkeys.Mode
	a.hotkeyQuit = s.Hotkeys.Quit
	a.proxyGroup = s.Proxy.LastGroup
	a.proxyNode = s.Proxy.LastNode
	a.delays = map[string]int{}
	a.delayErrs = map[string]string{}
	a.speeds = map[string]float64{}
	a.speedErrs = map[string]string{}
	a.proxyExpanded = map[string]bool{}
	a.lastProxyNow = map[string]string{}
	a.connRates = map[string]connRate{}
	// Pages cross-fade instead of snapping; springs keep a quick "push"
	// responsive when the user is switching back and forth.
	a.router.Transition = ui.TransitionMatch
	a.router.Animation = &ui.Smooth
	a.proxyColsIdx = 3
	a.proxySortIdx = 1
	a.bypass = strings.Join(s.Proxy.Bypass, ";")
	a.dnsEnable = s.DNS.Enable
	a.dnsListen = s.DNS.Listen
	a.dnsNameservers = strings.Join(s.DNS.Nameserver, ", ")
	a.dnsFallback = strings.Join(s.DNS.Fallback, ", ")
	a.dnsFakeIPRange = s.DNS.FakeIPRange
	a.dnsModeIdx = 1
	if s.DNS.EnhancedMode == "normal" {
		a.dnsModeIdx = 0
	}
	a.settingsSection = "general"
	a.logRetentionIdx = logRetentionIndex(s.Logs.Retention)
	a.logMaxMB = s.Logs.MaxSizeMB
	a.webdavURL = s.WebDAV.URL
	a.webdavUser = s.WebDAV.Username
	a.webdavPass = s.WebDAV.Password
	a.logMaxText = strconv.Itoa(a.logMaxMB)
	a.testDelayURL = s.Test.DelayURL
	a.testSpeedURL = s.Test.SpeedURL
	a.openLogFile(dir)
	go a.logMaintenance()
	a.unifiedDelay = s.Advanced.UnifiedDelay
	a.tcpConcurrent = s.Advanced.TCPConcurrent
	a.findProcessIdx = findProcessIndex(s.Advanced.FindProcessMode)
	a.fingerprint = s.Advanced.Fingerprint
	a.iface = s.Advanced.InterfaceName
	a.routingMark = strconv.Itoa(s.Advanced.RoutingMark)
	a.sniffEnable = s.Sniffer.Enable
	a.sniffForceDNS = s.Sniffer.ForceDNSMapping
	a.sniffOverride = s.Sniffer.OverrideDestination
	a.sniffPureIP = s.Sniffer.ParsePureIP
	a.sniffHTTP = s.Sniffer.HTTPPorts
	a.sniffTLS = s.Sniffer.TLSPorts
	a.sniffQUIC = s.Sniffer.QUICPorts
	a.sniffSkip = s.Sniffer.SkipDomain
	a.sniffForce = s.Sniffer.ForceDomain
	a.sniffSkipSrc = s.Sniffer.SkipSrc
	a.sniffSkipDst = s.Sniffer.SkipDst
	a.tunEnable = s.TUN.Enable
	a.tunStackIdx = tunStackIndex(s.TUN.Stack)
	a.tunDevice = s.TUN.Device
	a.tunAutoRoute = s.TUN.AutoRoute
	a.tunStrictRoute = s.TUN.StrictRoute
	a.tunDNSHijack = s.TUN.DNSHijack
	a.tunAutoDetect = s.TUN.AutoDetect
	a.autoStart = s.App.AutoStartProfile
	a.closeTray = s.App.CloseToTray
	a.openAtLogin = mygo.App.OpenAtLogin()
	a.elevatedAgent = platform.AgentAtLogin()
	a.notifications = s.App.Notifications
	acrylic.SetAppearance(s.UI.Theme)
	a.dark = mygo.Theme.IsDark()
	mygo.Theme.OnUpdated(func() {
		a.dark = mygo.Theme.IsDark()
		a.invalidate()
	})

	opts := mygo.WindowOptions{
		Title:          "myclash",
		Width:          1180,
		Height:         760,
		MinWidth:       940,
		MinHeight:      600,
		StateKey:       "main",
		TitleBarStyle:  mygo.TitleBarHiddenInset,
		TitleBarHeight: 42,
		Hidden:         mygo.App.WasOpenedAtLogin(),
		Content:        ui.View(a.view),
	}
	acrylic.Window(&opts, a.dark, acrylic.NormalizeBlur(effectiveBlur(s.UI.Preset)))
	if s.UI.Preset == "glass" {
		opts.Vibrancy = mygo.VibrancyGlass
	}
	if s.UI.Preset != "solid" {
		opts.BackgroundColor = "#00000000"
	}
	a.win = mygo.NewWindow(opts)
	a.win.OnClose(func(e *mygo.CloseEvent) {
		if a.attach {
			os.Exit(0)
		}
		if a.settings.Get().App.RunMode == "lightweight" && !a.forceQuit {
			e.PreventDefault()
			a.switchToAgent()
			return
		}
		if a.closeTray && !a.forceQuit {
			e.PreventDefault()
			a.win.Hide()
		}
	})
	if a.attach {
		if err := a.attachAgent(); err != nil {
			a.attach = false
			a.setError(err)
		}
	}
	if !a.attach {
		a.setupTray()
		a.registerHotkeys()
	}

	if !a.attach && s.App.AutoStartProfile && a.profiles.Selected() != "" {
		a.startCore()
	}
}

func (a *App) fatal(err error) {
	a.setError(err)
}

func (a *App) view(c *ui.Context) {
	p := acrylic.For(a.dark)
	c.SetTheme(a.uiTheme(p))
	s := a.settings.Get()
	if s.UI.Preset == "solid" {
		acrylic.Ground(c, p)
	} else {
		c.Root().Background(ui.Transparent)
	}
	z := s.UI.Scale
	if z < 1 {
		z = 1
	}
	if z > 1.25 {
		z = 1.25
	}
	if c.Zoom() != float32(z) {
		c.SetZoom(float32(z))
	}

	bar := c.TitleBar()
	topH := bar.Bottom
	if topH < 38 {
		topH = 38
	}
	ui.Column(c).Fill().Children(func() {
		ui.Row(c).Height(topH).Gap(10).AlignItems(ui.Center).
			Padding(0, bar.Right+12, 0, bar.Left+14).DragWindow().
			BorderWidth(0, 0, 1, 0).BorderColor(p.Hairline).Children(func() {
			ui.Text(c, "myclash").FontSize(14).Bold()
			ui.Text(c, "原生代理客户端").FontSize(11).TextColor(p.InkSubtle)
			ui.Spacer(c)
			s := a.state.View()
			if s.Started {
				acrylic.Chip(c, p, "运行中", p.Success)
			} else {
				acrylic.Chip(c, p, "已停止", p.InkSubtle)
			}
			if s.Mode != "" {
				acrylic.Chip(c, p, modeLabel(s.Mode), p.InkSubtle)
			}
			if a.tunEnable {
				acrylic.Chip(c, p, "TUN", p.Accent)
			}
			if platform.IsElevated() {
				acrylic.Chip(c, p, "管理员", p.Success)
			}
			if a.attach {
				acrylic.Chip(c, p, "Agent", p.Accent)
			}
		})

		ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
			a.sidebar(c, p)
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				ui.Scroll(c).Fill().Background(ui.Transparent).Clip().Gap(a.gap()).Padding(a.pagePad()).Children(func() {
					a.router.View(c, func(r *ui.Route) {
						switch {
						case r.Match("/overview"):
							r.Title("概览")
							a.pageOverview(c, p)
						case r.Match("/proxies"):
							r.Title("代理")
							a.pageProxies(c, p)
						case r.Match("/profiles"):
							r.Title("配置")
							a.pageProfiles(c, p)
						case r.Match("/connections"):
							r.Title("连接")
							a.pageConnections(c, p)
						case r.Match("/traffic"):
							r.Title("流量")
							a.pageTraffic(c, p)
						case r.Match("/rules"):
							r.Title("规则")
							a.pageRules(c, p)
						case r.Match("/logs"):
							r.Title("日志")
							a.pageLogs(c, p)
						case r.Match("/override"):
							r.Title("覆写")
							a.pageOverride(c, p)
						case r.Match("/resources"):
							r.Title("资源")
							a.pageResources(c, p)
						case r.Match("/tun"):
							r.Title("TUN")
							a.pageTUN(c, p)
						case r.Match("/dns"):
							r.Title("DNS")
							a.pageDNS(c, p)
						case r.Match("/sniffer"):
							r.Title("嗅探")
							a.pageSniffer(c, p)
						case r.Match("/uwp"):
							r.Title("UWP 回环")
							a.pageUWP(c, p)
						case r.Match("/sysproxy"):
							r.Title("系统代理")
							a.pageSysProxy(c, p)
						case r.Match("/settings"):
							r.Title("设置")
							a.pageSettings(c, p)
						default:
							ui.Text(c, "页面不存在")
						}
					})
				})
				a.statusStrip(c, p)
			})
		})
	})
	a.commandPalette(c, p)
}

func (a *App) statusStrip(c *ui.Context, p acrylic.Palette) {
	s := a.state.View()
	ui.Row(c).Height(30).Background(ui.Transparent).Padding(0, 24).Gap(14).AlignItems(ui.Center).
		BorderWidth(1, 0, 0, 0).BorderColor(p.Hairline).Children(func() {
		ui.Textf(c, "↓ %s", format.Rate(s.Down)).FontSize(11).TextColor(p.InkSubtle)
		ui.Textf(c, "↑ %s", format.Rate(s.Up)).FontSize(11).TextColor(p.InkSubtle)
		ui.Textf(c, "连接 %d", len(s.Connections)).FontSize(11).TextColor(p.InkSubtle)
		if s.Memory > 0 {
			ui.Textf(c, "内存 %s", format.Bytes(int64(s.Memory))).FontSize(11).TextColor(p.InkSubtle)
		}
		ui.Spacer(c)
		if acrylic.Button(c, p, fmt.Sprintf("缩放 %.0f%%", a.settings.Get().UI.Scale*100), false).Clicked() {
			a.scale = 1
			a.applyAppearance()
		}
		a.mu.Lock()
		status := a.status
		a.mu.Unlock()
		if status != "" {
			ui.Text(c, status).FontSize(11).TextColor(p.InkSubtle)
		}
	})
}

func modeLabel(mode string) string {
	switch mode {
	case "global":
		return "全局"
	case "direct":
		return "直连"
	default:
		return "规则"
	}
}
func (a *App) options() config.Options {
	s := a.settings.Get()
	return config.Options{
		MixedPort:      s.Core.MixedPort,
		ControllerPort: s.Core.ControllerPort,
		Mode:           s.Core.Mode,
		LogLevel:       s.Core.LogLevel,
		AllowLAN:       s.Core.AllowLAN,
		IPv6:           s.Core.IPv6,
		DNS: config.DNSOptions{
			Enable:       s.DNS.Enable,
			Listen:       s.DNS.Listen,
			EnhancedMode: s.DNS.EnhancedMode,
			IPv6:         s.DNS.IPv6,
			Nameserver:   s.DNS.Nameserver,
			Fallback:     s.DNS.Fallback,
			FakeIPRange:  s.DNS.FakeIPRange,
		},
		Advanced: config.AdvancedOptions{
			UnifiedDelay:    a.unifiedDelay,
			TCPConcurrent:   a.tcpConcurrent,
			FindProcessMode: []string{"off", "strict", "always"}[clampIdx(a.findProcessIdx, 3)],
			Fingerprint:     a.fingerprint,
			InterfaceName:   a.iface,
			RoutingMark:     atoiSafe(a.routingMark),
		},
		Sniffer: config.SnifferOptions{
			Enable:              a.sniffEnable,
			ForceDNSMapping:     a.sniffForceDNS,
			OverrideDestination: a.sniffOverride,
			ParsePureIP:         a.sniffPureIP,
			HTTPPorts:           splitList(a.sniffHTTP),
			TLSPorts:            splitList(a.sniffTLS),
			QUICPorts:           splitList(a.sniffQUIC),
			SkipDomain:          splitList(a.sniffSkip),
			ForceDomain:         splitList(a.sniffForce),
			SkipSrc:             splitList(a.sniffSkipSrc),
			SkipDst:             splitList(a.sniffSkipDst),
		},
		TUN: config.TUNOptions{
			Enable:      a.tunActive(),
			Stack:       tunStacks[clampIdx(a.tunStackIdx, len(tunStacks))],
			Device:      a.tunDevice,
			AutoRoute:   a.tunAutoRoute,
			StrictRoute: a.tunStrictRoute,
			DNSHijack:   a.tunDNSHijack,
			AutoDetect:  a.tunAutoDetect,
		},
	}
}

func (a *App) startCore() {
	if a.attach {
		a.setError(errors.New("当前由轻量 Agent 托管"))
		return
	}
	if a.profiles.Selected() == "" {
		a.setError(errors.New("先添加并启用一个配置"))
		return
	}
	a.mu.Lock()
	if a.busy {
		a.mu.Unlock()
		return
	}
	a.busy = true
	a.errMsg = ""
	a.status = "正在启动内核…"
	a.mu.Unlock()
	a.invalidate()

	raw, err := a.profiles.Runtime(a.options())
	if err != nil {
		a.setBusyError(err)
		return
	}
	host, err := core.NewEmbedded(core.Home(a.dataDir), a.settings.Get().Core.ControllerPort)
	if err != nil {
		a.setBusyError(err)
		return
	}

	go func() {
		ctx := context.Background()
		if err := host.Start(ctx, raw); err != nil {
			a.setBusyError(err)
			return
		}
		client := core.NewClient(host.Endpoint())
		version, err := client.Version(ctx)
		if err != nil {
			_ = host.Stop(ctx)
			a.setBusyError(err)
			return
		}
		proxies, err := client.Proxies(ctx)
		if err != nil {
			_ = host.Stop(ctx)
			a.setBusyError(err)
			return
		}
		rules, _ := client.Rules(ctx)
		proxyProviders, _ := client.ProxyProviders(ctx)
		ruleProviders, _ := client.RuleProviders(ctx)
		a.updateUI(func() {
			a.host = host
			a.client = client
			a.state.SetStarted(true)
			a.state.SetVersion(version, a.settings.Get().Core.Mode)
			a.state.SetProxies(proxies)
			a.state.SetRules(rules)
			a.proxyProviders = proxyProviders
			a.ruleProviders = ruleProviders
			a.mu.Lock()
			a.busy = false
			a.status = "内核已启动"
			a.mu.Unlock()
		})
		a.startStreams(client)
		a.restoreProxySelection(client, proxies)
	}()
}

// Traffic and connection streams are event driven; repainting each event
// burns frames without making the UI smoother. Coalesce them to a stable
// frame budget while still keeping every sample in the state.
func (a *App) paintTraffic() {
	a.paintMu.Lock()
	if time.Since(a.lastTrafficPaint) < 50*time.Millisecond {
		a.paintMu.Unlock()
		return
	}
	a.lastTrafficPaint = time.Now()
	a.paintMu.Unlock()
	a.invalidateWindows()
}

func (a *App) paintConnections() {
	a.paintMu.Lock()
	if time.Since(a.lastConnPaint) < 120*time.Millisecond {
		a.paintMu.Unlock()
		return
	}
	a.lastConnPaint = time.Now()
	a.paintMu.Unlock()
	a.invalidateWindows()
}

func (a *App) invalidateWindows() {
	a.invalidate()
	if a.floating != nil {
		a.floating.Invalidate()
	}
}

func (a *App) trackConnRates(conns []core.Connection) {
	now := time.Now()
	next := make(map[string]connRate, len(conns))
	a.mu.Lock()
	for _, c := range conns {
		r := connRate{up: c.Upload, down: c.Download, at: now}
		if prev, ok := a.connRates[c.ID]; ok {
			if dt := now.Sub(prev.at).Seconds(); dt > 0 {
				if c.Upload >= prev.up {
					r.upRate = float64(c.Upload-prev.up) / dt
				}
				if c.Download >= prev.down {
					r.downRate = float64(c.Download-prev.down) / dt
				}
			}
		}
		next[c.ID] = r
	}
	a.connRates = next
	a.mu.Unlock()
}

func (a *App) connRateFor(id string) (float64, float64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	r, ok := a.connRates[id]
	if !ok {
		return 0, 0
	}
	return r.upRate, r.downRate
}

func (a *App) startStreams(client *core.Client) {
	ctx, cancel := context.WithCancel(a.ctx)
	a.mu.Lock()
	old := a.cancel
	a.cancel = cancel
	level := a.settings.Get().Core.LogLevel
	a.mu.Unlock()
	old()

	go func() {
		last := time.Now()
		err := client.Traffic(ctx, func(up, down int64) {
			now := time.Now()
			dt := now.Sub(last)
			last = now
			a.state.SetTraffic(up, down, dt)
			a.recordTraffic(up, down)
			a.paintTraffic()
		})
		if err != nil && ctx.Err() == nil {
			a.setError(fmt.Errorf("流量流断开: %w", err))
		}
	}()

	go func() {
		err := client.Connections(ctx, func(v core.Connections) {
			a.trackConnRates(v.Connections)
			a.state.SetConnections(v)
			a.paintConnections()
		})
		if err != nil && ctx.Err() == nil {
			a.setError(fmt.Errorf("连接流断开: %w", err))
		}
	}()

	go func() {
		last := time.Time{}
		err := client.Logs(ctx, level, func(e core.LogEntry) {
			a.state.AppendLog(e)
			a.writeLogFile(e)
			if time.Since(last) > 150*time.Millisecond {
				last = time.Now()
				a.invalidate()
			}
		})
		if err != nil && ctx.Err() == nil {
			a.setError(fmt.Errorf("日志流断开: %w", err))
		}
	}()

	go func() {
		_ = client.Memory(ctx, func(inuse uint64) {
			a.state.SetMemory(inuse)
		})
	}()
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				rules, err := client.Rules(ctx)
				if err != nil {
					continue
				}
				a.updateUI(func() {
					a.state.SetRules(rules)
					a.invalidate()
				})
			}
		}
	}()
}

func (a *App) stopCore() {
	a.mu.Lock()
	host := a.host
	cancel := a.cancel
	a.host = nil
	a.client = nil
	a.busy = true
	a.status = "正在停止内核…"
	a.mu.Unlock()
	cancel()
	if host != nil {
		_ = host.Stop(context.Background())
	}
	a.updateUI(func() {
		a.state.SetStarted(false)
		a.mu.Lock()
		a.busy = false
		a.status = "内核已停止"
		a.mu.Unlock()
	})
	a.invalidate()
}

func (a *App) restartCore() {
	if a.attach {
		if err := a.agentRequest("/restart"); err != nil {
			a.setError(err)
			return
		}
		a.setStatus("Agent 内核已重载")
		return
	}
	// Use the same path as stop, so status, streams, connections and the
	// state snapshot are all cleared before the new core starts.
	a.stopCore()
	a.mu.Lock()
	a.busy = false
	a.errMsg = ""
	a.mu.Unlock()
	a.startCore()
}

func (a *App) refreshProxies() {
	a.mu.Lock()
	client := a.client
	a.mu.Unlock()
	if client == nil {
		return
	}
	go func() {
		proxies, err := client.Proxies(context.Background())
		if err != nil {
			a.setError(err)
			return
		}
		a.mu.Lock()
		valid := make(map[string]bool, len(proxies))
		for name := range proxies {
			valid[name] = true
		}
		for name := range a.delays {
			if !valid[name] {
				delete(a.delays, name)
			}
		}
		for name := range a.delayErrs {
			if !valid[name] {
				delete(a.delayErrs, name)
			}
		}
		for name := range a.speeds {
			if !valid[name] {
				delete(a.speeds, name)
			}
		}
		for name := range a.speedErrs {
			if !valid[name] {
				delete(a.speedErrs, name)
			}
		}
		for name := range a.lastProxyNow {
			if !valid[name] {
				delete(a.lastProxyNow, name)
			}
		}
		for group := range a.proxyExpanded {
			if !valid[group] {
				delete(a.proxyExpanded, group)
			}
		}
		for group, p := range proxies {
			if p.Now == "" {
				continue
			}
			if old, seen := a.lastProxyNow[group]; seen && old != p.Now {
				a.proxyGroup, a.proxyNode = group, p.Now
			}
			a.lastProxyNow[group] = p.Now
		}
		a.mu.Unlock()
		a.updateUI(func() { a.state.SetProxies(proxies) })
	}()
}

func (a *App) restoreProxySelection(client *core.Client, proxies map[string]core.Proxy) {
	a.mu.Lock()
	group, node := a.proxyGroup, a.proxyNode
	a.mu.Unlock()
	p, ok := proxies[group]
	if !ok || node == "" {
		return
	}
	known := node == p.Now
	for _, member := range p.All {
		if member == node {
			known = true
			break
		}
	}
	if !known {
		a.mu.Lock()
		a.proxyGroup, a.proxyNode = "", ""
		a.mu.Unlock()
		return
	}
	if p.Now == node {
		return
	}
	go func() {
		if err := client.Select(context.Background(), group, node); err == nil {
			a.refreshProxies()
		}
	}()
}

func (a *App) focusProxy(group, name string) {
	if name == "" {
		if p, ok := a.state.View().Proxies[group]; ok {
			name = p.Now
		}
	}
	a.mu.Lock()
	a.proxyGroup, a.proxyNode = group, name
	a.mu.Unlock()
	if name != "" {
		go a.persistProxy(group, name)
	}
	a.invalidate()
}

func (a *App) persistProxy(group, name string) {
	_ = a.settings.Update(func(s *settings.Settings) {
		s.Proxy.LastGroup = group
		s.Proxy.LastNode = name
	})
}

func (a *App) selectProxy(group, name string) {
	a.mu.Lock()
	client := a.client
	oldGroup, oldNode := a.proxyGroup, a.proxyNode
	a.proxyGroup, a.proxyNode = group, name
	a.mu.Unlock()
	if client == nil {
		return
	}
	go a.persistProxy(group, name)
	// Show the choice immediately; the REST call catches up in a moment.
	a.invalidate()
	go func() {
		if err := client.Select(context.Background(), group, name); err != nil {
			a.mu.Lock()
			a.proxyGroup, a.proxyNode = oldGroup, oldNode
			a.mu.Unlock()
			a.setError(err)
			a.invalidate()
			return
		}
		go a.persistProxy(group, name)
		a.refreshProxies()
	}()
}

func (a *App) delayFor(name string) (int, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	ms, ok := a.delays[name]
	return ms, ok
}

func (a *App) delayErrFor(name string) (string, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	msg, ok := a.delayErrs[name]
	return msg, ok
}

func (a *App) testDelay(name string) {
	a.mu.Lock()
	client := a.client
	testURL := a.testDelayURL
	a.mu.Unlock()
	if client == nil {
		return
	}
	go a.probeDelay(client, name, testURL)
}

func (a *App) probeDelay(client *core.Client, name, testURL string) {
	ms, err := client.Delay(context.Background(), name, testURL, 5000)
	a.updateUI(func() {
		a.mu.Lock()
		if err != nil {
			a.delayErrs[name] = "超时"
			delete(a.delays, name)
			a.status = fmt.Sprintf("%s 延迟测试失败", name)
		} else {
			a.delays[name] = ms
			delete(a.delayErrs, name)
			a.status = fmt.Sprintf("%s 延迟 %d ms", name, ms)
		}
		a.mu.Unlock()
	})
}

func (a *App) testGroupDelay(group string) {
	a.mu.Lock()
	client := a.client
	a.mu.Unlock()
	if client == nil {
		return
	}
	go func() {
		proxies, err := client.Proxies(context.Background())
		if err != nil {
			a.setError(err)
			return
		}
		members := proxies[group].All
		sem := make(chan struct{}, 8)
		var wg sync.WaitGroup
		for _, member := range members {
			member := member
			wg.Add(1)
			go func() {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				a.probeDelay(client, member, a.testDelayURL)
			}()
		}
		wg.Wait()
		a.updateUI(func() { a.status = group + " 延迟测试完成" })
	}()
}

func (a *App) applyMode(idx int) {
	modes := []string{"rule", "global", "direct"}
	if idx < 0 || idx >= len(modes) {
		return
	}
	mode := modes[idx]
	if err := a.settings.Update(func(s *settings.Settings) { s.Core.Mode = mode }); err != nil {
		a.setError(err)
		return
	}
	a.mu.Lock()
	host := a.host
	a.mu.Unlock()
	if host != nil {
		go func() {
			if err := host.SetMode(context.Background(), mode); err != nil {
				a.setError(err)
				return
			}
			a.updateUI(func() { a.state.SetMode(mode) })
		}()
	}
	a.refreshTrayMenu()
}

func (a *App) saveSettings() {
	port, err := strconv.Atoi(strings.TrimSpace(a.mixed))
	if err != nil || port < 1 || port > 65535 {
		a.setError(errors.New("混合端口必须是 1–65535"))
		return
	}
	controller, err := strconv.Atoi(strings.TrimSpace(a.controller))
	if err != nil || controller < 1 || controller > 65535 {
		a.setError(errors.New("Controller 端口必须是 1–65535"))
		return
	}
	old := a.settings.Get().Core
	if err := a.settings.Update(func(s *settings.Settings) {
		s.Core.MixedPort = port
		s.Core.ControllerPort = controller
	}); err != nil {
		a.setError(err)
		return
	}
	s := a.settings.Get()
	acrylic.SetAppearance(s.UI.Theme)
	if a.win != nil {
		a.invalidate()
	}
	if (port != old.MixedPort || controller != old.ControllerPort) && a.state.View().Started {
		a.restartCore()
	}
	a.mu.Lock()
	a.status = "设置已保存"
	a.mu.Unlock()
}
func (a *App) addSubscription() {
	url := strings.TrimSpace(a.subURL)
	if url == "" {
		return
	}
	a.mu.Lock()
	if a.busy {
		a.mu.Unlock()
		return
	}
	a.busy = true
	a.errMsg = ""
	a.mu.Unlock()
	a.invalidate()
	go func() {
		_, err := a.profiles.AddSubscription(context.Background(), "", url)
		a.updateUI(func() {
			a.mu.Lock()
			a.busy = false
			if err != nil {
				a.errMsg = err.Error()
			} else {
				a.subURL = ""
				a.status = "订阅已添加"
			}
			a.mu.Unlock()
		})
		if err == nil && !a.state.View().Started {
			a.startCore()
		}
	}()
}

func (a *App) importProfile() {
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent: a.win,
			Title:  "导入 mihomo YAML",
			Filters: []mygo.FileFilter{{
				Name:       "YAML",
				Extensions: []string{"yaml", "yml"},
			}},
		})
		if err != nil || len(paths) == 0 {
			return
		}
		raw, err := os.ReadFile(paths[0])
		if err != nil {
			a.setError(err)
			return
		}
		if _, err := a.profiles.AddLocal(filepath.Base(paths[0]), raw); err != nil {
			a.setError(err)
			return
		}
		a.updateUI(func() {
			a.mu.Lock()
			a.status = "本地配置已导入"
			a.mu.Unlock()
		})
		if !a.state.View().Started {
			a.startCore()
		}
	}()
}

func (a *App) selectProfile(uid string) {
	if err := a.profiles.Select(uid); err != nil {
		a.setError(err)
		return
	}
	a.restartCore()
}

func (a *App) deleteProfile(uid string) {
	if err := a.profiles.Delete(uid); err != nil {
		a.setError(err)
		return
	}
	a.invalidate()
}

func (a *App) setError(err error) {
	if err == nil {
		return
	}
	a.mu.Lock()
	a.errMsg = err.Error()
	a.busy = false
	a.mu.Unlock()
	a.writeLogFile(core.LogEntry{Type: "error", Payload: err.Error()})
	a.notify("myclash 需要处理", err.Error())
	a.invalidate()
}

func (a *App) setStatus(msg string) {
	a.mu.Lock()
	a.status = msg
	a.mu.Unlock()
	a.invalidate()
}

func (a *App) setBusyError(err error) {
	a.setError(err)
}

func (a *App) invalidate() {
	if a.win != nil {
		a.win.Invalidate()
	}
}

func (a *App) updateUI(fn func()) {
	if a.win != nil {
		a.win.Update(fn)
	}
}

func (a *App) errorBar(c *ui.Context, p acrylic.Palette) {
	a.mu.Lock()
	msg := a.errMsg
	status := a.status
	a.mu.Unlock()
	if msg != "" {
		ui.Text(c, msg).TextColor(p.Danger)
	}
	if status != "" {
		ui.Text(c, status).TextColor(p.InkSubtle).FontSize(12)
	}
}

func (a *App) header(c *ui.Context, p acrylic.Palette, title, sub string) {
	ui.Text(c, title).FontSize(22).Bold()
	if sub != "" {
		ui.Text(c, sub).TextColor(p.InkMuted).FontSize(12)
	}
}

func modeIndex(mode string) int {
	switch mode {
	case "global":
		return 1
	case "direct":
		return 2
	default:
		return 0
	}
}

func themeIndex(theme string) int {
	switch theme {
	case "light":
		return 1
	case "dark":
		return 2
	default:
		return 0
	}
}

func (a *App) syncProfile(uid string) {
	a.mu.Lock()
	if a.busy {
		a.mu.Unlock()
		return
	}
	a.busy = true
	a.errMsg = ""
	a.mu.Unlock()
	a.invalidate()
	go func() {
		_, err := a.profiles.Sync(context.Background(), uid)
		a.updateUI(func() {
			a.mu.Lock()
			a.busy = false
			if err != nil {
				a.errMsg = err.Error()
			} else {
				a.status = "订阅已同步"
			}
			a.mu.Unlock()
		})
		if err == nil && a.profiles.Selected() == uid && a.state.View().Started {
			a.restartCore()
		}
		a.invalidate()
	}()
}

func presetIndex(v string) int {
	switch v {
	case "clear":
		return 1
	case "solid":
		return 2
	}
	return 0
}

func tunStackIndex(v string) int {
	switch v {
	case "system":
		return 0
	case "gvisor":
		return 1
	case "mixed":
		return 2
	case "mips":
		return 3
	}
	return 3
}

func densityIndex(v string) int {
	if v == "compact" {
		return 1
	}
	return 0
}

var (
	presets   = []string{"glass", "clear", "solid"}
	densities = []string{"comfortable", "compact"}
	tunStacks = []string{"system", "gvisor", "mixed", "mips"}
)

func (a *App) applyAppearance() {
	preset := presets[clampIdx(a.presetIdx, len(presets))]
	density := densities[clampIdx(a.densityIdx, len(densities))]
	blur := effectiveBlur(preset)
	if err := a.settings.Update(func(s *settings.Settings) {
		s.UI.Preset = preset
		s.UI.Blur = blur
		s.UI.Density = density
		s.UI.CardOpacity = a.opacity
		s.UI.Scale = a.scale
	}); err != nil {
		a.setError(err)
		return
	}
	if a.win != nil {
		if preset == "glass" {
			a.win.SetVibrancy(mygo.VibrancyGlass)
		} else {
			a.win.SetVibrancy(mygo.VibrancyNone)
		}
		if preset != "solid" {
			_ = a.win.SetBackgroundColor("#00000000")
		}
		a.invalidate()
	}
}

func effectiveBlur(preset string) string {
	if preset == "glass" || preset == "acrylic" {
		return "deep"
	}
	return "none"
}

func (a *App) panel(c *ui.Context, p acrylic.Palette, opacity float64) ui.Element {
	return acrylic.Panel(c, p, opacity)
}

func (a *App) card(c *ui.Context, p acrylic.Palette, opacity float64) ui.Element {
	return acrylic.Card(c, p, opacity)
}

func (a *App) tunActive() bool {
	return a.tunEnable && platform.IsElevated()
}

func (a *App) toggleTUN() {
	if a.attach {
		enable := a.tunEnable
		path := "/tun?enabled=0"
		if enable {
			path = "/tun?enabled=1"
		}
		if err := a.agentRequest(path); err != nil {
			a.setError(err)
			a.tunEnable = !enable
			a.invalidate()
			return
		}
		a.saveTUN()
		a.setStatus("Agent TUN 已更新")
		a.invalidate()
		return
	}
	a.saveTUN()
	if !a.tunEnable {
		if a.state.View().Started {
			a.restartCore()
		}
		return
	}
	if !platform.IsElevated() {
		if err := platform.RelaunchElevated(); err != nil {
			a.setError(err)
			return
		}
		mygo.App.Quit()
		return
	}
	if a.state.View().Started {
		a.restartCore()
	}
}

func mihomoVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, dep := range info.Deps {
		if dep.Path == "github.com/metacubex/mihomo" {
			return strings.TrimPrefix(dep.Version, "v")
		}
	}
	return ""
}

func (a *App) cardOpacity() float64 {
	s := a.settings.Get()
	if s.UI.Preset == "solid" {
		return 1
	}
	return s.UI.CardOpacity
}

func clampIdx(i, n int) int {
	if i < 0 {
		return 0
	}
	if i >= n {
		return n - 1
	}
	return i
}

func (a *App) toggleSystemProxy(on bool) {
	if a.sysproxy == nil {
		return
	}
	if on && !a.state.View().Started {
		a.systemProxy = false
		a.setError(errors.New("先启动内核再开启系统代理"))
		return
	}
	s := a.settings.Get()
	if on {
		snap, err := a.sysproxy.Snapshot()
		if err != nil {
			a.systemProxy = false
			a.setError(err)
			return
		}
		a.proxySnapshot = snap
		if s.Proxy.Mode == "pac" {
			pac, ok := a.sysproxy.(platform.PACProxy)
			if !ok {
				a.systemProxy = false
				a.setError(platform.ErrUnsupported)
				return
			}
			pacURL, err := a.startPAC()
			if err != nil {
				a.systemProxy = false
				a.setError(err)
				return
			}
			if err := pac.EnablePAC(pacURL); err != nil {
				a.stopPAC()
				a.systemProxy = false
				a.setError(err)
				return
			}
		} else if err := a.sysproxy.Enable("127.0.0.1", s.Core.MixedPort, s.Proxy.Bypass); err != nil {
			a.systemProxy = false
			a.setError(err)
			return
		}
	} else if a.proxyOwned {
		if err := a.sysproxy.Restore(a.proxySnapshot); err != nil {
			a.setError(err)
		}
		a.stopPAC()
	} else if err := a.sysproxy.Disable(); err != nil {
		a.systemProxy = true
		a.setError(err)
		return
	} else {
		a.stopPAC()
	}
	a.systemProxy = on
	a.proxyOwned = on
	if err := a.settings.Update(func(s *settings.Settings) { s.Proxy.SystemEnabled = on }); err != nil {
		a.setError(err)
	}
	a.refreshTrayMenu()
	a.invalidate()
}

func (a *App) saveDNS() {
	nameservers := splitList(a.dnsNameservers)
	fallback := splitList(a.dnsFallback)
	if err := a.settings.Update(func(s *settings.Settings) {
		s.DNS.Enable = a.dnsEnable
		s.DNS.Listen = strings.TrimSpace(a.dnsListen)
		s.DNS.EnhancedMode = []string{"normal", "fake-ip"}[clampIdx(a.dnsModeIdx, 2)]
		s.DNS.Nameserver = nameservers
		s.DNS.Fallback = fallback
		s.DNS.FakeIPRange = strings.TrimSpace(a.dnsFakeIPRange)
	}); err != nil {
		a.setError(err)
		return
	}
	a.mu.Lock()
	a.status = "DNS 设置已保存"
	a.mu.Unlock()
	if a.state.View().Started {
		a.restartCore()
	}
}

func splitList(v string) []string {
	fields := strings.FieldsFunc(v, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == ' '
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if f = strings.TrimSpace(f); f != "" {
			out = append(out, f)
		}
	}
	return out
}

func (a *App) saveOverride() {
	current, ok := a.profiles.CurrentMeta()
	if !ok || a.override == nil {
		return
	}
	text := a.override.Buffer.String()
	if err := a.profiles.SaveOverride(current.UID, []byte(text)); err != nil {
		a.setError(err)
		return
	}
	a.mu.Lock()
	a.status = "覆写已保存"
	a.mu.Unlock()
	if a.profiles.Selected() == current.UID && a.state.View().Started {
		a.restartCore()
	} else {
		a.invalidate()
	}
}

func (a *App) saveScriptOverride() {
	current, ok := a.profiles.CurrentMeta()
	if !ok || a.overrideScript == nil {
		return
	}
	if err := a.profiles.SaveScriptOverride(current.UID, a.overrideScript.Buffer.String()); err != nil {
		a.setError(err)
		return
	}
	a.mu.Lock()
	a.status = "脚本覆写已保存"
	a.mu.Unlock()
	if a.profiles.Selected() == current.UID && a.state.View().Started {
		a.restartCore()
	} else {
		a.invalidate()
	}
}

func (a *App) closeConnection(id string) {
	a.mu.Lock()
	client := a.client
	a.mu.Unlock()
	if client == nil {
		return
	}
	go func() {
		if err := client.CloseConnection(a.ctx, id); err != nil {
			a.setError(err)
		}
	}()
}

func (a *App) restoreSystemProxy() {
	if a.sysproxy == nil {
		return
	}
	if err := a.sysproxy.Restore(a.proxySnapshot); err != nil {
		a.setError(err)
		return
	}
	a.systemProxy = false
	a.proxyOwned = false
	_ = a.settings.Update(func(s *settings.Settings) { s.Proxy.SystemEnabled = false })
	a.refreshTrayMenu()
	a.invalidate()
}

func (a *App) toggleConnPaused() {
	if a.connPaused {
		a.connPaused = false
		a.frozenConns = nil
		return
	}
	a.frozenConns = append([]core.Connection(nil), a.state.View().Connections...)
	a.connPaused = true
}

func logRetentionIndex(v string) int {
	switch v {
	case "forever":
		return 0
	case "day":
		return 1
	case "month":
		return 3
	default:
		return 2
	}
}

func logRetentionValue(idx int) string {
	values := []string{"forever", "day", "week", "month"}
	if idx < 0 || idx >= len(values) {
		return "week"
	}
	return values[idx]
}

func logLevelIndex(v string) int {
	switch v {
	case "debug":
		return 1
	case "warning":
		return 2
	case "error":
		return 3
	default:
		return 0
	}
}

func (a *App) applyLogLevel(idx int) {
	levels := []string{"info", "debug", "warning", "error"}
	level := levels[clampIdx(idx, len(levels))]
	if err := a.settings.Update(func(s *settings.Settings) { s.Core.LogLevel = level }); err != nil {
		a.setError(err)
		return
	}
	if a.state.View().Started {
		a.restartCore()
	}
}

func (a *App) saveBypass() {
	parts := splitList(a.bypass)
	if len(parts) == 0 {
		parts = []string{"<local>", "127.*"}
	}
	if err := a.settings.Update(func(s *settings.Settings) { s.Proxy.Bypass = parts }); err != nil {
		a.setError(err)
		return
	}
	a.bypass = strings.Join(parts, ";")
	if a.systemProxy {
		_ = a.sysproxy.Enable("127.0.0.1", a.settings.Get().Core.MixedPort, parts)
	}
	a.invalidate()
}

func findProcessIndex(v string) int {
	switch v {
	case "off":
		return 0
	case "always":
		return 2
	default:
		return 1
	}
}

func atoiSafe(v string) int {
	n, _ := strconv.Atoi(v)
	return n
}

func (a *App) applyAdvanced() {
	modes := []string{"off", "strict", "always"}
	if err := a.settings.Update(func(s *settings.Settings) {
		s.Advanced.UnifiedDelay = a.unifiedDelay
		s.Advanced.TCPConcurrent = a.tcpConcurrent
		s.Advanced.FindProcessMode = modes[clampIdx(a.findProcessIdx, 3)]
		s.Advanced.Fingerprint = strings.TrimSpace(a.fingerprint)
		s.Advanced.InterfaceName = strings.TrimSpace(a.iface)
		s.Advanced.RoutingMark = atoiSafe(a.routingMark)
	}); err != nil {
		a.setError(err)
		return
	}
	if a.state.View().Started {
		a.restartCore()
	}
}

func (a *App) beginRename(uid, name string) {
	a.renameUID = uid
	a.renameText = name
}

func (a *App) saveRename() {
	if a.renameUID == "" || strings.TrimSpace(a.renameText) == "" {
		return
	}
	if err := a.profiles.Rename(a.renameUID, strings.TrimSpace(a.renameText)); err != nil {
		a.setError(err)
		return
	}
	a.renameUID = ""
	a.renameText = ""
	a.invalidate()
}

func (a *App) duplicateProfile(uid string) {
	if _, err := a.profiles.Duplicate(uid); err != nil {
		a.setError(err)
		return
	}
	a.invalidate()
}

func (a *App) setRuleDisabled(index int, disabled bool) {
	if err := core.SetRuleDisabled(index, disabled); err != nil {
		a.setError(err)
		return
	}
	a.refreshRules()
}

func (a *App) uiTheme(p acrylic.Palette) *ui.Theme {
	th := acrylic.Theme(p)
	s := a.settings.Get()
	if s.UI.Density == "compact" {
		th.Spacing = 2.5
		th.FontSize = 11.5
		th.Radius = 5
	} else {
		th.Spacing = 4
		th.FontSize = 13.5
		th.Radius = 8
	}
	return th
}

func (a *App) pagePad() float32 {
	if a.settings.Get().UI.Density == "compact" {
		return 14
	}
	return 24
}

func (a *App) cardPad() float32 {
	if a.settings.Get().UI.Density == "compact" {
		return 10
	}
	return 16
}

func (a *App) gap() float32 {
	if a.settings.Get().UI.Density == "compact" {
		return 8
	}
	return 14
}

func hasArg(want string) bool {
	for _, arg := range os.Args[1:] {
		if arg == want {
			return true
		}
	}
	return false
}
