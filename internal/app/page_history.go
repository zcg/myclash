package app

import (
	"github.com/egoist/mygo/ui"

	"myclash/internal/ui/acrylic"
	"myclash/internal/ui/format"
	"myclash/internal/ui/widgets"
)

func (a *App) pageTraffic(c *ui.Context, p acrylic.Palette) {
	a.header(c, p, "流量", "按 30 秒采样的上下行历史；保留 7 天")
	hist := a.trafficHistorySnapshot()
	ui.Row(c).Gap(10).AlignItems(ui.Center).Wrap().Children(func() {
		s := a.state.Get()
		ui.Textf(c, "本次累计 ↑ %s / ↓ %s", format.Bytes(s.TotalUp), format.Bytes(s.TotalDown)).
			FontSize(11).TextColor(p.InkMuted)
		ui.Spacer(c)
		if acrylic.Button(c, p, "打开数据目录", false).Clicked() {
			a.openDataDir()
		}
	})
	a.errorBar(c, p)
	if len(hist) < 2 {
		ui.Text(c, "还没有足够的历史采样。内核运行后每 30 秒记录一次。").TextColor(p.InkMuted)
		return
	}
	down := make([]float64, 0, len(hist))
	up := make([]float64, 0, len(hist))
	for _, x := range hist {
		down = append(down, float64(x.Down))
		up = append(up, float64(x.Up))
	}
	panel := a.panel(c, p, a.cardOpacity()).Padding(a.cardPad())
	panel.Children(func() {
		ui.Row(c).Gap(10).Children(func() {
			ui.Text(c, "下行").Bold()
			ui.Text(c, "上行").TextColor(p.InkSubtle)
		})
		ui.Row(c).Gap(10).Children(func() {
			widgets.Sparkline(c, p.Accent.Alpha(0.8), down, 70)
			widgets.Sparkline(c, p.Success.Alpha(0.8), up, 70)
		})
	})
	rows := hist
	if len(rows) > 240 {
		rows = rows[len(rows)-240:]
	}
	ui.Column(c).Height(360).Background(p.Code).Radius(8).Padding(10).Clip().Children(func() {
		ui.List(c, &a.trafficState, len(rows), func(i int) {
			x := rows[len(rows)-1-i]
			ui.Textf(c, "%s   ↑ %-10s ↓ %-10s   累计 %s / %s",
				x.Time.Format("01-02 15:04:05"), format.Rate(x.Up), format.Rate(x.Down),
				format.Bytes(x.TotalUp), format.Bytes(x.TotalDown)).
				Font("monospace").FontSize(11).SingleLine()
		}).Grow(1)
	})
}
