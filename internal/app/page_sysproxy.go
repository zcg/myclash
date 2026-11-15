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

	ui.Text(c, "Windows 使用 WinINet 设置；macOS/Linux 的系统代理在后续版本接入。").FontSize(11).TextColor(p.InkSubtle)
}
