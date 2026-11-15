package widgets

import (
	"github.com/egoist/mygo/ui"

	"myclash/internal/ui/acrylic"
)

// Sparkline draws a live line for a bounded series. It is painted, not a
// widget tree: no allocation per point and no child per sample.
func Sparkline(c *ui.Context, color ui.Color, values []float64, height float32) ui.Element {
	return ui.Box(c).FillWidth().Height(height).Draw(func(p *ui.Painter, r ui.Rect) {
		if len(values) < 2 {
			p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - 1, W: r.W, H: 1}, color.Alpha(0.25), 0)
			return
		}
		maxV := 0.0
		for _, v := range values {
			if v > maxV {
				maxV = v
			}
		}
		if maxV <= 0 {
			maxV = 1
		}
		pts := make([][2]float32, len(values))
		for i, v := range values {
			x := r.X + r.W*float32(i)/float32(len(values)-1)
			y := r.Y + r.H - float32(v/maxV)*r.H
			pts[i] = [2]float32{x, y}
		}
		p.StrokePolyline(pts, 1.5, color)
	})
}

// StatCard is the overview's numbered tile: a label, a big value, an optional
// note and sparkline, all on one work card.
func StatCard(c *ui.Context, p acrylic.Palette, opacity float64, label, value, note string, icon *ui.SVG, color ui.Color, graph []float64) ui.Element {
	card := acrylic.Card(c, p, opacity).Padding(14).Grow(1)
	card.Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, label).FontSize(12).TextColor(p.InkSubtle)
			ui.Spacer(c)
			if icon != nil {
				ui.Icon(c, icon).FontSize(15).TextColor(p.InkSubtle)
			}
		})
		ui.Text(c, value).FontSize(26).Bold().TextColor(color)
		if note != "" {
			ui.Text(c, note).FontSize(11).TextColor(p.InkMuted)
		}
		if len(graph) > 1 {
			Sparkline(c, color.Alpha(0.8), graph, 34)
		}
	})
	return card
}

// SectionCard is a titled block for dense pages (settings, profiles).
func SectionCard(c *ui.Context, p acrylic.Palette, opacity float64, title string) ui.Element {
	card := acrylic.Card(c, p, opacity).Padding(14).Grow(1)
	card.Children(func() {
		ui.Text(c, title).Bold()
	})
	return card
}
