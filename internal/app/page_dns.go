package app

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"myclash/internal/ui/acrylic"
)

// pageDNS is a settings page, not a resolver console: this mirrors what other
// mihomo clients expose, a place to tune the core's DNS section.
func (a *App) pageDNS(c *ui.Context, p acrylic.Palette) {
	ui.Column(c).Gap(2).Children(func() {
		ui.Text(c, "DNS").FontSize(24).Bold()
		ui.Text(c, "应用级设置，保存后注入当前配置；空字段保留配置原值").FontSize(12).TextColor(p.InkSubtle)
	})
	a.errorBar(c, p)

	panel := a.panel(c, p, a.cardOpacity()).Padding(a.cardPad())
	panel.Children(func() {
		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Gap(2).Children(func() {
				ui.Text(c, "启用 DNS 覆盖").FontSize(14).Bold()
				ui.Text(c, "关闭时完全沿用配置文件的 dns 段").FontSize(11).TextColor(p.InkSubtle)
			})
			ui.Spacer(c)
			acrylic.Checkbox(c, p, &a.dnsEnable, "启用")
		})

		ui.Row(c).Gap(12).Children(func() {
			ui.TextInput(c, &a.dnsListen).Label("监听地址").Placeholder("127.0.0.1:1053").Width(220)
			ui.TextInput(c, &a.dnsFakeIPRange).Label("Fake-IP 范围").Placeholder("198.18.0.1/16").Width(220)
		})
		ui.Text(c, "增强模式").FontSize(11).TextColor(p.InkSubtle)
		acrylic.SegmentedText(c, p, &a.dnsModeIdx, "Normal", "Fake-IP")

		ui.Text(c, "Nameserver").FontSize(11).TextColor(p.InkSubtle)
		ui.TextInput(c, &a.dnsNameservers).Placeholder("https://223.5.5.5/dns-query, https://1.1.1.1/dns-query").FillWidth()
		ui.Text(c, "Fallback").FontSize(11).TextColor(p.InkSubtle)
		ui.TextInput(c, &a.dnsFallback).Placeholder("tls://8.8.8.8:853").FillWidth()
		ui.Text(c, "多个地址用逗号或换行分隔；保存会重启内核。").FontSize(11).TextColor(p.InkSubtle)

		ui.Row(c).Gap(10).Children(func() {
			if acrylic.PrimaryButton(c, p, "保存并重载", false).Clicked() {
				a.saveDNS()
			}
		})
	})

	effective := a.settings.Get().DNS
	ui.Textf(c, "当前：%s · %s · %s", boolLabel(effective.Enable), effective.EnhancedMode, strings.Join(effective.Nameserver, ", ")).
		FontSize(11).TextColor(p.InkMuted)
}

func boolLabel(v bool) string {
	if v {
		return "已启用"
	}
	return "未启用"
}
