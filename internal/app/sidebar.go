package app

import (
	"runtime"

	"github.com/egoist/mygo/ui"

	"myclash/internal/i18n"
	"myclash/internal/ui/acrylic"
	"myclash/internal/ui/icons"
)

func (a *App) sidebar(c *ui.Context, p acrylic.Palette) {
	if a.monochromeIcons {
		a.monoSidebar(c, p)
		return
	}
	sb := ui.Sidebar(c, &a.side, func() {
		ui.SidebarSection(c, i18n.T("主", "Main"), nil, func() {
			ui.SidebarItem(c, "overview", icons.Gauge, i18n.T("概览", "Overview"))
			ui.SidebarItem(c, "proxies", icons.Waypoints, i18n.T("代理", "Proxies"))
			ui.SidebarItem(c, "profiles", icons.Layers, i18n.T("配置", "Profiles"))
		})
		ui.SidebarSection(c, i18n.T("运行", "Runtime"), nil, func() {
			ui.SidebarItem(c, "connections", icons.Activity, i18n.T("连接", "Connections"))
			ui.SidebarItem(c, "traffic", icons.Gauge, i18n.T("流量", "Traffic"))
			ui.SidebarItem(c, "rules", icons.ListTree, i18n.T("规则", "Rules"))
			ui.SidebarItem(c, "logs", icons.ScrollText, i18n.T("日志", "Logs"))
		})
		ui.SidebarSection(c, i18n.T("网络", "Network"), nil, func() {
			ui.SidebarItem(c, "tun", icons.ShieldCheck, i18n.T("TUN", "TUN"))
			ui.SidebarItem(c, "dns", icons.Waypoints, i18n.T("DNS", "DNS"))
			ui.SidebarItem(c, "sniffer", icons.ScrollText, i18n.T("嗅探", "Sniffing"))
			ui.SidebarItem(c, "sysproxy", icons.ShieldCheck, i18n.T("系统代理", "System proxy"))
		})
		ui.SidebarSection(c, i18n.T("工具", "Tools"), nil, func() {
			ui.SidebarItem(c, "override", icons.Layers, i18n.T("覆写", "Override"))
			ui.SidebarItem(c, "resources", icons.Server, i18n.T("资源", "Resources"))
			if runtime.GOOS == "windows" {
				ui.SidebarItem(c, "uwp", icons.ShieldCheck, i18n.T("UWP 回环", "UWP loopback"))
			}
		})
		ui.SidebarSection(c, i18n.T("系统", "System"), nil, func() {
			ui.SidebarItem(c, "settings", icons.Settings, i18n.T("设置", "Settings"))
		})
	}).Width(232).Background(ui.Transparent).BorderWidth(0, 1, 0, 0).BorderColor(p.Hairline)
	if sb.Changed() {
		a.router.Push("/" + a.side)
	}
}

type navItem struct {
	id, label string
	icon      *ui.SVG
}

func (a *App) monoSidebar(c *ui.Context, p acrylic.Palette) {
	sections := []struct {
		title string
		items []navItem
	}{
		{i18n.T("主", "Main"), []navItem{
			{"overview", i18n.T("概览", "Overview"), icons.Gauge},
			{"proxies", i18n.T("代理", "Proxies"), icons.Waypoints},
			{"profiles", i18n.T("配置", "Profiles"), icons.Layers},
		}},
		{i18n.T("运行", "Runtime"), []navItem{
			{"connections", i18n.T("连接", "Connections"), icons.Activity},
			{"traffic", i18n.T("流量", "Traffic"), icons.Gauge},
			{"rules", i18n.T("规则", "Rules"), icons.ListTree},
			{"logs", i18n.T("日志", "Logs"), icons.ScrollText},
		}},
		{i18n.T("网络", "Network"), []navItem{
			{"tun", i18n.T("TUN", "TUN"), icons.ShieldCheck},
			{"dns", i18n.T("DNS", "DNS"), icons.Waypoints},
			{"sniffer", i18n.T("嗅探", "Sniffing"), icons.ScrollText},
			{"sysproxy", i18n.T("系统代理", "System proxy"), icons.ShieldCheck},
		}},
		{i18n.T("工具", "Tools"), []navItem{
			{"override", i18n.T("覆写", "Override"), icons.Layers},
			{"resources", i18n.T("资源", "Resources"), icons.Server},
			{"uwp", i18n.T("UWP 回环", "UWP loopback"), icons.ShieldCheck},
		}},
		{i18n.T("系统", "System"), []navItem{
			{"settings", i18n.T("设置", "Settings"), icons.Settings},
		}},
	}
	ui.Column(c).Width(232).Shrink(0).Background(ui.Transparent).
		BorderWidth(0, 1, 0, 0).BorderColor(p.Hairline).Children(func() {
		for _, section := range sections {
			ui.Text(c, section.title).FontSize(10).TextColor(p.InkTertiary).Padding(8, 10, 2, 10)
			for _, item := range section.items {
				item := item
				selected := a.side == item.id
				row := ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(7, 10).Radius(7).
					Cursor(ui.CursorPointer)
				if selected {
					row.Background(p.WashSelected)
				} else if row.Hovered() {
					row.Background(p.WashHover)
				}
				row = acrylic.Fade(row)
				row.Children(func() {
					color := p.InkSubtle
					if selected {
						color = p.Ink
					}
					acrylic.Icon(c, item.icon, 16, color)
					ui.Text(c, item.label).TextColor(color)
				})
				if row.Clicked() {
					a.setPage(item.id)
					a.router.Push("/" + item.id)
				}
			}
		}
	})
}
