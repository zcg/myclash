package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"myclash/internal/core"
	"myclash/internal/ui/acrylic"
)

var providerColumns = []ui.TableColumn{
	{Title: "名称", Width: 0},
	{Title: "类型", Width: 90},
	{Title: "条目", Width: 80, Align: ui.End},
	{Title: "更新时间", Width: 160},
	{Title: "操作", Width: 150, Align: ui.End},
}

func (a *App) pageResources(c *ui.Context, p acrylic.Palette) {
	ui.Column(c).Gap(2).Children(func() {
		ui.Text(c, "资源").FontSize(24).Bold()
		ui.Text(c, "代理提供者与规则提供者；Geo 与 Web 面板在 M2 接入").FontSize(12).TextColor(p.InkSubtle)
	})
	ui.Row(c).Gap(8).Children(func() {
		if acrylic.Button(c, p, "刷新", false).Clicked() {
			a.refreshProviders()
		}
	})
	a.errorBar(c, p)

	ui.Text(c, "代理提供者").Bold()
	if len(a.proxyProviders) == 0 {
		ui.Text(c, "没有代理提供者。").FontSize(11).TextColor(p.InkMuted)
	} else {
		renderProviders(c, p, &a.proxyProvState, a.proxyProviders, false, a)
	}
	ui.Text(c, "规则提供者").Bold()
	if len(a.ruleProviders) == 0 {
		ui.Text(c, "没有规则提供者。").FontSize(11).TextColor(p.InkMuted)
	} else {
		renderProviders(c, p, &a.ruleProvState, a.ruleProviders, true, a)
	}
}

func renderProviders(c *ui.Context, p acrylic.Palette, state *ui.ListState, rows []core.Provider, rules bool, a *App) {
	if len(rows) > 100 {
		rows = rows[:100]
	}
	ui.Column(c).Height(220).Background(p.Code).Radius(8).Padding(10).Clip().Children(func() {
		ui.Table(c, state, providerColumns, len(rows), func(row, col int) {
			pr := rows[row]
			switch col {
			case 0:
				ui.Text(c, pr.Name).SingleLine()
			case 1:
				ui.Text(c, pr.Type).SingleLine()
			case 2:
				n := pr.ProxyCount
				if rules {
					n = pr.RuleCount
				}
				ui.Textf(c, "%d", n)
			case 3:
				if pr.UpdatedAt.IsZero() {
					ui.Text(c, "—")
				} else {
					ui.Text(c, pr.UpdatedAt.Format("2006-01-02 15:04")).SingleLine()
				}
			case 4:
				if rules {
					if acrylic.Button(c, p, "更新", false).Clicked() {
						a.updateRuleProvider(pr.Name)
					}
				} else {
					if acrylic.Button(c, p, "更新", false).Clicked() {
						a.updateProxyProvider(pr.Name)
					}
					if acrylic.Button(c, p, "健康检查", false).Clicked() {
						a.healthCheckProvider(pr.Name)
					}
				}
			}
		}).Grow(1)
	})
}

func (a *App) refreshProviders() {
	a.mu.Lock()
	client := a.client
	a.mu.Unlock()
	if client == nil {
		return
	}
	go func() {
		proxies, err := client.ProxyProviders(a.ctx)
		if err != nil {
			a.setError(err)
			return
		}
		rules, err := client.RuleProviders(a.ctx)
		if err != nil {
			a.setError(err)
			return
		}
		a.updateUI(func() {
			a.proxyProviders = proxies
			a.ruleProviders = rules
		})
	}()
}

func (a *App) updateProxyProvider(name string) {
	a.mu.Lock()
	client := a.client
	a.mu.Unlock()
	if client == nil {
		return
	}
	go func() {
		if err := client.UpdateProxyProvider(a.ctx, name); err != nil {
			a.setError(err)
			return
		}
		a.refreshProviders()
	}()
}

func (a *App) updateRuleProvider(name string) {
	a.mu.Lock()
	client := a.client
	a.mu.Unlock()
	if client == nil {
		return
	}
	go func() {
		if err := client.UpdateRuleProvider(a.ctx, name); err != nil {
			a.setError(err)
			return
		}
		a.refreshProviders()
	}()
}

func (a *App) healthCheckProvider(name string) {
	a.mu.Lock()
	client := a.client
	a.mu.Unlock()
	if client == nil {
		return
	}
	go func() {
		if err := client.HealthCheckProvider(a.ctx, name); err != nil {
			a.setError(err)
			return
		}
		a.updateUI(func() {
			a.mu.Lock()
			a.status = fmt.Sprintf("%s 健康检查完成", name)
			a.mu.Unlock()
		})
	}()
}
