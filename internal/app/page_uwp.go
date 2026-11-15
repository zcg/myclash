package app

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/egoist/mygo/ui"

	"myclash/internal/platform"
	"myclash/internal/ui/acrylic"
)

func (a *App) loadUWP() {
	if runtime.GOOS != "windows" || a.uwpBusy {
		return
	}
	a.uwpBusy = true
	go func() {
		apps, err := platform.ListUWP()
		a.updateUI(func() {
			a.uwpBusy = false
			a.uwpLoaded = true
			if err != nil {
				a.setError(fmt.Errorf("读取 UWP 应用失败: %w", err))
				return
			}
			a.uwpApps = apps
		})
	}()
}

func (a *App) pageUWP(c *ui.Context, p acrylic.Palette) {
	a.header(c, p, "UWP 回环", "允许 Windows 商店应用通过本机回环访问代理端口")
	if runtime.GOOS != "windows" {
		ui.Text(c, "UWP 回环豁免仅在 Windows 可用。").TextColor(p.InkMuted)
		return
	}
	if !a.uwpLoaded && !a.uwpBusy {
		a.loadUWP()
	}
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		acrylic.SearchField(c, p, &a.uwpSearch, "搜索软件名、发布者或包名")
		ui.Spacer(c)
		if acrylic.Button(c, p, "刷新", a.uwpBusy).Clicked() {
			a.uwpLoaded = false
			a.loadUWP()
		}
	})
	ui.Text(c, "开启后调用 CheckNetIsolation LoopbackExempt；需要管理员权限，会弹出一次 UAC。").
		FontSize(11).TextColor(p.InkSubtle)
	a.errorBar(c, p)
	if a.uwpBusy && !a.uwpLoaded {
		ui.Text(c, "正在读取已安装的 UWP 应用…").TextColor(p.InkMuted)
		return
	}
	q := strings.ToLower(strings.TrimSpace(a.uwpSearch))
	var rows []platform.UWPApp
	for _, app := range a.uwpApps {
		label := app.DisplayName
		if label == "" {
			label = app.Name
		}
		if q == "" || strings.Contains(strings.ToLower(label+" "+app.Name+" "+app.Publisher+" "+app.PackageFullName), q) {
			rows = append(rows, app)
		}
	}
	if len(rows) == 0 {
		ui.Text(c, "没有匹配的 UWP 应用。").TextColor(p.InkMuted)
		return
	}
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Textf(c, "%d 个应用", len(rows)).FontSize(11).TextColor(p.InkSubtle)
		ui.Spacer(c)
		if acrylic.Button(c, p, "全部允许回环", a.uwpBusy || len(rows) == 0).Clicked() {
			a.setUWPLoopbackMany(rows, true)
		}
		if acrylic.Button(c, p, "全部移除", a.uwpBusy || len(rows) == 0).Clicked() {
			a.setUWPLoopbackMany(rows, false)
		}
	})
	ui.Column(c).Height(520).Background(p.Code).Radius(8).Padding(10).Clip().Children(func() {
		ui.List(c, &a.uwpList, len(rows), func(i int) {
			app := rows[i]
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Column(c).Gap(2).Grow(1).Children(func() {
					label := app.DisplayName
					if label == "" {
						label = app.Name
					}
					ui.Text(c, label).FontSize(12).Bold().SingleLine()
					if app.Publisher != "" {
						ui.Text(c, app.Publisher).FontSize(10).TextColor(p.InkSubtle).SingleLine()
					}
					ui.Text(c, app.PackageFullName).FontSize(9).TextColor(p.InkTertiary).SingleLine()
				})
				if app.Loopback {
					if acrylic.Button(c, p, "移除回环", false).Clicked() {
						a.setUWPLoopback(app, false)
					}
				} else if acrylic.Button(c, p, "允许回环", false).Clicked() {
					a.setUWPLoopback(app, true)
				}
			})
		}).Grow(1)
	})
}

func (a *App) setUWPLoopback(app platform.UWPApp, enable bool) {
	if a.uwpBusy {
		return
	}
	a.uwpBusy = true
	go func() {
		err := platform.SetUWPLoopback(app.PackageFullName, enable)
		a.updateUI(func() {
			a.uwpBusy = false
			if err != nil {
				a.setError(fmt.Errorf("设置 UWP 回环失败: %w", err))
				return
			}
			a.loadUWP()
		})
	}()
}

func (a *App) setUWPLoopbackMany(rows []platform.UWPApp, enable bool) {
	if a.uwpBusy {
		return
	}
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		names = append(names, row.PackageFullName)
	}
	a.uwpBusy = true
	go func() {
		err := platform.SetUWPLoopbackMany(names, enable)
		a.updateUI(func() {
			a.uwpBusy = false
			if err != nil {
				a.setError(fmt.Errorf("批量设置 UWP 回环失败: %w", err))
				return
			}
			a.loadUWP()
		})
	}()
}
