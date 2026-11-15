package app

import (
	"fmt"
	"net"

	"github.com/egoist/mygo/ui"

	"myclash/internal/platform"
	"myclash/internal/settings"
	"myclash/internal/ui/acrylic"
)

func (a *App) pageTUN(c *ui.Context, p acrylic.Palette) {
	ui.Column(c).Gap(2).Children(func() {
		ui.Text(c, "TUN").FontSize(24).Bold()
		ui.Text(c, "虚拟网卡接管全局流量；开启需要管理员权限").FontSize(12).TextColor(p.InkSubtle)
	})
	a.errorBar(c, p)

	card := a.panel(c, p, a.cardOpacity()).Padding(a.cardPad())
	card.Children(func() {
		if acrylic.Checkbox(c, p, &a.tunEnable, "启用 TUN").Changed() {
			a.saveTUN()
		}
		ui.Text(c, "栈").FontSize(11).TextColor(p.InkSubtle)
		stack := acrylic.SegmentedText(c, p, &a.tunStackIdx, "System", "GVisor", "Mixed", "Mips")
		if stack.Changed() {
			a.saveTUN()
		}
		ui.TextInput(c, &a.tunDevice).Label("网卡名").Width(280)
		ui.Row(c).Gap(18).Wrap().Children(func() {
			if acrylic.Checkbox(c, p, &a.tunAutoRoute, "自动路由").Changed() {
				a.saveTUN()
			}
			if acrylic.Checkbox(c, p, &a.tunStrictRoute, "严格路由").Changed() {
				a.saveTUN()
			}
			if acrylic.Checkbox(c, p, &a.tunDNSHijack, "DNS 劫持").Changed() {
				a.saveTUN()
			}
			if acrylic.Checkbox(c, p, &a.tunAutoDetect, "自动探测接口").Changed() {
				a.saveTUN()
			}
		})
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			if acrylic.PrimaryButton(c, p, "保存并重启内核", false).Clicked() {
				a.saveTUN()
				if a.tunEnable && !platform.IsElevated() {
					if err := platform.RelaunchElevated(); err != nil {
						a.setError(err)
					}
					return
				}
				if a.state.Get().Started {
					a.restartCore()
				}
			}
			if !platform.IsElevated() {
				if acrylic.Button(c, p, "以管理员身份重启", false).Clicked() {
					a.tunEnable = true
					a.saveTUN()
					if err := platform.RelaunchElevated(); err != nil {
						a.setError(err)
					}
				}
			}
		})
	})

	diag := a.panel(c, p, a.cardOpacity()).Padding(a.cardPad())
	diag.Children(func() {
		ui.Text(c, "诊断").Bold()
		a.tunDiagRow(c, p, "权限", elevationText())
		a.tunDiagRow(c, p, "协议栈", tunStacks[clampIdx(a.tunStackIdx, len(tunStacks))])
		if a.tunActive() {
			a.tunDiagRow(c, p, "内核注入", "已启用 TUN")
		} else {
			a.tunDiagRow(c, p, "内核注入", "未启用（需要管理员授权）")
		}
		a.tunDiagRow(c, p, "内核", startedText(a.state.Get().Started))
		a.tunDiagRow(c, p, "网卡", interfaceText(a.tunDevice))
		a.tunDiagRow(c, p, "系统代理", proxyText(a.systemProxy))
		if a.tunEnable && !platform.IsElevated() {
			ui.Text(c, "当前进程没有管理员权限，TUN 启动会被内核拒绝。请点击上方「以管理员身份重启」。").FontSize(11).TextColor(p.Danger)
		}
	})
}

func (a *App) tunDiagRow(c *ui.Context, p acrylic.Palette, label, value string) {
	ui.Row(c).Gap(10).Children(func() {
		ui.Text(c, label).FontSize(11).TextColor(p.InkSubtle).Width(90)
		ui.Text(c, value).FontSize(11).TextColor(p.InkMuted)
	})
}

func (a *App) saveTUN() {
	if a.tunEnable && a.systemProxy {
		a.toggleSystemProxy(false)
		a.tunProxyWasOn = true
	}
	if !a.tunEnable && a.tunProxyWasOn {
		a.toggleSystemProxy(true)
		a.tunProxyWasOn = false
	}
	stack := tunStacks[clampIdx(a.tunStackIdx, len(tunStacks))]
	if err := a.settings.Update(func(s *settings.Settings) {
		s.TUN.Enable = a.tunEnable
		s.TUN.Stack = stack
		s.TUN.Device = a.tunDevice
		s.TUN.AutoRoute = a.tunAutoRoute
		s.TUN.StrictRoute = a.tunStrictRoute
		s.TUN.DNSHijack = a.tunDNSHijack
		s.TUN.AutoDetect = a.tunAutoDetect
	}); err != nil {
		a.setError(err)
		return
	}
	if a.tunEnable && !platform.IsElevated() {
		a.setError(fmt.Errorf("TUN 需要管理员权限：点击「以管理员身份重启」"))
	}
	a.invalidate()
}

func elevationText() string {
	if platform.IsElevated() {
		return "已授权（管理员）"
	}
	return "未授权（普通用户）"
}

func startedText(started bool) string {
	if started {
		return "运行中"
	}
	return "已停止"
}

func interfaceText(name string) string {
	if name == "" {
		return "未设置"
	}
	if iface, err := net.InterfaceByName(name); err == nil {
		return fmt.Sprintf("%s（up=%v）", iface.Name, iface.Flags&net.FlagUp != 0)
	}
	return name + "（尚未创建，开启后由内核建立）"
}

func proxyText(on bool) string {
	if on {
		return "开启（TUN 会将其关闭并记忆）"
	}
	return "关闭"
}
