package app

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"myclash/internal/ui/acrylic"
)

type paletteItem struct {
	label string
	hint  string
	run   func()
}

func (a *App) paletteItems() []paletteItem {
	nav := func(label, path string) paletteItem {
		return paletteItem{label: "页面 · " + label, hint: path, run: func() {
			a.side = strings.TrimPrefix(path, "/")
			a.router.Push(path)
		}}
	}
	return []paletteItem{
		nav("概览", "/overview"),
		nav("代理", "/proxies"),
		nav("配置", "/profiles"),
		nav("连接", "/connections"),
		nav("规则", "/rules"),
		nav("日志", "/logs"),
		nav("TUN", "/tun"),
		nav("DNS", "/dns"),
		nav("系统代理", "/sysproxy"),
		nav("覆写", "/override"),
		nav("资源", "/resources"),
		nav("设置", "/settings"),
		{label: "操作 · 启动/停止内核", hint: "", run: func() {
			if a.state.Get().Started {
				a.stopCore()
			} else {
				a.startCore()
			}
		}},
		{label: "操作 · 重启内核", hint: "", run: a.restartCore},
		{label: "操作 · 规则模式", hint: "", run: func() { a.applyMode(0) }},
		{label: "操作 · 全局模式", hint: "", run: func() { a.applyMode(1) }},
		{label: "操作 · 直连模式", hint: "", run: func() { a.applyMode(2) }},
		{label: "操作 · 切换系统代理", hint: "", run: func() {
			a.systemProxy = !a.systemProxy
			a.toggleSystemProxy(a.systemProxy)
		}},
		{label: "操作 · 切换 TUN", hint: "", run: func() {
			a.tunEnable = !a.tunEnable
			a.toggleTUN()
		}},
		{label: "操作 · 连通性测试", hint: "", run: a.testSites},
		{label: "操作 · 清空日志", hint: "", run: a.clearLogs},
	}
}

func (a *App) commandPalette(c *ui.Context, p acrylic.Palette) {
	if !a.paletteOpen {
		return
	}
	q := strings.ToLower(strings.TrimSpace(a.paletteQuery))
	var hits []paletteItem
	for _, item := range a.paletteItems() {
		if q == "" || strings.Contains(strings.ToLower(item.label), q) {
			hits = append(hits, item)
		}
	}
	if a.paletteIndex >= len(hits) {
		a.paletteIndex = 0
	}
	acrylic.Modal(c, p, &a.paletteOpen, func() {
		ui.Column(c).Width(560).Gap(8).Children(func() {
			ui.Text(c, "命令面板").FontSize(16).Bold()
			input := ui.TextInput(c, &a.paletteQuery).Placeholder("搜索页面或操作").AutoFocus()
			if input.Changed() {
				a.paletteIndex = 0
			}
			if input.Submitted() && len(hits) > 0 {
				item := hits[a.paletteIndex]
				a.paletteOpen = false
				item.run()
				return
			}
			if len(hits) == 0 {
				ui.Text(c, "没有匹配的页面或操作。").TextColor(p.InkSubtle)
				return
			}
			ui.Column(c).Gap(2).Children(func() {
				for i, item := range hits {
					item := item
					row := acrylic.Row(c, p, i == a.paletteIndex)
					row.Children(func() {
						ui.Text(c, item.label).FontSize(12).SingleLine().Grow(1)
						if item.hint != "" {
							ui.Text(c, item.hint).FontSize(10).TextColor(p.InkSubtle)
						}
					})
					if row.Hovered() {
						a.paletteIndex = i
					}
					if row.Clicked() {
						a.paletteOpen = false
						item.run()
					}
				}
			})
			ui.Text(c, "Enter 执行首个结果 · Esc 关闭").FontSize(10).TextColor(p.InkSubtle)
		})
	})
}
