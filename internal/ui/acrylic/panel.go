package acrylic

import (
	"runtime"

	"github.com/egoist/mygo/ui"
)

// Panel is a card for a vertical stack (a page section), not for a row that
// stretches its cards to one height. It has no Grow and no MinHeight: its
// height is its content, so a long form is never clipped by a flex parent.
func Panel(c *ui.Context, p Palette, opacity float64) ui.Element {
	panel := ui.Column(c).FillWidth().Shrink(0).
		Radius(CardRadius(runtime.GOOS)).
		Background(CardFill(p, opacity)).
		Clip()
	return Raised(panel, p)
}
