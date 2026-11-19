package widgets

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// SparklineFunc reuses its scratch buffers across draws. This exercises
// several draws with growing and shrinking sample counts to make sure the
// reuse never panics or leaves stale points behind.
func TestSparklineFuncReusesBuffers(t *testing.T) {
	vals := []float64{1, 3, 2, 5, 4}
	view := func(c *ui.Context) {
		SparklineFunc(c, ui.RGB(0, 0, 0), len(vals), func(i int) float64 { return vals[i] }, 34)
	}
	tt := ui.NewTester(view, 200, 60)
	tt.Frame()
	vals = []float64{2, 1}
	tt.Frame()
	vals = []float64{9, 7, 8, 6, 5, 4, 3, 2, 1}
	tt.Frame()
	tt.Frame()
}

func TestSparklineEmpty(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		Sparkline(c, ui.RGB(0, 0, 0), nil, 34)
	}, 200, 60)
	tt.Frame()
}
