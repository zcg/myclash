package core

import (
	"sync"
	"time"
)

const (
	maxLogs        = 2000
	maxConnections = 1500
)

// Snapshot is an immutable view of the running core for the UI.
type RateSample struct {
	Up, Down int64
}

type Snapshot struct {
	Started   bool
	StartedAt time.Time
	Version   string
	Meta      bool
	Mode      string
	Up        int64 // bytes per second
	Down      int64 // bytes per second
	History   []RateSample
	TotalUp   int64
	TotalDown int64
	Proxies   map[string]Proxy
	Groups    []string
	Rules     []Rule

	Connections []Connection
	ConnUpload  int64
	ConnDown    int64

	Logs []LogEntry

	Memory    uint64
	LastError string
}

// State publishes snapshots from stream goroutines without holding a lock in
// the view. Get returns a copy.
type State struct {
	mu sync.RWMutex
	v  Snapshot
}

func NewState() *State { return &State{v: Snapshot{Proxies: map[string]Proxy{}}} }

func (s *State) Get() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v := s.v
	v.Proxies = make(map[string]Proxy, len(s.v.Proxies))
	for k, p := range s.v.Proxies {
		v.Proxies[k] = p
	}
	v.Groups = append([]string(nil), s.v.Groups...)
	v.Rules = append([]Rule(nil), s.v.Rules...)
	v.Connections = append([]Connection(nil), s.v.Connections...)
	v.Logs = append([]LogEntry(nil), s.v.Logs...)
	v.History = append([]RateSample(nil), s.v.History...)
	return v
}

func (s *State) SetStarted(started bool) {
	s.mu.Lock()
	s.v.Started = started
	if started {
		s.v.StartedAt = time.Now()
	}
	if !started {
		s.v.Connections = nil
		s.v.Logs = nil
		s.v.Up, s.v.Down = 0, 0
	}
	s.mu.Unlock()
}

func (s *State) SetVersion(v Version, mode string) {
	s.mu.Lock()
	s.v.Version = v.Version
	s.v.Meta = v.Meta
	s.v.Mode = mode
	s.mu.Unlock()
}

func (s *State) SetMode(mode string) {
	s.mu.Lock()
	s.v.Mode = mode
	s.mu.Unlock()
}

func (s *State) SetProxies(all map[string]Proxy) {
	groups := make([]string, 0, 8)
	for name, p := range all {
		if len(p.All) > 0 {
			groups = append(groups, name)
		}
	}
	s.mu.Lock()
	s.v.Proxies = all
	s.v.Groups = groups
	s.mu.Unlock()
}

func (s *State) SetTraffic(up, down int64, dt time.Duration) {
	if dt < 0 || dt > 5*time.Second {
		dt = time.Second
	}
	s.mu.Lock()
	s.v.Up = up
	s.v.Down = down
	s.v.TotalUp += int64(float64(up) * dt.Seconds())
	s.v.TotalDown += int64(float64(down) * dt.Seconds())
	s.v.History = append(s.v.History, RateSample{Up: up, Down: down})
	if len(s.v.History) > 60 {
		s.v.History = s.v.History[len(s.v.History)-60:]
	}
	s.mu.Unlock()
}

func (s *State) SetConnections(v Connections) {
	if len(v.Connections) > maxConnections {
		v.Connections = v.Connections[len(v.Connections)-maxConnections:]
	}
	s.mu.Lock()
	s.v.Connections = v.Connections
	s.v.ConnUpload = v.UploadTotal
	s.v.ConnDown = v.DownloadTotal
	s.mu.Unlock()
}

func (s *State) AppendLog(e LogEntry) {
	s.mu.Lock()
	s.v.Logs = append(s.v.Logs, e)
	if len(s.v.Logs) > maxLogs {
		copy(s.v.Logs, s.v.Logs[len(s.v.Logs)-maxLogs:])
		s.v.Logs = s.v.Logs[:maxLogs]
	}
	s.mu.Unlock()
}

func (s *State) ClearLogs() {
	s.mu.Lock()
	s.v.Logs = nil
	s.mu.Unlock()
}

func (s *State) SetRules(rules []Rule) {
	s.mu.Lock()
	s.v.Rules = rules
	s.mu.Unlock()
}

func (s *State) SetMemory(inuse uint64) {
	s.mu.Lock()
	s.v.Memory = inuse
	s.mu.Unlock()
}

func (s *State) SetError(err string) {
	s.mu.Lock()
	s.v.LastError = err
	s.mu.Unlock()
}
