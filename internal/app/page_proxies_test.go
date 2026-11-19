package app

import "testing"

func TestProxyColumns(t *testing.T) {
	a := newTestApp(t)

	tests := []struct {
		name string
		idx  int
		want int
	}{
		{name: "two columns", idx: 0, want: 2},
		{name: "three columns", idx: 1, want: 3},
		{name: "four columns", idx: 2, want: 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a.proxyColsIdx = tt.idx
			if got := a.proxyColumns(); got != tt.want {
				t.Fatalf("proxyColumns() = %d, want %d", got, tt.want)
			}
		})
	}

	a.proxyColsIdx = 3
	if got := a.proxyColumns(); got < 2 || got > 5 {
		t.Fatalf("auto proxyColumns() = %d, want between 2 and 5", got)
	}
}
