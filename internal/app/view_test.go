package app

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"myclash/internal/config"
	"myclash/internal/core"
	"myclash/internal/settings"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	dir := t.TempDir()
	st, err := settings.NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	ps, err := config.NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	return &App{
		dataDir:         dir,
		settings:        st,
		profiles:        ps,
		state:           core.NewState(),
		router:          ui.NewRouter("/overview"),
		settingsSection: "general",
		side:            "overview",
		dark:            true,
		cancel:          func() {},
	}
}

func TestViewRendersOverview(t *testing.T) {
	a := newTestApp(t)
	tt := ui.NewTester(a.view, 1180, 760)
	if !tt.HasText("概览") {
		t.Fatalf("overview title missing; texts=%q", tt.Texts())
	}
	if !tt.HasText("欢迎使用 myclash") {
		t.Fatalf("welcome missing; texts=%q", tt.Texts())
	}
}

func TestViewProfilesEmptyState(t *testing.T) {
	a := newTestApp(t)
	a.side = "profiles"
	a.router = ui.NewRouter("/profiles")
	tt := ui.NewTester(a.view, 1180, 760)
	if !tt.HasText("添加订阅") || !tt.HasText("导入本地 YAML") {
		t.Fatalf("profiles actions missing; texts=%q", tt.Texts())
	}
}

func TestViewSettings(t *testing.T) {
	a := newTestApp(t)
	a.side = "settings"
	a.router = ui.NewRouter("/settings")
	tt := ui.NewTester(a.view, 1180, 760)
	if !tt.HasText("设置") || !tt.HasText("外观") || !tt.HasText("内核") {
		t.Fatalf("settings sections missing; texts=%q", tt.Texts())
	}
}

func TestViewOverviewWithProfile(t *testing.T) {
	a := newTestApp(t)
	if _, err := a.profiles.AddLocal("demo", []byte("proxies: []\nrules:\n  - MATCH,DIRECT\n")); err != nil {
		t.Fatal(err)
	}
	a.state.SetStarted(true)
	tt := ui.NewTester(a.view, 1180, 760)
	if !tt.HasText("当前配置") || !tt.HasText("启动内核") && !tt.HasText("停止内核") {
		t.Fatalf("overview cards missing; texts=%q", tt.Texts())
	}
}

func TestViewSettingsHasOneNav(t *testing.T) {
	a := newTestApp(t)
	a.side = "settings"
	a.router = ui.NewRouter("/settings")
	tt := ui.NewTester(a.view, 1180, 760)
	n := 0
	for _, s := range tt.Texts() {
		if s == "托盘" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("settings nav duplicated (托盘): %d, texts=%q", n, tt.Texts())
	}
}

func TestViewDNSIsSettings(t *testing.T) {
	a := newTestApp(t)
	a.side = "dns"
	a.router = ui.NewRouter("/dns")
	tt := ui.NewTester(a.view, 1180, 760)
	if !tt.HasText("启用 DNS 覆盖") || tt.HasText("查询结果会显示在这里。") {
		t.Fatalf("dns page wrong; texts=%q", tt.Texts())
	}
}

func TestViewTUNPage(t *testing.T) {
	a := newTestApp(t)
	a.side = "tun"
	a.router = ui.NewRouter("/tun")
	tt := ui.NewTester(a.view, 1180, 760)
	if !tt.HasText("启用 TUN") || !tt.HasText("诊断") {
		t.Fatalf("TUN page missing; texts=%q", tt.Texts())
	}
}

func TestViewAppearanceModes(t *testing.T) {
	a := newTestApp(t)
	a.side = "settings"
	a.settingsSection = "appearance"
	a.router = ui.NewRouter("/settings")
	tt := ui.NewTester(a.view, 1180, 760)
	for _, want := range []string{"液态玻璃", "透明", "实色", "面板不透明度"} {
		if !tt.HasText(want) {
			t.Fatalf("appearance missing %q; texts=%q", want, tt.Texts())
		}
	}
	for _, gone := range []string{"混合", "模糊档位（系统材质）"} {
		if tt.HasText(gone) {
			t.Fatalf("legacy appearance option %q still shown", gone)
		}
	}
}
