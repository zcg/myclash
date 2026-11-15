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

const maxLogFileSize = 8 << 20

func (a *App) openLogFile(userData string) {
	dir := filepath.Join(userData, "logs")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	path := filepath.Join(dir, "core.log")
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
	if st, err := a.logFile.Stat(); err == nil && st.Size() > maxLogFileSize {
		name := a.logFile.Name()
		_ = a.logFile.Close()
		a.logFile = nil
		_ = os.Rename(name, name+".1")
		if f, err := os.OpenFile(name, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600); err == nil {
			a.logFile = f
		}
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
}

func (a *App) closeLogFile() {
	a.logMu.Lock()
	if a.logFile != nil {
		_ = a.logFile.Close()
		a.logFile = nil
	}
	a.logMu.Unlock()
}
