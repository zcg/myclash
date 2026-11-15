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
	Name string
	URL  string
}

// These are reachability probes, not a speed benchmark: any real HTTP
// response below 500 means the proxy path reached the site.
var siteTargets = []siteTarget{
	{"Google", "https://www.google.com/generate_204"},
	{"YouTube", "https://www.youtube.com/generate_204"},
	{"ChatGPT", "https://chatgpt.com/"},
	{"GitHub", "https://github.com/"},
	{"X", "https://x.com/"},
	{"Telegram", "https://telegram.org/"},
	{"Netflix", "https://www.netflix.com/"},
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
					if resp.StatusCode >= 500 {
						err = fmt.Errorf("HTTP %d", resp.StatusCode)
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
