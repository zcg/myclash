package app

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// trafficPoint is one persisted sample. Totals let a future history page show
// usage without replaying every second of traffic.
type trafficPoint struct {
	Time      time.Time `json:"t"`
	Up        int64     `json:"up"`
	Down      int64     `json:"down"`
	TotalUp   int64     `json:"total_up"`
	TotalDown int64     `json:"total_down"`
}

func (a *App) loadTrafficHistory(dir string) {
	path := filepath.Join(dir, "traffic.jsonl")
	a.trafficPath = path
	a.lastTrafficFlush = time.Now()
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		var p trafficPoint
		if json.Unmarshal(sc.Bytes(), &p) == nil && p.Time.After(cutoff) {
			a.trafficHistory = append(a.trafficHistory, p)
		}
	}
}

func (a *App) recordTraffic(up, down int64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.trafficPath == "" {
		return
	}
	now := time.Now()
	if now.Sub(a.lastTrafficSave) < 60*time.Second {
		return
	}
	a.lastTrafficSave = now
	s := a.state.View()
	p := trafficPoint{
		Time:      now,
		Up:        up,
		Down:      down,
		TotalUp:   s.TotalUp,
		TotalDown: s.TotalDown,
	}
	a.trafficHistory = append(a.trafficHistory, p)
	a.trafficPending = append(a.trafficPending, p)
	if len(a.trafficHistory) > 12000 {
		a.trafficHistory = a.trafficHistory[len(a.trafficHistory)-12000:]
	}
	// Buffer to disk every five minutes: 60s samples in memory, 144 writes a
	// day instead of 2880, and at most five minutes lost on a hard crash.
	if now.Sub(a.lastTrafficFlush) >= 5*time.Minute {
		a.flushTrafficLocked()
	}
}

func (a *App) flushTrafficHistory() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.flushTrafficLocked()
}

func (a *App) flushTrafficLocked() {
	if a.trafficPath == "" || len(a.trafficPending) == 0 {
		return
	}
	f, err := os.OpenFile(a.trafficPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	for _, p := range a.trafficPending {
		b, _ := json.Marshal(p)
		if _, err := f.Write(append(b, '\n')); err != nil {
			return
		}
	}
	a.trafficPending = nil
	a.lastTrafficFlush = time.Now()
}

func (a *App) trafficSeries(limit int, down bool) []float64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	if limit <= 0 || len(a.trafficHistory) == 0 {
		return nil
	}
	start := len(a.trafficHistory) - limit
	if start < 0 {
		start = 0
	}
	out := make([]float64, 0, len(a.trafficHistory)-start)
	for _, p := range a.trafficHistory[start:] {
		if down {
			out = append(out, float64(p.Down))
		} else {
			out = append(out, float64(p.Up))
		}
	}
	return out
}

func (a *App) trafficHistorySnapshot() []trafficPoint {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]trafficPoint, len(a.trafficHistory))
	copy(out, a.trafficHistory)
	return out
}
