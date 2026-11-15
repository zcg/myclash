package app

import (
	"github.com/egoist/mygo/ui"

	"myclash/internal/settings"
	"myclash/internal/ui/acrylic"
)

func (a *App) pageSniffer(c *ui.Context, p acrylic.Palette) {
	ui.Column(c).Gap(2).Children(func() {
		ui.Text(c, "嗅探").FontSize(24).Bold()
		ui.Text(c, "从连接中恢复域名；保存后重启内核").FontSize(12).TextColor(p.InkSubtle)
	})
	a.errorBar(c, p)

	panel := a.panel(c, p, a.cardOpacity()).Padding(a.cardPad())
	panel.Children(func() {
		acrylic.Checkbox(c, p, &a.sniffEnable, "启用域名嗅探")
		ui.Row(c).Gap(16).Wrap().Children(func() {
			acrylic.Checkbox(c, p, &a.sniffForceDNS, "强制 DNS 映射")
			acrylic.Checkbox(c, p, &a.sniffOverride, "覆盖连接地址")
			acrylic.Checkbox(c, p, &a.sniffPureIP, "纯 IP 嗅探")
		})
		field := func(label string, value *string, width float32) {
			ui.Column(c).Gap(2).Children(func() {
				ui.Text(c, label).FontSize(10).TextColor(p.InkSubtle)
				ui.TextInput(c, value).Width(width)
			})
		}
		ui.Row(c).Gap(12).Wrap().Children(func() {
			field("HTTP 端口", &a.sniffHTTP, 160)
			field("TLS 端口", &a.sniffTLS, 160)
			field("QUIC 端口", &a.sniffQUIC, 160)
		})
		ui.Row(c).Gap(12).Wrap().Children(func() {
			field("跳过域名", &a.sniffSkip, 260)
			field("强制域名", &a.sniffForce, 260)
		})
		ui.Row(c).Gap(12).Wrap().Children(func() {
			field("跳过源地址", &a.sniffSkipSrc, 260)
			field("跳过目标地址", &a.sniffSkipDst, 260)
		})
		if acrylic.PrimaryButton(c, p, "保存并重载", false).Clicked() {
			a.saveSniffer()
		}
	})
}

func (a *App) saveSniffer() {
	if err := a.settings.Update(func(s *settings.Settings) {
		s.Sniffer.Enable = a.sniffEnable
		s.Sniffer.ForceDNSMapping = a.sniffForceDNS
		s.Sniffer.OverrideDestination = a.sniffOverride
		s.Sniffer.ParsePureIP = a.sniffPureIP
		s.Sniffer.HTTPPorts = a.sniffHTTP
		s.Sniffer.TLSPorts = a.sniffTLS
		s.Sniffer.QUICPorts = a.sniffQUIC
		s.Sniffer.SkipDomain = a.sniffSkip
		s.Sniffer.ForceDomain = a.sniffForce
		s.Sniffer.SkipSrc = a.sniffSkipSrc
		s.Sniffer.SkipDst = a.sniffSkipDst
	}); err != nil {
		a.setError(err)
		return
	}
	if a.state.Get().Started {
		a.restartCore()
	}
}
