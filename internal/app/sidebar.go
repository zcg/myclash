package app

import (
	"github.com/egoist/mygo/ui"

	"myclash/internal/ui/acrylic"
	"myclash/internal/ui/icons"
)

func (a *App) sidebar(c *ui.Context, p acrylic.Palette) {
	if a.monochromeIcons {
		a.monoSidebar(c, p)
		return
	}
	sb := ui.Sidebar(c, &a.side, func() {
		ui.SidebarSection(c, "主", nil, func() {
			ui.SidebarItem(c, "overview", icons.Gauge, "概览")
			ui.SidebarItem(c, "proxies", icons.Waypoints, "代理")
			ui.SidebarItem(c, "profiles", icons.Layers, "配置")
		})
		ui.SidebarSection(c, "运行", nil, func() {
			ui.SidebarItem(c, "connections", icons.Activity, "连接")
			ui.SidebarItem(c, "rules", icons.ListTree, "规则")
			ui.SidebarItem(c, "logs", icons.ScrollText, "日志")
		})
		ui.SidebarSection(c, "网络", nil, func() {
			ui.SidebarItem(c, "tun", icons.ShieldCheck, "TUN")
			ui.SidebarItem(c, "dns", icons.Waypoints, "DNS")
			ui.SidebarItem(c, "sniffer", icons.ScrollText, "嗅探")
			ui.SidebarItem(c, "sysproxy", icons.ShieldCheck, "系统代理")
		})
		ui.SidebarSection(c, "工具", nil, func() {
			ui.SidebarItem(c, "override", icons.Layers, "覆写")
			ui.SidebarItem(c, "resources", icons.Server, "资源")
		})
		ui.SidebarSection(c, "系统", nil, func() {
			ui.SidebarItem(c, "settings", icons.Settings, "设置")
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
		{"主", []navItem{
			{"overview", "概览", icons.Gauge},
			{"proxies", "代理", icons.Waypoints},
			{"profiles", "配置", icons.Layers},
		}},
		{"运行", []navItem{
			{"connections", "连接", icons.Activity},
			{"rules", "规则", icons.ListTree},
			{"logs", "日志", icons.ScrollText},
		}},
		{"网络", []navItem{
			{"tun", "TUN", icons.ShieldCheck},
			{"dns", "DNS", icons.Waypoints},
			{"sniffer", "嗅探", icons.ScrollText},
			{"sysproxy", "系统代理", icons.ShieldCheck},
		}},
		{"工具", []navItem{
			{"override", "覆写", icons.Layers},
			{"resources", "资源", icons.Server},
		}},
		{"系统", []navItem{
			{"settings", "设置", icons.Settings},
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
					a.side = item.id
					a.router.Push("/" + item.id)
				}
			}
		}
	})
}
