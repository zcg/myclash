package app

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"myclash/internal/core"
	"myclash/internal/platform"
	"myclash/internal/settings"
	"myclash/internal/ui/acrylic"
	"myclash/internal/ui/format"
	"myclash/internal/ui/icons"
	"myclash/internal/ui/widgets"
)

func (a *App) pageOverview(c *ui.Context, p acrylic.Palette) {
	s := a.state.Get()
	opacity := a.cardOpacity()

	if _, ok := a.profiles.CurrentMeta(); !ok {
		a.welcome(c, p, opacity)
		return
	}

	ui.Column(c).Gap(2).Children(func() {
		ui.Text(c, "概览").FontSize(24).Bold()
		ui.Text(c, "内核状态、实时流量与当前节点").FontSize(12).TextColor(p.InkSubtle)
	})
	ui.Row(c).Gap(10).AlignItems(ui.Center).Wrap().Children(func() {
		label := "启动内核"
		if s.Started {
			label = "停止内核"
		}
		if acrylic.PrimaryButton(c, p, label, a.isBusy()).Clicked() {
			if s.Started {
				a.stopCore()
			} else {
				a.startCore()
			}
		}
		if acrylic.Button(c, p, "重启内核", !s.Started || a.isBusy()).Clicked() {
			a.restartCore()
		}
		if seg := acrylic.SegmentedText(c, p, &a.modeIdx, "规则", "全局", "直连"); seg.Changed() {
			a.applyMode(a.modeIdx)
		}
		if acrylic.Checkbox(c, p, &a.systemProxy, "系统代理").Changed() {
			a.toggleSystemProxy(a.systemProxy)
		}
		if acrylic.Checkbox(c, p, &a.tunEnable, "TUN").Changed() {
			a.toggleTUN()
		}
	})
	a.errorBar(c, p)

	ui.Row(c).Gap(8).Wrap().Children(func() {
		acrylic.Chip(c, p, modeLabel(s.Mode), p.InkSubtle)
		if a.tunEnable {
			if platform.IsElevated() {
				acrylic.Chip(c, p, "TUN 已授权", p.Success)
			} else {
				acrylic.Chip(c, p, "TUN 待管理员授权", p.Danger)
			}
		} else {
			acrylic.Chip(c, p, "TUN 关闭", p.InkSubtle)
		}
		if a.systemProxy {
			acrylic.Chip(c, p, "系统代理开", p.Success)
		} else {
			acrylic.Chip(c, p, "系统代理关", p.InkSubtle)
		}
		if platform.IsElevated() {
			acrylic.Chip(c, p, "管理员运行", p.Success)
		} else {
			acrylic.Chip(c, p, "普通权限", p.InkSubtle)
		}
		if meta, ok := a.profiles.CurrentMeta(); ok {
			acrylic.Chip(c, p, meta.Name, p.InkSubtle)
		}
	})

	down := rateSeries(s.History, true)
	up := rateSeries(s.History, false)
	ui.Row(c).Gap(12).AlignItems(ui.Stretch).Children(func() {
		widgets.StatCard(c, p, opacity, "下行", format.Rate(s.Down),
			"累计 "+format.Bytes(s.TotalDown), icons.ArrowDown, p.Accent, down)
		widgets.StatCard(c, p, opacity, "上行", format.Rate(s.Up),
			"累计 "+format.Bytes(s.TotalUp), icons.ArrowUp, p.Success, up)
	})
	ui.Row(c).Gap(12).AlignItems(ui.Stretch).Children(func() {
		widgets.StatCard(c, p, opacity, "连接", fmt.Sprint(len(s.Connections)),
			"会话总数", icons.Activity, p.Ink, nil)
		widgets.StatCard(c, p, opacity, "内核", kernelVersion(s.Version),
			"运行 "+uptime(s.StartedAt), icons.Server, p.Ink, nil)
	})

	ui.Row(c).Gap(12).AlignItems(ui.Stretch).Children(func() {
		profileCard := a.card(c, p, opacity).Padding(a.cardPad()).Grow(2)
		profileCard.Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "当前配置").Bold()
				ui.Spacer(c)
				if acrylic.Button(c, p, "管理", false).Clicked() {
					a.router.Push("/profiles")
				}
			})
			current, ok := a.profiles.CurrentMeta()
			if !ok {
				return
			}
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, current.Name).FontSize(18).Bold().SingleLine()
				acrylic.Chip(c, p, sourceLabel(current.Source), p.InkSubtle)
			})
			ui.Textf(c, "上次更新 %s", current.UpdatedAt.Format("2006-01-02 15:04")).FontSize(11).TextColor(p.InkSubtle)
			if u := current.Userinfo; u != nil && u.Total > 0 {
				used := u.Upload + u.Download
				ui.Progress(c, float64(used)/float64(u.Total)).Height(6).Width(320)
				ui.Textf(c, "已用 %s / %s", format.Bytes(used), format.Bytes(u.Total)).FontSize(11).TextColor(p.InkMuted)
				if u.Expire > 0 {
					ui.Textf(c, "到期 %s", time.Unix(u.Expire, 0).Format("2006-01-02")).FontSize(11).TextColor(p.InkSubtle)
				}
			}
		})

		nodeCard := a.card(c, p, opacity).Padding(a.cardPad()).Grow(1)
		nodeCard.Children(func() {
			group, proxy, ok := a.currentNode(s)
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "当前节点").Bold()
				ui.Spacer(c)
				if acrylic.Button(c, p, "代理页", false).Clicked() {
					if ok {
						a.proxyExpanded[group] = true
						a.scrollProxyGroup = group
					}
					a.router.Push("/proxies")
				}
			})
			if !ok {
				ui.Text(c, "内核启动后显示当前节点。").TextColor(p.InkSubtle)
				return
			}
			ui.Text(c, proxy.Now).FontSize(17).Bold().SingleLine()
			ui.Textf(c, "%s · %s", group, proxy.Type).FontSize(11).TextColor(p.InkSubtle)
			ui.Row(c).Gap(8).Children(func() {
				if acrylic.Button(c, p, "测延迟", false).Clicked() {
					a.testDelay(proxy.Now)
				}
			})
		})
	})

	if hist := a.trafficSeries(120, true); len(hist) > 1 {
		history := a.panel(c, p, opacity).Padding(a.cardPad())
		history.Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "流量历史").Bold()
				ui.Text(c, "最近 1 小时采样，每 30 秒一点").FontSize(11).TextColor(p.InkSubtle)
			})
			widgets.Sparkline(c, p.Accent.Alpha(0.8), hist, 48)
		})
	}

	connectivity := a.panel(c, p, opacity).Padding(a.cardPad())
	connectivity.Children(func() {
		results, testing := a.siteSnapshot()
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "连通性").Bold()
			ui.Text(c, "常用站点可达性").FontSize(11).TextColor(p.InkSubtle)
			ui.Spacer(c)
			label := "一键测试"
			if testing {
				label = "测试中…"
			}
			if acrylic.Button(c, p, label, testing).Clicked() {
				a.testSites()
			}
		})
		ui.Text(c, "经过当前 mixed 端口；HTTP 状态码低于 500 视为可达。").FontSize(11).TextColor(p.InkSubtle)
		if len(results) == 0 {
			ui.Text(c, "尚未测试。点击「一键测试」检查 Google、YouTube、ChatGPT、GitHub 等站点。").TextColor(p.InkMuted)
			return
		}
		ui.Grid(c).Columns(3).Gap(6).Children(func() {
			for _, r := range results {
				r := r
				ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(6, 8).Radius(7).
					Background(p.WashChip).Children(func() {
					ui.Text(c, r.Name).FontSize(12).Bold().SingleLine().Grow(1)
					if testing {
						ui.Text(c, "…").TextColor(p.InkSubtle)
					} else if r.OK {
						ui.Textf(c, "%d ms", r.MS).FontSize(11).Bold().TextColor(p.Success)
					} else {
						ui.Text(c, "不可达").FontSize(11).TextColor(p.Danger)
					}
				})
			}
		})
	})
}

func (a *App) welcome(c *ui.Context, p acrylic.Palette, opacity float64) {
	ui.Column(c).Height(620).Center().Children(func() {
		card := a.panel(c, p, opacity).Width(560).Padding(32).Gap(14)
		card.Children(func() {
			ui.Icon(c, icons.Layers).FontSize(44).TextColor(p.Accent)
			ui.Text(c, "欢迎使用 myclash").FontSize(26).Bold()
			ui.Text(c, "一个原生、轻量的 mihomo 桌面客户端。三步开始：").TextColor(p.InkMuted)
			for i, step := range []string{
				"添加订阅或导入本地 mihomo YAML",
				"选中配置，内核自动启动",
				"到「系统代理」开启，开始使用",
			} {
				ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
					ui.Textf(c, "%02d", i+1).FontSize(12).Bold().TextColor(p.Accent).Width(24)
					ui.Text(c, step).TextColor(p.InkMuted)
				})
			}
			ui.Row(c).Gap(10).Children(func() {
				if acrylic.PrimaryButton(c, p, "添加订阅", false).Clicked() {
					a.router.Push("/profiles")
				}
				if acrylic.Button(c, p, "导入本地 YAML", false).Clicked() {
					a.importProfile()
				}
			})
			ui.Text(c, "配置、连接、规则、日志与 DNS 工具都在左侧。").FontSize(11).TextColor(p.InkSubtle)
		})
	})
}
func (a *App) pageProxiesLegacy(c *ui.Context, p acrylic.Palette) {
	s := a.state.Get()
	opacity := a.cardOpacity()
	ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Gap(2).Children(func() {
			ui.Text(c, "代理").FontSize(24).Bold()
			ui.Textf(c, "%d 个代理组 · 点击节点切换", len(s.Groups)).FontSize(12).TextColor(p.InkSubtle)
		})
		ui.Spacer(c)
		acrylic.SearchField(c, p, &a.proxySearch, "搜索组或节点")
		if acrylic.Button(c, p, "刷新", false).Clicked() {
			a.refreshProxies()
		}
	})
	a.errorBar(c, p)

	groups := sortedGroups(s)
	if a.proxySearch != "" {
		q := strings.ToLower(a.proxySearch)
		filtered := groups[:0]
		for _, g := range groups {
			proxy := s.Proxies[g]
			if strings.Contains(strings.ToLower(g), q) || strings.Contains(strings.ToLower(proxy.Now), q) {
				filtered = append(filtered, g)
			}
		}
		groups = filtered
	}
	if len(groups) == 0 {
		ui.Text(c, "没有匹配的代理组。启动内核并载入包含 proxies 的配置后显示。").TextColor(p.InkMuted)
		return
	}

	for _, group := range groups {
		g := s.Proxies[group]
		card := a.panel(c, p, opacity).Padding(a.cardPad())
		card.Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, group).FontSize(15).Bold().SingleLine()
				acrylic.Chip(c, p, g.Type, p.InkSubtle)
				ui.Spacer(c)
				ui.Text(c, "当前 "+g.Now).FontSize(11).TextColor(p.InkMuted)
			})
			limit := len(g.All)
			if limit > 60 {
				limit = 60
			}
			for _, member := range g.All[:limit] {
				member := member
				row := acrylic.Row(c, p, member == g.Now).Cursor(ui.CursorPointer)
				row.Children(func() {
					ui.Text(c, member).SingleLine()
					ui.Spacer(c)
					if member == g.Now {
						acrylic.Chip(c, p, "当前", p.Success)
					}
					if ms, ok := a.delayFor(member); ok {
						ui.Textf(c, "%d ms", ms).FontSize(11).TextColor(p.InkSubtle)
					}
					if mbps, ok := a.speedFor(member); ok {
						ui.Textf(c, "%.1f Mbps", mbps).FontSize(11).TextColor(p.InkSubtle)
					}
					if acrylic.Button(c, p, "延迟", false).Clicked() {
						a.testDelay(member)
					}
					if acrylic.Button(c, p, "测速", false).Clicked() {
						a.testSpeed(group, member)
					}
					if acrylic.Button(c, p, "选择", false).Clicked() {
						a.selectProxy(group, member)
					}
				})
				if row.Clicked() && member != g.Now {
					a.selectProxy(group, member)
				}
			}
			if len(g.All) > limit {
				ui.Textf(c, "另有 %d 个节点未显示", len(g.All)-limit).FontSize(11).TextColor(p.InkSubtle)
			}
		})
	}
}

func sortedGroups(s core.Snapshot) []string {
	groups := append([]string(nil), s.Groups...)
	sort.Strings(groups)
	return groups
}
func (a *App) pageProfiles(c *ui.Context, p acrylic.Palette) {
	if a.baseUID != "" {
		a.pageProfileEditor(c, p)
		return
	}
	opacity := a.cardOpacity()
	ui.Column(c).Gap(2).Children(func() {
		ui.Text(c, "配置").FontSize(24).Bold()
		ui.Text(c, "订阅与本地 YAML；启用会原子切换内核配置").FontSize(12).TextColor(p.InkSubtle)
	})
	a.errorBar(c, p)

	form := a.panel(c, p, a.cardOpacity()).Padding(a.cardPad())
	form.Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "添加订阅").Bold()
			ui.Spacer(c)
			if acrylic.Button(c, p, "导入本地 YAML", false).Clicked() {
				a.importProfile()
			}
		})
		ui.TextInput(c, &a.subURL).Placeholder("https://example.com/sub").Width(520)
		if acrylic.PrimaryButton(c, p, "添加", a.isBusy()).Clicked() {
			a.addSubscription()
		}
	})

	list := a.profiles.List()
	selected := a.profiles.Selected()
	if len(list) == 0 {
		ui.Text(c, "还没有配置。添加订阅或导入本地 YAML 后，点「启用」开始代理。").TextColor(p.InkMuted)
		return
	}
	for _, prof := range list {
		prof := prof
		current := prof.UID == selected
		card := a.panel(c, p, opacity).Padding(a.cardPad())
		card.Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, prof.Name).FontSize(16).Bold().SingleLine()
				acrylic.Chip(c, p, sourceLabel(prof.Source), p.InkSubtle)
				if current {
					acrylic.Chip(c, p, "已启用", p.Success)
				}
				ui.Spacer(c)
				if prof.Source == "subscription" {
					if acrylic.Button(c, p, "编辑 YAML", false).Clicked() {
						a.openProfileEditor(prof.UID)
					}
					if acrylic.Button(c, p, "同步", a.isBusy()).Clicked() {
						a.syncProfile(prof.UID)
					}
				}
				if !current {
					if acrylic.Button(c, p, "重命名", false).Clicked() {
						a.beginRename(prof.UID, prof.Name)
					}
					if acrylic.Button(c, p, "复制", false).Clicked() {
						a.duplicateProfile(prof.UID)
					}
					if acrylic.Button(c, p, "启用", false).Clicked() {
						a.selectProfile(prof.UID)
					}
					if acrylic.Button(c, p, "删除", false).Clicked() {
						a.deleteProfile(prof.UID)
					}
				}
			})
			if a.renameUID == prof.UID {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.TextInput(c, &a.renameText).Width(260)
					if acrylic.PrimaryButton(c, p, "保存", false).Clicked() {
						a.saveRename()
					}
					if acrylic.Button(c, p, "取消", false).Clicked() {
						a.renameUID = ""
					}
				})
			}
			ui.Textf(c, "更新于 %s", prof.UpdatedAt.Format("2006-01-02 15:04")).FontSize(11).TextColor(p.InkSubtle)
			if u := prof.Userinfo; u != nil && u.Total > 0 {
				used := u.Upload + u.Download
				ui.Progress(c, float64(used)/float64(u.Total)).Height(6).Width(360)
				ui.Textf(c, "已用 %s / %s", format.Bytes(used), format.Bytes(u.Total)).FontSize(11).TextColor(p.InkMuted)
			}
		})
	}
}
func (a *App) pageSettings(c *ui.Context, p acrylic.Palette) {

	ui.Column(c).Gap(2).Children(func() {
		ui.Text(c, "设置").FontSize(24).Bold()
		ui.Text(c, "外观、内核与系统集成").FontSize(12).TextColor(p.InkSubtle)
	})
	a.errorBar(c, p)

	ui.Row(c).Fill().Gap(16).AlignItems(ui.Stretch).Children(func() {
		nav := ui.Column(c).Width(176).Shrink(0).Gap(2)
		nav.Children(func() {
			sections := []struct{ id, label string }{
				{"general", "常规"}, {"appearance", "外观"}, {"kernel", "内核"},
				{"system", "系统"}, {"advanced", "高级内核"}, {"tray", "托盘"},
				{"hotkeys", "热键"}, {"update", "更新"}, {"data", "数据"}, {"about", "关于"},
			}
			for _, s := range sections {
				s := s
				row := acrylic.Row(c, p, a.settingsSection == s.id)
				row.Children(func() { ui.Text(c, s.label) })
				if row.Clicked() {
					a.settingsSection = s.id
				}
			}
		})

		form := a.panel(c, p, a.cardOpacity()).Padding(a.cardPad()).Grow(1)
		form.Children(func() { a.settingsForm(c, p) })
	})
}

func (a *App) settingsForm(c *ui.Context, p acrylic.Palette) {
	switch a.settingsSection {
	case "appearance":
		ui.Text(c, "外观").FontSize(15).Bold()
		ui.Text(c, "材质").FontSize(11).TextColor(p.InkSubtle)
		seg := acrylic.SegmentedText(c, p, &a.presetIdx, "液态玻璃", "透明", "实色")
		if seg.Changed() {
			a.applyAppearance()
		}
		ui.Text(c, "液态玻璃模糊窗后内容；透明直接显示桌面；实色关闭透明，适合远程桌面。").
			FontSize(11).TextColor(p.InkSubtle)
		if acrylic.Checkbox(c, p, &a.monochromeIcons, "侧栏图标使用单色").Changed() {
			if err := a.settings.Update(func(s *settings.Settings) { s.UI.MonochromeIcons = a.monochromeIcons }); err != nil {
				a.setError(err)
			}
			a.invalidate()
		}
		if presets[clampIdx(a.presetIdx, len(presets))] != "solid" {
			changed := false
			ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "面板不透明度").FontSize(11).TextColor(p.InkSubtle).Width(120)
				if ui.Slider(c, &a.opacity, 0, 1).Width(260).Changed() {
					changed = true
				}
				ui.Textf(c, "%.0f%%", a.opacity*100).FontSize(11).TextColor(p.InkMuted).Width(48)
			})
			if changed {
				a.applyAppearance()
			}
		}
		ui.Text(c, "主题").FontSize(11).TextColor(p.InkSubtle)
		theme := acrylic.SegmentedText(c, p, &a.themeIdx, "跟随系统", "浅色", "深色")
		if theme.Changed() {
			a.applyTheme(a.themeIdx)
		}
		ui.Text(c, "密度").FontSize(11).TextColor(p.InkSubtle)
		density := acrylic.SegmentedText(c, p, &a.densityIdx, "舒适", "紧凑")
		if density.Changed() {
			a.applyAppearance()
		}
		zoomChanged := false
		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "界面缩放").FontSize(11).TextColor(p.InkSubtle).Width(120)
			if ui.Slider(c, &a.scale, 1, 1.25).Width(220).Changed() {
				zoomChanged = true
			}
			ui.Textf(c, "%.0f%%", a.scale*100).FontSize(11).TextColor(p.InkMuted).Width(48)
		})
		if zoomChanged {
			a.applyAppearance()
		}

	case "kernel":
		ui.Text(c, "内核").FontSize(15).Bold()
		ui.Row(c).Gap(12).Children(func() {
			ui.TextInput(c, &a.mixed).Label("混合端口").Width(150)
			ui.TextInput(c, &a.controller).Label("Controller 端口").Width(150)
		})
		ui.Text(c, "转发模式").FontSize(11).TextColor(p.InkSubtle)
		seg := acrylic.SegmentedText(c, p, &a.modeIdx, "规则", "全局", "直连")
		if seg.Changed() {
			a.applyMode(a.modeIdx)
		}
		ui.Text(c, "日志级别").FontSize(11).TextColor(p.InkSubtle)
		if lg := acrylic.SegmentedText(c, p, &a.logLevelIdx, "Info", "Debug", "Warning", "Error"); lg.Changed() {
			a.applyLogLevel(a.logLevelIdx)
		}
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			if acrylic.PrimaryButton(c, p, "保存", false).Clicked() {
				a.saveSettings()
			}
			ui.Text(c, "修改端口会重启内核。").FontSize(11).TextColor(p.InkSubtle)
		})

	case "system":
		ui.Text(c, "系统").FontSize(15).Bold()
		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Gap(2).Children(func() {
				ui.Text(c, "系统代理").FontSize(14).Bold()
				ui.Textf(c, "127.0.0.1:%d", a.settings.Get().Core.MixedPort).FontSize(11).TextColor(p.InkSubtle)
			})
			ui.Spacer(c)
			if acrylic.Checkbox(c, p, &a.systemProxy, "启用").Changed() {
				a.toggleSystemProxy(a.systemProxy)
			}
		})
		ui.Text(c, "绕过列表").FontSize(11).TextColor(p.InkSubtle)
		ui.TextInput(c, &a.bypass).Label("分号分隔，例如 <local>;127.*;192.168.*").Width(460)
		if acrylic.Button(c, p, "保存绕过列表", false).Clicked() {
			a.saveBypass()
		}
		ui.Text(c, "测试 URL").FontSize(13).Bold()
		ui.Text(c, "测延迟走 HEAD/204 小文件；测速下载下面这个真实文件。留空使用默认值。").FontSize(11).TextColor(p.InkSubtle)
		ui.TextInput(c, &a.testDelayURL).Label("测延迟 URL").Width(520)
		ui.TextInput(c, &a.testSpeedURL).Label("测速 URL（直链）").Width(520)
		if acrylic.Button(c, p, "保存测试设置", false).Clicked() {
			delayURL := strings.TrimSpace(a.testDelayURL)
			if delayURL == "" {
				delayURL = "https://www.gstatic.com/generate_204"
			}
			speedURL := strings.TrimSpace(a.testSpeedURL)
			if speedURL == "" {
				speedURL = "https://speed.cloudflare.com/__down?bytes=4000000"
			}
			a.testDelayURL, a.testSpeedURL = delayURL, speedURL
			if err := a.settings.Update(func(s *settings.Settings) {
				s.Test.DelayURL = delayURL
				s.Test.SpeedURL = speedURL
			}); err != nil {
				a.setError(err)
			} else {
				a.setStatus("测试 URL 已保存")
			}
		}
		ui.Text(c, "TUN 需要管理员权限；开启后会自动提权重启。").FontSize(11).TextColor(p.InkSubtle)

	case "general":
		ui.Text(c, "常规").FontSize(15).Bold()
		ui.Text(c, "启动、关闭与登录时的应用行为。").FontSize(11).TextColor(p.InkSubtle)
		ui.Text(c, "运行模式").FontSize(11).TextColor(p.InkSubtle)
		if seg := acrylic.SegmentedText(c, p, &a.runModeIdx, "标准（单进程）", "轻量（Agent）"); seg.Changed() {
			mode := "standard"
			if a.runModeIdx == 1 {
				mode = "lightweight"
			}
			if err := a.settings.Update(func(s *settings.Settings) { s.App.RunMode = mode }); err != nil {
				a.setError(err)
			} else {
				a.setStatus("运行模式已保存；关闭窗口后按该模式运行")
			}
		}
		ui.Text(c, "轻量模式关闭窗口后由 Agent 保留内核与托盘；再次打开窗口只附着 Agent，不再启动第二套内核。").
			FontSize(11).TextColor(p.InkSubtle)
		changed := false
		if acrylic.Checkbox(c, p, &a.autoStart, "启动时自动启用上次配置").Changed() {
			changed = true
		}
		if acrylic.Checkbox(c, p, &a.closeTray, "关闭窗口时保留在托盘").Changed() {
			changed = true
		}
		if acrylic.Checkbox(c, p, &a.openAtLogin, "登录时启动 myclash（静默驻留托盘）").Changed() {
			if err := mygo.App.SetOpenAtLogin(a.openAtLogin); err != nil {
				a.setError(err)
			} else {
				changed = true
			}
		}
		if acrylic.Checkbox(c, p, &a.notifications, "窗口不在前台时发送系统通知").Changed() {
			changed = true
		}
		if changed {
			_ = a.settings.Update(func(s *settings.Settings) {
				s.App.AutoStartProfile = a.autoStart
				s.App.CloseToTray = a.closeTray
				s.App.OpenAtLogin = a.openAtLogin
				s.App.Notifications = a.notifications
			})
		}
		ui.Text(c, "轻量模式（M3）：关闭窗口后由后台 Agent 保留内核与托盘。").FontSize(11).TextColor(p.InkSubtle)

	case "data":
		ui.Text(c, "数据").FontSize(15).Bold()
		ui.Text(c, "日志会写到数据目录的 logs/core.log；保留策略只影响这个文件。").FontSize(11).TextColor(p.InkSubtle)
		ui.Text(c, "保留时间").FontSize(11).TextColor(p.InkSubtle)
		acrylic.SegmentedText(c, p, &a.logRetentionIdx, "永久", "1 天", "7 天", "30 天")
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.TextInput(c, &a.logMaxText).Label("单文件上限 MB").Width(150)
			if acrylic.Button(c, p, "应用日志策略", false).Clicked() {
				mb := atoiSafe(a.logMaxText)
				if mb <= 0 {
					mb = 8
				}
				if mb > 1024 {
					mb = 1024
				}
				if err := a.settings.Update(func(s *settings.Settings) {
					s.Logs.Retention = logRetentionValue(a.logRetentionIdx)
					s.Logs.MaxSizeMB = mb
				}); err != nil {
					a.setError(err)
					return
				}
				a.mu.Lock()
				a.logMaxMB = mb
				a.logMaxText = strconv.Itoa(mb)
				a.mu.Unlock()
				a.cleanupLogs()
				a.setStatus("日志策略已应用")
			}
			if acrylic.Button(c, p, "立即清空日志", false).Clicked() {
				a.clearLogs()
				a.setStatus("日志已清空")
			}
		})
		ui.Text(c, "WebDAV 同步").FontSize(13).Bold()
		ui.Text(c, "备份 zip 会 PUT 到该地址；下载后覆盖并重启。密码保存在 settings.yaml，请使用专用应用密码。").
			FontSize(11).TextColor(p.InkSubtle)
		ui.TextInput(c, &a.webdavURL).Label("WebDAV 地址或文件 URL").Width(520)
		ui.Row(c).Gap(10).Children(func() {
			ui.TextInput(c, &a.webdavUser).Label("用户名").Width(220)
			ui.TextInput(c, &a.webdavPass).Label("密码 / 应用密码").Width(220)
		})
		ui.Row(c).Gap(8).Wrap().Children(func() {
			if acrylic.Button(c, p, "保存 WebDAV", false).Clicked() {
				a.saveWebDAV()
			}
			if acrylic.Button(c, p, "上传备份", false).Clicked() {
				a.uploadWebDAV()
			}
			if acrylic.Button(c, p, "从 WebDAV 恢复", false).Clicked() {
				a.downloadWebDAV()
			}
		})
		ui.Textf(c, "配置目录：%s", a.dataDir).FontSize(11).TextColor(p.InkMuted)
		ui.Text(c, "备份包含设置、订阅、覆写与日志；诊断包不包含订阅原始配置。").FontSize(11).TextColor(p.InkSubtle)
		ui.Row(c).Gap(8).Wrap().Children(func() {
			if acrylic.Button(c, p, "打开数据目录", false).Clicked() {
				a.openDataDir()
			}
			if acrylic.Button(c, p, "导出备份", false).Clicked() {
				a.backupData()
			}
			if acrylic.Button(c, p, "导入备份", false).Clicked() {
				a.restoreData()
			}
			if acrylic.Button(c, p, "导出诊断", false).Clicked() {
				a.exportDiagnostics()
			}
			if acrylic.Button(c, p, "重置数据", false).Clicked() {
				a.resetData()
			}
		})

	case "about":
		ui.Text(c, "关于").FontSize(15).Bold()
		ui.Textf(c, "myclash %s · mihomo %s", mygo.App.Version(), kernelVersion(a.state.Get().Version)).FontSize(12).TextColor(p.InkMuted)
		ui.Text(c, "GPL-3.0 · 源码与问题反馈见仓库").FontSize(11).TextColor(p.InkSubtle)
		if acrylic.Button(c, p, "退出 myclash", false).Clicked() {
			a.quitApp()
		}

	case "advanced":
		ui.Text(c, "高级内核").FontSize(15).Bold()
		changed := false
		if acrylic.Checkbox(c, p, &a.unifiedDelay, "unified-delay").Changed() {
			changed = true
		}
		if acrylic.Checkbox(c, p, &a.tcpConcurrent, "tcp-concurrent").Changed() {
			changed = true
		}
		ui.Text(c, "find-process-mode").FontSize(11).TextColor(p.InkSubtle)
		ui.Text(c, "Off 最省 CPU；Strict 只查询需要的连接；Always 每个连接都查询，连接很多时开销更高。").FontSize(11).TextColor(p.InkSubtle)
		fp := acrylic.SegmentedText(c, p, &a.findProcessIdx, "Off", "Strict", "Always")
		if fp.Changed() {
			changed = true
		}
		ui.Row(c).Gap(12).Children(func() {
			ui.TextInput(c, &a.fingerprint).Label("客户端指纹").Width(180)
			ui.TextInput(c, &a.iface).Label("出站网卡").Width(180)
			ui.TextInput(c, &a.routingMark).Label("routing-mark").Width(140)
		})
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			if acrylic.PrimaryButton(c, p, "保存", false).Clicked() {
				changed = true
			}
			if changed {
				ui.Text(c, "保存后重启内核生效。").FontSize(11).TextColor(p.InkSubtle)
			}
		})
		if changed {
			a.applyAdvanced()
		}
	case "tray":
		ui.Text(c, "托盘").FontSize(15).Bold()
		ui.Text(c, "关闭窗口后是否继续驻留，以及托盘菜单里能直接做什么。").FontSize(11).TextColor(p.InkSubtle)
		if acrylic.Checkbox(c, p, &a.closeTray, "关闭窗口时保留在托盘").Changed() {
			if err := a.settings.Update(func(s *settings.Settings) { s.App.CloseToTray = a.closeTray }); err != nil {
				a.setError(err)
			}
		}
		ui.Text(c, "右键托盘图标：显示窗口、当前配置、规则/全局/直连、TUN、重启内核、系统代理、退出。").FontSize(11).TextColor(p.InkMuted)
		ui.Text(c, "左键托盘图标：显示主窗口。退出会先移除图标并停止内核。").FontSize(11).TextColor(p.InkSubtle)
		ui.Row(c).Gap(8).Children(func() {
			if acrylic.Button(c, p, "显示窗口", false).Clicked() && a.win != nil {
				a.win.Show()
				a.win.Focus()
			}
			if acrylic.Button(c, p, "退出 myclash", false).Clicked() {
				a.quitApp()
			}
		})

	case "hotkeys":
		ui.Text(c, "热键").FontSize(15).Bold()
		ui.Text(c, "全局热键：窗口不在前台时也会触发。格式如 CmdOrCtrl+Shift+M，留空即禁用。").
			FontSize(11).TextColor(p.InkSubtle)
		type hotkeyEntry struct {
			title string
			desc  string
			value *string
			def   string
		}
		entries := []hotkeyEntry{
			{"显示窗口", "唤醒并聚焦主窗口", &a.hotkeyShow, "CmdOrCtrl+Shift+M"},
			{"切换模式", "规则 → 全局 → 直连 循环", &a.hotkeyMode, "CmdOrCtrl+Shift+R"},
			{"系统代理", "打开或关闭系统代理", &a.hotkeyProxy, "CmdOrCtrl+Shift+P"},
			{"TUN", "打开或关闭 TUN", &a.hotkeyTUN, "CmdOrCtrl+Shift+T"},
			{"退出应用", "立即退出并释放内核", &a.hotkeyQuit, "CmdOrCtrl+Shift+Q"},
		}
		seen := map[string]string{}
		conflict := ""
		for _, e := range entries {
			e := e
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Column(c).Width(180).Gap(2).Children(func() {
					ui.Text(c, e.title).FontSize(13).Bold()
					ui.Text(c, e.desc).FontSize(11).TextColor(p.InkSubtle).SingleLine()
				})
				ui.TextInput(c, e.value).Width(250)
				if acrylic.Button(c, p, "默认", false).Clicked() {
					*e.value = e.def
				}
				if acrylic.Button(c, p, "清除", false).Clicked() {
					*e.value = ""
				}
			})
			v := strings.ToLower(strings.TrimSpace(*e.value))
			if v == "" {
				continue
			}
			if prev, ok := seen[v]; ok {
				conflict = fmt.Sprintf("%s 与 %s 使用了同一个组合：%s", prev, e.title, *e.value)
			} else {
				seen[v] = e.title
			}
		}
		if conflict != "" {
			ui.Text(c, "冲突："+conflict).FontSize(11).TextColor(p.Danger)
		}
		ui.Text(c, "保存后立即重新注册；如果组合已被系统或其他应用占用，该项不会生效，可换一个组合。").
			FontSize(11).TextColor(p.InkSubtle)
		if acrylic.PrimaryButton(c, p, "保存并注册", false).Clicked() {
			if conflict != "" {
				a.setError(errors.New(conflict))
				return
			}
			if err := a.settings.Update(func(s *settings.Settings) {
				s.Hotkeys.Show = a.hotkeyShow
				s.Hotkeys.SystemProxy = a.hotkeyProxy
				s.Hotkeys.TUN = a.hotkeyTUN
				s.Hotkeys.Mode = a.hotkeyMode
				s.Hotkeys.Quit = a.hotkeyQuit
			}); err != nil {
				a.setError(err)
				return
			}
			a.registerHotkeys()
			a.setStatus("热键已注册")
		}

	case "update":
		ui.Text(c, "更新").FontSize(15).Bold()
		ui.Text(c, "应用与内核随安装包更新；运行新安装包覆盖安装即可，配置会保留。").FontSize(11).TextColor(p.InkSubtle)

	}
}
func (a *App) isBusy() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.busy
}

func (a *App) applyTheme(idx int) {
	modes := []string{"system", "light", "dark"}
	if idx < 0 || idx >= len(modes) {
		return
	}
	mode := modes[idx]
	if err := a.settings.Update(func(s *settings.Settings) { s.UI.Theme = mode }); err != nil {
		a.setError(err)
		return
	}
	acrylic.SetAppearance(mode)
	a.dark = mode == "dark" || (mode == "system" && mygo.Theme.IsDark())
	a.invalidate()
}

func (a *App) pageOverride(c *ui.Context, p acrylic.Palette) {
	current, ok := a.profiles.CurrentMeta()
	ui.Column(c).Gap(2).Children(func() {
		ui.Text(c, "覆写").FontSize(24).Bold()
		ui.Text(c, "YAML 合并：映射深合并、列表前插/后插、delete 点路径；JavaScript：function main(config) 返回新 config").FontSize(12).TextColor(p.InkSubtle)
	})
	if !ok {
		ui.Text(c, "先添加并启用一个配置。").TextColor(p.InkMuted)
		return
	}
	if a.override == nil || a.overrideUID != current.UID {
		data, _ := a.profiles.Override(current.UID)
		buf := editor.NewBuffer(string(data))
		buf.SetLanguage(editor.LanguageFor("override.yaml"))
		a.override = &editor.State{Buffer: buf}
		a.overrideUID = current.UID
	}
	if a.overrideScript == nil || a.overrideScriptUID != current.UID {
		data, _ := a.profiles.ScriptOverride(current.UID)
		buf := editor.NewBuffer(string(data))
		buf.SetLanguage(editor.LanguageFor("override.js"))
		a.overrideScript = &editor.State{Buffer: buf}
		a.overrideScriptUID = current.UID
	}
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		acrylic.SegmentedText(c, p, &a.overrideMode, "YAML 合并", "JavaScript")
	})
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Textf(c, "当前配置：%s", current.Name).FontSize(11).TextColor(p.InkSubtle)
		ui.Spacer(c)
		if acrylic.PrimaryButton(c, p, "保存并重载", false).Clicked() {
			if a.overrideMode == 1 {
				a.saveScriptOverride()
			} else {
				a.saveOverride()
			}
		}
		if acrylic.Button(c, p, "清空", false).Clicked() {
			if a.overrideMode == 1 {
				a.overrideScript.Buffer.Set("")
				a.saveScriptOverride()
			} else {
				a.override.Buffer.Set("")
				a.saveOverride()
			}
		}
	})
	a.errorBar(c, p)
	ui.Column(c).Height(520).Background(p.Code).Radius(8).Padding(10).Clip().Children(func() {
		view := a.override
		if a.overrideMode == 1 {
			view = a.overrideScript
		}
		editor.View(c, view).Fill()
	})
}

func rateSeries(h []core.RateSample, down bool) []float64 {
	out := make([]float64, 0, len(h))
	for _, v := range h {
		if down {
			out = append(out, float64(v.Down))
		} else {
			out = append(out, float64(v.Up))
		}
	}
	return out
}

func (a *App) currentNode(s core.Snapshot) (string, core.Proxy, bool) {
	a.mu.Lock()
	group, node := a.proxyGroup, a.proxyNode
	a.mu.Unlock()
	if group != "" {
		if p, ok := s.Proxies[group]; ok {
			if node == "" || node == p.Now {
				if p.Now != "" {
					return group, p, true
				}
			} else {
				for _, member := range p.All {
					if member == node {
						p.Now = node
						return group, p, true
					}
				}
			}
		}
	}
	if p, ok := s.Proxies["GLOBAL"]; ok && realNode(p.Now) {
		return "GLOBAL", p, true
	}
	for _, g := range s.Groups {
		if p, ok := s.Proxies[g]; ok && realNode(p.Now) {
			return g, p, true
		}
	}
	if p, ok := s.Proxies["GLOBAL"]; ok && p.Now != "" {
		return "GLOBAL", p, true
	}
	return "", core.Proxy{}, false
}

func realNode(name string) bool {
	switch name {
	case "", "DIRECT", "REJECT", "REJECT-DROP", "PASS":
		return false
	}
	return true
}
func kernelVersion(core string) string {
	if v := mihomoVersion(); v != "" {
		return v
	}
	return shortVersion(core)
}

func shortVersion(v string) string {
	if v == "" {
		return "未运行"
	}
	if len(v) > 12 {
		return v[:12]
	}
	return v
}

func uptime(start time.Time) string {
	if start.IsZero() {
		return "—"
	}
	return time.Since(start).Round(time.Second).String()
}

func sourceLabel(source string) string {
	if source == "subscription" {
		return "订阅"
	}
	return "本地"
}
