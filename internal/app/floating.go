package app

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"myclash/internal/ui/acrylic"
	"myclash/internal/ui/format"
	"myclash/internal/ui/icons"
)

func (a *App) setupFloating() {
	if a.floating != nil {
		return
	}
	opts := mygo.WindowOptions{
		Title:           "myclash 悬浮窗",
		StateKey:        "floating",
		Width:           220,
		Height:          84,
		Frameless:       true,
		AlwaysOnTop:     true,
		SkipTaskbar:     true,
		DisableResize:   true,
		Transparent:     true,
		BackgroundColor: "#00000000",
		Content:         ui.View(a.floatView),
	}
	a.floating = mygo.NewWindow(opts)
	a.floating.OnClose(func(e *mygo.CloseEvent) {
		e.PreventDefault()
		a.floating.Hide()
	})
}

func (a *App) toggleFloating() {
	a.setupFloating()
	if a.floating == nil {
		return
	}
	if a.floating.IsVisible() {
		a.floating.Hide()
	} else {
		a.floating.Show()
		a.floating.Invalidate()
	}
}

func (a *App) floatView(c *ui.Context) {
	p := acrylic.For(a.dark)
	c.SetTheme(a.uiTheme(p))
	s := a.state.View()
	ui.Column(c).Fill().Padding(10).DragWindow().Children(func() {
		card := acrylic.Panel(c, p, 0.92).Fill().Padding(10)
		card.Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "myclash").FontSize(11).Bold().TextColor(p.InkSubtle)
				ui.Spacer(c)
				acrylic.Chip(c, p, format.Rate(s.Down), p.Accent)
			})
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				acrylic.Icon(c, icons.ArrowDown, 14, p.Accent)
				ui.Text(c, format.Rate(s.Down)).FontSize(16).Bold().TextColor(p.Ink)
				acrylic.Icon(c, icons.ArrowUp, 14, p.Success)
				ui.Text(c, format.Rate(s.Up)).FontSize(14).Bold().TextColor(p.InkMuted)
			})
			ui.Textf(c, "连接 %d · %s", len(s.Connections), modeLabel(s.Mode)).FontSize(10).TextColor(p.InkSubtle)
		})
	})
}
