package core

import (
	"fmt"
	"testing"
)

func benchmarkState() *State {
	s := NewState()
	conns := make([]Connection, 1000)
	for i := range conns {
		conns[i].ID = fmt.Sprint(i)
		conns[i].Metadata.Host = fmt.Sprintf("host-%d.example", i)
		conns[i].Upload = int64(i)
		conns[i].Download = int64(i * 2)
	}
	s.SetConnections(Connections{Connections: conns})
	for i := 0; i < maxLogs; i++ {
		s.AppendLog(LogEntry{Type: "info", Payload: fmt.Sprintf("line %d", i)})
	}
	return s
}

func BenchmarkStateView(b *testing.B) {
	s := benchmarkState()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = s.View()
	}
}

func BenchmarkStateRecentLogs(b *testing.B) {
	s := benchmarkState()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = s.RecentLogs(1000)
	}
}
