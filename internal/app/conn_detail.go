package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"myclash/internal/core"
	"myclash/internal/ui/acrylic"
	"myclash/internal/ui/format"
)

func (a *App) connectionDetailModal(c *ui.Context, p acrylic.Palette) {
	acrylic.Modal(c, p, &a.connDetailOpen, func() {
		conn := a.connDetail
		up, down := a.connRateFor(conn.ID)
		ui.Column(c).Width(620).Gap(6).Children(func() {
			ui.Text(c, "连接详情").FontSize(16).Bold()
			ui.Text(c, connHost(conn)).FontSize(13).Bold()
			ui.Textf(c, "%s → %s", sourceAddress(conn), destinationAddress(conn)).FontSize(11).TextColor(p.InkSubtle)
			ui.Divider(c)
			detailRow(c, p, "进程", connProcess(conn))
			if conn.Metadata.ProcessPath != "" {
				detailRow(c, p, "进程路径", conn.Metadata.ProcessPath)
			}
			detailRow(c, p, "网络/类型", strings.TrimSpace(conn.Metadata.Network+" "+conn.Metadata.Type))
			detailRow(c, p, "规则", conn.Rule)
			detailRow(c, p, "策略链", strings.Join(conn.Chains, " → "))
			detailRow(c, p, "上传", format.Bytes(conn.Upload))
			detailRow(c, p, "下载", format.Bytes(conn.Download))
			detailRow(c, p, "实时", fmt.Sprintf("↑ %s  ↓ %s", format.Rate(int64(up)), format.Rate(int64(down))))
			detailRow(c, p, "持续时间", time.Since(conn.Start).Round(time.Second).String())
			detailRow(c, p, "连接 ID", conn.ID)
			ui.Row(c).Gap(8).Children(func() {
				if acrylic.Button(c, p, "复制详情", false).Clicked() {
					mygo.Clipboard.WriteText(connectionText(conn, up, down))
				}
				if acrylic.Button(c, p, "关闭连接", false).Clicked() {
					a.closeConnection(conn.ID)
					a.connDetailOpen = false
				}
				ui.Spacer(c)
				if acrylic.PrimaryButton(c, p, "关闭", false).Clicked() {
					a.connDetailOpen = false
				}
			})
		})
	})
}

func detailRow(c *ui.Context, p acrylic.Palette, label, value string) {
	ui.Row(c).Gap(12).AlignItems(ui.Start).Children(func() {
		ui.Text(c, label).FontSize(11).TextColor(p.InkSubtle).Width(90)
		ui.Text(c, value).FontSize(11).TextColor(p.Ink)
	})
}

func sourceAddress(c core.Connection) string {
	if c.Metadata.SourceIP == "" {
		return "—"
	}
	return c.Metadata.SourceIP + ":" + c.Metadata.SourcePort
}

func destinationAddress(c core.Connection) string {
	if c.Metadata.DestinationIP == "" {
		return "—"
	}
	return c.Metadata.DestinationIP + ":" + c.Metadata.DestinationPort
}

func connectionText(c core.Connection, upRate, downRate float64) string {
	return fmt.Sprintf("host: %s\nprocess: %s\npath: %s\nrule: %s\nchains: %s\nupload: %s\ndownload: %s\nrate: ↑%s ↓%s\nid: %s",
		connHost(c), connProcess(c), c.Metadata.ProcessPath, c.Rule,
		strings.Join(c.Chains, " → "), format.Bytes(c.Upload), format.Bytes(c.Download),
		format.Rate(int64(upRate)), format.Rate(int64(downRate)), c.ID)
}
