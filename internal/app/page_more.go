package app

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"myclash/internal/core"
	"myclash/internal/ui/acrylic"
	"myclash/internal/ui/format"
)

var connColumns = []ui.TableColumn{
	{Title: "主机", Width: 0},
	{Title: "进程", Width: 120},
	{Title: "规则", Width: 0},
	{Title: "链路", Width: 0},
	{Title: "实时 ↑/↓", Width: 130, Align: ui.End},
	{Title: "上传", Width: 78, Align: ui.End},
	{Title: "下载", Width: 78, Align: ui.End},
	{Title: "时长", Width: 70, Align: ui.End},
	{Title: "操作", Width: 64, Align: ui.End},
}

func (a *App) pageConnections(c *ui.Context, p acrylic.Palette) {
	s := a.state.Get()
	a.header(c, p, "连接", "实时会话、规则与流量")
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		acrylic.SearchField(c, p, &a.connSearch, "搜索主机、进程、规则或链路")
		ui.Spacer(c)
		label := "暂停"
		if a.connPaused {
			label = "继续"
		}
		if acrylic.Button(c, p, label, false).Clicked() {
			a.toggleConnPaused()
		}
		if acrylic.Button(c, p, "关闭全部", !s.Started).Clicked() {
			a.closeAllConnections()
		}
	})
	a.errorBar(c, p)

	var rows []core.Connection
	if a.connPaused {
		rows = filterConnections(a.frozenConns, a.connSearch)
	} else {
		rows = filterConnections(s.Connections, a.connSearch)
	}
	sortConnections(a, rows, a.connSortIdx)
	ui.Row(c).Gap(12).AlignItems(ui.Center).Wrap().Children(func() {
		ui.Text(c, "排序").FontSize(11).TextColor(p.InkSubtle)
		acrylic.SegmentedText(c, p, &a.connSortIdx, "时间", "上传", "下载", "速率")
		ui.Spacer(c)
		ui.Textf(c, "连接 %d · 累计 ↑ %s / ↓ %s", len(rows), format.Bytes(s.ConnUpload), format.Bytes(s.ConnDown)).
			FontSize(11).TextColor(p.InkMuted)
	})
	if len(rows) == 0 {
		ui.Text(c, "当前没有匹配的连接。").TextColor(p.InkMuted)
		return
	}
	processes := false
	for _, c := range rows {
		if connProcess(c) != "—" {
			processes = true
			break
		}
	}
	if !processes {
		ui.Text(c, "进程列为空：到 设置 → 高级内核 把 find-process-mode 设为 Always（会增加连接查找开销）。").FontSize(11).TextColor(p.InkSubtle)
	}
	ui.Column(c).Height(520).Background(p.Code).Radius(8).Padding(10).Clip().Children(func() {
		ui.Table(c, &a.connState, connColumns, len(rows), func(row, col int) {
			conn := rows[row]
			switch col {
			case 0:
				ui.Text(c, connHost(conn)).SingleLine()
			case 1:
				ui.Text(c, connProcess(conn)).SingleLine()
			case 2:
				ui.Text(c, conn.Rule).SingleLine()
			case 3:
				ui.Text(c, strings.Join(conn.Chains, " → ")).SingleLine()
			case 4:
				up, down := a.connRateFor(conn.ID)
				ui.Textf(c, "↑%s ↓%s", format.Rate(int64(up)), format.Rate(int64(down))).FontSize(11).SingleLine()
			case 5:
				ui.Text(c, format.Bytes(conn.Upload)).SingleLine()
			case 6:
				ui.Text(c, format.Bytes(conn.Download)).SingleLine()
			case 7:
				ui.Text(c, time.Since(conn.Start).Round(time.Second).String()).SingleLine()
			case 8:
				if acrylic.Button(c, p, "关闭", false).Clicked() {
					a.closeConnection(conn.ID)
				}
			}
		}).Grow(1)
	})
}

func connProcess(c core.Connection) string {
	if c.Metadata.Process != "" {
		return c.Metadata.Process
	}
	if c.Metadata.ProcessPath != "" {
		return filepath.Base(c.Metadata.ProcessPath)
	}
	return "—"
}

func sortConnections(a *App, rows []core.Connection, by int) {
	switch by {
	case 1:
		sort.SliceStable(rows, func(i, j int) bool { return rows[i].Upload > rows[j].Upload })
	case 2:
		sort.SliceStable(rows, func(i, j int) bool { return rows[i].Download > rows[j].Download })
	case 3:
		rate := func(c core.Connection) float64 {
			up, down := a.connRateFor(c.ID)
			return up + down
		}
		sort.SliceStable(rows, func(i, j int) bool { return rate(rows[i]) > rate(rows[j]) })
	default:
		sort.SliceStable(rows, func(i, j int) bool { return rows[i].Start.After(rows[j].Start) })
	}
}

var ruleColumns = []ui.TableColumn{
	{Title: "#", Width: 52, Align: ui.End},
	{Title: "类型", Width: 150},
	{Title: "内容", Width: 0},
	{Title: "策略", Width: 150},
	{Title: "命中", Width: 70, Align: ui.End},
	{Title: "操作", Width: 80, Align: ui.End},
}

func (a *App) pageRules(c *ui.Context, p acrylic.Palette) {
	s := a.state.Get()
	a.header(c, p, "规则", "按匹配顺序列出当前生效规则")
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		acrylic.SearchField(c, p, &a.ruleSearch, "搜索类型、内容或策略")
		ui.Spacer(c)
		if acrylic.Button(c, p, "刷新", !s.Started).Clicked() {
			a.refreshRules()
		}
	})
	a.errorBar(c, p)

	rows := filterRules(s.Rules, a.ruleSearch)
	if len(rows) == 0 {
		ui.Text(c, "没有匹配的规则。").TextColor(p.InkMuted)
		return
	}
	ui.Column(c).Height(520).Background(p.Code).Radius(8).Padding(10).Clip().Children(func() {
		ui.Table(c, &a.ruleState, ruleColumns, len(rows), func(row, col int) {
			rule := rows[row]
			switch col {
			case 0:
				ui.Textf(c, "%d", rule.Index+1)
			case 1:
				ui.Text(c, rule.Type).SingleLine()
			case 2:
				ui.Text(c, rule.Payload).SingleLine()
			case 3:
				ui.Text(c, rule.Proxy).SingleLine()
			case 4:
				if rule.Extra != nil {
					ui.Textf(c, "%d", rule.Extra.HitCount)
				} else {
					ui.Text(c, "—")
				}
			case 5:
				disabled := rule.Extra != nil && rule.Extra.Disabled
				label := "禁用"
				if disabled {
					label = "启用"
				}
				if acrylic.Button(c, p, label, false).Clicked() {
					a.setRuleDisabled(rule.Index, !disabled)
				}
			}
		}).Grow(1)
	})
}

func (a *App) pageLogs(c *ui.Context, p acrylic.Palette) {
	s := a.state.Get()
	logs := s.Logs
	if a.logsPaused {
		logs = a.frozenLogs
	}
	a.header(c, p, "日志", "内核实时日志")
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		acrylic.SearchField(c, p, &a.logSearch, "过滤日志")
		ui.Spacer(c)
		pause := "暂停"
		if a.logsPaused {
			pause = "继续"
		}
		if acrylic.Button(c, p, pause, false).Clicked() {
			a.toggleLogsPaused()
		}
		if acrylic.Button(c, p, "清空", false).Clicked() {
			a.clearLogs()
			a.frozenLogs = nil
		}
		if acrylic.Button(c, p, "复制", false).Clicked() {
			var b strings.Builder
			for _, e := range filterLogs(logs, a.logSearch) {
				fmt.Fprintf(&b, "%-5s %s\n", e.Type, e.Payload)
			}
			mygo.Clipboard.WriteText(b.String())
		}
	})
	a.errorBar(c, p)

	rows := filterLogs(logs, a.logSearch)
	if len(rows) > 600 {
		rows = rows[len(rows)-600:]
	}
	if len(rows) == 0 {
		ui.Text(c, "暂无日志。").TextColor(p.InkMuted)
		return
	}
	a.logState.FollowEnd = !a.logsPaused
	ui.Column(c).Height(520).Background(p.Code).Radius(8).Padding(10).Clip().Children(func() {
		ui.List(c, &a.logState, len(rows), func(i int) {
			e := rows[i]
			ui.Textf(c, "%-5s %s", e.Type, e.Payload).Font("monospace").FontSize(12).SingleLine()
		}).Grow(1)
	})
}

func (a *App) refreshRules() {
	a.mu.Lock()
	client := a.client
	a.mu.Unlock()
	if client == nil {
		return
	}
	go func() {
		rules, err := client.Rules(a.ctx)
		if err != nil {
			a.setError(err)
			return
		}
		a.win.Update(func() { a.state.SetRules(rules) })
	}()
}

func (a *App) closeAllConnections() {
	a.mu.Lock()
	client := a.client
	a.mu.Unlock()
	if client == nil {
		return
	}
	go func() {
		if err := client.CloseAllConnections(a.ctx); err != nil {
			a.setError(err)
		}
	}()
}

func (a *App) toggleLogsPaused() {
	if a.logsPaused {
		a.logsPaused = false
		a.frozenLogs = nil
		return
	}
	a.frozenLogs = append([]core.LogEntry(nil), a.state.Get().Logs...)
	a.logsPaused = true
}

func filterConnections(in []core.Connection, q string) []core.Connection {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return in
	}
	out := make([]core.Connection, 0, len(in))
	for _, c := range in {
		hay := strings.ToLower(c.Metadata.Host + " " + c.Metadata.Process + " " + c.Metadata.ProcessPath + " " + c.Metadata.SourceIP + " " + c.Metadata.DestinationIP + " " + c.Rule + " " + strings.Join(c.Chains, " "))
		if strings.Contains(hay, q) {
			out = append(out, c)
		}
	}
	return out
}

func filterRules(in []core.Rule, q string) []core.Rule {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return append([]core.Rule(nil), in...)
	}
	out := make([]core.Rule, 0, len(in))
	for _, r := range in {
		hay := strings.ToLower(r.Type + " " + r.Payload + " " + r.Proxy)
		if strings.Contains(hay, q) {
			out = append(out, r)
		}
	}
	return out
}

func filterLogs(in []core.LogEntry, q string) []core.LogEntry {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return in
	}
	out := make([]core.LogEntry, 0, len(in))
	for _, e := range in {
		if strings.Contains(strings.ToLower(e.Payload), q) || strings.Contains(strings.ToLower(e.Type), q) {
			out = append(out, e)
		}
	}
	return out
}

func connHost(c core.Connection) string {
	if c.Metadata.Host != "" {
		return c.Metadata.Host
	}
	if c.Metadata.DestinationIP != "" {
		return c.Metadata.DestinationIP
	}
	return c.Metadata.ProcessPath
}
