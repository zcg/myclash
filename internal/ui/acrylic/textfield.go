package acrylic

import "github.com/egoist/mygo/ui"

// TextFieldOptions describes a settings input that belongs to the acrylic
// layer: wash background, hairline border, focus accent, and a description
// under the box so long hints cannot overlap the next row.
type TextFieldOptions struct {
	Label       string
	Description string
	Placeholder string
	Width       float32
}

func TextField(c *ui.Context, p Palette, value *string, o TextFieldOptions) ui.Element {
	col := ui.Column(c).Gap(3)
	if o.Width > 0 {
		col.Width(o.Width)
	}
	col.Children(func() {
		ui.Text(c, o.Label).FontSize(11).Bold().TextColor(p.Ink)
		var in ui.Element
		box := ui.Row(c).Height(30).AlignItems(ui.Center).Padding(0, 8).Radius(RadiusSM).
			Background(p.WashChip).Children(func() {
			in = ui.TextInputBase(c, value).Grow(1).MinWidth(0).Label(o.Label)
			if o.Placeholder != "" {
				in.Placeholder(o.Placeholder)
			}
		})
		edge := p.Hairline
		if in.Valid() && in.Focused() {
			edge = p.Accent
		} else if box.Hovered() {
			edge = p.HairlineStrong
		}
		box.Border(1, edge)
		if o.Description != "" {
			ui.Text(c, o.Description).FontSize(10).TextColor(p.InkSubtle)
		}
	})
	return col
}
