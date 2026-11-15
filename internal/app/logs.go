package app

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"myclash/internal/core"
)

// openLogFile applies the retention policy before loading the day's history:
//   - lines older than logs.retention are dropped;
//   - the current file is truncated when it is already over logs.max_size_mb.
//
// The file keeps one line per event:
//
//	RFC3339 \t level \t payload
func (a *App) openLogFile(userData string) {
	dir := filepath.Join(userData, "logs")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	path := filepath.Join(dir, "core.log")
	a.mu.Lock()
	retention := logRetentionValue(a.logRetentionIdx)
	maxMB := a.logMaxMB
	a.mu.Unlock()
	cutoff := logCutoff(retention)
	_ = pruneLogFile(path, cutoff)
	_ = enforceLogSize(path, int64(maxMB)<<20)

	if st, err := os.Stat(path); err == nil && st.Size() > 0 {
		f, err := os.Open(path)
		if err == nil {
			sc := bufio.NewScanner(f)
			sc.Buffer(make([]byte, 64*1024), 1024*1024)
			for sc.Scan() {
				parts := strings.SplitN(sc.Text(), "\t", 3)
				if len(parts) == 3 {
					a.state.AppendLog(core.LogEntry{Type: parts[1], Payload: parts[2]})
				}
			}
			f.Close()
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	a.logMu.Lock()
	a.logFile = f
	a.logMu.Unlock()
}

func (a *App) writeLogFile(e core.LogEntry) {
	a.logMu.Lock()
	defer a.logMu.Unlock()
	if a.logFile == nil {
		return
	}
	line := fmt.Sprintf("%s\t%s\t%s\n", time.Now().Format(time.RFC3339), e.Type, e.Payload)
	_, _ = a.logFile.WriteString(line)

	a.mu.Lock()
	maxBytes := int64(a.logMaxMB) << 20
	a.mu.Unlock()
	if maxBytes <= 0 {
		return
	}
	if st, err := a.logFile.Stat(); err == nil && st.Size() > maxBytes {
		// The policy is "clear when the cap is reached", not an ever-growing
		// chain of rotated files.
		_ = a.logFile.Truncate(0)
		_, _ = a.logFile.Seek(0, 0)
	}
}

func (a *App) clearLogs() {
	a.state.ClearLogs()
	a.logMu.Lock()
	if a.logFile != nil {
		_ = a.logFile.Truncate(0)
		_, _ = a.logFile.Seek(0, 0)
	}
	a.logMu.Unlock()
	if a.dataDir != "" {
		_ = os.Remove(filepath.Join(a.dataDir, "logs", "core.log.1"))
	}
}

func (a *App) closeLogFile() {
	a.logMu.Lock()
	if a.logFile != nil {
		_ = a.logFile.Close()
		a.logFile = nil
	}
	a.logMu.Unlock()
}

// cleanupLogs applies the current retention and size limits immediately.
func (a *App) cleanupLogs() {
	if a.dataDir == "" {
		return
	}
	a.mu.Lock()
	retention := logRetentionValue(a.logRetentionIdx)
	maxMB := a.logMaxMB
	a.mu.Unlock()
	path := filepath.Join(a.dataDir, "logs", "core.log")

	a.logMu.Lock()
	if a.logFile != nil {
		_ = a.logFile.Close()
		a.logFile = nil
	}
	_ = pruneLogFile(path, logCutoff(retention))
	_ = enforceLogSize(path, int64(maxMB)<<20)
	if f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600); err == nil {
		a.logFile = f
	}
	a.logMu.Unlock()
	a.invalidate()
}

func (a *App) logMaintenance() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.cleanupLogs()
		}
	}
}

func logCutoff(retention string) time.Time {
	switch retention {
	case "day":
		return time.Now().Add(-24 * time.Hour)
	case "week":
		return time.Now().Add(-7 * 24 * time.Hour)
	case "month":
		return time.Now().Add(-30 * 24 * time.Hour)
	default:
		return time.Time{}
	}
}

// pruneLogFile rewrites path without lines whose RFC3339 timestamp is older
// than cutoff. Unparseable lines are kept rather than silently losing data.
func pruneLogFile(path string, cutoff time.Time) error {
	if cutoff.IsZero() {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	tmp := path + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 2 {
			if t, err := time.Parse(time.RFC3339, parts[0]); err == nil && t.Before(cutoff) {
				continue
			}
		}
		if _, err := out.WriteString(line + "\n"); err != nil {
			out.Close()
			_ = os.Remove(tmp)
			return err
		}
	}
	_ = f.Close()
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

func enforceLogSize(path string, maxBytes int64) error {
	if maxBytes <= 0 {
		return nil
	}
	st, err := os.Stat(path)
	if err != nil {
		return err
	}
	if st.Size() <= maxBytes {
		return nil
	}
	return os.Truncate(path, 0)
}
