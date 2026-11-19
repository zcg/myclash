package app

import (
	"testing"
	"time"

	"myclash/internal/core"
)

func TestTrackConnRatesReusesMapAndDropsClosed(t *testing.T) {
	a := newTestApp(t)
	a.connRates = map[string]connRate{"closed": {up: 1, at: time.Now()}}
	conns := []core.Connection{
		{ID: "a", Upload: 100, Download: 50},
		{ID: "b", Upload: 10, Download: 20},
	}
	a.trackConnRates(conns)
	if len(a.connRates) != 2 {
		t.Fatalf("connRates len = %d, want 2 (closed connection must be dropped)", len(a.connRates))
	}
	if _, ok := a.connRates["closed"]; ok {
		t.Fatal("closed connection rate was not dropped")
	}
	// A second update at the same snapshot must keep the map bounded and not
	// accumulate stale entries.
	a.trackConnRates(conns[:1])
	if len(a.connRates) != 1 {
		t.Fatalf("connRates len = %d, want 1", len(a.connRates))
	}
	if _, ok := a.connRates["a"]; !ok {
		t.Fatal("live connection rate was dropped")
	}
}

func TestTrafficHistorySnapshotReusesBuffer(t *testing.T) {
	a := newTestApp(t)
	a.trafficHistory = []trafficPoint{{Down: 1, Up: 2}, {Down: 3, Up: 4}}
	first := a.trafficHistorySnapshot()
	if len(first) != 2 || first[1].Down != 3 {
		t.Fatalf("unexpected snapshot: %+v", first)
	}
	buf := &first[0]

	// A second call must return the same backing array (no reallocation) and
	// reflect new data.
	a.trafficHistory = append(a.trafficHistory, trafficPoint{Down: 5, Up: 6})
	second := a.trafficHistorySnapshot()
	if &second[0] != buf {
		t.Fatal("snapshot buffer was reallocated")
	}
	if len(second) != 3 || second[2].Down != 5 {
		t.Fatalf("stale snapshot: %+v", second)
	}
}

func TestSetPageMirrorsActivePage(t *testing.T) {
	a := newTestApp(t)
	a.setPage("connections")
	if !a.onPage("connections") {
		t.Fatal("activePage not mirrored")
	}
	if a.onPage("rules") {
		t.Fatal("activePage mismatch reported as a hit")
	}
	a.setPage("rules")
	if !a.onPage("rules") || a.onPage("connections") {
		t.Fatal("activePage did not update")
	}
}
