package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"

	"myclash/internal/core"
	"myclash/internal/ui/acrylic"
)

// pageProxies shows every group as a panel and every member as a compact
// card. A card selects the node; the group header tests or sorts the whole
// group instead of making the user click each member.
func (a *App) pageProxies(c *ui.Context, p acrylic.Palette) {
	s := a.state.Get()
	opacity := a.cardOpacity()
	ui.Row(c).Gap(10).AlignItems(ui.Center).Wrap().Children(func() {
		ui.Column(c).Gap(2).Children(func() {
			ui.Text(c, "代理").FontSize(24).Bold()
			ui.Textf(c, "%d 个代理组 · 点击节点卡片切换", len(s.Groups)).FontSize(12).TextColor(p.InkSubtle)
		})
		ui.Spacer(c)
		acrylic.SearchField(c, p, &a.proxySearch, "搜索组或节点")
		sortSeg := acrylic.SegmentedText(c, p, &a.proxySortIdx, "名称", "延迟", "测速")
		_ = sortSeg
		colSeg := acrylic.SegmentedText(c, p, &a.proxyColsIdx, "2列", "3列", "4列", "自动")
		_ = colSeg
		if acrylic.Button(c, p, "刷新", false).Clicked() {
			a.refreshProxies()
		}
		if acrylic.Button(c, p, "定位当前", false).Clicked() {
			if group, _, ok := a.currentNode(s); ok {
				a.proxyExpanded[group] = true
				a.scrollProxyGroup = group
				a.invalidate()
			}
		}
		if acrylic.Button(c, p, "全部测延迟", false).Clicked() {
			for _, group := range proxyGroups(s) {
				a.testGroupDelay(group)
			}
		}
	})
	a.errorBar(c, p)

	groups := proxyGroups(s)
	if a.proxyLastSearch != a.proxySearch {
		a.proxyLastSearch = a.proxySearch
		if a.proxySearch != "" {
			for _, g := range groups {
				a.proxyExpanded[g] = true
			}
		}
	}
	if a.proxySearch != "" {
		q := strings.ToLower(a.proxySearch)
		filtered := groups[:0]
		for _, g := range groups {
			proxy := s.Proxies[g]
			match := strings.Contains(strings.ToLower(g), q) || strings.Contains(strings.ToLower(proxy.Now), q)
			if !match {
				for _, member := range proxy.All {
					if strings.Contains(strings.ToLower(member), q) {
						match = true
						break
					}
				}
			}
			if match {
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
		expanded := a.proxyExpanded[group]
		card := a.panel(c, p, opacity).Padding(a.cardPad())
		if a.scrollProxyGroup == group {
			a.scrollProxyGroup = ""
			card = card.ScrollIntoView(ui.ScrollIntoViewOptions{Block: ui.ScrollCenter, Animation: &ui.Smooth})
		}
		card.Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Wrap().Children(func() {
				label := "展开"
				if expanded {
					label = "收起"
				}
				if acrylic.Button(c, p, label, false).Clicked() {
					a.proxyExpanded[group] = !expanded
				}
				ui.Text(c, group).FontSize(15).Bold().SingleLine()
				acrylic.Chip(c, p, g.Type, p.InkSubtle)
				acrylic.Chip(c, p, fmt.Sprintf("%d 节点", len(g.All)), p.InkSubtle)
				ui.Spacer(c)
				ui.Text(c, "当前 "+g.Now).FontSize(11).TextColor(p.InkMuted).SingleLine()
				if acrylic.Button(c, p, "统一测延迟", false).Clicked() {
					a.testGroupDelay(group)
				}
				if acrylic.Button(c, p, "统一测速", false).Clicked() {
					a.testGroupSpeed(group)
				}
			})
			if !expanded {
				return
			}
			members := sortedMembers(a, s, group, a.proxySortIdx)
			limit := len(members)
			if limit > 60 {
				limit = 60
			}
			ui.Grid(c).Columns(a.proxyColumns()).Gap(8).Children(func() {
				for _, member := range members[:limit] {
					a.proxyNodeCard(c, p, s, group, member, member == g.Now)
				}
			})
			if len(members) > limit {
				ui.Textf(c, "另有 %d 个节点未显示", len(members)-limit).FontSize(11).TextColor(p.InkSubtle)
			}
		})
	}
}

func (a *App) proxyNodeCard(c *ui.Context, p acrylic.Palette, s core.Snapshot, group, member string, current bool) {
	info := s.Proxies[member]
	t := c.Theme()
	node := ui.Column(c).FillWidth().MinHeight(104).Padding(t.Space(3)).Gap(4).Radius(10).
		Background(p.WashChip).Border(1, p.Hairline).Cursor(ui.CursorPointer)
	if current {
		node.Background(p.Selection).Border(1, p.Accent)
	}
	node.Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, member).FontSize(12).Bold().SingleLine().Grow(1)
			if current {
				acrylic.Chip(c, p, "当前", p.Success)
			}
		})
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			typ := info.Type
			if typ == "" {
				typ = "节点"
			}
			ui.Text(c, typ).FontSize(10).TextColor(p.InkSubtle).SingleLine()
		})
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			if ms, ok := a.delayFor(member); ok {
				ui.Textf(c, "%d ms", ms).FontSize(10).TextColor(p.InkMuted)
			} else if msg, ok := a.delayErrFor(member); ok {
				ui.Text(c, msg).FontSize(10).TextColor(p.Danger).SingleLine()
			} else {
				ui.Text(c, "未测延迟").FontSize(10).TextColor(p.InkSubtle)
			}
			if mbps, ok := a.speedFor(member); ok {
				ui.Textf(c, "%.1f Mbps", mbps).FontSize(10).TextColor(p.InkMuted)
			}
		})
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			if acrylic.Button(c, p, "延迟", false).Clicked() {
				a.testDelay(member)
			}
			if acrylic.Button(c, p, "测速", false).Clicked() {
				a.testSpeed(group, member)
			}
			ui.Spacer(c)
			if !current {
				if acrylic.Button(c, p, "切换", false).Clicked() {
					a.selectProxy(group, member)
				}
			}
		})
	})
	if node.Clicked() {
		if current {
			a.focusProxy(group, member)
		} else {
			a.selectProxy(group, member)
		}
	}
}

func proxyGroups(s core.Snapshot) []string {
	groups := append([]string(nil), s.Groups...)
	sort.Strings(groups)
	return groups
}

func sortedMembers(a *App, s core.Snapshot, group string, by int) []string {
	members := append([]string(nil), s.Proxies[group].All...)
	switch by {
	case 1:
		sort.SliceStable(members, func(i, j int) bool {
			mi, iok := a.delayFor(members[i])
			mj, jok := a.delayFor(members[j])
			if iok != jok {
				return iok
			}
			if mi != mj {
				return mi < mj
			}
			return members[i] < members[j]
		})
	case 2:
		sort.SliceStable(members, func(i, j int) bool {
			mi, iok := a.speedFor(members[i])
			mj, jok := a.speedFor(members[j])
			if iok != jok {
				return iok
			}
			if mi != mj {
				return mi > mj
			}
			return members[i] < members[j]
		})
	default:
		sort.Strings(members)
	}
	return members
}

func (a *App) proxyColumns() int {
	switch a.proxyColsIdx {
	case 0:
		return 2
	case 1:
		return 3
	case 2:
		return 3
	}
	if a.win == nil {
		return 3
	}
	w, _ := a.win.Size()
	scale := a.settings.Get().UI.Scale
	if scale < 1 {
		scale = 1
	}
	content := float64(w)/scale - 260
	cols := int(content / 300)
	if cols < 2 {
		cols = 2
	}
	if cols > 3 {
		cols = 3
	}
	return cols
}
