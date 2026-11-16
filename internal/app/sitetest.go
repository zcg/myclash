package app

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type siteTarget struct {
	Name   string
	URL    string
	Status int
}

// These are reachability probes, not a speed benchmark. generate_204
// endpoints intentionally return 204; every other target must return 200.
var siteTargets = []siteTarget{
	{"Google", "https://www.google.com/generate_204", http.StatusNoContent},
	{"YouTube", "https://www.youtube.com/generate_204", http.StatusNoContent},
	{"ChatGPT", "https://chatgpt.com/", http.StatusOK},
	{"GitHub", "https://github.com/", http.StatusOK},
	{"X", "https://x.com/", http.StatusOK},
	{"Telegram", "https://telegram.org/", http.StatusOK},
	{"Netflix", "https://www.netflix.com/", http.StatusOK},
}

func siteShortError(s string) string {
	if s == "" {
		return "不可达"
	}
	r := []rune(s)
	if len(r) > 26 {
		return string(r[:26]) + "…"
	}
	return s
}

func (a *App) testSites() {
	a.mu.Lock()
	if a.siteTesting {
		a.mu.Unlock()
		return
	}
	a.siteTesting = true
	a.siteResults = make([]siteResult, len(siteTargets))
	for i, t := range siteTargets {
		a.siteResults[i].Name = t.Name
	}
	a.mu.Unlock()
	a.invalidate()

	port := a.settings.Get().Core.MixedPort
	proxyURL := &url.URL{Scheme: "http", Host: fmt.Sprintf("127.0.0.1:%d", port)}
	tr := &http.Transport{Proxy: http.ProxyURL(proxyURL), DisableKeepAlives: true}
	defer tr.CloseIdleConnections()
	client := &http.Client{Transport: tr, Timeout: 9 * time.Second}
	sem := make(chan struct{}, 6)
	var wg sync.WaitGroup
	for i, target := range siteTargets {
		i, target := i, target
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			start := time.Now()
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, target.URL, nil)
			if err == nil {
				req.Header.Set("User-Agent", "myclash/connectivity-check")
				var resp *http.Response
				resp, err = client.Do(req)
				if err == nil {
					_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 32*1024))
					resp.Body.Close()
					if resp.StatusCode != target.Status {
						err = fmt.Errorf("HTTP %d，期望 %d", resp.StatusCode, target.Status)
					}
				}
			}
			result := siteResult{Name: target.Name, MS: time.Since(start).Milliseconds()}
			if err != nil {
				result.Err = err.Error()
			} else {
				result.OK = true
			}
			a.mu.Lock()
			a.siteResults[i] = result
			a.mu.Unlock()
			a.invalidate()
		}()
	}
	wg.Wait()
	a.mu.Lock()
	a.siteTesting = false
	a.mu.Unlock()
	a.invalidate()
}

func (a *App) siteSnapshot() ([]siteResult, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]siteResult, len(a.siteResults))
	copy(out, a.siteResults)
	return out, a.siteTesting
}
