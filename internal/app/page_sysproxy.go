package app

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"myclash/internal/ui/acrylic"
)

func (a *App) pageSysProxy(c *ui.Context, p acrylic.Palette) {
	ui.Column(c).Gap(2).Children(func() {
		ui.Text(c, "系统代理").FontSize(24).Bold()
		ui.Text(c, "写入当前用户的系统代理设置；退出时恢复原值").FontSize(12).TextColor(p.InkSubtle)
	})
	a.errorBar(c, p)

	card := a.panel(c, p, a.cardOpacity()).Padding(a.cardPad())
	card.Children(func() {
		ui.Text(c, "代理方式").FontSize(11).TextColor(p.InkSubtle)
		if seg := acrylic.SegmentedText(c, p, &a.proxyModeIdx, "手动", "PAC"); seg.Changed() {
			a.applyProxyMode()
		}
		if a.proxyModeIdx == 1 {
			ui.Text(c, "PAC 由本机 127.0.0.1 提供，按绕过列表和 mixed 端口生成。").FontSize(11).TextColor(p.InkSubtle)
			if a.pacURL != "" {
				ui.Text(c, "PAC："+a.pacURL).FontSize(11).TextColor(p.InkMuted).SingleLine()
			}
		}
		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Gap(2).Children(func() {
				ui.Text(c, "启用系统代理").FontSize(15).Bold()
				ui.Textf(c, "127.0.0.1:%d", a.settings.Get().Core.MixedPort).FontSize(11).TextColor(p.InkSubtle)
			})
			ui.Spacer(c)
			if acrylic.Checkbox(c, p, &a.systemProxy, "启用").Changed() {
				a.toggleSystemProxy(a.systemProxy)
			}
		})
		ui.Text(c, "绕过列表："+strings.Join(a.settings.Get().Proxy.Bypass, "；")).FontSize(11).TextColor(p.InkMuted)
		ui.TextInput(c, &a.bypass).Label("绕过列表（分号分隔）").FillWidth()
		if acrylic.Button(c, p, "保存绕过列表", false).Clicked() {
			a.saveBypass()
		}
		if a.proxyOwned {
			if acrylic.Button(c, p, "还原原设置", false).Clicked() {
				a.restoreSystemProxy()
			}
		}
	})

	ui.Text(c, "Windows 使用 WinINet；macOS 使用 networksetup；Linux 使用 GNOME gsettings。其他桌面环境会提示不支持。").FontSize(11).TextColor(p.InkSubtle)
}
